package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type Params struct {
	driver      string
	driverDSN   string // DSN formatted for the specific driver
	originalDSN string // Original DSN for logging
	query       string
	params      []interface{}
	timeout     time.Duration
}

type Req struct {
	Driver  string        `map:"driver"`
	DSN     string        `map:"dsn"`
	Query   string        `map:"query"`
	Params  []interface{} `map:"params"`
	Timeout string        `map:"timeout"`
}

type Res struct {
	Code         int           `map:"code"`
	RowsAffected int64         `map:"rows_affected"`
	Rows         []interface{} `map:"rows"`
	Error        string        `map:"error"`
}

type Result struct {
	Req Req           `map:"req"`
	Res Res           `map:"res"`
	RT  time.Duration `map:"rt"`
}

func ParseParams(with map[string]string) (*Params, error) {
	dsn := with["dsn"]
	query := with["query"]

	// Validate required parameters
	if dsn == "" {
		return nil, fmt.Errorf("dsn parameter is required")
	}
	if query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	// Parse driver and DSN from URL-style DSN
	driver, driverDSN, err := parseDSN(dsn)
	if err != nil {
		return nil, err
	}

	params := &Params{
		driver:      driver,
		driverDSN:   driverDSN,
		originalDSN: dsn,
		query:       query,
		params:      make([]interface{}, 0),
	}

	// Parse timeout
	if timeoutStr := with["timeout"]; timeoutStr != "" {
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			// Try parsing as seconds (integer)
			if seconds, errInt := strconv.Atoi(timeoutStr); errInt == nil {
				timeout = time.Duration(seconds) * time.Second
			} else {
				return nil, fmt.Errorf("invalid timeout format: %s (use duration string like '30s' or integer seconds)", timeoutStr)
			}
		}
		params.timeout = timeout
	} else {
		params.timeout = 30 * time.Second // Default timeout
	}

	// Parse query parameters (param1, param2, etc.) in correct order
	paramMap := make(map[int]string)
	maxParamNum := 0
	
	// First collect all parameters with their numbers
	for key, value := range with {
		if strings.HasPrefix(key, "param") {
			// Extract parameter number
			paramNumStr := strings.TrimPrefix(key, "param")
			if paramNumStr == "" {
				continue
			}
			
			paramNum, err := strconv.Atoi(paramNumStr)
			if err != nil {
				continue
			}
			
			paramMap[paramNum] = value
			if paramNum > maxParamNum {
				maxParamNum = paramNum
			}
		}
	}
	
	// Add parameters in order
	for i := 1; i <= maxParamNum; i++ {
		if value, exists := paramMap[i]; exists {
			// Try to convert to appropriate type
			if intVal, err := strconv.Atoi(value); err == nil {
				params.params = append(params.params, intVal)
			} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
				params.params = append(params.params, floatVal)
			} else if boolVal, err := strconv.ParseBool(value); err == nil {
				params.params = append(params.params, boolVal)
			} else {
				params.params = append(params.params, value)
			}
		}
	}

	return params, nil
}

func parseDSN(dsn string) (driver, driverDSN string, err error) {
	// Handle file paths for SQLite
	if strings.HasSuffix(dsn, ".db") || strings.HasSuffix(dsn, ".sqlite") || strings.HasSuffix(dsn, ".sqlite3") {
		abs, err := filepath.Abs(dsn)
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve SQLite file path: %w", err)
		}
		
		// Check if file exists or can be created
		dir := filepath.Dir(abs)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return "", "", fmt.Errorf("directory does not exist for SQLite file: %s", dir)
		}
		
		return "sqlite3", abs, nil
	}

	// Parse URL-style DSN
	u, err := url.Parse(dsn)
	if err != nil {
		return "", "", fmt.Errorf("invalid DSN format: %w", err)
	}

	switch u.Scheme {
	case "mysql":
		// Convert to MySQL DSN format: user:password@tcp(host:port)/database?params
		userInfo := ""
		if u.User != nil {
			password, hasPassword := u.User.Password()
			if hasPassword {
				// Password is explicitly set (even if empty)
				userInfo = fmt.Sprintf("%s:%s@", u.User.Username(), password)
			} else {
				// No password field in URL
				userInfo = fmt.Sprintf("%s@", u.User.Username())
			}
		}
		
		host := u.Host
		if host == "" {
			host = "localhost:3306"
		}
		
		database := strings.TrimPrefix(u.Path, "/")
		query := ""
		if u.RawQuery != "" {
			query = "?" + u.RawQuery
		}
		
		driverDSN = fmt.Sprintf("%stcp(%s)/%s%s", userInfo, host, database, query)
		return "mysql", driverDSN, nil

	case "postgres", "postgresql":
		// PostgreSQL DSN can be used as-is with some modifications
		driverDSN = dsn
		// Replace scheme if needed
		if u.Scheme == "postgresql" {
			driverDSN = strings.Replace(driverDSN, "postgresql://", "postgres://", 1)
		}
		return "postgres", driverDSN, nil

	case "sqlite", "sqlite3":
		// SQLite DSN: just the file path
		return "sqlite3", strings.TrimPrefix(u.Path, "/"), nil

	default:
		return "", "", fmt.Errorf("unsupported database driver: %s (supported: mysql, postgres, sqlite3)", u.Scheme)
	}
}

func ExecuteQuery(params *Params, log hclog.Logger) (map[string]string, error) {
	start := time.Now()

	log.Debug("executing database query", "driver", params.driver, "query", params.query, "params", params.params)

	// Open database connection
	db, err := sql.Open(params.driver, params.driverDSN)
	if err != nil {
		return createErrorResult(params, start, fmt.Errorf("failed to open database: %w", err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to parse database", "error", err)
		}
	}()

	// Test connection
	if err := db.Ping(); err != nil {
		return createErrorResult(params, start, fmt.Errorf("failed to connect to database: %w", err))
	}

	// Determine query type
	trimmedQuery := strings.TrimSpace(strings.ToUpper(params.query))
	isSelect := strings.HasPrefix(trimmedQuery, "SELECT") ||
		strings.HasPrefix(trimmedQuery, "SHOW") ||
		strings.HasPrefix(trimmedQuery, "DESCRIBE") ||
		strings.HasPrefix(trimmedQuery, "EXPLAIN") ||
		strings.HasPrefix(trimmedQuery, "WITH") // CTE queries

	var result *Result

	if isSelect {
		result, err = executeSelectQuery(db, params, start)
	} else {
		result, err = executeNonSelectQuery(db, params, start)
	}

	if err != nil {
		return createErrorResult(params, start, err)
	}

	// Convert to map[string]string using probe's mapping function
	mapResult, err := probe.StructToMapByTags(result)
	if err != nil {
		return createErrorResult(params, start, fmt.Errorf("failed to convert result to map: %w", err))
	}

	// Flatten the result like other actions do
	return probe.FlattenInterface(mapResult), nil
}

func executeSelectQuery(db *sql.DB, params *Params, start time.Time) (res *Result, err error) {
	rows, err := db.Query(params.query, params.params...)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := rows.Close()
		if err == nil {
			err = closeErr
		}
	}()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []interface{}

	// Scan rows
	for rows.Next() {
		// Create a slice for scanning
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		// Scan the row
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		// Create a map for this row
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for text fields
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			rowMap[col] = val
		}
		results = append(results, rowMap)
	}

	// Check for errors from iteration
	if err := rows.Err(); err != nil {
		return nil, err
	}

	duration := time.Since(start)

	return &Result{
		Req: Req{
			Driver:  params.driver,
			DSN:     params.originalDSN,
			Query:   params.query,
			Params:  params.params,
			Timeout: params.timeout.String(),
		},
		Res: Res{
			Code:         0,
			RowsAffected: int64(len(results)),
			Rows:         results,
		},
		RT: duration,
	}, nil
}

func executeNonSelectQuery(db *sql.DB, params *Params, start time.Time) (*Result, error) {
	result, err := db.Exec(params.query, params.params...)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	duration := time.Since(start)

	return &Result{
		Req: Req{
			Driver:  params.driver,
			DSN:     params.originalDSN,
			Query:   params.query,
			Params:  params.params,
			Timeout: params.timeout.String(),
		},
		Res: Res{
			Code:         0,
			RowsAffected: rowsAffected,
			Rows:         []interface{}{},
		},
		RT: duration,
	}, nil
}

func createErrorResult(params *Params, start time.Time, err error) (map[string]string, error) {
	duration := time.Since(start)

	result := &Result{
		Req: Req{
			Driver:  params.driver,
			DSN:     params.originalDSN,
			Query:   params.query,
			Params:  params.params,
			Timeout: params.timeout.String(),
		},
		Res: Res{
			Code:         1,
			RowsAffected: 0,
			Rows:         []interface{}{},
			Error:        err.Error(),
		},
		RT: duration,
	}

	// Convert to map[string]string
	mapResult, mapErr := probe.StructToMapByTags(result)
	if mapErr != nil {
		return map[string]string{}, fmt.Errorf("failed to convert error result to map: %w", mapErr)
	}

	return probe.FlattenInterface(mapResult), err
}
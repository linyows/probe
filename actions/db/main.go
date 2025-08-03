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
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(args []string, with map[string]string) (map[string]string, error) {
	truncateLength := probe.MaxLogStringLength
	truncatedParams := probe.TruncateMapStringString(with, truncateLength)
	a.log.Debug("received db request parameters", "params", truncatedParams)

	// Validate required parameters
	dsn, exists := with["dsn"]
	if !exists || dsn == "" {
		return map[string]string{}, fmt.Errorf("dsn parameter is required")
	}

	query, exists := with["query"]
	if !exists || query == "" {
		return map[string]string{}, fmt.Errorf("query parameter is required")
	}

	// Parse parameters
	params, err := parseParams(with)
	if err != nil {
		a.log.Error("failed to parse parameters", "error", err)
		return map[string]string{}, err
	}

	// Execute database query
	result, err := executeQuery(params, a.log)
	if err != nil {
		a.log.Error("database query execution failed", "error", err)
		// Return result even on error for debugging
		return result, err
	}

	truncatedResult := probe.TruncateMapStringString(result, truncateLength)
	a.log.Debug("database query completed", "result", truncatedResult)

	return result, nil
}

type dbParams struct {
	driver      string
	driverDSN   string // DSN formatted for the specific driver
	originalDSN string // Original DSN for logging
	query       string
	params      []interface{}
	timeout     time.Duration
}

func parseParams(with map[string]string) (*dbParams, error) {
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

	params := &dbParams{
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
			return nil, fmt.Errorf("invalid timeout format: %s", timeoutStr)
		}
		params.timeout = timeout
	} else {
		params.timeout = 30 * time.Second // Default timeout
	}

	// Parse query parameters
	i := 0
	for {
		paramKey := fmt.Sprintf("params__%d", i)
		if paramValue, exists := with[paramKey]; exists {
			// Try to parse as different types
			if val, err := strconv.Atoi(paramValue); err == nil {
				params.params = append(params.params, val)
			} else if val, err := strconv.ParseFloat(paramValue, 64); err == nil {
				params.params = append(params.params, val)
			} else if val, err := strconv.ParseBool(paramValue); err == nil {
				params.params = append(params.params, val)
			} else {
				params.params = append(params.params, paramValue)
			}
			i++
		} else {
			break
		}
	}

	return params, nil
}

func parseDSN(dsn string) (driver, driverDSN string, err error) {
	// Parse as URL
	u, err := url.Parse(dsn)
	if err != nil {
		return "", "", fmt.Errorf("invalid DSN format: %s", dsn)
	}

	switch u.Scheme {
	case "mysql":
		// Convert mysql://user:pass@tcp(host:port)/database to user:pass@tcp(host:port)/database
		// Remove the mysql:// prefix and convert to Go MySQL driver format
		if u.Host == "" {
			return "", "", fmt.Errorf("mysql DSN requires host")
		}

		var auth string
		if u.User != nil {
			auth = u.User.String() + "@"
		}

		driverDSN = fmt.Sprintf("%stcp(%s)%s", auth, u.Host, u.Path)
		if u.RawQuery != "" {
			driverDSN += "?" + u.RawQuery
		}

		return "mysql", driverDSN, nil

	case "postgres":
		// PostgreSQL DSN can be used as-is
		return "postgres", dsn, nil

	case "sqlite":
		// Convert sqlite:///path or sqlite://./path to just the path
		var path string
		if strings.HasPrefix(dsn, "sqlite://./") {
			// Relative path: sqlite://./path/to/file.db
			path = strings.TrimPrefix(dsn, "sqlite://")
		} else if strings.HasPrefix(dsn, "sqlite:///") {
			// Absolute path: sqlite:///absolute/path/to/file.db
			path = strings.TrimPrefix(dsn, "sqlite://")
		} else {
			return "", "", fmt.Errorf("sqlite DSN format should be sqlite:///absolute/path or sqlite://./relative/path")
		}

		// Ensure directory exists for SQLite
		if dir := filepath.Dir(path); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", "", fmt.Errorf("failed to create directory for SQLite database: %w", err)
			}
		}

		return "sqlite3", path, nil

	default:
		return "", "", fmt.Errorf("unsupported database scheme: %s (supported: mysql, postgres, sqlite)", u.Scheme)
	}
}

type DbReq struct {
	Driver  string        `map:"driver"`
	DSN     string        `map:"dsn"`
	Query   string        `map:"query"`
	Params  []interface{} `map:"params"`
	Timeout string        `map:"timeout"`
}

type DbRes struct {
	Code         int           `map:"code"`
	RowsAffected int64         `map:"rows_affected"`
	Rows         []interface{} `map:"rows"`
	Error        string        `map:"error"`
}

type DbResult struct {
	Req DbReq         `map:"req"`
	Res DbRes         `map:"res"`
	RT  time.Duration `map:"rt"`
}

func executeQuery(params *dbParams, log hclog.Logger) (map[string]string, error) {
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

	var result *DbResult

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

func executeSelectQuery(db *sql.DB, params *dbParams, start time.Time) (res *DbResult, err error) {
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
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		results = append(results, rowMap)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	duration := time.Since(start)

	result := &DbResult{
		Req: DbReq{
			Driver:  params.driver,
			DSN:     maskDSN(params.originalDSN),
			Query:   params.query,
			Params:  params.params,
			Timeout: params.timeout.String(),
		},
		Res: DbRes{
			Code:         0,
			RowsAffected: int64(len(results)),
			Rows:         results,
			Error:        "",
		},
		RT: duration,
	}

	return result, nil
}

func executeNonSelectQuery(db *sql.DB, params *dbParams, start time.Time) (*DbResult, error) {
	result, err := db.Exec(params.query, params.params...)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// Some drivers might not support RowsAffected
		rowsAffected = 0
	}

	duration := time.Since(start)

	dbResult := &DbResult{
		Req: DbReq{
			Driver:  params.driver,
			DSN:     maskDSN(params.originalDSN),
			Query:   params.query,
			Params:  params.params,
			Timeout: params.timeout.String(),
		},
		Res: DbRes{
			Code:         0,
			RowsAffected: rowsAffected,
			Rows:         []interface{}{},
			Error:        "",
		},
		RT: duration,
	}

	return dbResult, nil
}

func createErrorResult(params *dbParams, start time.Time, err error) (map[string]string, error) {
	duration := time.Since(start)

	result := &DbResult{
		Req: DbReq{
			Driver:  params.driver,
			DSN:     maskDSN(params.originalDSN),
			Query:   params.query,
			Params:  params.params,
			Timeout: params.timeout.String(),
		},
		Res: DbRes{
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

// maskDSN masks sensitive information in DSN for logging
func maskDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		// If it's not a valid URL, check if it looks like a potentially sensitive DSN
		if strings.Contains(dsn, "://") || strings.Contains(dsn, "@") {
			return "****"
		}
		// For SQLite file paths, return as-is
		return dsn
	}

	if u.User != nil {
		// Mask password while keeping username
		username := u.User.Username()
		if _, hasPassword := u.User.Password(); hasPassword {
			u.User = url.UserPassword(username, "****")
		}
	}

	return u.String()
}

func Serve() {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	pl := &probe.ActionsPlugin{
		Impl: &Action{log: log},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: probe.Handshake,
		Plugins:         map[string]plugin.Plugin{"actions": pl},
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}

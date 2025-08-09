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

	"github.com/linyows/probe"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)


type Req struct {
	Driver  string        `map:"driver"`
	DSN     string        `map:"dsn"`
	Query   string        `map:"query"`
	Params  []interface{} `map:"params"`
	Timeout string        `map:"timeout"`
	cb      *Callback
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

func ParseRequest(with map[string]string) (*Req, string, time.Duration, error) {
	// First unflatten the input to handle nested structures like arrays
	unflattenedWith := probe.UnflattenInterface(with)
	
	// Use MapToStructByTags to directly populate Req struct
	req := &Req{}
	err := probe.MapToStructByTags(unflattenedWith, req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to parse request: %w", err)
	}

	// Validate required parameters
	if req.DSN == "" {
		return nil, "", 0, fmt.Errorf("dsn parameter is required")
	}
	if req.Query == "" {
		return nil, "", 0, fmt.Errorf("query parameter is required")
	}

	// Parse driver and DSN from URL-style DSN
	driver, driverDSN, err := parseDSN(req.DSN)
	if err != nil {
		return nil, "", 0, err
	}
	req.Driver = driver

	// Parse timeout
	timeout := 30 * time.Second // Default timeout
	if req.Timeout != "" {
		if parsedTimeout, err := time.ParseDuration(req.Timeout); err == nil {
			timeout = parsedTimeout
		} else if seconds, errInt := strconv.Atoi(req.Timeout); errInt == nil {
			timeout = time.Duration(seconds) * time.Second
		} else {
			return nil, "", 0, fmt.Errorf("invalid timeout format: %s (use duration string like '30s' or integer seconds)", req.Timeout)
		}
	}
	req.Timeout = timeout.String()

	return req, driverDSN, timeout, nil
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

func (r *Req) Execute(driverDSN string, timeout time.Duration) (map[string]string, error) {
	start := time.Now()

	// Before callback
	if r.cb != nil && r.cb.before != nil {
		r.cb.before(r.Query, r.Params)
	}

	// Open database connection
	db, err := sql.Open(r.Driver, driverDSN)
	if err != nil {
		return r.createErrorResult(start, fmt.Errorf("failed to open database: %w", err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			// Log close error if needed
		}
	}()

	// Test connection
	if err := db.Ping(); err != nil {
		return r.createErrorResult(start, fmt.Errorf("failed to connect to database: %w", err))
	}

	// Determine query type
	trimmedQuery := strings.TrimSpace(strings.ToUpper(r.Query))
	isSelect := strings.HasPrefix(trimmedQuery, "SELECT") ||
		strings.HasPrefix(trimmedQuery, "SHOW") ||
		strings.HasPrefix(trimmedQuery, "DESCRIBE") ||
		strings.HasPrefix(trimmedQuery, "EXPLAIN") ||
		strings.HasPrefix(trimmedQuery, "WITH") // CTE queries

	var result *Result

	if isSelect {
		result, err = r.executeSelectQuery(db, start)
	} else {
		result, err = r.executeNonSelectQuery(db, start)
	}

	if err != nil {
		return r.createErrorResult(start, err)
	}

	// After callback on success
	if r.cb != nil && r.cb.after != nil {
		r.cb.after(result)
	}

	// Convert to map[string]string using probe's mapping function
	mapResult, err := probe.StructToMapByTags(result)
	if err != nil {
		return r.createErrorResult(start, fmt.Errorf("failed to convert result to map: %w", err))
	}

	// Flatten the result like other actions do
	return probe.FlattenInterface(mapResult), nil
}

func (r *Req) executeSelectQuery(db *sql.DB, start time.Time) (res *Result, err error) {
	rows, err := db.Query(r.Query, r.Params...)
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
		Req: *r,
		Res: Res{
			Code:         0,
			RowsAffected: int64(len(results)),
			Rows:         results,
		},
		RT: duration,
	}, nil
}

func (r *Req) executeNonSelectQuery(db *sql.DB, start time.Time) (*Result, error) {
	result, err := db.Exec(r.Query, r.Params...)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	duration := time.Since(start)

	return &Result{
		Req: *r,
		Res: Res{
			Code:         0,
			RowsAffected: rowsAffected,
			Rows:         []interface{}{},
		},
		RT: duration,
	}, nil
}

func (r *Req) createErrorResult(start time.Time, err error) (map[string]string, error) {
	duration := time.Since(start)

	result := &Result{
		Req: *r,
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

type Option func(*Callback)

type Callback struct {
	before func(query string, params []interface{})
	after  func(result *Result)
}

func ExecuteQuery(data map[string]string, opts ...Option) (map[string]string, error) {
	req, driverDSN, timeout, err := ParseRequest(data)
	if err != nil {
		return map[string]string{}, err
	}

	cb := &Callback{}
	for _, opt := range opts {
		opt(cb)
	}
	req.cb = cb

	return req.Execute(driverDSN, timeout)
}

func WithBefore(f func(query string, params []interface{})) Option {
	return func(c *Callback) {
		c.before = f
	}
}

func WithAfter(f func(result *Result)) Option {
	return func(c *Callback) {
		c.after = f
	}
}


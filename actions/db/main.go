package db

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe/actionrpc"
	cl "github.com/linyows/probe/db"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(with map[string]any) (map[string]any, error) {
	// The password in the DSN is hidden by the workflow runner, which learns
	// it before the action starts and masks the records this action logs.
	actionrpc.LogParams(a.log, "received db request parameters", with)

	// Validate required parameters
	dsnVal, exists := with["dsn"]
	if !exists || dsnVal == nil {
		return map[string]any{}, fmt.Errorf("dsn parameter is required")
	}
	dsn, ok := dsnVal.(string)
	if !ok || dsn == "" {
		return map[string]any{}, fmt.Errorf("dsn parameter must be a non-empty string")
	}

	queryVal, exists := with["query"]
	if !exists || queryVal == nil {
		return map[string]any{}, fmt.Errorf("query parameter is required")
	}
	query, ok := queryVal.(string)
	if !ok || query == "" {
		return map[string]any{}, fmt.Errorf("query parameter must be a non-empty string")
	}

	// Execute database query with logger callbacks
	result, err := cl.ExecuteQuery(with,
		cl.WithBefore(func(query string, params []any) {
			a.log.Debug("executing database query", "query", query, "params", params)
		}),
		cl.WithAfter(func(result *cl.Result) {
			a.log.Debug("database query completed", "rows_affected", result.Res.RowsAffected, "duration", result.RT)
		}),
	)
	actionrpc.LogOutcome(a.log, "database query", result, err)

	return result, err
}

func Serve() {
	actionrpc.Serve(func(log hclog.Logger) actionrpc.Action {
		return &Action{log: log}
	})
}

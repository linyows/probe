package actions

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/linyows/probe"
	cl "github.com/linyows/probe/db"
)

func runDB(log hclog.Logger, with map[string]any) (map[string]any, error) {
	truncateLength := probe.MaxLogStringLength
	truncatedParams := probe.TruncateMapStringAny(with, truncateLength)
	log.Debug("received db request parameters", "params", truncatedParams)

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
			log.Debug("executing database query", "query", query, "params", params)
		}),
		cl.WithAfter(func(result *cl.Result) {
			log.Debug("database query completed", "rows_affected", result.Res.RowsAffected, "duration", result.RT)
		}),
	)
	if err != nil {
		log.Error("database query execution failed", "error", err)
		return result, err
	}

	truncatedResult := probe.TruncateMapStringAny(result, truncateLength)
	log.Debug("database query completed", "result", truncatedResult)

	return result, nil
}

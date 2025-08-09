package db

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
	cl "github.com/linyows/probe/db"
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

	// Execute database query with logger callbacks
	result, err := cl.ExecuteQuery(with,
		cl.WithBefore(func(query string, params []interface{}) {
			a.log.Debug("executing database query", "query", query, "params", params)
		}),
		cl.WithAfter(func(result *cl.Result) {
			a.log.Debug("database query completed", "rows_affected", result.Res.RowsAffected, "duration", result.RT)
		}),
	)
	if err != nil {
		a.log.Error("database query execution failed", "error", err)
		// Return result even on error for debugging
		return result, err
	}

	truncatedResult := probe.TruncateMapStringString(result, truncateLength)
	a.log.Debug("database query completed", "result", truncatedResult)

	return result, nil
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

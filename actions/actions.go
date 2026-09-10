// Package actions hosts the built-in actions and serves them to the workflow
// runner over the go-plugin protocol.
package actions

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
)

// RunFunc executes one action with the parameters a step declared under `with`.
type RunFunc func(log hclog.Logger, with map[string]any) (map[string]any, error)

// builtin maps an action name, as written in a step's `uses`, to its implementation.
var builtin = map[string]RunFunc{
	"browser":      runBrowser,
	"db":           runDB,
	"embedded":     runEmbedded,
	"grpc":         runGRPC,
	"hello":        runHello,
	"http":         runHTTP,
	"imap":         runIMAP,
	"mail-latency": runMailLatency,
	"shell":        runShell,
	"smtp":         runSMTP,
	"ssh":          runSSH,
}

// Lookup returns the implementation of a built-in action.
func Lookup(name string) (RunFunc, bool) {
	run, ok := builtin[name]
	return run, ok
}

type action struct {
	log hclog.Logger
	run RunFunc
}

func (a *action) Run(with map[string]any) (map[string]any, error) {
	return a.run(a.log, with)
}

// Serve starts the plugin server for a single action and blocks until the
// workflow runner closes the connection. Log records are emitted as JSON on
// stderr, where the runner picks them up and re-filters them by its own level.
func Serve(run RunFunc) {
	log := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: probe.Handshake,
		Plugins: map[string]plugin.Plugin{
			"actions": &probe.ActionsPlugin{Impl: &action{log: log, run: run}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

// logParams records the parameters an action received, truncating long values
// so that a large request body does not flood the log.
func logParams(log hclog.Logger, msg string, with map[string]any) {
	log.Debug(msg, "params", probe.TruncateMapStringAny(with, probe.MaxLogStringLength))
}

// logOutcome records how an action finished. subject names what was attempted,
// e.g. "http request", and is used to build the message.
func logOutcome(log hclog.Logger, subject string, ret map[string]any, err error) {
	if err != nil {
		log.Error(subject+" failed", "error", err)
		return
	}
	log.Debug(subject+" completed successfully", "result", probe.TruncateMapStringAny(ret, probe.MaxLogStringLength))
}

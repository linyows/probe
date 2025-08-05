package embedded

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/linyows/probe"
	"gopkg.in/go-playground/validator.v9"
)

type Action struct {
	log hclog.Logger
}

func (a *Action) Run(args []string, with map[string]string) (map[string]string, error) {
	m := probe.UnflattenInterface(with)
	r := &Req{}
	if err := probe.MapToStructByTags(m, r); err != nil {
		return map[string]string{}, err
	}
	if r.Path == "" {
		return nil, fmt.Errorf("path must be of type string")
	}

	// Execute embedded steps
	result, err := executeEmbeddedSteps(r, a.log)
	if err != nil {
		a.log.Error("embedded steps execution failed", "error", err)
		return result, err
	}

	return result, nil
}

type Req struct {
	Path string         `map:"path"`
	Vars map[string]any `map:"vars"`
}

type Res struct {
	Code    int            `map:"code"`
	Outputs map[string]any `map:"outputs"`
	Report  string         `map:"report"`
	Err     string         `map:"error"`
}

type Result struct {
	Req Req           `map:"req"`
	Res Res           `map:"res"`
	RT  time.Duration `map:"rt"`
}

func executeEmbeddedSteps(req *Req, log hclog.Logger) (map[string]string, error) {
	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		return map[string]string{}, fmt.Errorf("failed to resolve path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return map[string]string{}, fmt.Errorf("embedded steps file does not exist: %s", absPath)
	}

	log.Debug("loading embedded steps file", "path", absPath)

	data, err := os.ReadFile(absPath)
	if err != nil {
		return map[string]string{}, fmt.Errorf("failed to read embedded steps file: %w", err)
	}

	job := &probe.Job{}
	v := validator.New()
	dec := yaml.NewDecoder(bytes.NewReader([]byte(data)), yaml.Validator(v))
	if err = dec.Decode(job); err != nil {
		return map[string]string{}, fmt.Errorf("failed to decode YAML job: %w", err)
	}

	if len(job.Steps) == 0 {
		return map[string]string{}, fmt.Errorf("no steps found in embedded file: %s", absPath)
	}

	log.Debug("parsed embedded steps", "count", len(job.Steps))

	start := time.Now()
	jobID := "embedded"
	job.ID = jobID
	result := probe.NewResult()
	jr := &probe.JobResult{
		JobName:   job.Name,
		JobID:     jobID,
		StartTime: start,
	}
	result.Jobs[jobID] = jr

	verbose := true
	ctx := probe.JobContext{
		Vars:    req.Vars,
		Outputs: probe.NewOutputs(),
		Result:  result,
		Config: probe.Config{
			Verbose: verbose,
		},
		Printer: probe.NewPrinter(verbose, []string{jobID}),
	}

	code := 0
	er := ""
	if err := job.Start(ctx); err != nil {
		code = 1
		er = err.Error()
		jr.Status = "Failed"
	} else {
		jr.Status = "Completed"
	}
	duration := time.Since(start)
	jr.EndTime = jr.StartTime.Add(duration)

	log.Debug("embedded execution completed", "outputs", ctx.Outputs.GetAll())

	ret := &Result{
		Req: *req,
		Res: Res{
			Code:    code,
			Outputs: ctx.Outputs.GetAll(),
			Report:  ctx.Printer.GenerateReport(result, false),
			Err:     er,
		},
		RT: duration,
	}

	mapRet, err := probe.StructToMapByTags(ret)
	if err != nil {
		return map[string]string{}, err
	}

	return probe.FlattenInterface(mapRet), nil
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

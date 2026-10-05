package embedded

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/linyows/probe"
	"github.com/linyows/probe/mapping"
	"gopkg.in/go-playground/validator.v9"
)

type Req struct {
	Path string         `map:"path" validate:"required"`
	Vars map[string]any `map:"vars"`
	cb   *Callback
}

type Res struct {
	Code    int            `map:"code"`
	Outputs map[string]any `map:"outputs"`
	Report  string         `map:"report"`
	Error   string         `map:"error"`
	Dump    bool           `map:"dump"`
}

type Result struct {
	Req    Req           `map:"req"`
	Res    Res           `map:"res"`
	RT     time.Duration `map:"rt"`
	Status int           `map:"status"`
}

type Option func(*Callback)

type Callback struct {
	before func(path string, vars map[string]any)
	after  func(result *Result)
}

func NewReq() *Req {
	return &Req{
		Vars: make(map[string]any),
	}
}

func (r *Req) Do() (*Result, error) {
	if r.Path == "" {
		return nil, fmt.Errorf("Req.Path is required")
	}

	result := &Result{Req: *r}

	// callback before
	if r.cb != nil && r.cb.before != nil {
		r.cb.before(r.Path, r.Vars)
	}

	start := time.Now()

	absPath, err := filepath.Abs(r.Path)
	if err != nil {
		return result, fmt.Errorf("failed to resolve path: %w", err)
	}
	job, err := loadJob(absPath)
	if err != nil {
		return result, err
	}

	// A local action in the job is found next to the job file, as one in a
	// workflow is found next to the workflow file.
	jobID := "embedded"
	printer := probe.NewPrinter(true, []string{jobID})
	run := job.RunStandalone(r.Vars, printer, jobID, filepath.Dir(absPath))

	result.RT = time.Since(start)

	code := 0
	errorMsg := ""
	if !run.Success {
		code = 1
		// The message res.error has always carried for a failed step.
		errorMsg = "execution error in job_start: job execution failed"
	}
	if run.Err != nil {
		errorMsg = run.Err.Error()
	}

	result.Res = Res{
		Code:    code,
		Outputs: run.Outputs,
		Report:  run.Report,
		Error:   errorMsg,
		Dump:    false, // Don't dump request/response for embedded jobs
	}
	result.Status = code

	// callback after
	if r.cb != nil && r.cb.after != nil {
		r.cb.after(result)
	}

	// A step that fails is the embedded job's result, for the embedding step
	// to test. Only a job that could not run, such as one with an invalid
	// step, is an error.
	if run.Err != nil {
		detailedError := fmt.Sprintf("embedded job execution failed: %s", run.Err)
		if run.Report != "" {
			detailedError += "\nEmbedded job details:\n" + run.Report
		}
		return result, errors.New(detailedError)
	}

	return result, nil
}

// loadJob reads the job file at path and applies the job's defaults to its
// steps.
func loadJob(path string) (*probe.Job, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("embedded steps file does not exist: %s", path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded steps file: %w", err)
	}

	job := &probe.Job{}
	dec := yaml.NewDecoder(bytes.NewReader(data), yaml.Validator(validator.New()), yaml.AllowDuplicateMapKey())
	if err := dec.Decode(job); err != nil {
		return nil, fmt.Errorf("failed to decode YAML job: %w", err)
	}
	if len(job.Steps) == 0 {
		return nil, fmt.Errorf("no steps found in embedded file: %s", path)
	}

	job.ApplyDefaults()
	return job, nil
}

func Execute(data map[string]any, opts ...Option) (map[string]any, error) {
	// Create a copy to avoid modifying the original data
	dataCopy := make(map[string]any)
	maps.Copy(dataCopy, data)

	m := mapping.HeaderToStringValue(dataCopy)

	r := NewReq()

	cb := &Callback{}
	for _, opt := range opts {
		opt(cb)
	}
	r.cb = cb

	if err := mapping.MapToStructByTags(m, r); err != nil {
		return map[string]any{}, err
	}

	result, err := r.Do()
	if err != nil {
		// Even on error, try to return a structured result if we have one
		if result != nil {
			mapResult, mapErr := mapping.StructToMapByTags(result)
			if mapErr == nil {
				return mapResult, err
			}
		}
		return map[string]any{}, err
	}

	mapResult, err := mapping.StructToMapByTags(result)
	if err != nil {
		return map[string]any{}, err
	}

	return mapResult, nil
}

func WithBefore(f func(path string, vars map[string]any)) Option {
	return func(c *Callback) {
		c.before = f
	}
}

func WithAfter(f func(result *Result)) Option {
	return func(c *Callback) {
		c.after = f
	}
}

package embedded

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/linyows/probe"
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

	// Resolve absolute path
	absPath, err := filepath.Abs(r.Path)
	if err != nil {
		return result, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return result, fmt.Errorf("embedded steps file does not exist: %s", absPath)
	}

	// Read embedded steps file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return result, fmt.Errorf("failed to read embedded steps file: %w", err)
	}

	// Parse YAML job
	job := &probe.Job{}
	v := validator.New()
	dec := yaml.NewDecoder(bytes.NewReader(data), yaml.Validator(v))
	if err = dec.Decode(job); err != nil {
		return result, fmt.Errorf("failed to decode YAML job: %w", err)
	}

	if len(job.Steps) == 0 {
		return result, fmt.Errorf("no steps found in embedded file: %s", absPath)
	}

	// Apply defaults to steps if they exist
	applyDefaultsToSteps(job)

	// Execute job independently
	jobID := "embedded"
	printer := probe.NewPrinter(true, []string{jobID})
	success, outputs, report, errorMsg, _ := job.RunIndependently(r.Vars, printer, jobID)

	result.RT = time.Since(start)

	code := 0
	if !success {
		code = 1
	}

	result.Res = Res{
		Code:    code,
		Outputs: outputs,
		Report:  report,
		Error:   errorMsg,
	}
	result.Status = code

	// callback after
	if r.cb != nil && r.cb.after != nil {
		r.cb.after(result)
	}

	// Return error if execution failed
	if !success && errorMsg != "" {
		return result, fmt.Errorf("embedded job execution failed: %s", errorMsg)
	}

	return result, nil
}

// applyDefaultsToSteps applies defaults from job to steps
func applyDefaultsToSteps(job *probe.Job) {
	if job.Defaults == nil {
		return
	}

	dataMap, ok := job.Defaults.(map[string]any)
	if !ok {
		return
	}

	for key, values := range dataMap {
		defaults, defok := values.(map[string]any)
		if !defok {
			continue
		}

		for _, s := range job.Steps {
			if s.Uses != key {
				continue
			}

			if s.With == nil {
				s.With = make(map[string]any)
			}

			applyDefaults(s.With, defaults)
		}
	}
}

// applyDefaults recursively applies default values
func applyDefaults(data, defaults map[string]any) {
	for key, defaultValue := range defaults {
		// If key does not exist in data
		if _, exists := data[key]; !exists {
			data[key] = defaultValue
			continue
		}

		// If you have a nested map with a key of data
		if nestedDefault, ok := defaultValue.(map[string]any); ok {
			if nestedData, ok := data[key].(map[string]any); ok {
				// Recursively set default values
				applyDefaults(nestedData, nestedDefault)
			}
		}
	}
}

func Execute(data map[string]string, opts ...Option) (map[string]string, error) {
	// Create a copy to avoid modifying the original data
	dataCopy := make(map[string]string)
	for k, v := range data {
		dataCopy[k] = v
	}

	m := probe.HeaderToStringValue(probe.UnflattenInterface(dataCopy))

	r := NewReq()

	cb := &Callback{}
	for _, opt := range opts {
		opt(cb)
	}
	r.cb = cb

	if err := probe.MapToStructByTags(m, r); err != nil {
		return map[string]string{}, err
	}

	result, err := r.Do()
	if err != nil {
		// Even on error, try to return a structured result if we have one
		if result != nil {
			mapResult, mapErr := probe.StructToMapByTags(result)
			if mapErr == nil {
				return probe.FlattenInterface(mapResult), err
			}
		}
		return map[string]string{}, err
	}

	mapResult, err := probe.StructToMapByTags(result)
	if err != nil {
		return map[string]string{}, err
	}

	return probe.FlattenInterface(mapResult), nil
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

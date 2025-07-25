package probe

import (
	"sync"
)

// SharedResults manages step results across the entire workflow
type SharedResults struct {
	data map[string]map[string]any  // stepID -> resultName -> value
	mu   sync.RWMutex
}

// NewSharedResults creates a new SharedResults instance
func NewSharedResults() *SharedResults {
	return &SharedResults{
		data: make(map[string]map[string]any),
	}
}

// Set stores results for a step
func (sr *SharedResults) Set(stepID string, results map[string]any) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.data[stepID] = results
}

// Get retrieves results for a step
func (sr *SharedResults) Get(stepID string) (map[string]any, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	results, exists := sr.data[stepID]
	return results, exists
}

// GetAll returns all results (safe copy for expression evaluation)
func (sr *SharedResults) GetAll() map[string]map[string]any {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	copy := make(map[string]map[string]any)
	for k, v := range sr.data {
		copyResults := make(map[string]any)
		for rk, rv := range v {
			copyResults[rk] = rv
		}
		copy[k] = copyResults
	}
	return copy
}

type Workflow struct {
	Name        string         `yaml:"name" validate:"required"`
	Description string         `yaml:"description,omitempty"`
	Jobs        []Job          `yaml:"jobs" validate:"required"`
	Vars        map[string]any `yaml:"vars"`
	exitStatus  int
	env         map[string]string
	// Shared results across all jobs
	sharedResults *SharedResults
}

func (w *Workflow) SetExitStatus(isErr bool) {
	if isErr {
		w.exitStatus = 1
	}
}

func (w *Workflow) Env() map[string]string {
	if len(w.env) == 0 {
		w.env = EnvMap()
	}
	return w.env
}

func (w *Workflow) evalVars() (map[string]any, error) {
	env := StrmapToAnymap(w.Env())
	vars := make(map[string]any)

	expr := &Expr{}
	for k, v := range w.Vars {
		if mapV, ok := v.(map[string]any); ok {
			vars[k] = expr.EvalTemplateMap(mapV, env)
		} else if strV, ok2 := v.(string); ok2 {
			output, err := expr.EvalTemplate(strV, env)
			if err != nil {
				return vars, err
			}
			vars[k] = output
		}
	}

	return vars, nil
}

func (w *Workflow) newJobContext(c Config, vars map[string]any) JobContext {
	return JobContext{
		Vars:          vars,
		Logs:          []map[string]any{},
		Config:        c,
		Output:        c.Output,
		SharedResults: w.sharedResults,
	}
}
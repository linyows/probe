package probe

import (
	"sync"
)

// Outputs manages step outputs across the entire workflow
type Outputs struct {
	data map[string]map[string]any  // stepID -> outputName -> value
	mu   sync.RWMutex
}

// NewOutputs creates a new Outputs instance
func NewOutputs() *Outputs {
	return &Outputs{
		data: make(map[string]map[string]any),
	}
}

// Set stores outputs for a step
func (o *Outputs) Set(stepID string, outputs map[string]any) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.data[stepID] = outputs
}

// Get retrieves outputs for a step
func (o *Outputs) Get(stepID string) (map[string]any, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	outputs, exists := o.data[stepID]
	return outputs, exists
}

// GetAll returns all outputs (safe copy for expression evaluation)
func (o *Outputs) GetAll() map[string]map[string]any {
	o.mu.RLock()
	defer o.mu.RUnlock()
	copy := make(map[string]map[string]any)
	for k, v := range o.data {
		copyOutputs := make(map[string]any)
		for rk, rv := range v {
			copyOutputs[rk] = rv
		}
		copy[k] = copyOutputs
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
	// Shared outputs across all jobs
	sharedOutputs *Outputs
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
		Printer:        c.Printer,
		Outputs: w.sharedOutputs,
	}
}
package probe

import (
	"sync"
)

// Outputs manages step outputs across the entire workflow
type Outputs struct {
	data map[string]map[string]any // stepID -> outputName -> value
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

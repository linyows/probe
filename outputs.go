package probe

import (
	"fmt"
	"sync"
)

// Outputs manages step outputs across the entire workflow
type Outputs struct {
	data map[string]any // stores both stepID->outputs and outputName->value
	mu   sync.RWMutex
}

// NewOutputs creates a new Outputs instance
func NewOutputs() *Outputs {
	return &Outputs{
		data: make(map[string]any),
	}
}

// Set stores outputs for a step with flat access support
func (o *Outputs) Set(stepID string, outputs map[string]any) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Check if stepID conflicts with existing flat data
	stepIDConflictsWithFlat := false
	var conflictError error
	if existingValue, exists := o.data[stepID]; exists {
		if _, isMap := existingValue.(map[string]any); !isMap {
			// stepID conflicts with existing flat data - this will prevent step-based access
			stepIDConflictsWithFlat = true
			conflictError = fmt.Errorf("cannot create step-based outputs for '%s' because flat output with same name exists", stepID)
		}
	}

	// Store step-based outputs only if no conflict with flat data
	if !stepIDConflictsWithFlat {
		o.data[stepID] = outputs
	}

	// Store flat outputs if no conflicts (new functionality)
	for outputName, value := range outputs {
		// Skip if output name already exists
		if _, exists := o.data[outputName]; exists {
			continue
		}

		// Safe to store flat access
		o.data[outputName] = value
	}

	return conflictError
}

// Get retrieves outputs for a step (existing functionality)
func (o *Outputs) Get(stepID string) (map[string]any, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	value, exists := o.data[stepID]
	if !exists {
		return nil, false
	}
	
	if outputs, ok := value.(map[string]any); ok {
		return outputs, true
	}
	
	return nil, false
}

// GetFlat retrieves output by name directly (new functionality)
func (o *Outputs) GetFlat(outputName string) (any, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	value, exists := o.data[outputName]
	if !exists {
		return nil, false
	}
	
	// If it's a map[string]any, it's step-based data, not flat data
	if _, isMap := value.(map[string]any); isMap {
		return nil, false
	}
	
	return value, true
}

// GetAll returns all outputs (safe copy for expression evaluation)
func (o *Outputs) GetAll() map[string]any {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	copy := make(map[string]any)
	
	for k, v := range o.data {
		if stepOutputs, ok := v.(map[string]any); ok {
			// This is step-based data, create a deep copy
			copyOutputs := make(map[string]any)
			for rk, rv := range stepOutputs {
				copyOutputs[rk] = rv
			}
			copy[k] = copyOutputs
		} else {
			// This is flat data, copy directly
			copy[k] = v
		}
	}
	
	return copy
}

// GetAllWithFlat returns all outputs including flat access for expression evaluation
func (o *Outputs) GetAllWithFlat() map[string]any {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	copy := make(map[string]any)
	
	for k, v := range o.data {
		if stepOutputs, ok := v.(map[string]any); ok {
			// This is step-based data, create a deep copy
			copyOutputs := make(map[string]any)
			for rk, rv := range stepOutputs {
				copyOutputs[rk] = rv
			}
			copy[k] = copyOutputs
		} else {
			// This is flat data, copy directly
			copy[k] = v
		}
	}
	
	return copy
}

// GetConflicts returns information about conflicted output names
func (o *Outputs) GetConflicts() map[string][]string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	conflicts := make(map[string][]string)
	outputUsage := make(map[string][]string)

	// Scan all step-based outputs
	for stepID, value := range o.data {
		if stepOutputs, ok := value.(map[string]any); ok {
			for outputName := range stepOutputs {
				outputUsage[outputName] = append(outputUsage[outputName], stepID)
			}
		}
	}

	// Find conflicts (output names used by multiple steps)
	for outputName, stepIDs := range outputUsage {
		if len(stepIDs) > 1 {
			conflicts[outputName] = stepIDs
		}
	}

	return conflicts
}

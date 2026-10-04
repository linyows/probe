package probe

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"sync"
)

// Outputs manages step outputs across the entire workflow
type Outputs struct {
	data map[string]any // stores both stepID->outputs and outputName->value
	// owners records which step published each flat output name.
	owners map[string]string
	// ambiguous holds the names that more than one step published, with
	// those steps. Such a name has no flat value: which step's it would be
	// depends on the order the steps ran in, and for jobs that run at the
	// same time that order is not fixed.
	ambiguous map[string][]string
	mu        sync.RWMutex
}

// NewOutputs creates a new Outputs instance
func NewOutputs() *Outputs {
	return &Outputs{
		data:      make(map[string]any),
		owners:    make(map[string]string),
		ambiguous: make(map[string][]string),
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

	// Store flat outputs, by name alone
	var errs []error
	if conflictError != nil {
		errs = append(errs, conflictError)
	}
	names := make([]string, 0, len(outputs))
	for outputName := range outputs {
		names = append(names, outputName)
	}
	sort.Strings(names)
	for _, outputName := range names {
		value := outputs[outputName]

		if steps, ok := o.ambiguous[outputName]; ok {
			if !slices.Contains(steps, stepID) {
				o.ambiguous[outputName] = append(steps, stepID)
			}
			continue
		}

		owner, owned := o.owners[outputName]
		switch {
		case owned && owner == stepID:
			// The same step publishing again is not a second publisher: the
			// flat name follows its latest value, as outputs.<step_id>.<name>
			// does.
			o.data[outputName] = value
		case owned:
			// Another step publishes the name too, so the name alone no
			// longer says which value is meant.
			delete(o.data, outputName)
			delete(o.owners, outputName)
			o.ambiguous[outputName] = []string{owner, stepID}
			errs = append(errs, fmt.Errorf("output '%s' is published by both '%s' and '%s', so outputs.%s is not set; read outputs.%s.%s or outputs.%s.%s instead",
				outputName, owner, stepID, outputName, owner, outputName, stepID, outputName))
		default:
			// A step ID of the same name keeps the name for its outputs.
			if _, exists := o.data[outputName]; exists {
				continue
			}
			o.data[outputName] = value
			o.owners[outputName] = stepID
		}
	}

	return errors.Join(errs...)
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
			maps.Copy(copyOutputs, stepOutputs)
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
			maps.Copy(copyOutputs, stepOutputs)
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

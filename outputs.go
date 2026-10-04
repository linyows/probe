package probe

import (
	"errors"
	"fmt"
	"maps"
	"sort"
	"sync"
)

// Outputs manages step outputs across the entire workflow
type Outputs struct {
	data map[string]any // stores both stepID->outputs and outputName->value
	// owners records which step published each flat output name. The first
	// step to publish a name keeps it.
	owners map[string]string
	mu     sync.RWMutex
}

// NewOutputs creates a new Outputs instance
func NewOutputs() *Outputs {
	return &Outputs{
		data:   make(map[string]any),
		owners: make(map[string]string),
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

	// Store flat outputs. The first step to publish a name keeps it; a later
	// step's value under the same name is read through its step id, and is
	// reported so that it is not lost without a word.
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
		owner, owned := o.owners[outputName]
		_, isStep := o.data[outputName].(map[string]any)

		// Where this value can still be read when the name alone is taken:
		// through the step's id, unless that was taken as well above.
		instead := fmt.Sprintf("read outputs.%s.%s for this one", stepID, outputName)
		if stepIDConflictsWithFlat {
			instead = fmt.Sprintf("this one cannot be read, since outputs.%s is an output name too", stepID)
		}
		switch {
		case owned && owner == stepID:
			// The same step publishing again keeps the name, with its
			// latest value, as outputs.<step_id>.<name> has.
			o.data[outputName] = outputs[outputName]
		case owned:
			errs = append(errs, &NameTakenError{fmt.Sprintf("output '%s' of '%s' is also published by '%s', which came first, so outputs.%s keeps the value of '%s'; %s",
				outputName, stepID, owner, outputName, owner, instead)})
		case isStep:
			// A step ID of the same name keeps the name for its outputs.
			errs = append(errs, &NameTakenError{fmt.Sprintf("output '%s' of '%s' has the name of the step '%s', so outputs.%s is that step's outputs; %s",
				outputName, stepID, outputName, outputName, instead)})
		default:
			o.data[outputName] = outputs[outputName]
			o.owners[outputName] = stepID
		}
	}

	return errors.Join(errs...)
}

// NameTakenError reports an output whose name was taken before it was
// published, so that outputs.<name> holds something else. It is a warning:
// the step's value is kept under its id when that can be.
type NameTakenError struct {
	msg string
}

func (e *NameTakenError) Error() string {
	return e.msg
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

package probe

import (
	"fmt"
)

type JobContext struct {
	Vars map[string]any   `expr:"vars"`
	Logs []map[string]any `expr:"steps"`
	Config
	Failed bool
	// Current job ID for this context
	CurrentJobID string
	// Repeat tracking
	IsRepeating   bool
	RepeatCurrent int
	RepeatTotal   int
	StepCounters  map[int]StepRepeatCounter // step index -> counter
	// Print writer
	Printer PrintWriter
	// Result for managing job-level output
	Result *Result
	// Job scheduler for managing job dependencies and execution
	JobScheduler *JobScheduler
	// Shared outputs across all jobs (accessible via expressions as "outputs")
	Outputs *Outputs `expr:"outputs"`
}

func (j *JobContext) SetFailed() {
	j.Failed = true
}

type Job struct {
	Name     string   `yaml:"name" validate:"required"`
	ID       string   `yaml:"id,omitempty"`
	Needs    []string `yaml:"needs,omitempty"`
	Steps    []*Step  `yaml:"steps" validate:"required"`
	Repeat   *Repeat  `yaml:"repeat"`
	Defaults any      `yaml:"defaults"`
	ctx      *JobContext
}

func (j *Job) Start(ctx JobContext) error {
	// Set current job ID in context (already set by Executor.setJobID())
	ctx.CurrentJobID = j.ID

	j.ctx = &ctx
	expr := &Expr{}

	// Validate steps before execution
	if err := j.validateSteps(); err != nil {
		return NewExecutionError("job_start", "step validation failed", err)
	}

	if err := j.expandJobName(expr, ctx); err != nil {
		return NewExecutionError("job_start", "failed to expand job name", err)
	}

	j.executeSteps(expr, ctx)
	if j.ctx.Failed {
		return NewExecutionError("job_start", "job execution failed", nil).
			WithContext("job_name", j.Name)
	}
	return nil
}

// expandJobName evaluates and sets the job name, printing it if appropriate
func (j *Job) expandJobName(expr *Expr, ctx JobContext) error {
	if j.Name == "" {
		j.Name = "Unknown Job"
		return nil
	}

	name, err := expr.EvalTemplate(j.Name, ctx)
	if err != nil {
		return err
	}

	j.Name = name
	return nil
}

// executeSteps runs all steps in the job, handling iterations appropriately
func (j *Job) executeSteps(expr *Expr, ctx JobContext) {
	idx := 0
	for _, st := range j.Steps {
		st.expr = expr

		if len(st.Iter) == 0 {
			j.executeStep(st, &idx, *j.ctx, nil)
		} else {
			j.executeStepWithIterations(st, &idx, *j.ctx)
		}
	}
}

// executeStep executes a single step without iterations
func (j *Job) executeStep(st *Step, idx *int, ctx JobContext, vars map[string]any) {
	st.idx = *idx
	*idx++
	st.SetCtx(ctx, vars)
	st.Do(j.ctx)
}

// executeStepWithIterations executes a step multiple times with different variable sets
func (j *Job) executeStepWithIterations(st *Step, idx *int, ctx JobContext) {
	for _, vars := range st.Iter {
		j.executeStep(st, idx, ctx, vars)
	}
}

// validateSteps validates step configurations in the job
func (j *Job) validateSteps() error {
	stepIDs := make(map[string]int) // stepID -> stepIndex for duplicate check

	for i, step := range j.Steps {
		// Validate step ID format if provided
		if step.ID != "" {
			if !isValidStepID(step.ID) {
				return fmt.Errorf("step %d: invalid step ID '%s' - only [a-z0-9_-] characters are allowed", i, step.ID)
			}

			// Check for duplicate step IDs within the job
			if existingIndex, exists := stepIDs[step.ID]; exists {
				return fmt.Errorf("step %d: duplicate step ID '%s' (already used in step %d)", i, step.ID, existingIndex)
			}
			stepIDs[step.ID] = i
		}

		// Validate that outputs require an ID
		if len(step.Outputs) > 0 && step.ID == "" {
			stepName := step.Name
			if stepName == "" {
				stepName = fmt.Sprintf("step %d", i)
			}
			return fmt.Errorf("%s: step with outputs must have an 'id' field", stepName)
		}
	}

	return nil
}

// isValidStepID validates step ID format: only [a-z0-9_-] allowed
func isValidStepID(id string) bool {
	if id == "" {
		return false
	}

	// Check each character
	for _, char := range id {
		if !((char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-') {
			return false
		}
	}

	return true
}

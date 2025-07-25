package probe

import (
	"fmt"
	"sync"
	"time"
)

type JobStatus int

const (
	JobPending JobStatus = iota
	JobRunning
	JobCompleted
	JobFailed
)

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
	j.ctx = &ctx
	expr := &Expr{}

	// Validate steps before execution
	if err := j.validateSteps(); err != nil {
		return NewExecutionError("job_start", "step validation failed", err)
	}

	if err := j.processJobName(expr, ctx); err != nil {
		return NewExecutionError("job_start", "failed to process job name", err)
	}

	j.executeSteps(expr, ctx)
	if j.ctx.Failed {
		return NewExecutionError("job_start", "job execution failed", nil).
			WithContext("job_name", j.Name)
	}
	return nil
}

// processJobName evaluates and sets the job name, printing it if appropriate
func (j *Job) processJobName(expr *Expr, ctx JobContext) error {
	if j.Name == "" {
		j.Name = "Unknown Job"
	}

	name, err := expr.EvalTemplate(j.Name, ctx)
	if err != nil {
		ctx.Printer.PrintError("job name evaluation error: %v", err)
		return err
	}

	j.Name = name
	// Only print job name if not repeating and not using buffering (to avoid duplicate output)
	if !ctx.IsRepeating && !ctx.UseBuffering {
		ctx.Printer.PrintJobName(name)
	}

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

type JobContext struct {
	Vars map[string]any   `expr:"vars"`
	Logs []map[string]any `expr:"steps"`
	Config
	Failed bool
	// Repeat tracking
	IsRepeating   bool
	RepeatCurrent int
	RepeatTotal   int
	StepCounters  map[int]StepRepeatCounter // step index -> counter
	// Output buffering
	UseBuffering bool
	// Print writer
	Printer PrintWriter
	// Step results storage: stepID -> results map
	Results map[string]map[string]any `expr:"results"`
	// Shared results across all jobs (pointer to workflow results)
	SharedResults *SharedResults
}

func (j *JobContext) SetFailed() {
	j.Failed = true
}

type JobScheduler struct {
	jobs           map[string]*Job
	status         map[string]JobStatus
	results        map[string]bool
	repeatCounters map[string]int // Track repeat execution count
	repeatTargets  map[string]int // Target repeat count
	mutex          sync.RWMutex
	wg             sync.WaitGroup
}

func NewJobScheduler() *JobScheduler {
	return &JobScheduler{
		jobs:           make(map[string]*Job),
		status:         make(map[string]JobStatus),
		results:        make(map[string]bool),
		repeatCounters: make(map[string]int),
		repeatTargets:  make(map[string]int),
	}
}

func (js *JobScheduler) AddJob(job *Job) error {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	// Generate unique ID if not provided
	if job.ID == "" {
		job.ID = js.generateUniqueID(job.Name)
	}

	// Check for duplicate IDs
	if _, exists := js.jobs[job.ID]; exists {
		return fmt.Errorf("duplicate job ID: %s", job.ID)
	}

	js.jobs[job.ID] = job
	js.status[job.ID] = JobPending

	// Set repeat targets
	if job.Repeat != nil {
		js.repeatTargets[job.ID] = job.Repeat.Count
		js.repeatCounters[job.ID] = 0
	} else {
		js.repeatTargets[job.ID] = 1 // No repeat = run once
		js.repeatCounters[job.ID] = 0
	}

	return nil
}

// generateUniqueID generates a unique ID based on the job name
// If the name is already taken, it appends a number to make it unique
func (js *JobScheduler) generateUniqueID(baseName string) string {
	if baseName == "" {
		baseName = "job"
	}

	// First try the base name
	if _, exists := js.jobs[baseName]; !exists {
		return baseName
	}

	// If base name exists, try with incrementing numbers
	counter := 1
	for {
		candidateID := fmt.Sprintf("%s-%d", baseName, counter)
		if _, exists := js.jobs[candidateID]; !exists {
			return candidateID
		}
		counter++

		// Safety check to prevent infinite loop (though very unlikely)
		if counter > 10000 {
			return fmt.Sprintf("%s-%d", baseName, int(time.Now().UnixNano()))
		}
	}
}

func (js *JobScheduler) ValidateDependencies() error {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	// Check if all dependencies exist
	for jobID, job := range js.jobs {
		for _, dep := range job.Needs {
			if _, exists := js.jobs[dep]; !exists {
				return fmt.Errorf("job '%s' depends on non-existent job '%s'", jobID, dep)
			}
		}
	}

	// Check for circular dependencies
	return js.checkCircularDependencies()
}

func (js *JobScheduler) checkCircularDependencies() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for jobID := range js.jobs {
		if !visited[jobID] {
			if js.hasCycleDFS(jobID, visited, recStack) {
				return fmt.Errorf("circular dependency detected involving job '%s'", jobID)
			}
		}
	}

	return nil
}

func (js *JobScheduler) hasCycleDFS(jobID string, visited, recStack map[string]bool) bool {
	visited[jobID] = true
	recStack[jobID] = true

	job := js.jobs[jobID]
	for _, dep := range job.Needs {
		if !visited[dep] {
			if js.hasCycleDFS(dep, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	recStack[jobID] = false
	return false
}

func (js *JobScheduler) CanRunJob(jobID string) bool {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	job := js.jobs[jobID]
	if js.status[jobID] != JobPending {
		return false
	}

	// Check if all dependencies are fully completed (all repeats done)
	for _, dep := range job.Needs {
		if !js.isJobFullyCompleted(dep) {
			return false
		}
	}

	return true
}

// isJobFullyCompleted checks if a job has completed all its repeat executions
func (js *JobScheduler) isJobFullyCompleted(jobID string) bool {
	if js.status[jobID] != JobCompleted {
		return false
	}

	// Check if all repeat executions are done
	target := js.repeatTargets[jobID]
	counter := js.repeatCounters[jobID]

	return counter >= target && js.results[jobID]
}

func (js *JobScheduler) SetJobStatus(jobID string, status JobStatus, success bool) {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	js.status[jobID] = status
	if status == JobCompleted || status == JobFailed {
		js.results[jobID] = success
	}
}

// IncrementRepeatCounter increments the repeat counter for a job
func (js *JobScheduler) IncrementRepeatCounter(jobID string) {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	js.repeatCounters[jobID]++
}

// ShouldRepeatJob checks if a job should be repeated
func (js *JobScheduler) ShouldRepeatJob(jobID string) bool {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	counter := js.repeatCounters[jobID]
	target := js.repeatTargets[jobID]

	return counter < target
}

// GetRepeatInfo returns current repeat counter and target for a job
func (js *JobScheduler) GetRepeatInfo(jobID string) (current, target int) {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	return js.repeatCounters[jobID], js.repeatTargets[jobID]
}

func (js *JobScheduler) GetRunnableJobs() []string {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	var runnable []string
	for jobID := range js.jobs {
		if js.CanRunJob(jobID) {
			runnable = append(runnable, jobID)
		}
	}

	return runnable
}

func (js *JobScheduler) AllJobsCompleted() bool {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	for _, status := range js.status {
		if status != JobCompleted && status != JobFailed {
			return false
		}
	}

	return true
}

// MarkJobsWithFailedDependencies marks jobs as failed if their dependencies have failed
func (js *JobScheduler) MarkJobsWithFailedDependencies() []string {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	var skippedJobs []string

	for jobID, job := range js.jobs {
		if js.status[jobID] != JobPending {
			continue
		}

		// Check if any dependency has failed
		hasFailedDependency := false
		for _, dep := range job.Needs {
			if js.status[dep] == JobCompleted && !js.results[dep] {
				hasFailedDependency = true
				break
			}
		}

		if hasFailedDependency {
			js.status[jobID] = JobFailed
			js.results[jobID] = false
			skippedJobs = append(skippedJobs, jobID)
		}
	}

	return skippedJobs
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

		// Validate that results require an ID
		if len(step.Results) > 0 && step.ID == "" {
			stepName := step.Name
			if stepName == "" {
				stepName = fmt.Sprintf("step %d", i)
			}
			return fmt.Errorf("%s: step with results must have an 'id' field", stepName)
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

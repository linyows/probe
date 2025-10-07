package probe

import (
	"sync"
	"time"
)

// Executor handles job execution with buffered output
type Executor struct {
	workflow *Workflow
	job      *Job
}

// NewExecutor creates a new job executor
func NewExecutor(w *Workflow, job *Job) *Executor {
	return &Executor{
		workflow: w,
		job:      job,
	}
}

// setJobID ensures the job has a valid ID, using Name if ID is empty
func (e *Executor) setJobID() {
	if e.job.ID == "" {
		e.job.ID = e.job.Name
	}
}

// Execute runs a job with buffered output
func (e *Executor) Execute(ctx JobContext) bool {
	e.setJobID()
	jobID := e.job.ID

	jr := ctx.Result.Jobs[jobID]

	jr.mutex.Lock()
	jr.StartTime = time.Now()
	jr.Status = "Running"
	jr.mutex.Unlock()

	// Use the existing buffered execution logic
	return e.executeWithBuffering(ctx)
}

// executeWithBuffering contains the buffered execution logic
func (e *Executor) executeWithBuffering(ctx JobContext) bool {
	ctx = e.setupContext(ctx)

	overallSuccess := e.executeJobRepeatLoop(ctx)

	if ctx.IsRepeating {
		e.appendRepeatStepResults(&ctx)
	}

	e.finalize(overallSuccess, ctx)

	return overallSuccess
}

// setupContext initializes the job context for buffered execution
func (e *Executor) setupContext(ctx JobContext) JobContext {
	jobID := e.job.ID
	_, total := ctx.JobScheduler.GetRepeatInfo(jobID)
	ctx.IsRepeating = total > 1
	ctx.RepeatTotal = total
	ctx.StepCounters = make(map[int]StepRepeatCounter)
	return ctx
}

// executeJobRepeatLoop handles the main execution loop with repeat logic
func (e *Executor) executeJobRepeatLoop(ctx JobContext) bool {
	// Check if async execution is enabled
	if e.job.Repeat != nil && e.job.Repeat.Async {
		return e.executeJobRepeatLoopAsync(ctx)
	}

	// Synchronous execution (original behavior)
	jobID := e.job.ID
	overallSuccess := true

	for ctx.JobScheduler.ShouldRepeatJob(jobID) {
		current, _ := ctx.JobScheduler.GetRepeatInfo(jobID)
		ctx.RepeatCurrent = current

		// Execute single run
		err := e.job.Start(ctx)

		if err != nil {
			overallSuccess = false
			e.workflow.SetExitStatus(true)
		}

		ctx.JobScheduler.IncrementRepeatCounter(jobID)
		e.sleepBetweenRepeats(ctx)
	}

	return overallSuccess
}

// executeJobRepeatLoopAsync handles async execution loop with repeat logic
func (e *Executor) executeJobRepeatLoopAsync(ctx JobContext) bool {
	jobID := e.job.ID
	_, total := ctx.JobScheduler.GetRepeatInfo(jobID)

	var wg sync.WaitGroup
	var mu sync.Mutex
	overallSuccess := true

	interval := e.job.Repeat.Interval.Duration
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for ctx.JobScheduler.ShouldRepeatJob(jobID) {
		current, _ := ctx.JobScheduler.GetRepeatInfo(jobID)

		wg.Add(1)
		go func(repeatIndex int) {
			defer wg.Done()

			// Create a copy of context for this goroutine
			execCtx := ctx
			execCtx.RepeatCurrent = repeatIndex

			// Execute single run
			err := e.job.Start(execCtx)

			if err != nil {
				mu.Lock()
				overallSuccess = false
				e.workflow.SetExitStatus(true)
				mu.Unlock()
			}
		}(current)

		ctx.JobScheduler.IncrementRepeatCounter(jobID)

		// Wait for interval before next execution (except for the last one)
		if current+1 < total {
			<-ticker.C
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return overallSuccess
}

// sleepBetweenRepeats handles the interval sleep between job repetitions
func (e *Executor) sleepBetweenRepeats(ctx JobContext) {
	jobID := e.job.ID
	current, target := ctx.JobScheduler.GetRepeatInfo(jobID)
	if current < target && e.job.Repeat != nil {
		time.Sleep(e.job.Repeat.Interval.Duration)
	}
}

// finalize updates the final job status and marks it as completed
func (e *Executor) finalize(overallSuccess bool, ctx JobContext) {
	jobID := e.job.ID
	jr := ctx.Result.Jobs[jobID]
	jr.mutex.Lock()
	duration := time.Since(jr.StartTime)
	jr.EndTime = jr.StartTime.Add(duration)
	jr.Success = overallSuccess
	if overallSuccess {
		// Don't overwrite "skipped" status
		if jr.Status != "skipped" {
			jr.Status = "Completed"
		}
	} else {
		jr.Status = "Failed"
	}
	jr.mutex.Unlock()

	// Mark job as completed
	ctx.JobScheduler.SetJobStatus(jobID, JobCompleted, overallSuccess)
}

// appendRepeatStepResults appends the final results of repeat step executions to buffer
func (e *Executor) appendRepeatStepResults(ctx *JobContext) {
	jobID := e.job.ID

	// Create StepResults for repeat steps and add them to Result
	for i, step := range e.job.Steps {
		if counter, exists := ctx.StepCounters[i]; exists {
			hasTest := step.Test != ""

			// Determine status based on repeat counter results
			var status StatusType
			if hasTest {
				if counter.FailureCount == 0 {
					status = StatusSuccess
				} else if counter.SuccessCount == 0 {
					status = StatusError
				} else {
					status = StatusWarning
				}
			} else {
				status = StatusWarning // No test, so warning status
			}

			// Create StepResult with repeat counter
			stepResult := StepResult{
				Index:         i,
				Name:          step.Name,
				Status:        status,
				HasTest:       hasTest,
				RepeatCounter: &counter,
			}

			// Add step result to workflow buffer
			if ctx.Result != nil {
				ctx.Result.AddStepResult(jobID, stepResult)
			}
		}
	}
}

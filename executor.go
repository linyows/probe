package probe

import (
	"io"
	"os"
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
	
	jb := ctx.WorkflowBuffer.Jobs[jobID]

	jb.mutex.Lock()
	jb.StartTime = time.Now()
	jb.Status = "Running"
	jb.mutex.Unlock()

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
	jobID := e.job.ID
	jb := ctx.WorkflowBuffer.Jobs[jobID]
	overallSuccess := true

	for ctx.JobScheduler.ShouldRepeatJob(jobID) {
		current, _ := ctx.JobScheduler.GetRepeatInfo(jobID)
		ctx.RepeatCurrent = current

		// Serialize stdout redirection to prevent race conditions
		ctx.WorkflowBuffer.outputMutex.Lock()

		// Capture output by redirecting stdout temporarily
		originalStdout := os.Stdout
		r, wr, _ := os.Pipe()
		os.Stdout = wr

		// Execute single run
		err := e.job.Start(ctx)

		// Restore stdout and capture output
		wr.Close()
		os.Stdout = originalStdout

		ctx.WorkflowBuffer.outputMutex.Unlock()

		capturedOutput, _ := io.ReadAll(r)

		// Add captured output to buffer
		jb.mutex.Lock()
		jb.Buffer.Write(capturedOutput)
		jb.mutex.Unlock()

		if err != nil {
			overallSuccess = false
			e.workflow.SetExitStatus(true)
		}

		ctx.JobScheduler.IncrementRepeatCounter(jobID)
		e.sleepBetweenRepeats(ctx)
	}

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
	jb := ctx.WorkflowBuffer.Jobs[jobID]
	jb.mutex.Lock()
	duration := time.Since(jb.StartTime)
	jb.EndTime = jb.StartTime.Add(duration)
	jb.Success = overallSuccess
	if overallSuccess {
		jb.Status = "Completed"
	} else {
		jb.Status = "Failed"
	}
	jb.mutex.Unlock()

	// Mark job as completed
	ctx.JobScheduler.SetJobStatus(jobID, JobCompleted, overallSuccess)
}

// appendRepeatStepResults appends the final results of repeat step executions to buffer
func (e *Executor) appendRepeatStepResults(ctx *JobContext) {
	jobID := e.job.ID
	jb := ctx.WorkflowBuffer.Jobs[jobID]
	
	// Capture the step outputs output to buffer instead of printing directly
	originalStdout := os.Stdout
	r, wr, _ := os.Pipe()
	os.Stdout = wr

	for i, step := range e.job.Steps {
		if counter, exists := ctx.StepCounters[i]; exists {
			hasTest := step.Test != ""
			ctx.Printer.PrintStepRepeatResult(i, counter, hasTest)
		}
	}

	// Restore stdout and capture the step outputs
	wr.Close()
	os.Stdout = originalStdout

	capturedOutput, _ := io.ReadAll(r)

	// Add captured step outputs to job buffer
	jb.mutex.Lock()
	jb.Buffer.Write(capturedOutput)
	jb.mutex.Unlock()
}

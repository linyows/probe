package probe

import (
	"io"
	"os"
	"time"
)

// ExecutionResult represents the result of a job execution
type ExecutionResult struct {
	Success  bool
	Duration time.Duration
	Output   string
	Error    error
}

// ExecutionConfig contains configuration for job execution
type ExecutionConfig struct {
	HasDependencies bool
	WorkflowOutput  *WorkflowOutput
	JobScheduler    *JobScheduler
}

// JobExecutor defines the interface for executing jobs
type JobExecutor interface {
	Execute(job *Job, jobID string, ctx JobContext, config ExecutionConfig) ExecutionResult
}


// BufferedJobExecutor handles job execution with buffered output
type BufferedJobExecutor struct {
	workflow *Workflow
}

// NewBufferedJobExecutor creates a new buffered job executor
func NewBufferedJobExecutor(w *Workflow) *BufferedJobExecutor {
	return &BufferedJobExecutor{workflow: w}
}

// Execute runs a job with buffered output
func (e *BufferedJobExecutor) Execute(job *Job, jobID string, ctx JobContext, config ExecutionConfig) ExecutionResult {
	startTime := time.Now()

	jo := config.WorkflowOutput.Jobs[jobID]
	jo.mutex.Lock()
	jo.StartTime = startTime
	jo.Status = "Running"
	jo.mutex.Unlock()

	// Use the existing buffered execution logic
	result := e.executeWithBuffering(job, jobID, ctx, config)

	return ExecutionResult{
		Success:  result.Success,
		Duration: result.Duration,
		Output:   result.Output,
		Error:    result.Error,
	}
}

// executeWithBuffering contains the buffered execution logic
func (e *BufferedJobExecutor) executeWithBuffering(job *Job, jobID string, ctx JobContext, config ExecutionConfig) ExecutionResult {
	startTime := time.Now()
	jo := config.WorkflowOutput.Jobs[jobID]

	e.initializeJobForBuffering(jo, startTime)
	ctx = e.setupBufferedContext(ctx, jobID, config)

	overallSuccess := e.executeJobRepeatLoop(job, jobID, ctx, config, jo)

	if ctx.IsRepeating {
		e.printRepeatStepResults(&ctx, job, jo)
	}

	duration := e.finalizeJobExecution(jo, startTime, overallSuccess, jobID, config)

	return ExecutionResult{
		Success:  overallSuccess,
		Duration: duration,
		Output:   jo.Buffer.String(),
		Error:    nil,
	}
}

// initializeJobForBuffering sets up the job output for buffered execution
func (e *BufferedJobExecutor) initializeJobForBuffering(jo *JobOutput, startTime time.Time) {
	jo.mutex.Lock()
	jo.StartTime = startTime
	jo.Status = "Running"
	jo.mutex.Unlock()
}

// setupBufferedContext initializes the job context for buffered execution
func (e *BufferedJobExecutor) setupBufferedContext(ctx JobContext, jobID string, config ExecutionConfig) JobContext {
	_, total := config.JobScheduler.GetRepeatInfo(jobID)
	ctx.IsRepeating = total > 1
	ctx.RepeatTotal = total
	ctx.StepCounters = make(map[int]StepRepeatCounter)
	return ctx
}

// executeJobRepeatLoop handles the main execution loop with repeat logic
func (e *BufferedJobExecutor) executeJobRepeatLoop(job *Job, jobID string, ctx JobContext, config ExecutionConfig, jo *JobOutput) bool {
	overallSuccess := true

	for config.JobScheduler.ShouldRepeatJob(jobID) {
		current, _ := config.JobScheduler.GetRepeatInfo(jobID)
		ctx.RepeatCurrent = current

		// Serialize stdout redirection to prevent race conditions
		config.WorkflowOutput.outputMutex.Lock()

		// Capture output by redirecting stdout temporarily
		originalStdout := os.Stdout
		r, wr, _ := os.Pipe()
		os.Stdout = wr

		// Execute single run
		err := job.Start(ctx)

		// Restore stdout and capture output
		wr.Close()
		os.Stdout = originalStdout

		config.WorkflowOutput.outputMutex.Unlock()

		capturedOutput, _ := io.ReadAll(r)

		// Add captured output to buffer
		jo.mutex.Lock()
		jo.Buffer.Write(capturedOutput)
		jo.mutex.Unlock()

		if err != nil {
			overallSuccess = false
			e.workflow.SetExitStatus(true)
		}

		config.JobScheduler.IncrementRepeatCounter(jobID)
		e.sleepBetweenRepeats(jobID, job, config)
	}

	return overallSuccess
}

// executeJobIteration executes a single iteration of the job with output capture
//nolint:unused // Reserved for future use
func (e *BufferedJobExecutor) executeJobIteration(job *Job, ctx JobContext, config ExecutionConfig, jo *JobOutput) bool {
	// Serialize stdout redirection to prevent race conditions
	config.WorkflowOutput.outputMutex.Lock()
	defer config.WorkflowOutput.outputMutex.Unlock()

	capturedOutput := e.captureJobOutput(job, ctx)

	// Add captured output to buffer
	jo.mutex.Lock()
	jo.Buffer.Write(capturedOutput)
	jo.mutex.Unlock()

	// Return success (job.Start returns error on failure)
	return job.Start(ctx) == nil
}

// captureJobOutput captures the stdout output during job execution
//nolint:unused // Reserved for future use
func (e *BufferedJobExecutor) captureJobOutput(job *Job, ctx JobContext) []byte {
	// Capture output by redirecting stdout temporarily
	originalStdout := os.Stdout
	r, wr, _ := os.Pipe()
	os.Stdout = wr

	// Execute single run
	_ = job.Start(ctx) // Ignore error for output capture

	// Restore stdout and capture output
	wr.Close()
	os.Stdout = originalStdout

	capturedOutput, _ := io.ReadAll(r)
	return capturedOutput
}

// sleepBetweenRepeats handles the interval sleep between job repetitions
func (e *BufferedJobExecutor) sleepBetweenRepeats(jobID string, job *Job, config ExecutionConfig) {
	current, target := config.JobScheduler.GetRepeatInfo(jobID)
	if current < target && job.Repeat != nil {
		time.Sleep(job.Repeat.Interval.Duration)
	}
}

// finalizeJobExecution updates the final job status and marks it as completed
func (e *BufferedJobExecutor) finalizeJobExecution(jo *JobOutput, startTime time.Time, overallSuccess bool, jobID string, config ExecutionConfig) time.Duration {
	jo.mutex.Lock()
	duration := time.Since(startTime)
	jo.EndTime = jo.StartTime.Add(duration)
	jo.Success = overallSuccess
	if overallSuccess {
		jo.Status = "Completed"
	} else {
		jo.Status = "Failed"
	}
	jo.mutex.Unlock()

	// Mark job as completed
	config.JobScheduler.SetJobStatus(jobID, JobCompleted, overallSuccess)

	return duration
}

// printRepeatStepResults prints the final results of repeat step executions to buffer
func (e *BufferedJobExecutor) printRepeatStepResults(ctx *JobContext, job *Job, jo *JobOutput) {
	// Capture the step outputs output to buffer instead of printing directly
	originalStdout := os.Stdout
	r, wr, _ := os.Pipe()
	os.Stdout = wr

	output := ctx.Printer

	for i, step := range job.Steps {
		if counter, exists := ctx.StepCounters[i]; exists {
			hasTest := step.Test != ""
			output.PrintStepRepeatResult(i, counter, hasTest)
		}
	}

	// Restore stdout and capture the step outputs
	wr.Close()
	os.Stdout = originalStdout

	capturedOutput, _ := io.ReadAll(r)

	// Add captured step outputs to job buffer
	jo.mutex.Lock()
	jo.Buffer.Write(capturedOutput)
	jo.mutex.Unlock()
}

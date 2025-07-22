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
	UseBuffering     bool
	UseParallel      bool
	HasDependencies  bool
	WorkflowOutput   *WorkflowOutput
	JobScheduler     *JobScheduler
}

// JobExecutor defines the interface for executing jobs
type JobExecutor interface {
	Execute(job *Job, jobID string, ctx JobContext, config ExecutionConfig) ExecutionResult
}

// ParallelJobExecutor handles parallel job execution without dependencies
type ParallelJobExecutor struct {
	workflow *Workflow
}

// NewParallelJobExecutor creates a new parallel job executor
func NewParallelJobExecutor(w *Workflow) *ParallelJobExecutor {
	return &ParallelJobExecutor{workflow: w}
}

// Execute runs a job in parallel mode
func (e *ParallelJobExecutor) Execute(job *Job, jobID string, ctx JobContext, config ExecutionConfig) ExecutionResult {
	startTime := time.Now()
	
	// Initialize repeat tracking
	if job.Repeat != nil {
		ctx.IsRepeating = true
		ctx.RepeatTotal = job.Repeat.Count
		ctx.StepCounters = make(map[int]StepRepeatCounter)
	} else {
		ctx.IsRepeating = false
		ctx.RepeatTotal = 1
		ctx.StepCounters = make(map[int]StepRepeatCounter)
	}

	success := true
	
	// Execute with or without repeat
	if job.Repeat != nil {
		for i := 0; i < job.Repeat.Count; i++ {
			ctx.RepeatCurrent = i + 1
			if !job.Start(ctx) {
				success = false
				e.workflow.SetExitStatus(true)
			}
			// Sleep between repeats (except for the last one)
			if i < job.Repeat.Count-1 {
				time.Sleep(job.Repeat.Interval.Duration)
			}
		}
	} else {
		ctx.RepeatCurrent = 1
		if !job.Start(ctx) {
			success = false
			e.workflow.SetExitStatus(true)
		}
	}

	return ExecutionResult{
		Success:  success,
		Duration: time.Since(startTime),
		Output:   "",
		Error:    nil,
	}
}

// SequentialJobExecutor handles sequential job execution with dependencies
type SequentialJobExecutor struct {
	workflow *Workflow
}

// NewSequentialJobExecutor creates a new sequential job executor
func NewSequentialJobExecutor(w *Workflow) *SequentialJobExecutor {
	return &SequentialJobExecutor{workflow: w}
}

// Execute runs a job in sequential mode with dependency management
func (e *SequentialJobExecutor) Execute(job *Job, jobID string, ctx JobContext, config ExecutionConfig) ExecutionResult {
	startTime := time.Now()
	overallSuccess := true

	// Initialize repeat tracking
	_, total := config.JobScheduler.GetRepeatInfo(jobID)
	ctx.IsRepeating = total > 1
	ctx.RepeatTotal = total
	ctx.StepCounters = make(map[int]StepRepeatCounter)

	// Execute job with repeat logic
	for config.JobScheduler.ShouldRepeatJob(jobID) {
		current, _ := config.JobScheduler.GetRepeatInfo(jobID)
		ctx.RepeatCurrent = current

		// Execute single run
		success := !job.Start(ctx)

		if !success {
			overallSuccess = false
			e.workflow.SetExitStatus(true)
		}

		// Increment counter
		config.JobScheduler.IncrementRepeatCounter(jobID)

		// Sleep between repeats (except for the last one)
		current, target := config.JobScheduler.GetRepeatInfo(jobID)
		if current < target && job.Repeat != nil {
			time.Sleep(job.Repeat.Interval.Duration)
		}
	}

	// Mark job as completed
	config.JobScheduler.SetJobStatus(jobID, JobCompleted, overallSuccess)

	return ExecutionResult{
		Success:  overallSuccess,
		Duration: time.Since(startTime),
		Output:   "",
		Error:    nil,
	}
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
	overallSuccess := true

	jo := config.WorkflowOutput.Jobs[jobID]
	jo.mutex.Lock()
	jo.StartTime = startTime
	jo.Status = "Running"
	jo.mutex.Unlock()

	// Initialize repeat tracking and buffering flag
	_, total := config.JobScheduler.GetRepeatInfo(jobID)
	ctx.IsRepeating = total > 1
	ctx.RepeatTotal = total
	ctx.StepCounters = make(map[int]StepRepeatCounter)
	ctx.UseBuffering = true

	// Execute job with repeat logic
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
		success := !job.Start(ctx)

		// Restore stdout and capture output
		wr.Close()
		os.Stdout = originalStdout

		config.WorkflowOutput.outputMutex.Unlock()

		capturedOutput, _ := io.ReadAll(r)

		// Add captured output to buffer
		jo.mutex.Lock()
		jo.Buffer.Write(capturedOutput)
		jo.mutex.Unlock()

		if !success {
			overallSuccess = false
			e.workflow.SetExitStatus(true)
		}

		// Increment counter
		config.JobScheduler.IncrementRepeatCounter(jobID)

		// Sleep between repeats (except for the last one)
		current, target := config.JobScheduler.GetRepeatInfo(jobID)
		if current < target && job.Repeat != nil {
			time.Sleep(job.Repeat.Interval.Duration)
		}
	}

	// Print repeat step results after all executions complete
	if ctx.IsRepeating {
		e.printRepeatStepResults(&ctx, job, jo)
	}

	// Update final status
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

	return ExecutionResult{
		Success:  overallSuccess,
		Duration: duration,
		Output:   jo.Buffer.String(),
		Error:    nil,
	}
}

// printRepeatStepResults prints the final results of repeat step executions to buffer
func (e *BufferedJobExecutor) printRepeatStepResults(ctx *JobContext, job *Job, jo *JobOutput) {
	// Capture the step results output to buffer instead of printing directly
	originalStdout := os.Stdout
	r, wr, _ := os.Pipe()
	os.Stdout = wr
	
	output := NewOutput(ctx.Config.Verbose)
	
	for i, step := range job.Steps {
		if counter, exists := ctx.StepCounters[i]; exists {
			hasTest := step.Test != ""
			output.PrintStepRepeatResult(i, counter, hasTest)
		}
	}
	
	// Restore stdout and capture the step results
	wr.Close()
	os.Stdout = originalStdout
	
	capturedOutput, _ := io.ReadAll(r)
	
	// Add captured step results to job buffer
	jo.mutex.Lock()
	jo.Buffer.Write(capturedOutput)
	jo.mutex.Unlock()
}
package probe

import (
	"fmt"
	"io"
	"os"
	"time"
)

// executeJobWithRepeat handles the execution of a job with repeat support
func (w *Workflow) executeJobWithRepeat(scheduler *JobScheduler, job *Job, jobID string, ctx JobContext) {
	overallSuccess := true

	// Initialize repeat tracking
	_, total := scheduler.GetRepeatInfo(jobID)
	ctx.IsRepeating = total > 1
	ctx.RepeatTotal = total
	ctx.StepCounters = make(map[int]StepRepeatCounter)

	// Execute job with repeat logic
	for scheduler.ShouldRepeatJob(jobID) {
		current, _ := scheduler.GetRepeatInfo(jobID)
		ctx.RepeatCurrent = current

		// Execute single run
		success := !job.Start(ctx)

		if !success {
			overallSuccess = false
			w.SetExitStatus(true)
		}

		// Increment counter
		scheduler.IncrementRepeatCounter(jobID)

		// Sleep between repeats (except for the last one)
		current, target := scheduler.GetRepeatInfo(jobID)
		if current < target && job.Repeat != nil {
			time.Sleep(job.Repeat.Interval.Duration)
		}
	}

	// Mark job as completed
	scheduler.SetJobStatus(jobID, JobCompleted, overallSuccess)
}

// executeJobWithBuffering handles job execution with buffered output
func (w *Workflow) executeJobWithBuffering(scheduler *JobScheduler, job *Job, jobID string, ctx JobContext, wo *WorkflowOutput) {
	jo := wo.Jobs[jobID]
	jo.mutex.Lock()
	jo.StartTime = time.Now()
	jo.Status = "Running"
	jo.mutex.Unlock()

	// Initialize repeat tracking and buffering flag
	_, total := scheduler.GetRepeatInfo(jobID)
	ctx.IsRepeating = total > 1
	ctx.RepeatTotal = total
	ctx.StepCounters = make(map[int]StepRepeatCounter)
	ctx.UseBuffering = true // Set buffering flag to prevent duplicate job names

	// Store status update (don't print immediately)
	// Status updates will be shown in detailed results

	overallSuccess := true

	// Execute job with repeat logic
	for scheduler.ShouldRepeatJob(jobID) {
		current, _ := scheduler.GetRepeatInfo(jobID)
		ctx.RepeatCurrent = current
		// Serialize stdout redirection to prevent race conditions
		wo.outputMutex.Lock()

		// Capture output by redirecting stdout temporarily
		originalStdout := os.Stdout
		r, wr, _ := os.Pipe()
		os.Stdout = wr

		// Execute single run
		success := !job.Start(ctx)

		// Restore stdout and capture output
		wr.Close()
		os.Stdout = originalStdout

		wo.outputMutex.Unlock()

		capturedOutput, _ := io.ReadAll(r)

		// Add captured output to buffer
		jo.mutex.Lock()
		jo.Buffer.Write(capturedOutput)
		jo.mutex.Unlock()

		if !success {
			overallSuccess = false
			w.SetExitStatus(true)
		}

		// Increment counter
		scheduler.IncrementRepeatCounter(jobID)

		// Sleep between repeats (except for the last one)
		current, target := scheduler.GetRepeatInfo(jobID)
		if current < target && job.Repeat != nil {
			time.Sleep(job.Repeat.Interval.Duration)
		}
	}

	// Print repeat step results after all executions complete
	if ctx.IsRepeating {
		w.printRepeatStepResults(&ctx, job, jo)
	}

	// Update final status (don't print immediately)
	jo.mutex.Lock()
	jo.EndTime = time.Now()
	jo.Success = overallSuccess
	if overallSuccess {
		jo.Status = "Completed"
	} else {
		jo.Status = "Failed"
	}
	jo.mutex.Unlock()

	// Mark job as completed
	scheduler.SetJobStatus(jobID, JobCompleted, overallSuccess)
}

// printDetailedResults prints the final detailed results
func (w *Workflow) printDetailedResults(wo *WorkflowOutput, output OutputWriter) {
	fmt.Println()

	totalTime := time.Duration(0)
	successCount := 0

	// Process jobs in original order
	for _, job := range w.Jobs {
		jobID := job.ID
		if jobID == "" {
			jobID = job.Name
		}
		jo, exists := wo.Jobs[jobID]
		if !exists {
			continue
		}

		jo.mutex.Lock()
		duration := jo.EndTime.Sub(jo.StartTime)
		totalTime += duration

		status := StatusSuccess
		if jo.Status == "Skipped" {
			status = StatusWarning
		} else if !jo.Success {
			status = StatusError
		} else {
			successCount++
		}

		output.PrintJobResult(jo.JobName, status, duration.Seconds())
		output.PrintJobOutput(jo.Buffer.String())

		jo.mutex.Unlock()
	}

	// Print summary
	totalJobs := len(wo.Jobs)
	output.PrintWorkflowSummary(totalTime.Seconds(), successCount, totalJobs)
}

// printRepeatStepResults prints the final results of repeat step executions to buffer
func (w *Workflow) printRepeatStepResults(ctx *JobContext, job *Job, jo *JobOutput) {
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
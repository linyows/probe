package probe

import (
	"time"
)

// Start executes the workflow with the given configuration
func (w *Workflow) Start(c Config) error {
	output := c.Printer

	// Print workflow header at the beginning
	output.PrintWorkflowHeader(w.Name, w.Description)

	// Initialize shared outputs
	if w.sharedOutputs == nil {
		w.sharedOutputs = NewOutputs()
	}

	vars, err := w.evalVars()
	if err != nil {
		return err
	}

	ctx := w.newJobContext(c, vars)

	// Always use buffered execution with dependency management
	return w.startWithBufferedExecution(ctx)
}


// startWithBufferedExecution executes all jobs with buffered output and dependency management
func (w *Workflow) startWithBufferedExecution(ctx JobContext) error {
	scheduler, err := w.initializeJobScheduler()
	if err != nil {
		return err
	}

	workflowOutput := w.setupBufferedOutput()

	err = w.executeJobsWithDependencies(scheduler, ctx, workflowOutput)
	if err != nil {
		return err
	}

	w.printDetailedResults(workflowOutput, ctx.Printer)
	return nil
}

// initializeJobScheduler creates and sets up the job scheduler with dependencies
func (w *Workflow) initializeJobScheduler() (*JobScheduler, error) {
	scheduler := NewJobScheduler()

	// Add all jobs to scheduler
	for i := range w.Jobs {
		if err := scheduler.AddJob(&w.Jobs[i]); err != nil {
			return nil, err
		}
	}

	// Validate dependencies
	if err := scheduler.ValidateDependencies(); err != nil {
		return nil, err
	}

	return scheduler, nil
}

// setupBufferedOutput creates workflow output for buffering
func (w *Workflow) setupBufferedOutput() *WorkflowOutput {
	workflowOutput := NewWorkflowOutput()
	// Initialize job outputs
	for _, job := range w.Jobs {
		jobID := job.ID
		if jobID == "" {
			jobID = job.Name
		}
		jo := &JobOutput{
			JobName:   job.Name,
			JobID:     jobID,
			StartTime: time.Now(),
		}
		workflowOutput.Jobs[jobID] = jo
	}

	return workflowOutput
}

// executeJobsWithDependencies runs the main job execution loop with dependency management
func (w *Workflow) executeJobsWithDependencies(scheduler *JobScheduler, ctx JobContext, workflowOutput *WorkflowOutput) error {

	for !scheduler.AllJobsCompleted() {
		runnableJobs := scheduler.GetRunnableJobs()

		if len(runnableJobs) == 0 {
			if err := w.handleNoRunnableJobs(scheduler, workflowOutput); err != nil {
				return err
			}
			continue
		}

		w.processRunnableJobs(runnableJobs, scheduler, ctx, workflowOutput)
		scheduler.wg.Wait()
	}

	return nil
}

// handleNoRunnableJobs handles the case when no jobs can be run (failed dependencies or deadlock)
func (w *Workflow) handleNoRunnableJobs(scheduler *JobScheduler, workflowOutput *WorkflowOutput) error {
	skippedJobs := scheduler.MarkJobsWithFailedDependencies()

	// Update skipped jobs in workflow output
	if workflowOutput != nil {
		w.updateSkippedJobsOutput(skippedJobs, workflowOutput)
	}

	if len(skippedJobs) == 0 {
		// If no jobs were skipped, we might have a deadlock
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// updateSkippedJobsOutput updates the output for jobs that were skipped due to failed dependencies
func (w *Workflow) updateSkippedJobsOutput(skippedJobs []string, workflowOutput *WorkflowOutput) {
	for _, jobID := range skippedJobs {
		if jo, exists := workflowOutput.Jobs[jobID]; exists {
			jo.mutex.Lock()
			jo.EndTime = jo.StartTime // Set end time same as start time (0 duration)
			jo.Status = "Skipped"
			jo.Success = false
			jo.mutex.Unlock()
		}
	}
}

// processRunnableJobs starts execution of all currently runnable jobs
func (w *Workflow) processRunnableJobs(runnableJobs []string, scheduler *JobScheduler, ctx JobContext, workflowOutput *WorkflowOutput) {
	for _, jobID := range runnableJobs {
		job := scheduler.jobs[jobID]
		scheduler.SetJobStatus(jobID, JobRunning, false)
		scheduler.wg.Add(1)

		go func(j *Job, id string) {
			defer scheduler.wg.Done()

			config := ExecutionConfig{
				HasDependencies: true,
				WorkflowOutput:  workflowOutput,
				JobScheduler:    scheduler,
			}

			executor := NewBufferedJobExecutor(w)
			result := executor.Execute(j, id, ctx, config)
			if !result.Success {
				w.SetExitStatus(true)
			}
		}(job, jobID)
	}
}



// printDetailedResults prints the final detailed results
func (w *Workflow) printDetailedResults(wo *WorkflowOutput, printer PrintWriter) {
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

		printer.PrintJobResult(jo.JobName, status, duration.Seconds())
		printer.PrintJobOutput(jo.Buffer.String())

		jo.mutex.Unlock()
	}

	// Print summary
	totalJobs := len(wo.Jobs)
	printer.PrintWorkflowSummary(totalTime.Seconds(), successCount, totalJobs)
}

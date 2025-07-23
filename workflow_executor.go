package probe

import (
	"fmt"
	"sync"
	"time"
)

// Start executes the workflow with the given configuration
func (w *Workflow) Start(c Config) error {
	output := NewOutput(c.Verbose)

	// Print workflow header at the beginning
	output.PrintWorkflowHeader(w.Name, w.Description)

	vars, err := w.evalVars()
	if err != nil {
		return err
	}

	ctx := w.newJobContext(c, vars)

	// Check if any job has dependencies
	hasDependencies := false
	for _, job := range w.Jobs {
		if len(job.Needs) > 0 {
			hasDependencies = true
			break
		}
	}

	// If no dependencies, use the old parallel execution
	if !hasDependencies {
		return w.startParallel(ctx)
	}

	// Use dependency-aware execution
	return w.startWithDependencies(ctx)
}

// startParallel executes jobs in parallel without dependencies using unified executor
func (w *Workflow) startParallel(ctx JobContext) error {
	var wg sync.WaitGroup
	executor := NewParallelJobExecutor(w)

	for _, job := range w.Jobs {
		jobID := w.getJobID(job)
		config := w.createParallelExecutionConfig()

		// Handle jobs without repeat
		if job.Repeat == nil {
			w.executeJobOnce(&wg, executor, job, jobID, ctx, config)
			continue
		}

		// Handle jobs with repeat
		w.executeJobWithRepeat(&wg, job, jobID, ctx)
	}

	wg.Wait()
	return nil
}

// getJobID returns the job ID, using job name if ID is not set
func (w *Workflow) getJobID(job Job) string {
	if job.ID != "" {
		return job.ID
	}
	return job.Name
}

// createParallelExecutionConfig creates a configuration for parallel execution
func (w *Workflow) createParallelExecutionConfig() ExecutionConfig {
	return ExecutionConfig{
		UseBuffering:    false,
		UseParallel:     true,
		HasDependencies: false,
	}
}

// executeJobOnce executes a job once in a separate goroutine
func (w *Workflow) executeJobOnce(wg *sync.WaitGroup, executor JobExecutor, job Job, jobID string, ctx JobContext, config ExecutionConfig) {
	wg.Add(1)
	go func(j Job, jID string, jCtx JobContext) {
		defer wg.Done()
		result := executor.Execute(&j, jID, jCtx, config)
		if !result.Success {
			w.SetExitStatus(true)
		}
	}(job, jobID, ctx)
}

// executeJobWithRepeat handles jobs with repeat - creates separate goroutines for each repetition
func (w *Workflow) executeJobWithRepeat(wg *sync.WaitGroup, job Job, jobID string, ctx JobContext) {
	for i := 0; i < job.Repeat.Count; i++ {
		wg.Add(1)
		go func(j Job, jID string, jCtx JobContext, iteration int) {
			defer wg.Done()
			
			// Set up context for this specific iteration
			jCtx.IsRepeating = true
			jCtx.RepeatTotal = j.Repeat.Count
			jCtx.RepeatCurrent = iteration + 1
			jCtx.StepCounters = make(map[int]StepRepeatCounter)
			
			// Execute single iteration
			if !j.Start(jCtx) {
				w.SetExitStatus(true)
			}
		}(job, jobID, ctx, i)
		
		// Sleep between repeat launches (except for the last one)
		if i < job.Repeat.Count-1 {
			time.Sleep(job.Repeat.Interval.Duration)
		}
	}
}

// startWithDependencies executes jobs with dependency management
func (w *Workflow) startWithDependencies(ctx JobContext) error {
	scheduler, err := w.initializeJobScheduler()
	if err != nil {
		return err
	}

	workflowOutput := w.setupBufferedOutputIfNeeded()
	
	err = w.executeJobsWithDependencies(scheduler, ctx, workflowOutput)
	if err != nil {
		return err
	}

	w.printResultsIfBuffered(workflowOutput, ctx.Verbose)
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

// setupBufferedOutputIfNeeded creates workflow output for buffering if multiple jobs exist
func (w *Workflow) setupBufferedOutputIfNeeded() *WorkflowOutput {
	useBuffering := len(w.Jobs) > 1
	if !useBuffering {
		return nil
	}

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
	useBuffering := workflowOutput != nil

	for !scheduler.AllJobsCompleted() {
		runnableJobs := scheduler.GetRunnableJobs()

		if len(runnableJobs) == 0 {
			if err := w.handleNoRunnableJobs(scheduler, workflowOutput); err != nil {
				return err
			}
			continue
		}

		w.processRunnableJobs(runnableJobs, scheduler, ctx, useBuffering, workflowOutput)
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
func (w *Workflow) processRunnableJobs(runnableJobs []string, scheduler *JobScheduler, ctx JobContext, useBuffering bool, workflowOutput *WorkflowOutput) {
	for _, jobID := range runnableJobs {
		job := scheduler.jobs[jobID]
		scheduler.SetJobStatus(jobID, JobRunning, false)
		scheduler.wg.Add(1)

		go func(j *Job, id string) {
			defer scheduler.wg.Done()
			
			config := ExecutionConfig{
				UseBuffering:     useBuffering,
				UseParallel:      false,
				HasDependencies:  true,
				WorkflowOutput:   workflowOutput,
				JobScheduler:     scheduler,
			}

			executor := w.createJobExecutor(useBuffering)
			result := executor.Execute(j, id, ctx, config)
			if !result.Success {
				w.SetExitStatus(true)
			}
		}(job, jobID)
	}
}

// createJobExecutor creates the appropriate job executor based on buffering requirements
func (w *Workflow) createJobExecutor(useBuffering bool) JobExecutor {
	if useBuffering {
		return NewBufferedJobExecutor(w)
	}
	return NewSequentialJobExecutor(w)
}

// printResultsIfBuffered prints detailed results if buffered output was used
func (w *Workflow) printResultsIfBuffered(workflowOutput *WorkflowOutput, verbose bool) {
	if workflowOutput != nil {
		output := NewOutput(verbose)
		w.printDetailedResults(workflowOutput, output)
	}
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
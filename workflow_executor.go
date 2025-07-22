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
		jobID := job.ID
		if jobID == "" {
			jobID = job.Name
		}

		config := ExecutionConfig{
			UseBuffering:    false,
			UseParallel:     true,
			HasDependencies: false,
		}

		// Handle jobs without repeat
		if job.Repeat == nil {
			wg.Add(1)
			go func(j Job, jID string, jCtx JobContext) {
				defer wg.Done()
				result := executor.Execute(&j, jID, jCtx, config)
				if !result.Success {
					w.SetExitStatus(true)
				}
			}(job, jobID, ctx)
			continue
		}

		// Handle jobs with repeat - create separate goroutines for each repetition
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

	wg.Wait()
	return nil
}

// startWithDependencies executes jobs with dependency management
func (w *Workflow) startWithDependencies(ctx JobContext) error {
	scheduler := NewJobScheduler()

	// Add all jobs to scheduler
	for i := range w.Jobs {
		if err := scheduler.AddJob(&w.Jobs[i]); err != nil {
			return err
		}
	}

	// Validate dependencies
	if err := scheduler.ValidateDependencies(); err != nil {
		return err
	}

	// Use buffered output if multiple jobs
	useBuffering := len(w.Jobs) > 1
	var workflowOutput *WorkflowOutput
	if useBuffering {
		workflowOutput = NewWorkflowOutput()
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
	}

	// Execute jobs with dependency control and repeat support
	for !scheduler.AllJobsCompleted() {
		runnableJobs := scheduler.GetRunnableJobs()

		if len(runnableJobs) == 0 {
			// Check if there are pending jobs but none can run
			// This might indicate failed dependencies
			skippedJobs := scheduler.MarkJobsWithFailedDependencies()
			// Update skipped jobs in workflow output
			if useBuffering {
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
			if len(skippedJobs) == 0 {
				// If no jobs were skipped, we might have a deadlock
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		// Start all runnable jobs using unified executors
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

				var executor JobExecutor
				if useBuffering {
					executor = NewBufferedJobExecutor(w)
				} else {
					executor = NewSequentialJobExecutor(w)
				}

				result := executor.Execute(j, id, ctx, config)
				if !result.Success {
					w.SetExitStatus(true)
				}
			}(job, jobID)
		}

		// Wait for current batch to complete
		scheduler.wg.Wait()
	}

	// Print detailed results if buffering was used
	if useBuffering {
		output := NewOutput(ctx.Verbose)
		w.printDetailedResults(workflowOutput, output)
	}

	return nil
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
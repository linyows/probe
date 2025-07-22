package probe

import (
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

// startParallel executes jobs in parallel without dependencies
func (w *Workflow) startParallel(ctx JobContext) error {
	var wg sync.WaitGroup

	for _, job := range w.Jobs {
		// No repeat
		if job.Repeat == nil {
			// Set IsRepeating to false for jobs without repeat
			jobCtx := ctx
			jobCtx.IsRepeating = false
			jobCtx.RepeatTotal = 1
			jobCtx.RepeatCurrent = 1
			jobCtx.StepCounters = make(map[int]StepRepeatCounter)
			
			wg.Add(1)
			go func(j Job, jCtx JobContext) {
				defer wg.Done()
				w.SetExitStatus(j.Start(jCtx))
			}(job, jobCtx)
			continue
		}

		// Repeat
		for i := 0; i < job.Repeat.Count; i++ {
			// Set IsRepeating to true for jobs with repeat
			jobCtx := ctx
			jobCtx.IsRepeating = true
			jobCtx.RepeatTotal = job.Repeat.Count
			jobCtx.RepeatCurrent = i + 1
			jobCtx.StepCounters = make(map[int]StepRepeatCounter)
			
			wg.Add(1)
			go func(j Job, jCtx JobContext) {
				defer wg.Done()
				w.SetExitStatus(j.Start(jCtx))
			}(job, jobCtx)
			time.Sleep(job.Repeat.Interval.Duration)
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

		// Start all runnable jobs
		for _, jobID := range runnableJobs {
			job := scheduler.jobs[jobID]
			scheduler.SetJobStatus(jobID, JobRunning, false)
			scheduler.wg.Add(1)

			go func(j *Job, id string) {
				defer scheduler.wg.Done()
				if useBuffering {
					w.executeJobWithBuffering(scheduler, j, id, ctx, workflowOutput)
				} else {
					w.executeJobWithRepeat(scheduler, j, id, ctx)
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
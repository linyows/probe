package probe

import (
	"time"
)

type Workflow struct {
	Name        string         `yaml:"name" validate:"required"`
	Description string         `yaml:"description,omitempty"`
	Jobs        []Job          `yaml:"jobs" validate:"required"`
	Vars        map[string]any `yaml:"vars"`
	exitStatus  int
	env         map[string]string
	// Shared outputs across all jobs
	outputs *Outputs
	printer PrintWriter
}

// Start executes the workflow with the given configuration
func (w *Workflow) Start(c Config) error {
	if w.printer == nil {
		w.printer = NewPrinter(c.Verbose)
	}

	// Print workflow header at the beginning
	w.printer.PrintHeader(w.Name, w.Description)

	// Initialize shared outputs
	if w.outputs == nil {
		w.outputs = NewOutputs()
	}

	vars, err := w.evalVars()
	if err != nil {
		return err
	}

	ctx := w.newJobContext(c, vars)

	sc, err := w.initJobScheduler()
	if err != nil {
		return err
	}

	wb := w.setupWorkflowBuffer()

	err = w.startJobsWithDependencies(sc, ctx, wb)
	if err != nil {
		return err
	}

	w.printResults(wb)

	return nil
}

// initJobScheduler creates and sets up the job scheduler with dependencies
func (w *Workflow) initJobScheduler() (*JobScheduler, error) {
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

// setupWorkflowBuffer creates workflow buffer for buffering
func (w *Workflow) setupWorkflowBuffer() *WorkflowBuffer {
	wb := NewWorkflowBuffer()
	// Initialize job outputs
	for _, job := range w.Jobs {
		jobID := job.ID
		if jobID == "" {
			jobID = job.Name
		}
		jb := &JobBuffer{
			JobName:   job.Name,
			JobID:     jobID,
			StartTime: time.Now(),
		}
		wb.Jobs[jobID] = jb
	}

	return wb
}

// startJobsWithDependencies runs the main job execution loop with dependency management
func (w *Workflow) startJobsWithDependencies(sc *JobScheduler, ctx JobContext, wb *WorkflowBuffer) error {

	for !sc.AllJobsCompleted() {
		runnableJobs := sc.GetRunnableJobs()

		if len(runnableJobs) == 0 {
			if err := w.handleNoRunnableJobs(sc, wb); err != nil {
				return err
			}
			continue
		}

		w.processRunnableJobs(runnableJobs, sc, ctx, wb)
		sc.wg.Wait()
	}

	return nil
}

// handleNoRunnableJobs handles the case when no jobs can be run (failed dependencies or deadlock)
func (w *Workflow) handleNoRunnableJobs(scheduler *JobScheduler, workflowBuffer *WorkflowBuffer) error {
	skippedJobs := scheduler.MarkJobsWithFailedDependencies()

	// Update skipped jobs in workflow printer
	if workflowBuffer != nil {
		w.updateSkippedJobsOutput(skippedJobs, workflowBuffer)
	}

	if len(skippedJobs) == 0 {
		// If no jobs were skipped, we might have a deadlock
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// updateSkippedJobsOutput updates the output for jobs that were skipped due to failed dependencies
func (w *Workflow) updateSkippedJobsOutput(skippedJobs []string, workflowBuffer *WorkflowBuffer) {
	for _, jobID := range skippedJobs {
		if jb, exists := workflowBuffer.Jobs[jobID]; exists {
			jb.mutex.Lock()
			jb.EndTime = jb.StartTime // Set end time same as start time (0 duration)
			jb.Status = "Skipped"
			jb.Success = false
			jb.mutex.Unlock()
		}
	}
}

// processRunnableJobs starts execution of all currently runnable jobs
func (w *Workflow) processRunnableJobs(runnableJobs []string, scheduler *JobScheduler, ctx JobContext, workflowBuffer *WorkflowBuffer) {
	for _, jobID := range runnableJobs {
		job := scheduler.jobs[jobID]
		scheduler.SetJobStatus(jobID, JobRunning, false)
		scheduler.wg.Add(1)

		go func(j *Job, id string) {
			defer scheduler.wg.Done()

			config := ExecutionConfig{
				HasDependencies: true,
				WorkflowBuffer:  workflowBuffer,
				JobScheduler:    scheduler,
			}

			executor := NewExecutor(w)
			result := executor.Execute(j, id, ctx, config)
			if !result.Success {
				w.SetExitStatus(true)
			}
		}(job, jobID)
	}
}

// printResults prints the final detailed results
func (w *Workflow) printResults(wb *WorkflowBuffer) {
	totalTime := time.Duration(0)
	successCount := 0

	// Process jobs in original order
	for _, job := range w.Jobs {
		jobID := job.ID
		if jobID == "" {
			jobID = job.Name
		}
		jb, exists := wb.Jobs[jobID]
		if !exists {
			continue
		}

		jb.mutex.Lock()
		duration := jb.EndTime.Sub(jb.StartTime)
		totalTime += duration

		status := StatusSuccess
		if jb.Status == "Skipped" {
			status = StatusWarning
		} else if !jb.Success {
			status = StatusError
		} else {
			successCount++
		}

		w.printer.PrintJobStatus(jb.JobName, status, duration.Seconds())
		w.printer.PrintJobResults(jb.Buffer.String())

		jb.mutex.Unlock()
	}

	w.printer.PrintFooter(totalTime.Seconds(), successCount, len(wb.Jobs))
}

func (w *Workflow) SetExitStatus(isErr bool) {
	if isErr {
		w.exitStatus = 1
	}
}

func (w *Workflow) Env() map[string]string {
	if len(w.env) == 0 {
		w.env = EnvMap()
	}
	return w.env
}

func (w *Workflow) evalVars() (map[string]any, error) {
	env := StrmapToAnymap(w.Env())
	vars := make(map[string]any)

	expr := &Expr{}
	for k, v := range w.Vars {
		if mapV, ok := v.(map[string]any); ok {
			vars[k] = expr.EvalTemplateMap(mapV, env)
		} else if strV, ok2 := v.(string); ok2 {
			output, err := expr.EvalTemplate(strV, env)
			if err != nil {
				return vars, err
			}
			vars[k] = output
		}
	}

	return vars, nil
}

func (w *Workflow) newJobContext(c Config, vars map[string]any) JobContext {
	return JobContext{
		Vars:    vars,
		Logs:    []map[string]any{},
		Config:  c,
		Printer: w.printer,
		Outputs: w.outputs,
	}
}

package probe

// StepContext provides context data for step expression evaluation
type StepContext struct {
	Vars    map[string]any `expr:"vars"`
	Res     map[string]any `expr:"res"`
	Req     map[string]any `expr:"req"`
	RT      string         `expr:"rt"`
	RTSec   float64        `expr:"rt_sec"`
	Report  string         `expr:"report"`
	Outputs map[string]any `expr:"outputs"`
}

// JobContext provides context data for job execution
type JobContext struct {
	Vars map[string]any `expr:"vars"`
	Config
	Failed bool
	// Current job ID for this context
	CurrentJobID string
	// Repeat tracking
	IsRepeating   bool
	RepeatCurrent int
	RepeatTotal   int
	StepCounters  map[int]StepRepeatCounter // step index -> counter
	// Print writer
	Printer *Printer
	// Result for managing job-level output
	Result *Result
	// Job scheduler for managing job dependencies and execution
	JobScheduler *JobScheduler
	// Shared outputs across all jobs (accessible via expressions as "outputs")
	Outputs *Outputs `expr:"outputs"`
}

// SetFailed marks the job context as failed
func (j *JobContext) SetFailed() {
	j.Failed = true
}

package probe

// ResponseTime provides response time information for expressions
type ResponseTime struct {
	Duration string  `expr:"duration"`
	Sec      float64 `expr:"sec"`
}

// ExitStatus represents the unified status across all actions
type ExitStatus int

const (
	ExitStatusSuccess ExitStatus = 0 // Action succeeded
	ExitStatusFailure ExitStatus = 1 // Action failed
)

// Int returns the ExitStatus as an int for comparison
func (e ExitStatus) Int() int {
	return int(e)
}

// String returns the ExitStatus as a string
func (e ExitStatus) String() string {
	switch e {
	case ExitStatusSuccess:
		return "success"
	case ExitStatusFailure:
		return "failure"
	default:
		return "unknown"
	}
}

// StepContext provides context data for step expression evaluation
type StepContext struct {
	Vars        map[string]any `expr:"vars"`
	Res         map[string]any `expr:"res"`
	Req         map[string]any `expr:"req"`
	RT          ResponseTime   `expr:"rt"`
	Report      string         `expr:"report"`
	Outputs     map[string]any `expr:"outputs"`
	Status      int            `expr:"status"`
	RepeatIndex int            `expr:"repeat_index"`
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

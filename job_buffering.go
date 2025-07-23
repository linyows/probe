package probe

import (
	"strings"
	"sync"
	"time"
)

// JobOutput stores buffered output for a job
type JobOutput struct {
	JobName   string
	JobID     string
	Buffer    strings.Builder
	Status    string
	StartTime time.Time
	EndTime   time.Time
	Success   bool
	mutex     sync.Mutex
}

// WorkflowOutput manages output for multiple jobs
type WorkflowOutput struct {
	Jobs        map[string]*JobOutput
	mutex       sync.RWMutex //nolint:unused // Reserved for future concurrent access
	outputMutex sync.Mutex   // Protects stdout redirection
}

// NewWorkflowOutput creates a new WorkflowOutput instance
func NewWorkflowOutput() *WorkflowOutput {
	return &WorkflowOutput{
		Jobs: make(map[string]*JobOutput),
	}
}
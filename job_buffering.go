package probe

import (
	"strings"
	"sync"
	"time"
)

// JobPrinter stores buffered output for a job
type JobPrinter struct {
	JobName   string
	JobID     string
	Buffer    strings.Builder
	Status    string
	StartTime time.Time
	EndTime   time.Time
	Success   bool
	mutex     sync.Mutex
}

// WorkflowPrinter manages output for multiple jobs
type WorkflowPrinter struct {
	Jobs        map[string]*JobPrinter
	mutex       sync.RWMutex //nolint:unused // Reserved for future concurrent access
	outputMutex sync.Mutex   // Protects stdout redirection
}

// NewWorkflowPrinter creates a new WorkflowPrinter instance
func NewWorkflowPrinter() *WorkflowPrinter {
	return &WorkflowPrinter{
		Jobs: make(map[string]*JobPrinter),
	}
}
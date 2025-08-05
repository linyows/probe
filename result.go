package probe

import (
	"sync"
	"time"
)

// StatusType represents the status of execution
type StatusType int

const (
	StatusSuccess StatusType = iota
	StatusError
	StatusWarning
	StatusSkipped
)

// StepRepeatCounter tracks the execution results of repeated steps
type StepRepeatCounter struct {
	SuccessCount int
	FailureCount int
	Name         string
	LastResult   bool
}

// StepResult represents the result of a step execution
type StepResult struct {
	Index         int
	Name          string
	Status        StatusType
	RT            string
	WaitTime      string
	TestOutput    string
	EchoOutput    string
	Report        string
	HasTest       bool
	RepeatCounter *StepRepeatCounter // For repeat execution information
}

// JobResult stores execution results for a job
type JobResult struct {
	JobName     string
	JobID       string
	Status      string
	StartTime   time.Time
	EndTime     time.Time
	Success     bool
	StepResults []StepResult // Store all step results for this job
	mutex       sync.Mutex
}

// Result manages execution results for multiple jobs
type Result struct {
	Jobs map[string]*JobResult
}

// NewResult creates a new Result instance
func NewResult() *Result {
	return &Result{
		Jobs: make(map[string]*JobResult),
	}
}

// AddStepResult adds a StepResult to the specified job result
func (rs *Result) AddStepResult(jobID string, stepResult StepResult) {
	if jr, exists := rs.Jobs[jobID]; exists {
		jr.mutex.Lock()
		defer jr.mutex.Unlock()
		jr.StepResults = append(jr.StepResults, stepResult)
	}
}

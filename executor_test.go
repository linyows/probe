package probe

import (
	"sync"
	"testing"
	"time"
)

// Test basic executor creation and interface compliance
func TestJobExecutor_Creation(t *testing.T) {
	workflow := &Workflow{Name: "test-workflow"}
	job := &Job{Name: "test-job", Steps: []*Step{}}

	t.Run("Executor creation", func(t *testing.T) {
		executor := NewExecutor(workflow, job)
		if executor == nil {
			t.Fatal("NewExecutor should not return nil")
		}
	})
}

// TestExecutionResult_Structure is no longer needed as ExecutionResult has been removed

// TestExecutionConfig_Structure is no longer needed as ExecutionConfig has been removed

func TestExecutor_AppendRepeatStepResults(t *testing.T) {
	workflow := &Workflow{Name: "test-workflow"}
	job := &Job{
		Name: "test-job",
		ID:   "test-job",
		Steps: []*Step{
			{
				Name: "test-step",
				Test: "status == 200",
			},
		},
	}
	executor := NewExecutor(workflow, job)

	// Create WorkflowBuffer with JobResult
	result := NewResult()
	jobResult := &JobResult{
		JobName: job.Name,
		JobID:   "test-job",
	}
	result.Jobs["test-job"] = jobResult

	// Create test context with step counters
	ctx := JobContext{
		StepCounters: map[int]StepRepeatCounter{
			0: {
				SuccessCount: 3,
				FailureCount: 1,
				Name:         "test-step",
				LastResult:   true,
			},
		},
		Config:     Config{Verbose: false},
		Printer:    NewPrinter(false, []string{}),
		Result:     result,
		countersMu: &sync.Mutex{},
	}

	// Call the method
	executor.appendRepeatStepResults(&ctx)

	// Check if step results were added to WorkflowBuffer
	jobResult, exists := result.Jobs["test-job"]
	if !exists {
		t.Fatal("Job buffer should exist after appendRepeatStepResults")
	}
	if len(jobResult.StepResults) == 0 {
		t.Error("appendRepeatStepResults should add StepResults to WorkflowBuffer")
	}

	// Should have created a StepResult with RepeatCounter
	if len(jobResult.StepResults) != 1 {
		t.Errorf("Expected 1 step result, got %d", len(jobResult.StepResults))
	}

	stepResult := jobResult.StepResults[0]
	if stepResult.RepeatCounter == nil {
		t.Error("StepResult should have RepeatCounter")
	}

	if stepResult.RepeatCounter.SuccessCount != 3 {
		t.Errorf("Expected SuccessCount=3, got %d", stepResult.RepeatCounter.SuccessCount)
	}

	if stepResult.RepeatCounter.FailureCount != 1 {
		t.Errorf("Expected FailureCount=1, got %d", stepResult.RepeatCounter.FailureCount)
	}
}

func TestJobExecutor_Integration_WithMockJob(t *testing.T) {
	// Test that the executor can handle basic job execution scenarios
	// without relying on the actual job.Start() method which has plugin dependencies

	t.Run("executor creation and interface compliance", func(t *testing.T) {
		workflow := &Workflow{Name: "test-workflow"}
		job := &Job{Name: "test-job", Steps: []*Step{}}

		// Test that the executor can be created
		executor := NewExecutor(workflow, job)

		if executor == nil {
			t.Error("Executor creation failed")
		}
	})
}

func TestExecutor_AsyncRepeat(t *testing.T) {
	t.Run("async flag should be recognized", func(t *testing.T) {
		// Test that async flag is properly set and recognized
		asyncRepeat := &Repeat{
			Count:    10,
			Interval: Interval{Duration: 10 * time.Millisecond},
			Async:    true,
		}

		if !asyncRepeat.Async {
			t.Error("Async flag should be true")
		}

		syncRepeat := &Repeat{
			Count:    10,
			Interval: Interval{Duration: 10 * time.Millisecond},
			Async:    false,
		}

		if syncRepeat.Async {
			t.Error("Async flag should be false")
		}
	})

	t.Run("async repeat structure is valid", func(t *testing.T) {
		workflow := &Workflow{Name: "test-workflow"}
		job := &Job{
			Name: "test-job",
			ID:   "test-job",
			Repeat: &Repeat{
				Count:    5,
				Interval: Interval{Duration: 10 * time.Millisecond},
				Async:    true,
			},
			Steps: []*Step{},
		}

		executor := NewExecutor(workflow, job)
		if executor == nil {
			t.Fatal("Executor should not be nil")
		}

		if job.Repeat == nil {
			t.Fatal("Job repeat should not be nil")
		}

		if !job.Repeat.Async {
			t.Error("Job repeat async flag should be true")
		}
	})
}

// TestExecutor_AsyncRepeat_NoDataRace exercises the executeJobRepeatLoopAsync
// path end-to-end with a mocked ActionRunner so that, under `go test -race`,
// any concurrent mutation of shared *Step / *Job state is reported as a race.
//
// Before the fix, every goroutine spawned by the async repeat loop called
// e.job.Start on the same *Job, which in turn mutated j.Name (expandJobName)
// and st.Expr / st.ctx / st.startedAt / st.err / st.retryAttempt on the same
// *Step instances.
func TestExecutor_AsyncRepeat_NoDataRace(t *testing.T) {
	runner := NewMockActionRunner()
	runner.SetResult("hello", map[string]any{"status": 0})

	// Each Step instance has actionRunner pre-wired to the mock so that
	// st.executeAction takes the mock path instead of spawning the
	// real plugin process.
	step := &Step{
		Name:         "tick",
		Uses:         "hello",
		actionRunner: runner,
	}

	workflow := &Workflow{
		Name: "async-race-test",
		Jobs: []Job{
			{
				Name:  "racer",
				ID:    "racer",
				Steps: []*Step{step},
				Repeat: &Repeat{
					Count:    20,
					Interval: Interval{Duration: 1 * time.Millisecond},
					Async:    true,
				},
			},
		},
		printer: newBufferPrinter(),
	}

	config := Config{Verbose: false}
	if err := workflow.Start(config); err != nil {
		t.Fatalf("workflow failed: %v", err)
	}
}

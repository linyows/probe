package probe

import (
	"testing"
	"time"
)

func init() {
	// Disable security exits to prevent os.Exit(2) calls during tests
	DisableSecurityExit(true)
}

func TestNewJobScheduler(t *testing.T) {
	js := NewJobScheduler()

	if js == nil {
		t.Fatal("NewJobScheduler should return a non-nil JobScheduler")
	}

	if js.jobs == nil {
		t.Error("jobs map should be initialized")
	}

	if js.status == nil {
		t.Error("status map should be initialized")
	}

	if js.results == nil {
		t.Error("results map should be initialized")
	}

	if js.repeatCounters == nil {
		t.Error("repeatCounters map should be initialized")
	}

	if js.repeatTargets == nil {
		t.Error("repeatTargets map should be initialized")
	}
}

func TestJobScheduler_AddJob(t *testing.T) {
	js := NewJobScheduler()

	// Test adding a job without repeat
	job1 := &Job{
		Name:  "test-job-1",
		ID:    "job1",
		Steps: []*Step{},
	}

	err := js.AddJob(job1)
	if err != nil {
		t.Fatalf("AddJob should not return error: %v", err)
	}

	if js.jobs["job1"] != job1 {
		t.Error("Job should be stored in jobs map")
	}

	if js.status["job1"] != JobPending {
		t.Error("Job status should be JobPending")
	}

	if js.repeatTargets["job1"] != 1 {
		t.Error("Job without repeat should have target count 1")
	}

	if js.repeatCounters["job1"] != 0 {
		t.Error("Job counter should be initialized to 0")
	}

	// Test adding a job with repeat
	job2 := &Job{
		Name:  "test-job-2",
		ID:    "job2",
		Steps: []*Step{},
		Repeat: &Repeat{
			Count:    3,
			Interval: Interval{Duration: 1 * time.Second},
		},
	}

	err = js.AddJob(job2)
	if err != nil {
		t.Fatalf("AddJob should not return error: %v", err)
	}

	if js.repeatTargets["job2"] != 3 {
		t.Error("Job with repeat should have target count 3")
	}

	// Test adding duplicate job ID
	job3 := &Job{
		Name:  "test-job-3",
		ID:    "job1", // Same ID as job1
		Steps: []*Step{},
	}

	err = js.AddJob(job3)
	if err == nil {
		t.Error("AddJob should return error for duplicate job ID")
	}

	// Test auto-generating ID
	job4 := &Job{
		Name:  "test-job-4",
		Steps: []*Step{},
	}

	err = js.AddJob(job4)
	if err != nil {
		t.Fatalf("AddJob should not return error: %v", err)
	}

	if job4.ID != "test-job-4" {
		t.Error("Job ID should be auto-generated from name")
	}
}

func TestJobScheduler_UniqueIDGeneration(t *testing.T) {
	js := NewJobScheduler()

	// Test multiple jobs with same name
	job1 := &Job{Name: "same-name", Steps: []*Step{}}
	job2 := &Job{Name: "same-name", Steps: []*Step{}}
	job3 := &Job{Name: "same-name", Steps: []*Step{}}

	err := js.AddJob(job1)
	if err != nil {
		t.Fatalf("AddJob should not return error: %v", err)
	}

	err = js.AddJob(job2)
	if err != nil {
		t.Fatalf("AddJob should not return error: %v", err)
	}

	err = js.AddJob(job3)
	if err != nil {
		t.Fatalf("AddJob should not return error: %v", err)
	}

	// Check that all jobs have unique IDs
	expectedIDs := []string{"same-name", "same-name-1", "same-name-2"}
	actualIDs := []string{job1.ID, job2.ID, job3.ID}

	for i, expectedID := range expectedIDs {
		if actualIDs[i] != expectedID {
			t.Errorf("Expected ID[%d] to be '%s', got '%s'", i, expectedID, actualIDs[i])
		}
	}

	// Verify all jobs are stored with their unique IDs
	for _, id := range expectedIDs {
		if _, exists := js.jobs[id]; !exists {
			t.Errorf("Job with ID '%s' should exist in scheduler", id)
		}
	}
}

func TestJobScheduler_GenerateUniqueID(t *testing.T) {
	js := NewJobScheduler()

	// Test empty name
	id := js.generateUniqueID("")
	if id != "job" {
		t.Errorf("Expected 'job' for empty name, got '%s'", id)
	}

	// Test normal name
	id = js.generateUniqueID("test")
	if id != "test" {
		t.Errorf("Expected 'test' for unique name, got '%s'", id)
	}

	// Add a job to create conflict
	js.jobs["test"] = &Job{}

	// Test conflict resolution
	id = js.generateUniqueID("test")
	if id != "test-1" {
		t.Errorf("Expected 'test-1' for conflicting name, got '%s'", id)
	}

	// Add more conflicts
	js.jobs["test-1"] = &Job{}
	js.jobs["test-2"] = &Job{}

	id = js.generateUniqueID("test")
	if id != "test-3" {
		t.Errorf("Expected 'test-3' for multiple conflicts, got '%s'", id)
	}
}

func TestJobScheduler_ValidateDependencies(t *testing.T) {
	js := NewJobScheduler()

	// Test with no dependencies
	job1 := &Job{Name: "job1", ID: "job1", Steps: []*Step{}}
	if err := js.AddJob(job1); err != nil {
		t.Fatalf("Failed to add job1: %v", err)
	}

	err := js.ValidateDependencies()
	if err != nil {
		t.Errorf("ValidateDependencies should not return error: %v", err)
	}

	// Test with valid dependencies
	job2 := &Job{Name: "job2", ID: "job2", Needs: []string{"job1"}, Steps: []*Step{}}
	if err := js.AddJob(job2); err != nil {
		t.Fatalf("Failed to add job2: %v", err)
	}

	err = js.ValidateDependencies()
	if err != nil {
		t.Errorf("ValidateDependencies should not return error: %v", err)
	}

	// Test with missing dependency
	job3 := &Job{Name: "job3", ID: "job3", Needs: []string{"missing-job"}, Steps: []*Step{}}
	if err := js.AddJob(job3); err != nil {
		t.Fatalf("Failed to add job3: %v", err)
	}

	err = js.ValidateDependencies()
	if err == nil {
		t.Error("ValidateDependencies should return error for missing dependency")
	}
}

func TestJobScheduler_CircularDependencies(t *testing.T) {
	js := NewJobScheduler()

	// Create circular dependency: job1 -> job2 -> job1
	job1 := &Job{Name: "job1", ID: "job1", Needs: []string{"job2"}, Steps: []*Step{}}
	job2 := &Job{Name: "job2", ID: "job2", Needs: []string{"job1"}, Steps: []*Step{}}

	if err := js.AddJob(job1); err != nil {
		t.Fatalf("Failed to add job1: %v", err)
	}
	if err := js.AddJob(job2); err != nil {
		t.Fatalf("Failed to add job2: %v", err)
	}

	err := js.ValidateDependencies()
	if err == nil {
		t.Error("ValidateDependencies should detect circular dependency")
	}
}

func TestJobScheduler_CanRunJob(t *testing.T) {
	js := NewJobScheduler()

	// Job without dependencies
	job1 := &Job{Name: "job1", ID: "job1", Steps: []*Step{}}
	if err := js.AddJob(job1); err != nil {
		t.Fatalf("Failed to add job1: %v", err)
	}

	if !js.CanRunJob("job1") {
		t.Error("Job without dependencies should be runnable")
	}

	// Job with dependencies
	job2 := &Job{Name: "job2", ID: "job2", Needs: []string{"job1"}, Steps: []*Step{}}
	if err := js.AddJob(job2); err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	if js.CanRunJob("job2") {
		t.Error("Job with incomplete dependencies should not be runnable")
	}

	// Complete job1
	js.SetJobStatus("job1", JobCompleted, true)
	js.repeatCounters["job1"] = 1 // Simulate completion

	if !js.CanRunJob("job2") {
		t.Error("Job with completed dependencies should be runnable")
	}

	// Set job2 to running
	js.SetJobStatus("job2", JobRunning, false)

	if js.CanRunJob("job2") {
		t.Error("Running job should not be runnable again")
	}
}

func TestJobScheduler_RepeatFunctionality(t *testing.T) {
	js := NewJobScheduler()

	job := &Job{
		Name: "repeat-job",
		ID:   "repeat-job",
		Repeat: &Repeat{
			Count:    3,
			Interval: Interval{Duration: 0},
		},
		Steps: []*Step{},
	}

	if err := js.AddJob(job); err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	// Test initial state
	if !js.ShouldRepeatJob("repeat-job") {
		t.Error("Job should be repeatable initially")
	}

	current, target := js.GetRepeatInfo("repeat-job")
	if current != 0 || target != 3 {
		t.Errorf("Expected current=0, target=3, got current=%d, target=%d", current, target)
	}

	// Test after first execution
	js.IncrementRepeatCounter("repeat-job")

	current, target = js.GetRepeatInfo("repeat-job")
	if current != 1 || target != 3 {
		t.Errorf("Expected current=1, target=3, got current=%d, target=%d", current, target)
	}

	if !js.ShouldRepeatJob("repeat-job") {
		t.Error("Job should still be repeatable after first execution")
	}

	// Test after all executions
	js.IncrementRepeatCounter("repeat-job")
	js.IncrementRepeatCounter("repeat-job")

	current, target = js.GetRepeatInfo("repeat-job")
	if current != 3 || target != 3 {
		t.Errorf("Expected current=3, target=3, got current=%d, target=%d", current, target)
	}

	if js.ShouldRepeatJob("repeat-job") {
		t.Error("Job should not be repeatable after all executions")
	}
}

func TestJobScheduler_IsJobFullyCompleted(t *testing.T) {
	js := NewJobScheduler()

	// Job without repeat
	job1 := &Job{Name: "job1", ID: "job1", Steps: []*Step{}}
	if err := js.AddJob(job1); err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	// Not completed yet
	if js.isJobFullyCompleted("job1") {
		t.Error("Job should not be fully completed initially")
	}

	// Set to completed but counter not matching
	js.SetJobStatus("job1", JobCompleted, true)
	if js.isJobFullyCompleted("job1") {
		t.Error("Job should not be fully completed without proper counter")
	}

	// Increment counter to match target
	js.IncrementRepeatCounter("job1")
	if !js.isJobFullyCompleted("job1") {
		t.Error("Job should be fully completed when counter matches target")
	}

	// Job with repeat
	job2 := &Job{
		Name:   "job2",
		ID:     "job2",
		Repeat: &Repeat{Count: 2, Interval: Interval{Duration: 0}},
		Steps:  []*Step{},
	}
	if err := js.AddJob(job2); err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	js.SetJobStatus("job2", JobCompleted, true)
	js.IncrementRepeatCounter("job2")

	// Only one execution completed out of two
	if js.isJobFullyCompleted("job2") {
		t.Error("Job should not be fully completed with only one execution")
	}

	// Complete second execution
	js.IncrementRepeatCounter("job2")
	if !js.isJobFullyCompleted("job2") {
		t.Error("Job should be fully completed after all executions")
	}
}

func TestJobScheduler_GetRunnableJobs(t *testing.T) {
	js := NewJobScheduler()

	// Add jobs with dependencies
	job1 := &Job{Name: "job1", ID: "job1", Steps: []*Step{}}
	job2 := &Job{Name: "job2", ID: "job2", Needs: []string{"job1"}, Steps: []*Step{}}
	job3 := &Job{Name: "job3", ID: "job3", Steps: []*Step{}}

	if err := js.AddJob(job1); err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}
	if err := js.AddJob(job2); err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}
	if err := js.AddJob(job3); err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	runnable := js.GetRunnableJobs()

	// Should be able to run job1 and job3 (no dependencies)
	if len(runnable) != 2 {
		t.Errorf("Expected 2 runnable jobs, got %d", len(runnable))
	}

	containsJob1 := false
	containsJob3 := false
	for _, jobID := range runnable {
		if jobID == "job1" {
			containsJob1 = true
		}
		if jobID == "job3" {
			containsJob3 = true
		}
	}

	if !containsJob1 || !containsJob3 {
		t.Error("Should be able to run job1 and job3")
	}

	// Complete job1
	js.SetJobStatus("job1", JobCompleted, true)
	js.IncrementRepeatCounter("job1")

	// Mark job3 as running
	js.SetJobStatus("job3", JobRunning, false)

	runnable = js.GetRunnableJobs()

	// Now only job2 should be runnable
	if len(runnable) != 1 || runnable[0] != "job2" {
		t.Errorf("Expected only job2 to be runnable, got %v", runnable)
	}
}

func TestJobScheduler_AllJobsCompleted(t *testing.T) {
	js := NewJobScheduler()

	// No jobs
	if !js.AllJobsCompleted() {
		t.Error("Should return true when no jobs exist")
	}

	// Add jobs
	job1 := &Job{Name: "job1", ID: "job1", Steps: []*Step{}}
	job2 := &Job{Name: "job2", ID: "job2", Steps: []*Step{}}

	if err := js.AddJob(job1); err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}
	if err := js.AddJob(job2); err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	// Jobs pending
	if js.AllJobsCompleted() {
		t.Error("Should return false when jobs are pending")
	}

	// One job completed, one pending
	js.SetJobStatus("job1", JobCompleted, true)
	if js.AllJobsCompleted() {
		t.Error("Should return false when some jobs are still pending")
	}

	// One job completed, one failed
	js.SetJobStatus("job2", JobFailed, false)
	if !js.AllJobsCompleted() {
		t.Error("Should return true when all jobs are completed or failed")
	}
}

func TestJobScheduler_SetJobStatus(t *testing.T) {
	js := NewJobScheduler()

	job := &Job{Name: "job1", ID: "job1", Steps: []*Step{}}
	if err := js.AddJob(job); err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	// Test setting to running
	js.SetJobStatus("job1", JobRunning, false)
	if js.status["job1"] != JobRunning {
		t.Error("Job status should be set to JobRunning")
	}

	// Test setting to completed with success
	js.SetJobStatus("job1", JobCompleted, true)
	if js.status["job1"] != JobCompleted {
		t.Error("Job status should be set to JobCompleted")
	}
	if !js.results["job1"] {
		t.Error("Job result should be set to true")
	}

	// Test setting to failed
	js.SetJobStatus("job1", JobFailed, false)
	if js.status["job1"] != JobFailed {
		t.Error("Job status should be set to JobFailed")
	}
	if js.results["job1"] {
		t.Error("Job result should be set to false")
	}
}

func TestJobScheduler_ConcurrentAccess(t *testing.T) {
	js := NewJobScheduler()

	// Add a job
	job := &Job{Name: "concurrent-job", ID: "concurrent-job", Steps: []*Step{}}
	if err := js.AddJob(job); err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	// Test concurrent access to scheduler methods with smaller iteration count
	done := make(chan bool, 4)

	// Goroutine 1: Check if job can run (read-only operations)
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 10; i++ {
			js.CanRunJob("concurrent-job")
			time.Sleep(time.Millisecond)
		}
	}()

	// Goroutine 2: Increment repeat counter (write operations)
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 10; i++ {
			js.IncrementRepeatCounter("concurrent-job")
			time.Sleep(time.Millisecond)
		}
	}()

	// Goroutine 3: Check repeat status (read-only operations)
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 10; i++ {
			js.ShouldRepeatJob("concurrent-job")
			time.Sleep(time.Millisecond)
		}
	}()

	// Goroutine 4: Get runnable jobs (read-only operations)
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 10; i++ {
			js.GetRunnableJobs()
			time.Sleep(time.Millisecond)
		}
	}()

	// Wait for all goroutines to complete with timeout
	timeout := time.After(5 * time.Second)
	for i := 0; i < 4; i++ {
		select {
		case <-done:
			// Goroutine completed successfully
		case <-timeout:
			t.Fatal("Test timed out waiting for goroutines to complete")
		}
	}

	// Verify final state
	current, _ := js.GetRepeatInfo("concurrent-job")
	if current != 10 {
		t.Errorf("Expected counter to be 10, got %d", current)
	}
}

package probe

import (
	"testing"
)

func TestJobScheduler_MarkJobsWithFailedDependencies(t *testing.T) {
	t.Run("multi-level dependency failures are properly marked", func(t *testing.T) {
		// This test verifies that jobs with JobFailed dependencies are also marked as failed
		// Previously, only JobCompleted with !results was checked, causing deadlock
		scheduler := NewJobScheduler()

		// Level 1 job that will fail
		job1 := &Job{
			Name:  "level1",
			ID:    "level1",
			Steps: []*Step{},
		}
		if err := scheduler.AddJob(job1); err != nil {
			t.Fatalf("Failed to add job1: %v", err)
		}

		// Level 2 job that depends on level1
		job2 := &Job{
			Name:  "level2",
			ID:    "level2",
			Needs: []string{"level1"},
			Steps: []*Step{},
		}
		if err := scheduler.AddJob(job2); err != nil {
			t.Fatalf("Failed to add job2: %v", err)
		}

		// Level 3 job that depends on level2
		job3 := &Job{
			Name:  "level3",
			ID:    "level3",
			Needs: []string{"level2"},
			Steps: []*Step{},
		}
		if err := scheduler.AddJob(job3); err != nil {
			t.Fatalf("Failed to add job3: %v", err)
		}

		// Simulate level1 completing with failure
		scheduler.SetJobStatus("level1", JobCompleted, false)

		// Call MarkJobsWithFailedDependencies
		// With the fix, this should mark both level2 and level3 as failed in a single pass
		// because MarkJobsWithFailedDependencies now checks for both JobFailed and JobCompleted+!results
		skipped := scheduler.MarkJobsWithFailedDependencies()

		// Both level2 and level3 should be skipped
		// Note: The order may vary due to map iteration, so we check the length and contents
		if len(skipped) != 2 {
			t.Errorf("Expected 2 jobs to be skipped (level2 and level3), got %d: %v", len(skipped), skipped)
		}

		skippedMap := make(map[string]bool)
		for _, id := range skipped {
			skippedMap[id] = true
		}
		if !skippedMap["level2"] || !skippedMap["level3"] {
			t.Errorf("Expected level2 and level3 to be skipped, got: %v", skipped)
		}

		// Verify both are marked as JobFailed
		if scheduler.status["level2"] != JobFailed {
			t.Errorf("Expected level2 status to be JobFailed, got: %v", scheduler.status["level2"])
		}
		if scheduler.status["level3"] != JobFailed {
			t.Errorf("Expected level3 status to be JobFailed, got: %v", scheduler.status["level3"])
		}

		// Second call should return no skipped jobs (all are already failed)
		skipped2 := scheduler.MarkJobsWithFailedDependencies()
		if len(skipped2) != 0 {
			t.Errorf("Expected no more jobs to be skipped, got: %v", skipped2)
		}

		// Verify AllJobsCompleted returns true
		if !scheduler.AllJobsCompleted() {
			t.Error("Expected AllJobsCompleted to return true after all jobs are in terminal state")
		}
	})

	t.Run("independent jobs are not affected by failed dependencies", func(t *testing.T) {
		scheduler := NewJobScheduler()

		// Job that will fail
		job1 := &Job{
			Name:  "job1",
			ID:    "job1",
			Steps: []*Step{},
		}
		if err := scheduler.AddJob(job1); err != nil {
			t.Fatalf("Failed to add job1: %v", err)
		}

		// Job that depends on job1
		job2 := &Job{
			Name:  "job2",
			ID:    "job2",
			Needs: []string{"job1"},
			Steps: []*Step{},
		}
		if err := scheduler.AddJob(job2); err != nil {
			t.Fatalf("Failed to add job2: %v", err)
		}

		// Independent job
		job3 := &Job{
			Name:  "job3",
			ID:    "job3",
			Steps: []*Step{},
		}
		if err := scheduler.AddJob(job3); err != nil {
			t.Fatalf("Failed to add job3: %v", err)
		}

		// Simulate job1 completing with failure
		scheduler.SetJobStatus("job1", JobCompleted, false)

		// Mark failed dependencies
		skipped := scheduler.MarkJobsWithFailedDependencies()

		// Only job2 should be skipped, not job3
		if len(skipped) != 1 || skipped[0] != "job2" {
			t.Errorf("Expected only job2 to be skipped, got: %v", skipped)
		}

		// job3 should still be pending
		if scheduler.status["job3"] != JobPending {
			t.Errorf("Expected job3 to remain JobPending, got: %v", scheduler.status["job3"])
		}

		// job3 should be runnable
		if !scheduler.CanRunJob("job3") {
			t.Error("Expected job3 to be runnable")
		}
	})
}

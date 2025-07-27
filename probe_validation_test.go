package probe

import (
	"strings"
	"testing"
)

func TestProbe_validateIDs(t *testing.T) {
	tests := []struct {
		name        string
		workflow    Workflow
		expectError bool
		errorType   string
	}{
		{
			name: "no duplicate IDs",
			workflow: Workflow{
				Jobs: []Job{
					{
						ID:   "job1",
						Name: "Job 1",
						Steps: []*Step{
							{ID: "step1", Name: "Step 1", Uses: "hello"},
							{ID: "step2", Name: "Step 2", Uses: "hello"},
						},
					},
					{
						ID:   "job2",
						Name: "Job 2",
						Steps: []*Step{
							{ID: "step3", Name: "Step 3", Uses: "hello"},
							{ID: "step4", Name: "Step 4", Uses: "hello"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "duplicate job IDs",
			workflow: Workflow{
				Jobs: []Job{
					{
						ID:   "job1",
						Name: "Job 1",
						Steps: []*Step{
							{ID: "step1", Name: "Step 1", Uses: "hello"},
						},
					},
					{
						ID:   "job1", // duplicate job ID
						Name: "Job 2",
						Steps: []*Step{
							{ID: "step2", Name: "Step 2", Uses: "hello"},
						},
					},
				},
			},
			expectError: true,
			errorType:   "duplicate_job_id",
		},
		{
			name: "duplicate step IDs across jobs",
			workflow: Workflow{
				Jobs: []Job{
					{
						ID:   "job1",
						Name: "Job 1",
						Steps: []*Step{
							{ID: "step1", Name: "Step 1", Uses: "hello"},
						},
					},
					{
						ID:   "job2",
						Name: "Job 2",
						Steps: []*Step{
							{ID: "step1", Name: "Step 2", Uses: "hello"}, // duplicate step ID
						},
					},
				},
			},
			expectError: true,
			errorType:   "duplicate_step_id",
		},
		{
			name: "duplicate step IDs within same job",
			workflow: Workflow{
				Jobs: []Job{
					{
						ID:   "job1",
						Name: "Job 1",
						Steps: []*Step{
							{ID: "step1", Name: "Step 1", Uses: "hello"},
							{ID: "step1", Name: "Step 2", Uses: "hello"}, // duplicate step ID
						},
					},
				},
			},
			expectError: true,
			errorType:   "duplicate_step_id",
		},
		{
			name: "empty IDs are allowed for validation",
			workflow: Workflow{
				Jobs: []Job{
					{
						ID:   "",
						Name: "Job 1",
						Steps: []*Step{
							{ID: "", Name: "Step 1", Uses: "hello"},
							{ID: "", Name: "Step 2", Uses: "hello"},
						},
					},
					{
						ID:   "",
						Name: "Job 2",
						Steps: []*Step{
							{ID: "", Name: "Step 3", Uses: "hello"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "mixed empty and non-empty IDs",
			workflow: Workflow{
				Jobs: []Job{
					{
						ID:   "job1",
						Name: "Job 1",
						Steps: []*Step{
							{ID: "step1", Name: "Step 1", Uses: "hello"},
							{ID: "", Name: "Step 2", Uses: "hello"}, // empty ID is OK
						},
					},
					{
						ID:   "", // empty job ID is OK
						Name: "Job 2",
						Steps: []*Step{
							{ID: "step2", Name: "Step 3", Uses: "hello"},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{workflow: tt.workflow}
			err := p.validateIDs()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				
				if probeErr, ok := err.(*ProbeError); ok {
					if probeErr.Operation != tt.errorType {
						t.Errorf("expected error operation %s, got %s", tt.errorType, probeErr.Operation)
					}
				} else {
					t.Errorf("expected ProbeError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestProbe_initializeEmptyIDs(t *testing.T) {
	tests := []struct {
		name             string
		workflow         Workflow
		expectedJobIDs   []string
		expectedStepIDs  [][]string // stepIDs per job
	}{
		{
			name: "initialize all empty IDs",
			workflow: Workflow{
				Jobs: []Job{
					{
						ID:   "",
						Name: "Job 1",
						Steps: []*Step{
							{ID: "", Name: "Step 1", Uses: "hello"},
							{ID: "", Name: "Step 2", Uses: "hello"},
						},
					},
					{
						ID:   "",
						Name: "Job 2",
						Steps: []*Step{
							{ID: "", Name: "Step 3", Uses: "hello"},
						},
					},
				},
			},
			expectedJobIDs: []string{"job_0", "job_1"},
			expectedStepIDs: [][]string{
				{"step_0", "step_1"},
				{"step_0"},
			},
		},
		{
			name: "preserve existing IDs",
			workflow: Workflow{
				Jobs: []Job{
					{
						ID:   "custom_job",
						Name: "Job 1",
						Steps: []*Step{
							{ID: "custom_step", Name: "Step 1", Uses: "hello"},
							{ID: "", Name: "Step 2", Uses: "hello"},
						},
					},
					{
						ID:   "",
						Name: "Job 2",
						Steps: []*Step{
							{ID: "", Name: "Step 3", Uses: "hello"},
							{ID: "another_custom", Name: "Step 4", Uses: "hello"},
						},
					},
				},
			},
			expectedJobIDs: []string{"custom_job", "job_1"},
			expectedStepIDs: [][]string{
				{"custom_step", "step_1"},
				{"step_0", "another_custom"},
			},
		},
		{
			name: "no changes needed",
			workflow: Workflow{
				Jobs: []Job{
					{
						ID:   "job1",
						Name: "Job 1",
						Steps: []*Step{
							{ID: "step1", Name: "Step 1", Uses: "hello"},
							{ID: "step2", Name: "Step 2", Uses: "hello"},
						},
					},
				},
			},
			expectedJobIDs: []string{"job1"},
			expectedStepIDs: [][]string{
				{"step1", "step2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{workflow: tt.workflow}
			p.initializeEmptyIDs()

			// Check job IDs
			for i, expectedJobID := range tt.expectedJobIDs {
				if i >= len(p.workflow.Jobs) {
					t.Errorf("expected %d jobs, got %d", len(tt.expectedJobIDs), len(p.workflow.Jobs))
					continue
				}
				if p.workflow.Jobs[i].ID != expectedJobID {
					t.Errorf("job %d: expected ID %s, got %s", i, expectedJobID, p.workflow.Jobs[i].ID)
				}
			}

			// Check step IDs
			for jobIdx, expectedStepIDs := range tt.expectedStepIDs {
				if jobIdx >= len(p.workflow.Jobs) {
					continue
				}
				job := p.workflow.Jobs[jobIdx]
				for stepIdx, expectedStepID := range expectedStepIDs {
					if stepIdx >= len(job.Steps) {
						t.Errorf("job %d: expected %d steps, got %d", jobIdx, len(expectedStepIDs), len(job.Steps))
						continue
					}
					if job.Steps[stepIdx].ID != expectedStepID {
						t.Errorf("job %d, step %d: expected ID %s, got %s", 
							jobIdx, stepIdx, expectedStepID, job.Steps[stepIdx].ID)
					}
				}
			}
		})
	}
}

func TestProbe_validateAndInitializeIDs_Integration(t *testing.T) {
	tests := []struct {
		name                string
		workflow            Workflow
		expectValidationErr bool
		expectedJobIDs      []string
		expectedStepIDs     [][]string
	}{
		{
			name: "successful validation and initialization",
			workflow: Workflow{
				Jobs: []Job{
					{
						ID:   "",
						Name: "Job 1",
						Steps: []*Step{
							{ID: "", Name: "Step 1", Uses: "hello"},
							{ID: "custom_step", Name: "Step 2", Uses: "hello"},
						},
					},
					{
						ID:   "custom_job",
						Name: "Job 2",
						Steps: []*Step{
							{ID: "", Name: "Step 3", Uses: "hello"},
						},
					},
				},
			},
			expectValidationErr: false,
			expectedJobIDs:      []string{"job_0", "custom_job"},
			expectedStepIDs: [][]string{
				{"step_0", "custom_step"},
				{"step_0"},
			},
		},
		{
			name: "validation fails before initialization",
			workflow: Workflow{
				Jobs: []Job{
					{
						ID:   "duplicate",
						Name: "Job 1",
						Steps: []*Step{
							{ID: "step1", Name: "Step 1", Uses: "hello"},
						},
					},
					{
						ID:   "duplicate", // duplicate job ID
						Name: "Job 2",
						Steps: []*Step{
							{ID: "step2", Name: "Step 2", Uses: "hello"},
						},
					},
				},
			},
			expectValidationErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{workflow: tt.workflow}
			
			// Run validation
			err := p.validateIDs()
			
			if tt.expectValidationErr {
				if err == nil {
					t.Errorf("expected validation error but got none")
				}
				return // Don't test initialization if validation fails
			}
			
			if err != nil {
				t.Errorf("unexpected validation error: %v", err)
				return
			}

			// Run initialization
			p.initializeEmptyIDs()

			// Check results
			for i, expectedJobID := range tt.expectedJobIDs {
				if p.workflow.Jobs[i].ID != expectedJobID {
					t.Errorf("job %d: expected ID %s, got %s", i, expectedJobID, p.workflow.Jobs[i].ID)
				}
			}

			for jobIdx, expectedStepIDs := range tt.expectedStepIDs {
				job := p.workflow.Jobs[jobIdx]
				for stepIdx, expectedStepID := range expectedStepIDs {
					if job.Steps[stepIdx].ID != expectedStepID {
						t.Errorf("job %d, step %d: expected ID %s, got %s", 
							jobIdx, stepIdx, expectedStepID, job.Steps[stepIdx].ID)
					}
				}
			}
		})
	}
}

func TestProbe_validateIDs_ErrorMessages(t *testing.T) {
	t.Run("duplicate job ID error message", func(t *testing.T) {
		workflow := Workflow{
			Jobs: []Job{
				{ID: "job1", Name: "Job 1", Steps: []*Step{{Uses: "hello"}}},
				{ID: "job1", Name: "Job 2", Steps: []*Step{{Uses: "hello"}}},
			},
		}
		p := &Probe{workflow: workflow}
		err := p.validateIDs()

		if err == nil {
			t.Fatal("expected error but got none")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "duplicate job ID 'job1'") {
			t.Errorf("error message should contain duplicate job ID info, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "job 1") && !strings.Contains(errMsg, "job 0") {
			t.Errorf("error message should contain job indices, got: %s", errMsg)
		}
	})

	t.Run("duplicate step ID error message", func(t *testing.T) {
		workflow := Workflow{
			Jobs: []Job{
				{
					ID: "job1", Name: "Job 1",
					Steps: []*Step{
						{ID: "step1", Uses: "hello"},
						{ID: "step1", Uses: "hello"},
					},
				},
			},
		}
		p := &Probe{workflow: workflow}
		err := p.validateIDs()

		if err == nil {
			t.Fatal("expected error but got none")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "duplicate step ID 'step1'") {
			t.Errorf("error message should contain duplicate step ID info, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "job[0].step[1]") {
			t.Errorf("error message should contain step location info, got: %s", errMsg)
		}
	})
}
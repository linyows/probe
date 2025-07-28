package probe

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestLoad(t *testing.T) {
	p := &Probe{
		FilePath: "./testdata/workflow.yml",
		Config: Config{
			Log:     os.Stdout,
			Verbose: true,
			RT:      false,
		},
	}
	err := p.Load()
	if err != nil {
		t.Errorf("probe load error %s", err)
	}
	expects, err := os.ReadFile("./testdata/marshaled-workflow.yml")
	if err != nil {
		t.Errorf("file read error %s", err)
	}
	got, _ := yaml.Marshal(p.workflow)
	if string(got) != string(expects) {
		t.Errorf("\nExpected:\n%s\nGot:\n%s", expects, got)
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		verbose  bool
		expected *Probe
	}{
		{
			name:    "create new probe with verbose true",
			path:    "./test.yml",
			verbose: true,
			expected: &Probe{
				FilePath: "./test.yml",
				Config: Config{
					Log:     os.Stdout,
					Verbose: true,
					RT:      false,
				},
			},
		},
		{
			name:    "create new probe with verbose false",
			path:    "./another.yml",
			verbose: false,
			expected: &Probe{
				FilePath: "./another.yml",
				Config: Config{
					Log:     os.Stdout,
					Verbose: false,
					RT:      false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.path, tt.verbose)
			if got.FilePath != tt.expected.FilePath {
				t.Errorf("FilePath = %v, want %v", got.FilePath, tt.expected.FilePath)
			}
			if got.Config.Verbose != tt.expected.Config.Verbose {
				t.Errorf("Config.Verbose = %v, want %v", got.Config.Verbose, tt.expected.Config.Verbose)
			}
			if got.Config.RT != tt.expected.Config.RT {
				t.Errorf("Config.RT = %v, want %v", got.Config.RT, tt.expected.Config.RT)
			}
		})
	}
}

func TestIsYamlFile(t *testing.T) {
	p := &Probe{}
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"yml extension", "test.yml", true},
		{"yaml extension", "test.yaml", true},
		{"txt extension", "test.txt", false},
		{"no extension", "test", false},
		{"yml in middle", "test.yml.txt", false},
		{"yaml in middle", "test.yaml.txt", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.isYamlFile(tt.filename)
			if got != tt.expected {
				t.Errorf("isYamlFile(%s) = %v, want %v", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestYamlFiles(t *testing.T) {
	p := &Probe{}
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "single file",
			filePath: "./testdata/workflow.yml",
			wantErr:  false,
		},
		{
			name:     "nonexistent file",
			filePath: "./nonexistent.yml",
			wantErr:  true,
		},
		{
			name:     "directory",
			filePath: "./testdata",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.FilePath = tt.filePath
			files, err := p.yamlFiles()
			if (err != nil) != tt.wantErr {
				t.Errorf("yamlFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(files) == 0 {
				t.Errorf("yamlFiles() returned empty files slice")
			}
		})
	}
}

func TestReadYamlFiles(t *testing.T) {
	p := &Probe{}

	tests := []struct {
		name    string
		paths   []string
		wantErr bool
	}{
		{
			name:    "existing file",
			paths:   []string{"./testdata/workflow.yml"},
			wantErr: false,
		},
		{
			name:    "nonexistent file",
			paths:   []string{"nonexistent.yml"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.readYamlFiles(tt.paths)
			if (err != nil) != tt.wantErr {
				t.Errorf("readYamlFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	p := &Probe{}

	tests := []struct {
		name     string
		data     map[string]any
		defaults map[string]any
		expected map[string]any
	}{
		{
			name: "simple defaults",
			data: map[string]any{
				"existing": "value",
			},
			defaults: map[string]any{
				"new_key": "default_value",
			},
			expected: map[string]any{
				"existing": "value",
				"new_key":  "default_value",
			},
		},
		{
			name: "nested defaults",
			data: map[string]any{
				"nested": map[string]any{
					"existing": "value",
				},
			},
			defaults: map[string]any{
				"nested": map[string]any{
					"new_key": "default_value",
				},
			},
			expected: map[string]any{
				"nested": map[string]any{
					"existing": "value",
					"new_key":  "default_value",
				},
			},
		},
		{
			name: "overwrite protection",
			data: map[string]any{
				"key": "original",
			},
			defaults: map[string]any{
				"key": "default",
			},
			expected: map[string]any{
				"key": "original",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.setDefaults(tt.data, tt.defaults)
			if !reflect.DeepEqual(tt.data, tt.expected) {
				t.Errorf("setDefaults() = %v, want %v", tt.data, tt.expected)
			}
		})
	}
}

func TestExitStatus(t *testing.T) {
	p := &Probe{}

	// Initialize workflow
	p.workflow = Workflow{exitStatus: 0}

	if got := p.ExitStatus(); got != 0 {
		t.Errorf("ExitStatus() = %v, want %v", got, 0)
	}

	// Set exit status to 1
	p.workflow.exitStatus = 1

	if got := p.ExitStatus(); got != 1 {
		t.Errorf("ExitStatus() = %v, want %v", got, 1)
	}
}

func TestLoadWithInvalidYaml(t *testing.T) {
	p := &Probe{
		FilePath: "./nonexistent.yml",
		Config: Config{
			Log:     io.Discard,
			Verbose: false,
			RT:      false,
		},
	}

	err := p.Load()
	if err == nil {
		t.Error("Load() should return error for invalid file path, but got nil")
	}
}

func TestYamlFilesWithMultiplePaths(t *testing.T) {
	p := &Probe{}

	// Test with comma-separated paths
	p.FilePath = "./testdata/workflow.yml,./testdata/marshaled-workflow.yml"
	files, err := p.yamlFiles()
	if err != nil {
		t.Errorf("yamlFiles() with multiple paths error = %v", err)
		return
	}

	if len(files) != 2 {
		t.Errorf("yamlFiles() with multiple paths returned %d files, want 2", len(files))
	}

	// Check that both files are included
	foundWorkflow := false
	foundMarshaled := false
	for _, file := range files {
		if strings.Contains(file, "workflow.yml") {
			foundWorkflow = true
		}
		if strings.Contains(file, "marshaled-workflow.yml") {
			foundMarshaled = true
		}
	}

	if !foundWorkflow || !foundMarshaled {
		t.Errorf("yamlFiles() did not find both expected files: %v", files)
	}
}

func TestDoWithInvalidPath(t *testing.T) {
	p := &Probe{
		FilePath: "./nonexistent.yml",
		Config: Config{
			Log:     io.Discard,
			Verbose: false,
			RT:      false,
		},
	}

	err := p.Do()
	if err == nil {
		t.Error("Do() should return error for invalid file path, but got nil")
	}
}

// Validation tests
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
		name            string
		workflow        Workflow
		expectedJobIDs  []string
		expectedStepIDs [][]string // stepIDs per job
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

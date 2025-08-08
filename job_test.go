package probe

import (
	"testing"
)

func TestJob_validateSteps(t *testing.T) {
	tests := []struct {
		name      string
		steps     []*Step
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid steps without outputs",
			steps: []*Step{
				{Name: "Step 1", Uses: "echo"},
				{Name: "Step 2", Uses: "echo"},
			},
			expectErr: false,
		},
		{
			name: "valid steps with results and id",
			steps: []*Step{
				{ID: "auth_step", Name: "Auth", Uses: "http", Outputs: map[string]string{"token": "{{ res.body.token }}"}},
				{Name: "Simple step", Uses: "echo"},
			},
			expectErr: false,
		},
		{
			name: "invalid: outputs without id",
			steps: []*Step{
				{Name: "Step with outputs but no id", Uses: "http", Outputs: map[string]string{"data": "{{ res.body }}"}},
			},
			expectErr: true,
			errMsg:    "step with outputs must have an 'id' field",
		},
		{
			name: "invalid step id format - uppercase",
			steps: []*Step{
				{ID: "Auth_Step", Name: "Auth", Uses: "http", Outputs: map[string]string{"token": "{{ res.body.token }}"}},
			},
			expectErr: true,
			errMsg:    "invalid step ID 'Auth_Step' - only [a-z0-9_-] characters are allowed",
		},
		{
			name: "invalid step id format - special chars",
			steps: []*Step{
				{ID: "auth@step", Name: "Auth", Uses: "http", Outputs: map[string]string{"token": "{{ res.body.token }}"}},
			},
			expectErr: true,
			errMsg:    "invalid step ID 'auth@step' - only [a-z0-9_-] characters are allowed",
		},
		{
			name: "duplicate step ids",
			steps: []*Step{
				{ID: "auth_step", Name: "Auth 1", Uses: "http", Outputs: map[string]string{"token": "{{ res.body.token }}"}},
				{ID: "auth_step", Name: "Auth 2", Uses: "http", Outputs: map[string]string{"data": "{{ res.body.data }}"}},
			},
			expectErr: true,
			errMsg:    "duplicate step ID 'auth_step'",
		},
		{
			name: "valid step ids with allowed characters",
			steps: []*Step{
				{ID: "step_1", Name: "Step 1", Uses: "http", Outputs: map[string]string{"data": "{{ res.body }}"}},
				{ID: "step-2", Name: "Step 2", Uses: "http", Outputs: map[string]string{"result": "{{ res.status }}"}},
				{ID: "step3", Name: "Step 3", Uses: "http", Outputs: map[string]string{"count": "{{ res.body.count }}"}},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{
				Name:  "Test Job",
				Steps: tt.steps,
			}

			err := job.validateSteps()

			if tt.expectErr {
				if err == nil {
					t.Errorf("validateSteps() expected error but got none")
				} else if tt.errMsg != "" && err.Error() != "" {
					// Check if error message contains expected substring
					if len(tt.errMsg) > 0 {
						errorStr := err.Error()
						found := false
						// Simple substring check
						for i := 0; i <= len(errorStr)-len(tt.errMsg); i++ {
							if errorStr[i:i+len(tt.errMsg)] == tt.errMsg {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("validateSteps() error = %v, want error containing %v", err, tt.errMsg)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("validateSteps() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestIsValidStepID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{
			name:     "valid lowercase letters",
			id:       "auth",
			expected: true,
		},
		{
			name:     "valid with numbers",
			id:       "step1",
			expected: true,
		},
		{
			name:     "valid with underscores",
			id:       "auth_step",
			expected: true,
		},
		{
			name:     "valid with hyphens",
			id:       "auth-step",
			expected: true,
		},
		{
			name:     "valid mixed",
			id:       "auth_step-1",
			expected: true,
		},
		{
			name:     "invalid uppercase",
			id:       "Auth",
			expected: false,
		},
		{
			name:     "invalid special chars",
			id:       "auth@step",
			expected: false,
		},
		{
			name:     "invalid spaces",
			id:       "auth step",
			expected: false,
		},
		{
			name:     "invalid dots",
			id:       "auth.step",
			expected: false,
		},
		{
			name:     "empty string",
			id:       "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidStepID(tt.id)
			if result != tt.expected {
				t.Errorf("isValidStepID(%q) = %v, want %v", tt.id, result, tt.expected)
			}
		})
	}
}

func TestJob_shouldSkip(t *testing.T) {
	tests := []struct {
		name     string
		skipIf   string
		vars     map[string]any
		expected bool
	}{
		{
			name:     "empty skipif",
			skipIf:   "",
			expected: false,
		},
		{
			name:     "skipif true",
			skipIf:   "true",
			expected: true,
		},
		{
			name:     "skipif false",
			skipIf:   "false",
			expected: false,
		},
		{
			name:     "skipif with variable true",
			skipIf:   "vars.skip_job",
			vars:     map[string]any{"skip_job": true},
			expected: true,
		},
		{
			name:     "skipif with variable false",
			skipIf:   "vars.skip_job",
			vars:     map[string]any{"skip_job": false},
			expected: false,
		},
		{
			name:     "skipif with expression",
			skipIf:   `vars.env == "test"`,
			vars:     map[string]any{"env": "test"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{
				Name:   "Test Job",
				SkipIf: tt.skipIf,
			}

			ctx := JobContext{
				Vars:    tt.vars,
				Outputs: NewOutputs(),
				Printer: NewPrinter(false, []string{}),
			}
			job.ctx = &ctx

			expr := &Expr{}
			result := job.shouldSkip(expr, ctx)

			if result != tt.expected {
				t.Errorf("shouldSkip() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestJob_shouldSkip_errorHandling(t *testing.T) {
	tests := []struct {
		name     string
		skipIf   string
		expected bool
	}{
		{
			name:     "invalid expression",
			skipIf:   "invalid syntax ===",
			expected: false, // Should not skip on error
		},
		{
			name:     "non-boolean result",
			skipIf:   `"string_value"`,
			expected: false, // Should not skip on type error
		},
		{
			name:     "undefined variable",
			skipIf:   "vars.undefined_var",
			expected: false, // Should not skip on evaluation error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{
				Name:   "Test Job",
				SkipIf: tt.skipIf,
			}

			ctx := JobContext{
				Vars:    map[string]any{},
				Outputs: NewOutputs(),
				Printer: NewPrinter(false, []string{}),
			}
			job.ctx = &ctx

			expr := &Expr{}
			result := job.shouldSkip(expr, ctx)

			if result != tt.expected {
				t.Errorf("shouldSkip() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestJob_handleSkip(t *testing.T) {
	job := &Job{
		ID:   "test-job",
		Name: "Test Job",
	}

	// Create a result with the job entry
	result := NewResult()
	jobResult := &JobResult{
		JobID:   "test-job",
		JobName: "Test Job",
		Status:  "running",
		Success: false,
	}
	result.Jobs["test-job"] = jobResult

	ctx := JobContext{
		Result:  result,
		Printer: NewPrinter(false, []string{}),
		Config:  Config{Verbose: false},
	}
	job.ctx = &ctx

	// Call handleSkip
	job.handleSkip(ctx)

	// Verify the job was marked as skipped
	if jobResult.Status != "skipped" {
		t.Errorf("Expected job status to be 'skipped', got '%s'", jobResult.Status)
	}

	if !jobResult.Success {
		t.Errorf("Expected skipped job to be marked as successful")
	}
}

func TestJob_RunIndependently_Success(t *testing.T) {
	job := &Job{
		Name: "Test Job",
		Steps: []*Step{
			{
				Name: "Success Step",
				Uses: "hello",
				With: map[string]any{"msg": "test message"},
			},
		},
	}

	vars := map[string]any{"test_var": "test_value"}
	success, outputs, report, errorMsg, duration := job.RunIndependently(vars, false, "test-job")

	if !success {
		t.Errorf("Expected job to succeed, but it failed with error: %s", errorMsg)
	}

	if errorMsg != "" {
		t.Errorf("Expected no error message for successful job, got: %s", errorMsg)
	}

	if outputs == nil {
		t.Errorf("Expected outputs to be non-nil")
	}

	if report == "" {
		t.Errorf("Expected non-empty report")
	}

	if duration <= 0 {
		t.Errorf("Expected positive duration, got: %v", duration)
	}
}

func TestJob_RunIndependently_Failure(t *testing.T) {
	job := &Job{
		Name: "Test Job",
		Steps: []*Step{
			{
				Name: "Failure Step", 
				Uses: "shell",
				With: map[string]any{
					"cmd": "exit 1",
				},
				Test: "res.code == 0", // This will fail since exit code is 1
			},
		},
	}

	vars := map[string]any{"test_var": "test_value"}
	success, outputs, report, errorMsg, duration := job.RunIndependently(vars, false, "test-job")

	if success {
		t.Errorf("Expected job to fail, but it succeeded")
	}

	if errorMsg == "" {
		t.Errorf("Expected error message for failed job")
	}

	if outputs == nil {
		t.Errorf("Expected outputs to be non-nil even for failed job")
	}

	if report == "" {
		t.Errorf("Expected non-empty report even for failed job")
	}

	if duration <= 0 {
		t.Errorf("Expected positive duration, got: %v", duration)
	}
}

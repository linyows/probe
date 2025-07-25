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
				{ID: "auth_step", Name: "Auth", Uses: "http", Results: map[string]string{"token": "{{ res.body.token }}"}},
				{Name: "Simple step", Uses: "echo"},
			},
			expectErr: false,
		},
		{
			name: "invalid: results without id",
			steps: []*Step{
				{Name: "Step with results but no id", Uses: "http", Results: map[string]string{"data": "{{ res.body }}"}},
			},
			expectErr: true,
			errMsg:    "step with results must have an 'id' field",
		},
		{
			name: "invalid step id format - uppercase",
			steps: []*Step{
				{ID: "Auth_Step", Name: "Auth", Uses: "http", Results: map[string]string{"token": "{{ res.body.token }}"}},
			},
			expectErr: true,
			errMsg:    "invalid step ID 'Auth_Step' - only [a-z0-9_-] characters are allowed",
		},
		{
			name: "invalid step id format - special chars",
			steps: []*Step{
				{ID: "auth@step", Name: "Auth", Uses: "http", Results: map[string]string{"token": "{{ res.body.token }}"}},
			},
			expectErr: true,
			errMsg:    "invalid step ID 'auth@step' - only [a-z0-9_-] characters are allowed",
		},
		{
			name: "duplicate step ids",
			steps: []*Step{
				{ID: "auth_step", Name: "Auth 1", Uses: "http", Results: map[string]string{"token": "{{ res.body.token }}"}},
				{ID: "auth_step", Name: "Auth 2", Uses: "http", Results: map[string]string{"data": "{{ res.body.data }}"}},
			},
			expectErr: true,
			errMsg:    "duplicate step ID 'auth_step'",
		},
		{
			name: "valid step ids with allowed characters",
			steps: []*Step{
				{ID: "step_1", Name: "Step 1", Uses: "http", Results: map[string]string{"data": "{{ res.body }}"}},
				{ID: "step-2", Name: "Step 2", Uses: "http", Results: map[string]string{"result": "{{ res.status }}"}},
				{ID: "step3", Name: "Step 3", Uses: "http", Results: map[string]string{"count": "{{ res.body.count }}"}},
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
package probe

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestOsEnv(t *testing.T) {
	os.Setenv("HOST", "http://localhost")
	os.Setenv("TOKEN", "secrets")
	defer func() {
		os.Unsetenv("HOST")
		os.Unsetenv("TOKEN")
	}()

	expected := map[string]string{
		"HOST":  "http://localhost",
		"TOKEN": "secrets",
	}

	wf := &Workflow{}
	actual := wf.Env()

	if actual["HOST"] != expected["HOST"] || actual["TOKEN"] != expected["TOKEN"] {
		t.Errorf("expected %+v, got %+v", expected, actual)
	}
}

func TestEvalVars(t *testing.T) {
	tests := []struct {
		name     string
		wf       *Workflow
		expected map[string]string
		err      error
	}{
		{
			name: "use expr",
			wf: &Workflow{
				Name: "Test",
				Vars: map[string]string{
					"host":  "{HOST ?? 'http://localhost:3000'}",
					"token": "{TOKEN}",
				},
				env: map[string]string{
					"TOKEN": "secrets",
				},
			},
			expected: map[string]string{
				"host":  "http://localhost:3000",
				"token": "secrets",
			},
			err: nil,
		},
		{
			name: "not exists environment",
			wf: &Workflow{
				Name: "Test",
				Vars: map[string]string{
					"host":  "{HOST}",
					"token": "{TOKEN}",
				},
				env: map[string]string{
					"TOKEN": "secrets",
				},
			},
			expected: nil,
			err:      fmt.Errorf("environment(HOST) is nil"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tt.wf.evalVars()
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("expected error %+v, got %+v", tt.err, err)
			}
			if !reflect.DeepEqual(tt.expected, actual) {
				t.Errorf("expected %+v, got %+v", tt.expected, actual)
			}
		})
	}
}

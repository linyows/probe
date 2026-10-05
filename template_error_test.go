package probe

import (
	"strings"
	"sync"
	"testing"

	"github.com/linyows/probe/expr"
)

// runStepForTemplateError runs st in a job context of its own and returns
// how many times the action ran, the step's result and whether the job failed.
func runStepForTemplateError(t *testing.T, st *Step, vars map[string]any) (int, StepResult, bool) {
	t.Helper()

	calls := 0
	st.actionRunner = &CountingMockActionRunner{
		result:    map[string]any{"req": map[string]any{}, "res": map[string]any{"code": 200}},
		callCount: &calls,
	}
	st.Expr = &expr.Expr{}

	rs := NewResult()
	rs.Jobs["j"] = &JobResult{JobName: "j", JobID: "j"}
	jCtx := &JobContext{
		Vars:         vars,
		CurrentJobID: "j",
		Printer:      newBufferPrinter(),
		Result:       rs,
		Outputs:      NewOutputs(),
		countersMu:   &sync.Mutex{},
	}

	st.SetCtx(*jCtx, nil)
	st.Do(jCtx)

	results := rs.Jobs["j"].StepResults
	if len(results) != 1 {
		t.Fatalf("expected 1 step result, got %d", len(results))
	}
	return calls, results[0], jCtx.Failed
}

func TestStep_TemplateErrorFailsStep(t *testing.T) {
	tests := []struct {
		name    string
		step    *Step
		vars    map[string]any
		message []string
	}{
		{
			name: "a value in with",
			step: &Step{
				Name: "login",
				Uses: "http",
				With: map[string]any{
					"url": "http://localhost",
					"headers": map[string]any{
						"authorization": "Bearer {{outputs.login.token}}",
					},
				},
			},
			message: []string{"with.headers.authorization: {{outputs.login.token}}: "},
		},
		{
			name: "a value in with that is a single template",
			step: &Step{
				Name: "port",
				Uses: "http",
				With: map[string]any{"items": []any{"{{vars.base.port}}"}},
			},
			vars:    map[string]any{"base": nil},
			message: []string{"with.items[0]: {{vars.base.port}}: "},
		},
		{
			name: "a step var",
			step: &Step{
				Name: "var",
				Uses: "http",
				Vars: map[string]any{"token": "{{outputs.login.token}}"},
			},
			message: []string{"vars.token: {{outputs.login.token}}: "},
		},
		{
			name: "a step var that is a map",
			step: &Step{
				Name: "map var",
				Uses: "http",
				Vars: map[string]any{"auth": map[string]any{"user": "{{1 +}}"}},
			},
			message: []string{"vars.auth.user: {{1 +}}: "},
		},
		{
			name: "the step name",
			step: &Step{
				Name: "check {{outputs.login.token}}",
				Uses: "http",
			},
			message: []string{"name: {{outputs.login.token}}: "},
		},
		{
			name: "the name and a var together",
			step: &Step{
				Name: "check {{outputs.a.b}}",
				Uses: "http",
				Vars: map[string]any{"x": "{{outputs.c.d}}"},
			},
			message: []string{"name: {{outputs.a.b}}: ", "vars.x: {{outputs.c.d}}: "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, result, failed := runStepForTemplateError(t, tt.step, tt.vars)
			if calls != 0 {
				t.Errorf("the action ran %d times, want 0", calls)
			}
			if !failed {
				t.Error("the job is not marked failed")
			}
			if result.Status != StatusError {
				t.Errorf("status = %v, want %v", result.Status, StatusError)
			}
			if result.Failure == nil || result.Failure.Kind != FailureTemplate {
				t.Fatalf("failure = %+v, want kind %q", result.Failure, FailureTemplate)
			}
			for _, m := range tt.message {
				if !strings.Contains(result.Failure.Message, m) {
					t.Errorf("failure message %q does not contain %q", result.Failure.Message, m)
				}
			}
		})
	}
}

func TestStep_TemplateErrorInSkippedStep(t *testing.T) {
	st := &Step{
		Name:   "skipped",
		Uses:   "http",
		SkipIf: "true",
		Vars:   map[string]any{"token": "{{outputs.login.token}}"},
	}
	calls, result, failed := runStepForTemplateError(t, st, nil)
	if calls != 0 || failed {
		t.Errorf("calls = %d, failed = %v, want a skipped step", calls, failed)
	}
	if result.Status != StatusSkipped {
		t.Errorf("status = %v, want %v", result.Status, StatusSkipped)
	}
}

func TestStep_TemplateWithUndefinedNameRuns(t *testing.T) {
	st := &Step{
		Name: "fallback {{vars.missing ?? 'x'}}",
		Uses: "http",
		With: map[string]any{"url": "{{vars.base ?? 'http://localhost'}}"},
		Test: "res.code == 200",
	}
	calls, result, failed := runStepForTemplateError(t, st, map[string]any{})
	if calls != 1 || failed {
		t.Errorf("calls = %d, failed = %v, want one successful run", calls, failed)
	}
	if result.Name != "fallback x" {
		t.Errorf("name = %q, want %q", result.Name, "fallback x")
	}
}

func TestWorkflow_evalVarsTemplateError(t *testing.T) {
	wf := &Workflow{
		Name: "Test",
		Vars: map[string]any{
			"ok":   "fine",
			"bad":  "{{1 +}}",
			"auth": map[string]any{"user": "{{nothing.here}}"},
		},
		env: map[string]string{"UNUSED": ""},
	}
	vars, err := wf.evalVars()
	if err == nil {
		t.Fatal("expected an error")
	}
	if vars != nil {
		t.Errorf("expected no vars, got %#v", vars)
	}
	for _, m := range []string{"vars.auth.user: {{nothing.here}}: ", "vars.bad: {{1 +}}: "} {
		if !strings.Contains(err.Error(), m) {
			t.Errorf("error %q does not contain %q", err.Error(), m)
		}
	}
	if strings.Index(err.Error(), "vars.auth") > strings.Index(err.Error(), "vars.bad") {
		t.Errorf("errors are not in name order: %q", err.Error())
	}
}

func TestWorkflow_TemplateErrorExitCode(t *testing.T) {
	runner := NewMockActionRunner()
	runner.SetResult("ok", map[string]any{
		"req": map[string]any{},
		"res": map[string]any{"code": 200},
	})
	w := &Workflow{
		Name: "exit",
		Jobs: []Job{{Name: "a", Steps: []*Step{{
			Name:         "s",
			Uses:         "ok",
			With:         map[string]any{"url": "{{outputs.missing.url}}"},
			actionRunner: runner,
		}}}},
		printer: newBufferPrinter(),
	}
	if err := w.Start(Config{}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if w.exitStatus != ExitTestFailed {
		t.Errorf("exit status = %d, want %d", w.exitStatus, ExitTestFailed)
	}
}

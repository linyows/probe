package probe

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// echoRunner answers every action with its own parameters as res, as hello
// does, after sleeping for with.sleep when that is set. It records the
// values of with.got, so that a test can see what each run read.
type echoRunner struct {
	mu  sync.Mutex
	got []string
}

func (r *echoRunner) RunActions(name string, with map[string]any, opts RunOptions) (map[string]any, error) {
	if s, ok := with["sleep"].(string); ok {
		if d, err := time.ParseDuration(s); err == nil {
			time.Sleep(d)
		}
	}
	if got, ok := with["got"].(string); ok {
		r.mu.Lock()
		r.got = append(r.got, got)
		r.mu.Unlock()
	}
	res := make(map[string]any, len(with))
	for k, v := range with {
		res[k] = v
	}
	return map[string]any{"req": with, "res": res, "status": 0}, nil
}

// repeatOutputsWorkflow repeats a job that publishes a value naming its run,
// waits, and reads it back, followed by a job that reads it after. One step
// is skipped by the last run.
func repeatOutputsWorkflow(runner ActionRunner, async bool) *Workflow {
	repeated := Job{
		Name: "repeated",
		ID:   "repeated",
		Repeat: &Repeat{
			Count:    4,
			Interval: Interval{Duration: 10 * time.Millisecond},
			Async:    async,
		},
		Steps: []*Step{
			{
				Name: "publish", ID: "pub", Uses: "hello", actionRunner: runner,
				With:    map[string]any{"v": "run-{{repeat_index}}"},
				Outputs: map[string]string{"v": "res.v"},
			},
			{
				// The last run skips this one, so what the jobs after read is
				// what the run before left.
				Name: "early", ID: "early", Uses: "hello", actionRunner: runner,
				SkipIf:  "repeat_index == 3",
				With:    map[string]any{"e": "early-{{repeat_index}}"},
				Outputs: map[string]string{"e": "res.e"},
			},
			{
				// Long enough for the other runs to publish meanwhile.
				Name: "pause", ID: "pause", Uses: "hello", actionRunner: runner,
				With: map[string]any{"sleep": "100ms"},
			},
			{
				Name: "read own", ID: "own", Uses: "hello", actionRunner: runner,
				With: map[string]any{"got": "{{outputs.pub.v}}", "want": "run-{{repeat_index}}"},
				Test: "res.got == res.want",
			},
		},
	}
	after := Job{
		Name:  "after",
		ID:    "after",
		Needs: []string{"repeated"},
		Steps: []*Step{
			{
				Name: "read last", ID: "last", Uses: "hello", actionRunner: runner,
				With: map[string]any{"got": "{{outputs.pub.v}}", "early": "{{outputs.early.e}}"},
				Test: `res.got == "run-3" && res.early == "early-2"`,
			},
		},
	}
	return &Workflow{
		Name:    "repeat outputs",
		Jobs:    []Job{repeated, after},
		printer: newBufferPrinter(),
	}
}

func TestRepeatedJobOutputs(t *testing.T) {
	// A repeated job used to save no outputs at all, so its later steps
	// could not read its earlier ones, nor could the jobs after it. Each run
	// reads its own, also when the runs overlap, and the jobs after read
	// those of the last run.
	for _, async := range []bool{false, true} {
		name := "in turn"
		if async {
			name = "async"
		}
		t.Run(name, func(t *testing.T) {
			runner := &echoRunner{}
			w := repeatOutputsWorkflow(runner, async)
			if err := w.Start(Config{}); err != nil {
				t.Fatalf("Start() error: %v", err)
			}
			if w.exitStatus != 0 {
				runner.mu.Lock()
				got := strings.Join(runner.got, ", ")
				runner.mu.Unlock()
				t.Errorf("exit status = %d, want 0; values read: %s", w.exitStatus, got)
			}
		})
	}
}

func TestRunOutputs(t *testing.T) {
	parent := NewOutputs()
	_ = parent.Set("setup", map[string]any{"base": "b", "token": "parent-token"})

	run := newRunOutputs(parent)
	if err := run.Set("pub", map[string]any{"v": "run"}); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	// The run sees the parent's outputs and its own
	if v, _ := run.GetFlat("base"); v != "b" {
		t.Errorf("run base = %v, want the parent's", v)
	}
	if got, _ := run.Get("pub"); got["v"] != "run" {
		t.Errorf("run pub.v = %v, want its own", got["v"])
	}
	if all := run.GetAll(); all["base"] != "b" || all["v"] != "run" {
		t.Errorf("run GetAll = %v, want both", all)
	}

	// The parent does not see the run's until it is published
	if _, ok := parent.Get("pub"); ok {
		t.Error("parent has the run's outputs before publish")
	}

	// A name the parent's steps took stays theirs, as in a workflow
	err := run.Set("other", map[string]any{"token": "run-token"})
	var taken *NameTakenError
	if !errors.As(err, &taken) {
		t.Errorf("Set() = %v, want a name-taken warning", err)
	}
	if v, _ := run.GetFlat("token"); v != "parent-token" {
		t.Errorf("run token = %v, want the parent's", v)
	}

	if err := parent.publish(run); err == nil {
		t.Error("publish() gave no warning for the taken name")
	}
	if got, _ := parent.Get("pub"); got["v"] != "run" {
		t.Errorf("parent pub.v = %v after publish, want the run's", got["v"])
	}
	if v, _ := parent.GetFlat("v"); v != "run" {
		t.Errorf("parent v = %v after publish, want the run's", v)
	}
}

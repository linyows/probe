package probe

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
)

// countingRunner is a StatefulActionRunner whose actions count their calls
// in the state they keep. It records the count each call was given, or "-"
// for no state. A call with fail fails, and one with keep leaves no state.
type countingRunner struct {
	mu       sync.Mutex
	received []string
}

func (r *countingRunner) RunActions(name string, with map[string]any, opts RunOptions) (map[string]any, error) {
	ret, _, err := r.RunActionsWithState(name, with, nil, opts)
	return ret, err
}

func (r *countingRunner) RunActionsWithState(name string, with, state map[string]any, opts RunOptions) (map[string]any, map[string]any, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, _ := state["n"].(int)
	got := "-"
	if state != nil {
		got = fmt.Sprint(n)
	}
	r.received = append(r.received, name+":"+got)
	if with["fail"] == true {
		return nil, nil, errors.New("failed")
	}
	if with["keep"] == true {
		return map[string]any{"status": 0}, nil, nil
	}
	return map[string]any{"status": 0}, map[string]any{"n": n + 1}, nil
}

func TestActionStateIsKeptInARunOfAJob(t *testing.T) {
	runner := &countingRunner{}
	step := func(uses string, with map[string]any) *Step {
		return &Step{Name: uses, Uses: uses, With: with, actionRunner: runner}
	}

	workflow := &Workflow{
		Name: "state",
		Jobs: []Job{
			{
				Name: "counts",
				ID:   "counts",
				Steps: []*Step{
					step("counter", nil),
					step("counter", map[string]any{"fail": true}),
					step("counter", map[string]any{"keep": true}),
					step("other", nil),
					step("counter", nil),
				},
				Repeat: &Repeat{Count: 2},
			},
		},
		printer: newBufferPrinter(),
	}
	_ = workflow.Start(Config{})

	// A failed call and one that leaves no state keep the state as it was,
	// each action keeps its own, and the second run starts over.
	run := []string{"counter:-", "counter:1", "counter:1", "other:-", "counter:1"}
	want := append(append([]string{}, run...), run...)
	if !reflect.DeepEqual(runner.received, want) {
		t.Errorf("states given = %q, want %q", runner.received, want)
	}
}

func TestActionStateIsNotSharedBetweenJobs(t *testing.T) {
	runner := &countingRunner{}
	job := func(id string) Job {
		return Job{
			Name:  id,
			ID:    id,
			Steps: []*Step{{Name: "a", Uses: "counter", actionRunner: runner}, {Name: "b", Uses: "counter", actionRunner: runner}},
		}
	}
	second := job("second")
	second.Needs = []string{"first"}
	workflow := &Workflow{
		Name:    "state",
		Jobs:    []Job{job("first"), second},
		printer: newBufferPrinter(),
	}
	if err := workflow.Start(Config{}); err != nil {
		t.Fatalf("workflow failed: %v", err)
	}

	want := []string{"counter:-", "counter:1", "counter:-", "counter:1"}
	if !reflect.DeepEqual(runner.received, want) {
		t.Errorf("states given = %q, want %q", runner.received, want)
	}
}

func TestMockActionRunnerCarriesState(t *testing.T) {
	m := NewMockActionRunner()
	m.States["counter"] = map[string]any{"n": 1}

	var r ActionRunner = m
	sr, ok := r.(StatefulActionRunner)
	if !ok {
		t.Fatal("MockActionRunner should carry state")
	}
	_, state, err := sr.RunActionsWithState("counter", nil, map[string]any{"n": 0}, RunOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(state, map[string]any{"n": 1}) {
		t.Errorf("state = %v", state)
	}
	if !reflect.DeepEqual(m.Received["counter"], []map[string]any{{"n": 0}}) {
		t.Errorf("received = %v", m.Received["counter"])
	}
}

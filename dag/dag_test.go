package dag

import (
	"reflect"
	"testing"
)

func TestDetectCycleFn_NoCycle(t *testing.T) {
	getDeps := func(id string) []string {
		switch id {
		case "A":
			return []string{"B", "C"}
		case "B":
			return []string{"D"}
		case "C":
			return []string{"D"}
		default:
			return nil
		}
	}

	cycle := DetectCycleFn([]string{"A", "B", "C", "D"}, getDeps)
	if cycle != nil {
		t.Errorf("expected no cycle, got %v", cycle)
	}
}

func TestDetectCycleFn_WithCycle(t *testing.T) {
	getDeps := func(id string) []string {
		switch id {
		case "A":
			return []string{"B"}
		case "B":
			return []string{"C"}
		case "C":
			return []string{"A"} // Cycle back to A
		default:
			return nil
		}
	}

	cycle := DetectCycleFn([]string{"A", "B", "C"}, getDeps)
	if cycle == nil {
		t.Error("expected cycle, got nil")
	}
}

func TestHasCycleFn_NoCycle(t *testing.T) {
	getDeps := func(id int) []int {
		switch id {
		case 1:
			return []int{2, 3}
		case 2:
			return []int{4}
		default:
			return nil
		}
	}

	if HasCycleFn([]int{1, 2, 3, 4}, getDeps) {
		t.Error("expected no cycle")
	}
}

func TestHasCycleFn_WithCycle(t *testing.T) {
	getDeps := func(id int) []int {
		switch id {
		case 1:
			return []int{2}
		case 2:
			return []int{1} // Cycle
		default:
			return nil
		}
	}

	if !HasCycleFn([]int{1, 2}, getDeps) {
		t.Error("expected cycle")
	}
}

func TestFindRootsFn(t *testing.T) {
	getDeps := func(id string) []string {
		switch id {
		case "A":
			return []string{"B", "C"}
		case "B":
			return []string{"D"}
		case "C":
			return []string{"D"}
		default:
			return nil
		}
	}

	roots := FindRootsFn([]string{"A", "B", "C", "D"}, getDeps)
	if len(roots) != 1 || roots[0] != "A" {
		t.Errorf("expected root [A], got %v", roots)
	}
}

func TestFindLeavesFn(t *testing.T) {
	getDeps := func(id string) []string {
		switch id {
		case "A":
			return []string{"B", "C"}
		case "B":
			return []string{"D"}
		case "C":
			return []string{"D"}
		default:
			return nil
		}
	}

	leaves := FindLeavesFn([]string{"A", "B", "C", "D"}, getDeps)
	if len(leaves) != 1 || leaves[0] != "D" {
		t.Errorf("expected leaf [D], got %v", leaves)
	}
}

func TestTopologicalSortFn_Simple(t *testing.T) {
	getDeps := func(id string) []string {
		switch id {
		case "A":
			return []string{"B"}
		case "B":
			return []string{"C"}
		default:
			return nil
		}
	}

	sorted, err := TopologicalSortFn([]string{"A", "B", "C"}, getDeps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// C should come before B, B before A
	expected := []string{"C", "B", "A"}
	if !reflect.DeepEqual(sorted, expected) {
		t.Errorf("expected %v, got %v", expected, sorted)
	}
}

func TestTopologicalSortFn_WithCycle(t *testing.T) {
	getDeps := func(id string) []string {
		switch id {
		case "A":
			return []string{"B"}
		case "B":
			return []string{"A"} // Cycle
		default:
			return nil
		}
	}

	_, err := TopologicalSortFn([]string{"A", "B"}, getDeps)
	if err != ErrCycleDetected {
		t.Errorf("expected ErrCycleDetected, got %v", err)
	}
}

func TestComputeDescendantsFn(t *testing.T) {
	getDeps := func(id string) []string {
		switch id {
		case "A":
			return []string{"B", "C"}
		case "B":
			return []string{"D"}
		case "C":
			return []string{"D"}
		default:
			return nil
		}
	}

	descendants := ComputeDescendantsFn([]string{"A", "B", "C", "D"}, "D", getDeps)
	// D is depended on by B and C, which are depended on by A
	// So descendants of D are B, C, A
	if len(descendants) != 3 {
		t.Errorf("expected 3 descendants, got %v", descendants)
	}
}

func TestComputeAncestorsFn(t *testing.T) {
	getDeps := func(id string) []string {
		switch id {
		case "A":
			return []string{"B", "C"}
		case "B":
			return []string{"D"}
		case "C":
			return []string{"D"}
		default:
			return nil
		}
	}

	ancestors := ComputeAncestorsFn([]string{"A", "B", "C", "D"}, "A", getDeps)
	// A depends on B and C, which depend on D
	// So ancestors of A are B, C, D
	if len(ancestors) != 3 {
		t.Errorf("expected 3 ancestors, got %v", ancestors)
	}
}

// Task implements CycleDetectable for testing
type Task struct {
	name string
	deps []string
}

func (t Task) ID() string           { return t.name }
func (t Task) Dependencies() []string { return t.deps }

func TestDetectCycle_Interface(t *testing.T) {
	tasks := []Task{
		{name: "A", deps: []string{"B"}},
		{name: "B", deps: []string{"C"}},
		{name: "C", deps: nil},
	}

	cycle := DetectCycle[string](tasks)
	if cycle != nil {
		t.Errorf("expected no cycle, got %v", cycle)
	}
}

func TestFindRoots_Interface(t *testing.T) {
	tasks := []Task{
		{name: "A", deps: []string{"B"}},
		{name: "B", deps: []string{"C"}},
		{name: "C", deps: nil},
	}

	roots := FindRoots[string](tasks)
	if len(roots) != 1 || roots[0] != "A" {
		t.Errorf("expected root [A], got %v", roots)
	}
}

package actions

import (
	"slices"
	"testing"
)

func TestKeepingAreBuiltin(t *testing.T) {
	names := Names()
	for _, name := range Keeping() {
		if !slices.Contains(names, name) {
			t.Errorf("%s keeps to the guard but is not a built-in action", name)
		}
	}
	if !slices.IsSorted(Keeping()) {
		t.Errorf("Keeping() = %v, want it sorted", Keeping())
	}
}

func TestKeepingIsACopy(t *testing.T) {
	k := Keeping()
	k[0] = "shell"
	if slices.Contains(Keeping(), "shell") {
		t.Error("changing what Keeping returns should not change the actions that keep to the guard")
	}
}

package asciidag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadGolden loads expected output from testdata directory
func loadGolden(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name+".txt")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", path, err)
	}
	return strings.TrimSpace(string(data))
}

func TestExampleSimpleChain(t *testing.T) {
	dag := FromEdges(
		[]Node{{ID: 1, Label: "A"}, {ID: 2, Label: "B"}, {ID: 3, Label: "C"}},
		[]Edge{{From: 1, To: 2}, {From: 2, To: 3}},
	)

	expected := loadGolden(t, "simple_chain")
	actual := strings.TrimSpace(dag.Render())

	if actual != expected {
		t.Errorf("mismatch:\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestExampleDiamond(t *testing.T) {
	dag := FromEdges(
		[]Node{
			{ID: 1, Label: "Top"},
			{ID: 2, Label: "Left"},
			{ID: 3, Label: "Right"},
			{ID: 4, Label: "Bottom"},
		},
		[]Edge{
			{From: 1, To: 2},
			{From: 1, To: 3},
			{From: 2, To: 4},
			{From: 3, To: 4},
		},
	)

	expected := loadGolden(t, "diamond")
	actual := strings.TrimSpace(dag.Render())

	if actual != expected {
		t.Errorf("mismatch:\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestExampleDivergence(t *testing.T) {
	dag := FromEdges(
		[]Node{
			{ID: 1, Label: "Root"},
			{ID: 2, Label: "A"},
			{ID: 3, Label: "B"},
			{ID: 4, Label: "C"},
		},
		[]Edge{
			{From: 1, To: 2},
			{From: 1, To: 3},
			{From: 1, To: 4},
		},
	)

	expected := loadGolden(t, "divergence")
	actual := strings.TrimSpace(dag.Render())

	if actual != expected {
		t.Errorf("mismatch:\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestExampleConvergence(t *testing.T) {
	dag := FromEdges(
		[]Node{
			{ID: 1, Label: "A"},
			{ID: 2, Label: "B"},
			{ID: 3, Label: "C"},
			{ID: 4, Label: "Result"},
		},
		[]Edge{
			{From: 1, To: 4},
			{From: 2, To: 4},
			{From: 3, To: 4},
		},
	)

	expected := loadGolden(t, "convergence")
	actual := strings.TrimSpace(dag.Render())

	if actual != expected {
		t.Errorf("mismatch:\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestExampleCycleDetection(t *testing.T) {
	dag := New()
	dag.AddNode(1, "A")
	dag.AddNode(2, "B")
	dag.AddEdge(1, 2)
	dag.AddEdge(2, 1)

	expected := loadGolden(t, "cycle_detection")
	actual := strings.TrimSpace(dag.Render())

	if actual != expected {
		t.Errorf("mismatch:\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestExampleDOTSimple(t *testing.T) {
	dot := `
digraph simple {
    A -> B;
    B -> C;
    C -> D;
}
`
	dag, err := ParseDOT(dot)
	if err != nil {
		t.Fatal(err)
	}

	expected := loadGolden(t, "dot_simple")
	actual := strings.TrimSpace(dag.Render())

	if actual != expected {
		t.Errorf("mismatch:\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestExampleDOTDiamond(t *testing.T) {
	dot := `
digraph diamond {
    root [label="Root"];
    left [label="Left"];
    right [label="Right"];
    bottom [label="Bottom"];

    root -> left;
    root -> right;
    left -> bottom;
    right -> bottom;
}
`
	dag, err := ParseDOT(dot)
	if err != nil {
		t.Fatal(err)
	}

	expected := loadGolden(t, "dot_diamond")
	actual := strings.TrimSpace(dag.Render())

	if actual != expected {
		t.Errorf("mismatch:\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestExampleDOTPipeline(t *testing.T) {
	dot := `
digraph build {
    parse [label="Parse"];
    analyze [label="Analyze"];
    optimize [label="Optimize"];
    codegen [label="CodeGen"];
    link [label="Link"];

    parse -> analyze -> optimize -> codegen -> link;
}
`
	dag, err := ParseDOT(dot)
	if err != nil {
		t.Fatal(err)
	}

	expected := loadGolden(t, "dot_pipeline")
	actual := strings.TrimSpace(dag.Render())

	if actual != expected {
		t.Errorf("mismatch:\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestExampleDOTConvergence(t *testing.T) {
	dot := `
digraph errors {
    e1 [label="DB Error"];
    e2 [label="Network"];
    e3 [label="Timeout"];
    final [label="Fatal"];

    e1 -> final;
    e2 -> final;
    e3 -> final;
}
`
	dag, err := ParseDOT(dot)
	if err != nil {
		t.Fatal(err)
	}

	expected := loadGolden(t, "dot_convergence")
	actual := strings.TrimSpace(dag.Render())

	if actual != expected {
		t.Errorf("mismatch:\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestExampleDOTToDOT(t *testing.T) {
	dag := FromEdges(
		[]Node{
			{ID: 1, Label: "Start"},
			{ID: 2, Label: "Process"},
			{ID: 3, Label: "End"},
		},
		[]Edge{
			{From: 1, To: 2},
			{From: 2, To: 3},
		},
	)

	expected := loadGolden(t, "dot_to_dot")
	actual := strings.TrimSpace(dag.ToDOT("example"))

	if actual != expected {
		t.Errorf("mismatch:\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

package asciidag

import (
	"strings"
	"testing"
)

func TestParseDOT_SimpleDigraph(t *testing.T) {
	dot := `
digraph G {
    A -> B;
    B -> C;
}
`
	dag, err := ParseDOT(dot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dag.NodeCount() != 3 {
		t.Errorf("expected 3 nodes, got %d", dag.NodeCount())
	}
	if dag.EdgeCount() != 2 {
		t.Errorf("expected 2 edges, got %d", dag.EdgeCount())
	}
}

func TestParseDOT_WithLabels(t *testing.T) {
	dot := `
digraph G {
    A [label="Start"];
    B [label="Middle"];
    C [label="End"];
    A -> B;
    B -> C;
}
`
	dag, err := ParseDOT(dot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dag.NodeCount() != 3 {
		t.Errorf("expected 3 nodes, got %d", dag.NodeCount())
	}

	output := dag.Render()
	if !strings.Contains(output, "[Start]") {
		t.Errorf("output should contain [Start], got:\n%s", output)
	}
	if !strings.Contains(output, "[End]") {
		t.Errorf("output should contain [End], got:\n%s", output)
	}
}

func TestParseDOT_Diamond(t *testing.T) {
	dot := `
digraph diamond {
    root -> left;
    root -> right;
    left -> bottom;
    right -> bottom;
}
`
	dag, err := ParseDOT(dot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dag.NodeCount() != 4 {
		t.Errorf("expected 4 nodes, got %d", dag.NodeCount())
	}
	if dag.EdgeCount() != 4 {
		t.Errorf("expected 4 edges, got %d", dag.EdgeCount())
	}

	if dag.isSimpleChain() {
		t.Error("diamond should not be a simple chain")
	}
}

func TestParseDOT_WithComments(t *testing.T) {
	dot := `
// This is a comment
digraph G {
    // Another comment
    A -> B;
    /* Block comment */
    B -> C;
}
`
	dag, err := ParseDOT(dot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dag.NodeCount() != 3 {
		t.Errorf("expected 3 nodes, got %d", dag.NodeCount())
	}
}

func TestParseDOT_QuotedNames(t *testing.T) {
	dot := `
digraph G {
    "Node A" -> "Node B";
    "Node B" -> "Node C";
}
`
	dag, err := ParseDOT(dot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dag.NodeCount() != 3 {
		t.Errorf("expected 3 nodes, got %d", dag.NodeCount())
	}

	output := dag.Render()
	if !strings.Contains(output, "[Node A]") {
		t.Errorf("output should contain [Node A], got:\n%s", output)
	}
}

func TestParseDOT_StrictDigraph(t *testing.T) {
	dot := `
strict digraph G {
    A -> B;
    B -> C;
}
`
	dag, err := ParseDOT(dot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dag.NodeCount() != 3 {
		t.Errorf("expected 3 nodes, got %d", dag.NodeCount())
	}
}

func TestParseDOT_ComplexGraph(t *testing.T) {
	dot := `
digraph dependencies {
    // Build pipeline
    parse [label="Parse"];
    analyze [label="Analyze"];
    optimize [label="Optimize"];
    codegen [label="Code Gen"];
    link [label="Link"];

    parse -> analyze;
    analyze -> optimize;
    optimize -> codegen;
    codegen -> link;

    // Parallel analysis
    analyze -> lint;
    lint -> report;
}
`
	dag, err := ParseDOT(dot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dag.NodeCount() != 7 {
		t.Errorf("expected 7 nodes, got %d", dag.NodeCount())
	}
	if dag.EdgeCount() != 6 {
		t.Errorf("expected 6 edges, got %d", dag.EdgeCount())
	}
}

func TestToDOT(t *testing.T) {
	dag := FromEdges(
		[]Node{{ID: 1, Label: "A"}, {ID: 2, Label: "B"}, {ID: 3, Label: "C"}},
		[]Edge{{From: 1, To: 2}, {From: 2, To: 3}},
	)

	dot := dag.ToDOT("test")

	if !strings.Contains(dot, "digraph test {") {
		t.Errorf("DOT output should contain graph declaration, got:\n%s", dot)
	}
	if !strings.Contains(dot, "n1 -> n2") {
		t.Errorf("DOT output should contain edge, got:\n%s", dot)
	}
	if !strings.Contains(dot, `label="A"`) {
		t.Errorf("DOT output should contain label, got:\n%s", dot)
	}
}

func TestParseDOT_RoundTrip(t *testing.T) {
	original := FromEdges(
		[]Node{{ID: 1, Label: "Start"}, {ID: 2, Label: "Middle"}, {ID: 3, Label: "End"}},
		[]Edge{{From: 1, To: 2}, {From: 2, To: 3}},
	)

	// Convert to DOT
	dot := original.ToDOT("roundtrip")

	// Parse back
	parsed, err := ParseDOT(dot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.NodeCount() != original.NodeCount() {
		t.Errorf("node count mismatch: original=%d, parsed=%d", original.NodeCount(), parsed.NodeCount())
	}
	if parsed.EdgeCount() != original.EdgeCount() {
		t.Errorf("edge count mismatch: original=%d, parsed=%d", original.EdgeCount(), parsed.EdgeCount())
	}
}

func TestParseDOT_CycleDetection(t *testing.T) {
	dot := `
digraph cycle {
    A -> B;
    B -> C;
    C -> A;
}
`
	dag, err := ParseDOT(dot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !dag.HasCycle() {
		t.Error("should detect cycle")
	}
}

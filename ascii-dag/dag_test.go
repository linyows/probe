package asciidag

import (
	"strings"
	"testing"
)

func TestEmptyDAG(t *testing.T) {
	dag := New()
	if dag.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", dag.NodeCount())
	}
	if dag.EdgeCount() != 0 {
		t.Errorf("expected 0 edges, got %d", dag.EdgeCount())
	}
	output := dag.Render()
	if output != "" {
		t.Errorf("expected empty output, got %q", output)
	}
}

func TestSimpleChain(t *testing.T) {
	dag := FromEdges(
		[]Node{{1, "A"}, {2, "B"}, {3, "C"}},
		[]Edge{{1, 2}, {2, 3}},
	)

	if dag.NodeCount() != 3 {
		t.Errorf("expected 3 nodes, got %d", dag.NodeCount())
	}
	if dag.EdgeCount() != 2 {
		t.Errorf("expected 2 edges, got %d", dag.EdgeCount())
	}

	if !dag.isSimpleChain() {
		t.Error("expected simple chain")
	}

	output := dag.Render()
	if !strings.Contains(output, "[A]") {
		t.Errorf("output should contain [A], got:\n%s", output)
	}
	if !strings.Contains(output, "[B]") {
		t.Errorf("output should contain [B], got:\n%s", output)
	}
	if !strings.Contains(output, "[C]") {
		t.Errorf("output should contain [C], got:\n%s", output)
	}
}

func TestDiamond(t *testing.T) {
	dag := FromEdges(
		[]Node{{1, "Top"}, {2, "Left"}, {3, "Right"}, {4, "Bottom"}},
		[]Edge{{1, 2}, {1, 3}, {2, 4}, {3, 4}},
	)

	if dag.NodeCount() != 4 {
		t.Errorf("expected 4 nodes, got %d", dag.NodeCount())
	}
	if dag.EdgeCount() != 4 {
		t.Errorf("expected 4 edges, got %d", dag.EdgeCount())
	}

	if dag.isSimpleChain() {
		t.Error("diamond should not be a simple chain")
	}

	output := dag.Render()
	if !strings.Contains(output, "[Top]") {
		t.Errorf("output should contain [Top], got:\n%s", output)
	}
	if !strings.Contains(output, "[Bottom]") {
		t.Errorf("output should contain [Bottom], got:\n%s", output)
	}
}

func TestAutoCreatedNodes(t *testing.T) {
	dag := New()
	dag.AddNode(1, "A")
	dag.AddEdge(1, 2) // Node 2 is auto-created

	if dag.NodeCount() != 2 {
		t.Errorf("expected 2 nodes, got %d", dag.NodeCount())
	}

	if !dag.isAutoCreated(2) {
		t.Error("node 2 should be auto-created")
	}
	if dag.isAutoCreated(1) {
		t.Error("node 1 should not be auto-created")
	}

	output := dag.Render()
	if !strings.Contains(output, "[A]") {
		t.Errorf("output should contain [A], got:\n%s", output)
	}
	// Auto-created nodes show as ⟨ID⟩
	if !strings.Contains(output, "⟨2⟩") {
		t.Errorf("output should contain ⟨2⟩ for auto-created node, got:\n%s", output)
	}
}

func TestCycleDetection(t *testing.T) {
	dag := New()
	dag.AddNode(1, "A")
	dag.AddNode(2, "B")
	dag.AddEdge(1, 2)
	dag.AddEdge(2, 1) // Creates a cycle

	if !dag.HasCycle() {
		t.Error("should detect cycle")
	}

	output := dag.Render()
	if !strings.Contains(output, "Cycle") {
		t.Errorf("output should mention cycle, got:\n%s", output)
	}
}

func TestNoCycle(t *testing.T) {
	dag := FromEdges(
		[]Node{{1, "A"}, {2, "B"}},
		[]Edge{{1, 2}},
	)

	if dag.HasCycle() {
		t.Error("should not detect cycle")
	}
}

func TestHorizontalRender(t *testing.T) {
	dag := FromEdges(
		[]Node{{1, "A"}, {2, "B"}, {3, "C"}},
		[]Edge{{1, 2}, {2, 3}},
		WithRenderMode(RenderModeHorizontal),
	)

	output := dag.Render()
	// Horizontal should have arrow right
	if !strings.Contains(output, "→") {
		t.Errorf("horizontal render should contain →, got:\n%s", output)
	}
}

func TestVerticalRender(t *testing.T) {
	dag := FromEdges(
		[]Node{{1, "A"}, {2, "B"}, {3, "C"}},
		[]Edge{{1, 2}, {2, 3}},
		WithRenderMode(RenderModeVertical),
	)

	output := dag.Render()
	// Vertical should have nodes on separate lines
	lines := strings.Split(output, "\n")
	if len(lines) < 3 {
		t.Errorf("vertical render should have multiple lines, got:\n%s", output)
	}
}

func TestComputeLayout(t *testing.T) {
	dag := FromEdges(
		[]Node{{1, "A"}, {2, "B"}, {3, "C"}},
		[]Edge{{1, 2}, {1, 3}},
	)

	ir := dag.ComputeLayout()

	if len(ir.Nodes()) != 3 {
		t.Errorf("expected 3 layout nodes, got %d", len(ir.Nodes()))
	}

	if len(ir.Edges()) != 2 {
		t.Errorf("expected 2 layout edges, got %d", len(ir.Edges()))
	}

	// Node A should be at level 0
	nodeA := ir.NodeByID(1)
	if nodeA == nil {
		t.Fatal("node A not found")
	}
	if nodeA.Level != 0 {
		t.Errorf("node A should be at level 0, got %d", nodeA.Level)
	}

	// Nodes B and C should be at level 1
	nodeB := ir.NodeByID(2)
	nodeC := ir.NodeByID(3)
	if nodeB == nil || nodeC == nil {
		t.Fatal("nodes B or C not found")
	}
	if nodeB.Level != 1 || nodeC.Level != 1 {
		t.Errorf("nodes B and C should be at level 1, got B=%d C=%d", nodeB.Level, nodeC.Level)
	}
}

package asciidag

import (
	"strings"
	"testing"
)

// TestConvergenceBasic tests basic convergence pattern (3 sources -> 1 target)
func TestConvergenceBasic(t *testing.T) {
	dag := FromEdges(
		[]Node{
			{ID: 1, Label: "AAA"},
			{ID: 2, Label: "BBB"},
			{ID: 3, Label: "CCC"},
			{ID: 4, Label: "DDD"},
		},
		[]Edge{
			{From: 1, To: 4},
			{From: 2, To: 4},
			{From: 3, To: 4},
		},
	)

	output := dag.Render()
	lines := strings.Split(output, "\n")

	// Should have at least 4 lines: sources, connection, arrow, target
	// (no vertical line when using ┼)
	if len(lines) < 4 {
		t.Fatalf("expected at least 4 lines, got %d:\n%s", len(lines), output)
	}

	// First line should contain all source labels
	if !strings.Contains(lines[0], "[AAA]") || !strings.Contains(lines[0], "[BBB]") || !strings.Contains(lines[0], "[CCC]") {
		t.Errorf("first line should contain all source labels, got:\n%s", lines[0])
	}

	// Connection line should have └ and ┘
	connectionLine := lines[1]
	if !strings.Contains(connectionLine, "└") {
		t.Errorf("connection line should contain └, got:\n%s", connectionLine)
	}
	if !strings.Contains(connectionLine, "┘") {
		t.Errorf("connection line should contain ┘, got:\n%s", connectionLine)
	}

	// Connection line with middle source should have ┼ (cross)
	if !strings.Contains(connectionLine, "┼") {
		t.Errorf("connection line should contain ┼ for middle source, got:\n%s", connectionLine)
	}

	// Arrow line should have ↓ (directly after connection line when using ┼)
	arrowLine := lines[2]
	if !strings.Contains(arrowLine, "↓") {
		t.Errorf("arrow line should contain ↓, got:\n%s", arrowLine)
	}

	// Target line should contain target label
	targetLine := lines[3]
	if !strings.Contains(targetLine, "[DDD]") {
		t.Errorf("target line should contain [DDD], got:\n%s", targetLine)
	}
}

// TestConvergenceAlignment tests that junction and arrow are aligned
func TestConvergenceAlignment(t *testing.T) {
	dag := FromEdges(
		[]Node{
			{ID: 1, Label: "DB Error"},
			{ID: 2, Label: "Network"},
			{ID: 3, Label: "Timeout"},
			{ID: 4, Label: "Fatal"},
		},
		[]Edge{
			{From: 1, To: 4},
			{From: 2, To: 4},
			{From: 3, To: 4},
		},
	)

	output := dag.Render()
	lines := strings.Split(output, "\n")

	if len(lines) < 4 {
		t.Fatalf("expected at least 4 lines, got %d:\n%s", len(lines), output)
	}

	connectionLine := []rune(lines[1])
	arrowLine := []rune(lines[2]) // Arrow directly after connection when using ┼

	// Connection line should have ┼ for middle source
	if !strings.Contains(string(connectionLine), "┼") {
		t.Errorf("connection line should contain ┼ for middle source, got:\n%s", string(connectionLine))
	}

	// Find junction position (┼)
	junctionPos := -1
	for i, r := range connectionLine {
		if r == '┼' {
			junctionPos = i
			break
		}
	}

	// Find arrow position
	arrowPos := -1
	for i, r := range arrowLine {
		if r == '↓' {
			arrowPos = i
			break
		}
	}

	if junctionPos == -1 {
		t.Fatalf("junction ┼ not found in connection line:\n%s", string(connectionLine))
	}

	if arrowPos == -1 {
		t.Fatalf("arrow ↓ not found in arrow line:\n%s", string(arrowLine))
	}

	// Junction and arrow should be at the same position
	if junctionPos != arrowPos {
		t.Errorf("junction (pos %d) and arrow (pos %d) should be aligned:\n%s\n%s",
			junctionPos, arrowPos, string(connectionLine), string(arrowLine))
	}
}

// TestConvergenceLongLabels tests convergence with varying label lengths
func TestConvergenceLongLabels(t *testing.T) {
	dag := FromEdges(
		[]Node{
			{ID: 1, Label: "Database Error"},
			{ID: 2, Label: "Network"},
			{ID: 3, Label: "Timeout"},
			{ID: 4, Label: "Fatal"},
		},
		[]Edge{
			{From: 1, To: 4},
			{From: 2, To: 4},
			{From: 3, To: 4},
		},
	)

	output := dag.Render()
	lines := strings.Split(output, "\n")

	if len(lines) < 4 {
		t.Fatalf("expected at least 4 lines, got %d:\n%s", len(lines), output)
	}

	connectionLine := []rune(lines[1])
	arrowLine := []rune(lines[2]) // Arrow directly after connection when using ┼

	// Should have ┼ for middle source
	if !strings.Contains(string(connectionLine), "┼") {
		t.Errorf("connection line should contain ┼ for middle source, got:\n%s", string(connectionLine))
	}

	// Find junction position (┼)
	junctionPos := -1
	for i, r := range connectionLine {
		if r == '┼' {
			junctionPos = i
			break
		}
	}

	// Find arrow position
	arrowPos := -1
	for i, r := range arrowLine {
		if r == '↓' {
			arrowPos = i
			break
		}
	}

	if junctionPos == -1 {
		t.Fatalf("junction ┼ not found in connection line:\n%s", string(connectionLine))
	}

	if arrowPos == -1 {
		t.Fatalf("arrow ↓ not found in arrow line:\n%s", string(arrowLine))
	}

	// Junction and arrow should be aligned even with long labels
	if junctionPos != arrowPos {
		t.Errorf("junction (pos %d) and arrow (pos %d) should be aligned:\n%s\n%s",
			junctionPos, arrowPos, string(connectionLine), string(arrowLine))
	}
}

// TestConvergenceHorizontalLine tests that horizontal line connects all sources
func TestConvergenceHorizontalLine(t *testing.T) {
	dag := FromEdges(
		[]Node{
			{ID: 1, Label: "A"},
			{ID: 2, Label: "B"},
			{ID: 3, Label: "C"},
			{ID: 4, Label: "D"},
		},
		[]Edge{
			{From: 1, To: 4},
			{From: 2, To: 4},
			{From: 3, To: 4},
		},
	)

	output := dag.Render()
	lines := strings.Split(output, "\n")

	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d:\n%s", len(lines), output)
	}

	connectionLine := lines[1]

	// Should have horizontal line character
	if !strings.Contains(connectionLine, "─") {
		t.Errorf("connection line should contain horizontal line ─, got:\n%s", connectionLine)
	}

	// Count connection characters
	cornerLeft := strings.Count(connectionLine, "└")
	cornerRight := strings.Count(connectionLine, "┘")

	if cornerLeft != 1 {
		t.Errorf("expected exactly 1 └, got %d in:\n%s", cornerLeft, connectionLine)
	}
	if cornerRight != 1 {
		t.Errorf("expected exactly 1 ┘, got %d in:\n%s", cornerRight, connectionLine)
	}
}

// TestDivergenceBasic tests basic divergence pattern (1 source -> 3 targets)
func TestDivergenceBasic(t *testing.T) {
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

	output := dag.Render()
	lines := strings.Split(output, "\n")

	// Should have source, connection, arrows, targets
	if len(lines) < 4 {
		t.Fatalf("expected at least 4 lines, got %d:\n%s", len(lines), output)
	}

	// Source line should contain root
	if !strings.Contains(lines[0], "[Root]") {
		t.Errorf("first line should contain [Root], got:\n%s", lines[0])
	}

	// Connection line should have divergence characters
	connectionLine := lines[1]
	if !strings.Contains(connectionLine, "┌") {
		t.Errorf("connection line should contain ┌, got:\n%s", connectionLine)
	}
	if !strings.Contains(connectionLine, "┐") {
		t.Errorf("connection line should contain ┐, got:\n%s", connectionLine)
	}

	// Arrow line should have multiple ↓
	arrowLine := lines[2]
	arrowCount := strings.Count(arrowLine, "↓")
	if arrowCount < 2 {
		t.Errorf("arrow line should have multiple ↓, got %d in:\n%s", arrowCount, arrowLine)
	}
}

// TestDiamondPattern tests diamond pattern (divergence then convergence)
func TestDiamondPattern(t *testing.T) {
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

	output := dag.Render()

	// Should contain all labels
	for _, label := range []string{"[Top]", "[Left]", "[Right]", "[Bottom]"} {
		if !strings.Contains(output, label) {
			t.Errorf("output should contain %s, got:\n%s", label, output)
		}
	}

	// Should have both divergence (┌┐) and convergence (└┘) characters
	if !strings.Contains(output, "┌") || !strings.Contains(output, "┐") {
		t.Errorf("diamond should have divergence characters (┌┐), got:\n%s", output)
	}
	if !strings.Contains(output, "└") || !strings.Contains(output, "┘") {
		t.Errorf("diamond should have convergence characters (└┘), got:\n%s", output)
	}
}

package probe

import (
	"strings"
	"testing"
)

func TestNewDagAsciiRenderer(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{
			{Name: "Job 1", ID: "job1"},
			{Name: "Job 2", ID: "job2", Needs: []string{"job1"}},
		},
	}

	renderer := NewDagAsciiRenderer(w)

	if renderer.workflow != w {
		t.Error("workflow should be set")
	}

	if len(renderer.nodes) != 2 {
		t.Errorf("expected 2 boxes, got %d", len(renderer.nodes))
	}

	if renderer.jobIDToIdx["job1"] != 0 {
		t.Errorf("expected job1 at index 0, got %d", renderer.jobIDToIdx["job1"])
	}

	if renderer.jobIDToIdx["job2"] != 1 {
		t.Errorf("expected job2 at index 1, got %d", renderer.jobIDToIdx["job2"])
	}
}

func TestDagAsciiRenderer_Render_EmptyWorkflow(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{},
	}

	renderer := NewDagAsciiRenderer(w)
	result := renderer.Render()

	if result != "" {
		t.Errorf("expected empty string for empty workflow, got %q", result)
	}
}

func TestDagAsciiRenderer_Render_SingleJob(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{
			{
				Name: "Single Job",
				ID:   "single",
				Steps: []*Step{
					{Name: "Step 1", Uses: "hello"},
				},
			},
		},
	}

	renderer := NewDagAsciiRenderer(w)
	result := renderer.Render()

	// Check that the output contains the job name
	if !strings.Contains(result, "Single Job") {
		t.Errorf("expected output to contain 'Single Job', got:\n%s", result)
	}

	// Check that the output contains the step name
	if !strings.Contains(result, "Step 1") {
		t.Errorf("expected output to contain 'Step 1', got:\n%s", result)
	}

	// Check box characters
	if !strings.Contains(result, "╭") || !strings.Contains(result, "╯") {
		t.Errorf("expected output to contain box characters, got:\n%s", result)
	}
}

func TestDagAsciiRenderer_Render_TwoJobsWithDependency(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{
			{
				Name: "First Job",
				ID:   "first",
				Steps: []*Step{
					{Name: "First Step", Uses: "hello"},
				},
			},
			{
				Name:  "Second Job",
				ID:    "second",
				Needs: []string{"first"},
				Steps: []*Step{
					{Name: "Second Step", Uses: "hello"},
				},
			},
		},
	}

	renderer := NewDagAsciiRenderer(w)
	result := renderer.Render()

	// Check that the output contains both job names
	if !strings.Contains(result, "First Job") {
		t.Errorf("expected output to contain 'First Job', got:\n%s", result)
	}
	if !strings.Contains(result, "Second Job") {
		t.Errorf("expected output to contain 'Second Job', got:\n%s", result)
	}

	// Check for connection arrow
	if !strings.Contains(result, "↓") {
		t.Errorf("expected output to contain arrow '↓', got:\n%s", result)
	}
}

func TestDagAsciiRenderer_Render_ParallelJobs(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{
			{
				Name: "Root Job",
				ID:   "root",
				Steps: []*Step{
					{Name: "Root Step", Uses: "hello"},
				},
			},
			{
				Name:  "Branch A",
				ID:    "branch-a",
				Needs: []string{"root"},
				Steps: []*Step{
					{Name: "Branch A Step", Uses: "hello"},
				},
			},
			{
				Name:  "Branch B",
				ID:    "branch-b",
				Needs: []string{"root"},
				Steps: []*Step{
					{Name: "Branch B Step", Uses: "hello"},
				},
			},
		},
	}

	renderer := NewDagAsciiRenderer(w)
	result := renderer.Render()

	// Check that the output contains all job names
	if !strings.Contains(result, "Root Job") {
		t.Errorf("expected output to contain 'Root Job', got:\n%s", result)
	}
	if !strings.Contains(result, "Branch A") {
		t.Errorf("expected output to contain 'Branch A', got:\n%s", result)
	}
	if !strings.Contains(result, "Branch B") {
		t.Errorf("expected output to contain 'Branch B', got:\n%s", result)
	}
}

func TestDagAsciiRenderer_Render_MultipleSteps(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{
			{
				Name: "Multi Step Job",
				ID:   "multi",
				Steps: []*Step{
					{Name: "Step One", Uses: "hello"},
					{Name: "Step Two", Uses: "hello"},
					{Name: "Step Three", Uses: "hello"},
				},
			},
		},
	}

	renderer := NewDagAsciiRenderer(w)
	result := renderer.Render()

	// Check all steps are present
	if !strings.Contains(result, "Step One") {
		t.Errorf("expected output to contain 'Step One', got:\n%s", result)
	}
	if !strings.Contains(result, "Step Two") {
		t.Errorf("expected output to contain 'Step Two', got:\n%s", result)
	}
	if !strings.Contains(result, "Step Three") {
		t.Errorf("expected output to contain 'Step Three', got:\n%s", result)
	}

	// Check step bullet points
	if strings.Count(result, "○") != 3 {
		t.Errorf("expected 3 step bullets, got %d", strings.Count(result, "○"))
	}
}

func TestDagAsciiRenderer_CalculateLevels(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{
			{Name: "Job A", ID: "a"},
			{Name: "Job B", ID: "b", Needs: []string{"a"}},
			{Name: "Job C", ID: "c", Needs: []string{"b"}},
		},
	}

	renderer := NewDagAsciiRenderer(w)
	renderer.calculateLevels()

	if len(renderer.levels) != 3 {
		t.Errorf("expected 3 levels, got %d", len(renderer.levels))
	}

	// Level 0 should have Job A
	if len(renderer.levels[0]) != 1 {
		t.Errorf("expected 1 job at level 0, got %d", len(renderer.levels[0]))
	}

	// Level 1 should have Job B
	if len(renderer.levels[1]) != 1 {
		t.Errorf("expected 1 job at level 1, got %d", len(renderer.levels[1]))
	}

	// Level 2 should have Job C
	if len(renderer.levels[2]) != 1 {
		t.Errorf("expected 1 job at level 2, got %d", len(renderer.levels[2]))
	}
}

func TestDagAsciiRenderer_CalculateLevels_Parallel(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{
			{Name: "Root", ID: "root"},
			{Name: "Left", ID: "left", Needs: []string{"root"}},
			{Name: "Right", ID: "right", Needs: []string{"root"}},
			{Name: "Merge", ID: "merge", Needs: []string{"left", "right"}},
		},
	}

	renderer := NewDagAsciiRenderer(w)
	renderer.calculateLevels()

	if len(renderer.levels) != 3 {
		t.Errorf("expected 3 levels, got %d", len(renderer.levels))
	}

	// Level 0 should have Root
	if len(renderer.levels[0]) != 1 {
		t.Errorf("expected 1 job at level 0, got %d", len(renderer.levels[0]))
	}

	// Level 1 should have Left and Right (parallel)
	if len(renderer.levels[1]) != 2 {
		t.Errorf("expected 2 jobs at level 1, got %d", len(renderer.levels[1]))
	}

	// Level 2 should have Merge
	if len(renderer.levels[2]) != 1 {
		t.Errorf("expected 1 job at level 2, got %d", len(renderer.levels[2]))
	}
}

func TestRenderDagAscii(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{
			{
				Name: "Test Job",
				ID:   "test",
				Steps: []*Step{
					{Name: "Test Step", Uses: "hello"},
				},
			},
		},
	}

	result := w.RenderDagAscii()

	if result == "" {
		t.Error("expected non-empty result")
	}

	if !strings.Contains(result, "Test Job") {
		t.Errorf("expected output to contain 'Test Job', got:\n%s", result)
	}
}

func TestRenderDagAscii_Empty(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{},
	}

	result := w.RenderDagAscii()

	if result != "" {
		t.Errorf("expected empty result for empty workflow, got:\n%s", result)
	}
}

func TestTruncateWithEllipsis(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string unchanged",
			input:    "Hello",
			maxLen:   10,
			expected: "Hello",
		},
		{
			name:     "exact length unchanged",
			input:    "Hello",
			maxLen:   5,
			expected: "Hello",
		},
		{
			name:     "long string truncated",
			input:    "Hello World",
			maxLen:   8,
			expected: "Hello W…",
		},
		{
			name:     "very short maxLen",
			input:    "Hello",
			maxLen:   1,
			expected: "H",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "Japanese text truncated",
			input:    "こんにちは世界",
			maxLen:   5,
			expected: "こんにち…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateWithEllipsis(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateWithEllipsis(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestFixedBoxWidth(t *testing.T) {
	// Test that all boxes have the same fixed width
	w := &Workflow{
		Jobs: []Job{
			{Name: "Short", ID: "short", Steps: []*Step{{Name: "Step", Uses: "hello"}}},
			{Name: "This is a very long job name", ID: "long", Steps: []*Step{{Name: "Step", Uses: "hello"}}},
		},
	}

	renderer := NewDagAsciiRenderer(w)
	renderer.calculateLevels()
	renderer.createBoxes()

	// Both boxes should have the same fixed width
	if renderer.nodes[0].Width != fixedNodeWidth {
		t.Errorf("expected box width %d, got %d", fixedNodeWidth, renderer.nodes[0].Width)
	}
	if renderer.nodes[1].Width != fixedNodeWidth {
		t.Errorf("expected box width %d, got %d", fixedNodeWidth, renderer.nodes[1].Width)
	}
}

func TestRenderDagAsciiJobNode(t *testing.T) {
	job := &Job{
		Name: "Test",
		Steps: []*Step{
			{Name: "Step 1", Uses: "hello"},
		},
	}

	w := &Workflow{Jobs: []Job{*job}}
	renderer := NewDagAsciiRenderer(w)

	box := &DagAsciiJobNode{
		Job:   job,
		JobID: "test",
		Width: fixedNodeWidth,
	}

	lines := renderer.renderDagAsciiJobNode(box)

	// Should have: top border, name, separator, step, bottom border
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}

	// Check top border
	if !strings.HasPrefix(lines[0], "╭") || !strings.HasSuffix(lines[0], "╮") {
		t.Errorf("invalid top border: %s", lines[0])
	}

	// Check bottom border
	if !strings.HasPrefix(lines[4], "╰") || !strings.HasSuffix(lines[4], "╯") {
		t.Errorf("invalid bottom border: %s", lines[4])
	}

	// Check separator
	if !strings.HasPrefix(lines[2], "├") || !strings.HasSuffix(lines[2], "┤") {
		t.Errorf("invalid separator: %s", lines[2])
	}

	// Check all lines have same width
	for i, line := range lines {
		if runeWidth(line) != fixedNodeWidth {
			t.Errorf("line %d has width %d, expected %d: %s", i, runeWidth(line), fixedNodeWidth, line)
		}
	}
}

func TestRenderDagAsciiJobNode_EmbeddedAction(t *testing.T) {
	job := &Job{
		Name: "Test",
		Steps: []*Step{
			{Name: "Normal Step", Uses: "hello"},
			{Name: "Embedded Step", Uses: "embedded"},
		},
	}

	w := &Workflow{Jobs: []Job{*job}}
	renderer := NewDagAsciiRenderer(w)

	box := &DagAsciiJobNode{
		Job:   job,
		JobID: "test",
		Width: fixedNodeWidth,
	}

	lines := renderer.renderDagAsciiJobNode(box)

	// Should have: top border, name, separator, step1, step2, bottom border
	if len(lines) != 6 {
		t.Errorf("expected 6 lines, got %d", len(lines))
	}

	// Check normal step uses ○
	if !strings.Contains(lines[3], "○") {
		t.Errorf("expected normal step to have ○ bullet: %s", lines[3])
	}

	// Check embedded step uses ↗
	if !strings.Contains(lines[4], "↗") {
		t.Errorf("expected embedded step to have ↗ bullet: %s", lines[4])
	}
}

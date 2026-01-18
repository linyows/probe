/*
Package probe provides DAG ASCII rendering tests.

# Golden Test Cases

The golden tests cover comprehensive DAG patterns for ASCII rendering:

	Category              | Case Name                | Structure
	----------------------|--------------------------|------------------------------------------
	Basic                 | single                   | Single job
	                      | embedded_action          | Jobs with embedded actions (↗ bullet)
	                      | truncated_long_names     | Long names truncated with ellipsis (…)
	                      | linear_two               | A → B
	                      | linear_three             | A → B → C
	Divergence            | divergence_two           | A → [B, C]
	                      | divergence_three         | A → [B, C, D]
	                      | divergence_uneven_steps  | A → [B(3 steps), C] (uneven heights)
	Convergence           | convergence_two          | [A, B] → C
	                      | convergence_three        | [A, B, C] → D
	                      | convergence_uneven_steps | [A(3 steps), B] → C (uneven heights)
	                      | multi_child_convergence  | [A, B] → [C(both), D(A only)]
	Complex               | diamond                  | A → [B, C] → D
	                      | hourglass                | [A, B] → C → [D, E]
	Parallel              | parallel_roots           | [A], [B] (independent)
	                      | parallel_chains          | [A→B], [C→D] (independent chains)
	Mixed                 | mixed_standalone         | A → B, C (standalone)
	                      | sorted_children_first    | Jobs with children sorted before others
	                      | wide_divergence          | A → [B, C, D, E]

# Usage

Run golden tests:

	go test -run TestDagAsciiRenderer_Golden -v

Update golden files when output format changes:

	UPDATE_GOLDEN=1 go test -run TestDagAsciiRenderer_Golden

Golden files are stored in testdata/dag_ascii/*.golden.txt
*/
package probe

import (
	"os"
	"path/filepath"
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
	renderer.createNodes()

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

// dagAsciiGoldenTestCase defines a golden test case for DAG ASCII rendering.
type dagAsciiGoldenTestCase struct {
	name     string    // Test case name, used as golden file name (e.g., "diamond" -> "diamond.golden.txt")
	workflow *Workflow // Workflow to render
}

// getDagAsciiGoldenTestCases returns all golden test cases covering various DAG patterns:
//   - Basic: single job, embedded actions (↗ bullet), truncation (…), linear chains (2-3 levels)
//   - Divergence: one parent to multiple children (2-4 branches), including uneven step counts
//   - Convergence: multiple parents to one child (2-3 parents), including uneven step counts
//   - Complex: diamond (diverge then converge), hourglass (converge then diverge)
//   - Parallel: independent roots, independent chains
//   - Mixed: combinations with standalone jobs, sorting verification (children-first)
func getDagAsciiGoldenTestCases() []dagAsciiGoldenTestCase {
	return []dagAsciiGoldenTestCase{
		{
			name: "single",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build", ID: "build", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
				},
			},
		},
		{
			name: "embedded_action",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Setup", ID: "setup", Steps: []*Step{
						{Name: "Checkout", Uses: "git"},
						{Name: "Set env vars", Uses: "embedded"},
						{Name: "Validate config", Uses: "embedded"},
					}},
					{Name: "Test", ID: "test", Needs: []string{"setup"}, Steps: []*Step{
						{Name: "Run tests", Uses: "go"},
						{Name: "Upload coverage", Uses: "embedded"},
					}},
				},
			},
		},
		{
			// Long job names and step names should be truncated with ellipsis (…)
			// fixedNodeWidth=25, innerWidth=23, job name max=21, step name max=20
			name: "truncated_long_names",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build Application Server", ID: "build", Steps: []*Step{
						{Name: "Initialize build environment", Uses: "setup"},
						{Name: "Compile source code", Uses: "go"},
					}},
					{Name: "Run Integration Tests", ID: "test", Needs: []string{"build"}, Steps: []*Step{
						{Name: "Setup test database connection", Uses: "db"},
						{Name: "Execute integration test suite", Uses: "go"},
					}},
				},
			},
		},
		{
			name: "linear_two",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build", ID: "build", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Test", ID: "test", Needs: []string{"build"}, Steps: []*Step{{Name: "Run tests", Uses: "go"}}},
				},
			},
		},
		{
			name: "linear_three",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build", ID: "build", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Test", ID: "test", Needs: []string{"build"}, Steps: []*Step{{Name: "Run tests", Uses: "go"}}},
					{Name: "Deploy", ID: "deploy", Needs: []string{"test"}, Steps: []*Step{{Name: "Deploy app", Uses: "deploy"}}},
				},
			},
		},
		{
			name: "divergence_two",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build", ID: "build", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Unit Test", ID: "unit-test", Needs: []string{"build"}, Steps: []*Step{{Name: "Run unit", Uses: "go"}}},
					{Name: "Lint", ID: "lint", Needs: []string{"build"}, Steps: []*Step{{Name: "Run lint", Uses: "lint"}}},
				},
			},
		},
		{
			name: "divergence_three",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build", ID: "build", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Unit Test", ID: "unit-test", Needs: []string{"build"}, Steps: []*Step{{Name: "Run unit", Uses: "go"}}},
					{Name: "Integration", ID: "integration", Needs: []string{"build"}, Steps: []*Step{{Name: "Run integ", Uses: "go"}}},
					{Name: "Lint", ID: "lint", Needs: []string{"build"}, Steps: []*Step{{Name: "Run lint", Uses: "lint"}}},
				},
			},
		},
		{
			name: "divergence_uneven_steps",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build", ID: "build", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Unit Test", ID: "unit-test", Needs: []string{"build"}, Steps: []*Step{
						{Name: "Setup", Uses: "go"},
						{Name: "Run unit", Uses: "go"},
						{Name: "Teardown", Uses: "go"},
					}},
					{Name: "Lint", ID: "lint", Needs: []string{"build"}, Steps: []*Step{{Name: "Run lint", Uses: "lint"}}},
				},
			},
		},
		{
			name: "convergence_two",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build Linux", ID: "build-linux", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Build Mac", ID: "build-mac", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Release", ID: "release", Needs: []string{"build-linux", "build-mac"}, Steps: []*Step{{Name: "Upload", Uses: "release"}}},
				},
			},
		},
		{
			name: "convergence_three",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build Linux", ID: "build-linux", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Build Mac", ID: "build-mac", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Build Win", ID: "build-win", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Release", ID: "release", Needs: []string{"build-linux", "build-mac", "build-win"}, Steps: []*Step{{Name: "Upload", Uses: "release"}}},
				},
			},
		},
		{
			name: "convergence_uneven_steps",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build Linux", ID: "build-linux", Steps: []*Step{
						{Name: "Setup env", Uses: "setup"},
						{Name: "Compile", Uses: "go"},
						{Name: "Package", Uses: "tar"},
					}},
					{Name: "Build Mac", ID: "build-mac", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Release", ID: "release", Needs: []string{"build-linux", "build-mac"}, Steps: []*Step{{Name: "Upload", Uses: "release"}}},
				},
			},
		},
		{
			name: "multi_child_convergence",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build", ID: "build", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Lint", ID: "lint", Steps: []*Step{{Name: "Run lint", Uses: "lint"}}},
					{Name: "Unit Test", ID: "unit-test", Needs: []string{"build", "lint"}, Steps: []*Step{{Name: "Run unit", Uses: "go"}}},
					{Name: "Deploy", ID: "deploy", Needs: []string{"build"}, Steps: []*Step{{Name: "Deploy app", Uses: "deploy"}}},
				},
			},
		},
		{
			name: "diamond",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build", ID: "build", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Unit Test", ID: "unit-test", Needs: []string{"build"}, Steps: []*Step{{Name: "Run unit", Uses: "go"}}},
					{Name: "Lint", ID: "lint", Needs: []string{"build"}, Steps: []*Step{{Name: "Run lint", Uses: "lint"}}},
					{Name: "Deploy", ID: "deploy", Needs: []string{"unit-test", "lint"}, Steps: []*Step{{Name: "Deploy app", Uses: "deploy"}}},
				},
			},
		},
		{
			name: "hourglass",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build Linux", ID: "build-linux", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Build Mac", ID: "build-mac", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Integration", ID: "integration", Needs: []string{"build-linux", "build-mac"}, Steps: []*Step{{Name: "Test all", Uses: "go"}}},
					{Name: "Deploy Prod", ID: "deploy-prod", Needs: []string{"integration"}, Steps: []*Step{{Name: "Deploy", Uses: "deploy"}}},
					{Name: "Deploy Stage", ID: "deploy-stage", Needs: []string{"integration"}, Steps: []*Step{{Name: "Deploy", Uses: "deploy"}}},
				},
			},
		},
		{
			name: "parallel_roots",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build App", ID: "build-app", Steps: []*Step{{Name: "Compile app", Uses: "go"}}},
					{Name: "Build Lib", ID: "build-lib", Steps: []*Step{{Name: "Compile lib", Uses: "go"}}},
				},
			},
		},
		{
			name: "parallel_chains",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build App", ID: "build-app", Steps: []*Step{{Name: "Compile app", Uses: "go"}}},
					{Name: "Test App", ID: "test-app", Needs: []string{"build-app"}, Steps: []*Step{{Name: "Test app", Uses: "go"}}},
					{Name: "Build Lib", ID: "build-lib", Steps: []*Step{{Name: "Compile lib", Uses: "go"}}},
					{Name: "Test Lib", ID: "test-lib", Needs: []string{"build-lib"}, Steps: []*Step{{Name: "Test lib", Uses: "go"}}},
				},
			},
		},
		{
			name: "mixed_standalone",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build", ID: "build", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Test", ID: "test", Needs: []string{"build"}, Steps: []*Step{{Name: "Run tests", Uses: "go"}}},
					{Name: "Docs", ID: "docs", Steps: []*Step{{Name: "Build docs", Uses: "docs"}}},
				},
			},
		},
		{
			// Jobs with children should be sorted before jobs without children at the same level.
			// Definition order: [Docs(no children), Notify(no children), Build(has children), Lint(no children)]
			// Expected display order: [Build, Docs, Notify, Lint] (Build first because it has children)
			name: "sorted_children_first",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Docs", ID: "docs", Steps: []*Step{{Name: "Build docs", Uses: "docs"}}},
					{Name: "Notify", ID: "notify", Steps: []*Step{{Name: "Send notification", Uses: "slack"}}},
					{Name: "Build", ID: "build", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Lint", ID: "lint", Steps: []*Step{{Name: "Run lint", Uses: "lint"}}},
					{Name: "Test", ID: "test", Needs: []string{"build"}, Steps: []*Step{{Name: "Run tests", Uses: "go"}}},
				},
			},
		},
		{
			name: "wide_divergence",
			workflow: &Workflow{
				Jobs: []Job{
					{Name: "Build", ID: "build", Steps: []*Step{{Name: "Compile", Uses: "go"}}},
					{Name: "Unit Test", ID: "unit-test", Needs: []string{"build"}, Steps: []*Step{{Name: "Run unit", Uses: "go"}}},
					{Name: "Integration", ID: "integration", Needs: []string{"build"}, Steps: []*Step{{Name: "Run integ", Uses: "go"}}},
					{Name: "E2E Test", ID: "e2e", Needs: []string{"build"}, Steps: []*Step{{Name: "Run e2e", Uses: "e2e"}}},
					{Name: "Lint", ID: "lint", Needs: []string{"build"}, Steps: []*Step{{Name: "Run lint", Uses: "lint"}}},
				},
			},
		},
	}
}

// TestDagAsciiRenderer_Golden performs golden file testing for DAG ASCII rendering.
// It compares the actual output against expected output stored in testdata/dag_ascii/*.golden.txt.
//
// To update golden files when the output format intentionally changes:
//
//	UPDATE_GOLDEN=1 go test -run TestDagAsciiRenderer_Golden
func TestDagAsciiRenderer_Golden(t *testing.T) {
	testCases := getDagAsciiGoldenTestCases()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			renderer := NewDagAsciiRenderer(tc.workflow)
			actual := renderer.Render()

			goldenPath := filepath.Join("testdata", "dag_ascii", tc.name+".golden.txt")

			if os.Getenv("UPDATE_GOLDEN") == "1" {
				err := os.WriteFile(goldenPath, []byte(actual), 0644)
				if err != nil {
					t.Fatalf("failed to write golden file: %v", err)
				}
				return
			}

			expected, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("failed to read golden file %s: %v\nRun with UPDATE_GOLDEN=1 to create it", goldenPath, err)
			}

			if actual != string(expected) {
				t.Errorf("output does not match golden file %s\n\nExpected:\n%s\n\nActual:\n%s\n\nRun with UPDATE_GOLDEN=1 to update", goldenPath, string(expected), actual)
			}
		})
	}
}

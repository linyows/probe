/*
Package probe provides DAG Mermaid rendering tests.

# Golden Test Cases

The golden tests cover comprehensive DAG patterns for Mermaid rendering:

	Category              | Case Name                | Structure
	----------------------|--------------------------|------------------------------------------
	Basic                 | single                   | Single job
	                      | linear_two               | A → B
	                      | linear_three             | A → B → C
	Divergence            | divergence_two           | A → [B, C]
	                      | divergence_three         | A → [B, C, D]
	Convergence           | convergence_two          | [A, B] → C
	                      | convergence_three        | [A, B, C] → D
	Complex               | diamond                  | A → [B, C] → D
	                      | hourglass                | [A, B] → C → [D, E]
	Parallel              | parallel_roots           | [A], [B] (independent)
	                      | parallel_chains          | [A→B], [C→D] (independent chains)
	Mixed                 | mixed_standalone         | A → B, C (standalone)

# Usage

Run golden tests:

	go test -run TestDagMermaidRenderer_Golden -v

Update golden files when output format changes:

	UPDATE_GOLDEN=1 go test -run TestDagMermaidRenderer_Golden

Golden files are stored in testdata/dag_mermaid/*.golden.txt
*/
package probe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDagMermaidRenderer(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{
			{Name: "Job 1", ID: "job1"},
			{Name: "Job 2", ID: "job2", Needs: []string{"job1"}},
		},
	}

	renderer := NewDagMermaidRenderer(w)

	if renderer.workflow != w {
		t.Error("workflow should be set")
	}

	if renderer.jobIDToIdx["job1"] != 0 {
		t.Errorf("expected job1 at index 0, got %d", renderer.jobIDToIdx["job1"])
	}

	if renderer.jobIDToIdx["job2"] != 1 {
		t.Errorf("expected job2 at index 1, got %d", renderer.jobIDToIdx["job2"])
	}
}

func TestDagMermaidRenderer_Render_EmptyWorkflow(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{},
	}

	renderer := NewDagMermaidRenderer(w)
	result := renderer.Render()

	if result != "" {
		t.Errorf("expected empty string for empty workflow, got %q", result)
	}
}

func TestDagMermaidRenderer_Render_SingleJob(t *testing.T) {
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

	renderer := NewDagMermaidRenderer(w)
	result := renderer.Render()

	// Check that output starts with flowchart header
	if !strings.HasPrefix(result, "flowchart LR") {
		t.Errorf("expected output to start with 'flowchart LR', got:\n%s", result)
	}

	// Check that the output contains the job name
	if !strings.Contains(result, "Single Job") {
		t.Errorf("expected output to contain 'Single Job', got:\n%s", result)
	}

	// Check that the output contains the step name
	if !strings.Contains(result, "Step 1") {
		t.Errorf("expected output to contain 'Step 1', got:\n%s", result)
	}
}

func TestDagMermaidRenderer_Render_TwoJobsWithDependency(t *testing.T) {
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

	renderer := NewDagMermaidRenderer(w)
	result := renderer.Render()

	// Check that the output contains both job names
	if !strings.Contains(result, "First Job") {
		t.Errorf("expected output to contain 'First Job', got:\n%s", result)
	}
	if !strings.Contains(result, "Second Job") {
		t.Errorf("expected output to contain 'Second Job', got:\n%s", result)
	}

	// Check for connection arrow (Mermaid uses -->)
	if !strings.Contains(result, "-->") {
		t.Errorf("expected output to contain '-->', got:\n%s", result)
	}
}

func TestDagMermaidRenderer_Render_ParallelJobs(t *testing.T) {
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

	renderer := NewDagMermaidRenderer(w)
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

	// Check for edges
	if !strings.Contains(result, "root --> branch_a") {
		t.Errorf("expected output to contain 'root --> branch_a', got:\n%s", result)
	}
	if !strings.Contains(result, "root --> branch_b") {
		t.Errorf("expected output to contain 'root --> branch_b', got:\n%s", result)
	}
}

func TestDagMermaidRenderer_SanitizeID(t *testing.T) {
	renderer := &DagMermaidRenderer{}

	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with-dash", "with_dash"},
		{"with space", "with_space"},
		{"123numeric", "n123numeric"},
		{"special@#$chars", "special___chars"},
		{"", "node"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := renderer.sanitizeID(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDagMermaidRenderer_EscapeLabel(t *testing.T) {
	renderer := &DagMermaidRenderer{}

	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{`with "quotes"`, `with #quot;quotes#quot;`},
		{"no special", "no special"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := renderer.escapeLabel(tt.input)
			if result != tt.expected {
				t.Errorf("escapeLabel(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRenderDagMermaid(t *testing.T) {
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

	result := w.RenderDagMermaid()

	if result == "" {
		t.Error("expected non-empty result")
	}

	if !strings.Contains(result, "Test Job") {
		t.Errorf("expected output to contain 'Test Job', got:\n%s", result)
	}
}

func TestRenderDagMermaid_Empty(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{},
	}

	result := w.RenderDagMermaid()

	if result != "" {
		t.Errorf("expected empty result for empty workflow, got:\n%s", result)
	}
}

func TestDagMermaidRenderer_EmbeddedAction(t *testing.T) {
	w := &Workflow{
		Jobs: []Job{
			{
				Name: "Test",
				ID:   "test",
				Steps: []*Step{
					{Name: "Normal Step", Uses: "hello"},
					{Name: "Embedded Step", Uses: "embedded"},
				},
			},
		},
	}

	renderer := NewDagMermaidRenderer(w)
	result := renderer.Render()

	// Check that output contains both steps
	if !strings.Contains(result, "Normal Step") {
		t.Errorf("expected output to contain 'Normal Step', got:\n%s", result)
	}
	if !strings.Contains(result, "Embedded Step") {
		t.Errorf("expected output to contain 'Embedded Step', got:\n%s", result)
	}
}

func TestDagMermaidRenderer_EmbeddedExpanded(t *testing.T) {
	w := &Workflow{
		basePath: "testdata",
		Jobs: []Job{
			{
				Name: "Deploy",
				ID:   "deploy",
				Steps: []*Step{
					{Name: "Run job", Uses: "embedded", With: map[string]any{"path": "./embedded-success-job.yml"}},
				},
			},
		},
	}

	renderer := NewDagMermaidRenderer(w)
	result := renderer.Render()

	// Check that output contains the parent step
	if !strings.Contains(result, "Run job") {
		t.Errorf("expected output to contain 'Run job', got:\n%s", result)
	}

	// Check that output contains expanded embedded steps
	if !strings.Contains(result, "Simple success step") {
		t.Errorf("expected output to contain 'Simple success step' from embedded job, got:\n%s", result)
	}
	if !strings.Contains(result, "Another success step") {
		t.Errorf("expected output to contain 'Another success step' from embedded job, got:\n%s", result)
	}
}

// dagMermaidGoldenTestCase defines a golden test case for DAG Mermaid rendering.
type dagMermaidGoldenTestCase struct {
	name     string    // Test case name, used as golden file name
	workflow *Workflow // Workflow to render
}

// getDagMermaidGoldenTestCases returns all golden test cases covering various DAG patterns.
func getDagMermaidGoldenTestCases() []dagMermaidGoldenTestCase {
	return []dagMermaidGoldenTestCase{
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
			// Embedded action with path expands internal steps
			name: "embedded_expanded",
			workflow: &Workflow{
				basePath: "testdata",
				Jobs: []Job{
					{Name: "Deploy", ID: "deploy", Steps: []*Step{
						{Name: "Run job", Uses: "embedded", With: map[string]any{"path": "./embedded-success-job.yml"}},
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
	}
}

// TestDagMermaidRenderer_Golden performs golden file testing for DAG Mermaid rendering.
// It compares the actual output against expected output stored in testdata/dag_mermaid/*.golden.txt.
//
// To update golden files when the output format intentionally changes:
//
//	UPDATE_GOLDEN=1 go test -run TestDagMermaidRenderer_Golden
func TestDagMermaidRenderer_Golden(t *testing.T) {
	testCases := getDagMermaidGoldenTestCases()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			renderer := NewDagMermaidRenderer(tc.workflow)
			actual := renderer.Render()

			goldenPath := filepath.Join("testdata", "dag_mermaid", tc.name+".golden.txt")

			if os.Getenv("UPDATE_GOLDEN") == "1" {
				// Ensure directory exists
				if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
					t.Fatalf("failed to create golden directory: %v", err)
				}
				err := os.WriteFile(goldenPath, []byte(actual), 0o600)
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

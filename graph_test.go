package probe

import (
	"strings"
	"testing"
)

func TestTruncateLabel(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		maxLen   int
		expected string
	}{
		{
			name:     "short label unchanged",
			label:    "Build",
			maxLen:   15,
			expected: "Build",
		},
		{
			name:     "exact length unchanged",
			label:    "123456789012345",
			maxLen:   15,
			expected: "123456789012345",
		},
		{
			name:     "long label truncated",
			label:    "Build by docker container",
			maxLen:   15,
			expected: "Build by docke~",
		},
		{
			name:     "very short maxLen",
			label:    "Hello",
			maxLen:   1,
			expected: "H",
		},
		{
			name:     "empty label",
			label:    "",
			maxLen:   15,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateLabel(tt.label, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateLabel(%q, %d) = %q, want %q",
					tt.label, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestWorkflow_RenderDependencyGraph(t *testing.T) {
	tests := []struct {
		name     string
		workflow Workflow
		contains []string
	}{
		{
			name:     "empty workflow",
			workflow: Workflow{},
			contains: nil,
		},
		{
			name: "single job",
			workflow: Workflow{
				Jobs: []Job{
					{Name: "Build", ID: "build"},
				},
			},
			contains: []string{"[Build]"},
		},
		{
			name: "simple chain",
			workflow: Workflow{
				Jobs: []Job{
					{Name: "Setup", ID: "setup"},
					{Name: "Build", ID: "build", Needs: []string{"setup"}},
					{Name: "Test", ID: "test", Needs: []string{"build"}},
				},
			},
			contains: []string{"[Setup]", "[Build]", "[Test]"},
		},
		{
			name: "parallel jobs",
			workflow: Workflow{
				Jobs: []Job{
					{Name: "Setup", ID: "setup"},
					{Name: "Test A", ID: "test-a", Needs: []string{"setup"}},
					{Name: "Test B", ID: "test-b", Needs: []string{"setup"}},
				},
			},
			contains: []string{"[Setup]", "[Test A]", "[Test B]"},
		},
		{
			name: "long job name truncated",
			workflow: Workflow{
				Jobs: []Job{
					{Name: "Build container with docker compose", ID: "build"},
				},
			},
			contains: []string{"Build container w..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.workflow.RenderDependencyGraph()
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("RenderDependencyGraph() = %q, want to contain %q",
						result, want)
				}
			}
		})
	}
}

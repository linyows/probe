package dag

import (
	"fmt"
	"regexp"
	"strings"
)

// Mermaid draws a Graph as a Mermaid flowchart: a subgraph of steps for each
// job, and an edge from each job to the jobs that need it.
type Mermaid struct{}

// Render draws g, or returns an empty string when it has no jobs.
func (m Mermaid) Render(g Graph) string {
	if len(g.Jobs) == 0 {
		return ""
	}

	var sb strings.Builder

	// Mermaid flowchart header (left-right direction)
	sb.WriteString("flowchart LR\n")

	// Render node definitions with subgraphs for steps
	for _, job := range g.Jobs {
		safeID := m.sanitizeID(job.ID)
		displayName := m.escapeLabel(job.Name)

		// Create subgraph for job with steps
		if len(job.Steps) > 0 {
			fmt.Fprintf(&sb, "    subgraph %s[\"%s\"]\n", safeID, displayName)
			stepIndex := 0
			for _, step := range job.Steps {
				stepID := fmt.Sprintf("%s_step%d", safeID, stepIndex)
				stepLabel := m.escapeLabel(step.Name)
				fmt.Fprintf(&sb, "        %s[\"%s\"]\n", stepID, stepLabel)
				stepIndex++

				// Render the steps of an embedded job after the step that runs it
				for _, embStepName := range step.EmbeddedSteps {
					embStepID := fmt.Sprintf("%s_step%d", safeID, stepIndex)
					embStepLabel := m.escapeLabel(embStepName)
					fmt.Fprintf(&sb, "        %s[\"%s\"]\n", embStepID, embStepLabel)
					stepIndex++
				}
			}
			sb.WriteString("    end\n")
		} else {
			// Job without steps - simple node
			fmt.Fprintf(&sb, "    %s[\"%s\"]\n", safeID, displayName)
		}
	}

	sb.WriteString("\n")

	// Render edges (dependencies)
	for _, job := range g.Jobs {
		if len(job.Needs) == 0 {
			continue
		}

		safeJobID := m.sanitizeID(job.ID)

		for _, need := range job.Needs {
			safeNeedID := m.sanitizeID(need)
			fmt.Fprintf(&sb, "    %s --> %s\n", safeNeedID, safeJobID)
		}
	}

	return sb.String()
}

// sanitizeID converts a job ID/name to a valid Mermaid node ID
// Mermaid IDs should be alphanumeric with underscores
//
// NOTE: Original job/step IDs are validated for uniqueness when a workflow is loaded.
// However, sanitized IDs may collide (e.g., "unit-test" and "unit.test" both become "unit_test").
// This is acceptable as such naming conflicts are rare in practice.
func (Mermaid) sanitizeID(id string) string {
	// Replace non-alphanumeric characters with underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	sanitized := reg.ReplaceAllString(id, "_")

	// Ensure it starts with a letter (prepend 'n' if it starts with a number)
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "n" + sanitized
	}

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "node"
	}

	return sanitized
}

// escapeLabel escapes special characters for Mermaid labels
func (Mermaid) escapeLabel(label string) string {
	// Escape double quotes
	label = strings.ReplaceAll(label, "\"", "#quot;")
	return label
}

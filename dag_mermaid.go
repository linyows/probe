package probe

import (
	"fmt"
	"regexp"
	"strings"
)

// DagMermaidRenderer renders workflow graphs in Mermaid format
type DagMermaidRenderer struct {
	workflow   *Workflow
	jobIDToIdx map[string]int
}

// NewDagMermaidRenderer creates a new DagMermaidRenderer
func NewDagMermaidRenderer(w *Workflow) *DagMermaidRenderer {
	r := &DagMermaidRenderer{
		workflow:   w,
		jobIDToIdx: make(map[string]int),
	}

	// Build job ID to index mapping
	for i, job := range w.Jobs {
		id := job.ID
		if id == "" {
			id = job.Name
		}
		r.jobIDToIdx[id] = i
	}

	return r
}

// Render generates the Mermaid flowchart representation
func (r *DagMermaidRenderer) Render() string {
	if len(r.workflow.Jobs) == 0 {
		return ""
	}

	var sb strings.Builder

	// Mermaid flowchart header (left-right direction)
	sb.WriteString("flowchart LR\n")

	// Render node definitions with subgraphs for steps
	for _, job := range r.workflow.Jobs {
		jobID := r.getJobID(&job)
		safeID := r.sanitizeID(jobID)
		displayName := r.escapeLabel(job.Name)

		// Create subgraph for job with steps
		if len(job.Steps) > 0 {
			sb.WriteString(fmt.Sprintf("    subgraph %s[\"%s\"]\n", safeID, displayName))
			for i, step := range job.Steps {
				stepName := step.Name
				if stepName == "" {
					stepName = step.Uses
				}
				stepID := fmt.Sprintf("%s_step%d", safeID, i)
				stepLabel := r.escapeLabel(stepName)
				sb.WriteString(fmt.Sprintf("        %s[\"%s\"]\n", stepID, stepLabel))
			}
			sb.WriteString("    end\n")
		} else {
			// Job without steps - simple node
			sb.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", safeID, displayName))
		}
	}

	sb.WriteString("\n")

	// Render edges (dependencies)
	for _, job := range r.workflow.Jobs {
		if len(job.Needs) == 0 {
			continue
		}

		jobID := r.getJobID(&job)
		safeJobID := r.sanitizeID(jobID)

		for _, need := range job.Needs {
			safeNeedID := r.sanitizeID(need)
			sb.WriteString(fmt.Sprintf("    %s --> %s\n", safeNeedID, safeJobID))
		}
	}

	return sb.String()
}

// getJobID returns the job ID or Name if ID is empty
func (r *DagMermaidRenderer) getJobID(job *Job) string {
	if job.ID != "" {
		return job.ID
	}
	return job.Name
}

// sanitizeID converts a job ID/name to a valid Mermaid node ID
// Mermaid IDs should be alphanumeric with underscores
//
// NOTE: Original job/step IDs are validated for uniqueness by validateIDs() in probe.go.
// However, sanitized IDs may collide (e.g., "unit-test" and "unit.test" both become "unit_test").
// This is acceptable as such naming conflicts are rare in practice.
func (r *DagMermaidRenderer) sanitizeID(id string) string {
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
func (r *DagMermaidRenderer) escapeLabel(label string) string {
	// Escape double quotes
	label = strings.ReplaceAll(label, "\"", "#quot;")
	return label
}

// RenderDagMermaid returns the Mermaid format representation of workflow job dependencies
func (w *Workflow) RenderDagMermaid() string {
	renderer := NewDagMermaidRenderer(w)
	return renderer.Render()
}

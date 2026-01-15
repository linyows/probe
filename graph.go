package probe

import (
	asciidag "github.com/linyows/probe/ascii-dag"
)

const (
	// maxLabelLength is the maximum length of a job label in the graph
	maxLabelLength = 15
	truncateChar   = "~"
)

// RenderDependencyGraph renders the workflow job dependencies as ASCII art
func (w *Workflow) RenderDependencyGraph() string {
	if len(w.Jobs) == 0 {
		return ""
	}

	// Build job ID to index mapping
	jobIDToIndex := make(map[string]int)
	for i, job := range w.Jobs {
		id := job.ID
		if id == "" {
			id = job.Name
		}
		jobIDToIndex[id] = i
	}

	// Create nodes
	nodes := make([]asciidag.Node, len(w.Jobs))
	for i, job := range w.Jobs {
		label := job.Name
		if label == "" {
			label = job.ID
		}
		nodes[i] = asciidag.Node{
			ID:    i,
			Label: truncateLabel(label, maxLabelLength),
		}
	}

	// Create edges from dependencies
	var edges []asciidag.Edge
	for i, job := range w.Jobs {
		for _, need := range job.Needs {
			if fromIdx, ok := jobIDToIndex[need]; ok {
				edges = append(edges, asciidag.Edge{
					From: fromIdx,
					To:   i,
				})
			}
		}
	}

	// Build DAG and render
	dag := asciidag.FromEdges(nodes, edges)
	return dag.Render()
}

// truncateLabel truncates a label to maxLen characters with "-" suffix
func truncateLabel(label string, maxLen int) string {
	if len(label) <= maxLen {
		return label
	}
	if maxLen <= 1 {
		return label[:maxLen]
	}
	return label[:maxLen-1] + truncateChar
}

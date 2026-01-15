package asciidag

import (
	"github.com/linyows/probe/dag"
)

// HasCycle returns true if the DAG contains a cycle.
func (d *DAG) HasCycle() bool {
	if len(d.nodes) == 0 {
		return false
	}
	return dag.HasCycleFn(d.allNodeIDs(), d.getChildIDs)
}

// FindCyclePath finds and returns a cycle path if one exists.
// Returns nil if no cycle is found.
func (d *DAG) FindCyclePath() []int {
	if len(d.nodes) == 0 {
		return nil
	}
	return dag.DetectCycleFn(d.allNodeIDs(), d.getChildIDs)
}

// allNodeIDs returns all node IDs in the DAG.
func (d *DAG) allNodeIDs() []int {
	ids := make([]int, len(d.nodes))
	for i, n := range d.nodes {
		ids[i] = n.id
	}
	return ids
}

// getChildIDs returns the IDs of all children of a node.
func (d *DAG) getChildIDs(id int) []int {
	idx := d.nodeIndex(id)
	if idx < 0 || idx >= len(d.children) {
		return nil
	}
	childIndices := d.children[idx]
	childIDs := make([]int, len(childIndices))
	for i, childIdx := range childIndices {
		childIDs[i] = d.nodes[childIdx].id
	}
	return childIDs
}

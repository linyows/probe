// Package asciidag provides a zero-dependency ASCII DAG rendering engine.
package asciidag

import (
	"fmt"
	"strings"
)

// Node represents a node with an ID and label.
type Node struct {
	ID    int
	Label string
}

// Edge represents a directed edge from one node to another.
type Edge struct {
	From int
	To   int
}

// node is the internal representation of a node.
type node struct {
	id    int
	label string
}

// edge is the internal representation of an edge.
type edge struct {
	from int
	to   int
}

// DAG represents a Directed Acyclic Graph.
type DAG struct {
	nodes                   []node
	edges                   []edge
	renderMode              RenderMode
	autoCreated             map[int]bool // Tracks auto-created nodes
	idToIndex               map[int]int  // O(1) ID to index lookup
	nodeWidths              []int        // Cached formatted widths
	children                [][]int      // Adjacency list: children[idx] = child indices
	parents                 [][]int      // Adjacency list: parents[idx] = parent indices
	crossingReductionPasses int
}

// New creates a new empty DAG with optional configuration.
func New(opts ...Option) *DAG {
	d := &DAG{
		nodes:                   make([]node, 0),
		edges:                   make([]edge, 0),
		renderMode:              RenderModeAuto,
		autoCreated:             make(map[int]bool),
		idToIndex:               make(map[int]int),
		nodeWidths:              make([]int, 0),
		children:                make([][]int, 0),
		parents:                 make([][]int, 0),
		crossingReductionPasses: 4,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// FromEdges creates a DAG from a list of nodes and edges.
// This is more efficient than adding nodes and edges one by one.
func FromEdges(nodes []Node, edges []Edge, opts ...Option) *DAG {
	d := New(opts...)

	// Pre-allocate
	d.nodes = make([]node, 0, len(nodes))
	d.nodeWidths = make([]int, 0, len(nodes))
	d.children = make([][]int, 0, len(nodes))
	d.parents = make([][]int, 0, len(nodes))

	// Add nodes
	for _, n := range nodes {
		d.addNodeInternal(n.ID, n.Label)
	}

	// Add edges
	for _, e := range edges {
		d.AddEdge(e.From, e.To)
	}

	return d
}

// AddNode adds a node with the given ID and label.
// If a node with the same ID already exists, it updates the label.
func (d *DAG) AddNode(id int, label string) {
	if idx, exists := d.idToIndex[id]; exists {
		// Update existing node
		d.nodes[idx].label = label
		d.nodeWidths[idx] = d.computeNodeWidth(id, label)
		delete(d.autoCreated, id)
		return
	}
	d.addNodeInternal(id, label)
}

// addNodeInternal adds a node without checking for duplicates.
func (d *DAG) addNodeInternal(id int, label string) {
	idx := len(d.nodes)
	d.nodes = append(d.nodes, node{id: id, label: label})
	d.idToIndex[id] = idx
	d.nodeWidths = append(d.nodeWidths, d.computeNodeWidth(id, label))
	d.children = append(d.children, make([]int, 0))
	d.parents = append(d.parents, make([]int, 0))
}

// AddEdge adds a directed edge from one node to another.
// If either node doesn't exist, it will be auto-created.
func (d *DAG) AddEdge(from, to int) {
	d.ensureNodeExists(from)
	d.ensureNodeExists(to)

	fromIdx := d.idToIndex[from]
	toIdx := d.idToIndex[to]

	// Check if edge already exists
	for _, childIdx := range d.children[fromIdx] {
		if childIdx == toIdx {
			return // Edge already exists
		}
	}

	d.edges = append(d.edges, edge{from: from, to: to})
	d.children[fromIdx] = append(d.children[fromIdx], toIdx)
	d.parents[toIdx] = append(d.parents[toIdx], fromIdx)
}

// ensureNodeExists creates a node if it doesn't exist.
func (d *DAG) ensureNodeExists(id int) {
	if _, exists := d.idToIndex[id]; !exists {
		d.addNodeInternal(id, "")
		d.autoCreated[id] = true
	}
}

// computeNodeWidth calculates the display width of a node.
func (d *DAG) computeNodeWidth(id int, label string) int {
	if label == "" {
		// Auto-created node: display as ⟨ID⟩
		return len(fmt.Sprintf("⟨%d⟩", id))
	}
	// Normal node: display as [Label]
	return len(label) + 2 // "[" + label + "]"
}

// isAutoCreated returns true if the node was auto-created.
func (d *DAG) isAutoCreated(id int) bool {
	return d.autoCreated[id]
}

// nodeIndex returns the index of a node by ID, or -1 if not found.
func (d *DAG) nodeIndex(id int) int {
	if idx, exists := d.idToIndex[id]; exists {
		return idx
	}
	return -1
}

// getChildrenIndices returns the indices of child nodes.
func (d *DAG) getChildrenIndices(nodeIdx int) []int {
	if nodeIdx < 0 || nodeIdx >= len(d.children) {
		return nil
	}
	return d.children[nodeIdx]
}

// getParentsIndices returns the indices of parent nodes.
func (d *DAG) getParentsIndices(nodeIdx int) []int {
	if nodeIdx < 0 || nodeIdx >= len(d.parents) {
		return nil
	}
	return d.parents[nodeIdx]
}

// childrenCount returns the number of children for a node.
func (d *DAG) childrenCount(nodeIdx int) int {
	if nodeIdx < 0 || nodeIdx >= len(d.children) {
		return 0
	}
	return len(d.children[nodeIdx])
}

// parentsCount returns the number of parents for a node.
func (d *DAG) parentsCount(nodeIdx int) int {
	if nodeIdx < 0 || nodeIdx >= len(d.parents) {
		return 0
	}
	return len(d.parents[nodeIdx])
}

// NodeCount returns the number of nodes in the DAG.
func (d *DAG) NodeCount() int {
	return len(d.nodes)
}

// EdgeCount returns the number of edges in the DAG.
func (d *DAG) EdgeCount() int {
	return len(d.edges)
}

// isSimpleChain returns true if the DAG is a simple chain (no branching).
func (d *DAG) isSimpleChain() bool {
	for i := range d.nodes {
		if d.parentsCount(i) > 1 || d.childrenCount(i) > 1 {
			return false
		}
	}
	return true
}

// formatNode returns the formatted display string for a node.
func (d *DAG) formatNode(idx int) string {
	n := d.nodes[idx]
	if d.isAutoCreated(n.id) {
		return fmt.Sprintf("⟨%d⟩", n.id)
	}
	return fmt.Sprintf("[%s]", n.label)
}

// SetRenderMode sets the rendering mode.
func (d *DAG) SetRenderMode(mode RenderMode) {
	d.renderMode = mode
}

// SetCrossingReductionPasses sets the number of crossing reduction passes.
func (d *DAG) SetCrossingReductionPasses(passes int) {
	if passes < 0 {
		passes = 0
	}
	if passes > 1000 {
		passes = 0
	}
	d.crossingReductionPasses = passes
}

// Render returns the ASCII art representation of the DAG.
func (d *DAG) Render() string {
	var sb strings.Builder
	d.RenderTo(&sb)
	return sb.String()
}

// RenderTo writes the ASCII art representation to the given builder.
func (d *DAG) RenderTo(sb *strings.Builder) {
	if len(d.nodes) == 0 {
		return
	}

	// Check for cycles
	if d.HasCycle() {
		d.renderCycle(sb)
		return
	}

	// Determine render mode
	mode := d.renderMode
	if mode == RenderModeAuto {
		if d.isSimpleChain() {
			mode = RenderModeHorizontal
		} else {
			mode = RenderModeVertical
		}
	}

	switch mode {
	case RenderModeHorizontal:
		d.renderHorizontal(sb)
	default:
		d.renderVertical(sb)
	}
}

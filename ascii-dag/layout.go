package asciidag

import (
	"sort"
)

// levelInfo holds the index and level of a node.
type levelInfo struct {
	index int
	level int
}

// calculateLevels assigns hierarchical levels to all nodes.
// Uses fixed-point iteration: node's level = max(parent levels) + 1.
func (d *DAG) calculateLevels() []levelInfo {
	n := len(d.nodes)
	levels := make([]int, n)

	// Fixed-point iteration
	changed := true
	for changed {
		changed = false
		for _, e := range d.edges {
			fromIdx := d.nodeIndex(e.from)
			toIdx := d.nodeIndex(e.to)
			if fromIdx < 0 || toIdx < 0 {
				continue
			}

			newLevel := levels[fromIdx] + 1
			if newLevel > levels[toIdx] {
				levels[toIdx] = newLevel
				changed = true
			}
		}
	}

	result := make([]levelInfo, n)
	for i := 0; i < n; i++ {
		result[i] = levelInfo{index: i, level: levels[i]}
	}
	return result
}

// calculateLevelsForSubgraph calculates levels for a specific subgraph.
func (d *DAG) calculateLevelsForSubgraph(subgraphIndices []int) []levelInfo {
	n := len(d.nodes)
	levels := make([]int, n)

	// Build set of subgraph node IDs for quick lookup
	subgraphNodeIDs := make(map[int]bool)
	for _, idx := range subgraphIndices {
		subgraphNodeIDs[d.nodes[idx].id] = true
	}

	// Fixed-point iteration
	changed := true
	for changed {
		changed = false
		for _, e := range d.edges {
			// Only process edges within this subgraph
			if !subgraphNodeIDs[e.from] || !subgraphNodeIDs[e.to] {
				continue
			}

			fromIdx := d.nodeIndex(e.from)
			toIdx := d.nodeIndex(e.to)
			if fromIdx < 0 || toIdx < 0 {
				continue
			}

			newLevel := levels[fromIdx] + 1
			if newLevel > levels[toIdx] {
				levels[toIdx] = newLevel
				changed = true
			}
		}
	}

	result := make([]levelInfo, len(subgraphIndices))
	for i, idx := range subgraphIndices {
		result[i] = levelInfo{index: idx, level: levels[idx]}
	}
	return result
}

// reduceCrossings reduces edge crossings using median heuristic (Sugiyama).
func (d *DAG) reduceCrossings(levels [][]int, maxLevel int) {
	for pass := 0; pass < d.crossingReductionPasses; pass++ {
		// Top-down pass: order nodes by median of parents
		for levelIdx := 1; levelIdx <= maxLevel; levelIdx++ {
			parentLevel := levels[levelIdx-1]
			d.orderByMedianParents(levels[levelIdx], parentLevel)
		}

		// Bottom-up pass: order nodes by median of children
		for levelIdx := maxLevel - 1; levelIdx >= 0; levelIdx-- {
			childLevel := levels[levelIdx+1]
			d.orderByMedianChildren(levels[levelIdx], childLevel)
		}
	}
}

// orderByMedianParents orders nodes by median position of their parents.
func (d *DAG) orderByMedianParents(levelNodes []int, parentLevel []int) {
	type nodeMedian struct {
		idx    int
		median float64
	}

	medians := make([]nodeMedian, len(levelNodes))
	for pos, idx := range levelNodes {
		parentIndices := d.getParentsIndices(idx)

		if len(parentIndices) == 0 {
			medians[pos] = nodeMedian{idx: idx, median: float64(pos)}
			continue
		}

		// Find positions of parents in the parent level
		parentPositions := make([]int, 0, len(parentIndices))
		for _, pIdx := range parentIndices {
			for i, lvlIdx := range parentLevel {
				if lvlIdx == pIdx {
					parentPositions = append(parentPositions, i)
					break
				}
			}
		}

		if len(parentPositions) == 0 {
			medians[pos] = nodeMedian{idx: idx, median: float64(pos)}
			continue
		}

		sort.Ints(parentPositions)
		var median float64
		if len(parentPositions)%2 == 1 {
			median = float64(parentPositions[len(parentPositions)/2])
		} else {
			mid := len(parentPositions) / 2
			median = float64(parentPositions[mid-1]+parentPositions[mid]) / 2.0
		}
		medians[pos] = nodeMedian{idx: idx, median: median}
	}

	// Sort by median
	sort.Slice(medians, func(i, j int) bool {
		return medians[i].median < medians[j].median
	})

	for i, nm := range medians {
		levelNodes[i] = nm.idx
	}
}

// orderByMedianChildren orders nodes by median position of their children.
func (d *DAG) orderByMedianChildren(levelNodes []int, childLevel []int) {
	type nodeMedian struct {
		idx    int
		median float64
	}

	medians := make([]nodeMedian, len(levelNodes))
	for pos, idx := range levelNodes {
		childIndices := d.getChildrenIndices(idx)

		if len(childIndices) == 0 {
			medians[pos] = nodeMedian{idx: idx, median: float64(pos)}
			continue
		}

		// Find positions of children in the child level
		childPositions := make([]int, 0, len(childIndices))
		for _, cIdx := range childIndices {
			for i, lvlIdx := range childLevel {
				if lvlIdx == cIdx {
					childPositions = append(childPositions, i)
					break
				}
			}
		}

		if len(childPositions) == 0 {
			medians[pos] = nodeMedian{idx: idx, median: float64(pos)}
			continue
		}

		sort.Ints(childPositions)
		var median float64
		if len(childPositions)%2 == 1 {
			median = float64(childPositions[len(childPositions)/2])
		} else {
			mid := len(childPositions) / 2
			median = float64(childPositions[mid-1]+childPositions[mid]) / 2.0
		}
		medians[pos] = nodeMedian{idx: idx, median: median}
	}

	// Sort by median
	sort.Slice(medians, func(i, j int) bool {
		return medians[i].median < medians[j].median
	})

	for i, nm := range medians {
		levelNodes[i] = nm.idx
	}
}

// findSubgraphs finds disconnected subgraphs in the DAG.
func (d *DAG) findSubgraphs() [][]int {
	n := len(d.nodes)
	visited := make([]bool, n)
	var subgraphs [][]int

	for i := 0; i < n; i++ {
		if !visited[i] {
			subgraph := make([]int, 0)
			d.collectConnected(i, visited, &subgraph)
			subgraphs = append(subgraphs, subgraph)
		}
	}

	return subgraphs
}

// collectConnected collects all nodes connected to the given node.
func (d *DAG) collectConnected(startIdx int, visited []bool, subgraph *[]int) {
	stack := []int{startIdx}

	for len(stack) > 0 {
		idx := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[idx] {
			continue
		}
		visited[idx] = true
		*subgraph = append(*subgraph, idx)

		nodeID := d.nodes[idx].id

		// Follow edges in both directions
		for _, e := range d.edges {
			if e.from == nodeID {
				if childIdx := d.nodeIndex(e.to); childIdx >= 0 && !visited[childIdx] {
					stack = append(stack, childIdx)
				}
			}
			if e.to == nodeID {
				if parentIdx := d.nodeIndex(e.from); parentIdx >= 0 && !visited[parentIdx] {
					stack = append(stack, parentIdx)
				}
			}
		}
	}
}

// isSubgraphSimpleChain checks if a subgraph is a simple chain (no branching).
func (d *DAG) isSubgraphSimpleChain(subgraphIndices []int) bool {
	for _, idx := range subgraphIndices {
		if d.parentsCount(idx) > 1 || d.childrenCount(idx) > 1 {
			return false
		}
	}
	return true
}

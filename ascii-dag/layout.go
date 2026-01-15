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

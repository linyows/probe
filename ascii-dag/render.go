package asciidag

import (
	"strings"
)

// Unicode box-drawing characters
const (
	vLine      = '│'
	hLine      = '─'
	arrowDown  = '↓'
	arrowRight = '→'
	cycleArrow = '⇄'
	cornerDR   = '└' // Down-Right (bottom-left corner)
	cornerDL   = '┘' // Down-Left (bottom-right corner)
	cornerUR   = '┌' // Up-Right (top-left corner)
	cornerUL   = '┐' // Up-Left (top-right corner)
	teeDown    = '┬' // T pointing down
	teeUp      = '┴' // T pointing up
	cross      = '┼'
)

// renderCycle renders a cycle detection message.
func (d *DAG) renderCycle(sb *strings.Builder) {
	cyclePath := d.FindCyclePath()
	if cyclePath == nil {
		sb.WriteString("Error: Cycle detected in graph\n")
		return
	}

	sb.WriteString("Error: Cycle detected: ")
	for i, id := range cyclePath {
		if i > 0 {
			sb.WriteString(" → ")
		}
		idx := d.nodeIndex(id)
		if idx >= 0 {
			sb.WriteString(d.formatNode(idx))
		}
	}
	sb.WriteString(" → ")
	idx := d.nodeIndex(cyclePath[0])
	if idx >= 0 {
		sb.WriteString(d.formatNode(idx))
	}
	sb.WriteString(" ")
	sb.WriteRune(cycleArrow)
	sb.WriteString("\n")
}

// renderHorizontal renders a simple chain horizontally.
func (d *DAG) renderHorizontal(sb *strings.Builder) {
	// Find root nodes (nodes with no parents)
	roots := make([]int, 0)
	for i := range d.nodes {
		if d.parentsCount(i) == 0 {
			roots = append(roots, i)
		}
	}

	if len(roots) == 0 && len(d.nodes) > 0 {
		roots = append(roots, 0)
	}

	// For each root, traverse and render
	for _, rootIdx := range roots {
		d.renderChainFrom(sb, rootIdx)
		sb.WriteString("\n")
	}
}

// renderChainFrom renders a chain starting from the given node.
func (d *DAG) renderChainFrom(sb *strings.Builder, startIdx int) {
	visited := make(map[int]bool)
	current := startIdx

	for !visited[current] {
		visited[current] = true

		if current != startIdx {
			sb.WriteString(" ")
			sb.WriteRune(arrowRight)
			sb.WriteString(" ")
		}
		sb.WriteString(d.formatNode(current))

		// Move to next node
		children := d.getChildrenIndices(current)
		if len(children) == 0 {
			break
		}
		current = children[0]
	}
}

// renderVertical renders the DAG using Sugiyama layout.
func (d *DAG) renderVertical(sb *strings.Builder) {
	// Calculate levels
	levelInfos := d.calculateLevels()

	// Find max level
	maxLevel := 0
	for _, li := range levelInfos {
		if li.level > maxLevel {
			maxLevel = li.level
		}
	}

	// Build level arrays
	levels := make([][]int, maxLevel+1)
	for i := range levels {
		levels[i] = make([]int, 0)
	}
	for _, li := range levelInfos {
		levels[li.level] = append(levels[li.level], li.index)
	}

	// Reduce crossings
	d.reduceCrossings(levels, maxLevel)

	// Calculate total width for centering
	maxWidth := 0
	levelWidths := make([]int, maxLevel+1)
	for level := 0; level <= maxLevel; level++ {
		width := 0
		for i, idx := range levels[level] {
			if i > 0 {
				width += 3 // spacing between nodes
			}
			width += d.nodeWidths[idx]
		}
		levelWidths[level] = width
		if width > maxWidth {
			maxWidth = width
		}
	}

	// Calculate X coordinates with centering
	xCoords := make([]int, len(d.nodes))
	for level := 0; level <= maxLevel; level++ {
		offset := (maxWidth - levelWidths[level]) / 2
		x := offset
		for _, idx := range levels[level] {
			xCoords[idx] = x
			x += d.nodeWidths[idx] + 3
		}
	}

	// Calculate center X for each node
	centerXs := make([]int, len(d.nodes))
	for i := range d.nodes {
		centerXs[i] = xCoords[i] + d.nodeWidths[i]/2
	}

	// Render each level
	for level := 0; level <= maxLevel; level++ {
		// Draw nodes
		line := make([]rune, maxWidth+1)
		for i := range line {
			line[i] = ' '
		}

		for _, idx := range levels[level] {
			formatted := []rune(d.formatNode(idx))
			x := xCoords[idx]
			for i, r := range formatted {
				if x+i < len(line) {
					line[x+i] = r
				}
			}
		}
		sb.WriteString(strings.TrimRight(string(line), " "))
		sb.WriteString("\n")

		// Draw connections to next level
		if level < maxLevel {
			d.drawConnections(sb, levels[level], levels[level+1], xCoords, centerXs, maxWidth)
		}
	}
}

// connection represents a connection between two nodes
type connection struct {
	fromIdx    int
	toIdx      int
	fromX      int // center X of source
	toX        int // center X of target
}

// drawConnections draws the connections between two levels
func (d *DAG) drawConnections(sb *strings.Builder, sourceLevel, targetLevel []int, xCoords, centerXs []int, maxWidth int) {
	// Collect all connections
	var connections []connection
	for _, srcIdx := range sourceLevel {
		srcID := d.nodes[srcIdx].id
		for _, tgtIdx := range targetLevel {
			// Check if there's an edge from src to tgt
			for _, e := range d.edges {
				if e.from == srcID && e.to == d.nodes[tgtIdx].id {
					connections = append(connections, connection{
						fromIdx: srcIdx,
						toIdx:   tgtIdx,
						fromX:   centerXs[srcIdx],
						toX:     centerXs[tgtIdx],
					})
					break
				}
			}
		}
	}

	if len(connections) == 0 {
		sb.WriteString("\n")
		return
	}

	// Analyze pattern
	targetCounts := make(map[int]int) // how many sources point to each target
	sourceCounts := make(map[int]int) // how many targets each source has

	for _, conn := range connections {
		targetCounts[conn.toIdx]++
		sourceCounts[conn.fromIdx]++
	}

	hasConvergence := false
	hasDivergence := false
	for _, count := range targetCounts {
		if count > 1 {
			hasConvergence = true
			break
		}
	}
	for _, count := range sourceCounts {
		if count > 1 {
			hasDivergence = true
			break
		}
	}

	// Draw connection line
	line := make([]rune, maxWidth+1)
	for i := range line {
		line[i] = ' '
	}

	convergenceJunctionX := -1

	if hasDivergence && !hasConvergence {
		// Pure divergence: one source to multiple targets
		d.drawDivergence(line, connections, centerXs)
	} else if hasConvergence && !hasDivergence {
		// Pure convergence: multiple sources to one target
		convergenceJunctionX = d.drawConvergence(line, connections, centerXs)
	} else if hasConvergence && hasDivergence {
		// Mixed: draw both patterns
		d.drawMixed(line, connections, centerXs)
	} else {
		// Simple 1-to-1 connections
		d.drawSimple(line, connections)
	}

	sb.WriteString(strings.TrimRight(string(line), " "))
	sb.WriteString("\n")

	// No vertical line needed for convergence - junction characters (┬, ┼) already indicate downward connection

	// Draw arrow line
	arrowLine := make([]rune, maxWidth+1)
	for i := range arrowLine {
		arrowLine[i] = ' '
	}

	if hasConvergence && !hasDivergence && convergenceJunctionX >= 0 {
		// For convergence, arrow is at junction position
		if convergenceJunctionX >= 0 && convergenceJunctionX < len(arrowLine) {
			arrowLine[convergenceJunctionX] = arrowDown
		}
	} else {
		// For other patterns, arrow at target positions
		for _, conn := range connections {
			if conn.toX >= 0 && conn.toX < len(arrowLine) {
				arrowLine[conn.toX] = arrowDown
			}
		}
	}
	sb.WriteString(strings.TrimRight(string(arrowLine), " "))
	sb.WriteString("\n")
}

// drawDivergence draws a divergence pattern (one source to multiple targets)
func (d *DAG) drawDivergence(line []rune, connections []connection, centerXs []int) {
	if len(connections) == 0 {
		return
	}

	// Group by source
	sourceGroups := make(map[int][]connection)
	for _, conn := range connections {
		sourceGroups[conn.fromIdx] = append(sourceGroups[conn.fromIdx], conn)
	}

	for _, conns := range sourceGroups {
		if len(conns) == 0 {
			continue
		}

		srcX := conns[0].fromX

		// Find min and max target X
		minX, maxX := conns[0].toX, conns[0].toX
		for _, conn := range conns {
			if conn.toX < minX {
				minX = conn.toX
			}
			if conn.toX > maxX {
				maxX = conn.toX
			}
		}

		// Draw horizontal line
		for x := minX; x <= maxX; x++ {
			if x >= 0 && x < len(line) {
				if line[x] == ' ' {
					line[x] = hLine
				}
			}
		}

		// Draw corners and source connection
		if srcX >= 0 && srcX < len(line) {
			switch srcX {
			case minX:
				line[srcX] = cornerUR
			case maxX:
				line[srcX] = cornerUL
			default:
				line[srcX] = teeUp
			}
		}

		// Draw target corners: ┌ for leftmost, ┐ for rightmost, ┬ for middle
		for _, conn := range conns {
			if conn.toX >= 0 && conn.toX < len(line) {
				switch conn.toX {
				case srcX:
					// Target is at source position - use cross if middle, otherwise keep source char
					if conn.toX > minX && conn.toX < maxX {
						line[conn.toX] = cross // ┼
					}
				case minX:
					line[conn.toX] = cornerUR // ┌
				case maxX:
					line[conn.toX] = cornerUL // ┐
				default:
					line[conn.toX] = teeDown // ┬
				}
			}
		}
	}
}

// drawConvergence draws a convergence pattern (multiple sources to one target)
// Returns the junction X position where vertical line should be drawn
func (d *DAG) drawConvergence(line []rune, connections []connection, centerXs []int) int {
	if len(connections) == 0 {
		return -1
	}

	junctionX := -1

	// Group by target
	targetGroups := make(map[int][]connection)
	for _, conn := range connections {
		targetGroups[conn.toIdx] = append(targetGroups[conn.toIdx], conn)
	}

	for _, conns := range targetGroups {
		if len(conns) == 0 {
			continue
		}

		tgtX := conns[0].toX

		// Find min and max source X
		minX, maxX := conns[0].fromX, conns[0].fromX
		for _, conn := range conns {
			if conn.fromX < minX {
				minX = conn.fromX
			}
			if conn.fromX > maxX {
				maxX = conn.fromX
			}
		}

		// Build set of source positions
		sourcePositions := make(map[int]bool)
		for _, conn := range conns {
			sourcePositions[conn.fromX] = true
		}

		// Determine junction point - always use target's center position
		// This ensures the arrow points to the target node
		junctionX = tgtX

		// Draw horizontal line
		for x := minX; x <= maxX; x++ {
			if x >= 0 && x < len(line) {
				if line[x] == ' ' {
					line[x] = hLine
				}
			}
		}

		// Draw source corners: └ for leftmost, ┘ for rightmost
		// Middle sources just get horizontal line (no ┴)
		for _, conn := range conns {
			if conn.fromX >= 0 && conn.fromX < len(line) {
				switch conn.fromX {
				case minX:
					line[conn.fromX] = cornerDR // └
				case maxX:
					line[conn.fromX] = cornerDL // ┘
				}
				// Middle sources: leave as horizontal line (already drawn)
			}
		}

		// Draw junction point at target's center position
		// Find the nearest middle source to the junction point
		nearestMiddleSource := -1
		minDistance := -1
		for _, conn := range conns {
			if conn.fromX > minX && conn.fromX < maxX {
				// This is a middle source
				dist := conn.fromX - junctionX
				if dist < 0 {
					dist = -dist
				}
				if minDistance < 0 || dist < minDistance {
					minDistance = dist
					nearestMiddleSource = conn.fromX
				}
			}
		}

		// If there's a middle source, use ┼ at that position and adjust junction
		if nearestMiddleSource >= 0 {
			line[nearestMiddleSource] = cross // ┼
			junctionX = nearestMiddleSource
		} else if junctionX >= 0 && junctionX < len(line) && junctionX > minX && junctionX < maxX {
			// No middle source, draw ┬ at junction position
			line[junctionX] = teeDown // ┬
		}
	}

	return junctionX
}

// drawMixed draws mixed convergence and divergence patterns
func (d *DAG) drawMixed(line []rune, connections []connection, centerXs []int) {
	// For mixed patterns, draw simple vertical lines
	for _, conn := range connections {
		if conn.fromX == conn.toX {
			if conn.fromX >= 0 && conn.fromX < len(line) {
				line[conn.fromX] = vLine
			}
		} else {
			// Draw corner connection
			minX := conn.fromX
			maxX := conn.toX
			if minX > maxX {
				minX, maxX = maxX, minX
			}
			for x := minX; x <= maxX; x++ {
				if x >= 0 && x < len(line) {
					switch line[x] {
					case ' ':
						line[x] = hLine
					case vLine:
						line[x] = cross
					}
				}
			}
			if conn.fromX >= 0 && conn.fromX < len(line) {
				if conn.fromX < conn.toX {
					line[conn.fromX] = cornerDR
				} else {
					line[conn.fromX] = cornerDL
				}
			}
			if conn.toX >= 0 && conn.toX < len(line) {
				if conn.toX < conn.fromX {
					line[conn.toX] = cornerUR
				} else {
					line[conn.toX] = cornerUL
				}
			}
		}
	}
}

// drawSimple draws simple 1-to-1 connections
func (d *DAG) drawSimple(line []rune, connections []connection) {
	for _, conn := range connections {
		if conn.fromX == conn.toX {
			if conn.fromX >= 0 && conn.fromX < len(line) {
				line[conn.fromX] = vLine
			}
		} else {
			// Draw corner connection
			minX := conn.fromX
			maxX := conn.toX
			if minX > maxX {
				minX, maxX = maxX, minX
			}
			for x := minX; x <= maxX; x++ {
				if x >= 0 && x < len(line) {
					if line[x] == ' ' {
						line[x] = hLine
					}
				}
			}
			if conn.fromX >= 0 && conn.fromX < len(line) {
				if conn.fromX < conn.toX {
					line[conn.fromX] = cornerDR
				} else {
					line[conn.fromX] = cornerDL
				}
			}
			if conn.toX >= 0 && conn.toX < len(line) {
				if conn.toX < conn.fromX {
					line[conn.toX] = cornerUR
				} else {
					line[conn.toX] = cornerUL
				}
			}
		}
	}
}

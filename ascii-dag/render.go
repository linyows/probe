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

	for {
		if visited[current] {
			break
		}
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
			if srcX == minX {
				line[srcX] = cornerUR
			} else if srcX == maxX {
				line[srcX] = cornerUL
			} else {
				line[srcX] = teeUp
			}
		}

		// Draw target corners: ┌ for leftmost, ┐ for rightmost, ┬ for middle
		for _, conn := range conns {
			if conn.toX >= 0 && conn.toX < len(line) {
				if conn.toX == srcX {
					// Target is at source position - use cross if middle, otherwise keep source char
					if conn.toX > minX && conn.toX < maxX {
						line[conn.toX] = cross // ┼
					}
				} else if conn.toX == minX {
					line[conn.toX] = cornerUR // ┌
				} else if conn.toX == maxX {
					line[conn.toX] = cornerUL // ┐
				} else {
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
				if conn.fromX == minX {
					line[conn.fromX] = cornerDR // └
				} else if conn.fromX == maxX {
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
					if line[x] == ' ' {
						line[x] = hLine
					} else if line[x] == vLine {
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

// drawNode draws a node on the grid.
func (d *DAG) drawNode(grid [][]rune, node *LayoutNode, idx int) {
	formatted := d.formatNode(idx)
	runes := []rune(formatted)

	for i, r := range runes {
		x := node.X + i
		if x < len(grid[node.Y]) {
			grid[node.Y][x] = r
		}
	}
}

// drawEdge draws an edge on the grid.
func (d *DAG) drawEdge(grid [][]rune, edge LayoutEdge) {
	switch path := edge.Path.(type) {
	case EdgePathDirect:
		d.drawDirectEdge(grid, edge)
	case EdgePathCorner:
		d.drawCornerEdge(grid, edge, path.HorizontalY)
	case EdgePathSideChannel:
		d.drawSideChannelEdge(grid, edge, path)
	case EdgePathMultiSegment:
		d.drawMultiSegmentEdge(grid, edge, path)
	}
}

// drawDirectEdge draws a direct vertical edge.
func (d *DAG) drawDirectEdge(grid [][]rune, edge LayoutEdge) {
	x := edge.FromX
	for y := edge.FromY; y < edge.ToY; y++ {
		if y >= 0 && y < len(grid) && x >= 0 && x < len(grid[y]) {
			if y == edge.ToY-1 {
				grid[y][x] = arrowDown
			} else {
				grid[y][x] = vLine
			}
		}
	}
}

// drawCornerEdge draws an L-shaped edge.
func (d *DAG) drawCornerEdge(grid [][]rune, edge LayoutEdge, horizontalY int) {
	fromX := edge.FromX
	toX := edge.ToX

	// Vertical line from source
	for y := edge.FromY; y <= horizontalY; y++ {
		if y >= 0 && y < len(grid) && fromX >= 0 && fromX < len(grid[y]) {
			if grid[y][fromX] == ' ' {
				grid[y][fromX] = vLine
			}
		}
	}

	// Horizontal line
	minX, maxX := fromX, toX
	if minX > maxX {
		minX, maxX = maxX, minX
	}

	if horizontalY >= 0 && horizontalY < len(grid) {
		for x := minX; x <= maxX; x++ {
			if x >= 0 && x < len(grid[horizontalY]) {
				current := grid[horizontalY][x]
				if current == ' ' {
					grid[horizontalY][x] = hLine
				} else if current == vLine {
					grid[horizontalY][x] = cross
				}
			}
		}

		// Draw corners
		if fromX >= 0 && fromX < len(grid[horizontalY]) {
			if fromX < toX {
				grid[horizontalY][fromX] = cornerDR
			} else {
				grid[horizontalY][fromX] = cornerDL
			}
		}
		if toX >= 0 && toX < len(grid[horizontalY]) {
			if fromX < toX {
				grid[horizontalY][toX] = cornerUL
			} else {
				grid[horizontalY][toX] = cornerUR
			}
		}
	}

	// Vertical line to target
	for y := horizontalY + 1; y < edge.ToY; y++ {
		if y >= 0 && y < len(grid) && toX >= 0 && toX < len(grid[y]) {
			if grid[y][toX] == ' ' {
				grid[y][toX] = vLine
			}
		}
	}

	// Arrow at target
	if edge.ToY-1 >= 0 && edge.ToY-1 < len(grid) && toX >= 0 && toX < len(grid[edge.ToY-1]) {
		grid[edge.ToY-1][toX] = arrowDown
	}
}

// drawSideChannelEdge draws a side-channel routed edge.
func (d *DAG) drawSideChannelEdge(grid [][]rune, edge LayoutEdge, path EdgePathSideChannel) {
	// Similar to corner edge but with side routing
	d.drawCornerEdge(grid, edge, path.StartY)
}

// drawMultiSegmentEdge draws a multi-segment edge through waypoints.
func (d *DAG) drawMultiSegmentEdge(grid [][]rune, edge LayoutEdge, path EdgePathMultiSegment) {
	points := make([][2]int, 0, len(path.Waypoints)+2)
	points = append(points, [2]int{edge.FromX, edge.FromY})
	points = append(points, path.Waypoints...)
	points = append(points, [2]int{edge.ToX, edge.ToY})

	for i := 0; i < len(points)-1; i++ {
		from := points[i]
		to := points[i+1]

		if from[0] == to[0] {
			// Vertical segment
			minY, maxY := from[1], to[1]
			if minY > maxY {
				minY, maxY = maxY, minY
			}
			for y := minY; y <= maxY; y++ {
				if y >= 0 && y < len(grid) && from[0] >= 0 && from[0] < len(grid[y]) {
					if grid[y][from[0]] == ' ' {
						grid[y][from[0]] = vLine
					}
				}
			}
		} else {
			// Horizontal segment
			minX, maxX := from[0], to[0]
			if minX > maxX {
				minX, maxX = maxX, minX
			}
			y := from[1]
			if y >= 0 && y < len(grid) {
				for x := minX; x <= maxX; x++ {
					if x >= 0 && x < len(grid[y]) && grid[y][x] == ' ' {
						grid[y][x] = hLine
					}
				}
			}
		}
	}

	// Arrow at target
	if edge.ToY-1 >= 0 && edge.ToY-1 < len(grid) && edge.ToX >= 0 && edge.ToX < len(grid[edge.ToY-1]) {
		grid[edge.ToY-1][edge.ToX] = arrowDown
	}
}

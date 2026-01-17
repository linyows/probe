package probe

import (
	"strings"
)

const (
	// Node border characters (string)
	nodeTopLeft     = "╭"
	nodeTopRight    = "╮"
	nodeBottomLeft  = "╰"
	nodeBottomRight = "╯"
	nodeHorizontal  = "─"
	nodeVertical    = "│"
	nodeTeeRight    = "├"
	nodeTeeDown     = "┬"
	nodeTeeUp       = "┴"
	nodeTeeLeft     = "┤"
	nodeCross       = "┼"
	arrowDown          = "↓"
	stepBullet         = "○"
	stepBulletEmbedded = "↗"
	ellipsis           = "…"

	// Connection line characters (rune)
	connVertical          = '│'
	connHorizontal        = '─'
	connTeeRight          = '├'
	connTeeLeft           = '┤'
	connTeeDown           = '┬'
	connTeeUp             = '┴'
	connCross             = '┼'
	connCornerTopLeft     = '┌'
	connCornerTopRight    = '┐'
	connCornerBottomLeft  = '└'
	connCornerBottomRight = '┘'

	// Node dimensions
	fixedNodeWidth = 25
)

// DagAsciiJobNode represents a rendered job node
type DagAsciiJobNode struct {
	Job         *Job
	JobID       string
	Level       int      // Depth in DAG (0 = root)
	Width       int      // Box width
	Lines       []string // Rendered lines
	CenterX     int      // X coordinate of center (for connections)
	HasChildren bool     // Whether this job has dependent jobs
}

// DagAsciiRenderer renders detailed workflow graphs with job nodes and steps
type DagAsciiRenderer struct {
	workflow   *Workflow
	nodes      []*DagAsciiJobNode
	levels     [][]int             // levels[level] = []jobIndex
	jobIDToIdx map[string]int      // jobID -> index in workflow.Jobs
	children   map[string][]string // jobID -> list of child jobIDs (jobs that depend on this job)
	parents    map[string][]string // jobID -> list of parent jobIDs (jobs this job depends on)
}

// NewDagAsciiRenderer creates a new DagAsciiRenderer
func NewDagAsciiRenderer(w *Workflow) *DagAsciiRenderer {
	r := &DagAsciiRenderer{
		workflow:   w,
		nodes:      make([]*DagAsciiJobNode, len(w.Jobs)),
		jobIDToIdx: make(map[string]int),
		children:   make(map[string][]string),
		parents:    make(map[string][]string),
	}

	// Build job ID to index mapping
	for i, job := range w.Jobs {
		id := job.ID
		if id == "" {
			id = job.Name
		}
		r.jobIDToIdx[id] = i
	}

	// Build parent-child relationships
	for _, job := range w.Jobs {
		jobID := job.ID
		if jobID == "" {
			jobID = job.Name
		}
		for _, need := range job.Needs {
			r.children[need] = append(r.children[need], jobID)
			r.parents[jobID] = append(r.parents[jobID], need)
		}
	}

	return r
}

// Render generates the detailed ASCII art graph
func (r *DagAsciiRenderer) Render() string {
	if len(r.workflow.Jobs) == 0 {
		return ""
	}

	r.calculateLevels()
	r.createNodes()

	var result []string

	for level := 0; level < len(r.levels); level++ {
		// Render connection lines from previous level
		if level > 0 {
			connections := r.renderConnections(level - 1)
			result = append(result, connections...)
		}

		// Render nodes at this level
		levelLines := r.renderLevel(level)
		result = append(result, levelLines...)
	}

	return strings.Join(result, "\n") + "\n"
}

// calculateLevels assigns each job to a level based on dependencies (Sugiyama-style)
func (r *DagAsciiRenderer) calculateLevels() {
	jobLevels := make(map[string]int)

	// Calculate level for each job
	var calcLevel func(jobID string) int
	calcLevel = func(jobID string) int {
		if level, exists := jobLevels[jobID]; exists {
			return level
		}

		idx, ok := r.jobIDToIdx[jobID]
		if !ok {
			return 0
		}

		job := r.workflow.Jobs[idx]
		if len(job.Needs) == 0 {
			jobLevels[jobID] = 0
			return 0
		}

		maxParentLevel := -1
		for _, need := range job.Needs {
			parentLevel := calcLevel(need)
			if parentLevel > maxParentLevel {
				maxParentLevel = parentLevel
			}
		}

		level := maxParentLevel + 1
		jobLevels[jobID] = level
		return level
	}

	// Calculate levels for all jobs
	maxLevel := 0
	for _, job := range r.workflow.Jobs {
		jobID := job.ID
		if jobID == "" {
			jobID = job.Name
		}
		level := calcLevel(jobID)
		if level > maxLevel {
			maxLevel = level
		}
	}

	// Group jobs by level
	r.levels = make([][]int, maxLevel+1)
	for i, job := range r.workflow.Jobs {
		jobID := job.ID
		if jobID == "" {
			jobID = job.Name
		}
		level := jobLevels[jobID]
		r.levels[level] = append(r.levels[level], i)
	}

	// Sort jobs within each level: jobs with children first, then jobs without children
	for level := range r.levels {
		jobIndices := r.levels[level]
		withChildren := []int{}
		withoutChildren := []int{}

		for _, idx := range jobIndices {
			job := r.workflow.Jobs[idx]
			jobID := job.ID
			if jobID == "" {
				jobID = job.Name
			}
			if len(r.children[jobID]) > 0 {
				withChildren = append(withChildren, idx)
			} else {
				withoutChildren = append(withoutChildren, idx)
			}
		}

		r.levels[level] = append(withChildren, withoutChildren...)
	}
}

// createNodes creates DagAsciiJobNode for each job
func (r *DagAsciiRenderer) createNodes() {
	for i, job := range r.workflow.Jobs {
		jobID := job.ID
		if jobID == "" {
			jobID = job.Name
		}

		// Check if this job has children (other jobs depend on it)
		hasChildren := len(r.children[jobID]) > 0

		node := &DagAsciiJobNode{
			Job:         &r.workflow.Jobs[i],
			JobID:       jobID,
			Width:       fixedNodeWidth, // Use fixed width for all nodes
			HasChildren: hasChildren,
		}

		// Find level for this job
		for level, jobIndices := range r.levels {
			for _, idx := range jobIndices {
				if idx == i {
					node.Level = level
					break
				}
			}
		}

		node.Lines = r.renderDagAsciiJobNode(node)
		r.nodes[i] = node
	}
}

// renderDagAsciiJobNode renders a single job node
func (r *DagAsciiRenderer) renderDagAsciiJobNode(node *DagAsciiJobNode) []string {
	var lines []string
	width := node.Width
	innerWidth := width - 2 // Width inside borders

	// Top border
	lines = append(lines, nodeTopLeft+strings.Repeat(nodeHorizontal, innerWidth)+nodeTopRight)

	// Job name (centered, truncate with ellipsis if too long)
	name := truncateWithEllipsis(node.Job.Name, innerWidth-2) // -2 for padding
	padding := innerWidth - runeWidth(name)
	leftPad := padding / 2
	rightPad := padding - leftPad
	lines = append(lines, nodeVertical+strings.Repeat(" ", leftPad)+name+strings.Repeat(" ", rightPad)+nodeVertical)

	// Separator
	lines = append(lines, nodeTeeRight+strings.Repeat(nodeHorizontal, innerWidth)+nodeTeeLeft)

	// Steps
	for _, step := range node.Job.Steps {
		stepName := step.Name
		if stepName == "" {
			stepName = step.Uses
		}
		// Use different bullet for embedded actions
		bullet := stepBullet
		if step.Uses == "embedded" {
			bullet = stepBulletEmbedded
		}
		// Format: " ○ stepname" or " ↗ stepname" with truncation
		prefix := " " + bullet + " "
		prefixWidth := runeWidth(prefix)
		maxStepNameWidth := innerWidth - prefixWidth
		truncatedStepName := truncateWithEllipsis(stepName, maxStepNameWidth)
		stepLine := prefix + truncatedStepName
		// Pad to inner width
		stepLineWidth := runeWidth(stepLine)
		if stepLineWidth < innerWidth {
			stepLine = stepLine + strings.Repeat(" ", innerWidth-stepLineWidth)
		}
		lines = append(lines, nodeVertical+stepLine+nodeVertical)
	}

	// Bottom border - use ┬ in center if job has children
	if node.HasChildren {
		leftWidth := (innerWidth - 1) / 2
		rightWidth := innerWidth - 1 - leftWidth
		bottomBorder := nodeBottomLeft + strings.Repeat(nodeHorizontal, leftWidth) + nodeTeeDown + strings.Repeat(nodeHorizontal, rightWidth) + nodeBottomRight
		lines = append(lines, bottomBorder)
	} else {
		lines = append(lines, nodeBottomLeft+strings.Repeat(nodeHorizontal, innerWidth)+nodeBottomRight)
	}

	return lines
}

// truncateWithEllipsis truncates a string to maxLen runes, adding ellipsis if truncated
func truncateWithEllipsis(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + ellipsis
}

// runeWidth returns the display width of a string (counting runes)
func runeWidth(s string) int {
	return len([]rune(s))
}

// flagsToChar converts connection flags to the appropriate box-drawing character
// Flags: 1=from_above, 2=to_below, 4=from_left, 8=to_right
func flagsToChar(flags int) rune {
	fromAbove := flags&1 != 0
	toBelow := flags&2 != 0
	fromLeft := flags&4 != 0
	toRight := flags&8 != 0

	switch {
	case fromAbove && toBelow && fromLeft && toRight:
		return connCross
	case fromAbove && toBelow && fromLeft:
		return connTeeLeft
	case fromAbove && toBelow && toRight:
		return connTeeRight
	case fromAbove && fromLeft && toRight:
		return connTeeUp
	case toBelow && fromLeft && toRight:
		return connTeeDown
	case fromAbove && toBelow:
		return connVertical
	case fromLeft && toRight:
		return connHorizontal
	case fromAbove && toRight:
		return connCornerBottomLeft
	case fromAbove && fromLeft:
		return connCornerBottomRight
	case toBelow && toRight:
		return connCornerTopLeft
	case toBelow && fromLeft:
		return connCornerTopRight
	case fromAbove:
		return connVertical
	case toBelow:
		return connVertical
	case fromLeft, toRight:
		return connHorizontal
	default:
		return ' '
	}
}

// renderLevel renders all nodes at a given level side by side
func (r *DagAsciiRenderer) renderLevel(level int) []string {
	jobIndices := r.levels[level]
	if len(jobIndices) == 0 {
		return nil
	}

	// Get nodes for this level
	var levelNodes []*DagAsciiJobNode
	for _, idx := range jobIndices {
		levelNodes = append(levelNodes, r.nodes[idx])
	}

	// Find max height
	maxHeight := 0
	for _, node := range levelNodes {
		if len(node.Lines) > maxHeight {
			maxHeight = len(node.Lines)
		}
	}

	// Calculate positions and set center X
	spacing := 2
	currentX := 0
	for _, node := range levelNodes {
		node.CenterX = currentX + node.Width/2
		currentX += node.Width + spacing
	}

	// Render lines
	var result []string
	for lineIdx := 0; lineIdx < maxHeight; lineIdx++ {
		var line strings.Builder
		for i, node := range levelNodes {
			if i > 0 {
				line.WriteString(strings.Repeat(" ", spacing))
			}
			if lineIdx < len(node.Lines) {
				line.WriteString(node.Lines[lineIdx])
			} else {
				// Node ended but other nodes continue - draw vertical line if this job has children
				if node.HasChildren {
					// Draw vertical line at center position
					centerPos := node.Width / 2
					line.WriteString(strings.Repeat(" ", centerPos))
					line.WriteString("│")
					line.WriteString(strings.Repeat(" ", node.Width-centerPos-1))
				} else {
					line.WriteString(strings.Repeat(" ", node.Width))
				}
			}
		}
		result = append(result, line.String())
	}

	return result
}

// renderConnections renders connection lines between levels
func (r *DagAsciiRenderer) renderConnections(fromLevel int) []string {
	if fromLevel >= len(r.levels)-1 {
		return nil
	}

	parentIndices := r.levels[fromLevel]
	childIndices := r.levels[fromLevel+1]

	if len(parentIndices) == 0 || len(childIndices) == 0 {
		return nil
	}

	connections := r.buildConnectionMap(parentIndices, childIndices)
	if len(connections) == 0 {
		return nil
	}

	parentPositions, parentTotalWidth := r.calculateLevelPositions(parentIndices)
	childPositions, childTotalWidth := r.calculateLevelPositions(childIndices)

	totalWidth := max(parentTotalWidth, childTotalWidth)

	parentsWithConnections := r.getConnectedParents(connections)

	var result []string
	result = append(result, r.renderVerticalLine(parentPositions, parentsWithConnections, totalWidth))

	if r.needsRoutingLines(connections, parentPositions, childPositions) {
		result = append(result, r.renderRoutingLine(connections, parentPositions, childPositions, totalWidth))
	} else {
		result = append(result, r.renderVerticalLine(parentPositions, parentsWithConnections, totalWidth))
	}

	result = append(result, r.renderArrowLine(childPositions, connections, totalWidth))

	return result
}

// buildConnectionMap builds a map of childIdx -> []parentIdx for dependencies at the given level
func (r *DagAsciiRenderer) buildConnectionMap(parentIndices, childIndices []int) map[int][]int {
	connections := make(map[int][]int)
	for _, childIdx := range childIndices {
		childNode := r.nodes[childIdx]
		for _, parentIdx := range parentIndices {
			parentNode := r.nodes[parentIdx]
			for _, need := range childNode.Job.Needs {
				if need == parentNode.JobID {
					connections[childIdx] = append(connections[childIdx], parentIdx)
					break
				}
			}
		}
	}
	return connections
}

// calculateLevelPositions calculates the center X positions for each job at a level
func (r *DagAsciiRenderer) calculateLevelPositions(jobIndices []int) (positions map[int]int, totalWidth int) {
	const spacing = 2
	positions = make(map[int]int)
	currentX := 0
	for _, idx := range jobIndices {
		node := r.nodes[idx]
		positions[idx] = currentX + node.Width/2
		currentX += node.Width + spacing
	}
	if currentX > 0 {
		totalWidth = currentX - spacing
	}
	return positions, totalWidth
}

// getConnectedParents returns a set of parent indices that have connections
func (r *DagAsciiRenderer) getConnectedParents(connections map[int][]int) map[int]bool {
	parentsWithConnections := make(map[int]bool)
	for _, parentList := range connections {
		for _, parentIdx := range parentList {
			parentsWithConnections[parentIdx] = true
		}
	}
	return parentsWithConnections
}

// needsRoutingLines checks if routing lines are needed (when parent and child positions differ)
func (r *DagAsciiRenderer) needsRoutingLines(connections map[int][]int, parentPositions, childPositions map[int]int) bool {
	for childIdx, parents := range connections {
		childPos := childPositions[childIdx]
		for _, parentIdx := range parents {
			if parentPositions[parentIdx] != childPos {
				return true
			}
		}
	}
	return false
}

// renderVerticalLine renders a line with vertical bars at the specified positions
func (r *DagAsciiRenderer) renderVerticalLine(positions map[int]int, connectedIndices map[int]bool, totalWidth int) string {
	line := make([]rune, totalWidth)
	for i := range line {
		line[i] = ' '
	}
	for idx := range connectedIndices {
		if pos, ok := positions[idx]; ok && pos < len(line) {
			line[pos] = '│'
		}
	}
	return string(line)
}

// renderRoutingLine renders the routing line with appropriate box-drawing characters
func (r *DagAsciiRenderer) renderRoutingLine(connections map[int][]int, parentPositions, childPositions map[int]int, totalWidth int) string {
	// Collect connection flags for each position
	// Flags: 1=from_above, 2=to_below, 4=from_left, 8=to_right
	posFlags := make(map[int]int)

	for childIdx, parents := range connections {
		childPos := childPositions[childIdx]

		for _, parentIdx := range parents {
			parentPos := parentPositions[parentIdx]

			if parentPos == childPos {
				// Straight vertical
				posFlags[parentPos] |= 1 | 2 // from_above | to_below
			} else if parentPos < childPos {
				// Parent is left of child
				posFlags[parentPos] |= 1 | 8 // from_above | to_right
				posFlags[childPos] |= 4 | 2  // from_left | to_below
				for i := parentPos + 1; i < childPos; i++ {
					posFlags[i] |= 4 | 8 // horizontal
				}
			} else {
				// Parent is right of child
				posFlags[childPos] |= 8 | 2  // to_right | to_below
				posFlags[parentPos] |= 4 | 1 // from_left | from_above
				for i := childPos + 1; i < parentPos; i++ {
					posFlags[i] |= 4 | 8 // horizontal
				}
			}
		}
	}

	line := make([]rune, totalWidth)
	for i := range line {
		line[i] = ' '
	}
	for pos, flags := range posFlags {
		if pos < len(line) {
			line[pos] = flagsToChar(flags)
		}
	}
	return string(line)
}

// renderArrowLine renders the arrow line pointing to child positions
func (r *DagAsciiRenderer) renderArrowLine(childPositions map[int]int, connections map[int][]int, totalWidth int) string {
	line := make([]rune, totalWidth)
	for i := range line {
		line[i] = ' '
	}
	for childIdx := range connections {
		if pos, ok := childPositions[childIdx]; ok && pos < len(line) {
			line[pos] = '↓'
		}
	}
	return string(line)
}

package asciidag

// EdgePath describes how an edge is routed between nodes.
type EdgePath interface {
	isEdgePath()
}

// EdgePathDirect represents a direct vertical connection.
type EdgePathDirect struct{}

func (EdgePathDirect) isEdgePath() {}

// EdgePathCorner represents an L-shaped connection.
type EdgePathCorner struct {
	HorizontalY int
}

func (EdgePathCorner) isEdgePath() {}

// EdgePathSideChannel represents a side-channel routing for skip-level edges.
type EdgePathSideChannel struct {
	ChannelX int
	StartY   int
	EndY     int
}

func (EdgePathSideChannel) isEdgePath() {}

// EdgePathMultiSegment represents a multi-segment path through waypoints.
type EdgePathMultiSegment struct {
	Waypoints [][2]int // (x, y) pairs
}

func (EdgePathMultiSegment) isEdgePath() {}

// LayoutNode represents a positioned node in the layout.
type LayoutNode struct {
	ID            int
	Label         string
	X             int // Left edge (character cells)
	Y             int // Top edge (lines)
	Width         int // Width including brackets
	CenterX       int // Center X for edge routing
	Level         int // Depth in hierarchy
	LevelPosition int // Position within level (left to right, 0-indexed)
}

// LayoutEdge represents a routed edge in the layout.
type LayoutEdge struct {
	FromID    int
	ToID      int
	FromX     int // Center X of source node
	FromY     int // Bottom of source node
	ToX       int // Center X of target node
	ToY       int // Top of target node
	Path      EdgePath
	EdgeIndex int // For consistent coloring
}

// lineOccupancy tracks what's on each line for scanline rendering.
type lineOccupancy struct {
	nodeIndices []int
	edgeIndices []int
}

// LayoutIR is the intermediate representation of the layout.
type LayoutIR struct {
	nodes      []LayoutNode
	edges      []LayoutEdge
	width      int
	height     int
	levelCount int
	levels     [][]int       // Node indices per level
	idToIndex  map[int]int   // O(1) node ID lookup
	yIndex     []lineOccupancy // Lazy-built spatial index
}

// Nodes returns the layout nodes.
func (ir *LayoutIR) Nodes() []LayoutNode {
	return ir.nodes
}

// Edges returns the layout edges.
func (ir *LayoutIR) Edges() []LayoutEdge {
	return ir.edges
}

// Width returns the total width of the layout.
func (ir *LayoutIR) Width() int {
	return ir.width
}

// Height returns the total height of the layout.
func (ir *LayoutIR) Height() int {
	return ir.height
}

// LevelCount returns the number of levels.
func (ir *LayoutIR) LevelCount() int {
	return ir.levelCount
}

// NodeByID returns a node by its ID.
func (ir *LayoutIR) NodeByID(id int) *LayoutNode {
	if idx, ok := ir.idToIndex[id]; ok {
		return &ir.nodes[idx]
	}
	return nil
}

// buildYIndex builds the spatial index for scanline rendering.
func (ir *LayoutIR) buildYIndex() {
	if ir.yIndex != nil {
		return
	}

	ir.yIndex = make([]lineOccupancy, ir.height)

	// Add nodes to their lines
	for i, node := range ir.nodes {
		if node.Y >= 0 && node.Y < ir.height {
			ir.yIndex[node.Y].nodeIndices = append(ir.yIndex[node.Y].nodeIndices, i)
		}
	}

	// Add edges to the lines they cross
	for i, edge := range ir.edges {
		minY := edge.FromY
		maxY := edge.ToY
		if minY > maxY {
			minY, maxY = maxY, minY
		}
		for y := minY; y <= maxY && y < ir.height; y++ {
			ir.yIndex[y].edgeIndices = append(ir.yIndex[y].edgeIndices, i)
		}
	}
}

// getLineOccupancy returns what's on a specific line.
func (ir *LayoutIR) getLineOccupancy(y int) *lineOccupancy {
	ir.buildYIndex()
	if y < 0 || y >= len(ir.yIndex) {
		return nil
	}
	return &ir.yIndex[y]
}

// ComputeLayout computes the layout intermediate representation.
func (d *DAG) ComputeLayout() *LayoutIR {
	if len(d.nodes) == 0 {
		return &LayoutIR{
			idToIndex: make(map[int]int),
		}
	}

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

	// Assign X coordinates
	const nodeSpacing = 3
	xCoords := make([]int, len(d.nodes))
	levelYs := make([]int, maxLevel+1)

	// Calculate Y for each level (3 lines per level: node, connector, gap)
	y := 0
	for level := 0; level <= maxLevel; level++ {
		levelYs[level] = y
		y += 3 // Node line + connector lines + gap
	}

	// Calculate X for each node
	for level := 0; level <= maxLevel; level++ {
		x := 0
		for _, idx := range levels[level] {
			xCoords[idx] = x
			x += d.nodeWidths[idx] + nodeSpacing
		}
	}

	// Calculate total width
	totalWidth := 0
	for level := 0; level <= maxLevel; level++ {
		levelWidth := 0
		for _, idx := range levels[level] {
			end := xCoords[idx] + d.nodeWidths[idx]
			if end > levelWidth {
				levelWidth = end
			}
		}
		if levelWidth > totalWidth {
			totalWidth = levelWidth
		}
	}

	// Build layout nodes
	layoutNodes := make([]LayoutNode, len(d.nodes))
	idToIndex := make(map[int]int)
	for level := 0; level <= maxLevel; level++ {
		for pos, idx := range levels[level] {
			n := d.nodes[idx]
			width := d.nodeWidths[idx]
			x := xCoords[idx]
			layoutNodes[idx] = LayoutNode{
				ID:            n.id,
				Label:         n.label,
				X:             x,
				Y:             levelYs[level],
				Width:         width,
				CenterX:       x + width/2,
				Level:         level,
				LevelPosition: pos,
			}
			idToIndex[n.id] = idx
		}
	}

	// Build layout edges
	layoutEdges := make([]LayoutEdge, 0, len(d.edges))
	for i, e := range d.edges {
		fromIdx := d.nodeIndex(e.from)
		toIdx := d.nodeIndex(e.to)
		if fromIdx < 0 || toIdx < 0 {
			continue
		}

		fromNode := layoutNodes[fromIdx]
		toNode := layoutNodes[toIdx]

		var path EdgePath
		if fromNode.CenterX == toNode.CenterX {
			path = EdgePathDirect{}
		} else {
			path = EdgePathCorner{HorizontalY: fromNode.Y + 1}
		}

		layoutEdges = append(layoutEdges, LayoutEdge{
			FromID:    e.from,
			ToID:      e.to,
			FromX:     fromNode.CenterX,
			FromY:     fromNode.Y + 1, // Below the node
			ToX:       toNode.CenterX,
			ToY:       toNode.Y - 1, // Above the target node
			Path:      path,
			EdgeIndex: i,
		})
	}

	return &LayoutIR{
		nodes:      layoutNodes,
		edges:      layoutEdges,
		width:      totalWidth,
		height:     y,
		levelCount: maxLevel + 1,
		levels:     levels,
		idToIndex:  idToIndex,
	}
}

package asciidag

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

// ParseDOT parses a DOT format string and returns a DAG.
// Supports basic digraph syntax with nodes and edges.
//
// Example DOT format:
//
//	digraph G {
//	    A -> B;
//	    B -> C;
//	    A [label="Start"];
//	}
func ParseDOT(dot string, opts ...Option) (*DAG, error) {
	return ParseDOTReader(strings.NewReader(dot), opts...)
}

// ParseDOTFile reads and parses a DOT file.
func ParseDOTFile(filename string, opts ...Option) (*DAG, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = f.Close() }()
	return ParseDOTReader(f, opts...)
}

// ParseDOTReader parses DOT format from an io.Reader.
func ParseDOTReader(r io.Reader, opts ...Option) (*DAG, error) {
	parser := &dotParser{
		nodeLabels: make(map[string]string),
		nodeIDs:    make(map[string]int),
		edges:      make([][2]string, 0),
		nextID:     1,
	}

	scanner := bufio.NewScanner(r)
	var content strings.Builder
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	if err := parser.parse(content.String()); err != nil {
		return nil, err
	}

	return parser.toDAG(opts...)
}

// dotParser handles parsing of DOT format.
type dotParser struct {
	graphName  string
	isDigraph  bool
	nodeLabels map[string]string // node name -> label
	nodeIDs    map[string]int    // node name -> numeric ID
	edges      [][2]string       // (from, to) pairs
	nextID     int
}

// Regex patterns for DOT parsing
var (
	// Match digraph or graph declaration
	graphDeclRe = regexp.MustCompile(`(?i)^\s*(strict\s+)?(digraph|graph)\s+(\w+|"[^"]*")?\s*\{`)
	// Match node with attributes: A [label="..."]
	nodeAttrRe = regexp.MustCompile(`^\s*(\w+|"[^"]*")\s*\[(.*)\]\s*;?\s*$`)
	// Match label attribute
	labelRe = regexp.MustCompile(`label\s*=\s*("([^"]*)"|(\w+))`)
	// Match comment lines
	commentRe = regexp.MustCompile(`^\s*(//|#)`)
	// Match closing brace
	closeBraceRe = regexp.MustCompile(`^\s*\}\s*$`)
	// Match empty or whitespace-only lines
	emptyLineRe = regexp.MustCompile(`^\s*$`)
)

func (p *dotParser) parse(content string) error {
	// Remove block comments
	content = removeBlockComments(content)

	lines := strings.Split(content, "\n")
	inGraph := false

	for _, line := range lines {
		// Skip empty lines
		if emptyLineRe.MatchString(line) {
			continue
		}

		// Skip line comments
		if commentRe.MatchString(line) {
			continue
		}

		// Check for graph declaration
		if !inGraph {
			if matches := graphDeclRe.FindStringSubmatch(line); matches != nil {
				p.isDigraph = strings.ToLower(matches[2]) == "digraph"
				if len(matches) > 3 {
					p.graphName = unquote(matches[3])
				}
				inGraph = true
				continue
			}
		}

		// Check for closing brace
		if closeBraceRe.MatchString(line) {
			inGraph = false
			continue
		}

		if !inGraph {
			continue
		}

		// Try to parse as edge (supports chain: A -> B -> C)
		if strings.Contains(line, "->") || strings.Contains(line, "--") {
			edges := p.parseEdgeLine(line)
			if len(edges) > 0 {
				for _, edge := range edges {
					p.ensureNode(edge[0])
					p.ensureNode(edge[1])
					p.edges = append(p.edges, edge)
				}
				continue
			}
		}

		// Try to parse as node with attributes
		if matches := nodeAttrRe.FindStringSubmatch(line); matches != nil {
			nodeName := unquote(matches[1])
			attrs := matches[2]
			p.ensureNode(nodeName)

			// Extract label if present
			if labelMatches := labelRe.FindStringSubmatch(attrs); labelMatches != nil {
				if labelMatches[2] != "" {
					p.nodeLabels[nodeName] = labelMatches[2]
				} else if labelMatches[3] != "" {
					p.nodeLabels[nodeName] = labelMatches[3]
				}
			}
			continue
		}

		// Try to parse as simple node declaration (just a node name)
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimSuffix(trimmed, ";")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" && !strings.Contains(trimmed, "{") && !strings.Contains(trimmed, "}") {
			// Check if it's a valid identifier
			if isValidIdentifier(trimmed) {
				p.ensureNode(trimmed)
			}
		}
	}

	return nil
}

// parseEdgeLine parses an edge line that may contain chained edges (A -> B -> C)
func (p *dotParser) parseEdgeLine(line string) [][2]string {
	// Remove trailing semicolon and attributes
	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(line, ";")

	// Remove inline attributes [...]
	for {
		start := strings.Index(line, "[")
		if start == -1 {
			break
		}
		end := strings.Index(line[start:], "]")
		if end == -1 {
			break
		}
		line = line[:start] + line[start+end+1:]
	}

	// Split by -> or --
	var parts []string
	var separator string
	if strings.Contains(line, "->") {
		parts = strings.Split(line, "->")
		separator = "->"
	} else if strings.Contains(line, "--") {
		parts = strings.Split(line, "--")
		separator = "--"
	}

	if len(parts) < 2 {
		return nil
	}

	_ = separator // might be used for undirected graphs in future

	// Clean up parts
	var nodes []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			nodes = append(nodes, unquote(part))
		}
	}

	// Create edges between consecutive nodes
	var edges [][2]string
	for i := 0; i < len(nodes)-1; i++ {
		edges = append(edges, [2]string{nodes[i], nodes[i+1]})
	}

	return edges
}

func (p *dotParser) ensureNode(name string) {
	if _, exists := p.nodeIDs[name]; !exists {
		p.nodeIDs[name] = p.nextID
		p.nextID++
		// Default label is the node name
		if _, hasLabel := p.nodeLabels[name]; !hasLabel {
			p.nodeLabels[name] = name
		}
	}
}

func (p *dotParser) toDAG(opts ...Option) (*DAG, error) {
	dag := New(opts...)

	// Sort nodes by ID for stable ordering
	type nodeEntry struct {
		name string
		id   int
	}
	nodes := make([]nodeEntry, 0, len(p.nodeIDs))
	for name, id := range p.nodeIDs {
		nodes = append(nodes, nodeEntry{name, id})
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].id < nodes[j].id
	})

	// Add nodes in sorted order
	for _, n := range nodes {
		label := p.nodeLabels[n.name]
		dag.AddNode(n.id, label)
	}

	// Add edges
	for _, edge := range p.edges {
		fromID := p.nodeIDs[edge[0]]
		toID := p.nodeIDs[edge[1]]
		dag.AddEdge(fromID, toID)
	}

	return dag, nil
}

// removeBlockComments removes /* ... */ style comments
func removeBlockComments(s string) string {
	var result strings.Builder
	inComment := false

	for i := 0; i < len(s); i++ {
		if !inComment && i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			inComment = true
			i++ // skip *
			continue
		}
		if inComment && i+1 < len(s) && s[i] == '*' && s[i+1] == '/' {
			inComment = false
			i++ // skip /
			continue
		}
		if !inComment {
			result.WriteByte(s[i])
		}
	}

	return result.String()
}

// unquote removes surrounding quotes from a string
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// isValidIdentifier checks if a string is a valid DOT identifier
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	// Simple check: alphanumeric and underscore, not starting with digit
	for i, c := range s {
		isLower := c >= 'a' && c <= 'z'
		isUpper := c >= 'A' && c <= 'Z'
		isDigit := c >= '0' && c <= '9'
		isUnderscore := c == '_'

		if i == 0 {
			if !isLower && !isUpper && !isUnderscore {
				return false
			}
		} else {
			if !isLower && !isUpper && !isDigit && !isUnderscore {
				return false
			}
		}
	}
	return true
}

// ToDOT converts a DAG to DOT format string.
func (d *DAG) ToDOT(graphName string) string {
	var sb strings.Builder
	d.WriteDOT(&sb, graphName)
	return sb.String()
}

// WriteDOT writes the DAG in DOT format to the given writer.
func (d *DAG) WriteDOT(w io.Writer, graphName string) {
	if graphName == "" {
		graphName = "G"
	}

	_, _ = fmt.Fprintf(w, "digraph %s {\n", graphName)

	// Write nodes with labels
	for _, n := range d.nodes {
		label := n.label
		if label == "" {
			label = fmt.Sprintf("%d", n.id)
		}
		// Escape quotes in label
		label = strings.ReplaceAll(label, `"`, `\"`)
		_, _ = fmt.Fprintf(w, "    n%d [label=\"%s\"];\n", n.id, label)
	}

	// Write edges
	for _, e := range d.edges {
		_, _ = fmt.Fprintf(w, "    n%d -> n%d;\n", e.from, e.to)
	}

	_, _ = fmt.Fprintln(w, "}")
}

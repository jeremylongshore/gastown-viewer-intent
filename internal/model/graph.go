package model

import (
	"fmt"
	"strings"
)

// EdgeType represents the type of dependency relationship.
type EdgeType string

const (
	// Core dependency types
	EdgeTypeBlocks    EdgeType = "blocks"
	EdgeTypeBlockedBy EdgeType = "blocked_by"
	EdgeTypeParent    EdgeType = "parent"
	EdgeTypeChild     EdgeType = "child"

	// Async coordination
	EdgeTypeWaitsFor    EdgeType = "waits_for"
	EdgeTypeWaitedBy    EdgeType = "waited_by"
	EdgeTypeConditional EdgeType = "conditional_blocks"

	// Relationship types
	EdgeTypeRelates    EdgeType = "relates_to"
	EdgeTypeDuplicates EdgeType = "duplicates"
	EdgeTypeMentions   EdgeType = "mentions"

	// Derivation types
	EdgeTypeDerivedFrom EdgeType = "derived_from"
	EdgeTypeSupersedes  EdgeType = "supersedes"
	EdgeTypeImplements  EdgeType = "implements"

	// Unknown/default
	EdgeTypeUnknown EdgeType = "unknown"
)

// GraphNode represents a node in the dependency graph.
type GraphNode struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Status   Status   `json:"status"`
	Priority Priority `json:"priority"`
}

// GraphEdge represents a directed edge in the dependency graph.
type GraphEdge struct {
	From string   `json:"from"`
	To   string   `json:"to"`
	Type EdgeType `json:"type"`
}

// GraphStats contains statistics about the graph.
type GraphStats struct {
	NodeCount int `json:"node_count"`
	EdgeCount int `json:"edge_count"`
	MaxDepth  int `json:"max_depth"`
}

// Graph represents the full dependency graph.
type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Stats GraphStats  `json:"stats"`
}

// NewGraph creates an empty graph.
func NewGraph() Graph {
	return Graph{
		Nodes: []GraphNode{},
		Edges: []GraphEdge{},
	}
}

// AddNode adds a node to the graph.
func (g *Graph) AddNode(node GraphNode) {
	g.Nodes = append(g.Nodes, node)
	g.Stats.NodeCount++
}

// AddEdge adds an edge to the graph.
func (g *Graph) AddEdge(edge GraphEdge) {
	g.Edges = append(g.Edges, edge)
	g.Stats.EdgeCount++
}

// GraphFormat specifies output format for the graph endpoint.
type GraphFormat string

const (
	GraphFormatJSON GraphFormat = "json"
	GraphFormatDOT  GraphFormat = "dot"
)

// ParseEdgeType converts a string to EdgeType.
func ParseEdgeType(s string) EdgeType {
	switch s {
	case "blocks":
		return EdgeTypeBlocks
	case "blocked_by":
		return EdgeTypeBlockedBy
	case "parent":
		return EdgeTypeParent
	case "child":
		return EdgeTypeChild
	case "waits_for":
		return EdgeTypeWaitsFor
	case "waited_by":
		return EdgeTypeWaitedBy
	case "conditional_blocks":
		return EdgeTypeConditional
	case "relates_to":
		return EdgeTypeRelates
	case "duplicates":
		return EdgeTypeDuplicates
	case "mentions":
		return EdgeTypeMentions
	case "derived_from":
		return EdgeTypeDerivedFrom
	case "supersedes":
		return EdgeTypeSupersedes
	case "implements":
		return EdgeTypeImplements
	default:
		return EdgeTypeUnknown
	}
}

// ToDOT exports the graph in Graphviz DOT format.
func (g *Graph) ToDOT() string {
	var b strings.Builder
	b.WriteString("digraph dependencies {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [shape=box, style=rounded];\n\n")

	// Define node styles by status
	statusColors := map[Status]string{
		StatusPending:    "#3b82f6", // blue
		StatusInProgress: "#eab308", // yellow
		StatusDone:       "#22c55e", // green
		StatusBlocked:    "#ef4444", // red
	}

	// Write nodes
	for _, node := range g.Nodes {
		color := statusColors[node.Status]
		if color == "" {
			color = "#6b7280" // gray default
		}
		label := strings.ReplaceAll(node.Title, "\"", "\\\"")
		fmt.Fprintf(&b, "  \"%s\" [label=\"%s\", fillcolor=\"%s\", style=\"filled,rounded\"];\n",
			node.ID, label, color)
	}

	b.WriteString("\n")

	// Define edge styles by type
	edgeStyles := map[EdgeType]string{
		EdgeTypeBlocks:      "color=\"#ef4444\", penwidth=2",   // red, thick
		EdgeTypeBlockedBy:   "color=\"#ef4444\", style=dashed", // red, dashed
		EdgeTypeParent:      "color=\"#6b7280\", style=dashed", // gray, dashed
		EdgeTypeChild:       "color=\"#6b7280\", style=dotted", // gray, dotted
		EdgeTypeWaitsFor:    "color=\"#f97316\", style=dashed", // orange, dashed
		EdgeTypeConditional: "color=\"#a855f7\", style=dashed", // purple, dashed
		EdgeTypeRelates:     "color=\"#3b82f6\", style=dotted", // blue, dotted
		EdgeTypeImplements:  "color=\"#22c55e\", style=bold",   // green, bold
	}

	// Write edges
	for _, edge := range g.Edges {
		style := edgeStyles[edge.Type]
		if style == "" {
			style = "color=\"#9ca3af\""
		}
		fmt.Fprintf(&b, "  \"%s\" -> \"%s\" [%s, label=\"%s\"];\n",
			edge.From, edge.To, style, edge.Type)
	}

	b.WriteString("}\n")
	return b.String()
}

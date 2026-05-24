// Package beads provides integration with the Beads issue tracker via the bd CLI.
package beads

import (
	"context"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/model"
)

// Adapter defines the interface for interacting with Beads.
// All methods shell out to the bd CLI and parse JSON output.
type Adapter interface {
	// ListIssues returns all issues matching the optional filter.
	ListIssues(ctx context.Context, filter model.IssueFilter) ([]model.Issue, error)

	// GetIssue returns a single issue by ID with full details.
	GetIssue(ctx context.Context, id string) (*model.Issue, error)

	// Board returns issues grouped by status for board view.
	Board(ctx context.Context) (*model.Board, error)

	// Graph returns the dependency graph.
	Graph(ctx context.Context) (*model.Graph, error)

	// Memories returns every entry in the bd persistent memory layer, sorted
	// by key. Read-only — there is no write counterpart on this interface
	// because Council Q2 fixed the memories panel as a read-only mirror
	// (see 000-docs/004-AT-DECR-... Q2 architectural invariant). The bd
	// CLI remains the canonical writer.
	Memories(ctx context.Context) (*model.MemoriesResponse, error)

	// Memory recalls a single memory by key. Returns NotFoundError when the
	// key does not exist.
	Memory(ctx context.Context, key string) (*model.Memory, error)

	// SearchMemories returns memories whose content matches the query
	// substring (bd's `bd memories <query> --json` semantics). Same
	// response shape as Memories.
	SearchMemories(ctx context.Context, query string) (*model.MemoriesResponse, error)

	// IsInitialized checks if beads is initialized in the current directory.
	IsInitialized(ctx context.Context) (bool, error)

	// Version returns the bd CLI version.
	Version(ctx context.Context) (string, error)
}

// CLIAdapter implements Adapter by shelling out to the bd CLI.
type CLIAdapter struct {
	executor Executor
	workDir  string
}

// NewCLIAdapter creates a new CLI-based adapter.
// If workDir is empty, uses the current directory.
func NewCLIAdapter(workDir string) *CLIAdapter {
	return &CLIAdapter{
		executor: &DefaultExecutor{},
		workDir:  workDir,
	}
}

// NewCLIAdapterWithExecutor creates an adapter with a custom executor (for testing).
func NewCLIAdapterWithExecutor(workDir string, executor Executor) *CLIAdapter {
	return &CLIAdapter{
		executor: executor,
		workDir:  workDir,
	}
}

// ListIssues implements Adapter.ListIssues.
func (a *CLIAdapter) ListIssues(ctx context.Context, filter model.IssueFilter) ([]model.Issue, error) {
	args := []string{"list", "--json"}

	if filter.Status != "" {
		args = append(args, "--status", filter.Status)
	}

	output, err := a.executor.Execute(ctx, a.workDir, args...)
	if err != nil {
		return nil, err
	}

	bdIssues, err := ParseIssueList(output)
	if err != nil {
		return nil, &ParseError{Command: "list", Err: err}
	}

	issues := make([]model.Issue, 0, len(bdIssues))
	for _, bi := range bdIssues {
		issues = append(issues, bi.ToModelIssue())
	}

	return issues, nil
}

// GetIssue implements Adapter.GetIssue.
func (a *CLIAdapter) GetIssue(ctx context.Context, id string) (*model.Issue, error) {
	output, err := a.executor.Execute(ctx, a.workDir, "show", id, "--json")
	if err != nil {
		if IsNotFoundError(err) {
			return nil, &NotFoundError{ID: id}
		}
		return nil, err
	}

	bdIssues, err := ParseIssueList(output)
	if err != nil {
		return nil, &ParseError{Command: "show", Err: err}
	}

	if len(bdIssues) == 0 {
		return nil, &NotFoundError{ID: id}
	}

	issue := bdIssues[0].ToModelIssue()
	return &issue, nil
}

// Board implements Adapter.Board.
func (a *CLIAdapter) Board(ctx context.Context) (*model.Board, error) {
	issues, err := a.ListIssues(ctx, model.NewIssueFilter())
	if err != nil {
		return nil, err
	}

	board := model.NewBoard()
	for _, issue := range issues {
		board.AddIssue(model.IssueSummary{
			ID:       issue.ID,
			Title:    issue.Title,
			Status:   issue.Status,
			Priority: issue.Priority,
		})
	}

	return &board, nil
}

// Graph implements Adapter.Graph.
func (a *CLIAdapter) Graph(ctx context.Context) (*model.Graph, error) {
	// Get all issues with full details for nodes and relationships
	output, err := a.executor.Execute(ctx, a.workDir, "list", "--json")
	if err != nil {
		return nil, err
	}

	bdIssues, err := ParseIssueList(output)
	if err != nil {
		return nil, &ParseError{Command: "list", Err: err}
	}

	graph := model.NewGraph()
	nodeMap := make(map[string]bool)
	edgeSet := make(map[string]bool) // prevent duplicate edges

	// Add all issues as nodes
	for _, bi := range bdIssues {
		graph.AddNode(model.GraphNode{
			ID:       bi.ID,
			Title:    bi.Title,
			Status:   mapStatus(bi.Status),
			Priority: mapPriority(bi.Priority),
		})
		nodeMap[bi.ID] = true
	}

	// Extract edges from issue relationships
	for _, bi := range bdIssues {
		// Process dependencies (things this issue depends on)
		for _, dep := range bi.Dependencies {
			edgeType := mapDepTypeToEdgeType(dep.DepType)
			edgeKey := bi.ID + "->" + dep.ID + ":" + string(edgeType)
			if nodeMap[dep.ID] && !edgeSet[edgeKey] {
				graph.AddEdge(model.GraphEdge{
					From: dep.ID,
					To:   bi.ID,
					Type: edgeType,
				})
				edgeSet[edgeKey] = true
			}
		}

		// Process dependents (things that depend on this issue)
		for _, dep := range bi.Dependents {
			edgeType := mapDepTypeToEdgeType(dep.DepType)
			edgeKey := dep.ID + "->" + bi.ID + ":" + string(edgeType)
			if nodeMap[dep.ID] && !edgeSet[edgeKey] {
				graph.AddEdge(model.GraphEdge{
					From: bi.ID,
					To:   dep.ID,
					Type: edgeType,
				})
				edgeSet[edgeKey] = true
			}
		}

		// Process blocked_by array for explicit blocking relationships
		for _, blockerID := range bi.BlockedBy {
			edgeKey := blockerID + "->" + bi.ID + ":blocks"
			if nodeMap[blockerID] && !edgeSet[edgeKey] {
				graph.AddEdge(model.GraphEdge{
					From: blockerID,
					To:   bi.ID,
					Type: model.EdgeTypeBlocks,
				})
				edgeSet[edgeKey] = true
			}
		}
	}

	// Also get blocked info for any additional edges
	blockedOutput, err := a.executor.Execute(ctx, a.workDir, "blocked", "--json")
	if err == nil {
		blockedIssues, parseErr := ParseBlockedList(blockedOutput)
		if parseErr == nil {
			for _, bi := range blockedIssues {
				for _, blockerID := range bi.BlockedBy {
					edgeKey := blockerID + "->" + bi.ID + ":blocks"
					if nodeMap[blockerID] && nodeMap[bi.ID] && !edgeSet[edgeKey] {
						graph.AddEdge(model.GraphEdge{
							From: blockerID,
							To:   bi.ID,
							Type: model.EdgeTypeBlocks,
						})
						edgeSet[edgeKey] = true
					}
				}
			}
		}
	}

	return &graph, nil
}

// mapDepTypeToEdgeType converts bd dependency_type to model.EdgeType.
func mapDepTypeToEdgeType(depType string) model.EdgeType {
	switch depType {
	case "blocks":
		return model.EdgeTypeBlocks
	case "blocked_by":
		return model.EdgeTypeBlockedBy
	case "parent-child", "parent":
		return model.EdgeTypeParent
	case "child":
		return model.EdgeTypeChild
	case "waits_for":
		return model.EdgeTypeWaitsFor
	case "waited_by":
		return model.EdgeTypeWaitedBy
	case "conditional_blocks":
		return model.EdgeTypeConditional
	case "relates_to":
		return model.EdgeTypeRelates
	case "duplicates":
		return model.EdgeTypeDuplicates
	case "mentions":
		return model.EdgeTypeMentions
	case "derived_from":
		return model.EdgeTypeDerivedFrom
	case "supersedes":
		return model.EdgeTypeSupersedes
	case "implements":
		return model.EdgeTypeImplements
	default:
		return model.EdgeTypeBlocks // default to blocks for unknown
	}
}

// IsInitialized implements Adapter.IsInitialized.
func (a *CLIAdapter) IsInitialized(ctx context.Context) (bool, error) {
	_, err := a.executor.Execute(ctx, a.workDir, "status")
	if err != nil {
		if IsNotInitializedError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Version implements Adapter.Version.
func (a *CLIAdapter) Version(ctx context.Context) (string, error) {
	output, err := a.executor.Execute(ctx, a.workDir, "--version")
	if err != nil {
		return "", err
	}
	return ParseVersion(output), nil
}

// Memories implements Adapter.Memories by shelling `bd memories --json`.
// Returns a non-nil response with an empty Memories slice when no memories
// exist (bd 1.0.4 emits `{"schema_version": 1}` in that case).
func (a *CLIAdapter) Memories(ctx context.Context) (*model.MemoriesResponse, error) {
	output, err := a.executor.Execute(ctx, a.workDir, "memories", "--json")
	if err != nil {
		return nil, err
	}
	memories, schemaVersion, err := ParseMemories(output)
	if err != nil {
		return nil, &ParseError{Command: "memories", Err: err}
	}
	return &model.MemoriesResponse{
		Memories:      memories,
		Count:         len(memories),
		SchemaVersion: schemaVersion,
	}, nil
}

// Memory implements Adapter.Memory by shelling `bd memories <key> --json`
// then picking the matching entry. The `bd recall` subcommand emits raw
// text, not JSON, so it isn't useful as a single-entry retrieval path;
// the search variant + key filter is the cleanest read against the
// bd 1.0.4 surface.
func (a *CLIAdapter) Memory(ctx context.Context, key string) (*model.Memory, error) {
	if key == "" {
		return nil, &NotFoundError{ID: key}
	}
	output, err := a.executor.Execute(ctx, a.workDir, "memories", key, "--json")
	if err != nil {
		return nil, err
	}
	memories, _, err := ParseMemories(output)
	if err != nil {
		return nil, &ParseError{Command: "memories", Err: err}
	}
	for i := range memories {
		if memories[i].Key == key {
			m := memories[i]
			return &m, nil
		}
	}
	return nil, &NotFoundError{ID: key}
}

// SearchMemories implements Adapter.SearchMemories by shelling
// `bd memories <query> --json`. Same response shape as Memories.
func (a *CLIAdapter) SearchMemories(ctx context.Context, query string) (*model.MemoriesResponse, error) {
	args := []string{"memories", "--json"}
	if query != "" {
		args = []string{"memories", query, "--json"}
	}
	output, err := a.executor.Execute(ctx, a.workDir, args...)
	if err != nil {
		return nil, err
	}
	memories, schemaVersion, err := ParseMemories(output)
	if err != nil {
		return nil, &ParseError{Command: "memories", Err: err}
	}
	return &model.MemoriesResponse{
		Memories:      memories,
		Count:         len(memories),
		SchemaVersion: schemaVersion,
	}, nil
}

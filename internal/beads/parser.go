package beads

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/model"
)

// BDIssue represents an issue as returned by bd --json.
type BDIssue struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    int        `json:"priority"`
	IssueType   string     `json:"issue_type"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	// DeferUntil is the wake-up timestamp set by `bd defer --until`. Only present
	// when Status == "deferred"; absent (nil) for any other status. Pre-1.0 versions
	// of the viewer dropped this field on the floor and remapped deferred → pending,
	// losing the until-date — fixed in gastown-7fq.
	DeferUntil      *time.Time `json:"defer_until,omitempty"`
	DependencyCount int        `json:"dependency_count,omitempty"`
	DependentCount  int        `json:"dependent_count,omitempty"`
	Dependencies    []BDIssue  `json:"dependencies,omitempty"`
	Dependents      []BDIssue  `json:"dependents,omitempty"`
	BlockedBy       []string   `json:"blocked_by,omitempty"`
	BlockedByCount  int        `json:"blocked_by_count,omitempty"`
	DepType         string     `json:"dependency_type,omitempty"`
}

// ToModelIssue converts a BDIssue to the domain model Issue.
func (bi *BDIssue) ToModelIssue() model.Issue {
	issue := model.Issue{
		ID:          bi.ID,
		Title:       bi.Title,
		Description: bi.Description,
		Status:      mapStatus(bi.Status),
		Priority:    mapPriority(bi.Priority),
		CreatedAt:   bi.CreatedAt,
		UpdatedAt:   bi.UpdatedAt,
		Children:    []model.IssueSummary{},
		Blocks:      []model.IssueSummary{},
		BlockedBy:   []model.IssueSummary{},
	}

	// Parse "Done when:" from description
	issue.DoneWhen = parseDoneWhen(bi.Description)

	// Preserve the defer-until timestamp when the issue is in the deferred state.
	// We only attach DeferredUntil when the status is actually deferred so older
	// updates (when the until-date was set then later cleared by un-deferring)
	// do not leak a stale timestamp into a now-pending issue.
	if issue.Status == model.StatusDeferred && bi.DeferUntil != nil {
		t := *bi.DeferUntil
		issue.DeferredUntil = &t
	}

	// Map dependencies to BlockedBy (things this issue depends on)
	for _, dep := range bi.Dependencies {
		if dep.DepType == "blocks" {
			issue.BlockedBy = append(issue.BlockedBy, model.IssueSummary{
				ID:       dep.ID,
				Title:    dep.Title,
				Status:   mapStatus(dep.Status),
				Priority: mapPriority(dep.Priority),
			})
		} else if dep.DepType == "parent-child" {
			issue.Parent = &model.IssueSummary{
				ID:       dep.ID,
				Title:    dep.Title,
				Status:   mapStatus(dep.Status),
				Priority: mapPriority(dep.Priority),
			}
		}
	}

	// Map dependents to Blocks (things that depend on this issue)
	for _, dep := range bi.Dependents {
		if dep.DepType == "blocks" {
			issue.Blocks = append(issue.Blocks, model.IssueSummary{
				ID:       dep.ID,
				Title:    dep.Title,
				Status:   mapStatus(dep.Status),
				Priority: mapPriority(dep.Priority),
			})
		} else if dep.DepType == "parent-child" {
			issue.Children = append(issue.Children, model.IssueSummary{
				ID:       dep.ID,
				Title:    dep.Title,
				Status:   mapStatus(dep.Status),
				Priority: mapPriority(dep.Priority),
			})
		}
	}

	return issue
}

// ToSummary converts a BDIssue to an IssueSummary.
func (bi *BDIssue) ToSummary() model.IssueSummary {
	return model.IssueSummary{
		ID:       bi.ID,
		Title:    bi.Title,
		Status:   mapStatus(bi.Status),
		Priority: mapPriority(bi.Priority),
	}
}

// mapStatus converts bd status string to model.Status.
func mapStatus(s string) model.Status {
	switch strings.ToLower(s) {
	case "open", "pending":
		return model.StatusPending
	case "in_progress", "in-progress", "inprogress":
		return model.StatusInProgress
	case "closed", "done", "complete":
		return model.StatusDone
	case "blocked":
		return model.StatusBlocked
	case "deferred":
		return model.StatusDeferred
	default:
		return model.StatusPending
	}
}

// mapPriority converts bd priority int to model.Priority.
func mapPriority(p int) model.Priority {
	switch p {
	case 1:
		return model.PriorityHigh
	case 2:
		return model.PriorityMedium
	case 3:
		return model.PriorityLow
	default:
		return model.PriorityMedium
	}
}

// parseDoneWhen extracts "Done when:" bullets from description.
func parseDoneWhen(description string) []string {
	var items []string

	lines := strings.Split(description, "\n")
	inDoneWhen := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(strings.ToLower(line), "done when:") {
			inDoneWhen = true
			continue
		}

		if inDoneWhen {
			// Empty line ends the section
			if line == "" {
				break
			}
			// Lines starting with - are items
			if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
				items = append(items, strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* "))
			}
		}
	}

	return items
}

// ParseIssueList parses JSON output from bd list or bd show.
func ParseIssueList(data []byte) ([]BDIssue, error) {
	if len(data) == 0 {
		return []BDIssue{}, nil
	}

	var issues []BDIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, err
	}

	return issues, nil
}

// BDBlockedIssue represents an issue from bd blocked --json.
type BDBlockedIssue struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	BlockedByCount int      `json:"blocked_by_count"`
	BlockedBy      []string `json:"blocked_by"`
}

// ParseBlockedList parses JSON output from bd blocked.
func ParseBlockedList(data []byte) ([]BDBlockedIssue, error) {
	if len(data) == 0 {
		return []BDBlockedIssue{}, nil
	}

	var issues []BDBlockedIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, err
	}

	return issues, nil
}

// ParseVersion extracts version number from bd --version output.
func ParseVersion(data []byte) string {
	s := strings.TrimSpace(string(data))
	// Output is like "bd version 0.29.0 (dev)"
	parts := strings.Fields(s)
	for i, p := range parts {
		if p == "version" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	// Fallback: return trimmed output
	return s
}

// rawDoltStatus mirrors the JSON shape returned by `bd dolt status --json`.
// Field set captured from live bd 1.0.4 output 2026-05-24.
//
// The Error field captures the alternate shape bd emits in a non-beads dir:
// `{"error": "no active beads workspace found", "hint": "...", "schema_version": 1}`.
// Presence of a non-empty Error means the rest of the fields are unreliable
// and the adapter should report Health=unknown.
type rawDoltStatus struct {
	DataDir       string `json:"data_dir,omitempty"`
	PID           int    `json:"pid,omitempty"`
	Port          int    `json:"port,omitempty"`
	Running       bool   `json:"running"`
	SchemaVersion int    `json:"schema_version,omitempty"`
	Error         string `json:"error,omitempty"`
}

// rawDoltRemote mirrors one element of `bd dolt remote list --json`.
type rawDoltRemote struct {
	Name   string `json:"name"`
	SQLURL string `json:"sql_url,omitempty"`
	CLIURL string `json:"cli_url,omitempty"`
	Status string `json:"status,omitempty"`
}

// ParseDoltStatus turns `bd dolt status --json` output into a partially
// filled DoltSyncState. The Health field is NOT computed here — that's the
// adapter's job because health composes status + remote-list, and the
// parser must stay orthogonal to that composition.
//
// If bd emitted the error-JSON shape (top-level "error" field), the
// returned state carries that string in Error and the adapter will map
// it to Health=unknown.
func ParseDoltStatus(data []byte) (*model.DoltSyncState, error) {
	var raw rawDoltStatus
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return &model.DoltSyncState{
		Running:       raw.Running,
		Port:          raw.Port,
		SchemaVersion: raw.SchemaVersion,
		Error:         raw.Error,
		Remotes:       []model.DoltRemote{}, // adapter fills this
	}, nil
}

// ParseDoltRemotes turns `bd dolt remote list --json` output into the
// minimal []model.DoltRemote the wire response surfaces. Full URLs are
// intentionally NOT carried into the model — they would leak the user's
// DoltHub workspace path into any screen-recording that captures the
// dashboard. Malformed input returns an empty slice rather than an error
// so the caller's health-derivation path still functions on a green server
// with unparseable remote output.
func ParseDoltRemotes(data []byte) []model.DoltRemote {
	if len(data) == 0 || string(data) == "null" {
		return []model.DoltRemote{}
	}
	var raws []rawDoltRemote
	if err := json.Unmarshal(data, &raws); err != nil {
		return []model.DoltRemote{}
	}
	out := make([]model.DoltRemote, 0, len(raws))
	for _, r := range raws {
		out = append(out, model.DoltRemote{Name: r.Name, Status: r.Status})
	}
	return out
}

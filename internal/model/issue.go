// Package model defines the core domain types for Gastown Viewer Intent.
package model

import "time"

// Status represents the state of an issue.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusBlocked    Status = "blocked"
	// StatusDeferred is the time-windowed postpone state from `bd defer --until`.
	// Previously this mapped lossily to StatusPending, which dropped the until-date
	// on the floor — see Issue.DeferredUntil for the preserved timestamp.
	StatusDeferred Status = "deferred"
)

// Priority represents issue priority level.
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// IssueSummary is a compact representation for lists and references.
type IssueSummary struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Status   Status   `json:"status"`
	Priority Priority `json:"priority"`
}

// Issue is the full representation of a Beads issue.
type Issue struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Status      Status         `json:"status"`
	Priority    Priority       `json:"priority"`
	Parent      *IssueSummary  `json:"parent,omitempty"`
	Children    []IssueSummary `json:"children"`
	Blocks      []IssueSummary `json:"blocks"`
	BlockedBy   []IssueSummary `json:"blocked_by"`
	DoneWhen    []string       `json:"done_when,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	// DeferredUntil is the wake-up timestamp when Status == StatusDeferred.
	// Nil for any other status. Populated from bd's `defer_until` field.
	DeferredUntil *time.Time `json:"deferred_until,omitempty"`
}

// IssueListResponse is the response for GET /api/v1/issues.
type IssueListResponse struct {
	Issues []Issue `json:"issues"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// IssueFilter defines query parameters for listing issues.
type IssueFilter struct {
	Status string
	Parent string
	Search string
	Limit  int
	Offset int
}

// NewIssueFilter returns a filter with default values.
func NewIssueFilter() IssueFilter {
	return IssueFilter{
		Limit:  100,
		Offset: 0,
	}
}

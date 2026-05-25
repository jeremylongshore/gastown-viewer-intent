package beads

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/model"
)

func TestParseIssueList(t *testing.T) {
	input := []byte(`[
		{
			"id": "test-1",
			"title": "Test Issue",
			"description": "Test description\n\nDone when:\n- First item\n- Second item",
			"status": "open",
			"priority": 1,
			"issue_type": "task",
			"created_at": "2026-01-01T10:00:00Z",
			"updated_at": "2026-01-01T12:00:00Z"
		}
	]`)

	issues, err := ParseIssueList(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if issue.ID != "test-1" {
		t.Errorf("expected ID 'test-1', got '%s'", issue.ID)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("expected title 'Test Issue', got '%s'", issue.Title)
	}
	if issue.Priority != 1 {
		t.Errorf("expected priority 1, got %d", issue.Priority)
	}
}

func TestMapStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected model.Status
	}{
		{"open", model.StatusPending},
		{"pending", model.StatusPending},
		{"in_progress", model.StatusInProgress},
		{"in-progress", model.StatusInProgress},
		{"closed", model.StatusDone},
		{"done", model.StatusDone},
		{"blocked", model.StatusBlocked},
		{"deferred", model.StatusDeferred},
		{"DEFERRED", model.StatusDeferred}, // case-insensitive
		{"unknown", model.StatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapStatus(tt.input)
			if result != tt.expected {
				t.Errorf("mapStatus(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestDeferredUntilPreserved is the regression test for the parser.go bug where
// `bd defer <id> --until <when>` had its until-date silently dropped because
// the "deferred" status fell through mapStatus's default branch and remapped to
// "pending", and there was no DeferredUntil field on the model to carry the
// date even if the status had been preserved. Council decision Q0 + Q3 binding
// constraint (gastown-cr5 AT-DECR 2026-05-23). When this test fails, the
// dashboard is lying to the user about what work is actually open versus what
// is parked until a specific date.
func TestDeferredUntilPreserved(t *testing.T) {
	// Real bd 1.0.4 JSON shape captured 2026-05-23 from
	// `bd show gastown-rj5 --json` after `bd defer gastown-rj5 --until tomorrow`.
	input := []byte(`[
		{
			"id": "gastown-rj5",
			"title": "Janitorial sweep",
			"description": "",
			"status": "deferred",
			"priority": 2,
			"issue_type": "task",
			"created_at": "2026-05-24T04:07:03Z",
			"updated_at": "2026-05-24T04:35:24Z",
			"defer_until": "2026-05-25T04:35:24Z"
		}
	]`)
	issues, err := ParseIssueList(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	bd := issues[0]
	if bd.Status != "deferred" {
		t.Errorf("BDIssue.Status: expected %q, got %q", "deferred", bd.Status)
	}
	if bd.DeferUntil == nil {
		t.Fatalf("BDIssue.DeferUntil: expected non-nil for deferred issue, got nil — this is the parser.go:112 regression")
	}
	wantTS, _ := time.Parse(time.RFC3339, "2026-05-25T04:35:24Z")
	if !bd.DeferUntil.Equal(wantTS) {
		t.Errorf("BDIssue.DeferUntil: expected %v, got %v", wantTS, *bd.DeferUntil)
	}

	m := bd.ToModelIssue()
	if m.Status != model.StatusDeferred {
		t.Errorf("model.Issue.Status: expected %q, got %q — deferred must NOT remap to pending",
			model.StatusDeferred, m.Status)
	}
	if m.DeferredUntil == nil {
		t.Fatalf("model.Issue.DeferredUntil: expected non-nil — the until-date is being dropped on the floor again")
	}
	if !m.DeferredUntil.Equal(wantTS) {
		t.Errorf("model.Issue.DeferredUntil: expected %v, got %v", wantTS, *m.DeferredUntil)
	}
}

// TestDeferredUntilOnlyAttachedWhenDeferred guards against a stale until-date
// leaking into a non-deferred issue. When `bd update <id> --status open` is run
// against a previously-deferred issue, bd may keep emitting the old defer_until
// field in its JSON for a tick; the viewer must not present that as if the
// issue were still parked.
func TestDeferredUntilOnlyAttachedWhenDeferred(t *testing.T) {
	staleTS := time.Date(2026, 5, 25, 4, 35, 24, 0, time.UTC)
	bd := BDIssue{
		ID:         "gastown-rj5",
		Title:      "Janitorial sweep",
		Status:     "open", // un-deferred but defer_until still emitted
		DeferUntil: &staleTS,
	}
	m := bd.ToModelIssue()
	if m.Status != model.StatusPending {
		t.Errorf("expected status pending for un-deferred issue, got %q", m.Status)
	}
	if m.DeferredUntil != nil {
		t.Errorf("DeferredUntil must not leak onto a non-deferred issue; got %v", *m.DeferredUntil)
	}
}

func TestMapPriority(t *testing.T) {
	tests := []struct {
		input    int
		expected model.Priority
	}{
		{1, model.PriorityHigh},
		{2, model.PriorityMedium},
		{3, model.PriorityLow},
		{0, model.PriorityMedium},
		{99, model.PriorityMedium},
	}

	for _, tt := range tests {
		result := mapPriority(tt.input)
		if result != tt.expected {
			t.Errorf("mapPriority(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestParseDoneWhen(t *testing.T) {
	description := `Implement the feature.

Done when:
- First criterion
- Second criterion
- Third criterion

Additional notes here.`

	items := parseDoneWhen(description)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	expected := []string{"First criterion", "Second criterion", "Third criterion"}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("item %d: expected %q, got %q", i, expected[i], item)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"bd version 0.29.0 (dev)\n", "0.29.0"},
		{"bd version 1.0.0\n", "1.0.0"},
		{"0.42.0", "0.42.0"},
	}

	for _, tt := range tests {
		result := ParseVersion([]byte(tt.input))
		if result != tt.expected {
			t.Errorf("ParseVersion(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestBDIssueToModelIssue(t *testing.T) {
	bdIssue := BDIssue{
		ID:          "test-1",
		Title:       "Test Issue",
		Description: "Done when:\n- Item 1\n- Item 2",
		Status:      "in_progress",
		Priority:    1,
		Dependencies: []BDIssue{
			{ID: "dep-1", Title: "Dependency", Status: "closed", Priority: 2, DepType: "blocks"},
		},
		Dependents: []BDIssue{
			{ID: "dep-2", Title: "Dependent", Status: "open", Priority: 2, DepType: "blocks"},
		},
	}

	issue := bdIssue.ToModelIssue()

	if issue.ID != "test-1" {
		t.Errorf("expected ID 'test-1', got '%s'", issue.ID)
	}
	if issue.Status != model.StatusInProgress {
		t.Errorf("expected status in_progress, got %s", issue.Status)
	}
	if issue.Priority != model.PriorityHigh {
		t.Errorf("expected priority high, got %s", issue.Priority)
	}
	if len(issue.DoneWhen) != 2 {
		t.Errorf("expected 2 done_when items, got %d", len(issue.DoneWhen))
	}
	if len(issue.BlockedBy) != 1 {
		t.Errorf("expected 1 blocked_by, got %d", len(issue.BlockedBy))
	}
	if len(issue.Blocks) != 1 {
		t.Errorf("expected 1 blocks, got %d", len(issue.Blocks))
	}
}

func TestParseEmptyList(t *testing.T) {
	issues, err := ParseIssueList([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected empty list, got %d issues", len(issues))
	}
}

func TestParseDoltStatus(t *testing.T) {
	// JSON shape captured from live bd 1.0.4 2026-05-24.
	input := []byte(`{
		"data_dir": "/home/jeremy/.beads/dolt",
		"pid": 1247309,
		"port": 45435,
		"running": true,
		"schema_version": 1
	}`)
	st, err := ParseDoltStatus(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st.Running {
		t.Error("Running: expected true")
	}
	if st.Port != 45435 {
		t.Errorf("Port: got %d, want 45435", st.Port)
	}
	if st.SchemaVersion != 1 {
		t.Errorf("SchemaVersion: got %d, want 1", st.SchemaVersion)
	}
	// Remotes is initialized non-nil so the JSON encoder emits [] not null.
	if st.Remotes == nil {
		t.Error("Remotes: expected non-nil empty slice")
	}
}

func TestParseDoltStatus_NotRunning(t *testing.T) {
	input := []byte(`{"running": false}`)
	st, err := ParseDoltStatus(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Running {
		t.Error("Running: expected false")
	}
}

func TestParseDoltStatus_Malformed(t *testing.T) {
	if _, err := ParseDoltStatus([]byte(`not json`)); err == nil {
		t.Error("expected error on malformed JSON")
	}
}

func TestParseDoltRemotes(t *testing.T) {
	input := []byte(`[
		{"name": "origin", "sql_url": "https://example/sql", "cli_url": "https://example/cli", "status": "ok"},
		{"name": "backup", "status": "auth_failed"}
	]`)
	got := ParseDoltRemotes(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 remotes, got %d", len(got))
	}
	if got[0].Name != "origin" || got[0].Status != "ok" {
		t.Errorf("remote[0]: %+v", got[0])
	}
	if got[1].Name != "backup" || got[1].Status != "auth_failed" {
		t.Errorf("remote[1]: %+v", got[1])
	}
	// Sanity: URL fields are stripped — they leak the workspace name into
	// screen-recordings, so the wire response should never carry them.
	gotJSON, _ := json.Marshal(got)
	for _, banned := range []string{"sql_url", "cli_url", "https://"} {
		if bytesContains(gotJSON, banned) {
			t.Errorf("remote JSON should not include %q, got %s", banned, gotJSON)
		}
	}
}

func TestParseDoltRemotes_Null(t *testing.T) {
	// bd emits the literal "null" when no remotes are configured. Must
	// degrade to empty slice rather than nil-deref or returning null.
	got := ParseDoltRemotes([]byte(`null`))
	if got == nil || len(got) != 0 {
		t.Errorf("expected non-nil empty slice for null input, got %v", got)
	}
}

func TestParseDoltRemotes_Malformed(t *testing.T) {
	got := ParseDoltRemotes([]byte(`not json`))
	if got == nil || len(got) != 0 {
		t.Errorf("expected non-nil empty slice on malformed JSON, got %v", got)
	}
}

func bytesContains(haystack []byte, needle string) bool {
	return strings.Contains(string(haystack), needle)
}

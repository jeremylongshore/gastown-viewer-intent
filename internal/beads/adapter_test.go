package beads

import (
	"context"
	"testing"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/model"
)

func TestCLIAdapterListIssues(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("list --json", []byte(`[
		{"id": "test-1", "title": "Issue 1", "status": "open", "priority": 1},
		{"id": "test-2", "title": "Issue 2", "status": "in_progress", "priority": 2}
	]`))

	adapter := NewCLIAdapterWithExecutor("", mock)
	ctx := context.Background()

	issues, err := adapter.ListIssues(ctx, model.NewIssueFilter())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}

	if issues[0].ID != "test-1" {
		t.Errorf("expected first issue ID 'test-1', got '%s'", issues[0].ID)
	}
	if issues[1].Status != model.StatusInProgress {
		t.Errorf("expected second issue status in_progress, got %s", issues[1].Status)
	}
}

func TestCLIAdapterGetIssue(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("show test-1 --json", []byte(`[
		{
			"id": "test-1",
			"title": "Test Issue",
			"description": "Description here",
			"status": "open",
			"priority": 1
		}
	]`))

	adapter := NewCLIAdapterWithExecutor("", mock)
	ctx := context.Background()

	issue, err := adapter.GetIssue(ctx, "test-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if issue.ID != "test-1" {
		t.Errorf("expected ID 'test-1', got '%s'", issue.ID)
	}
	if issue.Description != "Description here" {
		t.Errorf("expected description 'Description here', got '%s'", issue.Description)
	}
}

func TestCLIAdapterGetIssueNotFound(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetError("show nonexistent --json", &NotFoundError{ID: "nonexistent"})

	adapter := NewCLIAdapterWithExecutor("", mock)
	ctx := context.Background()

	_, err := adapter.GetIssue(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestCLIAdapterBoard(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("list --json", []byte(`[
		{"id": "test-1", "title": "Issue 1", "status": "open", "priority": 1},
		{"id": "test-2", "title": "Issue 2", "status": "in_progress", "priority": 2},
		{"id": "test-3", "title": "Issue 3", "status": "closed", "priority": 2}
	]`))

	adapter := NewCLIAdapterWithExecutor("", mock)
	ctx := context.Background()

	board, err := adapter.Board(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if board.Total != 3 {
		t.Errorf("expected total 3, got %d", board.Total)
	}

	// Check columns
	var pending, inProgress, done int
	for _, col := range board.Columns {
		switch col.Status {
		case model.StatusPending:
			pending = col.Count
		case model.StatusInProgress:
			inProgress = col.Count
		case model.StatusDone:
			done = col.Count
		}
	}

	if pending != 1 {
		t.Errorf("expected 1 pending, got %d", pending)
	}
	if inProgress != 1 {
		t.Errorf("expected 1 in_progress, got %d", inProgress)
	}
	if done != 1 {
		t.Errorf("expected 1 done, got %d", done)
	}
}

func TestCLIAdapterVersion(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("--version", []byte("bd version 0.29.0 (dev)\n"))

	adapter := NewCLIAdapterWithExecutor("", mock)
	ctx := context.Background()

	version, err := adapter.Version(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if version != "0.29.0" {
		t.Errorf("expected version '0.29.0', got '%s'", version)
	}
}

func TestCLIAdapterIsInitialized(t *testing.T) {
	t.Run("initialized", func(t *testing.T) {
		mock := NewMockExecutor()
		mock.SetResponse("status", []byte("OK"))

		adapter := NewCLIAdapterWithExecutor("", mock)
		ctx := context.Background()

		ok, err := adapter.IsInitialized(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected true, got false")
		}
	})

	t.Run("not initialized", func(t *testing.T) {
		mock := NewMockExecutor()
		mock.SetError("status", &NotInitializedError{})

		adapter := NewCLIAdapterWithExecutor("", mock)
		ctx := context.Background()

		ok, err := adapter.IsInitialized(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected false, got true")
		}
	})
}

func TestCLIAdapterBDNotFound(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetError("list --json", &BDNotFoundError{})

	adapter := NewCLIAdapterWithExecutor("", mock)
	ctx := context.Background()

	_, err := adapter.ListIssues(ctx, model.NewIssueFilter())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsBDNotFoundError(err) {
		t.Errorf("expected BDNotFoundError, got %T: %v", err, err)
	}
}

// TestDoltSyncState_GreenServerAllRemotesOk is the happy-path canary for the
// header sync pill: dolt server running + every remote reports "ok" →
// health=green. JSON shapes captured from real bd 1.0.4 output 2026-05-24.
func TestDoltSyncState_GreenServerAllRemotesOk(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("dolt status --json", []byte(`{"running": true, "port": 45435, "schema_version": 1}`))
	mock.SetResponse("dolt remote list --json",
		[]byte(`[{"name":"origin","status":"ok"},{"name":"backup","status":"ok"}]`))
	adapter := NewCLIAdapterWithExecutor("", mock)

	state, err := adapter.DoltSyncState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Health != model.DoltHealthGreen {
		t.Errorf("Health: got %q, want green", state.Health)
	}
	if !state.Running {
		t.Error("Running: expected true")
	}
	if len(state.Remotes) != 2 {
		t.Errorf("Remotes: expected 2, got %d", len(state.Remotes))
	}
}

func TestDoltSyncState_YellowOnDegradedRemote(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("dolt status --json", []byte(`{"running": true, "port": 45435}`))
	mock.SetResponse("dolt remote list --json",
		[]byte(`[{"name":"origin","status":"auth_failed"}]`))
	adapter := NewCLIAdapterWithExecutor("", mock)

	state, err := adapter.DoltSyncState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Health != model.DoltHealthYellow {
		t.Errorf("Health: got %q, want yellow", state.Health)
	}
}

func TestDoltSyncState_RedOnServerDown(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("dolt status --json", []byte(`{"running": false}`))
	mock.SetResponse("dolt remote list --json", []byte(`[]`))
	adapter := NewCLIAdapterWithExecutor("", mock)

	state, err := adapter.DoltSyncState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Health != model.DoltHealthRed {
		t.Errorf("Health: got %q, want red", state.Health)
	}
}

func TestDoltSyncState_UnknownOnBDMissing(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetError("dolt status --json", &BDNotFoundError{})
	adapter := NewCLIAdapterWithExecutor("", mock)

	state, err := adapter.DoltSyncState(context.Background())
	if err != nil {
		t.Fatalf("DoltSyncState should not propagate BDNotFound, got: %v", err)
	}
	if state.Health != model.DoltHealthUnknown {
		t.Errorf("Health: got %q, want unknown", state.Health)
	}
	if state.Error == "" {
		t.Error("Error: expected non-empty string carrying the underlying bd error")
	}
}

func TestDoltSyncState_RemoteListFailureDegradesGracefully(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("dolt status --json", []byte(`{"running": true}`))
	mock.SetError("dolt remote list --json", &ExecutionError{Command: "dolt remote list", Stderr: "boom"})
	adapter := NewCLIAdapterWithExecutor("", mock)

	state, err := adapter.DoltSyncState(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Health != model.DoltHealthGreen {
		t.Errorf("Health: got %q, want green (server up, no remotes known)", state.Health)
	}
	if len(state.Remotes) != 0 {
		t.Errorf("Remotes: expected empty on list-error, got %d", len(state.Remotes))
	}
}

func TestHumanFlags_EmptyOnNullOutput(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("human list --json", []byte(`null`))
	adapter := NewCLIAdapterWithExecutor("", mock)

	flags, err := adapter.HumanFlags(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flags == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(flags) != 0 {
		t.Errorf("expected empty slice, got %d issues", len(flags))
	}
}

func TestHumanFlags_ParsesIssues(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("human list --json", []byte(`[
		{"id":"x-1","title":"needs human","status":"open","priority":1,"issue_type":"task",
		 "created_at":"2026-05-24T00:00:00Z","updated_at":"2026-05-24T00:00:00Z"}
	]`))
	adapter := NewCLIAdapterWithExecutor("", mock)

	flags, err := adapter.HumanFlags(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flags) != 1 {
		t.Fatalf("expected 1 flag, got %d", len(flags))
	}
	if flags[0].ID != "x-1" {
		t.Errorf("ID: got %q, want x-1", flags[0].ID)
	}
	if flags[0].Status != model.StatusPending {
		t.Errorf("Status: got %q, want pending (mapped from 'open')", flags[0].Status)
	}
}

func TestHumanFlags_EmptyOnEmptyOutput(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetResponse("human list --json", []byte(`   `))
	adapter := NewCLIAdapterWithExecutor("", mock)

	flags, err := adapter.HumanFlags(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flags) != 0 {
		t.Errorf("expected empty slice on whitespace-only output, got %d", len(flags))
	}
}

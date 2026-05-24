package gastown

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFSAdapter_Status_NoTown(t *testing.T) {
	// Use a non-existent path
	adapter := NewFSAdapter("/tmp/nonexistent-gastown-test")

	status, err := adapter.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() returned error: %v", err)
	}

	if status.Healthy {
		t.Error("Expected Healthy=false for non-existent town")
	}

	if status.Error == "" {
		t.Error("Expected Error message for non-existent town")
	}
}

func TestFSAdapter_Status_WithTown(t *testing.T) {
	// Create a temporary town structure
	tmpDir := t.TempDir()

	// Create minimal town structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "mayor"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "mayor", "town.json"), []byte(`{"name":"test-town"}`), 0644); err != nil {
		t.Fatal(err)
	}

	adapter := NewFSAdapter(tmpDir)

	status, err := adapter.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() returned error: %v", err)
	}

	if !status.Healthy {
		t.Errorf("Expected Healthy=true, got false. Error: %s", status.Error)
	}

	if status.TownRoot != tmpDir {
		t.Errorf("Expected TownRoot=%s, got %s", tmpDir, status.TownRoot)
	}
}

func TestFSAdapter_Town_NoTown(t *testing.T) {
	adapter := NewFSAdapter("/tmp/nonexistent-gastown-test")

	_, err := adapter.Town(context.Background())
	if err == nil {
		t.Error("Expected error for non-existent town")
	}
}

func TestFSAdapter_Rigs_Empty(t *testing.T) {
	// Create a temporary town with no rigs
	tmpDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tmpDir, "mayor"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "mayor", "town.json"), []byte(`{"name":"test-town"}`), 0644); err != nil {
		t.Fatal(err)
	}

	adapter := NewFSAdapter(tmpDir)

	rigs, err := adapter.Rigs(context.Background())
	if err != nil {
		t.Fatalf("Rigs() returned error: %v", err)
	}

	if len(rigs) != 0 {
		t.Errorf("Expected 0 rigs, got %d", len(rigs))
	}
}

func TestFSAdapter_Agents_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tmpDir, "mayor"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "mayor", "town.json"), []byte(`{"name":"test-town"}`), 0644); err != nil {
		t.Fatal(err)
	}

	adapter := NewFSAdapter(tmpDir)

	agents, err := adapter.Agents(context.Background())
	if err != nil {
		t.Fatalf("Agents() returned error: %v", err)
	}

	// Should have at least mayor (offline)
	if len(agents) < 1 {
		t.Errorf("Expected at least 1 agent (mayor), got %d", len(agents))
	}
}

func TestNewFSAdapter_DefaultPath(t *testing.T) {
	adapter := NewFSAdapter("")

	// Should default to ~/gt
	home := os.Getenv("HOME")
	expected := filepath.Join(home, "gt")

	status, _ := adapter.Status(context.Background())
	if status.TownRoot != expected {
		t.Errorf("Expected default path %s, got %s", expected, status.TownRoot)
	}
}

// TestParseWispListOutput_Array exercises the common case where
// `gt wisps list --json` returns a JSON array of wisp objects. Schema fields
// match what gt 0.9 emits for the wisps store (rename of gt 0.8 molecules).
func TestParseWispListOutput_Array(t *testing.T) {
	input := []byte(`[
		{
			"id": "wisp-a",
			"title": "Foundation gates",
			"status": "in_progress",
			"formula": "phase-2-burst",
			"current_step": 1,
			"agent": "polecat-1",
			"rig": "alpha",
			"steps": [
				{"index": 0, "id": "s0", "description": "parser fix", "status": "complete"},
				{"index": 1, "id": "s1", "description": "wisps adapter", "status": "in_progress"}
			],
			"created_at": "2026-05-23T00:00:00Z",
			"updated_at": "2026-05-24T00:00:00Z"
		},
		{
			"id": "wisp-b",
			"title": "Hardening",
			"status": "complete"
		}
	]`)
	raws, err := parseWispListOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(raws) != 2 {
		t.Fatalf("expected 2 wisps, got %d", len(raws))
	}

	a := raws[0].toMolecule()
	if a.ID != "wisp-a" || a.Status != MolStatusInProgress {
		t.Errorf("wisp-a: ID=%q status=%q", a.ID, a.Status)
	}
	if a.Total != 2 || a.Progress != 1 {
		t.Errorf("wisp-a progress: total=%d progress=%d (want 2 / 1)", a.Total, a.Progress)
	}
	if a.Agent != "polecat-1" || a.Rig != "alpha" {
		t.Errorf("wisp-a agent/rig: agent=%q rig=%q", a.Agent, a.Rig)
	}

	b := raws[1].toMolecule()
	if b.Status != MolStatusComplete {
		t.Errorf("wisp-b status: %q (want complete)", b.Status)
	}
}

// TestParseWispListOutput_SingleObject covers the defensive fallback where the
// gt wisps subcommand returns a single object instead of an array — same
// posture Convoys() takes against `gt convoy list --json` for single-result
// responses.
func TestParseWispListOutput_SingleObject(t *testing.T) {
	input := []byte(`{"id":"lone","title":"Only","status":"blocked"}`)
	raws, err := parseWispListOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(raws) != 1 || raws[0].ID != "lone" {
		t.Fatalf("expected 1 wisp lone, got %+v", raws)
	}
	if raws[0].toMolecule().Status != MolStatusBlocked {
		t.Errorf("expected blocked, got %q", raws[0].toMolecule().Status)
	}
}

// TestParseWispListOutput_Malformed asserts the contract that bogus output
// returns an error rather than empty silent success — callers (Molecules())
// translate that to a graceful nil, but the parser itself should distinguish
// "no wisps" from "I have no idea what this is."
func TestParseWispListOutput_Malformed(t *testing.T) {
	_, err := parseWispListOutput([]byte(`not json at all`))
	if err == nil {
		t.Error("expected error on malformed JSON, got nil")
	}
}

// TestMolecules_GtAbsent confirms the graceful-degradation contract: when
// `gt` is not installed or the town directory doesn't exist, Molecules()
// returns (nil, nil) so the gas-town view degrades gracefully rather than
// 500'ing. This is the same posture Convoys() takes and is documented in
// repo CLAUDE.md as a key design decision.
func TestMolecules_GtAbsent(t *testing.T) {
	adapter := NewFSAdapter("/tmp/nonexistent-gastown-test")
	mols, err := adapter.Molecules(context.Background())
	if err != nil {
		t.Errorf("Molecules() with no gt should not error, got: %v", err)
	}
	if mols != nil && len(mols) != 0 {
		t.Errorf("expected nil or empty Molecules, got %d", len(mols))
	}
}

func TestLatestActivity(t *testing.T) {
	now := time.Now()

	t.Run("missing dir returns very stale", func(t *testing.T) {
		got := latestActivity("/tmp/nonexistent-gastown-test-dir", now)
		if got < 999*time.Hour {
			t.Errorf("missing dir should be reported as very stale, got %v", got)
		}
	})

	t.Run("empty dir uses dir mtime", func(t *testing.T) {
		dir := t.TempDir()
		past := now.Add(-15 * time.Minute)
		if err := os.Chtimes(dir, past, past); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
		got := latestActivity(dir, now)
		if got < 14*time.Minute || got > 16*time.Minute {
			t.Errorf("expected ~15m, got %v", got)
		}
	})

	t.Run("newer child file shadows older dir mtime", func(t *testing.T) {
		// This is the scenario CodeRabbit flagged: dir mtime stale because no
		// add/remove, but a long-running agent rewrote a file inside it.
		dir := t.TempDir()
		oldDir := now.Add(-30 * time.Minute)
		if err := os.Chtimes(dir, oldDir, oldDir); err != nil {
			t.Fatalf("chtimes dir: %v", err)
		}
		child := filepath.Join(dir, "seance.json")
		if err := os.WriteFile(child, []byte("{}"), 0o644); err != nil {
			t.Fatalf("write child: %v", err)
		}
		// os.WriteFile sets mtime to ~now; verify the helper picks it up over the
		// stale dir mtime.
		got := latestActivity(dir, now)
		if got > 1*time.Minute {
			t.Errorf("expected newest file to dominate, got %v (older than 1 min)", got)
		}
	})
}

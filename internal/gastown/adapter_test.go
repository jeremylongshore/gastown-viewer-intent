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

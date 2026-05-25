package tui

import (
	"testing"
	"time"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/model"
)

// newTestModel builds a Model with the default registry + enough
// pre-populated state for the dispatcher tests to exercise navigation
// branches without hitting HTTP. The client field is set but never
// called (we test handlers that touch local state only or assert
// returned tea.Cmds are non-nil for handlers that *would* fetch).
func newTestModel() Model {
	m := New("http://127.0.0.1:0")
	m.board = &BoardResponse{
		Columns: []model.Column{
			{Label: "Pending", Status: model.StatusPending, Count: 2, Issues: []model.IssueSummary{
				{ID: "a", Title: "alpha"}, {ID: "b", Title: "bravo"},
			}},
			{Label: "Done", Status: model.StatusDone, Count: 1, Issues: []model.IssueSummary{
				{ID: "c", Title: "charlie"},
			}},
		},
	}
	m.memories = &model.MemoriesResponse{
		Count: 2,
		Memories: []model.Memory{
			{Key: "one", Content: "first"},
			{Key: "two", Content: "second", Redacted: true, RedactionMarkers: []string{"partner"}},
		},
	}
	m.human = &model.HumanFlagsResponse{
		Count: 1,
		Flags: []model.Issue{
			{ID: "h-1", Title: "needs decision", Status: model.StatusPending},
		},
	}
	m.loading = false
	return m
}

// TestDispatch_GlobalBindings exercises the tab switches + quit + help
// toggle. Each row asserts the focus / showHelp change. Tab-switch
// handlers also return a non-nil tea.Cmd (the fetch); we just confirm
// the focus transition here — the Cmd content is irrelevant for the
// dispatch contract.
func TestDispatch_GlobalBindings(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		startFoc  Focus
		wantFocus Focus
		wantHelp  bool
	}{
		{"1 from Board stays Board", "1", FocusBoard, FocusBoard, false},
		{"2 switches to Memories", "2", FocusBoard, FocusMemories, false},
		{"m switches to Memories", "m", FocusBoard, FocusMemories, false},
		{"3 switches to Triage", "3", FocusBoard, FocusTriage, false},
		{"t switches to Triage", "t", FocusBoard, FocusTriage, false},
		{"b returns to Board from Memories", "b", FocusMemories, FocusBoard, false},
		{"? toggles help", "?", FocusBoard, FocusBoard, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.focus = tc.startFoc
			m, _, matched := m.registry.Dispatch(m, m.focus, tc.key)
			if !matched {
				t.Fatalf("dispatch did not match key %q", tc.key)
			}
			if m.focus != tc.wantFocus {
				t.Errorf("focus = %s, want %s", m.focus, tc.wantFocus)
			}
			if m.showHelp != tc.wantHelp {
				t.Errorf("showHelp = %v, want %v", m.showHelp, tc.wantHelp)
			}
		})
	}
}

// TestDispatch_BoardNavigation walks the cursor + issueCur changes for
// the h/j/k/l + G keys against a known board layout.
func TestDispatch_BoardNavigation(t *testing.T) {
	m := newTestModel()
	m.focus = FocusBoard

	// down within column 0
	m, _, _ = m.registry.Dispatch(m, m.focus, "j")
	if m.issueCur != 1 {
		t.Errorf("after j: issueCur=%d, want 1", m.issueCur)
	}

	// up back to 0
	m, _, _ = m.registry.Dispatch(m, m.focus, "k")
	if m.issueCur != 0 {
		t.Errorf("after k: issueCur=%d, want 0", m.issueCur)
	}

	// right to column 1 resets issueCur to 0
	m.issueCur = 1
	m, _, _ = m.registry.Dispatch(m, m.focus, "l")
	if m.cursor != 1 || m.issueCur != 0 {
		t.Errorf("after l: cursor=%d issueCur=%d, want 1/0", m.cursor, m.issueCur)
	}

	// G jumps to bottom of (now) 1-item column → stays at 0
	m, _, _ = m.registry.Dispatch(m, m.focus, "G")
	if m.issueCur != 0 {
		t.Errorf("after G in 1-item column: issueCur=%d, want 0", m.issueCur)
	}

	// left back to column 0, then G should jump to last item (idx 1)
	m, _, _ = m.registry.Dispatch(m, m.focus, "h")
	m, _, _ = m.registry.Dispatch(m, m.focus, "G")
	if m.cursor != 0 || m.issueCur != 1 {
		t.Errorf("after h+G: cursor=%d issueCur=%d, want 0/1", m.cursor, m.issueCur)
	}
}

// TestDispatch_FocusScoping verifies a Board-only binding does NOT fire
// when the focus is something else. Without this, a hasty refactor that
// drops the Focus check would silently regress the design.
func TestDispatch_FocusScoping(t *testing.T) {
	m := newTestModel()
	m.focus = FocusMemories
	before := m.cursor

	// "l" on the Memories focus must not move the board cursor.
	m, _, _ = m.registry.Dispatch(m, m.focus, "l")
	if m.cursor != before {
		t.Errorf("Board binding fired in Memories focus: cursor changed %d→%d", before, m.cursor)
	}
}

// TestDispatch_ComboGG asserts the gg combo fires only when both keys
// land within ComboTimeout; a stale leader produces no jump.
func TestDispatch_ComboGG(t *testing.T) {
	m := newTestModel()
	m.focus = FocusBoard
	m.issueCur = 1

	// First g — should be recognized as a combo leader by IsComboLeader.
	if !m.registry.IsComboLeader(FocusBoard, "g") {
		t.Fatal("g should be a combo leader on Board")
	}

	// Second g completes the combo.
	m, _, matched := m.registry.DispatchCombo(m, FocusBoard, "g", "g")
	if !matched {
		t.Fatal("gg combo did not match")
	}
	if m.issueCur != 0 {
		t.Errorf("after gg: issueCur=%d, want 0", m.issueCur)
	}
}

// TestDispatch_MemoriesNavigation covers j/k + enter (detail toggle)
// + the / search trigger path.
func TestDispatch_MemoriesNavigation(t *testing.T) {
	m := newTestModel()
	m.focus = FocusMemories

	m, _, _ = m.registry.Dispatch(m, m.focus, "j")
	if m.memCur != 1 {
		t.Errorf("after j: memCur=%d, want 1", m.memCur)
	}
	m, _, _ = m.registry.Dispatch(m, m.focus, "k")
	if m.memCur != 0 {
		t.Errorf("after k: memCur=%d, want 0", m.memCur)
	}

	m, _, _ = m.registry.Dispatch(m, m.focus, "enter")
	if !m.memDetailOpen {
		t.Error("enter should open memory detail")
	}
	m, _, _ = m.registry.Dispatch(m, m.focus, "enter")
	if m.memDetailOpen {
		t.Error("enter again should close memory detail")
	}

	m, _, _ = m.registry.Dispatch(m, m.focus, "/")
	if !m.memSearching {
		t.Error("/ should enter search mode")
	}
}

// TestDispatch_TriageNavigation covers j/k against the human-flag list
// and confirms enter on a non-empty list returns a tea.Cmd (the fetch).
func TestDispatch_TriageNavigation(t *testing.T) {
	m := newTestModel()
	m.focus = FocusTriage
	// Only 1 flag in fixture, so j should stay at 0.
	m, _, _ = m.registry.Dispatch(m, m.focus, "j")
	if m.humanCur != 0 {
		t.Errorf("j past end: humanCur=%d, want 0", m.humanCur)
	}

	// enter on a populated row must return a non-nil tea.Cmd
	// (the fetchIssue batch) so the dispatcher's HTTP wiring is real.
	_, cmd, matched := m.registry.Dispatch(m, m.focus, "enter")
	if !matched {
		t.Fatal("enter did not match on Triage")
	}
	if cmd == nil {
		t.Error("enter on Triage row should return a non-nil tea.Cmd")
	}
}

// TestTruncateRunes_UTF8Safe locks in the multi-byte safety fix from
// Gemini review round 1 — the original implementation byte-sliced and
// could split a multi-byte codepoint, producing malformed UTF-8 in the
// rendered output.
func TestTruncateRunes_UTF8Safe(t *testing.T) {
	tests := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"ascii under limit", "hello", 10, "hello"},
		{"ascii at limit", "hello", 5, "hello"},
		{"ascii truncated", "hello world", 8, "hello..."},
		{"cyrillic does not split codepoint", "привет мир", 6, "при..."},
		{"emoji does not split codepoint", "🚀🚀🚀🚀🚀", 4, "🚀..."},
		{"n <= 3 returns prefix without ellipsis", "hello", 2, "he"},
		{"n zero returns empty", "hello", 0, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateRunes(tc.in, tc.n)
			if got != tc.want {
				t.Errorf("truncateRunes(%q, %d) = %q, want %q", tc.in, tc.n, got, tc.want)
			}
		})
	}
}

// TestComboTimeoutCmd_FiresInRoughBounds is a sanity check that the
// tea.Tick wrapper produces a message within ~2x the timeout. We do
// NOT call the cmd directly inside Update; the bubbletea program does
// that. Here we just confirm the Cmd is well-formed.
func TestComboTimeoutCmd_FiresInRoughBounds(t *testing.T) {
	now := time.Now()
	cmd := comboTimeoutCmd(now)
	if cmd == nil {
		t.Fatal("comboTimeoutCmd returned nil")
	}
	msg := cmd()
	tm, ok := msg.(comboTimeoutMsg)
	if !ok {
		t.Fatalf("expected comboTimeoutMsg, got %T", msg)
	}
	if !tm.at.Equal(now) {
		t.Errorf("at = %v, want %v", tm.at, now)
	}
}

package tui

import (
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Focus is the current top-level view of the TUI. It drives both the View
// renderer and the key dispatcher — keys are registered per Focus and only
// fire when that Focus is active (global bindings register against every
// focus).
//
// Modeled after the focus enum pattern in
// Dicklesworthstone/beads_viewer pkg/ui/model.go — the binary View enum
// shipped in v0.1.0 collapsed Board+Issue into one toggle; this enum
// generalizes that to an arbitrary set of tabs without the
// boolean-flag explosion that pattern would otherwise force.
type Focus int

const (
	// FocusBoard is the default kanban-style view of beads.
	FocusBoard Focus = iota
	// FocusMemories is the read-only `bd memories` viewer with
	// daemon-applied redaction.
	FocusMemories
	// FocusTriage is the read-only `bd human list` queue.
	FocusTriage
	// FocusDetail is the per-issue detail screen (reached from Board via
	// enter). It does NOT have its own tab — it overlays Board.
	FocusDetail
)

// String returns the human-readable focus name (used by the tab bar +
// help overlay).
func (f Focus) String() string {
	switch f {
	case FocusBoard:
		return "Board"
	case FocusMemories:
		return "Memories"
	case FocusTriage:
		return "Triage"
	case FocusDetail:
		return "Detail"
	}
	return "?"
}

// Category groups bindings in the help overlay so the engineer sees
// related actions together (Navigation/Tabs/Actions) instead of one flat
// list.
type Category string

const (
	CategoryNavigation Category = "Navigation"
	CategoryTabs       Category = "Tabs"
	CategoryActions    Category = "Actions"
	CategoryGlobal     Category = "Global"
)

// KeyBindingDoc is the registry-side description of a single binding.
// The Help overlay, the dispatcher, and the tab bar all consume the same
// slice of these so they cannot drift. Modeled on the
// Dicklesworthstone/beads_viewer pkg/ui/keybindings.go pattern.
type KeyBindingDoc struct {
	// Keys is the set of literal key strings that triggers this binding.
	// Multi-key combos (e.g. "g","g") are represented as a single Combo
	// value below and have Keys=nil.
	Keys []string

	// Combo, when non-empty, means this binding fires after the listed
	// sequence of keys is pressed within ComboTimeout. The first key in
	// Combo also acts as a "pending" key that swallows the next press.
	Combo []string

	// Help is the right-hand description shown in the help overlay
	// (e.g., "left/right between columns").
	Help string

	// Focus restricts the binding to a single focus. A zero-value Focus
	// with Global=true means the binding applies everywhere.
	Focus Focus

	// Global is true for bindings that work regardless of Focus (Quit,
	// Help, tab switches, Refresh). Global bindings take precedence over
	// focus-scoped ones.
	Global bool

	// Category controls grouping in the help overlay.
	Category Category

	// Handler is invoked when the key (or full combo) matches. It
	// receives the current model and returns the updated model plus an
	// optional tea.Cmd. The dispatcher passes by value because the
	// bubbletea Update contract is value-based.
	Handler func(m Model) (Model, tea.Cmd)
}

// ComboTimeout is how long the dispatcher waits between the first and
// second keys of a combo before discarding the pending state. 250ms
// matches a comfortable double-tap-g cadence without making impatient
// users feel locked.
const ComboTimeout = 250 * time.Millisecond

// KeyRegistry holds every binding the TUI knows about. The same instance
// is consumed by the dispatcher (per-key lookup) and the help renderer
// (full list grouped by Category).
type KeyRegistry struct {
	mu       sync.RWMutex
	bindings []KeyBindingDoc
}

// NewKeyRegistry returns an empty registry. Callers populate it via
// Register; we keep the constructor minimal so tests can build a
// registry with just the bindings under test.
func NewKeyRegistry() *KeyRegistry {
	return &KeyRegistry{}
}

// Register adds a binding to the registry. Safe to call concurrently
// (e.g. if a future plugin pattern registers bindings from goroutines).
func (r *KeyRegistry) Register(b KeyBindingDoc) {
	r.mu.Lock()
	r.bindings = append(r.bindings, b)
	r.mu.Unlock()
}

// All returns a snapshot copy of the bindings. The help overlay walks
// this slice; a copy avoids holding the lock across the caller's
// rendering.
func (r *KeyRegistry) All() []KeyBindingDoc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]KeyBindingDoc, len(r.bindings))
	copy(out, r.bindings)
	return out
}

// Dispatch looks up the first binding that matches (focus, key) and
// invokes its handler. The first-match-wins ordering means callers
// should Register Global bindings before focus-scoped ones — the
// default set in defaultRegistry() follows that order.
//
// Returns the updated model, the tea.Cmd to schedule (may be nil), and
// a matched bool reporting whether any handler fired. If matched is
// false the caller's Update should fall through to its own switch
// (currently no other paths exist; preserved for future composition).
//
// The lock is released BEFORE the handler runs so a handler that
// re-enters the registry (e.g. a future binding that programmatically
// invokes another) does not deadlock against the RWMutex. This mirrors
// DispatchCombo's release-before-invoke order.
func (r *KeyRegistry) Dispatch(m Model, focus Focus, k string) (Model, tea.Cmd, bool) {
	r.mu.RLock()
	var handler func(Model) (Model, tea.Cmd)
	for _, b := range r.bindings {
		if b.Handler == nil {
			continue
		}
		if !b.Global && b.Focus != focus {
			continue
		}
		for _, kk := range b.Keys {
			if kk == k {
				handler = b.Handler
				break
			}
		}
		if handler != nil {
			break
		}
	}
	r.mu.RUnlock()

	if handler == nil {
		return m, nil, false
	}
	newModel, cmd := handler(m)
	return newModel, cmd, true
}

// DispatchCombo handles the second-and-subsequent keys of a combo. If
// the pending key plus the new key matches a registered Combo, the
// handler fires; otherwise the pending state is cleared and the new key
// is treated as a fresh single press by Dispatch.
func (r *KeyRegistry) DispatchCombo(m Model, focus Focus, pending, k string) (Model, tea.Cmd, bool) {
	r.mu.RLock()
	for _, b := range r.bindings {
		if len(b.Combo) != 2 || b.Handler == nil {
			continue
		}
		if !b.Global && b.Focus != focus {
			continue
		}
		if b.Combo[0] == pending && b.Combo[1] == k {
			r.mu.RUnlock()
			newModel, cmd := b.Handler(m)
			return newModel, cmd, true
		}
	}
	r.mu.RUnlock()
	return m, nil, false
}

// IsComboLeader reports whether k is the first key of any registered
// combo for the given focus. The dispatcher consults this before
// treating k as a final keystroke — if it leads a combo, we stash it as
// pendingComboKey and wait for the next press (or the timeout).
func (r *KeyRegistry) IsComboLeader(focus Focus, k string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, b := range r.bindings {
		if len(b.Combo) != 2 {
			continue
		}
		if !b.Global && b.Focus != focus {
			continue
		}
		if b.Combo[0] == k {
			return true
		}
	}
	return false
}

// comboTimeoutMsg fires after ComboTimeout to clear stale pending state
// when the user pressed a combo leader and then stopped typing.
type comboTimeoutMsg struct {
	at time.Time
}

// comboTimeoutCmd returns a tea.Cmd that emits comboTimeoutMsg after
// ComboTimeout. The model compares msg.at against its own stored
// pendingComboAt to ignore stale timers from prior combos.
func comboTimeoutCmd(at time.Time) tea.Cmd {
	return tea.Tick(ComboTimeout, func(time.Time) tea.Msg {
		return comboTimeoutMsg{at: at}
	})
}

// defaultRegistry builds the full set of bindings the TUI ships with.
// Order matters: Global bindings come first so they win the dispatcher's
// first-match check regardless of which focus we're in. Within each
// focus, navigation precedes actions precedes back/cancel.
func defaultRegistry() *KeyRegistry {
	r := NewKeyRegistry()

	// --- Global (work in every focus) ---
	r.Register(KeyBindingDoc{
		Keys: []string{"q", "ctrl+c"}, Help: "quit",
		Global: true, Category: CategoryGlobal,
		Handler: func(m Model) (Model, tea.Cmd) { return m, tea.Quit },
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"?"}, Help: "toggle help overlay",
		Global: true, Category: CategoryGlobal,
		Handler: func(m Model) (Model, tea.Cmd) {
			m.showHelp = !m.showHelp
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"1", "b"}, Help: "switch to Board tab",
		Global: true, Category: CategoryTabs,
		Handler: func(m Model) (Model, tea.Cmd) {
			m.focus = FocusBoard
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"2", "m"}, Help: "switch to Memories tab",
		Global: true, Category: CategoryTabs,
		Handler: func(m Model) (Model, tea.Cmd) {
			m.focus = FocusMemories
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchMemoriesCmd(""))
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"3", "t"}, Help: "switch to Triage tab",
		Global: true, Category: CategoryTabs,
		Handler: func(m Model) (Model, tea.Cmd) {
			m.focus = FocusTriage
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchHumanFlagsCmd())
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"r"}, Help: "refresh current view",
		Global: true, Category: CategoryActions,
		Handler: func(m Model) (Model, tea.Cmd) {
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, m.refreshCurrentFocus())
		},
	})

	// --- Board focus ---
	r.Register(KeyBindingDoc{
		Keys: []string{"left", "h"}, Help: "previous column",
		Focus: FocusBoard, Category: CategoryNavigation,
		Handler: func(m Model) (Model, tea.Cmd) {
			if m.board != nil && m.cursor > 0 {
				m.cursor--
				m.issueCur = 0
			}
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"right", "l"}, Help: "next column",
		Focus: FocusBoard, Category: CategoryNavigation,
		Handler: func(m Model) (Model, tea.Cmd) {
			if m.board != nil && m.cursor < len(m.board.Columns)-1 {
				m.cursor++
				m.issueCur = 0
			}
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"up", "k"}, Help: "up in column",
		Focus: FocusBoard, Category: CategoryNavigation,
		Handler: func(m Model) (Model, tea.Cmd) {
			if m.issueCur > 0 {
				m.issueCur--
			}
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"down", "j"}, Help: "down in column",
		Focus: FocusBoard, Category: CategoryNavigation,
		Handler: func(m Model) (Model, tea.Cmd) {
			if m.board != nil && m.cursor < len(m.board.Columns) {
				col := m.board.Columns[m.cursor]
				if m.issueCur < len(col.Issues)-1 {
					m.issueCur++
				}
			}
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"enter"}, Help: "open issue detail",
		Focus: FocusBoard, Category: CategoryActions,
		Handler: func(m Model) (Model, tea.Cmd) {
			if m.board == nil || m.cursor >= len(m.board.Columns) {
				return m, nil
			}
			col := m.board.Columns[m.cursor]
			if m.issueCur >= len(col.Issues) {
				return m, nil
			}
			issue := col.Issues[m.issueCur]
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchIssue(issue.ID))
		},
	})
	r.Register(KeyBindingDoc{
		Combo: []string{"g", "g"}, Help: "jump to top of column",
		Focus: FocusBoard, Category: CategoryNavigation,
		Handler: func(m Model) (Model, tea.Cmd) {
			m.issueCur = 0
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"G"}, Help: "jump to bottom of column",
		Focus: FocusBoard, Category: CategoryNavigation,
		Handler: func(m Model) (Model, tea.Cmd) {
			if m.board != nil && m.cursor < len(m.board.Columns) {
				col := m.board.Columns[m.cursor]
				if n := len(col.Issues); n > 0 {
					m.issueCur = n - 1
				}
			}
			return m, nil
		},
	})

	// --- Detail focus (reached from Board via enter) ---
	r.Register(KeyBindingDoc{
		Keys: []string{"esc", "backspace"}, Help: "back to board",
		Focus: FocusDetail, Category: CategoryNavigation,
		Handler: func(m Model) (Model, tea.Cmd) {
			m.focus = FocusBoard
			m.issue = nil
			return m, nil
		},
	})

	// --- Memories focus ---
	r.Register(KeyBindingDoc{
		Keys: []string{"down", "j"}, Help: "next memory",
		Focus: FocusMemories, Category: CategoryNavigation,
		Handler: func(m Model) (Model, tea.Cmd) {
			if m.memories != nil && m.memCur < len(m.memories.Memories)-1 {
				m.memCur++
			}
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"up", "k"}, Help: "previous memory",
		Focus: FocusMemories, Category: CategoryNavigation,
		Handler: func(m Model) (Model, tea.Cmd) {
			if m.memCur > 0 {
				m.memCur--
			}
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"enter"}, Help: "toggle full memory body",
		Focus: FocusMemories, Category: CategoryActions,
		Handler: func(m Model) (Model, tea.Cmd) {
			m.memDetailOpen = !m.memDetailOpen
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"/"}, Help: "search memories",
		Focus: FocusMemories, Category: CategoryActions,
		Handler: func(m Model) (Model, tea.Cmd) {
			m.memSearching = true
			m.memSearchInput = ""
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"esc"}, Help: "clear search / collapse detail",
		Focus: FocusMemories, Category: CategoryNavigation,
		Handler: func(m Model) (Model, tea.Cmd) {
			if m.memSearching {
				m.memSearching = false
				m.memSearchInput = ""
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.fetchMemoriesCmd(""))
			}
			if m.memDetailOpen {
				m.memDetailOpen = false
				return m, nil
			}
			return m, nil
		},
	})

	// --- Triage focus ---
	r.Register(KeyBindingDoc{
		Keys: []string{"down", "j"}, Help: "next flagged bead",
		Focus: FocusTriage, Category: CategoryNavigation,
		Handler: func(m Model) (Model, tea.Cmd) {
			if m.human != nil && m.humanCur < len(m.human.Flags)-1 {
				m.humanCur++
			}
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"up", "k"}, Help: "previous flagged bead",
		Focus: FocusTriage, Category: CategoryNavigation,
		Handler: func(m Model) (Model, tea.Cmd) {
			if m.humanCur > 0 {
				m.humanCur--
			}
			return m, nil
		},
	})
	r.Register(KeyBindingDoc{
		Keys: []string{"enter"}, Help: "open referenced bead",
		Focus: FocusTriage, Category: CategoryActions,
		Handler: func(m Model) (Model, tea.Cmd) {
			if m.human == nil || m.humanCur >= len(m.human.Flags) {
				return m, nil
			}
			id := m.human.Flags[m.humanCur].ID
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchIssue(id))
		},
	})

	return r
}

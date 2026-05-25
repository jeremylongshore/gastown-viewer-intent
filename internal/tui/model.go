package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/intent-solutions-io/gastown-viewer-intent/internal/model"
)

// Model is the main TUI model. Field ordering: client + transport,
// per-focus data, dispatcher + key state, ui state.
type Model struct {
	client *Client

	// per-focus data caches
	board    *BoardResponse
	issue    *model.Issue
	memories *model.MemoriesResponse
	human    *model.HumanFlagsResponse

	// dispatcher + combo state
	registry        *KeyRegistry
	pendingComboKey string
	pendingComboAt  time.Time

	// ui state
	focus    Focus
	showHelp bool
	err      error
	loading  bool
	spinner  spinner.Model

	// board cursor
	cursor   int
	issueCur int

	// memories cursor + search + detail toggle
	memCur         int
	memDetailOpen  bool
	memSearching   bool
	memSearchInput string

	// triage cursor
	humanCur int

	width  int
	height int
}

// New creates a new TUI model wired to the given daemon URL.
func New(apiURL string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		client:   NewClient(apiURL),
		spinner:  s,
		registry: defaultRegistry(),
		loading:  true,
		width:    80,
		height:   24,
		focus:    FocusBoard,
	}
}

// --- Messages ---

type boardMsg *BoardResponse
type issueMsg *model.Issue
type memoriesMsg *model.MemoriesResponse
type humanMsg *model.HumanFlagsResponse
type errMsg error

// --- Command builders ---

func (m Model) fetchBoard() tea.Msg {
	board, err := m.client.Board()
	if err != nil {
		return errMsg(err)
	}
	return boardMsg(board)
}

func (m Model) fetchIssue(id string) tea.Cmd {
	return func() tea.Msg {
		issue, err := m.client.Issue(id)
		if err != nil {
			return errMsg(err)
		}
		return issueMsg(issue)
	}
}

// fetchMemoriesCmd returns a tea.Cmd that loads memories. If q is
// non-empty, it hits the search endpoint; otherwise the full list.
func (m Model) fetchMemoriesCmd(q string) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.SearchMemories(q)
		if err != nil {
			return errMsg(err)
		}
		return memoriesMsg(resp)
	}
}

func (m Model) fetchHumanFlagsCmd() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.HumanFlags()
		if err != nil {
			return errMsg(err)
		}
		return humanMsg(resp)
	}
}

// refreshCurrentFocus returns the tea.Cmd that re-fetches whatever the
// current focus shows. Triggered by `r` from anywhere.
func (m Model) refreshCurrentFocus() tea.Cmd {
	switch m.focus {
	case FocusMemories:
		return m.fetchMemoriesCmd(m.memSearchInput)
	case FocusTriage:
		return m.fetchHumanFlagsCmd()
	case FocusDetail:
		if m.issue != nil {
			return m.fetchIssue(m.issue.ID)
		}
		return m.fetchBoard
	default:
		return m.fetchBoard
	}
}

// --- Init / Update / View ---

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchBoard)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case comboTimeoutMsg:
		// Only clear if this timeout matches the still-pending key —
		// otherwise a later combo already replaced the pending state.
		if !m.pendingComboAt.IsZero() && msg.at.Equal(m.pendingComboAt) {
			m.pendingComboKey = ""
			m.pendingComboAt = time.Time{}
		}
		return m, nil

	case boardMsg:
		m.loading = false
		m.board = msg
		m.err = nil
		return m, nil

	case issueMsg:
		m.loading = false
		m.issue = msg
		m.focus = FocusDetail
		m.err = nil
		return m, nil

	case memoriesMsg:
		m.loading = false
		m.memories = msg
		if m.memories != nil && m.memCur >= len(m.memories.Memories) {
			m.memCur = 0
		}
		m.err = nil
		return m, nil

	case humanMsg:
		m.loading = false
		m.human = msg
		if m.human != nil && m.humanCur >= len(m.human.Flags) {
			m.humanCur = 0
		}
		m.err = nil
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg
		return m, nil
	}

	return m, nil
}

// handleKey routes a tea.KeyMsg through the search-input capture (when
// active), then the combo state machine, then the registry. The
// dispatcher returns matched=false only when no binding fires; in that
// case we silently drop the key.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()

	// Search-mode capture on Memories tab. While searching, regular
	// keys feed the input; enter commits, esc aborts.
	if m.focus == FocusMemories && m.memSearching {
		switch k {
		case "enter":
			m.memSearching = false
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchMemoriesCmd(m.memSearchInput))
		case "esc":
			m.memSearching = false
			m.memSearchInput = ""
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchMemoriesCmd(""))
		case "backspace":
			if n := len(m.memSearchInput); n > 0 {
				m.memSearchInput = m.memSearchInput[:n-1]
			}
			return m, nil
		}
		if len(k) == 1 {
			m.memSearchInput += k
			return m, nil
		}
		return m, nil
	}

	// Combo finalization — second key after a pending leader.
	if m.pendingComboKey != "" {
		pending := m.pendingComboKey
		m.pendingComboKey = ""
		m.pendingComboAt = time.Time{}
		if newModel, cmd, matched := m.registry.DispatchCombo(m, m.focus, pending, k); matched {
			return newModel, cmd
		}
		// Fall through and let `k` be dispatched as a fresh press.
	}

	// Combo leader — stash and arm timeout.
	if m.registry.IsComboLeader(m.focus, k) {
		now := time.Now()
		m.pendingComboKey = k
		m.pendingComboAt = now
		return m, comboTimeoutCmd(now)
	}

	// Help overlay swallows everything except `?`, `q`, ctrl+c, and
	// the tab-switch keys — those are the only Global bindings the
	// registry exposes outside the gameplay.
	if m.showHelp {
		switch k {
		case "?", "q", "ctrl+c", "1", "2", "3", "b", "m", "t":
			// fall through to dispatcher
		default:
			return m, nil
		}
	}

	newModel, cmd, _ := m.registry.Dispatch(m, m.focus, k)
	return newModel, cmd
}

// --- Styles ---

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Padding(0, 1)

	columnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(24)

	selectedColumnStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("205")).
				Padding(0, 1).
				Width(24)

	issueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	selectedIssueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	statusPending = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	statusInProgress = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))

	statusDone = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	statusBlocked = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	detailStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	redactionMarkerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)
)

// View renders the model.
func (m Model) View() string {
	if m.err != nil {
		return m.viewError()
	}
	if m.loading && m.board == nil && m.focus == FocusBoard {
		return m.viewLoading()
	}

	tabs := m.viewTabs()

	var content string
	switch m.focus {
	case FocusBoard:
		content = m.viewBoard()
	case FocusDetail:
		content = m.viewIssue()
	case FocusMemories:
		content = m.viewMemories()
	case FocusTriage:
		content = m.viewTriage()
	}

	body := tabs + "\n" + content
	if m.showHelp {
		body += "\n\n" + m.viewHelp()
	} else {
		body += "\n\n" + hintStyle.Render("? help · q quit · 1/2/3 tabs")
	}
	return body
}

// viewTabs renders the top tab bar (Board · Memories · Triage). Detail
// is treated as part of Board for tab-highlight purposes.
func (m Model) viewTabs() string {
	tab := func(label string, focus Focus) string {
		active := m.focus == focus || (focus == FocusBoard && m.focus == FocusDetail)
		if active {
			return tabActiveStyle.Render("[ " + label + " ]")
		}
		return tabInactiveStyle.Render("  " + label + "  ")
	}
	return tab("Board", FocusBoard) + tab("Memories", FocusMemories) + tab("Triage", FocusTriage)
}

func (m Model) viewLoading() string {
	return fmt.Sprintf("\n  %s Loading...\n\n", m.spinner.View())
}

func (m Model) viewError() string {
	return fmt.Sprintf("\n  %s\n\n  %s\n\n  Press 'r' to retry or 'q' to quit.\n",
		errorStyle.Render("Error connecting to daemon:"),
		m.err.Error())
}

func (m Model) viewBoard() string {
	if m.board == nil {
		return m.viewLoading()
	}

	var b strings.Builder

	title := titleStyle.Render("Gastown Viewer Intent")
	if m.loading {
		title += " " + m.spinner.View()
	}
	b.WriteString(title + "\n\n")

	var columns []string
	for i, col := range m.board.Columns {
		colStyle := columnStyle
		if i == m.cursor {
			colStyle = selectedColumnStyle
		}

		var headerStyle lipgloss.Style
		switch col.Status {
		case model.StatusPending:
			headerStyle = statusPending
		case model.StatusInProgress:
			headerStyle = statusInProgress
		case model.StatusDone:
			headerStyle = statusDone
		case model.StatusBlocked:
			headerStyle = statusBlocked
		default:
			headerStyle = statusPending
		}

		header := headerStyle.Render(fmt.Sprintf("%s (%d)", col.Label, col.Count))

		var issues []string
		for j, issue := range col.Issues {
			style := issueStyle
			if i == m.cursor && j == m.issueCur {
				style = selectedIssueStyle
			}
			title := issue.Title
			if len(title) > 20 {
				title = title[:17] + "..."
			}
			issues = append(issues, style.Render(title))
		}

		content := header + "\n" + strings.Join(issues, "\n")
		columns = append(columns, colStyle.Render(content))
	}

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, columns...))
	return b.String()
}

func (m Model) viewIssue() string {
	if m.issue == nil {
		return m.viewLoading()
	}

	var b strings.Builder
	b.WriteString(labelStyle.Render("< Press ESC to go back") + "\n\n")
	b.WriteString(titleStyle.Render(m.issue.Title) + "\n")

	var sStyle lipgloss.Style
	switch m.issue.Status {
	case model.StatusPending:
		sStyle = statusPending
	case model.StatusInProgress:
		sStyle = statusInProgress
	case model.StatusDone:
		sStyle = statusDone
	case model.StatusBlocked:
		sStyle = statusBlocked
	}
	b.WriteString(fmt.Sprintf("%s  %s\n\n",
		sStyle.Render(string(m.issue.Status)),
		labelStyle.Render(fmt.Sprintf("[%s]", m.issue.Priority))))

	b.WriteString(labelStyle.Render("ID: ") + m.issue.ID + "\n\n")

	if m.issue.Description != "" {
		b.WriteString(labelStyle.Render("Description:\n"))
		desc := m.issue.Description
		if len(desc) > 500 {
			desc = desc[:497] + "..."
		}
		b.WriteString(desc + "\n\n")
	}

	if len(m.issue.DoneWhen) > 0 {
		b.WriteString(labelStyle.Render("Done when:\n"))
		for _, item := range m.issue.DoneWhen {
			b.WriteString("  - " + item + "\n")
		}
		b.WriteString("\n")
	}
	if len(m.issue.Blocks) > 0 {
		b.WriteString(labelStyle.Render("Blocks:\n"))
		for _, dep := range m.issue.Blocks {
			b.WriteString(fmt.Sprintf("  - %s (%s)\n", dep.Title, dep.ID))
		}
		b.WriteString("\n")
	}
	if len(m.issue.BlockedBy) > 0 {
		b.WriteString(labelStyle.Render("Blocked by:\n"))
		for _, dep := range m.issue.BlockedBy {
			b.WriteString(fmt.Sprintf("  - %s (%s)\n", dep.Title, dep.ID))
		}
		b.WriteString("\n")
	}

	width := m.width - 4
	if width < 20 {
		width = 20
	}
	return detailStyle.Width(width).Render(b.String())
}

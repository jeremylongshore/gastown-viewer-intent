package tui

import (
	"fmt"
	"strings"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/model"
)

// viewTriage renders the human-flag queue. Read-only by design
// (Council Q0 Surface 3); enter jumps into the Board's Issue Detail
// for the selected bead.
func (m Model) viewTriage() string {
	if m.human == nil {
		return m.viewLoading()
	}

	var b strings.Builder
	header := titleStyle.Render("Triage")
	if m.loading {
		header += " " + m.spinner.View()
	}
	b.WriteString(header + "\n")
	b.WriteString(hintStyle.Render(
		fmt.Sprintf("%d beads flagged for human decision · enter to open · read-only", m.human.Count)) + "\n\n")

	if len(m.human.Flags) == 0 {
		b.WriteString(hintStyle.Render("  (queue is empty — nothing needs human attention)") + "\n")
		return b.String()
	}

	for i, f := range m.human.Flags {
		marker := " "
		idStyle := issueStyle
		if i == m.humanCur {
			marker = ">"
			idStyle = selectedIssueStyle
		}

		var sStyle = statusPending
		switch f.Status {
		case model.StatusInProgress:
			sStyle = statusInProgress
		case model.StatusDone:
			sStyle = statusDone
		case model.StatusBlocked:
			sStyle = statusBlocked
		}

		title := f.Title
		maxLen := m.width - 30
		if maxLen < 20 {
			maxLen = 20
		}
		if len(title) > maxLen {
			title = title[:maxLen-3] + "..."
		}

		b.WriteString(fmt.Sprintf("%s %s  %s  %s  %s\n",
			marker,
			idStyle.Render(f.ID),
			sStyle.Render(string(f.Status)),
			labelStyle.Render(fmt.Sprintf("[%s]", f.Priority)),
			title,
		))
	}

	return b.String()
}

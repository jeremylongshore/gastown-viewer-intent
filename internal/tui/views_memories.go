package tui

import (
	"fmt"
	"strings"
)

// viewMemories renders the memories tab. List + optional detail
// expansion under the selected key. Reveal is intentionally NOT
// exposed — Council Q2 read-only-forever invariant.
func (m Model) viewMemories() string {
	if m.memories == nil {
		return m.viewLoading()
	}

	var b strings.Builder
	header := titleStyle.Render("Memories")
	if m.loading {
		header += " " + m.spinner.View()
	}
	b.WriteString(header + "\n")

	// Search box / state
	if m.memSearching {
		b.WriteString(labelStyle.Render("/ ") + m.memSearchInput + "_\n")
		b.WriteString(hintStyle.Render("enter to search · esc to cancel") + "\n\n")
	} else if m.memSearchInput != "" {
		b.WriteString(labelStyle.Render(fmt.Sprintf("filter: %q · esc to clear", m.memSearchInput)) + "\n\n")
	} else {
		b.WriteString(hintStyle.Render(fmt.Sprintf("%d memories · / to search · enter to expand", m.memories.Count)) + "\n\n")
	}

	if len(m.memories.Memories) == 0 {
		b.WriteString(hintStyle.Render("  (no memories)") + "\n")
		return b.String()
	}

	for i, mem := range m.memories.Memories {
		marker := " "
		keyStyle := issueStyle
		if i == m.memCur {
			marker = ">"
			keyStyle = selectedIssueStyle
		}
		line := fmt.Sprintf("%s %s", marker, keyStyle.Render(mem.Key))
		if mem.Redacted {
			line += " " + redactionMarkerStyle.Render("[redacted]")
		}
		b.WriteString(line + "\n")

		if i == m.memCur && m.memDetailOpen {
			b.WriteString(m.viewMemoryDetail(mem.Key, mem.Content, mem.Redacted, mem.RedactionMarkers))
		} else if i == m.memCur && !m.memDetailOpen {
			preview := strings.ReplaceAll(mem.Content, "\n", " ")
			preview = truncateRunes(preview, 80)
			b.WriteString(labelStyle.Render("    "+preview) + "\n")
		}
	}

	return b.String()
}

// viewMemoryDetail renders the expanded memory body under the selected
// row. The Copy hint surfaces the bd CLI handoff (the canonical path
// for revealing redacted content; the TUI never reveals in-process).
func (m Model) viewMemoryDetail(key, content string, redacted bool, markers []string) string {
	var b strings.Builder
	b.WriteString("\n")
	if redacted {
		markerStr := strings.Join(markers, ", ")
		if markerStr == "" {
			markerStr = "see daemon policy"
		}
		b.WriteString(redactionMarkerStyle.Render(
			fmt.Sprintf("    [redacted: %s]", markerStr)) + "\n")
	}

	width := m.width - 8
	if width < 20 {
		width = 20
	}
	for _, line := range wrap(content, width) {
		b.WriteString("    " + line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render(
		fmt.Sprintf("    to reveal full content: bd recall %s", key)) + "\n")
	return b.String()
}

// truncateRunes returns s shortened to at most n runes, suffixed with
// "..." when truncation actually occurred. Works on the rune count so
// multi-byte UTF-8 (Cyrillic, CJK, emoji in memory content) is never
// sliced mid-codepoint. n is the visible width budget INCLUDING the
// ellipsis — callers pass the final desired width.
func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}

// wrap is a tiny word-wrap helper. We avoid pulling in a wrapper
// dependency because the TUI already keeps its render layer minimal.
func wrap(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		if len(para) <= width {
			out = append(out, para)
			continue
		}
		words := strings.Fields(para)
		var line strings.Builder
		for _, w := range words {
			if line.Len() == 0 {
				line.WriteString(w)
				continue
			}
			if line.Len()+1+len(w) > width {
				out = append(out, line.String())
				line.Reset()
				line.WriteString(w)
				continue
			}
			line.WriteString(" ")
			line.WriteString(w)
		}
		if line.Len() > 0 {
			out = append(out, line.String())
		}
	}
	return out
}

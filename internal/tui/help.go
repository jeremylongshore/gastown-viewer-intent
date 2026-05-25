package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// viewHelp renders the help overlay from the same KeyRegistry the
// dispatcher consumes — that single source of truth is the load-bearing
// idea borrowed from Dicklesworthstone/beads_viewer. Help can never
// drift from the registered bindings because both sides walk the same
// slice.
func (m Model) viewHelp() string {
	docs := m.registry.All()

	// Filter: include Global bindings + bindings scoped to current
	// focus. Each focus is shown in isolation — Detail has its own
	// minimal set (esc/back) and intentionally does NOT inherit Board
	// navigation, so a user reading an issue isn't tempted to move the
	// kanban cursor while in the detail screen.
	relevant := make([]KeyBindingDoc, 0, len(docs))
	for _, d := range docs {
		if d.Global {
			relevant = append(relevant, d)
			continue
		}
		if d.Focus == m.focus {
			relevant = append(relevant, d)
		}
	}

	// Group by Category.
	groups := map[Category][]KeyBindingDoc{}
	for _, d := range relevant {
		groups[d.Category] = append(groups[d.Category], d)
	}

	// Stable rendering order for category groups.
	order := []Category{
		CategoryTabs,
		CategoryNavigation,
		CategoryActions,
		CategoryGlobal,
	}

	titleBar := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render(fmt.Sprintf("Help — %s tab (press ? to close)", m.focus))

	var sections []string
	for _, cat := range order {
		bindings, ok := groups[cat]
		if !ok {
			continue
		}
		sort.SliceStable(bindings, func(i, j int) bool {
			return bindings[i].Help < bindings[j].Help
		})
		var rows []string
		for _, b := range bindings {
			rows = append(rows, fmt.Sprintf("  %-14s  %s",
				formatKeys(b), b.Help))
		}
		section := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214")).
			Render(string(cat)) + "\n" + strings.Join(rows, "\n")
		sections = append(sections, section)
	}

	body := titleBar + "\n\n" + strings.Join(sections, "\n\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Render(body)
}

// formatKeys produces the left-column key glyph for a binding.
// Single-key bindings collapse synonyms (e.g. "left/h"), combos render
// as concatenated literals ("gg").
func formatKeys(b KeyBindingDoc) string {
	if len(b.Combo) > 0 {
		return strings.Join(b.Combo, "")
	}
	pretty := make([]string, 0, len(b.Keys))
	for _, k := range b.Keys {
		switch k {
		case "left":
			pretty = append(pretty, "←")
		case "right":
			pretty = append(pretty, "→")
		case "up":
			pretty = append(pretty, "↑")
		case "down":
			pretty = append(pretty, "↓")
		default:
			pretty = append(pretty, k)
		}
	}
	return strings.Join(pretty, "/")
}

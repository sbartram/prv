package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/ansi"
)

// View renders the current state. The alt screen is declared here in v2.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m Model) render() string {
	switch m.state {
	case stateLoading:
		return m.wrap(m.spin.View() + " Fetching your PRs…")
	case stateError:
		return m.wrap(m.renderError())
	case stateEmpty:
		return m.wrap(fmt.Sprintf("No open PRs authored by or assigned to @%s, and none closed in the last %d days.\n\nq quit",
			m.snap.Viewer, m.opts.ClosedDays))
	}
	return m.renderList()
}

// wrap word-wraps a message screen to the pane width, so no line is wider than
// the terminal. Before the first size message it returns s unchanged.
func (m Model) wrap(s string) string {
	if m.w <= 0 {
		return s
	}
	lines := strings.Split(lipgloss.NewStyle().Width(m.w).Render(s), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

// hinter is implemented by errors that know how to tell the user what to do.
// It is declared here so that ui never imports internal/github.
type hinter interface{ Hint() string }

func (m Model) renderError() string {
	var b strings.Builder
	b.WriteString("Could not load your PRs\n\n")
	b.WriteString(m.err.Error())
	var h hinter
	if errors.As(m.err, &h) {
		if hint := h.Hint(); hint != "" {
			b.WriteString("\n\n" + hint)
		}
	}
	b.WriteString("\n\nr retry · q quit")
	return b.String()
}

// cut shortens a list-view line to the pane width. It leaves the line alone
// before the first size message.
func (m Model) cut(line string) string {
	if m.w > 0 && lipgloss.Width(line) > m.w {
		return ansi.Truncate(line, m.w, "")
	}
	return line
}

func (m Model) renderList() string {
	// Build every cell first, so column widths stay the same while scrolling.
	cells := make([][]string, 0, len(m.snap.PRs)+1)
	cells = append(cells, []string{"", "PR", "TITLE", "AUTHOR", "STATE", "REVIEW", "CI"})
	for i, pr := range m.snap.PRs {
		marker := " "
		if i == m.cursor {
			marker = ">"
		}
		title := pr.Title
		if pr.IsDraft {
			title = "[draft] " + title
		}
		author := pr.Author
		if strings.EqualFold(pr.Author, m.snap.Viewer) {
			author = "me"
		}
		cells = append(cells, []string{
			marker, pr.Ref.String(), title, author, pr.State.String(),
			pr.Review.Badge(),
			pr.CI.Badge(),
		})
	}
	widths := columnWidths(cells, m.w)
	for _, row := range cells {
		for c := range row {
			row[c] = truncateCell(row[c], widths[c])
		}
	}

	// Only the visible slice goes to the table. The table's own Height and
	// YOffset are not used: they add an overflow row and shift StyleFunc rows.
	end := min(len(m.snap.PRs), m.offset+m.visibleRows())
	t := table.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
		BorderColumn(false).BorderHeader(false).BorderRow(false).
		Headers(cells[0]...).
		Rows(cells[1+m.offset : 1+end]...).
		StyleFunc(func(row, col int) lipgloss.Style {
			// Width includes padding in Lip Gloss v2: every column but the
			// last carries one cell of gap.
			s := lipgloss.NewStyle().Width(widths[col])
			if col < colCount-1 {
				s = s.Width(widths[col] + columnGap).PaddingRight(columnGap)
			}
			if row == table.HeaderRow {
				return s.Inherit(headerStyle)
			}
			idx := m.offset + row
			pr := m.snap.PRs[idx]
			if dimmed(pr) {
				s = s.Faint(true)
			}
			switch col {
			case colReview:
				s = s.Inherit(reviewStyle(pr.Review))
			case colCI:
				s = s.Inherit(ciStyle(pr.CI))
			}
			if idx == m.cursor {
				s = s.Inherit(selectedStyle)
			}
			return s
		})

	footer := "j/k move · q quit"
	if m.snap.Truncated {
		footer += " · list truncated"
	}
	if len(m.snap.Warnings) > 0 {
		footer += " · some data could not be loaded"
	}

	lines := []string{"prv · @" + m.snap.Viewer + " · " + strconv.Itoa(len(m.snap.PRs)) + " PRs"}
	lines = append(lines, strings.Split(t.String(), "\n")...)
	lines = append(lines, footer)
	for i, l := range lines {
		lines[i] = m.cut(l)
	}
	return strings.Join(lines, "\n")
}

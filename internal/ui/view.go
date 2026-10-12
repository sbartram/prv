package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// Table column indexes.
const (
	colReview = 5
	colCI     = 6
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
		return m.spin.View() + " Fetching your PRs…"
	case stateError:
		return m.renderError()
	case stateEmpty:
		return fmt.Sprintf("No open PRs authored by or assigned to @%s, and none closed in the last %d days.\n\nq quit",
			m.snap.Viewer, m.opts.ClosedDays)
	}
	return m.renderList()
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

func (m Model) renderList() string {
	var b strings.Builder
	b.WriteString("prv · @" + m.snap.Viewer + " · " + strconv.Itoa(len(m.snap.PRs)) + " PRs\n\n")

	rows := make([][]string, 0, len(m.snap.PRs))
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
		rows = append(rows, []string{
			marker, pr.Ref.String(), title, author, pr.State.String(),
			pr.Review.Badge(),
			pr.CI.Badge(),
		})
	}

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
		BorderColumn(false).BorderHeader(false).BorderRow(false).
		Headers("", "PR", "TITLE", "AUTHOR", "STATE", "REVIEW", "CI").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle().PaddingRight(2)
			if row == table.HeaderRow {
				return base.Inherit(headerStyle)
			}
			pr := m.snap.PRs[row]
			s := base
			if dimmed(pr) {
				s = s.Faint(true)
			}
			switch col {
			case colReview:
				s = s.Inherit(reviewStyle(pr.Review))
			case colCI:
				s = s.Inherit(ciStyle(pr.CI))
			}
			if row == m.cursor {
				s = s.Inherit(selectedStyle)
			}
			return s
		})
	b.WriteString(t.String())
	b.WriteString("\n\nj/k move · q quit")
	if m.snap.Truncated {
		b.WriteString(" · list truncated")
	}
	if len(m.snap.Warnings) > 0 {
		b.WriteString(" · some data could not be loaded")
	}
	return b.String()
}

package ui

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
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
		return "Could not load your PRs\n\n" + m.err.Error() + "\n\nq quit"
	}
	return m.renderList()
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
		StyleFunc(func(_, _ int) lipgloss.Style {
			return lipgloss.NewStyle().PaddingRight(2)
		})
	b.WriteString(t.String())
	b.WriteString("\n\nj/k move · q quit")
	return b.String()
}

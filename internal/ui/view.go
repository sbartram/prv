package ui

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

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
		return m.renderEmpty()
	}
	return m.renderList()
}

// renderEmpty is the screen for a snapshot with no PRs. When the snapshot is
// partial (truncated or with warnings) it must not claim there are no open PRs
// (WR-02): the notes in the footer say why the list is short.
func (m Model) renderEmpty() string {
	msg := fmt.Sprintf("No open PRs authored by or assigned to @%s, and none closed in the last %d days.",
		m.snap.Viewer, m.opts.ClosedDays)
	if m.snap.Truncated || len(m.snap.Warnings) > 0 {
		msg = fmt.Sprintf("No PRs could be listed for @%s.", m.snap.Viewer)
	}
	return m.header() + "\n" + m.wrap(msg) + "\n\n" + m.footer()
}

// footer is the last line of the list and empty screens. Integrity notes come
// first and outlive the key hints: on a narrow pane the hints go, then the
// notes are cut with an ellipsis, but the notes are never dropped (WR-03).
func (m Model) footer() string {
	var notes []string
	if m.snap.Truncated {
		notes = append(notes, "list truncated")
	}
	if len(m.snap.Warnings) > 0 {
		notes = append(notes, "some data could not be loaded")
	}
	const sep = " · "
	line := strings.Join(notes, sep)

	h := m.help // a copy: SetWidth must not change the model
	room := 0   // 0: not sized yet, the hints are not cut
	if m.w > 0 {
		room = m.w
		if line != "" {
			if lipgloss.Width(line) >= m.w {
				return ansi.Truncate(line, m.w, "…")
			}
			room = m.w - lipgloss.Width(line) - lipgloss.Width(sep)
		}
		if room < 8 {
			return line // too little room for a useful hint
		}
	}
	h.SetWidth(room)
	hints := h.ShortHelpView(m.keys.ShortHelp())
	if line == "" {
		return hints
	}
	return line + sep + hints
}

// clock formats t in the zone of the model's clock: the local zone in
// production, whatever zone the test clock uses in tests.
func (m Model) clock(t time.Time, layout string) string {
	return t.In(m.now().Location()).Format(layout)
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

// shorter is implemented by errors that name themselves in a word or two for
// the header (D-07). Declared here for the same reason as hinter.
type shorter interface{ Short() string }

// segment is one piece of the header line. A higher prio survives longer when
// the pane is too narrow for the whole line.
type segment struct {
	text string
	prio int
}

// fit joins segs with sep. While the line is wider than width it drops the
// segment with the lowest prio (the leftmost on a tie). When one segment is
// left and still too wide it is cut with an ellipsis. A width of 0 or less
// (not sized yet) keeps everything.
func fit(segs []segment, sep string, width int) string {
	segs = slices.Clone(segs)
	join := func() string {
		texts := make([]string, len(segs))
		for i, s := range segs {
			texts[i] = s.text
		}
		return strings.Join(texts, sep)
	}
	if width <= 0 {
		return join()
	}
	for lipgloss.Width(join()) > width && len(segs) > 1 {
		lowest := 0
		for i, s := range segs {
			if s.prio < segs[lowest].prio {
				lowest = i
			}
		}
		segs = slices.Delete(segs, lowest, lowest+1)
	}
	line := join()
	if lipgloss.Width(line) > width {
		return ansi.Truncate(line, width, "…")
	}
	return line
}

// header builds the first line of the list view. Prio 3 and 6 are left free:
// the filter segment (plan 02-08) takes 3 and the rate warning (plan 02-05) 6.
func (m Model) header() string {
	segs := []segment{
		{"prv", 1},
		{"@" + m.snap.Viewer, 2},
		{strconv.Itoa(len(m.rows)) + " PRs", 4},
	}
	if !m.lastOK.IsZero() {
		// Clock times, not "5s ago": nothing redraws once a second (D-05).
		text, prio := "updated "+m.clock(m.lastOK, "15:04:05"), 5
		if m.err != nil {
			// A failed refresh keeps the rows, so the header must say that
			// they are older than they look (D-07).
			reason := "error"
			var sh shorter
			if errors.As(m.err, &sh) {
				reason = sh.Short()
			}
			text = warnStyle.Render("✗ refresh failed " + m.clock(m.failedAt, "15:04") +
				" (" + reason + ") · data from " + m.clock(m.lastOK, "15:04:05"))
			prio = 7
		}
		if m.inFlight {
			text += " " + m.spin.View()
		}
		segs = append(segs, segment{text, prio})
	}
	return fit(segs, " · ", m.w)
}

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
	// D-09: the panel says when the automatic retry comes.
	if m.inFlight {
		b.WriteString("\n\nretrying now… " + m.spin.View())
	} else if !m.nextAt.IsZero() {
		b.WriteString("\n\nretrying at " + m.clock(m.nextAt, "15:04"))
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
	cells := make([][]string, 0, len(m.rows)+1)
	cells = append(cells, []string{"", "PR", "TITLE", "AUTHOR", "STATE", "REVIEW", "CI"})
	for i, pr := range m.rows {
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
	end := min(len(m.rows), m.offset+m.visibleRows())
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
			pr := m.rows[idx]
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

	lines := []string{m.header()}
	lines = append(lines, strings.Split(t.String(), "\n")...)
	lines = append(lines, m.footer())
	for i, l := range lines {
		lines[i] = m.cut(l)
	}
	return strings.Join(lines, "\n")
}

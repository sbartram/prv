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

	"github.com/sbartram/prv/internal/domain"
)

// View renders the current state. The alt screen is declared here in v2.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m Model) render() string {
	if m.showHelp {
		return m.renderHelp()
	}
	if m.detail.open {
		return m.renderDetail() // D-08: full screen, like the help overlay
	}
	switch m.state {
	case stateLoading:
		return m.wrap(m.spin.View() + " Fetching your PRs…")
	case stateError:
		return m.wrap(m.renderError())
	case stateEmpty:
		return m.renderEmpty()
	}
	if len(m.rows) == 0 && len(m.snap.PRs) > 0 {
		return m.renderFilteredEmpty()
	}
	return m.renderList()
}

// stateIndicator shows which states are visible, one slot per state: the name
// when visible and "·" when hidden, as "[open merged ·]" (D-10).
func (m Model) stateIndicator() string {
	slot := func(st domain.State) string {
		if m.show.Has(st) {
			return st.String()
		}
		return "·"
	}
	return "[" + slot(domain.StateOpen) + " " + slot(domain.StateMerged) + " " + slot(domain.StateClosed) + "]"
}

// filterLabel is the state indicator followed by the kept number filter, as
// "[open · closed] #12".
func (m Model) filterLabel() string {
	if m.numFilter == "" {
		return m.stateIndicator()
	}
	return m.stateIndicator() + " #" + m.numFilter
}

// renderFilteredEmpty is the screen for a snapshot that has PRs when the
// filters hide every one of them (D-13). It must not read like an empty
// account: it names the filters and how many PRs they hide. The footer still
// carries the integrity notes (WR-02).
func (m Model) renderFilteredEmpty() string {
	escHint := ""
	if m.numFilter != "" {
		escHint = "esc clear · "
	}
	msg := "No PRs match: " + m.filterLabel() +
		" — " + strconv.Itoa(len(m.snap.PRs)-len(m.rows)) + " hidden by filters · " + escHint + "1/2/3 toggle"
	return m.header() + "\n" + m.wrap(msg) + "\n\n" + m.bottom()
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
	return m.header() + "\n" + m.wrap(msg) + "\n\n" + m.bottom()
}

// notes are the footer's integrity notes, flash first so it is cut last.
func (m Model) notes() []string {
	var notes []string
	if m.flash != "" && m.now().Before(m.flashUntil) {
		notes = append(notes, warnStyle.Render(m.flash))
	}
	if m.snap.Truncated {
		notes = append(notes, "list truncated")
	}
	if len(m.snap.Warnings) > 0 {
		notes = append(notes, "some data could not be loaded")
	}
	return notes
}

// warning is the config-warning text while its 8 s are running, else "".
func (m Model) warning() string {
	if len(m.warnings) == 0 || !m.now().Before(m.warnUntil) {
		return ""
	}
	return strings.Join(m.warnings, " · ")
}

// bottom is the last line of every screen that has rows or a filter: the
// number prompt while it is open, otherwise the footer. The prompt replaces
// the footer, so it never costs a list row (chromeLines stays 3).
func (m Model) bottom() string {
	if m.add.open {
		return m.addLine()
	}
	if m.confirm.open {
		return m.confirmLine()
	}
	if !m.prompting {
		return m.footer()
	}
	line := "/ " + m.numFilter + "█   (" + strconv.Itoa(len(m.rows)) + " of " + strconv.Itoa(len(m.snap.PRs)) +
		" · enter keep · esc clear)"
	if notes := m.notes(); len(notes) > 0 {
		line += " · " + strings.Join(notes, " · ")
	}
	return m.cut(line)
}

// footer is the last line of the list and empty screens. Integrity notes come
// first, then the key hints; the faint "api N" budget sits at the right edge.
// On a narrow pane the hints go first, then the notes are cut with an
// ellipsis, but neither the notes nor the api segment is dropped (WR-03, D-06).
// The api segment is absent until the first successful response has told us
// the limit: showing a number before that would be made up.
func (m Model) footer() string {
	const sep = " · "
	line := strings.Join(m.notes(), sep)

	api := ""
	if m.snap.RateLimit.Limit > 0 {
		api = faintStyle.Render("api " + strconv.Itoa(m.snap.RateLimit.Remaining))
	}
	join := func(a, b string) string {
		if a == "" || b == "" {
			return a + b
		}
		return a + sep + b
	}

	h := m.help // a copy: SetWidth must not change the model
	hints := m.keys.ShortHelp()
	if m.detail.open {
		hints = m.keys.DetailShortHelp() // D-09
	}
	if m.w <= 0 {
		// Not sized yet: nothing is cut or padded.
		h.SetWidth(0)
		if w := m.warning(); w != "" {
			line = join(line, warnStyle.Render(w))
		}
		return join(join(line, h.ShortHelpView(hints)), api)
	}

	avail := m.w // cells left for the notes, the warning and the hints
	if api != "" {
		avail -= lipgloss.Width(api) + 1 // and one space before the api segment
	}
	if avail < 1 {
		return ansi.Truncate(api, m.w, "")
	}
	left := line
	if line != "" && lipgloss.Width(line) >= avail {
		left = ansi.Truncate(line, avail, "…")
	} else {
		room := avail
		if line != "" {
			room -= lipgloss.Width(line) + lipgloss.Width(sep)
		}
		// The config warning outranks the key hints but never the notes (D-15).
		if w := m.warning(); w != "" && room >= 12 {
			w = ansi.Truncate(w, room, "…")
			left = join(left, warnStyle.Render(w))
			room -= lipgloss.Width(w) + lipgloss.Width(sep)
		}
		if room >= 8 { // less is too little room for a useful hint
			h.SetWidth(room)
			hv := h.ShortHelpView(hints)
			// help lets an item overflow when its ellipsis would not fit
			// either: drop whole hints from the end until the line fits.
			for n := len(hints) - 1; n > 0 && lipgloss.Width(hv) > room; n-- {
				hv = h.ShortHelpView(hints[:n])
			}
			left = join(left, ansi.Truncate(hv, room, "…"))
		}
	}
	if api == "" {
		return left
	}
	pad := m.w - lipgloss.Width(left) - lipgloss.Width(api)
	return left + strings.Repeat(" ", max(pad, 1)) + api
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

// header builds the first line of the list view. The filter segment takes
// prio 3, the rate warning and the unavailable count 6.
func (m Model) header() string {
	count := strconv.Itoa(len(m.rows)) + " PRs"
	if len(m.rows) != len(m.snap.PRs) {
		// Filters hide some PRs: say so, so that a filter never looks like
		// data loss.
		count = strconv.Itoa(len(m.rows)) + " of " + strconv.Itoa(len(m.snap.PRs)) + " PRs"
	}
	segs := []segment{
		{"prv", 1},
		{"@" + m.snap.Viewer, 2},
		{m.filterLabel(), 3},
		{count, 4},
	}
	// D-15: an integrity note. It counts the unavailable rows of the whole
	// snapshot, so a filter never hides the note. On a tie with the rate warning
	// the leftmost segment is dropped first, and this one sits left of it.
	if n := m.unavailable(); n > 0 {
		segs = append(segs, segment{warnStyle.Render(strconv.Itoa(n) + " unavailable"), 6})
	}
	return fit(append(segs, m.statusSegments()...), " · ", m.w)
}

// unavailable is how many PRs of the snapshot could not be loaded (LIST-09).
func (m Model) unavailable() int {
	n := 0
	for _, pr := range m.snap.PRs {
		if pr.Unavail.Kind != domain.UnavailNone {
			n++
		}
	}
	return n
}

// statusSegments are the header segments that tell how fresh the data is: the
// update time or the failed refresh, the rate-limit pause and the rate
// warning. The list header and the detail header share them.
func (m Model) statusSegments() []segment {
	var segs []segment
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
	if m.now().Before(m.pausedUntil) {
		// D-08: a pause always says when refreshing resumes.
		segs = append(segs, segment{warnStyle.Render("paused until " + m.clock(m.pausedUntil, "15:04") + " (rate limit)"), 7})
	}
	// D-06: below 20% of the limit the budget moves up into the header.
	if rl := m.snap.RateLimit; rl.Limit > 0 && rl.Remaining*5 < rl.Limit {
		segs = append(segs, segment{warnStyle.Render("api " + strconv.Itoa(rl.Remaining) + "/" +
			strconv.Itoa(rl.Limit) + " · resets " + m.clock(rl.ResetAt, "15:04")), 6})
	}
	return segs
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
	} else if m.now().Before(m.pausedUntil) {
		b.WriteString("\n\npaused until " + m.clock(m.pausedUntil, "15:04") + " (rate limit)")
	} else if !m.nextAt.IsZero() {
		b.WriteString("\n\nretrying at " + m.clock(m.nextAt, "15:04"))
	}
	if w := m.warning(); w != "" {
		b.WriteString("\n\n" + warnStyle.Render(w))
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
	headers := [colCount]string{colPR: "PR", colTitle: "TITLE", colAuthor: "AUTHOR", colState: "STATE", colReview: "REVIEW", colReviewers: "REVIEWERS", colMerge: "MERGE", colCI: "CI"}
	cells := make([][colCount]string, 0, len(m.rows))
	var sz sizes
	for c, h := range headers {
		sz.natural[c] = lipgloss.Width(h)
	}
	sz.natural[colMarker] = 1
	minCI := lipgloss.Width(headers[colCI])
	// markW is the width of the "+ " marker of a saved PR (D-05): 2 when any row
	// is saved, else 0, so a list with no saved PR is drawn as before.
	markW := 0
	for _, pr := range m.rows {
		if pr.Manual {
			markW = 2
			break
		}
	}
	prefix := func(pr domain.PR) string {
		switch {
		case pr.Manual:
			return "+ "
		case markW > 0:
			return "  "
		}
		return ""
	}
	sz.collapsed[colReviewers] = lipgloss.Width("REVS")
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
		row := [colCount]string{
			colMarker: marker, colPR: prefix(pr) + pr.Ref.String(), colTitle: title, colAuthor: author, colState: pr.State.String(),
			colReview: pr.Review.Badge(), colReviewers: reviewersCell(pr.Reviewers, m.snap.Viewer, 0),
			colMerge: pr.Merge.Badge(), colCI: ciCell(pr, maxCIColW),
		}
		if unavail(pr) {
			// D-09: only the marker and the ref; the reason is drawn across the
			// other columns and never widens one.
			row = [colCount]string{colMarker: marker, colPR: row[colPR]}
		}
		cells = append(cells, row)
		for c, text := range row {
			sz.natural[c] = max(sz.natural[c], lipgloss.Width(text))
		}
		if unavail(pr) {
			continue
		}
		// The CI cell can shrink to its bare badge and no further.
		minCI = max(minCI, lipgloss.Width(pr.CI.Badge()))
		sz.collapsed[colReviewers] = max(sz.collapsed[colReviewers], lipgloss.Width(reviewersCount(pr.Reviewers)))
	}
	sz.natural[colPR] = min(sz.natural[colPR], maxPRColW)
	sz.natural[colAuthor] = min(sz.natural[colAuthor], maxAuthorCol)
	sz.natural[colReviewers] = min(sz.natural[colReviewers], maxReviewersW)
	sz.natural[colCI] = min(sz.natural[colCI], maxCIColW)
	sz.min = sz.natural
	sz.min[colCI] = min(minCI, sz.natural[colCI])

	lay := layoutColumns(sz, m.w)
	if lay.compact {
		return m.renderCompact()
	}
	var vis []int // the shown columns, in display order
	for c := range lay.show {
		if lay.show[c] {
			vis = append(vis, c)
		}
	}
	shownHeaders := make([]string, len(vis))
	for i, c := range vis {
		h := headers[c]
		if lay.collapsed[c] {
			h = "REVS" // only REVIEWERS collapses
		}
		shownHeaders[i] = truncateCell(h, lay.width[c])
	}
	shownRows := make([][]string, len(cells))
	for r, pr := range m.rows {
		shownRows[r] = make([]string, len(vis))
		for i, c := range vis {
			text := cells[r][c]
			if unavail(pr) && c != colMarker && c != colPR {
				continue // blank: unavailLine draws the row
			}
			switch c {
			case colPR:
				text = prefix(pr) + trimLeft(pr.Ref.String(), max(lay.width[c]-markW, 1))
			case colReviewers:
				if lay.collapsed[c] {
					text = reviewersCount(pr.Reviewers)
				} else {
					text = reviewersCell(pr.Reviewers, m.snap.Viewer, lay.width[c])
				}
			case colCI:
				text = ciCell(pr, lay.width[c])
			default:
				text = truncateCell(text, lay.width[c])
			}
			shownRows[r][i] = text
		}
	}

	// Only the visible slice goes to the table. The table's own Height and
	// YOffset are not used: they add an overflow row and shift StyleFunc rows.
	end := min(len(m.rows), m.offset+m.visibleRows())
	t := table.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).BorderBottom(false).BorderLeft(false).BorderRight(false).
		BorderColumn(false).BorderHeader(false).BorderRow(false).
		Headers(shownHeaders...).
		Rows(shownRows[m.offset:end]...).
		StyleFunc(func(row, col int) lipgloss.Style {
			c := vis[col]
			// Width includes padding in Lip Gloss v2: every column but the
			// last shown one carries one cell of gap.
			s := lipgloss.NewStyle().Width(lay.width[c])
			if col < len(vis)-1 {
				s = s.Width(lay.width[c] + columnGap).PaddingRight(columnGap)
			}
			if row == table.HeaderRow {
				return s.Inherit(headerStyle)
			}
			idx := m.offset + row
			pr := m.rows[idx]
			if dimmed(pr) {
				s = s.Faint(true)
			}
			switch c {
			case colReview:
				s = s.Inherit(reviewStyle(pr.Review))
			case colMerge:
				s = s.Inherit(mergeStyle(pr.Merge))
			case colCI:
				s = s.Inherit(ciStyle(pr.CI))
			}
			if idx == m.cursor {
				s = s.Inherit(selectedStyle)
			}
			return s
		})

	tableLines := strings.Split(t.String(), "\n")
	// The rows on screen are m.rows[m.offset:end], one table line each after the
	// column-header line. An unavailable row's line is drawn by hand (D-09).
	headerW := lipgloss.Width(tableLines[0])
	for idx := m.offset; idx < end; idx++ {
		if j := 1 + idx - m.offset; unavail(m.rows[idx]) && j < len(tableLines) {
			pr := m.rows[idx]
			prCell := prefix(pr) + trimLeft(pr.Ref.String(), max(lay.width[colPR]-markW, 1))
			tableLines[j] = m.unavailLine(pr, idx == m.cursor, cells[idx][colMarker], prCell, lay, headerW)
		}
	}
	return m.stack(tableLines[0], tableLines[1:])
}

// unavail reports whether pr is a saved PR that could not be loaded.
func unavail(pr domain.PR) bool { return pr.Unavail.Kind != domain.UnavailNone }

// padCells pads s with spaces to w terminal cells; a wider s is returned as is.
func padCells(s string, w int) string {
	if n := w - lipgloss.Width(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// unavailLine draws the line of an unavailable row (D-09): the cursor marker and
// the PR cell as in every other row, then "⚠ unavailable: REASON" across the
// space of the TITLE..CI columns in the warning colour, never faint. The reason
// is cut to that space with an ellipsis. Each segment is styled on its own, so
// text with escape codes is never wrapped in a style; widths are display cells.
// headerW is the width of the column-header line.
func (m Model) unavailLine(pr domain.PR, selected bool, marker, prCell string, lay layout, headerW int) string {
	base := lipgloss.NewStyle()
	if selected {
		base = selectedStyle
	}
	head := padCells(marker, lay.width[colMarker]+columnGap) + padCells(prCell, lay.width[colPR]+columnGap)
	out := base.Render(head)
	if rest := headerW - lipgloss.Width(head); rest > 0 {
		reason := ansi.Truncate("⚠ unavailable: "+pr.Unavail.Reason, rest, "…")
		out += base.Inherit(warnStyle).Render(padCells(reason, rest))
	}
	return out
}

// stack joins the header line, the column-header line, the rows and the
// bottom line of a list screen. A pane shorter than that drops the
// column-header line first, then the bottom line, then the header line, so
// the selected row is always on screen and the output never has more lines
// than the pane. Every line is cut to the pane width.
func (m Model) stack(colHeader string, rows []string) string {
	showColHeader, showBottom, showHeader := true, true, true
	if excess := 3 + len(rows) - m.h; m.h > 0 && excess > 0 {
		showColHeader = false
		if excess > 1 {
			showBottom = false
		}
		if excess > 2 {
			showHeader = false
		}
	}
	var lines []string
	if showHeader {
		lines = append(lines, m.header())
	}
	if showColHeader {
		lines = append(lines, colHeader)
	}
	lines = append(lines, rows...)
	if showBottom {
		lines = append(lines, m.bottom())
	}
	for i, l := range lines {
		lines[i] = m.cut(l)
	}
	return strings.Join(lines, "\n")
}

// shortRef is a PR's ref without the owner: the repository name, then "#" and
// the number.
func shortRef(r domain.Ref) string {
	repo := r.Repo
	if i := strings.LastIndex(repo, "/"); i >= 0 {
		repo = repo[i+1:]
	}
	return repo + "#" + strconv.Itoa(r.Number)
}

// renderCompact is the list of a pane under compactBelow cells (D-14): a
// marker, the short ref and the REVIEW, MERGE and CI glyphs on every row. The
// column-header line carries the hint to widen the pane.
func (m Model) renderCompact() string {
	end := min(len(m.rows), m.offset+m.visibleRows())
	rows := make([]string, 0, end-m.offset)
	for i := m.offset; i < end; i++ {
		rows = append(rows, m.compactRow(i))
	}
	return m.stack(headerStyle.Render("  PR")+faintStyle.Render("  widen for columns"), rows)
}

// compactRow draws row i as "> + ref ✓✓✓". A saved PR has "+ " after the cursor
// marker, and an unavailable one has a single warning glyph instead of the
// three (D-05, D-09). Each segment is styled on its own and the selected row's
// segments each carry the reverse style, so a style never wraps text that
// already holds escape codes. A ref that does not fit is cut from the left and
// keeps its number; when there is no room for even one cell of it, only the
// marker, the "+ " and the glyphs are drawn.
func (m Model) compactRow(i int) string {
	pr := m.rows[i]
	marker := "  "
	if i == m.cursor {
		marker = "> "
	}
	if pr.Manual {
		marker += "+ "
	}
	review, merge, ci := pr.Review.Glyph(), pr.Merge.Glyph(), pr.CI.Glyph()
	glyphW := lipgloss.Width(review) + lipgloss.Width(merge) + lipgloss.Width(ci)
	if unavail(pr) {
		glyphW = lipgloss.Width("⚠")
	}
	ref := shortRef(pr.Ref)
	if budget := m.w - lipgloss.Width(marker) - 1 - glyphW; m.w > 0 {
		if budget < 1 {
			ref = ""
		} else {
			ref = trimLeft(ref, budget)
		}
	}
	text := marker
	if ref != "" {
		text += ref + " "
	}
	base := lipgloss.NewStyle()
	if dimmed(pr) {
		base = base.Faint(true)
	}
	if i == m.cursor {
		base = base.Inherit(selectedStyle)
	}
	if unavail(pr) {
		return m.cut(base.Render(text) + base.Inherit(warnStyle).Render("⚠"))
	}
	return m.cut(base.Render(text) +
		base.Inherit(reviewStyle(pr.Review)).Render(review) +
		base.Inherit(mergeStyle(pr.Merge)).Render(merge) +
		base.Inherit(ciStyle(pr.CI)).Render(ci))
}

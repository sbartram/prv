package ui

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Table columns, in display order (D-02).
const (
	colMarker = iota
	colPR
	colTitle
	colAuthor
	colState
	colReview
	colCI
	colCount
)

const (
	columnGap    = 1  // spaces between columns
	minTitleW    = 10 // TITLE never gets fewer cells than this
	maxPRColW    = 40 // cap for the PR column
	maxAuthorCol = 16 // cap for the AUTHOR column
	minPRColW    = 8  // the PR column is never cut below this
	// chromeLines are the lines around the rows: the "prv · @login" line, the
	// table's column-header row and the footer line.
	chromeLines = 3
)

// columnWidths returns the width in terminal cells of each column. cells holds
// every row, header included, so each fixed column gets the widest of its
// header and its cells (PR and AUTHOR are capped). TITLE is the flexible
// column (D-02): it gets what is left of total after the fixed columns and the
// gaps, at least minTitleW and at most its widest cell. When total is 0 (the
// terminal has not reported its size yet) TITLE gets its widest cell.
// Widths are cells, never bytes or runes.
func columnWidths(cells [][]string, total int) []int {
	ws := make([]int, colCount)
	for _, row := range cells {
		for c := 0; c < colCount && c < len(row); c++ {
			ws[c] = max(ws[c], lipgloss.Width(row[c]))
		}
	}
	ws[colPR] = min(ws[colPR], maxPRColW)
	ws[colAuthor] = min(ws[colAuthor], maxAuthorCol)
	if total <= 0 {
		return ws
	}
	avail := total - (colCount-1)*columnGap
	for c, w := range ws {
		if c != colTitle {
			avail -= w
		}
	}
	ws[colTitle] = max(minTitleW, min(ws[colTitle], avail))
	return ws
}

// truncateCell cuts s to w terminal cells, ending in "…", when it is wider.
func truncateCell(s string, w int) string {
	if lipgloss.Width(s) > w {
		return ansi.Truncate(s, w, "…")
	}
	return s
}

// visibleRows is how many PR rows fit: the height minus the lines around
// them, and at least 1. Before the first size message every row is visible.
func (m Model) visibleRows() int {
	if m.h <= 0 {
		return len(m.rows)
	}
	return max(1, m.h-chromeLines)
}

// clampOffset keeps the cursor inside [offset, offset+visibleRows()).
func (m *Model) clampOffset() {
	v := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+v {
		m.offset = m.cursor - v + 1
	}
	m.offset = max(0, min(m.offset, len(m.rows)-v))
}

// sizes holds what layoutColumns needs to know about each column: the width it
// wants and the narrowest it may get before the layout gives up on it.
type sizes struct{ natural, min [colCount]int }

// layout says which columns are shown and how wide each one is.
type layout struct {
	compact bool
	show    [colCount]bool
	width   [colCount]int
}

// dropStep is one step of the D-13 drop order.
type dropStep struct {
	col      int
	collapse bool
}

// dropSteps is the D-13 order in which a narrowing pane loses columns.
var dropSteps = []dropStep{{col: colAuthor}, {col: colState}, {col: colTitle}}

// layoutColumns is a stub (RED).
func layoutColumns(sz sizes, total int) layout { return layout{} }

// trimLeft is a stub (RED).
func trimLeft(s string, w int) string { return s }

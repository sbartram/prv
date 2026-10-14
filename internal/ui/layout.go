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
	colReviewers
	colMerge
	colCI
	colCount
)

const (
	columnGap    = 1  // spaces between columns
	minTitleW    = 10 // TITLE never gets fewer cells than this
	maxPRColW    = 40 // cap for the PR column
	maxAuthorCol = 16 // cap for the AUTHOR column
	// maxReviewersW caps the REVIEWERS column; a longer list ends in "+N".
	maxReviewersW = 22
	minPRColW     = 8 // the PR column is never cut below this
	// compactBelow is the pane width under which the list shows compact rows
	// (D-14).
	compactBelow = 40
	// chromeLines are the lines around the rows: the "prv · @login" line, the
	// table's column-header row and the footer line.
	chromeLines = 3
)

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
type sizes struct{ natural, min, collapsed [colCount]int }

// layout says which columns are shown and how wide each one is.
type layout struct {
	compact bool
	show    [colCount]bool
	width   [colCount]int
	// collapsed marks a column shown in its count-only form.
	collapsed [colCount]bool
}

// dropStep is one step of the D-13 drop order.
type dropStep struct {
	col      int
	collapse bool
}

// dropSteps is the D-13 order in which a narrowing pane loses columns. A
// collapse step folds a column into its count-only form first, so REVIEWERS
// shrinks to "+N" before it goes.
var dropSteps = []dropStep{
	{col: colAuthor}, {col: colState},
	{col: colReviewers, collapse: true}, {col: colReviewers},
	{col: colMerge}, {col: colTitle},
}

// layoutColumns decides which columns are shown and how wide each one is
// for a pane total cells wide. A total of 0 or less means the pane has not
// reported its size yet: every column is shown uncut and the list is never
// compact (D-14 revised). Below compactBelow cells the layout is compact: no
// columns, the caller draws compact rows.
//
// The pane narrows in the D-13 order. TITLE is the flexible column: it gets
// what is left, down to minTitleW cells. When it cannot keep that, the next
// step of dropSteps (a column dropped or collapsed) is taken and TITLE gets
// another try; once TITLE
// itself is dropped, the CI cell narrows to its bare badge and then the PR
// column narrows (never below minPRColW). A column never comes back at a
// smaller width, because each step only adds to the set of dropped columns.
// Widths are terminal cells.
func layoutColumns(sz sizes, total int) layout {
	var l layout
	for c := range l.show {
		l.show[c] = true
		l.width[c] = sz.natural[c]
	}
	l.width[colMarker] = 1
	if total <= 0 {
		return l
	}
	if total < compactBelow {
		return layout{compact: true}
	}
	// need is the width of the shown columns other than TITLE, gaps included.
	need := func() int {
		sum, n := 0, 0
		for c := range l.show {
			if !l.show[c] {
				continue
			}
			n++
			if c != colTitle {
				sum += l.width[c]
			}
		}
		return sum + columnGap*(n-1)
	}
	for k := 0; ; k++ {
		if l.show[colTitle] {
			if n := need(); n+min(sz.natural[colTitle], minTitleW) <= total {
				l.width[colTitle] = min(sz.natural[colTitle], total-n)
				return l
			}
		} else if need() <= total {
			return l
		}
		if k >= len(dropSteps) {
			break
		}
		if st := dropSteps[k]; st.collapse {
			l.width[st.col] = sz.collapsed[st.col]
			l.collapsed[st.col] = true
		} else {
			l.show[st.col] = false
			l.width[st.col] = 0
			l.collapsed[st.col] = false
		}
	}
	// Even PR, REVIEW and CI at their natural widths do not fit. The CI cell
	// drops its check name first, then the PR ref is cut from the left.
	l.width[colCI] = max(sz.min[colCI], total-(need()-l.width[colCI]))
	if n := need(); n > total {
		l.width[colPR] = max(minPRColW, total-(n-l.width[colPR]))
	}
	return l
}

// trimLeft cuts s to w terminal cells from the left, putting "…" in front,
// so that the end of a PR ref (its "#number") stays. A string that fits is
// returned unchanged; for w of 1 or less there is no room for the "…".
func trimLeft(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	if w <= 1 {
		return ansi.Truncate(s, max(w, 0), "")
	}
	return "…" + ansi.TruncateLeft(s, lipgloss.Width(s)-(w-1), "")
}

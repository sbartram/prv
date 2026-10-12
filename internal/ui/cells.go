package ui

import (
	"strconv"

	"charm.land/lipgloss/v2"

	"github.com/sbartram/prv/internal/domain"
)

// maxCIColW is the widest the CI cell gets, in terminal cells.
const maxCIColW = 24

// ciCell is the text of a row's CI cell. A failing PR whose failing check is
// known shows "✗ NAME", then " (cancelled)" when only cancelled checks are
// left, then " +N" when N other checks fail (LIST-07). Every other PR, and a
// failing PR without a known name, shows the plain badge. The colour and the
// attention bucket come from the rollup state, never from this text.
//
// A text wider than w cells (w > 0) is replaced by the plain badge cut to w.
// Widths are cells, never bytes or runes.
func ciCell(pr domain.PR, w int) string {
	if !pr.Failing.Known {
		return pr.CI.Badge()
	}
	text := "✗ " + pr.Failing.Name
	if pr.Failing.Cancelled {
		text += " (cancelled)"
	}
	if pr.Failing.More > 0 {
		text += " +" + strconv.Itoa(pr.Failing.More)
	}
	if w <= 0 || lipgloss.Width(text) <= w {
		return text
	}
	return truncateCell(pr.CI.Badge(), w)
}

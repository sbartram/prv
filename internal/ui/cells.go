package ui

import (
	"strconv"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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
// A text wider than w cells (w > 0) is shortened in the name only: the
// " (cancelled)" and " +N" suffixes stay whole. When not even 2 cells of the
// name fit, the plain badge cut to w is shown instead. Widths are cells, never
// bytes or runes.
func ciCell(pr domain.PR, w int) string {
	if !pr.Failing.Known {
		return pr.CI.Badge()
	}
	const prefix = "✗ "
	suffix := ""
	if pr.Failing.Cancelled {
		suffix += " (cancelled)"
	}
	if pr.Failing.More > 0 {
		suffix += " +" + strconv.Itoa(pr.Failing.More)
	}
	text := prefix + pr.Failing.Name + suffix
	if w <= 0 || lipgloss.Width(text) <= w {
		return text
	}
	room := w - lipgloss.Width(prefix) - lipgloss.Width(suffix)
	if room < 2 {
		return truncateCell(pr.CI.Badge(), w)
	}
	return prefix + ansi.Truncate(pr.Failing.Name, room, "…") + suffix
}

// reviewersCell is a RED stub, replaced in GREEN.
func reviewersCell(rs []domain.Reviewer, viewer string, w int) string { return "" }

// reviewersCount is a RED stub, replaced in GREEN.
func reviewersCount(rs []domain.Reviewer) string { return "" }

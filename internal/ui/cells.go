package ui

import (
	"strconv"
	"strings"

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

// reviewersCell is the text of a row's REVIEWERS cell: one token per reviewer
// in slice order (already changes requested, pending, approved), each the
// state glyph glued to the name, joined by single spaces (LIST-04, D-06). The
// viewer shows as "me", a team as "@org/slug", and all hidden requests (teams
// the viewer cannot see) make one token at the place the first one sorts:
// "…team" for one, "…N teams" for more.
//
// A text wider than w cells (w > 0) keeps the longest run of leading tokens
// that fits together with " +K", K counting every reviewer not shown (each
// hidden request counts), so the cell never drops a reviewer without saying
// so. When not even one token fits, the cell is "+N" for all N reviewers, cut
// to w if it is still too wide. There is no colour: the glyphs carry the state
// and a styled run inside a cell would break the selected-row style.
func reviewersCell(rs []domain.Reviewer, viewer string, w int) string {
	if len(rs) == 0 {
		return ""
	}
	type token struct {
		text string
		n    int // reviewers the token stands for
	}
	var toks []token
	hidden, hiddenAt := 0, -1
	hiddenGlyph := ""
	for _, r := range rs {
		switch {
		case r.Kind == domain.ReviewerHidden:
			if hidden == 0 {
				hiddenAt, hiddenGlyph = len(toks), r.State.Glyph()
				toks = append(toks, token{})
			}
			hidden++
		case r.Kind == domain.ReviewerTeam:
			toks = append(toks, token{r.State.Glyph() + "@" + r.Name, 1})
		case viewer != "" && strings.EqualFold(r.Name, viewer):
			toks = append(toks, token{r.State.Glyph() + "me", 1})
		default:
			toks = append(toks, token{r.State.Glyph() + r.Name, 1})
		}
	}
	if hidden > 0 {
		text := hiddenGlyph + "team"
		if hidden > 1 {
			text = hiddenGlyph + strconv.Itoa(hidden) + " teams"
		}
		toks[hiddenAt] = token{text, hidden}
	}
	join := func(k int) string {
		texts := make([]string, k)
		for i, t := range toks[:k] {
			texts[i] = t.text
		}
		return strings.Join(texts, " ")
	}
	full := join(len(toks))
	if w <= 0 || lipgloss.Width(full) <= w {
		return full
	}
	// Try the longest prefix first; rest counts the reviewers after it.
	rest := 0
	for k := len(toks) - 1; k >= 1; k-- {
		rest += toks[k].n
		if cand := join(k) + " +" + strconv.Itoa(rest); lipgloss.Width(cand) <= w {
			return cand
		}
	}
	return truncateCell("+"+strconv.Itoa(len(rs)), w)
}

// reviewersCount is the REVIEWERS cell of a collapsed column: "+N" for N
// reviewers, empty for none.
func reviewersCount(rs []domain.Reviewer) string {
	if len(rs) == 0 {
		return ""
	}
	return "+" + strconv.Itoa(len(rs))
}

package ui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"

	"github.com/sbartram/prv/internal/domain"
)

// Only the terminal's own 16 ANSI colours are used (D-05), so the palette
// follows the user's theme and reads on light and dark backgrounds and inside
// multiplexer panes. Status never relies on colour alone: every badge also
// carries a glyph and a word.
var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	faintStyle    = lipgloss.NewStyle().Faint(true)
	// warnStyle marks a failed refresh in the header (D-07). The glyph and the
	// words keep it readable without colour.
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Yellow)
)

// reviewStyle colours the REVIEW badge.
func reviewStyle(r domain.Review) lipgloss.Style {
	switch r {
	case domain.ReviewApproved:
		return lipgloss.NewStyle().Foreground(lipgloss.Green)
	case domain.ReviewChangesRequested:
		return lipgloss.NewStyle().Foreground(lipgloss.Red)
	case domain.ReviewRequired:
		return lipgloss.NewStyle().Foreground(lipgloss.Yellow)
	}
	return faintStyle
}

// ciStyle colours the CI badge.
func ciStyle(c domain.CI) lipgloss.Style {
	switch c {
	case domain.CIPassing:
		return lipgloss.NewStyle().Foreground(lipgloss.Green)
	case domain.CIFailing:
		return lipgloss.NewStyle().Foreground(lipgloss.Red)
	case domain.CIRunning:
		return lipgloss.NewStyle().Foreground(lipgloss.Yellow)
	}
	return faintStyle
}

// dimmed reports whether the whole row is dimmed: drafts (D-04) and merged or
// closed PRs (D-07). An unknown state is not dimmed.
func dimmed(pr domain.PR) bool {
	return pr.IsDraft || pr.State == domain.StateMerged || pr.State == domain.StateClosed
}

// helpStyles restyles the footer key hints in ANSI-16 only. help.DefaultStyles
// and its dark and light variants use hex colours, which break the D-05 rule.
func helpStyles() help.Styles {
	return help.Styles{
		ShortKey:       lipgloss.NewStyle(),
		ShortDesc:      faintStyle,
		ShortSeparator: faintStyle,
		Ellipsis:       faintStyle,
		FullKey:        lipgloss.NewStyle(),
		FullDesc:       faintStyle,
		FullSeparator:  faintStyle,
	}
}

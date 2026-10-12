package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/sbartram/prv/internal/domain"
)

// RED stubs: replaced by the real styles in the GREEN commit.

func reviewStyle(domain.Review) lipgloss.Style { return lipgloss.NewStyle() }

func ciStyle(domain.CI) lipgloss.Style { return lipgloss.NewStyle() }

func dimmed(domain.PR) bool { return false }

var (
	headerStyle   = lipgloss.NewStyle()
	selectedStyle = lipgloss.NewStyle()
)

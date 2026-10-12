package ui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/sbartram/prv/internal/domain"
)

// newTestModel returns a model sized w x h that has loaded snap.
func newTestModel(t *testing.T, snap domain.Snapshot, w, h int) Model {
	t.Helper()
	var m tea.Model = New(t.Context(), &fakeFetcher{}, Options{ClosedDays: 14})
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m, _ = m.Update(fetchedMsg{snap: snap})
	return m.(Model)
}

// fixtureSix has one PR for each combination the colour rules care about.
func fixtureSix() domain.Snapshot {
	at := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	return domain.Snapshot{Viewer: "octo", PRs: []domain.PR{
		{ID: "1", Ref: domain.Ref{Repo: "acme/api", Number: 101}, Title: "Add retry to the client", Author: "octo",
			State: domain.StateOpen, Review: domain.ReviewApproved, CI: domain.CIPassing, UpdatedAt: at},
		{ID: "2", Ref: domain.Ref{Repo: "acme/api", Number: 102}, Title: "Rework the cache layer", Author: "alice",
			State: domain.StateOpen, Review: domain.ReviewChangesRequested, CI: domain.CIFailing, UpdatedAt: at},
		{ID: "3", Ref: domain.Ref{Repo: "acme/web", Number: 7}, Title: "Spike a new sidebar", Author: "octo", IsDraft: true,
			State: domain.StateOpen, Review: domain.ReviewNone, CI: domain.CIRunning, UpdatedAt: at},
		{ID: "4", Ref: domain.Ref{Repo: "acme/web", Number: 6}, Title: "Fix the login redirect", Author: "bob",
			State: domain.StateMerged, Review: domain.ReviewApproved, CI: domain.CIPassing, UpdatedAt: at, ClosedAt: at},
		{ID: "5", Ref: domain.Ref{Repo: "acme/docs", Number: 3}, Title: "Drop the old guide", Author: "octo",
			State: domain.StateClosed, Review: domain.ReviewRequired, CI: domain.CINone, UpdatedAt: at, ClosedAt: at},
		{ID: "6", Ref: domain.Ref{Repo: "acme/docs", Number: 2}, Title: "Typo in the readme", Author: "carol",
			State: domain.StateOpen, Review: domain.ReviewNone, CI: domain.CIUnknown, UpdatedAt: at},
	}}
}

func TestGoldenListColor(t *testing.T) {
	m := newTestModel(t, fixtureSix(), 120, 40)
	golden.RequireEqual(t, m.View().Content)
}

func TestBadgesReadWithoutColour(t *testing.T) {
	m := newTestModel(t, fixtureSix(), 120, 40)
	v := ansi.Strip(m.View().Content)
	for _, want := range []string{
		"✓ approved", "✗ changes", "○ required", "– none",
		"✓ passing", "✗ failing", "● running", "– no checks", "? unknown",
		"[draft]",
	} {
		if !strings.Contains(v, want) {
			t.Errorf("stripped view missing %q:\n%s", want, v)
		}
	}
	// A draft keeps its true state (D-04).
	for _, line := range strings.Split(v, "\n") {
		if strings.Contains(line, "[draft]") && !strings.Contains(line, "open") {
			t.Errorf("draft row lost STATE open: %q", line)
		}
	}
}

func TestReviewAndCIColours(t *testing.T) {
	green, red, yellow := lipgloss.Green, lipgloss.Red, lipgloss.Yellow
	reviews := map[domain.Review]any{
		domain.ReviewApproved: green, domain.ReviewChangesRequested: red,
		domain.ReviewRequired: yellow, domain.ReviewNone: nil,
	}
	for r, want := range reviews {
		s := reviewStyle(r)
		if want == nil {
			if !s.GetFaint() {
				t.Errorf("reviewStyle(%v) should be faint", r)
			}
			continue
		}
		if s.GetForeground() != want {
			t.Errorf("reviewStyle(%v) foreground = %v, want %v", r, s.GetForeground(), want)
		}
	}
	cis := map[domain.CI]any{
		domain.CIPassing: green, domain.CIFailing: red, domain.CIRunning: yellow,
		domain.CINone: nil, domain.CIUnknown: nil,
	}
	for c, want := range cis {
		s := ciStyle(c)
		if want == nil {
			if !s.GetFaint() {
				t.Errorf("ciStyle(%v) should be faint", c)
			}
			continue
		}
		if s.GetForeground() != want {
			t.Errorf("ciStyle(%v) foreground = %v, want %v", c, s.GetForeground(), want)
		}
	}
}

func TestDimmedRule(t *testing.T) {
	tests := []struct {
		name string
		pr   domain.PR
		want bool
	}{
		{"open", domain.PR{State: domain.StateOpen}, false},
		{"draft", domain.PR{State: domain.StateOpen, IsDraft: true}, true},
		{"merged", domain.PR{State: domain.StateMerged}, true},
		{"closed", domain.PR{State: domain.StateClosed}, true},
		{"unknown", domain.PR{State: domain.StateUnknown}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dimmed(tt.pr); got != tt.want {
				t.Errorf("dimmed = %v, want %v", got, tt.want)
			}
		})
	}
}

var reverseSeq = regexp.MustCompile(`\x1b\[(?:[0-9;]*;)?7(?:;[0-9;]*)?m`)

func TestSelectionNotColourOnly(t *testing.T) {
	m := newTestModel(t, fixtureSix(), 120, 40)
	raw := strings.Split(m.View().Content, "\n")
	var selected []string
	for _, line := range raw {
		if strings.HasPrefix(ansi.Strip(line), ">") {
			selected = append(selected, line)
		}
	}
	if len(selected) != 1 {
		t.Fatalf("%d selected lines, want exactly 1", len(selected))
	}
	if !reverseSeq.MatchString(selected[0]) {
		t.Errorf("selected line has no reverse-video sequence: %q", selected[0])
	}
	for _, line := range raw {
		if !strings.HasPrefix(ansi.Strip(line), ">") && strings.Contains(ansi.Strip(line), "acme/") && reverseSeq.MatchString(line) {
			t.Errorf("an unselected row is reversed: %q", line)
		}
	}
	// Moving the cursor moves the marker.
	var tm tea.Model = m
	tm, _ = tm.Update(key("j"))
	for _, line := range strings.Split(ansi.Strip(tm.View().Content), "\n") {
		if strings.HasPrefix(line, ">") && !strings.Contains(line, "acme/api#102") {
			t.Errorf("after j the marker is on %q", line)
		}
	}
}

var (
	rgbFg = regexp.MustCompile(`38;2;`)
	rgbBg = regexp.MustCompile(`48;2;`)
)

func TestStylesUseAnsi16(t *testing.T) {
	m := newTestModel(t, fixtureSix(), 120, 40)
	raw := m.View().Content
	if rgbFg.MatchString(raw) || rgbBg.MatchString(raw) {
		t.Errorf("view holds a 24-bit colour sequence: %q", raw)
	}
	if !strings.Contains(raw, "\x1b[") {
		t.Error("view holds no styling at all")
	}
	// 256-colour sequences are not ANSI-16 either.
	if strings.Contains(raw, "38;5;") || strings.Contains(raw, "48;5;") {
		t.Errorf("view holds a 256-colour sequence: %q", raw)
	}
}

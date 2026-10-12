package ui

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/sbartram/prv/internal/domain"
)

// fixedNow is the clock of the view tests, so goldens do not depend on the
// machine or the date.
func fixedNow() time.Time { return time.Date(2026, 10, 11, 14, 32, 5, 0, time.UTC) }

// newTestModel returns a model sized w x h that has loaded snap.
func newTestModel(t *testing.T, snap domain.Snapshot, w, h int) Model {
	t.Helper()
	var m tea.Model = New(t.Context(), &fakeFetcher{}, Options{ClosedDays: 14, Now: fixedNow})
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

// ---- layout, scrolling and golden tests (plan 01-03 task 2) ----

// fixtureThirty has 30 PRs, including a 120-character title, a CJK title that
// is 24 cells wide, and a title with an emoji ZWJ sequence.
func fixtureThirty() domain.Snapshot {
	at := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	snap := domain.Snapshot{Viewer: "octo"}
	for i := 1; i <= 30; i++ {
		pr := domain.PR{
			ID: "N" + strings.Repeat("x", i%3), Ref: domain.Ref{Repo: "acme/api", Number: i},
			Title: "Change number " + string(rune('A'+i%26)), Author: "alice", UpdatedAt: at,
			State: domain.StateOpen, Review: domain.ReviewApproved, CI: domain.CIPassing,
		}
		switch i {
		case 5:
			pr.Title = strings.Repeat("long title ", 11) + "end!" // 120 characters
		case 7:
			pr.Title = "数据迁移工具修复问题测试"
		case 9:
			pr.Title = "Family 👨‍👩‍👧‍👦 sync and a long tail of words after it"
		case 11:
			pr.IsDraft = true
		case 12:
			pr.State, pr.ClosedAt = domain.StateMerged, at
		}
		snap.PRs = append(snap.PRs, pr)
	}
	return snap
}

// hintedError is the 401-style error of the error panel golden.
func hintedError() error {
	return &hintErr{
		msg:  "GitHub request failed: HTTP 401: Bad credentials",
		hint: "Your token was rejected. Run `gh auth login`, or fix or unset GH_TOKEN and GITHUB_TOKEN if one of them is set.",
	}
}

func stateModel(t *testing.T, state string, w, h int) Model {
	t.Helper()
	switch state {
	case "loading":
		var m tea.Model = New(t.Context(), &fakeFetcher{}, Options{ClosedDays: 14, Now: fixedNow})
		m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
		return m.(Model)
	case "error":
		var m tea.Model = New(t.Context(), &fakeFetcher{}, Options{ClosedDays: 14, Now: fixedNow})
		m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
		m, _ = m.Update(fetchedMsg{err: hintedError()})
		return m.(Model)
	case "empty":
		return newTestModel(t, domain.Snapshot{Viewer: "octo"}, w, h)
	case "list":
		return newTestModel(t, fixtureThirty(), w, h)
	}
	t.Fatalf("unknown state %q", state)
	return Model{}
}

func TestGolden(t *testing.T) {
	sizes := [][2]int{{80, 24}, {120, 40}, {0, 0}}
	for _, state := range []string{"loading", "error", "empty", "list"} {
		for _, sz := range sizes {
			name := state + "_" + itoa(sz[0]) + "x" + itoa(sz[1])
			t.Run(name, func(t *testing.T) {
				m := stateModel(t, state, sz[0], sz[1])
				golden.RequireEqual(t, ansi.Strip(m.View().Content))
			})
		}
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func lines(s string) []string { return strings.Split(ansi.Strip(s), "\n") }

func TestTitleTruncatesFirst(t *testing.T) {
	snap := fixtureSix()
	long := strings.Repeat("0123456789", 12) // 120 characters
	snap.PRs = append(snap.PRs, domain.PR{
		ID: "7", Ref: domain.Ref{Repo: "acme/api", Number: 999}, Title: long, Author: "dave",
		State: domain.StateOpen, Review: domain.ReviewApproved, CI: domain.CIPassing,
	})
	for _, w := range []int{80, 120} {
		m := newTestModel(t, snap, w, 40)
		ls := lines(m.View().Content)
		var header, row string
		for _, l := range ls {
			if strings.Contains(l, "TITLE") {
				header = l
			}
			if strings.Contains(l, "acme/api#999") {
				row = l
			}
		}
		if got := strings.Fields(header); strings.Join(got, ",") != "PR,TITLE,AUTHOR,STATE,REVIEW,CI" {
			t.Errorf("w=%d header = %q, want every column header whole", w, header)
		}
		if !strings.Contains(row, "…") {
			t.Errorf("w=%d long row is not cut with …: %q", w, row)
		}
		if !strings.Contains(row, "✓ approved") || !strings.Contains(row, "✓ passing") {
			t.Errorf("w=%d long row lost its badges (only TITLE may be cut): %q", w, row)
		}
	}
}

func TestTruncateBoundary(t *testing.T) {
	cells := [][]string{
		{"", "PR", "TITLE", "AUTHOR", "STATE", "REVIEW", "CI"},
		{">", "acme/api#1", strings.Repeat("x", 200), "alice", "open", "✓ approved", "✓ passing"},
	}
	const total = 80
	ws := columnWidths(cells, total)
	if len(ws) != 7 {
		t.Fatalf("columnWidths returned %d widths, want 7", len(ws))
	}
	w := ws[2]
	if w < 10 || w > total {
		t.Fatalf("TITLE width = %d", w)
	}
	sum := 6 // gaps
	for _, c := range ws {
		sum += c
	}
	if sum != total {
		t.Errorf("columns plus gaps = %d cells, want %d", sum, total)
	}
	exact := strings.Repeat("x", w)
	if got := truncateCell(exact, w); got != exact || strings.Contains(got, "…") {
		t.Errorf("title of exactly %d cells was cut: %q", w, got)
	}
	over := truncateCell(strings.Repeat("x", w+1), w)
	if !strings.HasSuffix(over, "…") || lipgloss.Width(over) != w {
		t.Errorf("title of %d cells = %q (width %d), want it to end in … and fit %d", w+1, over, lipgloss.Width(over), w)
	}
	// A narrow pane never takes TITLE below 10 cells.
	if got := columnWidths(cells, 30)[2]; got != 10 {
		t.Errorf("TITLE width in a 30-cell pane = %d, want the floor of 10", got)
	}
	// Not sized yet: TITLE gets its widest cell.
	if got := columnWidths(cells, 0)[2]; got != 200 {
		t.Errorf("TITLE width when not sized = %d, want 200", got)
	}
}

func TestCellWidth(t *testing.T) {
	const cjk = "数据迁移工具修复问题测试"
	if got := lipgloss.Width(cjk); got != 24 {
		t.Fatalf("CJK title measures %d cells, want 24", got)
	}
	cut := truncateCell(cjk, 11)
	if !utf8.ValidString(cut) || lipgloss.Width(cut) > 11 || !strings.HasSuffix(cut, "…") {
		t.Errorf("truncateCell(cjk, 11) = %q (width %d)", cut, lipgloss.Width(cut))
	}
	for _, w := range []int{80, 120} {
		m := newTestModel(t, fixtureThirty(), w, 40)
		for _, l := range lines(m.View().Content) {
			if !utf8.ValidString(l) {
				t.Errorf("w=%d invalid UTF-8 line %q", w, l)
			}
			if lipgloss.Width(l) > w {
				t.Errorf("w=%d line is %d cells wide: %q", w, lipgloss.Width(l), l)
			}
		}
	}
}

func TestScrollFollowsCursor(t *testing.T) {
	var tm tea.Model = newTestModel(t, fixtureThirty(), 80, 24)
	has := func(n int) bool {
		re := regexp.MustCompile(`acme/api#` + strconv.Itoa(n) + `\s`)
		return re.MatchString(ansi.Strip(tm.View().Content))
	}
	if !has(1) {
		t.Fatal("first PR is not visible at the start")
	}
	for range 25 {
		tm, _ = tm.Update(key("j"))
	}
	if !has(26) {
		t.Errorf("after 25 j the 26th PR is not visible:\n%s", ansi.Strip(tm.View().Content))
	}
	if has(1) {
		t.Error("after 25 j the first PR is still visible")
	}
	for _, l := range lines(tm.View().Content) {
		if strings.HasPrefix(l, ">") && !regexp.MustCompile(`acme/api#26\s`).MatchString(l) {
			t.Errorf("marker is on %q, want PR 26", l)
		}
	}
	if !strings.Contains(ansi.Strip(tm.View().Content), ">") {
		t.Error("no marker visible")
	}
	for range 25 {
		tm, _ = tm.Update(key("k"))
	}
	if !has(1) {
		t.Error("after 25 k the first PR is not visible again")
	}
	if has(26) {
		t.Error("after 25 k the 26th PR is still visible")
	}
}

func TestViewFitsScreen(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {120, 40}} {
		for _, state := range []string{"loading", "error", "empty", "list"} {
			t.Run(state+"_"+itoa(sz[0])+"x"+itoa(sz[1]), func(t *testing.T) {
				ls := lines(stateModel(t, state, sz[0], sz[1]).View().Content)
				if len(ls) > sz[1] {
					t.Errorf("%d lines, want at most %d", len(ls), sz[1])
				}
				for _, l := range ls {
					if lipgloss.Width(l) > sz[0] {
						t.Errorf("line is %d cells wide, want at most %d: %q", lipgloss.Width(l), sz[0], l)
					}
				}
			})
		}
	}
	t.Run("not sized does not panic", func(t *testing.T) {
		for _, state := range []string{"loading", "error", "empty", "list"} {
			_ = stateModel(t, state, 0, 0).View()
		}
	})
}

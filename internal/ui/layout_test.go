package ui

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sbartram/prv/internal/domain"
)

// testSizes are the column sizes of a typical list: TITLE wants 40 cells and
// the other columns 55 plus 7 gaps, so everything fits from 72 cells up.
func testSizes() sizes {
	var sz sizes
	sz.natural = [colCount]int{colMarker: 1, colPR: 12, colTitle: 40, colAuthor: 6, colState: 6, colReview: 10, colMerge: 11, colCI: 9}
	sz.min = sz.natural
	return sz
}

func shownCols(l layout) []int {
	var cols []int
	for c := range l.show {
		if l.show[c] {
			cols = append(cols, c)
		}
	}
	return cols
}

func TestLayoutColumns(t *testing.T) {
	all := []int{colMarker, colPR, colTitle, colAuthor, colState, colReview, colMerge, colCI}
	noAuthor := []int{colMarker, colPR, colTitle, colState, colReview, colMerge, colCI}
	noState := []int{colMarker, colPR, colTitle, colReview, colMerge, colCI}
	noMerge := []int{colMarker, colPR, colTitle, colReview, colCI}
	noTitle := []int{colMarker, colPR, colReview, colCI}

	shortTitle := testSizes()
	shortTitle.natural[colTitle], shortTitle.min[colTitle] = 7, 7

	bigPR := testSizes()
	bigPR.natural[colPR], bigPR.min[colPR] = 30, 30

	bigCI := testSizes()
	bigCI.natural[colPR], bigCI.min[colPR] = 40, 40
	bigCI.natural[colCI], bigCI.min[colCI] = 24, 11

	for _, tt := range []struct {
		name  string
		sz    sizes
		total int
		shown []int
		title int // wanted TITLE width, 0 when TITLE is not shown
		pr    int // wanted PR width
		ci    int // wanted CI width
	}{
		{"unsized keeps every column uncut", testSizes(), 0, all, 40, 12, 9},
		{"negative width is unsized", testSizes(), -5, all, 40, 12, 9},
		{"room for everything", testSizes(), 200, all, 40, 12, 9},
		{"TITLE takes what is left", testSizes(), 82, all, 20, 12, 9},
		{"TITLE at its minimum", testSizes(), 72, all, 10, 12, 9},
		{"AUTHOR drops first", testSizes(), 71, noAuthor, 16, 12, 9},
		{"AUTHOR gone, TITLE at its minimum", testSizes(), 65, noAuthor, 10, 12, 9},
		{"STATE drops second", testSizes(), 64, noState, 16, 12, 9},
		{"STATE gone, TITLE at its minimum", testSizes(), 58, noState, 10, 12, 9},
		{"MERGE drops third", testSizes(), 57, noMerge, 21, 12, 9},
		{"MERGE gone, TITLE at its minimum", testSizes(), 46, noMerge, 10, 12, 9},
		{"TITLE drops last", testSizes(), 45, noTitle, 0, 12, 9},
		{"only the four columns left, exactly fitting", bigPR, 53, noTitle, 0, 30, 9},
		{"a short TITLE does not force a drop", shortTitle, 69, all, 7, 12, 9},
		{"a short TITLE still drops AUTHOR when it must", shortTitle, 68, noAuthor, 7, 12, 9},
		{"PR narrows when nothing else can", bigPR, 40, noTitle, 0, 17, 9},
		{"CI narrows to its minimum, then PR", bigCI, 40, noTitle, 0, 15, 11},
	} {
		t.Run(tt.name, func(t *testing.T) {
			l := layoutColumns(tt.sz, tt.total)
			if l.compact {
				t.Errorf("compact = true at %d, want false", tt.total)
			}
			if got := shownCols(l); !slices.Equal(got, tt.shown) {
				t.Fatalf("shown columns = %v, want %v", got, tt.shown)
			}
			if tt.title > 0 && l.width[colTitle] != tt.title {
				t.Errorf("TITLE width = %d, want %d", l.width[colTitle], tt.title)
			}
			if l.width[colPR] != tt.pr {
				t.Errorf("PR width = %d, want %d", l.width[colPR], tt.pr)
			}
			if l.width[colCI] != tt.ci {
				t.Errorf("CI width = %d, want %d", l.width[colCI], tt.ci)
			}
			if l.width[colMarker] != 1 {
				t.Errorf("marker width = %d, want 1", l.width[colMarker])
			}
			if tt.total > 0 {
				sum, n := 0, 0
				for _, c := range shownCols(l) {
					sum += l.width[c]
					n++
				}
				if sum+columnGap*(n-1) > tt.total {
					t.Errorf("columns are %d cells wide, want at most %d", sum+columnGap*(n-1), tt.total)
				}
			}
		})
	}
}

// TestDropOrder renders the list at every width from 120 down to 40 and
// reads the column-header row: PR, REVIEW and CI stay, the other columns only
// ever disappear, in the order AUTHOR, STATE, MERGE, TITLE.
func TestDropOrder(t *testing.T) {
	snap := fixtureSix()
	snap.PRs = append(snap.PRs, domain.PR{
		ID: "7", Ref: domain.Ref{Repo: "acme/api", Number: 999}, Title: strings.Repeat("0123456789", 12), Author: "dave",
		State: domain.StateOpen, Review: domain.ReviewApproved, CI: domain.CIPassing,
	})
	order := []string{"AUTHOR", "STATE", "MERGE", "TITLE"}
	prev := map[string]bool{"AUTHOR": true, "STATE": true, "MERGE": true, "TITLE": true}
	for w := 120; w >= 40; w-- {
		ls := lines(newTestModel(t, snap, w, 40).View().Content)
		for _, l := range ls {
			if lipgloss.Width(l) > w {
				t.Fatalf("w=%d: line is %d cells wide: %q", w, lipgloss.Width(l), l)
			}
		}
		fields := strings.Fields(ls[1])
		has := map[string]bool{}
		for _, f := range fields {
			has[f] = true
		}
		for _, must := range []string{"PR", "REVIEW", "CI"} {
			if !has[must] {
				t.Fatalf("w=%d: column header %q lacks %s", w, ls[1], must)
			}
		}
		for i, c := range order {
			if has[c] && !prev[c] {
				t.Errorf("w=%d: column %s came back after it was dropped", w, c)
			}
			// A column may only be gone when every column before it in the
			// drop order is gone too.
			if !has[c] {
				for _, earlier := range order[:i] {
					if has[earlier] {
						t.Errorf("w=%d: %s dropped while %s is still shown: %q", w, c, earlier, ls[1])
					}
				}
			}
		}
		prev = has
	}
	if prev["TITLE"] || prev["AUTHOR"] || prev["STATE"] || prev["MERGE"] {
		t.Errorf("at 40 cells the header is %v, want only PR, REVIEW and CI", prev)
	}
}

func TestTrimLeft(t *testing.T) {
	for _, tt := range []struct {
		s    string
		w    int
		want string
	}{
		{"acme/api#1", 20, "acme/api#1"},
		{"acme/api#1", 10, "acme/api#1"},
		{"acme-corporation/backend-service#4321", 20, "…ackend-service#4321"},
		{"acme/api#1", 5, "…pi#1"},
		{"acme/api#1", 1, "a"},
		{"acme/api#1", 0, ""},
	} {
		got := trimLeft(tt.s, tt.w)
		if got != tt.want {
			t.Errorf("trimLeft(%q, %d) = %q, want %q", tt.s, tt.w, got, tt.want)
		}
		if lipgloss.Width(got) > max(tt.w, 0) {
			t.Errorf("trimLeft(%q, %d) is %d cells wide", tt.s, tt.w, lipgloss.Width(got))
		}
	}
	if got := trimLeft("acme-corporation/backend-service#4321", 20); !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, "#4321") {
		t.Errorf("a cut ref must start with … and keep its number: %q", got)
	}
}

// TestLayoutCompact: below 40 cells the list turns into compact rows; an
// unsized pane never does (D-14).
func TestLayoutCompact(t *testing.T) {
	for _, tt := range []struct {
		total   int
		compact bool
	}{{-1, false}, {0, false}, {1, true}, {20, true}, {39, true}, {40, false}, {41, false}, {120, false}} {
		if got := layoutColumns(testSizes(), tt.total).compact; got != tt.compact {
			t.Errorf("layoutColumns(total=%d).compact = %v, want %v", tt.total, got, tt.compact)
		}
	}
}

// compactSnap has three open PRs with github.com URLs, so o can open them.
func compactSnap() domain.Snapshot {
	snap := fixtureSix()
	for i := range snap.PRs {
		snap.PRs[i].URL = fmt.Sprintf("https://github.com/%s/pull/%d", snap.PRs[i].Ref.Repo, snap.PRs[i].Ref.Number)
	}
	return snap
}

func TestCompactRows(t *testing.T) {
	ls := lines(newTestModel(t, fixtureSix(), 36, 12).View().Content)
	want := map[string]string{
		"api#101": "> api#101 ✓✓✓",
		"api#102": "  api#102 ✗⊘✗",
		"web#7":   "  web#7 ––●",
		"docs#3":  "  docs#3 ○ –",
		"docs#2":  "  docs#2 –↓?",
		"web#6":   "  web#6 ✓ ✓",
	}
	for ref, row := range want {
		var found string
		for _, l := range ls {
			if strings.Contains(l, ref) {
				found = l
			}
		}
		if found != row {
			t.Errorf("row of %s = %q, want %q", ref, found, row)
		}
	}
	for _, l := range ls {
		if strings.Contains(l, "acme/") {
			t.Errorf("a compact row shows the owner: %q", l)
		}
	}

	t.Run("a ref that does not fit is cut from the left and keeps all three glyphs", func(t *testing.T) {
		snap := domain.Snapshot{Viewer: "octo", PRs: []domain.PR{{
			ID: "1", Ref: domain.Ref{Repo: "acme/long-repository-name", Number: 1234}, Title: "t", Author: "octo",
			State: domain.StateOpen, Review: domain.ReviewChangesRequested, CI: domain.CIFailing, Merge: domain.MergeBlocked,
		}}}
		for _, w := range []int{13, 12, 10, 9, 8} {
			row := lines(newTestModel(t, snap, w, 12).View().Content)[2]
			if !strings.HasSuffix(row, " ✗⊘✗") || !strings.Contains(row, "…") {
				t.Errorf("w=%d: row = %q, want a left-cut ref, then \" ✗⊘✗\"", w, row)
			}
			if w >= 12 && !strings.Contains(row, "#1234") {
				t.Errorf("w=%d: row = %q, want the cut ref to keep #1234", w, row)
			}
			if lipgloss.Width(row) > w {
				t.Errorf("w=%d: row is %d cells wide: %q", w, lipgloss.Width(row), row)
			}
		}
	})

	t.Run("the selected row is reversed segment by segment", func(t *testing.T) {
		raw := strings.Split(newTestModel(t, fixtureSix(), 36, 12).View().Content, "\n")[2]
		reverse := regexp.MustCompile(`\x1b\[(?:[0-9]+;)*7(?:;[0-9]+)*m`)
		if got := len(reverse.FindAllString(raw, -1)); got < 4 {
			t.Errorf("the selected compact row has %d reverse-video segments, want 4 (ref, review glyph, merge glyph, CI glyph): %q", got, raw)
		}
	})
}

func TestCompactHeaderHint(t *testing.T) {
	for _, w := range []int{20, 36, 39} {
		ls := lines(newTestModel(t, fixtureSix(), w, 12).View().Content)
		if want := "  PR  widen for columns"; !strings.HasPrefix(want, ls[1]) && ls[1] != want {
			t.Errorf("w=%d: column-header line = %q, want %q cut to the width", w, ls[1], want)
		}
		if strings.Contains(ls[0], "widen") {
			t.Errorf("w=%d: the header line carries the hint: %q", w, ls[0])
		}
	}
	if got := lines(newTestModel(t, fixtureSix(), 36, 12).View().Content)[1]; got != "  PR  widen for columns" {
		t.Errorf("w=36: column-header line = %q", got)
	}
	for _, w := range []int{40, 41, 80} {
		if out := newTestModel(t, fixtureSix(), w, 12).View().Content; strings.Contains(out, "widen for columns") {
			t.Errorf("w=%d: the hint is shown at 40 cells or more", w)
		}
	}
}

// markedLine is the one line that starts with the marker, or "".
func markedLine(t *testing.T, m tea.Model) string {
	t.Helper()
	var found []string
	for _, l := range strings.Split(plain(m), "\n") {
		if strings.HasPrefix(l, ">") {
			found = append(found, l)
		}
	}
	if len(found) > 1 {
		t.Fatalf("%d rows are marked: %q", len(found), found)
	}
	if len(found) == 0 {
		return ""
	}
	return found[0]
}

func TestCompactNavigation(t *testing.T) {
	o := &fakeOpener{}
	var m tea.Model = filterModelOpts(36, 12, compactSnap(), Options{ClosedDays: 14, Now: fixedNow, Opener: o})
	first := markedLine(t, m)
	if first != "> api#101 ✓✓✓" {
		t.Fatalf("first marked row = %q", first)
	}
	m = press(m, "j")
	second := markedLine(t, m)
	if second == first || second == "" {
		t.Errorf("j did not move the marker: %q", second)
	}
	m = press(m, "k")
	if got := markedLine(t, m); got != first {
		t.Errorf("k did not move the marker back: %q", got)
	}

	// o opens the selected PR.
	m, cmd := m.Update(key("o"))
	if cmd == nil {
		t.Fatal("o returned no command in the compact list")
	}
	cmd()
	if len(o.urls) != 1 || o.urls[0] != "https://github.com/acme/api/pull/101" {
		t.Errorf("opened %v, want the selected PR", o.urls)
	}

	// 1/2/3 and / keep working.
	m = press(m, "1")
	if strings.Contains(plain(m), "api#101") {
		t.Errorf("1 did not hide the open PRs in the compact list:\n%s", plain(m))
	}
	m = press(m, "1", "/", "1", "0", "2")
	if got := lastLine(m); !strings.HasPrefix(got, "/ 102█") {
		t.Errorf("the number prompt line = %q", got)
	}
	if got := markedLine(t, m); got != "> api#102 ✗⊘✗" {
		t.Errorf("filtering by 102 left %q marked", got)
	}
}

func TestResizeKeepsSelection(t *testing.T) {
	var m tea.Model = selModel(120, 40, rangeInts(1, 30)...)
	m = press(m, strings.Split(strings.Repeat("j", 20), "")...)
	for _, sz := range [][2]int{{120, 40}, {60, 15}, {30, 10}, {120, 40}, {36, 6}, {80, 24}} {
		m, _ = m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		if got := markedLine(t, m); !strings.Contains(got, "#21") {
			t.Errorf("after resizing to %dx%d the marked row is %q, want PR #21", sz[0], sz[1], got)
		}
		if n := len(strings.Split(plain(m), "\n")); n > sz[1] {
			t.Errorf("%dx%d: %d lines", sz[0], sz[1], n)
		}
	}
}

func rangeInts(from, to int) []int {
	var out []int
	for i := from; i <= to; i++ {
		out = append(out, i)
	}
	return out
}

func TestTinyHeights(t *testing.T) {
	for _, w := range []int{80, 36} {
		for h, want := range map[int][]string{
			1: {">"},
			2: {"PRs", ">"},
			3: {"PRs", ">", "j/k"},
		} {
			ls := lines(newTestModel(t, fixtureSix(), w, h).View().Content)
			if len(ls) != len(want) {
				t.Errorf("w=%d h=%d: %d lines, want %d: %q", w, h, len(ls), len(want), ls)
				continue
			}
			for i, part := range want {
				ok := strings.Contains(ls[i], part)
				if part == ">" {
					ok = strings.HasPrefix(ls[i], ">")
				}
				if !ok {
					t.Errorf("w=%d h=%d: line %d = %q, want it to hold %q", w, h, i, ls[i], part)
				}
			}
		}
	}
}

// TestNeverOverflows renders the list at every width and height the property
// covers, prompt closed and open: no panic, no line wider than the pane, no
// more lines than the pane.
func TestNeverOverflows(t *testing.T) {
	check := func(w, h int, prompt bool) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("%dx%d prompt=%v: panic: %v", w, h, prompt, r)
			}
		}()
		var m tea.Model = newTestModel(t, fixtureThirty(), w, h)
		if prompt {
			m = press(m, "/", "1")
		}
		ls := lines(m.View().Content)
		if len(ls) > h {
			t.Errorf("%dx%d prompt=%v: %d lines, want at most %d", w, h, prompt, len(ls), h)
		}
		for _, l := range ls {
			if lipgloss.Width(l) > w {
				t.Errorf("%dx%d prompt=%v: line is %d cells wide: %q", w, h, prompt, lipgloss.Width(l), l)
				break
			}
		}
	}
	for _, prompt := range []bool{false, true} {
		for _, h := range []int{3, 15, 50} {
			for w := 1; w <= 200; w++ {
				check(w, h, prompt)
			}
		}
		for _, w := range []int{30, 60, 120} {
			for h := 1; h <= 50; h++ {
				check(w, h, prompt)
			}
		}
	}
}

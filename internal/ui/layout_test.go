package ui

import (
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/sbartram/prv/internal/domain"
)

// testSizes are the column sizes of a typical list: TITLE wants 40 cells and
// the other columns 44 plus 6 gaps, so everything fits from 60 cells up.
func testSizes() sizes {
	var sz sizes
	sz.natural = [colCount]int{colMarker: 1, colPR: 12, colTitle: 40, colAuthor: 6, colState: 6, colReview: 10, colCI: 9}
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
	all := []int{colMarker, colPR, colTitle, colAuthor, colState, colReview, colCI}
	noAuthor := []int{colMarker, colPR, colTitle, colState, colReview, colCI}
	noState := []int{colMarker, colPR, colTitle, colReview, colCI}
	noTitle := []int{colMarker, colPR, colReview, colCI}

	shortTitle := testSizes()
	shortTitle.natural[colTitle], shortTitle.min[colTitle] = 7, 7

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
		{"TITLE takes what is left", testSizes(), 70, all, 20, 12, 9},
		{"TITLE at its minimum", testSizes(), 60, all, 10, 12, 9},
		{"AUTHOR drops first", testSizes(), 59, noAuthor, 16, 12, 9},
		{"AUTHOR gone, TITLE at its minimum", testSizes(), 53, noAuthor, 10, 12, 9},
		{"STATE drops second", testSizes(), 52, noState, 16, 12, 9},
		{"STATE gone, TITLE at its minimum", testSizes(), 46, noState, 10, 12, 9},
		{"TITLE drops last", testSizes(), 45, noTitle, 0, 12, 9},
		{"only the four columns left, exactly fitting", testSizes(), 35, noTitle, 0, 12, 9},
		{"a short TITLE does not force a drop", shortTitle, 57, all, 7, 12, 9},
		{"a short TITLE still drops AUTHOR when it must", shortTitle, 56, noAuthor, 7, 12, 9},
		{"PR narrows when nothing else can", testSizes(), 34, noTitle, 0, 11, 9},
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
// ever disappear, in the order AUTHOR, STATE, TITLE.
func TestDropOrder(t *testing.T) {
	snap := fixtureSix()
	snap.PRs = append(snap.PRs, domain.PR{
		ID: "7", Ref: domain.Ref{Repo: "acme/api", Number: 999}, Title: strings.Repeat("0123456789", 12), Author: "dave",
		State: domain.StateOpen, Review: domain.ReviewApproved, CI: domain.CIPassing,
	})
	order := []string{"AUTHOR", "STATE", "TITLE"}
	prev := map[string]bool{"AUTHOR": true, "STATE": true, "TITLE": true}
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
	if prev["TITLE"] || prev["AUTHOR"] || prev["STATE"] {
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

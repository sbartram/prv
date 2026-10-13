package ui

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sbartram/prv/internal/domain"
)

// unavailSnap holds, in the order Fetch gives: two unavailable saved PRs (not
// found, SSO), a failing discovered PR and a loaded saved PR that passes.
func unavailSnap() domain.Snapshot {
	at := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	snap := domain.Snapshot{Viewer: "octo", PRs: []domain.PR{
		{ID: "PR_1", Ref: domain.Ref{Repo: "acme/api", Number: 1}, Title: "Rework the cache layer", Author: "alice", Auto: true,
			State: domain.StateOpen, Review: domain.ReviewChangesRequested, CI: domain.CIFailing, UpdatedAt: at},
		{ID: "PR_42", Ref: domain.Ref{Repo: "octo/tools", Number: 42}, Title: "Speed up the build", Author: "alice", Manual: true,
			Watched: domain.Ref{Repo: "octo/tools", Number: 42}, State: domain.StateOpen, Review: domain.ReviewApproved,
			CI: domain.CIPassing, UpdatedAt: at},
		{Ref: domain.Ref{Repo: "samlcorp/app", Number: 3}, Manual: true, Watched: domain.Ref{Repo: "samlcorp/app", Number: 3},
			Unavail: domain.SSORequired("samlcorp")},
		{Ref: domain.Ref{Repo: "acme/gone", Number: 5}, Manual: true, Watched: domain.Ref{Repo: "acme/gone", Number: 5},
			Unavail: domain.NotFound()},
	}}
	domain.Sort(snap.PRs)
	return snap
}

// rowLine returns the stripped line that holds needle, or "".
func rowLine(m Model, needle string) string {
	for _, l := range lines(m.View().Content) {
		if strings.Contains(l, needle) {
			return l
		}
	}
	return ""
}

func TestUnavailableRow(t *testing.T) {
	for _, w := range []int{120, 80, 60} {
		t.Run(itoa(w), func(t *testing.T) {
			m := newTestModel(t, unavailSnap(), w, 24)
			ls := lines(m.View().Content)
			// Spike (RESEARCH Pattern 8): table line 1 + i is row m.offset + i.
			if !strings.Contains(ls[2], "acme/gone#5") || !strings.Contains(ls[3], "samlcorp/app#3") {
				t.Fatalf("the first two rows are not acme/gone#5, samlcorp/app#3:\n%s", strings.Join(ls, "\n"))
			}
			if !strings.HasPrefix(ls[2], ">") || !strings.Contains(ls[2], "+ acme/gone#5") {
				t.Errorf("row 1 = %q, want the cursor and '+ acme/gone#5'", ls[2])
			}
			if !strings.Contains(ls[2], "⚠ unavailable: not") {
				t.Errorf("row 1 = %q, want '⚠ unavailable: not ...'", ls[2])
			}
			if w >= 80 && !strings.Contains(ls[2], "⚠ unavailable: not found") {
				t.Errorf("row 1 = %q, want '⚠ unavailable: not found'", ls[2])
			}
			if w == 120 && !strings.Contains(ls[2], "⚠ unavailable: "+domain.NotFound().Reason) {
				t.Errorf("row 1 = %q, want the whole reason at 120 columns", ls[2])
			}
			if !strings.Contains(ls[3], "⚠ unavailable: SSO") {
				t.Errorf("row 2 = %q, want the SSO reason", ls[3])
			}
			for i, l := range ls {
				if lw := lipgloss.Width(l); lw > w {
					t.Errorf("line %d is %d cells wide, pane is %d: %q", i, lw, w, l)
				}
			}
			if l := rowLine(m, "octo/tools#42"); !strings.Contains(l, "+ octo/tools#42") || !strings.Contains(l, "✓ passing") {
				t.Errorf("loaded saved row = %q, want '+ octo/tools#42' and '✓ passing'", l)
			}
			if l := rowLine(m, "acme/api#1"); !strings.Contains(l, "✗ failing") {
				t.Errorf("discovered row = %q, want '✗ failing'", l)
			}
		})
	}
}

func TestUnavailableNoWiden(t *testing.T) {
	header := func(reason string) string {
		snap := unavailSnap()
		for i := range snap.PRs {
			if snap.PRs[i].Unavail.Kind != domain.UnavailNone {
				snap.PRs[i].Unavail.Reason = reason
			}
		}
		return lines(newTestModel(t, snap, 120, 24).View().Content)[1]
	}
	if short, long := header("x"), header(strings.Repeat("long reason ", 7)); short != long {
		t.Errorf("column-header line changed with the reason:\n%q\n%q", short, long)
	}
}

var (
	yellowSeq = regexp.MustCompile(`\x1b\[(?:[0-9;]*;)?33(?:;[0-9;]*)?m`)
	faintSeq  = regexp.MustCompile(`\x1b\[(?:[0-9;]*;)?2(?:;[0-9;]*)?m`)
)

func TestUnavailableStyles(t *testing.T) {
	rawRow := func(m Model, needle string) string {
		for _, l := range strings.Split(m.View().Content, "\n") {
			if strings.Contains(ansi.Strip(l), needle) {
				return l
			}
		}
		return ""
	}
	m := newTestModel(t, unavailSnap(), 120, 24)
	sel := rawRow(m, "acme/gone#5")
	if !reverseSeq.MatchString(sel) {
		t.Errorf("selected unavailable row has no reverse video: %q", sel)
	}
	if !yellowSeq.MatchString(sel) || faintSeq.MatchString(sel) {
		t.Errorf("selected unavailable row: yellow %v, faint %v, want yellow and not faint: %q",
			yellowSeq.MatchString(sel), faintSeq.MatchString(sel), sel)
	}
	next := newTestModel(t, unavailSnap(), 120, 24)
	var tm tea.Model = next
	tm, _ = tm.Update(key("j"))
	row := rawRow(tm.(Model), "acme/gone#5")
	if !yellowSeq.MatchString(row) {
		t.Errorf("unselected unavailable row has no yellow: %q", row)
	}
	if faintSeq.MatchString(row) {
		t.Errorf("unselected unavailable row is faint (faint means draft, merged or closed, D-09): %q", row)
	}
	if reverseSeq.MatchString(row) {
		t.Errorf("unselected unavailable row is reversed: %q", row)
	}
}

func TestUnavailableHeaderCount(t *testing.T) {
	m := newTestModel(t, unavailSnap(), 120, 24)
	if h := lines(m.View().Content)[0]; !strings.Contains(h, "4 PRs · 2 unavailable") {
		t.Errorf("header = %q, want '4 PRs · 2 unavailable'", h)
	}
	narrow := newTestModel(t, unavailSnap(), 30, 24)
	h := lines(narrow.View().Content)[0]
	if !strings.Contains(h, "2 unavailable") || strings.Contains(h, "updated") {
		t.Errorf("30-column header = %q, want '2 unavailable' kept and 'updated' dropped", h)
	}
	if h := lines(newTestModel(t, fixtureSix(), 120, 24).View().Content)[0]; strings.Contains(h, "unavailable") {
		t.Errorf("header with no unavailable row = %q, want no 'unavailable'", h)
	}
	// Filters hide the loaded rows; the count of unavailable rows stays.
	var tm tea.Model = m
	tm = press(tm, "/", "5")
	if h := lines(tm.View().Content)[0]; !strings.Contains(h, "1 of 4 PRs") || !strings.Contains(h, "2 unavailable") {
		t.Errorf("filtered header = %q, want '1 of 4 PRs' and '2 unavailable'", h)
	}
}

func TestUnavailableFilters(t *testing.T) {
	rows := func(m tea.Model) []string {
		var out []string
		for _, pr := range m.(Model).rows {
			out = append(out, pr.Ref.String())
		}
		return out
	}
	var m tea.Model = newTestModel(t, unavailSnap(), 120, 24)
	// Hide open and closed: only merged is shown; no loaded row is merged.
	m = press(m, "1", "3")
	if got := strings.Join(rows(m), " "); got != "acme/gone#5 samlcorp/app#3" {
		t.Errorf("only Merged shown: rows = %q, want both unavailable rows (D-13)", got)
	}
	m = press(m, "1", "3")
	m = press(m, "/", "5")
	if got := strings.Join(rows(m), " "); got != "acme/gone#5" {
		t.Errorf("/5: rows = %q, want 'acme/gone#5' (matched by number)", got)
	}
}

func TestCompactWatchRows(t *testing.T) {
	m := newTestModel(t, unavailSnap(), 36, 24)
	ls := lines(m.View().Content)
	if got := strings.TrimRight(ls[2], " "); got != "> + gone#5 ⚠" {
		t.Errorf("unavailable compact row = %q, want '> + gone#5 ⚠'", got)
	}
	tools := domain.PR{Review: domain.ReviewApproved, CI: domain.CIPassing}
	want := "  + tools#42 " + tools.Review.Glyph() + tools.Merge.Glyph() + tools.CI.Glyph()
	if got := strings.TrimRight(rowLine(m, "tools#42"), " "); got != want {
		t.Errorf("saved compact row = %q, want %q", got, want)
	}
	// With no room for the ref the + stays.
	tiny := newTestModel(t, unavailSnap(), 6, 24)
	row := lines(tiny.View().Content)[0]
	for _, l := range lines(tiny.View().Content) {
		if strings.HasPrefix(l, ">") {
			row = l
		}
	}
	if !strings.HasPrefix(row, "> + ") || !strings.Contains(row, "⚠") {
		t.Errorf("6-column compact row = %q, want '> + ' kept and the warning glyph", row)
	}
}

// toolsRef is the saved PR the two-strike tests follow.
var toolsRef = domain.Ref{Repo: "octo/tools", Number: 42}

// toolsSnap is a snapshot whose saved PR octo/tools#42 is in state: nil means it
// loaded.
func toolsSnap(unavail *domain.Unavail) domain.Snapshot {
	at := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	pr := domain.PR{ID: "PR_42", Ref: toolsRef, Title: "Speed up the build", Author: "alice", Manual: true, Watched: toolsRef,
		State: domain.StateOpen, Review: domain.ReviewApproved, CI: domain.CIPassing, UpdatedAt: at}
	if unavail != nil {
		pr = domain.PR{Ref: toolsRef, Manual: true, Watched: toolsRef, URL: toolsRef.URL(), Unavail: *unavail}
	}
	other := domain.PR{ID: "PR_1", Ref: domain.Ref{Repo: "acme/api", Number: 1}, Title: "Api change", Author: "octo", Auto: true,
		State: domain.StateOpen, UpdatedAt: at}
	snap := domain.Snapshot{Viewer: "octo", PRs: []domain.PR{pr, other}}
	domain.Sort(snap.PRs)
	return snap
}

func TestTwoStrikeRule(t *testing.T) {
	failed := domain.LoadFailed("server error", "")
	notFound := domain.NotFound()
	row := func(m tea.Model) string { return rowLine(m.(Model), "octo/tools#42") }
	update := func(m tea.Model, snap domain.Snapshot, err error) tea.Model {
		m, _ = m.Update(fetchedMsg{gen: m.(Model).gen, snap: snap, err: err})
		return m
	}

	var m tea.Model = newTestModel(t, toolsSnap(nil), 120, 24)

	m = update(m, toolsSnap(&failed), nil) // strike 1
	if r := row(m); !strings.Contains(r, "Speed up the build") || !strings.Contains(r, "✓ passing") || strings.Contains(r, "unavailable") {
		t.Errorf("after one failed refresh the row = %q, want its last data", r)
	}

	m = update(m, domain.Snapshot{}, errors.New("boom")) // the whole refresh fails: no strike
	if r := row(m); !strings.Contains(r, "Speed up the build") {
		t.Errorf("after a failed refresh the row = %q, want its last data", r)
	}
	if n := m.(Model).strikes[toolsRef]; n != 1 {
		t.Errorf("strikes = %d after a refresh that failed as a whole, want 1 (unchanged)", n)
	}

	m = update(m, toolsSnap(&failed), nil) // strike 2
	if r := row(m); !strings.Contains(r, "⚠ unavailable: could not load (server error)") {
		t.Errorf("after two failed refreshes the row = %q, want '⚠ unavailable: could not load (server error)'", r)
	}

	m = update(m, toolsSnap(nil), nil) // loads again
	if r := row(m); !strings.Contains(r, "Speed up the build") || strings.Contains(r, "unavailable") {
		t.Errorf("after the PR loads the row = %q, want a normal row", r)
	}
	if n := m.(Model).strikes[toolsRef]; n != 0 {
		t.Errorf("strikes = %d after a good load, want 0", n)
	}

	m = update(m, toolsSnap(&failed), nil) // strike 1 again: the count started over
	if r := row(m); !strings.Contains(r, "Speed up the build") {
		t.Errorf("first failure after a good load: row = %q, want its last data", r)
	}

	m = update(m, toolsSnap(&notFound), nil) // a real answer: at once
	if r := row(m); !strings.Contains(r, "⚠ unavailable: not found") {
		t.Errorf("after NOT_FOUND the row = %q, want it unavailable at once", r)
	}
}

func TestClosedSavedFilters(t *testing.T) {
	long := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	snap := domain.Snapshot{Viewer: "octo", PRs: []domain.PR{
		{ID: "PR_7", Ref: domain.Ref{Repo: "octo/old", Number: 7}, Title: "Dropped idea", Author: "octo", Manual: true,
			Watched: domain.Ref{Repo: "octo/old", Number: 7}, State: domain.StateClosed, ClosedAt: long, UpdatedAt: long},
		{ID: "PR_8", Ref: domain.Ref{Repo: "octo/old", Number: 8}, Title: "Shipped idea", Author: "octo", Manual: true,
			Watched: domain.Ref{Repo: "octo/old", Number: 8}, State: domain.StateMerged, ClosedAt: long, UpdatedAt: long},
	}}
	domain.Sort(snap.PRs)
	var m tea.Model = newTestModel(t, snap, 120, 24)
	have := func() string {
		var out []string
		for _, pr := range m.(Model).rows {
			out = append(out, pr.Ref.String())
		}
		return strings.Join(out, " ")
	}
	if got := have(); got != "octo/old#8 octo/old#7" {
		t.Fatalf("rows = %q, want the merged then the closed saved PR", got)
	}
	if l := rowLine(m.(Model), "octo/old#7"); !strings.Contains(l, "closed") {
		t.Errorf("row = %q, want 'closed'", l)
	}
	m = press(m, "3")
	if got := have(); got != "octo/old#8" {
		t.Errorf("3 hides closed: rows = %q, want 'octo/old#8'", got)
	}
	m = press(m, "3", "2")
	if got := have(); got != "octo/old#7" {
		t.Errorf("2 hides merged: rows = %q, want 'octo/old#7'", got)
	}
	m = press(m, "2")
	if got := have(); got != "octo/old#8 octo/old#7" {
		t.Errorf("toggled back: rows = %q, want both", got)
	}
}

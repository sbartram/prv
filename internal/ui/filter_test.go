package ui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// mixedSnap holds open #1, merged #2, closed #3 and unknown-state #4.
func mixedSnap() domain.Snapshot {
	s := selSnap(1, 2, 3, 4)
	s.PRs[0].State = domain.StateOpen
	s.PRs[1].State = domain.StateMerged
	s.PRs[2].State = domain.StateClosed
	s.PRs[3].State = domain.StateUnknown
	return s
}

// filterModel builds a sized model that has loaded snap.
func filterModel(w, h int, snap domain.Snapshot) tea.Model {
	return filterModelOpts(w, h, snap, Options{ClosedDays: 14})
}

func filterModelOpts(w, h int, snap domain.Snapshot, opts Options) tea.Model {
	var m tea.Model = New(context.Background(), &fakeFetcher{}, opts)
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m, _ = m.Update(result(m, snap, nil))
	return m
}

// rowRefs lists the PR column of every table row on screen, in order.
func rowRefs(m tea.Model) []string {
	var out []string
	for _, l := range strings.Split(plain(m), "\n") {
		f := strings.Fields(strings.TrimPrefix(l, ">"))
		if len(f) > 0 && strings.HasPrefix(f[0], "a/b#") {
			out = append(out, f[0])
		}
	}
	return out
}

func TestStateToggles(t *testing.T) {
	m := filterModel(120, 40, mixedSnap())
	if got := headerOf(m); !strings.Contains(got, "[open merged closed]") {
		t.Fatalf("header = %q, want every state on at launch", got)
	}
	for _, tt := range []struct {
		keys   []string
		header string
		refs   []string
	}{
		{[]string{"2"}, "[open · closed]", []string{"a/b#1", "a/b#3", "a/b#4"}},
		{[]string{"2", "3"}, "[open · ·]", []string{"a/b#1", "a/b#4"}},
		{[]string{"1", "2", "3"}, "[· · ·]", []string{"a/b#4"}},
		{[]string{"1"}, "[· merged closed]", []string{"a/b#2", "a/b#3", "a/b#4"}},
	} {
		got := press(filterModel(120, 40, mixedSnap()), tt.keys...)
		if h := headerOf(got); !strings.Contains(h, tt.header) {
			t.Errorf("after %v header = %q, want %q", tt.keys, h, tt.header)
		}
		if refs := rowRefs(got); strings.Join(refs, ",") != strings.Join(tt.refs, ",") {
			t.Errorf("after %v rows = %v, want %v", tt.keys, refs, tt.refs)
		}
	}
	for _, k := range []string{"m", "o", "c"} {
		before := plain(m)
		if after := plain(press(m, k)); after != before {
			t.Errorf("letter %q changed the screen", k)
		}
	}
}

func TestToggleTwiceRestores(t *testing.T) {
	for _, k := range []string{"1", "2", "3"} {
		// Select the unknown-state PR, which no toggle can hide, so the
		// selection has no reason to move.
		m := press(filterModel(120, 40, mixedSnap()), "j", "j", "j")
		before := plain(m)
		beforeSel := m.(Model).selected
		got := press(m, k, k)
		if after := plain(got); after != before {
			t.Errorf("toggling %s twice changed the screen:\nbefore:\n%s\nafter:\n%s", k, before, after)
		}
		if got.(Model).selected != beforeSel {
			t.Errorf("toggling %s twice moved the selection to %v", k, got.(Model).selected)
		}
	}
}

func TestUnknownAlwaysVisible(t *testing.T) {
	m := press(filterModel(120, 40, mixedSnap()), "1", "2", "3")
	if refs := rowRefs(m); len(refs) != 1 || refs[0] != "a/b#4" {
		t.Errorf("rows = %v, want only the unknown-state PR", refs)
	}
	if got := headerOf(m); !strings.Contains(got, "1 of 4 PRs") {
		t.Errorf("header = %q, want 1 of 4 PRs", got)
	}
}

func TestHiddenCount(t *testing.T) {
	m := filterModel(120, 40, mixedSnap())
	if got := headerOf(m); !strings.Contains(got, "4 PRs") || strings.Contains(got, " of ") {
		t.Errorf("header = %q, want plain '4 PRs' when nothing is hidden", got)
	}
	m = press(m, "2")
	if got := headerOf(m); !strings.Contains(got, "3 of 4 PRs") {
		t.Errorf("header = %q, want '3 of 4 PRs'", got)
	}
	m = press(m, "2")
	if got := headerOf(m); !strings.Contains(got, "4 PRs") || strings.Contains(got, " of ") {
		t.Errorf("header = %q, want plain count again", got)
	}
}

func TestFilteredEmpty(t *testing.T) {
	snap := selSnap(2, 3)
	snap.PRs[0].State = domain.StateMerged
	snap.PRs[1].State = domain.StateClosed
	m := press(filterModel(120, 24, snap), "2", "3")
	v := plain(m)
	if !strings.Contains(v, "No PRs match: [open · ·] — 2 hidden by filters · 1/2/3 toggle") {
		t.Errorf("view = %q, want the filtered-empty message", v)
	}
	if strings.Contains(v, "No open PRs authored by") {
		t.Errorf("a filter must not claim the account is empty:\n%s", v)
	}
	if !strings.HasPrefix(v, "prv · @octo") {
		t.Errorf("filtered-empty screen has no header line:\n%s", v)
	}
	t.Run("integrity notes still show", func(t *testing.T) {
		snap := snap
		snap.Truncated = true
		snap.Warnings = []string{"x"}
		v := plain(press(filterModel(120, 24, snap), "2", "3"))
		for _, want := range []string{"list truncated", "some data could not be loaded"} {
			if !strings.Contains(v, want) {
				t.Errorf("view lacks %q:\n%s", want, v)
			}
		}
	})
	t.Run("toggle back restores the rows", func(t *testing.T) {
		got := press(m, "2")
		if refs := rowRefs(got); len(refs) != 1 || refs[0] != "a/b#2" {
			t.Errorf("rows = %v, want a/b#2", refs)
		}
	})
	t.Run("zero PRs keeps the Phase 1 sentence", func(t *testing.T) {
		v := plain(press(filterModel(120, 24, domain.Snapshot{Viewer: "octo"}), "2"))
		if !strings.Contains(v, "No open PRs authored by or assigned to @octo") || strings.Contains(v, "No PRs match") {
			t.Errorf("view = %q", v)
		}
	})
}

func TestDefaultShowFromOptions(t *testing.T) {
	m := filterModelOpts(120, 40, mixedSnap(), Options{ClosedDays: 14, Show: 1 << domain.StateOpen})
	if got := headerOf(m); !strings.Contains(got, "[open · ·]") {
		t.Errorf("header = %q, want [open · ·] from Options.Show", got)
	}
	if refs := rowRefs(m); len(refs) != 2 {
		t.Errorf("rows = %v, want open #1 and unknown #4", refs)
	}
}

func TestToggleSurvivesRefresh(t *testing.T) {
	m := press(filterModel(120, 40, mixedSnap()), "3")
	next := selSnap(1, 3, 5, 6)
	next.PRs[1].State = domain.StateClosed
	next.PRs[2].State = domain.StateClosed
	m = refreshTo2(m, next)
	if refs := rowRefs(m); strings.Join(refs, ",") != "a/b#1,a/b#6" {
		t.Errorf("rows = %v, want the closed PRs hidden in the new data", refs)
	}
	if got := headerOf(m); !strings.Contains(got, "[open merged ·]") || !strings.Contains(got, "2 of 4 PRs") {
		t.Errorf("header = %q", got)
	}
}

// refreshTo2 delivers a successful refresh of a full snapshot.
func refreshTo2(m tea.Model, snap domain.Snapshot) tea.Model {
	m, _ = m.Update(result(m, snap, nil))
	return m
}

func TestSelectionAfterToggle(t *testing.T) {
	snap := selSnap(1, 2, 3)
	snap.PRs[1].State = domain.StateMerged
	m := press(filterModel(120, 40, snap), "j") // select the merged PR
	if got := markedRef(t, m); got != "a/b#2" {
		t.Fatalf("setup: marked = %q", got)
	}
	m = press(m, "2")
	if got := markedRef(t, m); got != "a/b#3" {
		t.Errorf("marked = %q, want the nearest visible neighbour a/b#3", got)
	}
}

func TestFilterIndicator(t *testing.T) {
	m := press(filterModel(120, 40, mixedSnap()), "2", "3")
	if got := headerOf(m); !strings.Contains(got, "[open · ·]") {
		t.Errorf("header = %q", got)
	}
	// The indicator survives a pane too narrow for the whole line.
	if h := headerOf(press(filterModel(36, 24, mixedSnap()), "2")); !strings.Contains(h, "[open · closed]") && !strings.Contains(h, "3 of 4 PRs") {
		t.Errorf("narrow header = %q, want the filter or the count kept", h)
	}
	if got := plain(filterModel(120, 40, mixedSnap())); !strings.Contains(got, "1/2/3 states") {
		t.Errorf("footer hints lack '1/2/3 states':\n%s", got)
	}
}

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
	for _, k := range []string{"m", "x", "c"} { // "o" is the open key since plan 02-10
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

// ---- number prompt (plan 02-08 task 2) ----

// promptSnap holds a/b#12, #120, #1234, #312 and #5.
func promptSnap() domain.Snapshot { return selSnap(12, 120, 1234, 312, 5) }

func lastLine(m tea.Model) string {
	ls := strings.Split(plain(m), "\n")
	return ls[len(ls)-1]
}

func special(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

func send(m tea.Model, msgs ...tea.Msg) tea.Model {
	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}
	return m
}

func TestNumberPrompt(t *testing.T) {
	open := func() tea.Model { return press(filterModel(120, 40, promptSnap()), "/") }
	t.Run("open", func(t *testing.T) {
		got := lastLine(open())
		if !strings.HasPrefix(got, "/ █") || !strings.Contains(got, "(5 of 5 · enter keep · esc clear)") {
			t.Errorf("last line = %q", got)
		}
	})
	t.Run("narrow", func(t *testing.T) {
		m := press(open(), "1", "2")
		if got := lastLine(m); !strings.HasPrefix(got, "/ 12█   (3 of 5 · enter keep · esc clear)") {
			t.Errorf("last line = %q", got)
		}
		if refs := rowRefs(m); strings.Join(refs, ",") != "a/b#12,a/b#120,a/b#1234" {
			t.Errorf("rows = %v", refs)
		}
	})
	t.Run("cap", func(t *testing.T) {
		m := press(open(), "1", "2", "3", "4", "5", "6", "7", "8")
		if got := m.(Model).numFilter; got != "1234567" {
			t.Errorf("numFilter = %q, want 1234567", got)
		}
	})
	t.Run("backspace", func(t *testing.T) {
		m := send(open(), special(tea.KeyBackspace))
		if got := m.(Model).numFilter; got != "" {
			t.Errorf("numFilter = %q after backspace on empty", got)
		}
		m = send(press(m, "1", "2"), special(tea.KeyBackspace))
		if got := m.(Model).numFilter; got != "1" {
			t.Errorf("numFilter = %q, want 1", got)
		}
	})
	t.Run("ignored keys", func(t *testing.T) {
		m := press(open(), "a", "?", "-")
		if got := m.(Model).numFilter; got != "" {
			t.Errorf("numFilter = %q, want nothing typed", got)
		}
		if !m.(Model).prompting {
			t.Error("an ignored key closed the prompt")
		}
	})
	t.Run("quit keys", func(t *testing.T) {
		if _, cmd := open().Update(key("q")); cmd != nil {
			t.Error("q returned a Cmd while the prompt is open")
		}
		_, cmd := open().Update(key("ctrl+c"))
		if cmd == nil {
			t.Fatal("ctrl+c returned no Cmd")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Error("ctrl+c did not quit")
		}
	})
	t.Run("digits are not toggles", func(t *testing.T) {
		m := press(open(), "1", "2", "3")
		if got := headerOf(m); !strings.Contains(got, "[open merged closed] #123") {
			t.Errorf("header = %q, want all states on and #123", got)
		}
	})
	t.Run("paste", func(t *testing.T) {
		m := send(open(), tea.PasteMsg{Content: "PR #4a2"})
		if got := m.(Model).numFilter; got != "42" {
			t.Errorf("numFilter = %q, want 42", got)
		}
		m = send(open(), tea.PasteMsg{Content: "\x1b[31m99999999999\x1b[0m"})
		if got := m.(Model).numFilter; got != "3199999" {
			t.Errorf("numFilter = %q, want only digits, at most %d (3199999)", got, maxFilterDigits)
		}
		// Outside the prompt a paste is ignored.
		m = send(filterModel(120, 40, promptSnap()), tea.PasteMsg{Content: "12"})
		if got := m.(Model).numFilter; got != "" {
			t.Errorf("numFilter = %q, want a paste outside the prompt ignored", got)
		}
	})
	t.Run("enter keeps", func(t *testing.T) {
		m := send(press(open(), "1", "2"), special(tea.KeyEnter))
		if m.(Model).prompting {
			t.Error("enter left the prompt open")
		}
		if got := headerOf(m); !strings.Contains(got, "[open merged closed] #12") {
			t.Errorf("header = %q, want #12 after the state indicator", got)
		}
		if strings.HasPrefix(lastLine(m), "/ ") || !strings.Contains(lastLine(m), "j/k move") {
			t.Errorf("last line = %q, want the footer back", lastLine(m))
		}
		if refs := rowRefs(m); len(refs) != 3 {
			t.Errorf("rows = %v, want 3 kept", refs)
		}
		if got := markedRef(t, press(m, "j")); got != "a/b#120" {
			t.Errorf("j moved to %q, want a/b#120", got)
		}
	})
	t.Run("esc clears", func(t *testing.T) {
		kept := send(press(open(), "1", "2"), special(tea.KeyEnter))
		m := send(kept, special(tea.KeyEscape))
		if got := headerOf(m); strings.Contains(got, "#12") || m.(Model).numFilter != "" {
			t.Errorf("header = %q, want the kept filter cleared", got)
		}
		if refs := rowRefs(m); len(refs) != 5 {
			t.Errorf("rows = %v, want all 5 back", refs)
		}
		m = send(press(open(), "1"), special(tea.KeyEscape))
		if m.(Model).prompting || m.(Model).numFilter != "" || len(rowRefs(m)) != 5 {
			t.Error("esc while typing did not clear and close the prompt")
		}
		// A kept filter can be reopened and edited.
		m = press(kept, "/", "3")
		if got := m.(Model).numFilter; got != "123" {
			t.Errorf("numFilter = %q, want the kept digits editable", got)
		}
	})
	t.Run("filtered empty", func(t *testing.T) {
		m := press(open(), "9", "9")
		v := plain(m)
		want := "No PRs match: [open merged closed] #99 — 5 hidden by filters · esc clear · 1/2/3 toggle"
		if !strings.Contains(v, want) {
			t.Errorf("view lacks %q:\n%s", want, v)
		}
		if !strings.HasPrefix(lastLine(m), "/ 99█") {
			t.Errorf("last line = %q, want the prompt", lastLine(m))
		}
	})
}

func TestPrefixMatch(t *testing.T) {
	m := press(filterModel(120, 40, promptSnap()), "/", "1", "2")
	if refs := rowRefs(m); strings.Join(refs, ",") != "a/b#12,a/b#120,a/b#1234" {
		t.Errorf("12 matched %v, want #12, #120 and #1234 but not #312", refs)
	}
	m = press(filterModel(120, 40, selSnap(1, 10)), "/", "0", "1")
	if refs := rowRefs(m); len(refs) != 0 {
		t.Errorf("01 matched %v, want nothing", refs)
	}
	if !strings.Contains(plain(m), "No PRs match:") {
		t.Errorf("a number matching nothing must show the filtered-empty message:\n%s", plain(m))
	}
}

func TestPromptKeepsOrder(t *testing.T) {
	snap := selSnap(1200, 12, 120, 99)
	snap.PRs[2].CI = domain.CIFailing // attention order puts #120 first
	domain.Sort(snap.PRs)
	full := rowRefs(filterModel(120, 40, snap))
	m := press(filterModel(120, 40, snap), "/", "1", "2")
	var want []string
	for _, r := range full {
		if strings.HasPrefix(strings.TrimPrefix(r, "a/b#"), "12") {
			want = append(want, r)
		}
	}
	if got := rowRefs(m); strings.Join(got, ",") != strings.Join(want, ",") || len(want) != 3 {
		t.Errorf("filtered rows = %v, want %v (the full-list order)", got, want)
	}
	if want[0] != "a/b#120" {
		t.Errorf("setup: full order = %v, want #120 first", full)
	}
}

func TestRefreshWhilePrompting(t *testing.T) {
	m := press(filterModel(120, 40, promptSnap()), "/", "1", "2")
	m = refreshTo(m, 12, 3, 125, 7)
	if !m.(Model).prompting || m.(Model).numFilter != "12" {
		t.Errorf("prompting = %v, numFilter = %q, want the prompt kept open with 12", m.(Model).prompting, m.(Model).numFilter)
	}
	if refs := rowRefs(m); strings.Join(refs, ",") != "a/b#12,a/b#125" {
		t.Errorf("rows = %v, want the filter applied to the new data", refs)
	}
	if got := lastLine(m); !strings.HasPrefix(got, "/ 12█   (2 of 4 ") {
		t.Errorf("last line = %q", got)
	}
}

func TestFilterHint(t *testing.T) {
	if got := lastLine(filterModel(120, 40, promptSnap())); !strings.Contains(got, "/ PR #") {
		t.Errorf("footer = %q, want the '/ PR #' hint", got)
	}
}

func TestPromptRowBudget(t *testing.T) {
	nums := make([]int, 26)
	for i := range nums {
		nums[i] = i + 1
	}
	m := filterModel(80, 24, selSnap(nums...))
	before := len(strings.Split(plain(m), "\n"))
	after := len(strings.Split(plain(press(m, "/")), "\n"))
	if before != after || after > 24 {
		t.Errorf("lines before the prompt = %d, with it = %d, want equal and at most 24", before, after)
	}
}

package ui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// removeSnap holds a discovered PR, the saved PR octo/tools#42 and a second
// discovered PR, sorted as Fetch sorts them.
func removeSnap() domain.Snapshot {
	snap := domain.Snapshot{Viewer: "octo", PRs: []domain.PR{
		{ID: "PR_1", Ref: domain.Ref{Repo: "acme/api", Number: 1}, Title: "Api change", Author: "octo", Auto: true,
			State: domain.StateOpen, Review: domain.ReviewApproved, CI: domain.CIFailing, UpdatedAt: addUpdated},
		{ID: "PR_42", Ref: toolsRef, Title: "Tools change", Author: "sam", Manual: true, Watched: toolsRef,
			State: domain.StateOpen, Review: domain.ReviewApproved, CI: domain.CIPassing, UpdatedAt: addUpdated},
		{ID: "PR_3", Ref: domain.Ref{Repo: "zeta/web", Number: 3}, Title: "Web change", Author: "octo", Auto: true,
			State: domain.StateOpen, Review: domain.ReviewApproved, CI: domain.CIPassing, UpdatedAt: addUpdated},
	}}
	domain.Sort(snap.PRs)
	return snap
}

// selectRef moves the cursor down with j until ref is selected.
func selectRef(t *testing.T, m tea.Model, ref domain.Ref) tea.Model {
	t.Helper()
	for range 10 {
		if selectedRef(m) == ref {
			return m
		}
		m = press(m, "j")
	}
	t.Fatalf("could not select %v; selected %v", ref, selectedRef(m))
	return m
}

// rowBelow returns the ref of the row under ref in the model's rows.
func rowBelow(t *testing.T, m tea.Model, ref domain.Ref) domain.Ref {
	t.Helper()
	rows := m.(Model).rows
	i := indexOf(rows, ref)
	if i < 0 || i+1 >= len(rows) {
		t.Fatalf("no row below %v in %v", ref, rows)
	}
	return rows[i+1].Ref
}

func TestRemoveSaved(t *testing.T) {
	w := &memWatch{removed: true}
	m := addModel(t, &addFetcher{}, w, removeSnap())
	m = selectRef(t, m, toolsRef)
	below := rowBelow(t, m, toolsRef)

	m, cmd := m.Update(key("d"))
	if cmd != nil {
		t.Errorf("d returned a Cmd; asking sends nothing")
	}
	if got, want := lastLine(m), "remove octo/tools#42 from watch list? y/n"; got != want {
		t.Errorf("last line = %q, want %q", got, want)
	}
	if len(w.removes) != 0 {
		t.Fatalf("Remove called before y: %v", w.removes)
	}

	m, cmd = m.Update(key("y"))
	if cmd == nil {
		t.Fatal("y returned no Cmd")
	}
	m = drainUI(m, cmd)

	if len(w.removes) != 1 || w.removes[0] != toolsRef {
		t.Errorf("Remove calls = %v, want [%v]", w.removes, toolsRef)
	}
	if rowsHave(m, toolsRef) || countRef(m, toolsRef) != 0 {
		t.Errorf("octo/tools#42 is still listed:\n%s", plain(m))
	}
	if got, want := flashOf(m), "removed octo/tools#42"; got != want {
		t.Errorf("flash = %q, want %q", got, want)
	}
	if !strings.Contains(lastLine(m), "removed octo/tools#42") {
		t.Errorf("last line = %q, want it to hold 'removed octo/tools#42'", lastLine(m))
	}
	if got := selectedRef(m); got != below {
		t.Errorf("selected = %v, want the row below, %v", got, below)
	}
}

// confirmOn selects ref and presses d.
func confirmOn(t *testing.T, m tea.Model, ref domain.Ref) tea.Model {
	t.Helper()
	m = selectRef(t, m, ref)
	m, _ = m.Update(key("d"))
	if !m.(Model).confirm.open {
		t.Fatalf("d on %v did not open the confirm line; last line %q", ref, lastLine(m))
	}
	return m
}

// removeVia presses y on an open confirm line and delivers the result.
func removeVia(m tea.Model) tea.Model {
	m, cmd := m.Update(key("y"))
	return drainUI(m, cmd)
}

// TestRemoveConfirmCancels is the locked rule of 2026-10-12: while the confirm
// line shows, y removes, ctrl+c quits and every other key cancels, q included.
func TestRemoveConfirmCancels(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  tea.KeyPressMsg
	}{
		{"q", key("q")}, {"n", key("n")}, {"esc", special(tea.KeyEscape)}, {"a", key("a")},
		{"d", key("d")}, {"j", key("j")}, {"enter", special(tea.KeyEnter)}, {"?", key("?")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &memWatch{removed: true}
			m := confirmOn(t, addModel(t, &addFetcher{}, w, removeSnap()), toolsRef)
			m, cmd := m.Update(tc.msg)

			mm := m.(Model)
			if mm.confirm.open {
				t.Error("the confirm line is still open")
			}
			if mm.showHelp || mm.add.open || mm.detail.open {
				t.Errorf("the key was acted on: showHelp %v, add %v, detail %v", mm.showHelp, mm.add.open, mm.detail.open)
			}
			if got := lastLine(m); !strings.Contains(got, "j/k move") {
				t.Errorf("last line = %q, want the footer hints back", got)
			}
			if cmd != nil {
				t.Errorf("the key returned a Cmd (%T), want nil", cmd())
			}
			if len(w.removes) != 0 {
				t.Errorf("Remove called: %v", w.removes)
			}
			if !rowsHave(m, toolsRef) {
				t.Error("the row left the list")
			}
		})
	}
	t.Run("ctrl+c", func(t *testing.T) {
		w := &memWatch{removed: true}
		m := confirmOn(t, addModel(t, &addFetcher{}, w, removeSnap()), toolsRef)
		_, cmd := m.Update(key("ctrl+c"))
		if cmd == nil {
			t.Fatal("ctrl+c returned no Cmd")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("ctrl+c Cmd yields %T, want tea.QuitMsg", cmd())
		}
		if len(w.removes) != 0 {
			t.Errorf("Remove called: %v", w.removes)
		}
	})
}

func TestRemoveAutoOnly(t *testing.T) {
	autoRef := domain.Ref{Repo: "acme/api", Number: 1}
	w := &memWatch{removed: true}
	m := selectRef(t, addModel(t, &addFetcher{}, w, removeSnap()), autoRef)
	m, _ = m.Update(key("d"))

	if m.(Model).confirm.open {
		t.Error("d on a discovery-only row opened the confirm line")
	}
	want := "acme/api#1 is authored/assigned — only + rows can be removed"
	if got := flashOf(m); got != want {
		t.Errorf("flash = %q, want %q", got, want)
	}
	if !strings.Contains(lastLine(m), want) {
		t.Errorf("last line = %q, want it to hold %q", lastLine(m), want)
	}
	if len(w.removes) != 0 || !rowsHave(m, autoRef) {
		t.Errorf("Remove calls %v, row listed %v; nothing may change", w.removes, rowsHave(m, autoRef))
	}
}

// TestRemoveUnpin: a row that is saved and also found by discovery stays listed
// without its + marker (D-07).
func TestRemoveUnpin(t *testing.T) {
	ref := domain.Ref{Repo: "acme/api", Number: 1}
	saved := domain.Ref{Repo: "ACME/API", Number: 1}
	snap := removeSnap()
	for i := range snap.PRs {
		if snap.PRs[i].Ref == ref {
			snap.PRs[i].Manual, snap.PRs[i].Watched = true, saved
		}
	}
	w := &memWatch{removed: true}
	m := confirmOn(t, addModel(t, &addFetcher{}, w, snap), ref)
	if row := rowLine(m.(Model), "acme/api#1"); !strings.Contains(row, "+ acme/api#1") {
		t.Fatalf("setup: row = %q, want the + marker", row)
	}
	m = removeVia(m)

	if len(w.removes) != 1 || w.removes[0] != saved {
		t.Errorf("Remove calls = %v, want [%v]", w.removes, saved)
	}
	if !rowsHave(m, ref) || countRef(m, ref) != 1 {
		t.Fatalf("the row is not listed once:\n%s", plain(m))
	}
	if row := rowLine(m.(Model), "acme/api#1"); strings.Contains(row, "+ ") {
		t.Errorf("row = %q, want no + marker", row)
	}
	for _, pr := range m.(Model).snap.PRs {
		if pr.Ref == ref && (pr.Manual || pr.Watched != (domain.Ref{}) || !pr.Auto) {
			t.Errorf("row = Manual %v, Watched %v, Auto %v; want unpinned and still Auto", pr.Manual, pr.Watched, pr.Auto)
		}
	}
	if got, want := flashOf(m), "unpinned — still listed (authored/assigned)"; got != want {
		t.Errorf("flash = %q, want %q", got, want)
	}
}

// TestRemoveUnavailable: d then y works on an unavailable row like on any saved
// row (D-11).
func TestRemoveUnavailable(t *testing.T) {
	goneRef := domain.Ref{Repo: "acme/gone", Number: 5}
	w := &memWatch{removed: true}
	m := addModel(t, &addFetcher{}, w, unavailSnap())
	if h := lines(m.View().Content)[0]; !strings.Contains(h, "2 unavailable") {
		t.Fatalf("setup: header = %q, want '2 unavailable'", h)
	}
	m = confirmOn(t, m, goneRef)
	if got, want := lastLine(m), "remove acme/gone#5 from watch list? y/n"; got != want {
		t.Errorf("last line = %q, want %q", got, want)
	}
	m = removeVia(m)

	if len(w.removes) != 1 || w.removes[0] != goneRef {
		t.Errorf("Remove calls = %v, want [%v]", w.removes, goneRef)
	}
	if rowsHave(m, goneRef) {
		t.Errorf("acme/gone#5 is still listed:\n%s", plain(m))
	}
	if h := lines(m.View().Content)[0]; !strings.Contains(h, "1 unavailable") || strings.Contains(h, "2 unavailable") {
		t.Errorf("header = %q, want '1 unavailable'", h)
	}
}

// TestRemoveNotInDetail: a and d do nothing in the detail view (D-08).
func TestRemoveNotInDetail(t *testing.T) {
	w := &memWatch{removed: true}
	f := newDetailFake(fixtureDetail())
	var m tea.Model = New(t.Context(), f, Options{ClosedDays: 14, Now: fixedNow, Watchlist: w})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(fetchedMsg{snap: removeSnap()})
	m = selectRef(t, m, toolsRef)
	m, cmd := m.Update(key("enter"))
	m = runCmd(m, cmd)
	if !m.(Model).detail.open {
		t.Fatal("setup: the detail view did not open")
	}

	for _, k := range []string{"a", "d", "y"} {
		m, cmd = m.Update(key(k))
		mm := m.(Model)
		if !mm.detail.open || mm.confirm.open || mm.add.open {
			t.Errorf("after %q: detail %v, confirm %v, add %v; want the detail open and nothing else", k, mm.detail.open, mm.confirm.open, mm.add.open)
		}
		if cmd != nil {
			t.Errorf("after %q: got a Cmd (%T), want none", k, cmd())
		}
	}
	if len(w.removes) != 0 || !rowsHave(m, toolsRef) {
		t.Errorf("Remove calls %v, listed %v; nothing may change", w.removes, rowsHave(m, toolsRef))
	}
}

// TestRemoveAlreadyGone: an entry that is not in the file any more (another
// pane removed it) still takes the row off the list, without an error.
func TestRemoveAlreadyGone(t *testing.T) {
	w := &memWatch{removed: false}
	m := removeVia(confirmOn(t, addModel(t, &addFetcher{}, w, removeSnap()), toolsRef))

	if len(w.removes) != 1 {
		t.Errorf("Remove calls = %v, want 1", w.removes)
	}
	if rowsHave(m, toolsRef) {
		t.Errorf("the row is still listed:\n%s", plain(m))
	}
	if got, want := flashOf(m), "removed octo/tools#42"; got != want {
		t.Errorf("flash = %q, want %q", got, want)
	}
	if strings.Contains(plain(m), "could not") {
		t.Errorf("error text on screen:\n%s", plain(m))
	}
}

func TestRemoveError(t *testing.T) {
	w := &memWatch{removed: true, removeErr: errors.New("disk full")}
	m := removeVia(confirmOn(t, addModel(t, &addFetcher{}, w, removeSnap()), toolsRef))

	if !rowsHave(m, toolsRef) {
		t.Errorf("the row left the list although the write failed:\n%s", plain(m))
	}
	if got, want := flashOf(m), "could not update the watch list"; got != want {
		t.Errorf("flash = %q, want %q", got, want)
	}
	if !strings.Contains(lastLine(m), "could not update the watch list") {
		t.Errorf("last line = %q", lastLine(m))
	}
	if m.(Model).confirm.open {
		t.Error("the confirm line is still open")
	}
}

func TestRemoveLastRow(t *testing.T) {
	snap := domain.Snapshot{Viewer: "octo", PRs: []domain.PR{removeSnap().PRs[1]}}
	if snap.PRs[0].Ref != toolsRef {
		t.Fatalf("setup: %v", snap.PRs[0].Ref)
	}
	w := &memWatch{removed: true}
	m := removeVia(confirmOn(t, addModel(t, &addFetcher{}, w, snap), toolsRef))

	if m.(Model).state != stateEmpty {
		t.Errorf("state = %v, want stateEmpty", m.(Model).state)
	}
	if v := plain(m); !strings.Contains(v, "No open PRs authored by or assigned to @octo") {
		t.Errorf("the empty screen is not shown:\n%s", v)
	}
}

// TestRemoveStaleFetch: a refresh that started before the removal and returns
// after it is discarded and run again, so the removed row never comes back
// (RESEARCH Pitfall 1).
func TestRemoveStaleFetch(t *testing.T) {
	w := &memWatch{removed: true}
	f := &addFetcher{snap: removeSnap()}
	m := addModel(t, f, w, removeSnap())
	m, _ = m.Update(key("r"))
	started := m.(Model)
	if !started.inFlight {
		t.Fatal("setup: r did not start a refresh")
	}
	stale := fetchedMsg{gen: started.gen, epoch: started.fetchEpoch, snap: removeSnap()}

	m = removeVia(confirmOn(t, m, toolsRef))
	if rowsHave(m, toolsRef) {
		t.Fatal("setup: the row is still listed after the removal")
	}
	epochBefore := m.(Model).fetchEpoch

	m, cmd := m.Update(stale)
	if rowsHave(m, toolsRef) {
		t.Errorf("the stale refresh brought the row back:\n%s", plain(m))
	}
	if cmd == nil || m.(Model).fetchEpoch != epochBefore+1 || !m.(Model).inFlight {
		t.Errorf("no new fetch started: Cmd nil %v, epoch %d (was %d), inFlight %v", cmd == nil, m.(Model).fetchEpoch, epochBefore, m.(Model).inFlight)
	}

	// The fresh result, which no longer lists the PR, is applied.
	m, _ = m.Update(freshResult(m, domain.Snapshot{Viewer: "octo", PRs: []domain.PR{removeSnap().PRs[0]}}, nil))
	if rowsHave(m, toolsRef) || !rowsHave(m, domain.Ref{Repo: "acme/api", Number: 1}) {
		t.Errorf("after the fresh result:\n%s", plain(m))
	}
}

func TestRemoveCursorNeighbour(t *testing.T) {
	w := &memWatch{removed: true}
	m := confirmOn(t, addModel(t, &addFetcher{}, w, removeSnap()), toolsRef)
	rows := m.(Model).rows
	i := indexOf(rows, toolsRef)
	if i != 1 || len(rows) != 3 {
		t.Fatalf("setup: tools at %d of %d, want the middle of 3", i, len(rows))
	}
	below := rows[i+1].Ref
	m = removeVia(m)

	if got := selectedRef(m); got != below {
		t.Errorf("selected = %v, want the row below, %v", got, below)
	}
	if got := m.(Model).rows[m.(Model).cursor].Ref; got != below {
		t.Errorf("cursor row = %v, want %v", got, below)
	}
	if mk := markedRef(t, m); mk != below.String() {
		t.Errorf("marked row = %q, want %q", mk, below.String())
	}
}

package ui

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// addFetcher is a Fetcher that is also a Looker. Plans 04-05..04-07 reuse it.
type addFetcher struct {
	snap    domain.Snapshot
	err     error
	calls   int
	pr      domain.PR
	lookErr error
	looked  []domain.Ref
}

func (f *addFetcher) Fetch(context.Context) (domain.Snapshot, error) {
	f.calls++
	return f.snap, f.err
}

func (f *addFetcher) Lookup(_ context.Context, ref domain.Ref) (domain.PR, error) {
	f.looked = append(f.looked, ref)
	return f.pr, f.lookErr
}

// memWatch is a Watchlist that keeps nothing: it records the refs it is given
// and returns the configured values.
type memWatch struct {
	added     bool
	addErr    error
	adds      []domain.Ref
	removed   bool
	removeErr error
	removes   []domain.Ref
}

func (w *memWatch) Add(ref domain.Ref) (bool, error) {
	w.adds = append(w.adds, ref)
	return w.added, w.addErr
}

func (w *memWatch) Remove(ref domain.Ref) (bool, error) {
	w.removes = append(w.removes, ref)
	return w.removed, w.removeErr
}

// addModelWith builds a 120x40 model with the fakes, loaded with snap. after
// replaces the timer function, so no test waits on a real timer.
func addModelWith(t *testing.T, f *addFetcher, w *memWatch, snap domain.Snapshot, after func(time.Duration, tea.Msg) tea.Cmd) tea.Model {
	t.Helper()
	m := New(t.Context(), f, Options{ClosedDays: 14, Now: fixedNow, Watchlist: w})
	m.after = after
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	tm, _ = tm.Update(fetchedMsg{snap: snap})
	return tm
}

// addModel is addModelWith with timers that never fire.
func addModel(t *testing.T, f *addFetcher, w *memWatch, snap domain.Snapshot) tea.Model {
	t.Helper()
	return addModelWith(t, f, w, snap, func(time.Duration, tea.Msg) tea.Cmd { return nil })
}

// submit presses a, pastes text and presses enter.
func submit(m tea.Model, text string) (tea.Model, tea.Cmd) {
	m, _ = m.Update(key("a"))
	m, _ = m.Update(tea.PasteMsg{Content: text})
	return m.Update(special(tea.KeyEnter))
}

// drainUI runs cmd, recurses into tea.BatchMsg, passes every message except
// spinner.TickMsg to Update, and repeats with the Cmds Update returns, for at
// most 5 rounds.
func drainUI(m tea.Model, cmd tea.Cmd) tea.Model {
	cmds := []tea.Cmd{cmd}
	for round := 0; round < 5 && len(cmds) > 0; round++ {
		var next []tea.Cmd
		for _, c := range cmds {
			m, next = deliverCmd(m, c, next)
		}
		cmds = next
	}
	return m
}

func deliverCmd(m tea.Model, c tea.Cmd, next []tea.Cmd) (tea.Model, []tea.Cmd) {
	if c == nil {
		return m, next
	}
	switch msg := c().(type) {
	case nil:
	case spinner.TickMsg:
	case tea.BatchMsg:
		for _, bc := range msg {
			m, next = deliverCmd(m, bc, next)
		}
	default:
		var nc tea.Cmd
		m, nc = m.Update(msg)
		next = append(next, nc)
	}
	return m, next
}

// freshResult is a fetch result stamped with the model's generation and epoch
// mutEpoch, so no earlier add or removal discards it.
func freshResult(m tea.Model, snap domain.Snapshot, err error) tea.Msg {
	mm := m.(Model)
	return fetchedMsg{gen: mm.gen, epoch: mm.mutEpoch, snap: snap, err: err}
}

var addUpdated = time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)

// addSnap holds a discovered PR, a saved PR, an unavailable saved PR and a
// merged discovered PR.
func addSnap() domain.Snapshot {
	return domain.Snapshot{Viewer: "octo", PRs: []domain.PR{
		{ID: "PR_1", Ref: domain.Ref{Repo: "acme/api", Number: 1}, Title: "Api change", Author: "octo", Auto: true, UpdatedAt: addUpdated},
		{ID: "PR_2", Ref: domain.Ref{Repo: "acme/web", Number: 2}, Title: "Web change", Author: "sam", Manual: true,
			Watched: domain.Ref{Repo: "acme/web", Number: 2}, UpdatedAt: addUpdated},
		{Ref: domain.Ref{Repo: "acme/gone", Number: 9}, Manual: true, Watched: domain.Ref{Repo: "acme/gone", Number: 9}, Unavail: domain.NotFound()},
		{ID: "PR_3", Ref: domain.Ref{Repo: "acme/api", Number: 3}, Title: "Merged change", Author: "octo", Auto: true,
			State: domain.StateMerged, UpdatedAt: addUpdated, ClosedAt: addUpdated},
	}}
}

// toolsPR is a PR that a Lookup returns for octo/tools#42.
func toolsPR(state domain.State) domain.PR {
	return domain.PR{ID: "PR_42", Ref: domain.Ref{Repo: "octo/tools", Number: 42}, Title: "Tools change", Author: "octo",
		State: state, UpdatedAt: addUpdated}
}

func selectedRef(m tea.Model) domain.Ref { return m.(Model).selected }

func flashOf(m tea.Model) string { return m.(Model).flash }

func countRef(m tea.Model, ref domain.Ref) int {
	n := 0
	for _, pr := range m.(Model).snap.PRs {
		if pr.Ref == ref {
			n++
		}
	}
	return n
}

func rowsHave(m tea.Model, ref domain.Ref) bool {
	return slices.ContainsFunc(m.(Model).rows, func(p domain.PR) bool { return p.Ref == ref })
}

func TestAddDuplicateByRef(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		want       domain.Ref
	}{
		{"discovered, other letter case", "ACME/API#1", domain.Ref{Repo: "acme/api", Number: 1}},
		{"saved, pull URL", "https://github.com/acme/web/pull/2", domain.Ref{Repo: "acme/web", Number: 2}},
		{"unavailable saved row", "acme/gone#9", domain.Ref{Repo: "acme/gone", Number: 9}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, w := &addFetcher{}, &memWatch{added: true}
			m := addModel(t, f, w, addSnap())
			if selectedRef(m) == tc.want {
				m = press(m, "j")
			}
			if selectedRef(m) == tc.want {
				t.Fatalf("setup: %v is already selected", tc.want)
			}
			m, cmd := submit(m, tc.text)
			m = drainUI(m, cmd)

			if len(f.looked) != 0 || len(w.adds) != 0 {
				t.Errorf("looked %v, saved %v; a duplicate sends no Lookup and saves nothing", f.looked, w.adds)
			}
			if m.(Model).add.open {
				t.Error("the prompt is still open")
			}
			if got := selectedRef(m); got != tc.want {
				t.Errorf("selected = %v, want %v", got, tc.want)
			}
			if got, want := flashOf(m), "already watching "+tc.want.String(); got != want {
				t.Errorf("flash = %q, want %q", got, want)
			}
		})
	}
}

func TestAddDuplicateHidden(t *testing.T) {
	f, w := &addFetcher{}, &memWatch{added: true}
	m := addModel(t, f, w, addSnap())
	m = press(m, "2") // hide merged PRs
	before := selectedRef(m)
	merged := domain.Ref{Repo: "acme/api", Number: 3}
	if rowsHave(m, merged) {
		t.Fatal("setup: the merged PR is still visible")
	}
	m, cmd := submit(m, "acme/api#3")
	m = drainUI(m, cmd)

	if len(f.looked) != 0 || len(w.adds) != 0 {
		t.Errorf("looked %v, saved %v; want neither", f.looked, w.adds)
	}
	if got, want := flashOf(m), "already watching acme/api#3 · hidden by filters"; got != want {
		t.Errorf("flash = %q, want %q", got, want)
	}
	if got := selectedRef(m); got != before {
		t.Errorf("selected = %v, want %v (unchanged)", got, before)
	}
}

func TestAddDuplicateByID(t *testing.T) {
	// The repository was renamed: the pasted ref is not on the list, but the PR
	// GitHub returns has the node ID of a listed row.
	f, w := &addFetcher{pr: domain.PR{ID: "PR_2", Ref: domain.Ref{Repo: "acme/web", Number: 2}, Title: "Web change"}}, &memWatch{added: true}
	m := addModel(t, f, w, addSnap())
	want := domain.Ref{Repo: "acme/web", Number: 2}
	if selectedRef(m) == want {
		m = press(m, "j")
	}
	m, cmd := submit(m, "acme/old-web#2")
	m = drainUI(m, cmd)

	if len(f.looked) != 1 {
		t.Fatalf("looked %v, want one Lookup", f.looked)
	}
	if len(w.adds) != 0 {
		t.Errorf("saved %v, want nothing", w.adds)
	}
	if m.(Model).add.open {
		t.Error("the prompt is still open")
	}
	if got := selectedRef(m); got != want {
		t.Errorf("selected = %v, want %v", got, want)
	}
	if got, want := flashOf(m), "already watching acme/web#2"; got != want {
		t.Errorf("flash = %q, want %q", got, want)
	}
	if n := countRef(m, want); n != 1 {
		t.Errorf("%d rows for acme/web#2, want 1", n)
	}
}

func TestAddSavedElsewhere(t *testing.T) {
	// Add reports false: another pane saved the PR first. It is still listed.
	f, w := &addFetcher{pr: toolsPR(domain.StateOpen)}, &memWatch{added: false}
	m := addModel(t, f, w, addSnap())
	m, cmd := submit(m, "octo/tools#42")
	m = drainUI(m, cmd)

	ref := domain.Ref{Repo: "octo/tools", Number: 42}
	if len(w.adds) != 1 || w.adds[0] != ref {
		t.Errorf("saved %v, want [%v]", w.adds, ref)
	}
	if n := countRef(m, ref); n != 1 {
		t.Errorf("%d rows for %v, want 1", n, ref)
	}
	if got, want := flashOf(m), "already watching octo/tools#42"; got != want {
		t.Errorf("flash = %q, want %q", got, want)
	}
	if m.(Model).add.open {
		t.Error("the prompt is still open")
	}
}

func TestAddHiddenAfterSave(t *testing.T) {
	f, w := &addFetcher{pr: toolsPR(domain.StateMerged)}, &memWatch{added: true}
	m := addModel(t, f, w, addSnap())
	m = press(m, "2") // hide merged PRs
	before := selectedRef(m)
	m, cmd := submit(m, "octo/tools#42")
	m = drainUI(m, cmd)

	ref := domain.Ref{Repo: "octo/tools", Number: 42}
	if n := countRef(m, ref); n != 1 {
		t.Errorf("%d PRs for %v in the snapshot, want 1", n, ref)
	}
	if rowsHave(m, ref) {
		t.Error("the merged PR is on screen although the filter hides merged PRs")
	}
	if got := selectedRef(m); got != before {
		t.Errorf("selected = %v, want %v (unchanged)", got, before)
	}
	if got, want := flashOf(m), "added octo/tools#42 · hidden by filters"; got != want {
		t.Errorf("flash = %q, want %q", got, want)
	}
}

func TestAddFromEmpty(t *testing.T) {
	f, w := &addFetcher{pr: toolsPR(domain.StateOpen)}, &memWatch{added: true}
	m := addModel(t, f, w, domain.Snapshot{Viewer: "octo"})
	if m.(Model).state != stateEmpty {
		t.Fatalf("setup: state = %v, want the empty screen", m.(Model).state)
	}
	m, cmd := submit(m, "octo/tools#42")
	m = drainUI(m, cmd)

	if m.(Model).state != stateList {
		t.Errorf("state = %v, want the list screen", m.(Model).state)
	}
	ref := domain.Ref{Repo: "octo/tools", Number: 42}
	if got := selectedRef(m); got != ref {
		t.Errorf("selected = %v, want %v", got, ref)
	}
	if v := plain(m); !strings.Contains(v, "octo/tools#42") {
		t.Errorf("the new row is not on screen:\n%s", v)
	}
}

func TestStaleFetchAfterAdd(t *testing.T) {
	ref := domain.Ref{Repo: "octo/tools", Number: 42}
	// build returns a model with a refresh in flight (epoch 1) and an add done.
	build := func(t *testing.T) (tea.Model, *addFetcher, *afterRec) {
		t.Helper()
		f, w := &addFetcher{pr: toolsPR(domain.StateOpen)}, &memWatch{added: true}
		rec := &afterRec{}
		m := addModelWith(t, f, w, addSnap(), rec.after)
		m, _ = m.Update(key("r")) // a refresh starts: epoch 1 is in flight
		if mm := m.(Model); !mm.inFlight || mm.fetchEpoch != 1 {
			t.Fatalf("setup: inFlight=%v fetchEpoch=%d, want true and 1", mm.inFlight, mm.fetchEpoch)
		}
		m, cmd := submit(m, "octo/tools#42")
		m = drainUI(m, cmd)
		if countRef(m, ref) != 1 {
			t.Fatalf("setup: the added PR is not listed")
		}
		return m, f, rec
	}

	t.Run("discarded and refetched", func(t *testing.T) {
		m, f, _ := build(t)
		old := addSnap() // the epoch-1 fetch read watchlist.json before the add
		var cmd tea.Cmd
		m, cmd = m.Update(fetchedMsg{gen: m.(Model).gen, epoch: 1, snap: old})
		if countRef(m, ref) != 1 {
			t.Fatal("the stale result removed the added row")
		}
		if cmd == nil || !m.(Model).inFlight {
			t.Fatalf("cmd=%v inFlight=%v; a new fetch should have started", cmd != nil, m.(Model).inFlight)
		}
		withNew := addSnap()
		withNew.PRs = append(withNew.PRs, toolsPR(domain.StateOpen))
		f.snap = withNew
		f.calls = 0
		m = drainUI(m, cmd)
		if f.calls != 1 {
			t.Errorf("fetches = %d, want 1", f.calls)
		}
		if mm := m.(Model); mm.inFlight || countRef(m, ref) != 1 {
			t.Errorf("inFlight=%v rows for the PR=%d; the epoch-2 result should apply", mm.inFlight, countRef(m, ref))
		}
	})

	t.Run("pause arms the tick instead", func(t *testing.T) {
		m, f, rec := build(t)
		mm := m.(Model)
		mm.pausedUntil = fixedNow().Add(10 * time.Minute)
		m = mm
		f.calls = 0
		var cmd tea.Cmd
		m, cmd = m.Update(fetchedMsg{gen: mm.gen, epoch: 1, snap: addSnap()})
		if m.(Model).inFlight {
			t.Error("a fetch started during the pause")
		}
		if cmd == nil {
			t.Fatal("no tick was armed")
		}
		last := rec.calls[len(rec.calls)-1]
		if last.d != 10*time.Minute || last.msg != (tickMsg{gen: mm.gen}) {
			t.Errorf("armed %v %#v, want 10m tickMsg{gen: %d}", last.d, last.msg, mm.gen)
		}
		m = drainUI(m, cmd)
		if f.calls != 0 {
			t.Errorf("fetches = %d during the pause, want 0", f.calls)
		}
		if countRef(m, ref) != 1 {
			t.Error("the added row vanished")
		}
		// A result stamped after the add applies normally.
		withNew := addSnap()
		withNew.PRs = append(withNew.PRs, toolsPR(domain.StateOpen))
		m, _ = m.Update(freshResult(m, withNew, nil))
		if n := len(m.(Model).snap.PRs); n != 5 {
			t.Errorf("PRs after a fresh result = %d, want 5", n)
		}
	})
}

func TestFetchWithoutEpochApplies(t *testing.T) {
	m := newModel(&fakeFetcher{}, 14)
	m, _ = m.Update(result(m, addSnap(), nil))
	if m.(Model).state != stateList || len(m.(Model).snap.PRs) != 4 {
		t.Errorf("state=%v PRs=%d, want the list with 4 PRs", m.(Model).state, len(m.(Model).snap.PRs))
	}
}

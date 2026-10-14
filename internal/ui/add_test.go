package ui

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sbartram/prv/internal/domain"
	"github.com/sbartram/prv/internal/watchlist"
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
	// note and removeNote are the StartedOver values AddRef and RemoveRef report.
	note, removeNote string
}

func (w *memWatch) AddRef(ref domain.Ref) (watchlist.Result, error) {
	w.adds = append(w.adds, ref)
	return watchlist.Result{Changed: w.added, StartedOver: w.note}, w.addErr
}

func (w *memWatch) RemoveRef(ref domain.Ref) (watchlist.Result, error) {
	w.removes = append(w.removes, ref)
	return watchlist.Result{Changed: w.removed, StartedOver: w.removeNote}, w.removeErr
}

// addModelWith builds a 120x40 model with the fakes, loaded with snap. after
// replaces the timer function, so no test waits on a real timer.
func addModelWith(t *testing.T, f Fetcher, w *memWatch, snap domain.Snapshot, after func(time.Duration, tea.Msg) tea.Cmd) tea.Model {
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

// TestAddStartedOverNotice: an add that replaced a broken watchlist.json says where
// the old list went, for warnFor and ahead of any other footer text (D-04).
func TestAddStartedOverNotice(t *testing.T) {
	const note = "watchlist.json.corrupt-0a1b2c3d"
	const notice = "previous list saved to " + note
	ref := domain.Ref{Repo: "octo/tools", Number: 42}
	cases := []struct {
		name   string
		added  bool
		merged bool // the merged PR is hidden after pressing 2
		want   string
	}{
		{"visible row", true, false, notice},
		{"hidden by filters", true, true, notice + " · added octo/tools#42 · hidden by filters"},
		{"already saved", false, false, notice + " · already watching octo/tools#42"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state := domain.StateOpen
			if c.merged {
				state = domain.StateMerged
			}
			f, w := &addFetcher{pr: toolsPR(state)}, &memWatch{added: c.added, note: note}
			m := addModel(t, f, w, addSnap())
			if c.merged {
				m = press(m, "2")
			}
			m, cmd := submit(m, "octo/tools#42")
			m = drainUI(m, cmd)
			if got := flashOf(m); got != c.want {
				t.Errorf("flash = %q, want %q", got, c.want)
			}
			if got, want := m.(Model).flashUntil, fixedNow().Add(warnFor); !got.Equal(want) {
				t.Errorf("flashUntil = %v, want %v (warnFor)", got, want)
			}
			if !c.merged && !rowsHave(m, ref) {
				t.Error("the new row is not listed")
			}
		})
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

// ---- the input: where a works, typing, errors, esc, pause, widths (task 2) ----

// addDetailFetcher is an addFetcher that can also load details, so the detail
// view can be opened.
type addDetailFetcher struct{ addFetcher }

func (f *addDetailFetcher) FetchDetail(context.Context, string) (domain.Detail, error) {
	return domain.Detail{}, nil
}

// limitErr is a rate-limited request error, like *github.Error with a Retry-After.
type limitErr struct{}

func (limitErr) Error() string                             { return "secondary limit, token ghp_SECRET" }
func (limitErr) Short() string                             { return "rate limited" }
func (limitErr) Backoff() (time.Duration, time.Time, bool) { return time.Minute, time.Time{}, true }

func addOpen(m tea.Model) bool { return m.(Model).add.open }

func TestAddOnlyOnListScreens(t *testing.T) {
	f, w := &addFetcher{}, &memWatch{added: true}
	sized := func(m tea.Model) tea.Model { m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40}); return m }
	fresh := func() Model { return New(t.Context(), f, Options{ClosedDays: 14, Now: fixedNow, Watchlist: w}) }

	opens := map[string]tea.Model{
		"list":           addModel(t, f, w, addSnap()),
		"empty":          addModel(t, f, w, domain.Snapshot{Viewer: "octo"}),
		"filtered-empty": press(addModel(t, f, w, domain.Snapshot{Viewer: "octo", PRs: addSnap().PRs[:1]}), "1"),
	}
	if fe := opens["filtered-empty"].(Model); fe.state != stateList || len(fe.rows) != 0 {
		t.Fatalf("setup: state=%v rows=%d, want the filtered-empty screen", fe.state, len(fe.rows))
	}
	for name, m := range opens {
		t.Run("opens on "+name, func(t *testing.T) {
			m = press(m, "a")
			if !addOpen(m) {
				t.Fatal("the prompt did not open")
			}
			if got := lastLine(m); !strings.Contains(got, "add PR: █") {
				t.Errorf("last line = %q, want the prompt", got)
			}
		})
	}

	t.Run("loading", func(t *testing.T) {
		if m := press(sized(fresh()), "a"); addOpen(m) {
			t.Error("the prompt opened on the loading screen")
		}
	})
	t.Run("error panel", func(t *testing.T) {
		var m tea.Model = sized(fresh())
		m, _ = m.Update(fetchedMsg{err: errors.New("boom")})
		if m = press(m, "a"); addOpen(m) {
			t.Error("the prompt opened on the error screen")
		}
	})
	t.Run("detail view", func(t *testing.T) {
		df := &addDetailFetcher{}
		m := addModelWith(t, df, w, addSnap(), func(time.Duration, tea.Msg) tea.Cmd { return nil })
		m = press(m, "j") // an ordinary row
		m, _ = m.Update(special(tea.KeyEnter))
		if !m.(Model).detail.open {
			t.Fatal("setup: the detail view did not open")
		}
		m = press(m, "a")
		if addOpen(m) || !m.(Model).detail.open {
			t.Errorf("add open=%v detail open=%v; a does nothing in the detail view", addOpen(m), m.(Model).detail.open)
		}
	})
	t.Run("number prompt", func(t *testing.T) {
		m := press(addModel(t, f, w, addSnap()), "/", "a")
		if addOpen(m) || !m.(Model).prompting || m.(Model).numFilter != "" {
			t.Errorf("add open=%v prompting=%v numFilter=%q; a is ignored in the number prompt", addOpen(m), m.(Model).prompting, m.(Model).numFilter)
		}
	})
	t.Run("help overlay", func(t *testing.T) {
		m := press(addModel(t, f, w, addSnap()), "?")
		if !m.(Model).showHelp {
			t.Fatal("setup: help did not open")
		}
		m = press(m, "a")
		if addOpen(m) || m.(Model).showHelp {
			t.Errorf("add open=%v help open=%v; a only closes the overlay", addOpen(m), m.(Model).showHelp)
		}
	})
}

func TestAddNotAvailable(t *testing.T) {
	t.Run("no Looker", func(t *testing.T) {
		m := modelIn(t, "list")
		m = press(m, "a")
		if addOpen(m) || flashOf(m) != "adding PRs is not available" {
			t.Errorf("add open=%v flash=%q", addOpen(m), flashOf(m))
		}
	})
	t.Run("no Watchlist", func(t *testing.T) {
		var m tea.Model = New(t.Context(), &addFetcher{}, Options{ClosedDays: 14, Now: fixedNow})
		m, _ = m.Update(fetchedMsg{snap: onePR()})
		m = press(m, "a")
		if addOpen(m) || flashOf(m) != "adding PRs is not available" {
			t.Errorf("add open=%v flash=%q", addOpen(m), flashOf(m))
		}
	})
}

func TestAddTyping(t *testing.T) {
	f, w := &addFetcher{}, &memWatch{}
	m := press(addModel(t, f, w, addSnap()), "a", "q", "a", "d", "1", "/", ":")
	if got := m.(Model).add.text; got != "qad1/:" {
		t.Errorf("text = %q, want %q", got, "qad1/:")
	}
	if !addOpen(m) {
		t.Error("a typed q closed the prompt or quit")
	}
	if m.(Model).show != domain.AllStates || m.(Model).numFilter != "" {
		t.Error("a typed 1 or / acted on the list filters")
	}
	m, _ = m.Update(special(tea.KeyBackspace))
	if got := m.(Model).add.text; got != "qad1/" {
		t.Errorf("after backspace text = %q, want %q", got, "qad1/")
	}
	if _, cmd := m.Update(key("ctrl+c")); cmd == nil {
		t.Error("ctrl+c returned no command")
	} else if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("ctrl+c did not quit")
	}
	if _, cmd := m.Update(key("q")); cmd != nil {
		t.Error("q returned a command (a quit) while typing")
	}

	t.Run("300 runes keep 256", func(t *testing.T) {
		m := press(addModel(t, f, w, addSnap()), "a")
		for range 300 {
			m, _ = m.Update(key("x"))
		}
		if n := len([]rune(m.(Model).add.text)); n != 256 {
			t.Errorf("text has %d runes, want 256", n)
		}
	})
}

func TestAddPasteSanitised(t *testing.T) {
	f, w := &addFetcher{}, &memWatch{}
	m := addModel(t, f, w, addSnap())
	m, _ = m.Update(key("a"))
	m, _ = m.Update(tea.PasteMsg{Content: "\x1b[31macme/api#1\x1b[0m\u202e"})
	if got := m.(Model).add.text; got != "acme/api#1" {
		t.Errorf("text = %q, want %q", got, "acme/api#1")
	}

	t.Run("two lines are refused as one reference", func(t *testing.T) {
		m, cmd := submit(addModel(t, f, w, addSnap()), "acme/api#1\r\nacme/web#2")
		m = drainUI(m, cmd)
		if got := lastLine(m); !strings.Contains(got, "✗ not a PR reference · esc cancel") {
			t.Errorf("last line = %q, want the not-a-reference error", got)
		}
		if len(f.looked) != 0 {
			t.Errorf("looked %v, want no Lookup", f.looked)
		}
	})
	t.Run("a long paste is cut", func(t *testing.T) {
		m := addModel(t, f, w, addSnap())
		m, _ = m.Update(key("a"))
		m, _ = m.Update(tea.PasteMsg{Content: strings.Repeat("x", 1000)})
		if n := len([]rune(m.(Model).add.text)); n != 256 {
			t.Errorf("text has %d runes, want 256", n)
		}
	})
}

func TestAddParseErrorKeepsText(t *testing.T) {
	f, w := &addFetcher{}, &memWatch{}
	m, cmd := submit(addModel(t, f, w, addSnap()), "github.com/o/r/issues/3")
	m = drainUI(m, cmd)
	if got := lastLine(m); !strings.Contains(got, "✗ that is an issue, not a PR · esc cancel") || !strings.Contains(got, "github.com/o/r/issues/3") {
		t.Errorf("last line = %q, want the issue error and the kept text", got)
	}
	if len(f.looked) != 0 || len(w.adds) != 0 {
		t.Errorf("looked %v saved %v, want neither", f.looked, w.adds)
	}
	m, _ = m.Update(key("x"))
	got := lastLine(m)
	if strings.Contains(got, "✗") || !strings.Contains(got, "(URL or owner/repo#n") {
		t.Errorf("last line after typing = %q, want the input hint and no error", got)
	}
}

func TestAddLookupErrors(t *testing.T) {
	sso := domain.Unavail{Kind: domain.UnavailSSO, Reason: "SSO: authorize your token for acme", Help: "run gh auth refresh"}
	for _, tc := range []struct {
		name    string
		pr      domain.PR
		lookErr error
		want    []string
	}{
		{"not found", domain.PR{Unavail: domain.NotFound()}, nil, []string{"✗ not found (deleted, renamed, or no access)"}},
		{"sso", domain.PR{Unavail: sso}, nil, []string{"✗ SSO: authorize your token for acme", "run gh auth refresh"}},
		{"request failed", domain.PR{}, &shortErr{short: "network"}, []string{"✗ could not check (network)"}},
		{"rate limited", domain.PR{}, limitErr{}, []string{"✗ could not check (rate limited)"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, w := &addFetcher{pr: tc.pr, lookErr: tc.lookErr}, &memWatch{added: true}
			m, cmd := submit(addModel(t, f, w, addSnap()), "acme/new#7")
			m = drainUI(m, cmd)
			got := lastLine(m)
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("last line = %q, want it to contain %q", got, want)
				}
			}
			if !strings.Contains(got, "acme/new#7") || !addOpen(m) {
				t.Errorf("the text did not stay: open=%v line=%q", addOpen(m), got)
			}
			if len(w.adds) != 0 {
				t.Errorf("saved %v, want nothing", w.adds)
			}
			if tc.name == "rate limited" && !m.(Model).pausedUntil.After(fixedNow()) {
				t.Errorf("pausedUntil = %v, want a pause after a rate-limited Lookup", m.(Model).pausedUntil)
			}
		})
	}
}

func TestAddDuringPause(t *testing.T) {
	f, w := &addFetcher{}, &memWatch{added: true}
	m := addModel(t, f, w, addSnap())
	mm := m.(Model)
	mm.pausedUntil = fixedNow().Add(10 * time.Minute)
	m = mm
	m, cmd := submit(m, "acme/new#7")
	m = drainUI(m, cmd)
	if got := lastLine(m); !strings.Contains(got, "✗ rate limited — resumes 14:42 · esc cancel") {
		t.Errorf("last line = %q, want the pause error", got)
	}
	if len(f.looked) != 0 || len(w.adds) != 0 {
		t.Errorf("looked %v saved %v, want neither during a pause", f.looked, w.adds)
	}
}

func TestAddEscDropsLateResult(t *testing.T) {
	f, w := &addFetcher{pr: toolsPR(domain.StateOpen)}, &memWatch{added: true}
	m := addModel(t, f, w, addSnap())
	m, first := submit(m, "octo/tools#42")
	seq := m.(Model).addSeq
	if !m.(Model).add.checking {
		t.Fatal("setup: the prompt is not checking")
	}
	m, _ = m.Update(special(tea.KeyEscape))
	if addOpen(m) {
		t.Fatal("esc did not close the prompt while checking")
	}

	// The Lookup finishes after esc: dropped.
	m = drainUI(m, first)
	if len(w.adds) != 0 || countRef(m, domain.Ref{Repo: "octo/tools", Number: 42}) != 0 {
		t.Errorf("a late Lookup result changed things: saved %v", w.adds)
	}
	// So is a late save result.
	m, _ = m.Update(addSavedMsg{seq: seq, pr: toolsPR(domain.StateOpen), added: true})
	if countRef(m, domain.Ref{Repo: "octo/tools", Number: 42}) != 0 {
		t.Error("a late save result listed the PR")
	}

	// A new add afterwards works, and a result of the old attempt still does not count.
	f.pr = domain.PR{ID: "PR_77", Ref: domain.Ref{Repo: "octo/other", Number: 77}, Title: "Other", UpdatedAt: addUpdated}
	m, second := submit(m, "octo/other#77")
	m, _ = m.Update(addCheckedMsg{seq: seq, ref: domain.Ref{Repo: "octo/tools", Number: 42}, pr: toolsPR(domain.StateOpen)})
	if len(w.adds) != 0 {
		t.Errorf("an old Lookup result started a save: %v", w.adds)
	}
	m = drainUI(m, second)
	if want := (domain.Ref{Repo: "octo/other", Number: 77}); len(w.adds) != 1 || w.adds[0] != want {
		t.Errorf("saved %v, want [%v]", w.adds, want)
	}
}

// savingModel submits octo/tools#42, delivers the lookup answer and returns the
// model in the saving state, with the save Cmd that has not run yet.
func savingModel(t *testing.T, w *memWatch) (tea.Model, tea.Cmd) {
	t.Helper()
	f := &addFetcher{pr: toolsPR(domain.StateOpen)}
	m, _ := submit(addModel(t, f, w, addSnap()), "octo/tools#42")
	m, save := m.Update(addCheckedMsg{seq: m.(Model).addSeq, ref: domain.Ref{Repo: "octo/tools", Number: 42}, pr: toolsPR(domain.StateOpen)})
	return m, save
}

// TestAddEscIgnoredWhileSaving: once the save is dispatched, esc, typed keys and
// paste change nothing and the result applies (D-05, WR-05). ctrl+c still quits.
func TestAddEscIgnoredWhileSaving(t *testing.T) {
	w := &memWatch{added: true}
	m, save := savingModel(t, w)
	if save == nil {
		t.Fatal("the lookup answer returned no save Cmd")
	}
	if mm := m.(Model); !mm.add.saving || mm.add.checking {
		t.Fatalf("saving = %v, checking = %v, want true, false", mm.add.saving, mm.add.checking)
	}

	m, _ = m.Update(special(tea.KeyEscape))
	if !addOpen(m) || !m.(Model).add.saving {
		t.Fatal("esc closed the prompt or ended the saving state")
	}
	m, _ = m.Update(key("x"))
	m, _ = m.Update(tea.PasteMsg{Content: "zzz"})
	if got, want := m.(Model).add.text, "octo/tools#42"; got != want {
		t.Errorf("text = %q, want %q (typing and paste are ignored)", got, want)
	}

	if _, cmd := m.Update(key("ctrl+c")); cmd == nil {
		t.Error("ctrl+c returned no Cmd")
	} else if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c Cmd gave %T, want tea.QuitMsg", cmd())
	}

	m = drainUI(m, save)
	ref := domain.Ref{Repo: "octo/tools", Number: 42}
	if n := countRef(m, ref); n != 1 {
		t.Errorf("%d rows for %v, want 1", n, ref)
	}
	if got := selectedRef(m); got != ref {
		t.Errorf("selected = %v, want %v", got, ref)
	}
	if addOpen(m) {
		t.Error("the prompt is still open after the save")
	}
	if len(w.adds) != 1 || w.adds[0] != ref {
		t.Errorf("saved %v, want [%v]", w.adds, ref)
	}
}

// TestAddSavingLine: the prompt says it is saving and offers no cancel.
func TestAddSavingLine(t *testing.T) {
	m, _ := savingModel(t, &memWatch{added: true})
	got := lastLine(m)
	if !strings.Contains(got, "saving octo/tools#42") {
		t.Errorf("last line = %q, want it to contain 'saving octo/tools#42'", got)
	}
	for _, bad := range []string{"esc cancel", "checking"} {
		if strings.Contains(got, bad) {
			t.Errorf("last line = %q, must not contain %q", got, bad)
		}
	}
}

// TestAddSpinnerWhileSaving: the spinner keeps turning during a slow save.
func TestAddSpinnerWhileSaving(t *testing.T) {
	m, _ := savingModel(t, &memWatch{added: true})
	mm := m.(Model)
	if mm.inFlight || !mm.add.saving {
		t.Fatalf("setup: inFlight = %v, saving = %v, want false, true", mm.inFlight, mm.add.saving)
	}
	if _, cmd := m.Update(mm.spin.Tick()); cmd == nil {
		t.Error("the spinner tick returned no Cmd while saving; the spinner froze")
	}
}

func TestAddSaveError(t *testing.T) {
	f, w := &addFetcher{pr: toolsPR(domain.StateOpen)}, &memWatch{addErr: errors.New("disk full")}
	m, cmd := submit(addModel(t, f, w, addSnap()), "octo/tools#42")
	m = drainUI(m, cmd)
	if got := lastLine(m); !strings.Contains(got, "✗ could not save (write failed)") || !strings.Contains(got, "octo/tools#42") {
		t.Errorf("last line = %q, want the save error and the kept text", got)
	}
	if !addOpen(m) {
		t.Error("the prompt closed after a failed save")
	}
	if n := countRef(m, domain.Ref{Repo: "octo/tools", Number: 42}); n != 0 {
		t.Errorf("%d rows for the PR after a failed save, want 0", n)
	}
}

// saveFailures are the store errors saveReason must turn into a short fixed
// phrase. The wrapped ones carry full paths, as the real store's errors do.
func saveFailures() []struct {
	name   string
	err    error
	reason string // the phrase inside "could not save (...)" / "could not update (...)"
} {
	return []struct {
		name   string
		err    error
		reason string
	}{
		{"busy", watchlist.ErrBusy, "another prv is writing the list"},
		{"version", &watchlist.VersionError{Found: 2}, "watchlist.json is version 2; this prv writes version 1"},
		{"no version", &watchlist.VersionError{Missing: true}, "watchlist.json has no version number; this prv writes version 1"},
		{"permission", fmt.Errorf("reading /home/u/.local/share/prv/watchlist.json: %w",
			&fs.PathError{Op: "open", Path: "/home/u/.local/share/prv/watchlist.json", Err: syscall.EACCES}), "permission denied"},
		{"disk full", fmt.Errorf("writing /home/u/x: %w", &fs.PathError{Op: "write", Path: "/home/u/x", Err: syscall.ENOSPC}), "disk full"},
		{"other", errors.New("\x1b[31m/secret/path exploded"), "write failed"},
	}
}

// noLeak fails the test when text holds a path fragment or a control character.
func noLeak(t *testing.T, text string) {
	t.Helper()
	for _, bad := range []string{"home", "secret"} {
		if strings.Contains(text, bad) {
			t.Errorf("%q leaks %q", text, bad)
		}
	}
	for _, r := range text {
		if unicode.IsControl(r) {
			t.Errorf("%q holds the control character %U", text, r)
		}
	}
}

// TestAddSaveReasons: a failed save says why in a few fixed words and never shows
// a path or OS text (D-06, IN-02).
func TestAddSaveReasons(t *testing.T) {
	cases := append(saveFailures(), struct {
		name   string
		err    error
		reason string
	}{"full", watchlist.ErrFull, ""})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want := "could not save (" + c.reason + ")"
			if c.err == watchlist.ErrFull {
				want = "the watch list is full (500 PRs); remove a PR first"
			}
			f, w := &addFetcher{pr: toolsPR(domain.StateOpen)}, &memWatch{addErr: c.err}
			m, cmd := submit(addModel(t, f, w, addSnap()), "octo/tools#42")
			m = drainUI(m, cmd)
			mm := m.(Model)
			if mm.add.errText != want {
				t.Errorf("errText = %q, want %q", mm.add.errText, want)
			}
			if !mm.add.open || mm.add.text != "octo/tools#42" {
				t.Errorf("open = %v, text = %q, want the prompt open with the text kept", mm.add.open, mm.add.text)
			}
			noLeak(t, mm.add.errText)
		})
	}
}

func TestAddPromptWidths(t *testing.T) {
	url := "https://github.com/acme/" + strings.Repeat("a", 96)
	if len([]rune(url)) != 120 {
		t.Fatalf("setup: url has %d runes", len([]rune(url)))
	}
	for _, w := range []int{120, 60, 40, 20} {
		for _, errState := range []bool{false, true} {
			name := "typing"
			if errState {
				name = "error"
			}
			t.Run(name+"/"+strconv.Itoa(w), func(t *testing.T) {
				f, wl := &addFetcher{}, &memWatch{}
				m := addModel(t, f, wl, addSnap())
				m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: 24})
				m, _ = m.Update(key("a"))
				m, _ = m.Update(tea.PasteMsg{Content: url})
				if errState {
					m, _ = m.Update(special(tea.KeyEnter))
				}
				got := lastLine(m)
				if lipgloss.Width(got) > w {
					t.Errorf("last line is %d cells wide in a %d-cell pane: %q", lipgloss.Width(got), w, got)
				}
				if !strings.Contains(got, "█") {
					t.Errorf("last line has no cursor: %q", got)
				}
				if errState {
					if !strings.Contains(got, "✗ not") {
						t.Errorf("last line has no start of the error: %q", got)
					}
					if w >= 60 && !strings.Contains(got, "✗ not a PR reference · esc cancel") {
						t.Errorf("last line lost the error text: %q", got)
					}
				} else if strings.Contains(got, "(URL or") {
					t.Errorf("last line kept the hint although the text is long: %q", got)
				}
				if w == 120 && !strings.Contains(got, "…") {
					t.Errorf("last line does not show that the text is cut: %q", got)
				}
			})
		}
	}

	t.Run("not sized", func(t *testing.T) {
		var m tea.Model = New(t.Context(), &addFetcher{}, Options{ClosedDays: 14, Now: fixedNow, Watchlist: &memWatch{}})
		m, _ = m.Update(fetchedMsg{snap: addSnap()})
		m, _ = m.Update(key("a"))
		m, _ = m.Update(tea.PasteMsg{Content: url})
		if got := lastLine(m); !strings.Contains(got, url+"█") {
			t.Errorf("last line = %q, want the full text and the cursor", got)
		}
	})
}

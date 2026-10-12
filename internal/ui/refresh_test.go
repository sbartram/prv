package ui

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// afterRec records every refresh timer the model arms. The recorder, not the
// returned Cmd, shows what was armed, so no test ever waits on a real timer.
type afterRec struct {
	calls []afterCall
}

type afterCall struct {
	d   time.Duration
	msg tea.Msg
}

func (r *afterRec) after(d time.Duration, msg tea.Msg) tea.Cmd {
	r.calls = append(r.calls, afterCall{d, msg})
	return func() tea.Msg { return nil }
}

// recModel builds a sized model whose timers are recorded. now may be nil.
func recModel(f Fetcher, interval time.Duration, now func() time.Time) (tea.Model, *afterRec) {
	m := New(context.Background(), f, Options{ClosedDays: 14, Interval: interval, Now: now})
	rec := &afterRec{}
	m.after = rec.after
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return tm, rec
}

func TestInitArmsNoTick(t *testing.T) {
	f := &fakeFetcher{snap: onePR()}
	m, rec := recModel(f, 0, nil)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned no command")
	}
	if len(rec.calls) != 0 {
		t.Errorf("Init armed %d timers, want 0", len(rec.calls))
	}
	if f.calls != 0 {
		t.Errorf("building Init's command ran %d fetches, want 0", f.calls)
	}
}

func TestAutoRefreshTickChain(t *testing.T) {
	t.Run("default interval", func(t *testing.T) {
		f := &fakeFetcher{snap: onePR()}
		m, rec := recModel(f, 0, nil)

		m = runCmd(m, m.Init())
		if f.calls != 1 {
			t.Fatalf("fetches after Init = %d, want 1", f.calls)
		}
		if len(rec.calls) != 1 {
			t.Fatalf("armed timers after the first result = %d, want 1", len(rec.calls))
		}
		if got := rec.calls[0]; got.d != 60*time.Second || got.msg != (tickMsg{gen: 0}) {
			t.Fatalf("armed %v %#v, want 60s tickMsg{gen: 0}", got.d, got.msg)
		}

		m, cmd := m.Update(rec.calls[0].msg)
		if cmd == nil {
			t.Fatal("the tick started no fetch")
		}
		m = runCmd(m, cmd)
		if f.calls != 2 {
			t.Errorf("fetches after the tick = %d, want 2", f.calls)
		}
		if len(rec.calls) != 2 {
			t.Errorf("armed timers after the second result = %d, want 2 (one per result)", len(rec.calls))
		}
		_ = m
	})
	t.Run("configured interval", func(t *testing.T) {
		f := &fakeFetcher{snap: onePR()}
		m, rec := recModel(f, 20*time.Second, nil)
		m = runCmd(m, m.Init())
		_ = m
		if len(rec.calls) != 1 || rec.calls[0].d != 20*time.Second {
			t.Errorf("armed = %+v, want one 20s timer", rec.calls)
		}
	})
	t.Run("a failed result also arms one tick", func(t *testing.T) {
		f := &fakeFetcher{err: errors.New("boom")}
		m, rec := recModel(f, 0, nil)
		m = runCmd(m, m.Init())
		_ = m
		if len(rec.calls) != 1 || rec.calls[0].d != 60*time.Second {
			t.Errorf("armed = %+v, want one 60s timer", rec.calls)
		}
	})
}

func TestTickWhileInFlightDropped(t *testing.T) {
	f := &fakeFetcher{snap: onePR()}
	m, rec := recModel(f, 0, nil)
	m = runCmd(m, m.Init())
	tick := rec.calls[0].msg

	m, first := m.Update(tick)
	if first == nil {
		t.Fatal("the first tick started no fetch")
	}
	// The fetch has not finished: a second tick must start nothing.
	m, second := m.Update(tick)
	if second != nil {
		t.Error("a tick while a fetch runs returned a command")
	}
	m = runCmd(m, first)
	if f.calls != 2 {
		t.Errorf("fetches = %d, want 2 (Init and one tick)", f.calls)
	}
	if len(rec.calls) != 2 {
		t.Errorf("armed timers = %d, want 2: the next tick comes from the fetch result", len(rec.calls))
	}
	_ = m
}

func TestStaleTickDropped(t *testing.T) {
	f := &fakeFetcher{snap: onePR()}
	m, rec := recModel(f, 0, nil)
	m = runCmd(m, m.Init())

	m, cmd := m.Update(tickMsg{gen: 7})
	if cmd != nil {
		t.Error("a tick from another generation returned a command")
	}
	if f.calls != 1 || len(rec.calls) != 1 {
		t.Errorf("fetches = %d, timers = %d, want 1 and 1", f.calls, len(rec.calls))
	}
	_ = m
}

func TestLastSuccessTime(t *testing.T) {
	at := time.Date(2026, 10, 11, 14, 32, 5, 0, time.UTC)
	f := &fakeFetcher{snap: onePR()}
	m, _ := recModel(f, 0, func() time.Time { return at })
	if !m.(Model).lastOK.IsZero() {
		t.Fatal("lastOK is set before any fetch")
	}
	m = runCmd(m, m.Init())
	if got := m.(Model).lastOK; !got.Equal(at) {
		t.Errorf("lastOK = %v, want %v", got, at)
	}
}

// result builds a fetch result stamped with the model's current generation.
func result(m tea.Model, snap domain.Snapshot, err error) tea.Msg {
	return fetchedMsg{gen: m.(Model).gen, snap: snap, err: err}
}

// clock is a test clock for Options.Now.
type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func TestFailedRefreshKeepsData(t *testing.T) {
	loaded := time.Date(2026, 10, 11, 14, 32, 5, 0, time.UTC)
	failed := time.Date(2026, 10, 11, 14, 33, 5, 0, time.UTC)

	f := &fakeFetcher{snap: onePR()}
	clk := &clock{t: loaded}
	m, rec := recModel(f, 0, clk.now)
	m = runCmd(m, m.Init())
	before := plain(m)
	if !strings.Contains(before, "acme/api#1") {
		t.Fatalf("setup: no row on screen:\n%s", before)
	}

	f.err = errors.New("boom")
	clk.t = failed
	for i := 1; i <= 2; i++ {
		tick := rec.calls[len(rec.calls)-1].msg
		var cmd tea.Cmd
		m, cmd = m.Update(tick)
		m = runCmd(m, cmd)

		v := plain(m)
		if !strings.Contains(v, "acme/api#1") || !strings.Contains(v, "A change") {
			t.Errorf("failure %d: the rows left the screen:\n%s", i, v)
		}
		if strings.Contains(v, "Could not load your PRs") {
			t.Errorf("failure %d: the error panel replaced the data:\n%s", i, v)
		}
		// Only the header line changes (the D-07 marker, tested in
		// TestFailureMarker); the rows are the same and not dimmed.
		if got, want := strings.Join(strings.Split(v, "\n")[1:], "\n"), strings.Join(strings.Split(before, "\n")[1:], "\n"); got != want {
			t.Errorf("failure %d: the rows changed:\n%s", i, v)
		}
		if got := m.(Model).lastOK; !got.Equal(loaded) {
			t.Errorf("failure %d: lastOK = %v, want %v", i, got, loaded)
		}
		if got := m.(Model).failedAt; !got.Equal(failed) {
			t.Errorf("failure %d: failedAt = %v, want %v", i, got, failed)
		}
		if len(rec.calls) != 1+i {
			t.Errorf("failure %d: armed timers = %d, want %d (one per result)", i, len(rec.calls), 1+i)
		}
	}

	// A later success clears the failure.
	f.err = nil
	clk.t = failed.Add(time.Minute)
	m, cmd := m.Update(rec.calls[len(rec.calls)-1].msg)
	m = runCmd(m, cmd)
	if !m.(Model).failedAt.IsZero() {
		t.Error("a successful refresh did not clear failedAt")
	}
	if got := m.(Model).lastOK; !got.Equal(clk.t) {
		t.Errorf("lastOK = %v, want %v", got, clk.t)
	}
}

func TestFirstFetchFailureRetries(t *testing.T) {
	f := &fakeFetcher{err: errors.New("boom")}
	m, rec := recModel(f, 0, nil)
	m = runCmd(m, m.Init())

	if v := plain(m); !strings.Contains(v, "Could not load your PRs") {
		t.Fatalf("no error panel after a failed first fetch:\n%s", v)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("armed timers after the failure = %d, want exactly 1", len(rec.calls))
	}

	// The tick starts one fetch; the panel stays while it runs.
	f.err, f.snap = nil, onePR()
	m, cmd := m.Update(rec.calls[0].msg)
	if cmd == nil {
		t.Fatal("the tick started no fetch")
	}
	if v := plain(m); !strings.Contains(v, "Could not load your PRs") {
		t.Errorf("the panel left the screen while the retry runs:\n%s", v)
	}
	if _, again := m.Update(key("r")); again != nil {
		t.Error("r while the automatic retry runs started a second fetch")
	}
	m = runCmd(m, cmd)
	if f.calls != 2 {
		t.Errorf("fetches = %d, want 2 (Init and the retry)", f.calls)
	}
	if v := plain(m); !strings.Contains(v, "acme/api#1") || strings.Contains(v, "Could not load your PRs") {
		t.Errorf("after the retry succeeded the screen is:\n%s", v)
	}
	if len(rec.calls) != 2 {
		t.Errorf("armed timers = %d, want 2", len(rec.calls))
	}
}

func TestStaleResultDropped(t *testing.T) {
	f := &fakeFetcher{err: errors.New("boom")}
	m, rec := recModel(f, 0, nil)
	m = runCmd(m, m.Init()) // generation 0 fails: error panel
	armed := len(rec.calls)

	m, cmd := m.Update(key("r")) // generation 1
	if cmd == nil {
		t.Fatal("r in the error state returned no command")
	}
	if got := m.(Model).gen; got != 1 {
		t.Fatalf("generation after r = %d, want 1", got)
	}
	screen, errBefore := plain(m), m.(Model).err

	// A result from generation 0 arrives late.
	m, cmd = m.Update(fetchedMsg{gen: 0, snap: onePR()})
	if cmd != nil {
		t.Error("a stale result returned a command")
	}
	if plain(m) != screen {
		t.Errorf("a stale result changed the screen:\n%s", plain(m))
	}
	if m.(Model).err != errBefore || !m.(Model).lastOK.IsZero() {
		t.Error("a stale result changed the error or the last-success time")
	}
	if len(rec.calls) != armed {
		t.Errorf("a stale result armed a timer (%d -> %d)", armed, len(rec.calls))
	}

	// The current generation is accepted.
	m, cmd = m.Update(result(m, onePR(), nil))
	if cmd == nil || len(rec.calls) != armed+1 {
		t.Errorf("a current result armed %d timers, want 1", len(rec.calls)-armed)
	}
	if v := plain(m); !strings.Contains(v, "acme/api#1") {
		t.Errorf("a current result did not show the list:\n%s", v)
	}
}

func TestManualRefreshOrphansTick(t *testing.T) {
	f := &fakeFetcher{snap: onePR()}
	m, rec := recModel(f, 0, nil)
	m = runCmd(m, m.Init())
	oldTick := rec.calls[0].msg
	if f.calls != 1 {
		t.Fatalf("setup: fetches = %d, want 1", f.calls)
	}

	m, cmd := m.Update(key("r"))
	if cmd == nil {
		t.Fatal("r returned no command")
	}
	// The old tick fires while the manual refresh runs, then after it ends.
	if _, c := m.Update(oldTick); c != nil {
		t.Error("the old tick started a fetch while the manual refresh ran")
	}
	m = runCmd(m, cmd)
	if f.calls != 2 {
		t.Fatalf("fetches after r = %d, want 2", f.calls)
	}
	if _, c := m.Update(oldTick); c != nil {
		t.Error("the tick armed before r started a fetch")
	}
	if f.calls != 2 {
		t.Errorf("fetches after the old tick = %d, want 2", f.calls)
	}
	// The tick armed by the manual refresh's result is current and works.
	last := rec.calls[len(rec.calls)-1].msg
	if last != (tickMsg{gen: 1}) {
		t.Fatalf("last armed tick = %#v, want tickMsg{gen: 1}", last)
	}
	if _, c := m.Update(last); c == nil {
		t.Error("the current tick started no fetch")
	}
}

// blockingFetcher holds every Fetch until release is signalled or ctx ends,
// and records how many calls were inside at once.
type blockingFetcher struct {
	release   chan struct{}
	inside    atomic.Int32
	maxInside atomic.Int32
	done      atomic.Int32
}

func (f *blockingFetcher) Fetch(ctx context.Context) (domain.Snapshot, error) {
	n := f.inside.Add(1)
	for {
		old := f.maxInside.Load()
		if n <= old || f.maxInside.CompareAndSwap(old, n) {
			break
		}
	}
	defer func() {
		f.inside.Add(-1)
		f.done.Add(1)
	}()
	select {
	case <-f.release:
		return onePR(), nil
	case <-ctx.Done():
		return domain.Snapshot{}, ctx.Err()
	}
}

func TestNoOverlappingFetches(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f := &blockingFetcher{release: make(chan struct{})}
	p := tea.NewProgram(New(ctx, f, Options{Interval: 5 * time.Millisecond}),
		tea.WithContext(ctx), tea.WithInput(nil), tea.WithOutput(io.Discard))

	runDone := make(chan error, 1)
	go func() {
		_, err := p.Run()
		runDone <- err
	}()

	stop := make(chan struct{})
	var feeders sync.WaitGroup
	feeders.Add(1)
	go func() {
		defer feeders.Done()
		for {
			select {
			case <-stop:
				return
			case f.release <- struct{}{}:
			case <-time.After(time.Millisecond):
			}
		}
	}()

	var senders sync.WaitGroup
	for g := 0; g < 4; g++ {
		senders.Add(1)
		go func() {
			defer senders.Done()
			for i := 0; i < 50; i++ {
				p.Send(tea.KeyPressMsg{Code: 'r', Text: "r"})
				time.Sleep(time.Millisecond)
			}
		}()
	}

	deadline := time.After(5 * time.Second)
	for f.done.Load() < 5 {
		select {
		case <-deadline:
			t.Fatalf("only %d fetches completed in 5 s, want at least 5", f.done.Load())
		case <-time.After(2 * time.Millisecond):
		}
	}
	senders.Wait()
	p.Quit()
	cancel()
	select {
	case <-runDone:
	case <-time.After(5 * time.Second):
		t.Fatal("program did not return within 5 s of Quit")
	}
	close(stop)
	feeders.Wait()
	if got := f.maxInside.Load(); got != 1 {
		t.Errorf("max fetches in flight = %d, want exactly 1", got)
	}
}

// headerOf returns the first line of the plain view: the list header.
func headerOf(m tea.Model) string {
	return strings.SplitN(plain(m), "\n", 2)[0]
}

func TestUpdatedTimeShown(t *testing.T) {
	at := time.Date(2026, 10, 11, 14, 32, 5, 0, time.UTC)
	f := &fakeFetcher{snap: onePR()}
	m, _ := recModel(f, 0, func() time.Time { return at })
	if strings.Contains(plain(m), "updated") {
		t.Errorf("the loading screen shows an updated time:\n%s", plain(m))
	}
	m = runCmd(m, m.Init())
	if h := headerOf(m); !strings.Contains(h, "updated 14:32:05") {
		t.Errorf("header = %q, want it to contain 'updated 14:32:05'", h)
	}
}

func TestUpdatedOnlyOnSuccess(t *testing.T) {
	t1 := time.Date(2026, 10, 11, 14, 32, 5, 0, time.UTC)
	f := &fakeFetcher{snap: onePR()}
	clk := &clock{t: t1}
	m, rec := recModel(f, 0, clk.now)
	m = runCmd(m, m.Init())
	first := plain(m)

	refresh := func() {
		var cmd tea.Cmd
		m, cmd = m.Update(rec.calls[len(rec.calls)-1].msg)
		m = runCmd(m, cmd)
	}

	// A failed refresh a minute later leaves the time alone.
	f.err = errors.New("boom")
	clk.t = t1.Add(time.Minute)
	refresh()
	if h := headerOf(m); !strings.Contains(h, "updated 14:32:05") {
		t.Errorf("after a failure the header = %q, want 'updated 14:32:05'", h)
	}
	if got := plain(m); got != first {
		t.Errorf("a failed refresh changed the screen:\n%s", got)
	}

	// Two successes with identical data differ only in the updated time.
	f.err = nil
	clk.t = t1.Add(2 * time.Minute)
	refresh()
	second := plain(m)
	if !strings.Contains(headerOf(m), "updated 14:34:05") {
		t.Errorf("after a success the header = %q, want 'updated 14:34:05'", headerOf(m))
	}
	blank := func(s string) string {
		return strings.NewReplacer("updated 14:32:05", "updated --", "updated 14:34:05", "updated --").Replace(s)
	}
	if blank(first) != blank(second) {
		t.Errorf("identical data changed the screen beyond the updated time:\n%s\n---\n%s", first, second)
	}
}

func TestSpinnerWhileRefreshing(t *testing.T) {
	at := time.Date(2026, 10, 11, 14, 32, 5, 0, time.UTC)
	f := &fakeFetcher{snap: onePR()}
	m, rec := recModel(f, 0, func() time.Time { return at })
	m = runCmd(m, m.Init())

	frame := m.(Model).spin.View()
	if strings.TrimSpace(frame) == "" {
		t.Fatal("setup: the spinner has no frame")
	}
	if h := headerOf(m); strings.Contains(h, frame) {
		t.Errorf("idle header shows the spinner: %q", h)
	}
	// A stray spinner tick with no fetch running ends the chain.
	if _, cmd := m.Update(m.(Model).spin.Tick()); cmd != nil {
		t.Error("a spinner tick while idle returned a command")
	}

	var cmd tea.Cmd
	m, cmd = m.Update(rec.calls[0].msg)
	if cmd == nil {
		t.Fatal("the tick started no fetch")
	}
	if h := headerOf(m); !strings.Contains(h, "updated 14:32:05 "+frame) {
		t.Errorf("header while refreshing = %q, want the spinner after the updated time", h)
	}
	if _, next := m.Update(m.(Model).spin.Tick()); next == nil {
		t.Error("a spinner tick while refreshing did not continue the chain")
	}

	m = runCmd(m, cmd)
	if h := headerOf(m); !strings.HasSuffix(h, "updated 14:32:05") {
		t.Errorf("header after the result = %q, want it to end with the updated time and no spinner", h)
	}
}

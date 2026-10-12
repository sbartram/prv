package ui

import (
	"context"
	"errors"
	"strings"
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
		if v != before {
			t.Errorf("failure %d: the screen changed (rows are not dimmed or marked in this plan):\n%s", i, v)
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

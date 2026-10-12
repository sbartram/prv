package ui

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
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

package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// limitedErr stands in for *github.Error: it carries Backoff().
type limitedErr struct {
	after   time.Duration
	reset   time.Time
	limited bool
}

func (e *limitedErr) Error() string { return "API rate limit exceeded" }
func (e *limitedErr) Short() string { return "rate limit" }
func (e *limitedErr) Backoff() (time.Duration, time.Time, bool) {
	return e.after, e.reset, e.limited
}

// simStart is the virtual start of every simulation.
var simStart = time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)

// scriptStep is one scripted fetch result.
type scriptStep struct {
	snap domain.Snapshot
	err  error
}

// scriptFetcher records the virtual time of every call and answers from a
// script by call index; calls past the script get def.
type scriptFetcher struct {
	clk   *clock
	steps []scriptStep
	def   scriptStep
	at    []time.Time
}

func (f *scriptFetcher) Fetch(context.Context) (domain.Snapshot, error) {
	i := len(f.at)
	f.at = append(f.at, f.clk.t)
	if i < len(f.steps) {
		return f.steps[i].snap, f.steps[i].err
	}
	return f.def.snap, f.def.err
}

// gaps returns the time between consecutive calls.
func (f *scriptFetcher) gaps() []time.Duration {
	var g []time.Duration
	for i := 1; i < len(f.at); i++ {
		g = append(g, f.at[i].Sub(f.at[i-1]))
	}
	return g
}

func okStep(rl domain.RateLimit) scriptStep {
	s := onePR()
	s.RateLimit = rl
	return scriptStep{snap: s}
}

func healthy() scriptStep {
	return okStep(domain.RateLimit{Limit: 5000, Cost: 1, Remaining: 4000, ResetAt: simStart.Add(80 * time.Minute)})
}

type simItem struct {
	due time.Time
	seq int
	msg tea.Msg
}

// sim drives a Model on a virtual clock. The model's timer is replaced by a
// queue; nothing waits in real time and the virtual clock is the only time
// source.
type sim struct {
	t     *testing.T
	clk   *clock
	f     *scriptFetcher
	m     tea.Model
	q     []simItem
	seq   int
	ticks bool // assert after every step that exactly one current tick is pending
}

func newSim(t *testing.T, steps []scriptStep, def scriptStep) *sim {
	t.Helper()
	clk := &clock{t: simStart}
	f := &scriptFetcher{clk: clk, steps: steps, def: def}
	s := &sim{t: t, clk: clk, f: f}
	m := New(context.Background(), f, Options{ClosedDays: 14, Interval: time.Minute, Now: clk.now})
	m.after = s.after
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	s.m = tm
	return s
}

func (s *sim) after(d time.Duration, msg tea.Msg) tea.Cmd {
	s.q = append(s.q, simItem{due: s.clk.t.Add(d), seq: s.seq, msg: msg})
	s.seq++
	return func() tea.Msg { return nil }
}

// press schedules an r key press at virtual offset at from the start.
func (s *sim) press(at time.Duration) {
	s.q = append(s.q, simItem{due: simStart.Add(at), seq: s.seq, msg: key("r")})
	s.seq++
}

// run starts the model and delivers queued items in time order until the
// queue is past the horizon.
func (s *sim) run(horizon time.Duration) {
	s.t.Helper()
	s.m = runCmd(s.m, s.m.Init())
	s.checkTicks()
	for {
		best := -1
		for i, it := range s.q {
			if best < 0 || it.due.Before(s.q[best].due) || (it.due.Equal(s.q[best].due) && it.seq < s.q[best].seq) {
				best = i
			}
		}
		if best < 0 || s.q[best].due.After(simStart.Add(horizon)) {
			return
		}
		it := s.q[best]
		s.q = append(s.q[:best], s.q[best+1:]...)
		if it.due.After(s.clk.t) {
			s.clk.t = it.due
		}
		s.step(it.msg)
	}
}

func (s *sim) step(msg tea.Msg) {
	s.t.Helper()
	cur := s.m.(Model)
	before, pausedUntil, gen := len(s.f.at), cur.pausedUntil, cur.gen

	var cmd tea.Cmd
	s.m, cmd = s.m.Update(msg)
	s.m = runCmd(s.m, cmd)

	started := len(s.f.at) > before
	if started && s.clk.t.Before(pausedUntil) {
		s.t.Errorf("a call started at %s while paused until %s", s.clk.t.Format("15:04:05"), pausedUntil.Format("15:04:05"))
	}
	if tk, ok := msg.(tickMsg); ok && tk.gen != gen && started {
		s.t.Errorf("a tick of generation %d started a fetch at generation %d", tk.gen, gen)
	}
	if _, ok := msg.(tea.KeyPressMsg); ok && s.clk.t.Before(pausedUntil) {
		if v := plain(s.m); !strings.Contains(v, "rate limited — resumes") {
			s.t.Errorf("r during a pause did not say when refreshing resumes:\n%s", v)
		}
	}
	s.checkTicks()
}

func (s *sim) checkTicks() {
	s.t.Helper()
	if !s.ticks {
		return
	}
	gen, n := s.m.(Model).gen, 0
	for _, it := range s.q {
		if tk, ok := it.msg.(tickMsg); ok && tk.gen == gen {
			n++
		}
	}
	if n != 1 {
		s.t.Fatalf("at %s: %d pending ticks of generation %d, want exactly 1", s.clk.t.Format("15:04:05"), n, gen)
	}
}

func equalGaps(t *testing.T, got []time.Duration, want ...time.Duration) {
	t.Helper()
	if len(got) < len(want) {
		t.Fatalf("gaps = %v, want at least %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("gap %d = %v, want %v (all gaps: %v)", i, got[i], w, got)
		}
	}
}

func TestSimulatedHour(t *testing.T) {
	const hour = time.Hour
	sec := func(n int) time.Duration { return time.Duration(n) * time.Second }

	t.Run("steady", func(t *testing.T) {
		s := newSim(t, nil, healthy())
		s.ticks = true
		s.run(hour)
		if n := len(s.f.at); n < 59 || n > 61 {
			t.Errorf("calls in an hour = %d, want 59 to 61", n)
		}
	})

	t.Run("server-errors", func(t *testing.T) {
		boom := scriptStep{err: &limitedErr{}} // limited false: a plain failure
		s := newSim(t, []scriptStep{healthy(), boom, boom, boom, boom, healthy(), healthy()}, healthy())
		s.ticks = true
		s.run(hour)
		equalGaps(t, s.f.gaps(), sec(60), sec(60), sec(120), sec(240), sec(300), sec(60))
		wantAt := []time.Duration{0, sec(60), sec(120), sec(240), sec(480), sec(780), sec(840)}
		for i, w := range wantAt {
			if got := s.f.at[i].Sub(simStart); got != w {
				t.Errorf("call %d at +%v, want +%v", i, got, w)
			}
		}
	})

	t.Run("exhausted", func(t *testing.T) {
		reset := simStart.Add(20 * time.Minute)
		first := okStep(domain.RateLimit{Limit: 5000, Cost: 1, Remaining: 5, ResetAt: reset})
		s := newSim(t, []scriptStep{first}, healthy())
		s.ticks = true
		s.press(5 * time.Minute)
		s.run(hour)
		if len(s.f.at) < 2 {
			t.Fatalf("calls = %d, want at least 2", len(s.f.at))
		}
		if got, want := s.f.at[1], reset.Add(time.Second); !got.Equal(want) {
			t.Errorf("second call at %s, want exactly %s (reset + 1 s)", got.Format("15:04:05"), want.Format("15:04:05"))
		}
	})

	t.Run("retry-after", func(t *testing.T) {
		limited := scriptStep{err: &limitedErr{after: sec(90), limited: true}}
		s := newSim(t, []scriptStep{healthy(), limited}, healthy())
		s.ticks = true
		s.run(hour)
		equalGaps(t, s.f.gaps(), sec(60), sec(90))
	})

	t.Run("hostile-retry-after", func(t *testing.T) {
		hostile := scriptStep{err: &limitedErr{after: 100 * time.Hour, limited: true}}
		s := newSim(t, []scriptStep{hostile}, healthy())
		s.ticks = true
		s.run(2 * hour)
		equalGaps(t, s.f.gaps(), 65*time.Minute)
	})

	t.Run("presses", func(t *testing.T) {
		s := newSim(t, nil, healthy())
		s.ticks = true
		for at := sec(7); at <= hour; at += sec(7) {
			s.press(at)
		}
		s.run(hour)
		if n := len(s.f.at); n < 500 {
			t.Errorf("calls = %d, want one per press (about 514)", n)
		}
	})
}

// limitedSnap is a successful result that leaves 5 of 5000 points until T+10m.
func limitedSnap(at time.Time) domain.Snapshot {
	s := onePR()
	s.RateLimit = domain.RateLimit{Limit: 5000, Cost: 1, Remaining: 5, ResetAt: at.Add(10 * time.Minute)}
	return s
}

func TestPauseRefusesRefresh(t *testing.T) {
	clk := &clock{t: simStart}
	f := &fakeFetcher{snap: limitedSnap(simStart)}
	m, rec := recModel(f, 0, clk.now)
	m = runCmd(m, m.Init())
	if got, want := rec.calls[0].d, 10*time.Minute+time.Second; got != want {
		t.Fatalf("armed %v after the exhausting fetch, want %v", got, want)
	}

	clk.t = simStart.Add(5 * time.Minute)
	genBefore, armed := m.(Model).gen, len(rec.calls)
	m, cmd := m.Update(key("r"))
	if cmd == nil {
		t.Fatal("r during a pause returned no redraw timer")
	}
	if f.calls != 1 {
		t.Errorf("r during a pause fetched (calls = %d, want 1)", f.calls)
	}
	if m.(Model).gen != genBefore {
		t.Error("r during a pause bumped the generation and orphaned the pending tick")
	}
	if len(rec.calls) != armed+1 || rec.calls[armed].msg != (redrawMsg{}) || rec.calls[armed].d != 4*time.Second {
		t.Fatalf("armed after r = %+v, want one 4s redrawMsg", rec.calls[armed:])
	}
	if v := plain(m); !strings.Contains(v, "rate limited — resumes 12:10") {
		t.Errorf("footer shows no refusal with the resume time:\n%s", v)
	}

	// The pause ends at reset + 1 s; r works again.
	clk.t = simStart.Add(10*time.Minute + 2*time.Second)
	m, cmd = m.Update(key("r"))
	m = runCmd(m, cmd)
	if f.calls != 2 {
		t.Errorf("r after the pause fetched %d times in total, want 2", f.calls)
	}
}

func TestPausedHeader(t *testing.T) {
	clk := &clock{t: simStart}
	m, _ := recModel(&fakeFetcher{snap: limitedSnap(simStart)}, 0, clk.now)
	m = runCmd(m, m.Init())
	if h := headerOf(m); !strings.Contains(h, "paused until 12:10 (rate limit)") {
		t.Errorf("header during a pause = %q", h)
	}
	clk.t = simStart.Add(11 * time.Minute)
	if h := headerOf(m); strings.Contains(h, "paused until") {
		t.Errorf("header after the pause still says paused: %q", h)
	}
}

func TestFlashExpires(t *testing.T) {
	clk := &clock{t: simStart}
	m, _ := recModel(&fakeFetcher{snap: limitedSnap(simStart)}, 0, clk.now)
	m = runCmd(m, m.Init())
	clk.t = simStart.Add(time.Minute)
	m, _ = m.Update(key("r"))

	clk.t = simStart.Add(time.Minute + 3*time.Second)
	if v := plain(m); !strings.Contains(v, "rate limited — resumes") {
		t.Errorf("flash gone after 3 s:\n%s", v)
	}
	clk.t = simStart.Add(time.Minute + 5*time.Second)
	m, cmd := m.Update(redrawMsg{})
	if cmd != nil {
		t.Error("redrawMsg returned a command")
	}
	if v := plain(m); strings.Contains(v, "rate limited — resumes") {
		t.Errorf("flash still shown after 5 s:\n%s", v)
	}
}

func TestErrorPanelPaused(t *testing.T) {
	t.Run("rate limited", func(t *testing.T) {
		clk := &clock{t: simStart}
		f := &fakeFetcher{err: &limitedErr{reset: simStart.Add(10 * time.Minute), limited: true}}
		m, _ := recModel(f, 0, clk.now)
		m = runCmd(m, m.Init())
		v := plain(m)
		if !strings.Contains(v, "Could not load your PRs") || !strings.Contains(v, "paused until 12:10 (rate limit)") {
			t.Errorf("error panel during a rate-limit pause:\n%s", v)
		}
		if strings.Contains(v, "retrying at") {
			t.Errorf("error panel promises a retry during a pause:\n%s", v)
		}
		clk.t = simStart.Add(time.Minute)
		if _, cmd := m.Update(key("r")); cmd != nil {
			t.Error("r in the error panel during a pause returned a command")
		}
		if f.calls != 1 {
			t.Errorf("r in the error panel during a pause fetched (calls = %d)", f.calls)
		}
	})
	t.Run("plain failure", func(t *testing.T) {
		clk := &clock{t: simStart}
		m, _ := recModel(&fakeFetcher{err: &limitedErr{}}, 0, clk.now)
		m = runCmd(m, m.Init())
		v := plain(m)
		if !strings.Contains(v, "retrying at 12:01") || strings.Contains(v, "paused until") {
			t.Errorf("error panel after a plain failure:\n%s", v)
		}
	})
}

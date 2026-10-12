// Package ui is prv's Bubble Tea v2 model and view. Update and View do no I/O;
// all I/O runs inside a tea.Cmd. The package depends on its own Fetcher
// interface and never imports internal/github.
package ui

import (
	"context"
	"errors"
	"time"

	bkey "charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// Fetcher loads a snapshot of the viewer's pull requests.
type Fetcher interface {
	Fetch(ctx context.Context) (domain.Snapshot, error)
}

// defaultInterval is the refresh interval when Options.Interval is zero. The
// 15 s floor is the config package's job, so tests can use short intervals.
const defaultInterval = 60 * time.Second

// Options configures the model.
type Options struct {
	ClosedDays int
	Interval   time.Duration    // zero means defaultInterval
	Now        func() time.Time // nil means time.Now
}

type viewState int

const (
	stateLoading viewState = iota
	stateList
	stateEmpty
	stateError
)

// Model is the Bubble Tea model.
type Model struct {
	ctx     context.Context
	fetcher Fetcher
	opts    Options
	keys    keyMap

	state  viewState
	spin   spinner.Model
	snap   domain.Snapshot
	err    error
	cursor int
	offset int // index of the first visible row
	w, h   int

	// Refresh state. gen is bumped when a manual retry orphans the pending
	// tick. Exactly one tick is pending: it is armed only from a fetch result
	// of the current generation.
	gen      int
	inFlight bool
	interval time.Duration
	now      func() time.Time
	after    func(time.Duration, tea.Msg) tea.Cmd
	lastOK   time.Time // time of the last successful fetch
	nextAt   time.Time // when the pending tick is due
	failedAt time.Time // time of the last failed refresh while data is shown
}

// New builds a Model that starts in the loading state. Init starts the first
// fetch, so the model starts with a fetch in flight.
func New(ctx context.Context, f Fetcher, opts Options) Model {
	if opts.ClosedDays <= 0 {
		opts.ClosedDays = domain.DefaultClosedDays
	}
	interval := opts.Interval
	if interval <= 0 {
		interval = defaultInterval
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return Model{
		ctx:      ctx,
		fetcher:  f,
		opts:     opts,
		keys:     defaultKeys(),
		state:    stateLoading,
		inFlight: true,
		spin:     spinner.New(spinner.WithSpinner(spinner.Dot)),
		interval: interval,
		now:      now,
		after:    afterTick,
	}
}

// Init starts the spinner and the first fetch. It arms no tick: the first
// result arms it.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, fetch(m.ctx, m.fetcher, m.gen))
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case bkey.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case bkey.Matches(msg, m.keys.Refresh):
			if m.inFlight {
				return m, nil // merges with the running refresh
			}
			m.gen++ // orphans the pending tick
			if m.state == stateError {
				m.state = stateLoading
			}
			return m, m.beginFetch()
		case bkey.Matches(msg, m.keys.Down):
			if m.state == stateList && m.cursor < len(m.snap.PRs)-1 {
				m.cursor++
				m.clampOffset()
			}
		case bkey.Matches(msg, m.keys.Up):
			if m.state == stateList && m.cursor > 0 {
				m.cursor--
				m.clampOffset()
			}
		}
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.clampOffset()
	case spinner.TickMsg:
		if m.inFlight || m.state == stateLoading {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
	case tickMsg:
		if msg.gen != m.gen || m.inFlight {
			return m, nil
		}
		return m, m.beginFetch()
	case fetchedMsg:
		if errors.Is(msg.err, context.Canceled) {
			return m, nil // the program is quitting
		}
		if msg.gen != m.gen {
			return m, nil
		}
		m.inFlight = false
		if msg.err != nil {
			m.err = msg.err
			if !m.lastOK.IsZero() {
				// Data has been shown: keep the rows and the state, undimmed.
				m.failedAt = m.now()
				return m, m.arm(m.interval)
			}
			// No data yet: the error panel, and the tick retries on its own.
			m.cursor, m.offset = 0, 0
			m.state = stateError
			return m, m.arm(m.interval)
		}
		m.cursor, m.offset = 0, 0
		m.err = nil
		m.failedAt = time.Time{}
		m.snap = msg.snap
		m.lastOK = m.now()
		if len(msg.snap.PRs) == 0 {
			m.state = stateEmpty
		} else {
			m.state = stateList
		}
		return m, m.arm(m.interval)
	}
	return m, nil
}

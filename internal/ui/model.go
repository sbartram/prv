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

	state viewState
	spin  spinner.Model
	snap  domain.Snapshot
	err   error
	w, h  int

	// Selection is tracked by PR ref (D-03). selected is the source of truth;
	// rebuild derives cursor from it after every snapshot. rows is what is on
	// screen; plan 02-08 makes it a filtered view of snap.PRs.
	selected domain.Ref
	rows     []domain.PR
	cursor   int // index of selected in rows; 0 when there are no rows
	offset   int // index of the first visible row

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
			if m.state == stateList && m.cursor < len(m.rows)-1 {
				m.cursor++
				m.selected = m.rows[m.cursor].Ref
				m.clampOffset()
			}
		case bkey.Matches(msg, m.keys.Up):
			if m.state == stateList && m.cursor > 0 {
				m.cursor--
				m.selected = m.rows[m.cursor].Ref
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
		m.err = nil
		m.failedAt = time.Time{}
		m.snap = msg.snap
		m.lastOK = m.now()
		m.rebuild()
		if len(msg.snap.PRs) == 0 {
			m.state = stateEmpty
		} else {
			m.state = stateList
		}
		return m, m.arm(m.interval)
	}
	return m, nil
}

// rebuild derives the rows on screen and the cursor from the snapshot and the
// selected ref. A refresh that moves the selected PR moves the cursor with it;
// when the PR left the list the cursor lands on its nearest surviving
// neighbour in the old order (D-03).
func (m *Model) rebuild() {
	oldRows, oldCur := m.rows, m.cursor
	m.rows = m.snap.PRs
	switch {
	case len(m.rows) == 0:
		// Keep m.selected: the PR may come back.
		m.cursor, m.offset = 0, 0
		return
	case m.selected == (domain.Ref{}):
		m.cursor = 0
		m.selected = m.rows[0].Ref
	default:
		if i := indexOf(m.rows, m.selected); i >= 0 {
			m.cursor = i
		} else {
			m.cursor = nearest(oldRows, oldCur, m.rows)
			m.selected = m.rows[m.cursor].Ref
		}
	}
	m.clampOffset()
}

// nearest returns the index in rows of the PR closest to oldRows[oldCur] that
// is still present. At each distance it looks below first, then above. When no
// old neighbour survived it keeps the old row index, clamped to the new list.
// rows must not be empty.
func nearest(oldRows []domain.PR, oldCur int, rows []domain.PR) int {
	for d := 1; oldCur+d < len(oldRows) || oldCur-d >= 0; d++ {
		for _, i := range []int{oldCur + d, oldCur - d} {
			if i >= 0 && i < len(oldRows) {
				if j := indexOf(rows, oldRows[i].Ref); j >= 0 {
					return j
				}
			}
		}
	}
	return min(oldCur, len(rows)-1)
}

// indexOf returns the index of ref in rows, or -1.
func indexOf(rows []domain.PR, ref domain.Ref) int {
	for i, pr := range rows {
		if pr.Ref == ref {
			return i
		}
	}
	return -1
}

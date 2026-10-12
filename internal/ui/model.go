// Package ui is prv's Bubble Tea v2 model and view. Update and View do no I/O;
// all I/O runs inside a tea.Cmd. The package depends on its own Fetcher
// interface and never imports internal/github.
package ui

import (
	"context"
	"errors"
	"time"

	"charm.land/bubbles/v2/help"
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

// maxFilterDigits is how many digits the number prompt keeps. A PR number has
// at most 7 digits in practice; a longer prefix matches nothing anyway.
const maxFilterDigits = 7

// warnFor is how long the config warnings stay on screen, counted from the
// first fetch result: the loading screen has no footer to show them in.
const warnFor = 8 * time.Second

// Options configures the model.
type Options struct {
	ClosedDays int
	Show       domain.StateSet  // states visible at launch; zero means domain.AllStates (D-12)
	ConfigPath string           // display form of the config path, for the help overlay
	Warnings   []string         // config warnings, shown for warnFor from the first fetch result (D-15)
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
	help    help.Model // footer key hints, rendered from keys (D-17)

	state viewState
	spin  spinner.Model
	snap  domain.Snapshot
	err   error
	w, h  int

	// show is which lifecycle states are visible (D-10). It lasts for the
	// session only: nothing is written to disk (D-12).
	show domain.StateSet

	// numFilter is the digit prefix typed after "/" (D-11). The prompt is a
	// hand-rolled buffer: it takes digits only, so nothing else from a key
	// press or a paste is stored or drawn. prompting is true while the prompt
	// line is open.
	numFilter string
	prompting bool
	pkeys     promptKeys

	// showHelp is true while the full-screen help overlay is open (D-17).
	// Refreshes carry on behind it.
	showHelp bool

	// Selection is tracked by PR ref (D-03). selected is the source of truth;
	// rebuild derives cursor from it after every snapshot. rows is what is on
	// screen: snap.PRs after the filters.
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

	// Backoff state. failures counts consecutive failed fetches. While now is
	// before pausedUntil no request goes out, not even on r. flash is a short
	// message in the footer that expires at flashUntil.
	failures    int
	pausedUntil time.Time
	flash       string
	flashUntil  time.Time

	// Config warnings (D-15) have their own fields and footer segment, so a
	// flash never overwrites them and they never hide a flash. warnUntil is
	// zero until the first fetch result starts the clock.
	configPath string
	warnings   []string
	warnUntil  time.Time
}

// backoffer is implemented by errors that know when to retry. ui declares it
// so that it never imports internal/github. It returns the Retry-After delay,
// the time the limit resets (zero when unknown), and whether the error is a
// rate limit.
type backoffer interface {
	Backoff() (time.Duration, time.Time, bool)
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
	h := help.New()
	h.ShortSeparator = " · "
	h.Styles = helpStyles()
	show := opts.Show
	if show == 0 {
		show = domain.AllStates
	}
	return Model{
		show:       show,
		warnings:   opts.Warnings,
		ctx:        ctx,
		fetcher:    f,
		opts:       opts,
		keys:       defaultKeys(),
		pkeys:      defaultPromptKeys(),
		help:       h,
		state:      stateLoading,
		inFlight:   true,
		spin:       spinner.New(spinner.WithSpinner(spinner.Dot)),
		interval:   interval,
		configPath: opts.ConfigPath,
		now:        now,
		after:      afterTick,
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
	case tea.PasteMsg:
		if m.prompting {
			for _, r := range msg.Content {
				if r >= '0' && r <= '9' && len(m.numFilter) < maxFilterDigits {
					m.numFilter += string(r)
				}
			}
			m.rebuild()
		}
	case tea.KeyPressMsg:
		if m.showHelp {
			// D-17: any key returns to the list and is consumed; q closes the
			// overlay rather than quitting. Only ctrl+c still quits.
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			m.showHelp = false
			return m, nil
		}
		if m.prompting {
			m.promptKey(msg)
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}
		switch {
		case bkey.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case bkey.Matches(msg, m.keys.Help):
			m.showHelp = true
		case bkey.Matches(msg, m.keys.Refresh):
			if m.inFlight {
				return m, nil // merges with the running refresh
			}
			if m.now().Before(m.pausedUntil) {
				// D-08: no request during a pause; say when refreshing resumes.
				return m, m.setFlash("rate limited — resumes "+m.clock(m.pausedUntil, "15:04"), flashFor)
			}
			m.gen++ // orphans the pending tick
			if m.state == stateError {
				m.state = stateLoading
			}
			return m, m.beginFetch()
		case bkey.Matches(msg, m.keys.Filter):
			if m.state == stateList { // the prompt line only exists on the list screens
				m.prompting = true
			}
		case m.numFilter != "" && bkey.Matches(msg, m.keys.Clear):
			m.numFilter = ""
			m.rebuild()
		case bkey.Matches(msg, m.keys.ToggleOpen):
			m.show = m.show.Toggle(domain.StateOpen)
			m.rebuild()
		case bkey.Matches(msg, m.keys.ToggleMerged):
			m.show = m.show.Toggle(domain.StateMerged)
			m.rebuild()
		case bkey.Matches(msg, m.keys.ToggleClosed):
			m.show = m.show.Toggle(domain.StateClosed)
			m.rebuild()
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
		if now := m.now(); now.Before(m.pausedUntil) {
			// The timer fired early: no request during a pause.
			return m, m.arm(m.pausedUntil.Sub(now))
		}
		return m, m.beginFetch()
	case redrawMsg:
		// Nothing to change: Bubble Tea redraws after every Update, which
		// lets an expired flash leave the footer.
		return m, nil
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
				return m, tea.Batch(m.schedule(msg.err), m.startWarning())
			}
			// No data yet: the error panel, and the tick retries on its own.
			m.cursor, m.offset = 0, 0
			m.state = stateError
			return m, tea.Batch(m.schedule(msg.err), m.startWarning())
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
		return m, tea.Batch(m.schedule(nil), m.startWarning())
	}
	return m, nil
}

// startWarning starts the config-warning clock on the first fetch result and
// returns the Cmd that redraws the screen when the warning expires. Later
// results return nil, so the warnings show once (D-15).
func (m *Model) startWarning() tea.Cmd {
	if len(m.warnings) == 0 || !m.warnUntil.IsZero() {
		return nil
	}
	m.warnUntil = m.now().Add(warnFor)
	return m.after(warnFor, redrawMsg{})
}

// promptKey handles one key press while the number prompt is open (D-11).
// Digits are typed (1, 2 and 3 do not toggle), enter keeps the filter, esc
// clears it, backspace deletes. Every other key is ignored; q does not quit.
// The caller handles ctrl+c.
func (m *Model) promptKey(msg tea.KeyPressMsg) {
	switch {
	case bkey.Matches(msg, m.pkeys.Accept):
		m.prompting = false
	case bkey.Matches(msg, m.pkeys.Cancel):
		m.numFilter, m.prompting = "", false
		m.rebuild()
	case bkey.Matches(msg, m.pkeys.Delete):
		if m.numFilter != "" {
			m.numFilter = m.numFilter[:len(m.numFilter)-1]
			m.rebuild()
		}
	default:
		if r := []rune(msg.Text); len(r) == 1 && r[0] >= '0' && r[0] <= '9' && len(m.numFilter) < maxFilterDigits {
			m.numFilter += string(r)
			m.rebuild()
		}
	}
}

// schedule applies the backoff policy to the result of the current-generation
// fetch (err is nil on success) and arms the one tick for the next refresh. It
// also sets or clears the rate-limit pause.
func (m *Model) schedule(err error) tea.Cmd {
	out := domain.Outcome{Failed: err != nil}
	if err == nil {
		m.failures = 0
	} else {
		m.failures++
		var b backoffer
		if errors.As(err, &b) {
			out.RetryAfter, out.ResetAt, out.RateLimited = b.Backoff()
		}
	}
	d := domain.NextDelay(m.now(), m.interval, m.failures, m.snap.RateLimit, out)
	m.pausedUntil = d.PausedUntil
	return m.arm(d.Wait)
}

// rebuild derives the rows on screen and the cursor from the snapshot, the
// filters and the selected ref. A refresh that moves the selected PR moves the cursor with it;
// when the PR left the list the cursor lands on its nearest surviving
// neighbour in the old order (D-03).
func (m *Model) rebuild() {
	oldRows, oldCur := m.rows, m.cursor
	m.rows = domain.Visible(m.snap.PRs, m.show, m.numFilter)
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

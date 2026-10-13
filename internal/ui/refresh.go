package ui

import (
	"context"
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// tickMsg is the auto-refresh timer. It carries the generation that armed it;
// a tick from another generation is dropped.
type tickMsg struct{ gen int }

// redrawMsg makes Bubble Tea redraw when a flash expires. Update ignores it.
type redrawMsg struct{}

// flashFor is how long a transient footer message stays.
const flashFor = 4 * time.Second

// fetchedMsg is the result of one fetch, stamped with the generation that
// started it.
type fetchedMsg struct {
	gen  int
	snap domain.Snapshot
	err  error
	// epoch is the fetch's epoch; a result whose epoch is below Model.mutEpoch
	// started before a local watch-list change and is discarded (RESEARCH
	// Pitfall 1).
	epoch int
}

// fetch runs one fetch. The closure captures only ctx, f and gen, never the
// Model. A panic inside the fetcher becomes an error with a fixed message: the
// panic value is never shown, and the terminal is not left in the alt screen.
func fetch(ctx context.Context, f Fetcher, gen int) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if recover() != nil {
				msg = fetchedMsg{gen: gen, err: errors.New("unexpected internal error while loading PRs")}
			}
		}()
		snap, err := f.Fetch(ctx)
		return fetchedMsg{gen: gen, snap: snap, err: err}
	}
}

// detailMsg is the result of one detail fetch, stamped with the sequence number
// and the PR it was started for.
type detailMsg struct {
	seq  int
	ref  domain.Ref
	data domain.Detail
	err  error
}

// fetchDetail runs one detail fetch. The closure captures only its arguments,
// never the Model. A panic inside the fetcher becomes an error with a fixed
// message: the panic value is never shown.
func fetchDetail(ctx context.Context, f DetailFetcher, id string, ref domain.Ref, seq int) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if recover() != nil {
				msg = detailMsg{seq: seq, ref: ref, err: errors.New("unexpected internal error while loading details")}
			}
		}()
		data, err := f.FetchDetail(ctx, id)
		return detailMsg{seq: seq, ref: ref, data: data, err: err}
	}
}

// afterTick returns a Cmd that delivers msg after d. It uses tea.Tick, whose
// timer starts when the Cmd is created, not a wall-clock-aligned tick.
func afterTick(d time.Duration, msg tea.Msg) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return msg })
}

// beginFetch marks a fetch as running and returns the Cmds that start it.
func (m *Model) beginFetch() tea.Cmd {
	m.inFlight = true
	return tea.Batch(m.spin.Tick, fetch(m.ctx, m.fetcher, m.gen))
}

// arm schedules the next refresh tick for the current generation.
func (m *Model) arm(d time.Duration) tea.Cmd {
	m.nextAt = m.now().Add(d)
	return m.after(d, tickMsg{gen: m.gen})
}

// setFlash shows text in the footer for d and returns the Cmd that redraws
// the screen when it expires.
func (m *Model) setFlash(text string, d time.Duration) tea.Cmd {
	m.flash = text
	m.flashUntil = m.now().Add(d)
	return m.after(d, redrawMsg{})
}

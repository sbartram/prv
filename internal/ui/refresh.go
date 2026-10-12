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

// fetchedMsg is the result of one fetch, stamped with the generation that
// started it.
type fetchedMsg struct {
	gen  int
	snap domain.Snapshot
	err  error
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

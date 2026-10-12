// Package ui is prv's Bubble Tea v2 model and view. Update and View do no I/O;
// all I/O runs inside a tea.Cmd. The package depends on its own Fetcher
// interface and never imports internal/github.
package ui

import (
	"context"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// Fetcher loads a snapshot of the viewer's pull requests.
type Fetcher interface {
	Fetch(ctx context.Context) (domain.Snapshot, error)
}

// Options configures the model.
type Options struct {
	ClosedDays int
}

type viewState int

const (
	stateLoading viewState = iota
	stateList
	stateError
)

// Model is the Bubble Tea model.
type Model struct {
	ctx     context.Context
	fetcher Fetcher
	opts    Options

	state   viewState
	loading bool
	spin    spinner.Model
	snap    domain.Snapshot
	err     error
	cursor  int
	w, h    int
}

// New builds a Model that starts in the loading state.
func New(ctx context.Context, f Fetcher, opts Options) Model {
	if opts.ClosedDays <= 0 {
		opts.ClosedDays = domain.DefaultClosedDays
	}
	return Model{
		ctx:     ctx,
		fetcher: f,
		opts:    opts,
		state:   stateLoading,
		loading: true,
		spin:    spinner.New(spinner.WithSpinner(spinner.Dot)),
	}
}

type fetchedMsg struct {
	snap domain.Snapshot
	err  error
}

func (m Model) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		snap, err := m.fetcher.Fetch(m.ctx)
		return fetchedMsg{snap: snap, err: err}
	}
}

// Init starts the spinner and the first fetch.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, m.fetchCmd())
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.snap.PRs)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		}
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case spinner.TickMsg:
		if m.state == stateLoading {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
	case fetchedMsg:
		m.loading = false
		m.cursor = 0
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.state = stateList
		m.err = nil
		m.snap = msg.snap
	}
	return m, nil
}

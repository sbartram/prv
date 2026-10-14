package ui

import (
	"context"
	"errors"

	bkey "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// confirmState is the remove confirm line (D-06). open is true from d until the
// next key press; pr is the row being removed, copied when d was pressed.
type confirmState struct {
	open bool
	pr   domain.PR
}

// removedMsg is the result of one Watchlist.Remove. ref is the saved ref that
// was removed. removed is false when the entry was already gone from the file.
type removedMsg struct {
	ref     domain.Ref
	removed bool
	err     error
}

// removeCmd removes one saved ref. The closure captures only its arguments,
// never the Model. A panic inside the watch list becomes an error with a fixed
// message.
func removeCmd(w Watchlist, ref domain.Ref) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if recover() != nil {
				msg = removedMsg{ref: ref, err: errors.New("unexpected internal error while updating the watch list")}
			}
		}()
		r, err := w.RemoveRef(ref)
		return removedMsg{ref: ref, removed: r.Changed, err: err}
	}
}

// askRemove is d on the selected row: a saved row opens the confirm line; a row
// that only discovery found says why it cannot be removed (D-08).
func (m *Model) askRemove() tea.Cmd {
	pr := m.rows[m.cursor]
	switch {
	case !pr.Manual:
		return m.setFlash(pr.Ref.String()+" is authored/assigned — only + rows can be removed", flashFor)
	case m.watchlist == nil:
		return m.setFlash("removing PRs is not available", flashFor)
	}
	m.confirm = confirmState{open: true, pr: pr}
	return nil
}

// confirmKey handles one key press while the confirm line is open. Only y
// removes and only ctrl+c quits; every other key, q included, cancels (the
// locked decision of 2026-10-12: q never quits from inside the confirm).
func (m *Model) confirmKey(msg tea.KeyPressMsg) tea.Cmd {
	pr := m.confirm.pr
	m.confirm = confirmState{}
	switch {
	case msg.String() == "ctrl+c":
		return tea.Quit
	case bkey.Matches(msg, m.pkeys.Yes):
		return removeCmd(m.watchlist, pr.Watched)
	}
	return nil
}

// applyRemoved handles the result of a removal: on success the saved row leaves
// the list; on a failed write the row stays. An entry that was already gone from
// the file counts as removed.
func (m *Model) applyRemoved(msg removedMsg) tea.Cmd {
	if errors.Is(msg.err, context.Canceled) {
		return nil
	}
	if msg.err != nil {
		return m.setFlash("could not update the watch list", flashFor)
	}
	m.markMutation()
	k := msg.ref.Key()
	label := "removed " + msg.ref.String()
	prs := make([]domain.PR, 0, len(m.snap.PRs))
	for _, pr := range m.snap.PRs {
		switch {
		case !pr.Manual || pr.Watched.Key() != k:
		case pr.Auto:
			// D-07: discovery lists it too, so it stays, no longer saved.
			pr.Manual, pr.Watched = false, domain.Ref{}
			label = "unpinned — still listed (authored/assigned)"
		default:
			label = "removed " + pr.Ref.String()
			continue
		}
		prs = append(prs, pr)
	}
	m.snap.PRs = prs
	m.rebuild()
	if len(prs) == 0 {
		m.state = stateEmpty
	}
	return m.setFlash(label, flashFor)
}

// confirmLine is the confirm line. It replaces the footer, so it costs no list
// row.
func (m Model) confirmLine() string {
	return m.cut("remove " + m.confirm.pr.Ref.String() + " from watch list? y/n")
}

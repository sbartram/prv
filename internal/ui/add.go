package ui

import (
	"context"
	"errors"
	"slices"
	"strings"

	bkey "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// Looker loads one PR by ref for the add prompt (D-02). It is optional: the
// model type-asserts its Fetcher, and a Fetcher without Lookup makes adding
// unavailable. *github.Client implements it.
type Looker interface {
	Lookup(ctx context.Context, ref domain.Ref) (domain.PR, error)
}

// Watchlist saves and removes manually added PRs. It reads and writes a file, so
// the model only calls it inside a tea.Cmd. *watchlist.Store implements it.
type Watchlist interface {
	Add(ref domain.Ref) (bool, error)
	Remove(ref domain.Ref) (bool, error)
}

// maxAddRunes caps the add prompt's text. A longer paste is cut.
const maxAddRunes = 256

// addState is the add prompt (WATCH-04). open is true while the prompt line is
// on screen; checking is true from enter until the PR is checked and saved.
type addState struct {
	open, checking bool
	text           string
	ref            domain.Ref // the ref being checked
	errText        string     // why the last attempt failed; the typed text stays
}

// addCheckedMsg is the result of one Lookup, stamped with the sequence number of
// the attempt that started it.
type addCheckedMsg struct {
	seq int
	ref domain.Ref
	pr  domain.PR
	err error
}

// addSavedMsg is the result of one save, stamped like addCheckedMsg.
type addSavedMsg struct {
	seq   int
	pr    domain.PR
	added bool
	err   error
}

// lookupCmd checks one PR. The closure captures only its arguments, never the
// Model. A panic inside the looker becomes an error with a fixed message.
func lookupCmd(ctx context.Context, l Looker, ref domain.Ref, seq int) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if recover() != nil {
				msg = addCheckedMsg{seq: seq, ref: ref, err: errors.New("unexpected internal error while checking the PR")}
			}
		}()
		pr, err := l.Lookup(ctx, ref)
		return addCheckedMsg{seq: seq, ref: ref, pr: pr, err: err}
	}
}

// saveCmd saves the PR's watched ref. Same shape as lookupCmd.
func saveCmd(w Watchlist, pr domain.PR, seq int) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if recover() != nil {
				msg = addSavedMsg{seq: seq, pr: pr, err: errors.New("unexpected internal error while saving the watch list")}
			}
		}()
		added, err := w.Add(pr.Watched)
		return addSavedMsg{seq: seq, pr: pr, added: added, err: err}
	}
}

// looker returns the Fetcher as a Looker, or nil when it has no Lookup.
func (m Model) looker() Looker {
	l, _ := m.fetcher.(Looker)
	return l
}

// openAdd opens the add prompt, or says that adding is not available.
func (m *Model) openAdd() tea.Cmd {
	if m.looker() == nil || m.watchlist == nil {
		return m.setFlash("adding PRs is not available", flashFor)
	}
	m.add = addState{open: true}
	return nil
}

// addKey handles one key press while the add prompt is open (D-01). Every key
// but ctrl+c, enter, esc and backspace is typed text: q, a, d and digits do not
// act on the list.
func (m *Model) addKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case msg.String() == "ctrl+c":
		return tea.Quit
	case bkey.Matches(msg, m.pkeys.Cancel):
		m.add = addState{}
		m.addSeq++ // a result that is still on its way is dropped
		return nil
	case m.add.checking:
		return nil
	case bkey.Matches(msg, m.pkeys.Accept):
		return m.submitAdd()
	case bkey.Matches(msg, m.pkeys.Delete):
		if r := []rune(m.add.text); len(r) > 0 {
			m.add.text = string(r[:len(r)-1])
		}
		m.add.errText = ""
	default:
		m.appendAdd(msg.Text)
	}
	return nil
}

// appendAdd adds typed or pasted text to the prompt, cleaned, up to
// maxAddRunes runes.
func (m *Model) appendAdd(s string) {
	s = domain.Clean(s)
	if s == "" {
		return
	}
	r := []rune(m.add.text)
	room := maxAddRunes - len(r)
	if room <= 0 {
		return
	}
	add := []rune(s)
	if len(add) > room {
		add = add[:room]
	}
	m.add.text += string(add)
	m.add.errText = ""
}

// addPaste handles a bracketed paste into the add prompt. Newlines become
// spaces first, so a pasted link with a trailing newline is not run as enter.
func (m *Model) addPaste(s string) {
	if m.add.checking {
		return
	}
	s = strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
	m.appendAdd(s)
}

// submitAdd is enter: parse the text, then check the PR before anything is saved
// (D-02). A text that is not a PR reference shows its reason and sends nothing.
func (m *Model) submitAdd() tea.Cmd {
	ref, err := domain.ParseRef(m.add.text)
	if err != nil {
		var re *domain.RefError
		if errors.As(err, &re) {
			m.add.errText = re.Reason
		} else {
			m.add.errText = domain.ReasonNotRef
		}
		return nil
	}
	m.addSeq++
	m.add.checking = true
	m.add.errText = ""
	m.add.ref = ref
	return tea.Batch(m.spin.Tick, lookupCmd(m.ctx, m.looker(), ref, m.addSeq))
}

// applyAddChecked handles the Lookup result: an error or an unavailable PR shows
// its reason and keeps the text; a PR that loaded goes on to be saved.
func (m *Model) applyAddChecked(msg addCheckedMsg) tea.Cmd {
	if errors.Is(msg.err, context.Canceled) || !m.add.open || msg.seq != m.addSeq {
		return nil
	}
	m.add.checking = false
	switch {
	case msg.err != nil:
		m.add.errText = "could not check (" + errReason(msg.err) + ")"
		return nil
	case msg.pr.Unavail.Kind != domain.UnavailNone:
		m.add.errText = msg.pr.Unavail.Reason
		return nil
	}
	// Save the canonical ref GitHub returned, so the file matches discovery.
	saved := msg.ref
	if domain.ValidRef(msg.pr.Ref) {
		saved = msg.pr.Ref
	}
	pr := msg.pr
	pr.Manual = true
	pr.Watched = saved
	m.add.checking = true
	return saveCmd(m.watchlist, pr, m.addSeq)
}

// applyAddSaved handles the save result: on success the prompt closes and the
// PR joins the rows with the cursor on it.
func (m *Model) applyAddSaved(msg addSavedMsg) tea.Cmd {
	if !m.add.open || msg.seq != m.addSeq {
		return nil
	}
	if msg.err != nil {
		m.add.checking = false
		m.add.errText = "could not save the watch list"
		return nil
	}
	m.add = addState{}
	prs := slices.Clone(m.snap.PRs)
	dup := slices.ContainsFunc(prs, func(p domain.PR) bool {
		return (msg.pr.ID != "" && p.ID == msg.pr.ID) || p.Ref.Key() == msg.pr.Ref.Key()
	})
	if !dup {
		prs = append(prs, msg.pr)
	}
	domain.Sort(prs)
	m.snap.PRs = prs
	m.selected = msg.pr.Ref
	m.state = stateList
	m.rebuild()
	return nil
}

// addLine is the prompt line. It replaces the footer, so it costs no list row.
func (m Model) addLine() string {
	line := "add PR: " + m.add.text + "█"
	switch {
	case m.add.errText != "":
		line += "   " + warnStyle.Render("✗ "+m.add.errText) + " · esc cancel"
	case m.add.checking:
		line += "   checking " + m.add.ref.String() + " " + m.spin.View()
	default:
		line += faintStyle.Render("   (URL or owner/repo#n · enter add · esc cancel)")
	}
	return m.cut(line)
}

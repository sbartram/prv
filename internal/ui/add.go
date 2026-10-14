package ui

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	bkey "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sbartram/prv/internal/domain"
	"github.com/sbartram/prv/internal/watchlist"
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
	AddRef(ref domain.Ref) (watchlist.Result, error)
	RemoveRef(ref domain.Ref) (watchlist.Result, error)
}

// maxAddRunes caps the add prompt's text. A longer paste is cut.
const maxAddRunes = 256

// addState is the add prompt (WATCH-04). open is true while the prompt line is
// on screen; checking is true from enter until the PR is looked up. saving is
// true from the moment the save Cmd is dispatched until its result arrives; it
// is separate from checking because esc can drop a lookup but not a save that
// the file will honour (D-05).
type addState struct {
	open, checking bool
	saving         bool
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
	note  string // the backup's base name when the save replaced a broken file
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
		r, err := w.AddRef(pr.Watched)
		return addSavedMsg{seq: seq, pr: pr, added: r.Changed, note: r.StartedOver, err: err}
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
	case m.add.saving:
		// The save is on its way and the file will change: esc, typing and
		// paste do nothing, so the screen and the file agree (D-05).
		return nil
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
	if m.add.checking || m.add.saving {
		return
	}
	s = strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
	m.appendAdd(s)
}

// submitAdd is enter: parse the text, then check the PR before anything is saved
// (D-02). A text that is not a PR reference shows its reason and sends nothing.
func (m *Model) submitAdd() tea.Cmd {
	if now := m.now(); now.Before(m.pausedUntil) {
		// Phase 2 D-08: no request during a rate-limit pause.
		m.add.errText = "rate limited — resumes " + m.clock(m.pausedUntil, "15:04")
		return nil
	}
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
	if hit, ok := m.findListed(ref, ""); ok {
		return m.closeAsDuplicate(hit)
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
		m.pauseForRateLimit(msg.err) // the same pause a detail fetch sets
		return nil
	case msg.pr.Unavail.Kind != domain.UnavailNone:
		m.add.errText = msg.pr.Unavail.Reason
		if msg.pr.Unavail.Kind == domain.UnavailSSO && msg.pr.Unavail.Help != "" {
			// The prompt is the only place the user sees this ref before it is
			// saved, so it carries the SSO help.
			m.add.errText += " · " + msg.pr.Unavail.Help
		}
		return nil
	}
	// D-03: GitHub may return a PR that is already a row under another name (a
	// renamed repository), found by its node ID.
	if hit, ok := m.findListed(msg.pr.Ref, msg.pr.ID); ok {
		return m.closeAsDuplicate(hit)
	}
	// Save the canonical ref GitHub returned, so the file matches discovery.
	saved := msg.ref
	if domain.ValidRef(msg.pr.Ref) {
		saved = msg.pr.Ref
	}
	pr := msg.pr
	pr.Manual = true
	pr.Watched = saved
	m.add.saving = true
	return saveCmd(m.watchlist, pr, m.addSeq)
}

// applyAddSaved handles the save result: on success the prompt closes and the
// PR joins the rows with the cursor on it.
func (m *Model) applyAddSaved(msg addSavedMsg) tea.Cmd {
	if !m.add.open || msg.seq != m.addSeq {
		return nil
	}
	if msg.err != nil {
		m.add.checking, m.add.saving = false, false
		m.add.errText = "could not save the watch list"
		return nil
	}
	m.add = addState{}
	prs := slices.Clone(m.snap.PRs)
	target := msg.pr
	if i := m.listedIndex(msg.pr.Ref, msg.pr.ID); i >= 0 {
		// A refresh listed it first: it is a saved PR from now on.
		prs[i].Manual = true
		prs[i].Watched = msg.pr.Watched
		target = prs[i]
	} else {
		prs = append(prs, msg.pr)
	}
	domain.Sort(prs)
	m.snap.PRs = prs
	m.markMutation()
	m.state = stateList
	m.rebuild()
	notice, d := startedOverNotice(msg.note)
	switch {
	case !msg.added:
		// Another pane saved it first; it is listed all the same.
		return m.jumpTo(target, joinNotice(notice, "already watching "+target.Ref.String()), d)
	case indexOf(m.rows, target.Ref) >= 0:
		m.selected = target.Ref
		m.rebuild()
		if notice != "" {
			return m.setFlash(notice, d)
		}
		return nil
	}
	return m.setFlash(joinNotice(notice, "added "+target.Ref.String()+" · hidden by filters"), d)
}

// startedOverNotice is the footer text for a save that replaced a broken
// watchlist.json (D-04), and how long it stays. note is the backup's base name;
// with none, the text is empty and the duration is the usual flashFor.
func startedOverNotice(note string) (string, time.Duration) {
	if note == "" {
		return "", flashFor
	}
	return "previous list saved to " + note, warnFor
}

// joinNotice puts the notice ahead of text. A narrow pane cuts the end of the
// footer, so the notice, which matters more, comes first.
func joinNotice(notice, text string) string {
	if notice == "" {
		return text
	}
	return notice + " · " + text
}

// listedIndex returns the index in m.snap.PRs of the first PR that is ref or id:
// its non-empty ID equals id, its Ref has the same Key, or it is a saved PR whose
// saved ref has the same Key. It returns -1 when none is.
func (m *Model) listedIndex(ref domain.Ref, id string) int {
	k := ref.Key()
	return slices.IndexFunc(m.snap.PRs, func(p domain.PR) bool {
		return (id != "" && p.ID == id) || p.Ref.Key() == k || (p.Manual && p.Watched.Key() == k)
	})
}

// findListed returns the listed PR that is ref or id (see listedIndex).
func (m *Model) findListed(ref domain.Ref, id string) (domain.PR, bool) {
	if i := m.listedIndex(ref, id); i >= 0 {
		return m.snap.PRs[i], true
	}
	return domain.PR{}, false
}

// closeAsDuplicate closes the prompt for a PR that is already listed and moves
// the cursor to it (D-03). Nothing is looked up or saved.
func (m *Model) closeAsDuplicate(hit domain.PR) tea.Cmd {
	m.add = addState{}
	m.addSeq++
	return m.jumpTo(hit, "already watching "+hit.Ref.String(), flashFor)
}

// jumpTo selects pr and shows text in the footer for d. When the filters hide the
// row the selection stays and the text says so.
func (m *Model) jumpTo(pr domain.PR, text string, d time.Duration) tea.Cmd {
	if indexOf(m.rows, pr.Ref) >= 0 {
		m.selected = pr.Ref
		m.rebuild()
		return m.setFlash(text, d)
	}
	return m.setFlash(text+" · hidden by filters", d)
}

// addLine is the prompt line. It replaces the footer, so it costs no list row.
// On a narrow pane (RESEARCH Pitfall 6) the input hint goes first, then the typed
// text shows its tail behind a "…" and, if even that has no room, goes: the
// cursor, the checking text and the error outrank the text.
func (m Model) addLine() string {
	const prefix = "add PR: "
	var suffix string
	hint := false
	switch {
	case m.add.errText != "":
		suffix = "   " + warnStyle.Render("✗ "+m.add.errText) + " · esc cancel"
	case m.add.saving:
		// No cancel hint: the save cannot be cancelled (D-05).
		suffix = "   saving " + m.add.ref.String() + " " + m.spin.View()
	case m.add.checking:
		suffix = "   checking " + m.add.ref.String() + " " + m.spin.View()
	default:
		suffix = faintStyle.Render("   (URL or owner/repo#n · enter add · esc cancel)")
		hint = true
	}
	text := m.add.text
	if m.w > 0 {
		fixed := lipgloss.Width(prefix) + 1 // the prefix and the cursor
		if hint && fixed+lipgloss.Width(text)+lipgloss.Width(suffix) > m.w {
			suffix = ""
		}
		if room := m.w - fixed - lipgloss.Width(suffix); lipgloss.Width(text) > room {
			switch {
			case room > 1:
				text = trimLeft(text, room)
			case room == 1:
				text = "…"
			default:
				text = ""
			}
		}
	}
	return m.cut(prefix + text + "█" + suffix)
}

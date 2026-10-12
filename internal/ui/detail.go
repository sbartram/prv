package ui

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	bkey "charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sbartram/prv/internal/domain"
)

// maxCheckNameW and maxReviewerNameW cap the name columns of the detail view.
const (
	maxCheckNameW    = 40
	maxReviewerNameW = 30
	minCheckNameW    = 4
)

// detailState is the full-screen detail view of one pull request (DETL-01). It
// keeps its own copy of the PR and of its ref, so the view never reads the
// list rows by cursor once it is open.
type detailState struct {
	open     bool
	ref      domain.Ref
	pr       domain.PR
	data     *domain.Detail // nil until the first result
	err      error          // the last failed fetch; shown only while data is nil
	at       time.Time      // when data was read
	expanded bool           // the folded passing checks are listed
	gone     bool           // the PR left the rows on screen; the last data stays (D-12)
	errAt    time.Time      // when the last detail fetch failed
	vp       viewport.Model // scrolls the body; driven only by prv's keymap
}

// openDetail opens the detail view of pr and returns the Cmd that fetches its
// checks and reviewers. Without a DetailFetcher it returns nil and the view
// says so.
func (m *Model) openDetail(pr domain.PR) tea.Cmd {
	m.detail = detailState{open: true, ref: pr.Ref, pr: pr, vp: viewport.New()}
	m.syncDetail()
	return m.startDetailFetch()
}

// startDetailFetch starts a detail fetch for the open PR. It returns nil when
// the fetcher has no detail method or a detail fetch is already running.
func (m *Model) startDetailFetch() tea.Cmd {
	df, ok := m.fetcher.(DetailFetcher)
	if !ok || m.detailInFlight {
		return nil
	}
	m.detailSeq++
	m.detailInFlight = true
	return fetchDetail(m.ctx, df, m.detail.pr.ID, m.detail.ref, m.detailSeq)
}

// closeDetail returns to the list. The selection was never touched, so the
// same PR is selected. Bumping the sequence number orphans a result still on
// its way.
func (m *Model) closeDetail() {
	m.detail = detailState{}
	m.detailSeq++
	m.detailInFlight = false
	m.clampOffset()
}

// detailKey handles one key press while the detail view is open (D-09). Keys
// that the detail does not use are ignored: the number toggles and the prompt
// belong to the list.
func (m Model) detailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case bkey.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case bkey.Matches(msg, m.keys.Help):
		m.showHelp = true
	case bkey.Matches(msg, m.keys.Back):
		m.closeDetail()
	case bkey.Matches(msg, m.keys.Open):
		cmd := m.openURL(m.detail.pr.URL)
		return m, cmd
	case bkey.Matches(msg, m.keys.Refresh):
		refresh := m.manualRefresh()
		var retry tea.Cmd
		if m.detail.data == nil && m.detail.err != nil {
			retry = m.startDetailFetch() // the CHECKS line promises "r retry"
		}
		return m, tea.Batch(refresh, retry)
	case bkey.Matches(msg, m.keys.Down):
		m.detail.vp.ScrollDown(1)
	case bkey.Matches(msg, m.keys.Up):
		m.detail.vp.ScrollUp(1)
	case bkey.Matches(msg, m.keys.Expand):
		m.detail.expanded = !m.detail.expanded
		m.syncDetail()
	}
	return m, nil
}

// applyDetail stores a detail result. A result for a view that was closed, or
// from an older fetch, or for another PR, is dropped.
func (m *Model) applyDetail(msg detailMsg) {
	if errors.Is(msg.err, context.Canceled) {
		return // the program is quitting
	}
	if !m.detail.open || msg.seq != m.detailSeq || msg.ref != m.detail.ref {
		return
	}
	m.detailInFlight = false
	if msg.err != nil {
		m.detail.err = msg.err
	} else {
		data := msg.data
		m.detail.data, m.detail.err, m.detail.at = &data, nil, m.now()
	}
	m.syncDetail()
}

// syncDetail gives the viewport its size and its lines. The body is the pane
// minus the header and the footer. Before the first size message there is
// nothing to scroll: the view then draws every line.
func (m *Model) syncDetail() {
	if m.w <= 0 || m.h <= 0 {
		return
	}
	m.detail.vp.SetWidth(m.w)
	m.detail.vp.SetHeight(max(0, m.h-2))
	m.detail.vp.SetContentLines(m.detailLines(m.w))
}

// detailHeader is the first line of the detail view: the app name, the PR and
// the same status segments as the list header.
func (m Model) detailHeader() string {
	segs := []segment{{"prv", 1}, {m.detail.ref.String(), 6}}
	return fit(append(segs, m.statusSegments()...), " · ", m.w)
}

// renderDetail is the detail view: a header, the scrolling body and the
// footer. A pane of one line shows only the header, a pane of two lines the
// header and the footer. Not sized yet, every line is drawn uncut.
func (m Model) renderDetail() string {
	lines := []string{m.detailHeader()}
	switch {
	case m.w <= 0 || m.h <= 0:
		lines = append(lines, m.detailLines(0)...)
		lines = append(lines, m.footer())
	default:
		if m.h >= 3 {
			for _, l := range strings.Split(m.detail.vp.View(), "\n") {
				lines = append(lines, strings.TrimRight(l, " "))
			}
		}
		if m.h >= 2 {
			lines = append(lines, m.footer())
		}
	}
	for i, l := range lines {
		lines[i] = m.cut(l)
	}
	return strings.Join(lines, "\n")
}

// detailLines builds the body: the title and status, the CHECKS section and
// the REVIEWERS section. A width of 0 or less (not sized yet) cuts nothing.
func (m Model) detailLines(width int) []string {
	lines := m.detailTitle(width)
	lines = append(lines, "")
	lines = append(lines, m.checkLines(width)...)
	lines = append(lines, "")
	lines = append(lines, m.reviewerLines(width)...)
	if width > 0 {
		for i, l := range lines {
			lines[i] = ansi.Truncate(l, width, "")
		}
	}
	return lines
}

// detailTitle is the title and the "STATE · REVIEW · MERGE" status. Side by
// side when the title keeps at least 10 cells, else on two lines.
func (m Model) detailTitle(width int) []string {
	pr := m.detail.pr
	title := pr.Title
	if pr.IsDraft {
		title = "[draft] " + title
	}
	plainStatus := pr.State.String() + " · " + pr.Review.Badge()
	status := pr.State.String() + " · " + reviewStyle(pr.Review).Render(pr.Review.Badge())
	if pr.Merge != domain.MergeNone {
		plainStatus += " · " + pr.Merge.Badge()
		status += " · " + mergeStyle(pr.Merge).Render(pr.Merge.Badge())
	}
	if width <= 0 {
		return []string{headerStyle.Render(title) + "  " + status}
	}
	sw := lipgloss.Width(plainStatus)
	if sw+2+min(lipgloss.Width(title), 10) <= width {
		room := width - sw - 2
		t := ansi.Truncate(title, room, "…")
		pad := room - lipgloss.Width(t)
		return []string{headerStyle.Render(t) + strings.Repeat(" ", pad) + "  " + status}
	}
	return []string{
		headerStyle.Render(ansi.Truncate(title, width, "…")),
		ansi.Truncate(status, width, "…"),
	}
}

// checksSummary is the text after "CHECKS": the counts of the de-duplicated
// checks (D-03, D-08), or why there are none to count.
func (m Model) checksSummary() string {
	d := m.detail
	if d.data == nil {
		if _, ok := m.fetcher.(DetailFetcher); !ok {
			return "details not available"
		}
		if d.err != nil {
			reason := "error"
			var sh shorter
			if errors.As(d.err, &sh) {
				reason = sh.Short()
			}
			return checkStyle(domain.CheckFailing).Render("✗ could not load checks ("+reason+")") + " · r retry"
		}
		return "loading…"
	}
	c := d.data.Counts
	var segs []string
	add := func(n int, word string, st domain.CheckState) {
		if n > 0 {
			segs = append(segs, checkStyle(st).Render(strconv.Itoa(n)+" "+word))
		}
	}
	add(c.Failing, "failing", domain.CheckFailing)
	add(c.Cancelled, "cancelled", domain.CheckCancelled)
	add(c.Running, "running", domain.CheckRunning)
	add(c.Passing, "passing", domain.CheckPassing)
	add(c.Neutral, "skipped", domain.CheckNeutral)
	if d.data.Truncated {
		// D-03, D-08: keep the exact counts and say how much was read.
		segs = append(segs, faintStyle.Render("read "+strconv.Itoa(d.data.Read)+" of "+strconv.Itoa(d.data.Total)+" runs"))
	}
	if len(segs) == 0 {
		return "no checks"
	}
	return strings.Join(segs, " · ")
}

// checkLines is the CHECKS section: the summary line, then one line per check
// in the order the detail data gives. Unless expanded, only failing, cancelled
// and running checks are listed; the others make one faint fold line (D-10).
func (m Model) checkLines(width int) []string {
	lines := []string{headerStyle.Render("CHECKS") + "  " + m.checksSummary()}
	if m.detail.data == nil {
		return lines
	}
	checks, folded := m.detail.data.Checks, 0
	if !m.detail.expanded {
		var shown []domain.Check
		for _, c := range checks {
			switch c.State {
			case domain.CheckFailing, domain.CheckCancelled, domain.CheckRunning:
				shown = append(shown, c)
			default:
				folded++
			}
		}
		checks = shown
	}

	nameW, wordW, srcW := 0, 0, 0
	for _, c := range checks {
		nameW = max(nameW, lipgloss.Width(c.Name))
		wordW = max(wordW, lipgloss.Width(c.Word))
		srcW = max(srcW, lipgloss.Width(c.Source))
	}
	nameW = min(nameW, maxCheckNameW)
	// glyph, space and the two-cell indent are 4 cells; two gaps follow.
	rowW := func(withSource bool) int {
		w := 4 + nameW + 2 + wordW
		if withSource && srcW > 0 {
			w += 2 + srcW
		}
		return w
	}
	withSource := true
	if width > 0 && rowW(true) > width {
		withSource = false // drop the Source column for every line
		if over := rowW(false) - width; over > 0 {
			nameW = max(minCheckNameW, nameW-over)
		}
	}
	for _, c := range checks {
		name := ansi.Truncate(c.Name, nameW, "…")
		namePad := strings.Repeat(" ", nameW-lipgloss.Width(name))
		wordPad := strings.Repeat(" ", wordW-lipgloss.Width(c.Word))
		st := checkStyle(c.State)
		line := "  " + st.Render(c.State.Glyph()) + " " + name + namePad + "  " + st.Render(c.Word)
		if withSource && c.Source != "" {
			line += wordPad + "  " + c.Source
		}
		if c.State == domain.CheckNeutral || c.State == domain.CheckOther {
			line = faintStyle.Render(ansi.Strip(line)) // neutral and unknown: faint as a whole
		}
		lines = append(lines, line)
	}
	if folded > 0 {
		lines = append(lines, faintStyle.Render("  … +"+strconv.Itoa(folded)+" more passing"))
	}
	return lines
}

// reviewerName is how a reviewer reads in the detail view: the viewer as "me",
// a team as "@org/slug".
func (m Model) reviewerName(r domain.Reviewer) string {
	switch {
	case r.Kind == domain.ReviewerTeam:
		return "@" + r.Name
	case m.snap.Viewer != "" && strings.EqualFold(r.Name, m.snap.Viewer):
		return "me"
	}
	return r.Name
}

// reviewerLines is the REVIEWERS section: every reviewer with the state of
// the review. Before the first detail result it shows the reviewers of the
// list row. Reviewers whose team is hidden make one line.
func (m Model) reviewerLines(width int) []string {
	revs := m.detail.pr.Reviewers
	if m.detail.data != nil {
		revs = m.detail.data.Reviewers
	}
	if len(revs) == 0 {
		return []string{headerStyle.Render("REVIEWERS") + "  none"}
	}
	lines := []string{headerStyle.Render("REVIEWERS")}

	names := make([]string, len(revs))
	nameW, hidden := 0, 0
	for i, r := range revs {
		if r.Kind == domain.ReviewerHidden {
			hidden++
			continue
		}
		names[i] = ansi.Truncate(m.reviewerName(r), maxReviewerNameW, "…")
		nameW = max(nameW, lipgloss.Width(names[i]))
	}
	hiddenText := "1 team (name hidden)"
	if hidden > 1 {
		hiddenText = strconv.Itoa(hidden) + " teams (names hidden)"
	}
	hiddenDone := false
	for i, r := range revs {
		name := names[i]
		if r.Kind == domain.ReviewerHidden {
			if hiddenDone {
				continue
			}
			hiddenDone, name = true, hiddenText
		}
		pad := strings.Repeat(" ", max(0, nameW-lipgloss.Width(name)))
		st := reviewerStyle(r.State)
		line := "  " + st.Render(r.State.Glyph()) + " " + name + pad + "  " + st.Render(r.State.Word())
		if r.State == domain.ReviewerCommented || r.State == domain.ReviewerDismissed {
			line = faintStyle.Render(ansi.Strip(line))
		}
		lines = append(lines, line)
	}
	return lines
}

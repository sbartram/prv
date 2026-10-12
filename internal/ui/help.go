package ui

import (
	"strconv"
	"strings"
	"time"

	"github.com/sbartram/prv/internal/domain"
)

// renderHelp is the full-screen help overlay (D-17): the key table drawn from
// the same keymap as the footer hints, a legend of every badge, the settings in
// effect, the API budget and the config warnings. It fits the pane: when it is
// too tall the warnings are cut and the return hint stays on the last line.
func (m Model) renderHelp() string {
	const returnHint = "press any key to return"

	var b strings.Builder
	b.WriteString(headerStyle.Render("prv — keys and badges"))
	b.WriteString("\n\n")

	h := m.help // a copy: SetWidth must not change the model
	h.SetWidth(m.w)
	b.WriteString(h.FullHelpView(m.keys.FullHelp()))
	b.WriteString("\n\n")

	b.WriteString("Badges\n")
	b.WriteString("  review  ")
	reviews := []domain.Review{domain.ReviewApproved, domain.ReviewChangesRequested, domain.ReviewRequired, domain.ReviewNone}
	for i, r := range reviews {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(reviewStyle(r).Render(r.Badge()))
	}
	b.WriteString("\n  ci      ")
	cis := []domain.CI{domain.CIPassing, domain.CIFailing, domain.CIRunning, domain.CINone, domain.CIUnknown}
	for i, c := range cis {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(ciStyle(c).Render(c.Badge()))
	}
	b.WriteString("\n  merge   ")
	merges := []domain.Merge{domain.MergeMergeable, domain.MergeBlocked, domain.MergeBehind, domain.MergeConflicts}
	for i, mg := range merges {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(mergeStyle(mg).Render(mg.Badge()))
	}
	b.WriteString("  – unknown or draft")
	b.WriteString("\n  reviewers ✗ changes requested  … pending  ✓ approved")
	b.WriteString("\n            …team name hidden  +N more")
	b.WriteString("\n  checks    ")
	checks := []struct {
		state domain.CheckState
		word  string
	}{{domain.CheckFailing, "failing"}, {domain.CheckCancelled, "cancelled"}, {domain.CheckRunning, "running"},
		{domain.CheckPassing, "passing"}, {domain.CheckNeutral, "skipped"}}
	for i, c := range checks {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(checkStyle(c.state).Render(c.state.Glyph() + " " + c.word))
	}
	b.WriteString("\n  [draft] draft PR · dimmed rows are drafts, merged or closed\n\n")

	b.WriteString("config " + domain.Clean(m.configPath) + " · every " + fmtInterval(m.interval) +
		" · " + strconv.Itoa(m.opts.ClosedDays) + " days\n")
	if rl := m.snap.RateLimit; rl.Limit > 0 {
		b.WriteString("api " + strconv.Itoa(rl.Remaining) + " / " + strconv.Itoa(rl.Limit) +
			", resets " + m.clock(rl.ResetAt, "15:04") + "\n")
	} else {
		b.WriteString("api limit not known yet\n")
	}
	// D-15: the warnings stay listed for the whole session, unlike the footer.
	for _, w := range m.warnings {
		b.WriteString(warnStyle.Render(domain.Clean(w)) + "\n")
	}
	b.WriteString("\n" + returnHint)

	lines := strings.Split(m.wrap(b.String()), "\n")
	if m.h > 0 && len(lines) > m.h {
		lines = append(lines[:m.h-1], returnHint)
	}
	return strings.Join(lines, "\n")
}

// fmtInterval writes a refresh interval the way the config does: "2m" for
// whole minutes of two or more, else whole seconds as "60s".
func fmtInterval(d time.Duration) string {
	if d >= 2*time.Minute && d%time.Minute == 0 {
		return strconv.Itoa(int(d/time.Minute)) + "m"
	}
	if d%time.Second == 0 {
		return strconv.Itoa(int(d/time.Second)) + "s"
	}
	return d.String()
}

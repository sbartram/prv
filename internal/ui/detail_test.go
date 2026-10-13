package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/sbartram/prv/internal/domain"
)

// detailFake is a Fetcher that also fetches details. It records the ids it is
// asked for, and returns a configured Detail or error, or panics.
type detailFake struct {
	*fakeFetcher
	detail   domain.Detail
	err      error
	panicVal any
	ids      []string
}

func newDetailFake(d domain.Detail) *detailFake {
	return &detailFake{fakeFetcher: &fakeFetcher{}, detail: d}
}

func (f *detailFake) FetchDetail(_ context.Context, id string) (domain.Detail, error) {
	f.ids = append(f.ids, id)
	if f.panicVal != nil {
		panic(f.panicVal)
	}
	return f.detail, f.err
}

// fixtureDetail is the Detail of the PR acme/api#102 of fixtureSix: already
// sorted, with the counts of its de-duplicated checks, and fully read.
func fixtureDetail() domain.Detail {
	checks := []domain.Check{
		{Name: "unit", Source: "ci.yml", State: domain.CheckFailing, Word: "failure"},
		{Name: "lint", Source: "ci.yml", State: domain.CheckFailing, Word: "timed out"},
		{Name: "integration", Source: "ci.yml", State: domain.CheckCancelled, Word: "cancelled"},
		{Name: "e2e", Source: "e2e.yml", State: domain.CheckRunning, Word: "running"},
		{Name: "build", Source: "ci.yml", State: domain.CheckPassing, Word: "success"},
		{Name: "docs", Source: "ci.yml", State: domain.CheckPassing, Word: "success"},
		{Name: "vet", Source: "ci.yml", State: domain.CheckPassing, Word: "success"},
		{Name: "preview", Source: "deploy.yml", State: domain.CheckNeutral, Word: "skipped"},
	}
	domain.SortChecks(checks)
	return domain.Detail{
		Checks: checks,
		Reviewers: []domain.Reviewer{
			{Name: "bob", State: domain.ReviewerChangesRequested},
			{Name: "carol", State: domain.ReviewerPending},
			{Kind: domain.ReviewerHidden, State: domain.ReviewerPending},
			{Kind: domain.ReviewerHidden, State: domain.ReviewerPending},
			{Name: "alice", State: domain.ReviewerApproved},
			{Name: "dave", State: domain.ReviewerCommented},
			{Name: "erin", State: domain.ReviewerDismissed},
		},
		Counts: domain.CountChecks(checks),
		Read:   len(checks), Total: len(checks),
	}
}

// detailModel builds a sized model on the list screen with fixtureSix loaded
// and f as its fetcher.
func detailModel(t *testing.T, f Fetcher, opts Options, w, h int) tea.Model {
	t.Helper()
	opts.ClosedDays, opts.Now = 14, fixedNow
	var m tea.Model = New(t.Context(), f, opts)
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m, _ = m.Update(fetchedMsg{snap: fixtureSix()})
	return m
}

// openDetailOn presses j once, so acme/api#102 is selected, then enter, and
// delivers the detail result.
func openDetailOn(t *testing.T, m tea.Model) tea.Model {
	t.Helper()
	m = press(m, "j")
	m, cmd := m.Update(key("enter"))
	return runCmd(m, cmd)
}

func TestDetailOpenClose(t *testing.T) {
	f := newDetailFake(fixtureDetail())
	m := detailModel(t, f, Options{}, 120, 40)
	m = press(m, "j")
	listBefore := plain(m)
	cursor := m.(Model).cursor

	m, cmd := m.Update(key("enter"))
	if !m.(Model).detail.open {
		t.Fatal("enter did not open the detail view")
	}
	if cmd == nil {
		t.Fatal("enter started no detail fetch")
	}
	if v := plain(m); !strings.Contains(v, "CHECKS  loading…") {
		t.Errorf("before the first result the view should say loading:\n%s", v)
	}
	m = runCmd(m, cmd)
	v := plain(m)
	if !strings.HasPrefix(v, "prv · acme/api#102") {
		t.Errorf("the view should start with the header, got:\n%s", v)
	}
	if strings.Contains(v, "CHECKS  loading…") {
		t.Errorf("the result did not replace the loading text:\n%s", v)
	}
	if len(f.ids) != 1 || f.ids[0] != "2" {
		t.Errorf("FetchDetail ids = %v, want exactly [2]", f.ids)
	}
	if strings.Contains(v, "Add retry to the client") {
		t.Errorf("the detail view should replace the list:\n%s", v)
	}

	m = press(m, "esc")
	if m.(Model).detail.open {
		t.Fatal("esc did not close the detail view")
	}
	if got := plain(m); got != listBefore {
		t.Errorf("after esc the list differs from before enter:\nbefore:\n%s\nafter:\n%s", listBefore, got)
	}
	if mm := m.(Model); mm.cursor != cursor || mm.selected != (domain.Ref{Repo: "acme/api", Number: 102}) {
		t.Errorf("cursor/selected = %d/%v after esc, want %d/acme/api#102", mm.cursor, mm.selected, cursor)
	}
	if got := markedRef(t, m); got != "acme/api#102" {
		t.Errorf("marked row = %q, want acme/api#102", got)
	}
}

func TestDetailEnterOnlyOnRows(t *testing.T) {
	f := newDetailFake(fixtureDetail())
	for _, state := range []string{"loading", "empty"} {
		var m tea.Model = New(t.Context(), f, Options{ClosedDays: 14, Now: fixedNow})
		m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		if state == "empty" {
			m, _ = m.Update(fetchedMsg{snap: domain.Snapshot{Viewer: "octo"}})
		}
		m, cmd := m.Update(key("enter"))
		if m.(Model).detail.open || cmd != nil {
			t.Errorf("%s: enter opened a detail view", state)
		}
	}
	// A filter that hides every row leaves nothing to open.
	m := detailModel(t, f, Options{}, 120, 40)
	m = press(m, "1", "2", "3")
	if m, _ = m.Update(key("enter")); m.(Model).detail.open {
		t.Error("enter opened a detail view with no rows")
	}
}

func TestDetailRender(t *testing.T) {
	// Expanded, so that the passing and skipped checks are listed too.
	m := press(openDetailOn(t, detailModel(t, newDetailFake(fixtureDetail()), Options{}, 120, 40)), "e")
	v := plain(m)
	order := []string{
		"prv · acme/api#102 · updated 14:32:05",
		"Rework the cache layer", "open · ✗ changes · ⊘ blocked",
		"CHECKS  2 failing · 1 cancelled · 1 running · 3 passing · 1 skipped",
		"✗ lint", "✗ unit", "⊘ integration", "● e2e", "✓ build", "✓ docs", "✓ vet", "– preview",
		"REVIEWERS",
		"✗ bob", "… carol", "… 2 teams (names hidden)", "✓ alice", "· dave", "– erin",
	}
	at := 0
	for _, want := range order {
		i := strings.Index(v[at:], want)
		if i < 0 {
			t.Fatalf("%q not found in order after offset %d:\n%s", want, at, v)
		}
		at += i + len(want)
	}
	for _, want := range []string{"timed out", "ci.yml", "e2e.yml", "changes requested", "approved", "commented", "dismissed", "pending"} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing %q:\n%s", want, v)
		}
	}
	ls := strings.Split(v, "\n")
	if last := ls[len(ls)-1]; !strings.Contains(last, "esc back") {
		t.Errorf("footer = %q, want the detail hints", last)
	}
	for _, l := range ls {
		if l != strings.TrimRight(l, " ") {
			t.Errorf("line has trailing spaces: %q", l)
		}
	}

	t.Run("the viewer reads me and a team reads @org/slug", func(t *testing.T) {
		d := fixtureDetail()
		d.Reviewers = []domain.Reviewer{
			{Name: "OCTO", State: domain.ReviewerApproved},
			{Name: "acme/core", Kind: domain.ReviewerTeam, State: domain.ReviewerPending},
			{Kind: domain.ReviewerHidden, State: domain.ReviewerPending},
		}
		v := plain(openDetailOn(t, detailModel(t, newDetailFake(d), Options{}, 120, 40)))
		for _, want := range []string{"✓ me", "… @acme/core", "… 1 team (name hidden)  pending"} {
			if !strings.Contains(v, want) {
				t.Errorf("view missing %q:\n%s", want, v)
			}
		}
	})
	t.Run("a narrow pane puts the status on its own line", func(t *testing.T) {
		v := plain(openDetailOn(t, detailModel(t, newDetailFake(fixtureDetail()), Options{}, 30, 24)))
		ls := strings.Split(v, "\n")
		if len(ls) < 3 || ls[1] != "Rework the cache layer" || ls[2] != "open · ✗ changes · ⊘ blocked" {
			t.Errorf("title and status should be lines 2 and 3:\n%s", v)
		}
	})
	t.Run("a wide pane puts the status at the right edge", func(t *testing.T) {
		ls := strings.Split(plain(openDetailOn(t, detailModel(t, newDetailFake(fixtureDetail()), Options{}, 80, 24))), "\n")
		if want := "Rework the cache layer"; !strings.HasPrefix(ls[1], want) || !strings.HasSuffix(ls[1], "⊘ blocked") {
			t.Errorf("line 2 = %q, want the title then the status", ls[1])
		}
	})
}

func TestDetailKeys(t *testing.T) {
	op := &fakeOpener{}
	snap := fixtureSix()
	for i := range snap.PRs {
		snap.PRs[i].URL = "https://github.com/" + snap.PRs[i].Ref.Repo + "/pull/" + itoa(snap.PRs[i].Ref.Number)
	}
	build := func() tea.Model {
		f := newDetailFake(fixtureDetail())
		var m tea.Model = New(t.Context(), f, Options{ClosedDays: 14, Now: fixedNow, Opener: op})
		m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
		m, _ = m.Update(fetchedMsg{snap: snap})
		return openDetailOn(t, m)
	}

	t.Run("j, k and the arrows scroll the detail and not the list", func(t *testing.T) {
		m := build()
		before := m.(Model)
		if before.detail.vp.YOffset() != 0 {
			t.Fatalf("offset starts at %d", before.detail.vp.YOffset())
		}
		for _, step := range []struct {
			k    string
			want int
		}{{"j", 1}, {"down", 2}, {"k", 1}, {"up", 0}} {
			if step.k == "down" || step.k == "up" {
				m, _ = m.Update(special(map[string]rune{"down": tea.KeyDown, "up": tea.KeyUp}[step.k]))
			} else {
				m = press(m, step.k)
			}
			mm := m.(Model)
			if got := mm.detail.vp.YOffset(); got != step.want {
				t.Errorf("after %s offset = %d, want %d", step.k, got, step.want)
			}
			if mm.cursor != before.cursor || mm.selected != before.selected {
				t.Errorf("after %s the list selection moved to %d/%v", step.k, mm.cursor, mm.selected)
			}
		}
	})
	t.Run("q quits and ctrl+c quits", func(t *testing.T) {
		for _, k := range []string{"q", "ctrl+c"} {
			_, cmd := build().Update(key(k))
			if cmd == nil {
				t.Fatalf("%s returned no command", k)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Errorf("%s did not quit", k)
			}
		}
	})
	t.Run("? opens the help and any key returns to the detail", func(t *testing.T) {
		m := press(build(), "?")
		if v := plain(m); !strings.Contains(v, "press any key to return") {
			t.Fatalf("no help overlay:\n%s", v)
		}
		m = press(m, "x")
		v := plain(m)
		if !m.(Model).detail.open || !strings.Contains(v, "acme/api#102") || strings.Contains(v, "press any key to return") {
			t.Errorf("the detail should be back:\n%s", v)
		}
	})
	t.Run("1, 2, 3 and / do nothing", func(t *testing.T) {
		m := build()
		want := plain(m)
		for _, k := range []string{"1", "2", "3", "/"} {
			m = press(m, k)
			if got := plain(m); got != want {
				t.Errorf("%s changed the view:\n%s", k, got)
			}
		}
		if mm := m.(Model); mm.prompting || mm.show != domain.AllStates || !mm.detail.open {
			t.Errorf("state changed: prompting=%v show=%v open=%v", mm.prompting, mm.show, mm.detail.open)
		}
	})
	t.Run("o opens the PR of the detail", func(t *testing.T) {
		op.urls = nil
		_, cmd := build().Update(key("o"))
		if cmd == nil {
			t.Fatal("o returned no command")
		}
		cmd()
		if len(op.urls) != 1 || op.urls[0] != "https://github.com/acme/api/pull/102" {
			t.Errorf("opened %v, want the URL of acme/api#102", op.urls)
		}
	})
	t.Run("r refreshes", func(t *testing.T) {
		m := build()
		if m.(Model).inFlight {
			t.Fatal("a fetch is already running")
		}
		m, cmd := m.Update(key("r"))
		if !m.(Model).inFlight || cmd == nil {
			t.Errorf("r did not start a refresh: inFlight=%v cmd=%v", m.(Model).inFlight, cmd != nil)
		}
	})
	t.Run("enter and other keys are ignored", func(t *testing.T) {
		m := build()
		want := plain(m)
		for _, k := range []string{"enter", "x", "z"} {
			var cmd tea.Cmd
			m, cmd = m.Update(key(k))
			if cmd != nil || plain(m) != want {
				t.Errorf("%s did something", k)
			}
		}
	})
}

func TestDetailStaleDropped(t *testing.T) {
	ref := domain.Ref{Repo: "acme/api", Number: 102}
	m := detailModel(t, newDetailFake(fixtureDetail()), Options{}, 120, 40)
	m = press(m, "j", "enter") // seq 1
	m = press(m, "esc", "j")   // closes: seq 2
	m = press(m, "enter")      // seq 3 on acme/web#7
	m, _ = m.Update(detailMsg{seq: 1, ref: ref, data: fixtureDetail()})
	if m.(Model).detail.data != nil {
		t.Error("a result with an old sequence number was applied")
	}
	if !m.(Model).detailInFlight {
		t.Error("a dropped result cleared detailInFlight")
	}
	ref3 := m.(Model).detail.ref
	d := fixtureDetail()
	m, _ = m.Update(detailMsg{seq: 3, ref: ref, data: d})
	if m.(Model).detail.data != nil {
		t.Errorf("a result for another PR (%v, open %v) was applied", ref, ref3)
	}
	m, _ = m.Update(detailMsg{seq: 3, ref: ref3, data: d})
	if m.(Model).detail.data == nil {
		t.Error("the current result was dropped")
	}
	if m.(Model).detailInFlight {
		t.Error("the current result left detailInFlight set")
	}
	t.Run("a result after close is dropped", func(t *testing.T) {
		m := press(detailModel(t, newDetailFake(fixtureDetail()), Options{}, 120, 40), "enter")
		seq := m.(Model).detailSeq
		m = press(m, "esc")
		m, _ = m.Update(detailMsg{seq: seq, ref: m.(Model).selected, data: fixtureDetail()})
		if m.(Model).detail.data != nil || m.(Model).detail.open {
			t.Error("a closed detail view took a result")
		}
	})
	t.Run("a canceled fetch is ignored", func(t *testing.T) {
		m := press(detailModel(t, newDetailFake(fixtureDetail()), Options{}, 120, 40), "enter")
		mm := m.(Model)
		m, _ = m.Update(detailMsg{seq: mm.detailSeq, ref: mm.detail.ref, err: context.Canceled})
		if m.(Model).detail.err != nil {
			t.Error("context.Canceled was stored as an error")
		}
	})
}

func TestDetailWithoutDetailFetcher(t *testing.T) {
	m := detailModel(t, &fakeFetcher{}, Options{}, 120, 40)
	m = press(m, "j")
	m, cmd := m.Update(key("enter"))
	if cmd != nil {
		t.Error("a fetcher without FetchDetail started a fetch")
	}
	v := plain(m)
	for _, want := range []string{"CHECKS  details not available", "REVIEWERS", "✗ bob", "… carol", "✓ dave"} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing %q:\n%s", want, v)
		}
	}
	m = press(m, "esc")
	if m.(Model).detail.open {
		t.Error("esc did not close")
	}
}

func TestDetailPanicRecovered(t *testing.T) {
	f := newDetailFake(domain.Detail{})
	f.panicVal = "secret-token-ghp_123"
	m := detailModel(t, f, Options{}, 120, 40)
	m = press(m, "j")
	m, cmd := m.Update(key("enter"))
	m = runCmd(m, cmd)
	v := plain(m)
	if !strings.Contains(v, "CHECKS  ✗ could not load checks") {
		t.Errorf("no failure line:\n%s", v)
	}
	if strings.Contains(v, "ghp_123") || strings.Contains(v, "secret") {
		t.Errorf("the panic value reached the screen:\n%s", v)
	}
	if err := m.(Model).detail.err; err == nil || err.Error() != "unexpected internal error while loading details" {
		t.Errorf("err = %v, want the fixed message", err)
	}
	// The reviewers of the list row are still shown.
	if !strings.Contains(v, "✗ bob") {
		t.Errorf("reviewers missing after a failed fetch:\n%s", v)
	}
	t.Run("a short reason is shown, the message is not", func(t *testing.T) {
		f := newDetailFake(domain.Detail{})
		f.err = &shortErr{short: "HTTP 502"}
		v := plain(runDetailOpen(t, detailModel(t, f, Options{}, 120, 40)))
		if !strings.Contains(v, "could not load checks (HTTP 502)") || strings.Contains(v, "ghp_SECRET") {
			t.Errorf("failure line wrong:\n%s", v)
		}
	})
	t.Run("r retries the fetch after a failure", func(t *testing.T) {
		f := newDetailFake(domain.Detail{})
		f.err = errors.New("boom")
		m := runDetailOpen(t, detailModel(t, f, Options{}, 120, 40))
		f.err = nil
		f.detail = fixtureDetail()
		m, cmd := m.Update(key("r"))
		m = runCmd(m, cmd)
		if !strings.Contains(plain(m), "CHECKS  2 failing") {
			t.Errorf("r did not refetch the detail:\n%s", plain(m))
		}
	})
}

// runDetailOpen presses enter on the first row and delivers the result.
func runDetailOpen(t *testing.T, m tea.Model) tea.Model {
	t.Helper()
	m, cmd := m.Update(key("enter"))
	return runCmd(m, cmd)
}

func TestDetailEmpty(t *testing.T) {
	v := plain(runDetailOpen(t, detailModel(t, newDetailFake(domain.Detail{}), Options{}, 120, 40)))
	for _, want := range []string{"CHECKS  no checks", "REVIEWERS  none"} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing %q:\n%s", want, v)
		}
	}
}

// bigDetail is a detail with 2 failing, 1 cancelled, 1 running, 10 passing and
// 2 skipped checks, and the reviewers of fixtureDetail.
func bigDetail() domain.Detail {
	var checks []domain.Check
	add := func(state domain.CheckState, word string, names ...string) {
		for _, n := range names {
			checks = append(checks, domain.Check{Name: n, Source: "ci.yml", State: state, Word: word})
		}
	}
	add(domain.CheckFailing, "failure", "fail-a", "fail-b")
	add(domain.CheckCancelled, "cancelled", "cancel-a")
	add(domain.CheckRunning, "running", "run-a")
	add(domain.CheckPassing, "success", "pass-01", "pass-02", "pass-03", "pass-04", "pass-05",
		"pass-06", "pass-07", "pass-08", "pass-09", "pass-10")
	add(domain.CheckNeutral, "skipped", "skip-a", "skip-b")
	domain.SortChecks(checks)
	d := fixtureDetail()
	d.Checks, d.Counts, d.Read, d.Total = checks, domain.CountChecks(checks), len(checks), len(checks)
	return d
}

func TestDetailFold(t *testing.T) {
	const fold = "… +12 more passing or skipped"
	always := []string{"fail-a", "fail-b", "cancel-a", "run-a"}
	folded := []string{"pass-01", "pass-05", "pass-10", "skip-a", "skip-b"}
	open := func(d domain.Detail) tea.Model {
		return runDetailOpen(t, detailModel(t, newDetailFake(d), Options{}, 120, 40))
	}

	m := open(bigDetail())
	v := plain(m)
	for _, want := range always {
		if !strings.Contains(v, want) {
			t.Errorf("collapsed view missing %q:\n%s", want, v)
		}
	}
	for _, not := range folded {
		if strings.Contains(v, not) {
			t.Errorf("collapsed view lists %q:\n%s", not, v)
		}
	}
	if i, j, k := strings.Index(v, "run-a"), strings.Index(v, fold), strings.Index(v, "REVIEWERS"); !(i < j && j < k) {
		t.Errorf("the fold line should sit after the last listed check and before REVIEWERS:\n%s", v)
	}
	if !strings.Contains(v, "2 failing · 1 cancelled · 1 running · 10 passing · 2 skipped") {
		t.Errorf("the summary should count every check even when folded:\n%s", v)
	}

	m = press(m, "e")
	v = plain(m)
	for _, want := range append(always, folded...) {
		if !strings.Contains(v, want) {
			t.Errorf("expanded view missing %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "more passing or skipped") {
		t.Errorf("expanded view still has the fold line:\n%s", v)
	}

	m = press(m, "e")
	if v := plain(m); !strings.Contains(v, fold) || strings.Contains(v, "pass-01") {
		t.Errorf("the second e should collapse again:\n%s", v)
	}

	t.Run("failing, cancelled and running checks are never folded", func(t *testing.T) {
		d := bigDetail()
		d.Checks = d.Checks[:4]
		if v := plain(open(d)); strings.Contains(v, "more passing or skipped") {
			t.Errorf("a fold line with nothing folded:\n%s", v)
		}
	})
	t.Run("an unknown check is listed, never folded", func(t *testing.T) {
		d := fixtureDetail()
		d.Checks = append(d.Checks, domain.Check{Name: "mystery", State: domain.CheckOther, Word: "unknown"})
		domain.SortChecks(d.Checks)
		d.Counts = domain.CountChecks(d.Checks)
		d.Read, d.Total = len(d.Checks), len(d.Checks)

		v := plain(open(d))
		i, j, k := strings.Index(v, "e2e"), strings.Index(v, "? mystery"), strings.Index(v, "… +4 more passing or skipped")
		if i < 0 || j < 0 || k < 0 || !(i < j && j < k) {
			t.Errorf("the unknown check should sit after the running check and before the fold line (e2e=%d mystery=%d fold=%d):\n%s", i, j, k, v)
		}
		if want := "2 failing · 1 cancelled · 1 running · 3 passing · 1 skipped · 1 unknown"; !strings.Contains(v, want) {
			t.Errorf("summary missing %q:\n%s", want, v)
		}

		narrow := runDetailOpen(t, detailModel(t, newDetailFake(d), Options{}, 36, 40))
		if v := plain(narrow); !strings.Contains(v, "? mystery") {
			t.Errorf("the compact view folded the unknown check:\n%s", v)
		}
	})
	t.Run("e scrolls back into range", func(t *testing.T) {
		m := detailModel(t, newDetailFake(bigDetail()), Options{}, 80, 10)
		m = press(runDetailOpen(t, m), "e")
		m = press(m, "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j", "j")
		if mm := m.(Model); mm.detail.vp.YOffset() == 0 {
			t.Fatal("the expanded detail did not scroll")
		}
		m = press(m, "e")
		if got := plain(m); !strings.Contains(got, "esc back") {
			t.Errorf("collapsing broke the view:\n%s", got)
		}
	})
}

func TestDetailTruncatedSummary(t *testing.T) {
	d := fixtureDetail()
	d.Counts = domain.CheckCounts{Failing: 2, Running: 1, Passing: 280, Neutral: 3}
	d.Truncated, d.Read, d.Total = true, 500, 731
	v := plain(runDetailOpen(t, detailModel(t, newDetailFake(d), Options{}, 120, 40)))
	if want := "CHECKS  2 failing · 1 running · 280 passing · 3 skipped · read 500 of 731 runs"; !strings.Contains(v, want) {
		t.Errorf("view missing %q:\n%s", want, v)
	}
	d.Truncated = false
	if v := plain(runDetailOpen(t, detailModel(t, newDetailFake(d), Options{}, 120, 40))); strings.Contains(v, "read ") {
		t.Errorf("a complete read says how much was read:\n%s", v)
	}
}

// TestDetailPartialNotice pins the rendered text of every combination of
// Truncated (check runs were not read) and Partial (a GraphQL error was kept
// next to the data): "read N of M runs" only for the first, the faint notice
// only for the second, and neither one makes "no checks" read as a run count
// (G-03-6, WR-05, D-03, D-08).
func TestDetailPartialNotice(t *testing.T) {
	const notice = "some details could not be loaded"
	const counts = "CHECKS  2 failing · 1 cancelled · 1 running · 3 passing · 1 skipped"
	build := func(f func(d *domain.Detail)) domain.Detail {
		d := fixtureDetail()
		f(&d)
		return d
	}
	for _, tc := range []struct {
		name          string
		d             domain.Detail
		width         int
		want, wantNot []string
		noticeFirst   bool
	}{
		{
			name: "every run read", width: 120, noticeFirst: true,
			d: build(func(d *domain.Detail) {
				d.Partial = true
				d.Read, d.Total = len(d.Checks), len(d.Checks)
			}),
			want:    []string{notice, counts},
			wantNot: []string{" runs"},
		},
		{
			name: "no checks", width: 120,
			d:       domain.Detail{Partial: true},
			want:    []string{"CHECKS  no checks", "REVIEWERS  none", notice},
			wantNot: []string{" runs"},
		},
		{
			name: "Truncated only", width: 120,
			d: build(func(d *domain.Detail) {
				d.Truncated, d.Read, d.Total = true, 500, 731
			}),
			want:    []string{"read 500 of 731 runs"},
			wantNot: []string{notice},
		},
		{
			name: "Truncated and Partial", width: 120,
			d: build(func(d *domain.Detail) {
				d.Truncated, d.Partial, d.Read, d.Total = true, true, 500, 731
			}),
			want: []string{"read 500 of 731 runs", notice},
		},
		{
			name: "neither", width: 120,
			d:       fixtureDetail(),
			want:    []string{counts},
			wantNot: []string{" runs", notice},
		},
		{
			name: "compact pane", width: 36,
			d:       domain.Detail{Partial: true},
			want:    []string{notice},
			wantNot: []string{" runs"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.noticeFirst && (tc.d.Read != tc.d.Total || tc.d.Total == 0) {
				t.Fatalf("Read/Total = %d/%d, want Read == Total > 0 (WR-05 case 1)", tc.d.Read, tc.d.Total)
			}
			v := plain(runDetailOpen(t, detailModel(t, newDetailFake(tc.d), Options{}, tc.width, 40)))
			for _, w := range tc.want {
				if !strings.Contains(v, w) {
					t.Errorf("view missing %q:\n%s", w, v)
				}
			}
			for _, w := range tc.wantNot {
				if strings.Contains(v, w) {
					t.Errorf("view must not contain %q:\n%s", w, v)
				}
			}
			if tc.noticeFirst {
				noticeAt, checksAt := -1, -1
				for i, l := range strings.Split(v, "\n") {
					if noticeAt < 0 && strings.Contains(l, notice) {
						noticeAt = i
					}
					if checksAt < 0 && strings.HasPrefix(strings.TrimSpace(l), "CHECKS") {
						checksAt = i
					}
				}
				if noticeAt < 0 || checksAt < 0 || noticeAt >= checksAt {
					t.Errorf("notice line = %d, CHECKS line = %d, want the notice above CHECKS:\n%s", noticeAt, checksAt, v)
				}
			}
		})
	}
}

func TestDetailFooterHints(t *testing.T) {
	m := runDetailOpen(t, detailModel(t, newDetailFake(fixtureDetail()), Options{}, 120, 40))
	ls := strings.Split(plain(m), "\n")
	last := ls[len(ls)-1]
	at := 0
	for _, want := range []string{"esc back", "o open", "r refresh", "j/k scroll", "e expand", "? help", "q quit"} {
		i := strings.Index(last[at:], want)
		if i < 0 {
			t.Fatalf("footer %q missing %q in order", last, want)
		}
		at += i + len(want)
	}
	if strings.Contains(last, "enter") || strings.Contains(last, "PR #") {
		t.Errorf("footer %q holds list hints", last)
	}
}

// ---- the open detail stays current (plan 03-08 task 1) ----

// urlSnap is selSnap with a github.com URL on every PR.
func urlSnap(nums ...int) domain.Snapshot {
	s := selSnap(nums...)
	for i := range s.PRs {
		s.PRs[i].URL = "https://github.com/a/b/pull/" + itoa(s.PRs[i].Ref.Number)
	}
	return s
}

// liveDetail builds a 120 x h model on a test clock whose timers are recorded
// and never run, with the PRs a/b#nums loaded.
func liveDetail(t *testing.T, f Fetcher, clk *clock, op Opener, h int, nums ...int) tea.Model {
	t.Helper()
	m := New(t.Context(), f, Options{ClosedDays: 14, Now: clk.now, Opener: op})
	m.after = (&afterRec{}).after
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 120, Height: h})
	tm, _ = tm.Update(result(tm, urlSnap(nums...), nil))
	return tm
}

// openRow moves to the n-th row (1-based), presses enter and delivers the
// detail result.
func openRow(t *testing.T, m tea.Model, n int) tea.Model {
	t.Helper()
	for range n - 1 {
		m = press(m, "j")
	}
	m, cmd := m.Update(key("enter"))
	return runCmd(m, cmd)
}

// refresh delivers a successful list refresh with the PRs a/b#nums and runs
// the Cmds it returns.
func refresh(m tea.Model, nums ...int) tea.Model {
	m, cmd := m.Update(result(m, urlSnap(nums...), nil))
	return runCmd(m, cmd)
}

func TestDetailRefreshes(t *testing.T) {
	f := newDetailFake(fixtureDetail())
	m := openRow(t, liveDetail(t, f, &clock{t: fixedNow()}, nil, 40, 1, 2, 3, 4, 5), 3)
	if len(f.ids) != 1 || f.ids[0] != "PR_3" {
		t.Fatalf("FetchDetail ids = %v after enter, want [PR_3]", f.ids)
	}

	m, cmd := m.Update(result(m, urlSnap(1, 2, 3, 4, 5), nil))
	if cmd == nil {
		t.Fatal("a successful refresh returned no commands")
	}
	m = runCmd(m, cmd)
	if len(f.ids) != 2 || f.ids[1] != "PR_3" {
		t.Errorf("FetchDetail ids = %v after a refresh, want [PR_3 PR_3]", f.ids)
	}

	m, cmd = m.Update(result(m, domain.Snapshot{}, errors.New("boom")))
	m = runCmd(m, cmd)
	if len(f.ids) != 2 {
		t.Errorf("a failed refresh made a detail call: ids = %v", f.ids)
	}

	// The r key goes through the same path as the tick.
	m, cmd = m.Update(key("r"))
	m = runCmd(m, cmd)
	m = refresh(m, 1, 2, 3, 4, 5)
	if len(f.ids) != 3 {
		t.Errorf("FetchDetail ids = %v after r and a refresh, want 3 calls", f.ids)
	}
}

func TestDetailNoFetchWhenClosed(t *testing.T) {
	f := newDetailFake(fixtureDetail())
	m := openRow(t, liveDetail(t, f, &clock{t: fixedNow()}, nil, 40, 1, 2, 3), 2)
	m = press(m, "esc")
	m = refresh(m, 1, 2, 3)
	m = refresh(m, 1, 2, 3)
	if len(f.ids) != 1 {
		t.Errorf("FetchDetail ids = %v, want only the one from enter", f.ids)
	}
	if mm := m.(Model); mm.detail.open || mm.detailInFlight {
		t.Errorf("open=%v inFlight=%v after esc and refreshes", mm.detail.open, mm.detailInFlight)
	}
}

func TestDetailNoOverlap(t *testing.T) {
	f := newDetailFake(fixtureDetail())
	var m tea.Model = liveDetail(t, f, &clock{t: fixedNow()}, nil, 40, 1, 2, 3)
	m, cmd := m.Update(key("enter"))
	first := cmd() // the request goes out; its result has not come back
	if len(f.ids) != 1 {
		t.Fatalf("ids = %v, want one call", f.ids)
	}
	m = refresh(m, 1, 2, 3)
	if len(f.ids) != 1 {
		t.Errorf("a refresh started a second detail fetch while one was in flight: ids = %v", f.ids)
	}
	m, _ = m.Update(first)
	m = refresh(m, 1, 2, 3)
	if len(f.ids) != 2 {
		t.Errorf("ids = %v after the result was applied, want 2 calls", f.ids)
	}
}

func TestDetailPausedNoRequest(t *testing.T) {
	clk := &clock{t: fixedNow()}
	f := newDetailFake(fixtureDetail())
	m := liveDetail(t, f, clk, nil, 40, 1, 2, 3)
	mm := m.(Model)
	mm.pausedUntil = clk.t.Add(10 * time.Minute)
	m = mm

	m, cmd := m.Update(key("enter"))
	if cmd != nil {
		t.Error("enter during a pause started a request")
	}
	if v := plain(m); !strings.Contains(v, "CHECKS  paused until 14:42 (rate limit)") {
		t.Errorf("no pause line in the CHECKS section:\n%s", v)
	}
	m = refresh(m, 1, 2, 3)
	if len(f.ids) != 0 {
		t.Errorf("a refresh during the pause made a detail call: %v", f.ids)
	}
	clk.t = clk.t.Add(11 * time.Minute)
	m = refresh(m, 1, 2, 3)
	if len(f.ids) != 1 {
		t.Errorf("ids = %v after the pause, want one call", f.ids)
	}
	if v := plain(m); !strings.Contains(v, "CHECKS  2 failing") {
		t.Errorf("the detail was not loaded after the pause:\n%s", v)
	}
}

func TestDetailRateLimitExtendsPause(t *testing.T) {
	clk := &clock{t: fixedNow()}
	open := func(err error, pause time.Duration) (tea.Model, *detailFake) {
		f := newDetailFake(fixtureDetail())
		f.err = err
		var m tea.Model = liveDetail(t, f, clk, nil, 40, 1, 2, 3)
		m, cmd := m.Update(key("enter"))
		if pause != 0 { // a pause that is already running
			mm := m.(Model)
			mm.pausedUntil = clk.t.Add(pause)
			m = mm
		}
		return runCmd(m, cmd), f
	}

	t.Run("a reset time", func(t *testing.T) {
		reset := clk.t.Add(10 * time.Minute)
		m, f := open(&limitedErr{reset: reset, limited: true}, 0)
		if got, want := m.(Model).pausedUntil, reset.Add(time.Second); !got.Equal(want) {
			t.Fatalf("pausedUntil = %v, want %v", got, want)
		}
		m, cmd := m.Update(key("r"))
		m = runCmd(m, cmd)
		if mm := m.(Model); mm.inFlight || f.fakeFetcher.calls != 0 {
			t.Errorf("r during the pause started a request: inFlight=%v calls=%d", mm.inFlight, f.fakeFetcher.calls)
		}
		if v := plain(m); !strings.Contains(v, "rate limited — resumes 14:42") {
			t.Errorf("r shows no pause flash:\n%s", v)
		}
	})
	t.Run("Retry-After", func(t *testing.T) {
		m, _ := open(&limitedErr{after: 5 * time.Minute, limited: true}, 0)
		if got, want := m.(Model).pausedUntil, clk.t.Add(5*time.Minute); !got.Equal(want) {
			t.Errorf("pausedUntil = %v, want %v", got, want)
		}
	})
	t.Run("a huge Retry-After is cut to 65 min", func(t *testing.T) {
		m, _ := open(&limitedErr{after: 2147483647 * time.Second, limited: true}, 0)
		if got, want := m.(Model).pausedUntil, clk.t.Add(domain.MaxServerWait); !got.Equal(want) {
			t.Errorf("pausedUntil = %v, want %v", got, want)
		}
	})
	t.Run("a reset ten years ahead is cut to 65 min", func(t *testing.T) {
		m, _ := open(&limitedErr{reset: clk.t.AddDate(10, 0, 0), limited: true}, 0)
		if got, want := m.(Model).pausedUntil, clk.t.Add(domain.MaxServerWait); !got.Equal(want) {
			t.Errorf("pausedUntil = %v, want %v", got, want)
		}
	})
	t.Run("a reset in the past pauses for the floor", func(t *testing.T) {
		m, f := open(&limitedErr{reset: clk.t.Add(-time.Hour), limited: true}, 0)
		if got, want := m.(Model).pausedUntil, clk.t.Add(domain.RateLimitFloor); !got.Equal(want) {
			t.Fatalf("pausedUntil = %v, want %v", got, want)
		}
		m, cmd := m.Update(key("r"))
		m = runCmd(m, cmd)
		if mm := m.(Model); mm.inFlight || f.fakeFetcher.calls != 0 {
			t.Errorf("r during the pause started a request: inFlight=%v calls=%d", mm.inFlight, f.fakeFetcher.calls)
		}
		if v := plain(m); !strings.Contains(v, "rate limited — resumes") {
			t.Errorf("r shows no pause flash:\n%s", v)
		}
	})
	t.Run("a reset equal to now pauses for the floor", func(t *testing.T) {
		m, _ := open(&limitedErr{reset: clk.t, limited: true}, 0)
		if got, want := m.(Model).pausedUntil, clk.t.Add(domain.RateLimitFloor); !got.Equal(want) {
			t.Errorf("pausedUntil = %v, want %v", got, want)
		}
	})
	t.Run("a tiny Retry-After is raised to 1 s", func(t *testing.T) {
		m, _ := open(&limitedErr{after: time.Millisecond, limited: true}, 0)
		if got, want := m.(Model).pausedUntil, clk.t.Add(domain.MinServerWait); !got.Equal(want) {
			t.Errorf("pausedUntil = %v, want %v", got, want)
		}
	})
	t.Run("Retry-After beats a reset time", func(t *testing.T) {
		m, _ := open(&limitedErr{after: 5 * time.Minute, reset: clk.t.Add(10 * time.Minute), limited: true}, 0)
		if got, want := m.(Model).pausedUntil, clk.t.Add(5*time.Minute); !got.Equal(want) {
			t.Errorf("pausedUntil = %v, want %v", got, want)
		}
	})
	t.Run("never earlier", func(t *testing.T) {
		m, _ := open(&limitedErr{reset: clk.t.Add(10 * time.Minute), limited: true}, 20*time.Minute)
		if got, want := m.(Model).pausedUntil, clk.t.Add(20*time.Minute); !got.Equal(want) {
			t.Errorf("pausedUntil = %v, want the later %v", got, want)
		}
	})
	t.Run("another error pauses nothing", func(t *testing.T) {
		m, _ := open(&limitedErr{reset: clk.t.Add(10 * time.Minute)}, 0)
		if got := m.(Model).pausedUntil; !got.IsZero() {
			t.Errorf("pausedUntil = %v for an error that is no rate limit", got)
		}
	})
	t.Run("a refresh result does not end the pause", func(t *testing.T) {
		reset := clk.t.Add(10 * time.Minute)
		m, f := open(&limitedErr{reset: reset, limited: true}, 0)
		m = refresh(m, 1, 2, 3)
		if got, want := m.(Model).pausedUntil, reset.Add(time.Second); !got.Equal(want) {
			t.Errorf("pausedUntil = %v after a list result, want %v", got, want)
		}
		if len(f.ids) != 1 {
			t.Errorf("ids = %v: a request went out inside the pause", f.ids)
		}
	})
}

func TestDetailErrorKeepsData(t *testing.T) {
	clk := &clock{t: fixedNow()}
	f := newDetailFake(fixtureDetail())
	m := openRow(t, liveDetail(t, f, clk, nil, 40, 1, 2, 3), 1)
	if v := plain(m); strings.Contains(v, "refresh failed") {
		t.Fatalf("a fresh detail shows a failure:\n%s", v)
	}

	clk.t = time.Date(2026, 10, 11, 14, 33, 10, 0, time.UTC)
	f.err = &shortErr{short: "network"}
	m = refresh(m, 1, 2, 3)
	v := plain(m)
	if want := "✗ refresh failed 14:33 (network) · details from 14:32:05"; !strings.Contains(v, want) {
		t.Errorf("view missing %q:\n%s", want, v)
	}
	if !strings.Contains(v, "✗ lint") || !strings.Contains(v, "CHECKS  2 failing") {
		t.Errorf("the last data left the screen:\n%s", v)
	}
	if strings.Contains(v, "ghp_SECRET") {
		t.Errorf("the error message reached the screen:\n%s", v)
	}
	ls := strings.Split(v, "\n")
	if len(ls) < 3 || !strings.HasPrefix(ls[2], "✗ refresh failed") {
		t.Errorf("the failure line should sit right under the title (line 3):\n%s", v)
	}

	clk.t = time.Date(2026, 10, 11, 14, 34, 0, 0, time.UTC)
	f.err = nil
	m = refresh(m, 1, 2, 3)
	if v := plain(m); strings.Contains(v, "refresh failed") || strings.Contains(v, "details from") {
		t.Errorf("a successful detail result left the failure line:\n%s", v)
	}
}

func TestDetailKeepsScrollAndExpand(t *testing.T) {
	f := newDetailFake(bigDetail())
	m := openRow(t, liveDetail(t, f, &clock{t: fixedNow()}, nil, 10, 1, 2, 3), 1)
	m = press(m, "e", "j", "j", "j")
	before := m.(Model)
	if before.detail.vp.YOffset() != 3 || !before.detail.expanded {
		t.Fatalf("setup: offset=%d expanded=%v", before.detail.vp.YOffset(), before.detail.expanded)
	}

	d := bigDetail()
	d.Checks = append(d.Checks, domain.Check{Name: "pass-11", Source: "ci.yml", State: domain.CheckPassing, Word: "success"})
	domain.SortChecks(d.Checks)
	d.Counts, d.Read, d.Total = domain.CountChecks(d.Checks), len(d.Checks), len(d.Checks)
	f.detail = d
	m = refresh(m, 1, 2, 3)

	after := m.(Model)
	if got := after.detail.vp.YOffset(); got != 3 {
		t.Errorf("offset = %d after the refresh, want 3", got)
	}
	if !after.detail.expanded {
		t.Error("the refresh collapsed the checks")
	}
	if got, want := after.detail.vp.TotalLineCount(), before.detail.vp.TotalLineCount()+1; got != want {
		t.Errorf("content lines = %d, want %d (one more check)", got, want)
	}

	t.Run("a shorter content clamps the offset", func(t *testing.T) {
		f.detail = fixtureDetail()
		m := refresh(m, 1, 2, 3)
		mm := m.(Model)
		if mm.detail.vp.YOffset() > max(0, mm.detail.vp.TotalLineCount()-mm.detail.vp.Height()) {
			t.Errorf("offset %d is past the end of %d lines", mm.detail.vp.YOffset(), mm.detail.vp.TotalLineCount())
		}
	})
}

func TestDetailGone(t *testing.T) {
	setup := func(t *testing.T) (tea.Model, *detailFake) {
		f := newDetailFake(fixtureDetail())
		return openRow(t, liveDetail(t, f, &clock{t: fixedNow()}, nil, 40, 1, 2, 3, 4, 5), 3), f
	}

	t.Run("the view stays open with its last data and a note", func(t *testing.T) {
		m, f := setup(t)
		m = refresh(m, 1, 2, 4, 5)
		v := plain(m)
		if !m.(Model).detail.open {
			t.Fatal("the detail view closed")
		}
		for _, want := range []string{"no longer in list", "a/b#3", "✗ lint", "✗ unit"} {
			if !strings.Contains(v, want) {
				t.Errorf("view missing %q:\n%s", want, v)
			}
		}
		ls := strings.Split(v, "\n")
		if len(ls) < 3 || ls[2] != "no longer in list" {
			t.Errorf("the note should be line 3, right under the title:\n%s", v)
		}
		m = refresh(m, 1, 2, 4, 5)
		if len(f.ids) != 1 {
			t.Errorf("ids = %v: a gone PR is still fetched", f.ids)
		}
		if m.(Model).detailInFlight {
			t.Error("a detail fetch is in flight for a gone PR")
		}
	})
	t.Run("esc lands on the nearest neighbour below", func(t *testing.T) {
		m, _ := setup(t)
		m = refresh(m, 1, 2, 4, 5)
		m = press(m, "esc")
		if got := markedRef(t, m); got != "a/b#4" {
			t.Errorf("marked row = %q, want a/b#4", got)
		}
	})
	t.Run("the last row falls back to the one above", func(t *testing.T) {
		f := newDetailFake(fixtureDetail())
		m := openRow(t, liveDetail(t, f, &clock{t: fixedNow()}, nil, 40, 1, 2, 3), 3)
		m = refresh(m, 1, 2)
		m = press(m, "esc")
		if got := markedRef(t, m); got != "a/b#2" {
			t.Errorf("marked row = %q, want a/b#2", got)
		}
	})
	t.Run("a PR that comes back resumes fetching and is selected again", func(t *testing.T) {
		m, f := setup(t)
		m = refresh(m, 1, 2, 4, 5)
		m = refresh(m, 1, 2, 3, 4, 5)
		if v := plain(m); strings.Contains(v, "no longer in list") {
			t.Errorf("the note stayed after the PR came back:\n%s", v)
		}
		if len(f.ids) != 2 {
			t.Errorf("ids = %v, want a second call after the PR came back", f.ids)
		}
		m = press(m, "esc")
		if got := markedRef(t, m); got != "a/b#3" {
			t.Errorf("marked row = %q, want a/b#3", got)
		}
	})
}

func TestDetailGoneOpensDetailURL(t *testing.T) {
	op := &fakeOpener{}
	f := newDetailFake(fixtureDetail())
	m := openRow(t, liveDetail(t, f, &clock{t: fixedNow()}, op, 40, 1, 2, 3, 4, 5), 3)
	m = refresh(m, 1, 2, 4, 5)
	if got := m.(Model).rows[m.(Model).cursor].Ref.Number; got != 4 {
		t.Fatalf("setup: the cursor row is #%d, want #4", got)
	}
	_, cmd := m.Update(key("o"))
	if cmd == nil {
		t.Fatal("o returned no command")
	}
	cmd()
	if len(op.urls) != 1 || op.urls[0] != "https://github.com/a/b/pull/3" {
		t.Errorf("opened %v, want the URL of #3 and not of the row under the cursor", op.urls)
	}
}

func TestDetailEscAfterReorder(t *testing.T) {
	f := newDetailFake(fixtureDetail())
	m := openRow(t, liveDetail(t, f, &clock{t: fixedNow()}, nil, 7, 1, 2, 3, 4, 5), 2)
	m = refresh(m, 5, 4, 3, 2, 1)
	m = refresh(m, 3, 5, 1, 4, 2)
	m = refresh(m, 5, 4, 3, 2, 1)
	if v := plain(m); strings.Contains(v, "no longer in list") {
		t.Fatalf("a reorder was taken for a PR leaving the list:\n%s", v)
	}
	m = press(m, "esc")
	if got := markedRef(t, m); got != "a/b#2" {
		t.Errorf("marked row = %q, want a/b#2 marked and on screen", got)
	}
}

func TestDetailStaleAfterSwitch(t *testing.T) {
	f := newDetailFake(fixtureDetail())
	var m tea.Model = liveDetail(t, f, &clock{t: fixedNow()}, nil, 40, 1, 2, 3)
	m, _ = m.Update(key("enter")) // PR #1, sequence A
	seqA := m.(Model).detailSeq
	refA := m.(Model).detail.ref
	m = press(m, "esc", "j", "enter") // PR #2, sequence B
	mm := m.(Model)
	if mm.detailSeq == seqA || mm.detail.ref == refA {
		t.Fatalf("setup: seq %d/%d ref %v/%v", seqA, mm.detailSeq, refA, mm.detail.ref)
	}
	m, _ = m.Update(detailMsg{seq: seqA, ref: refA, data: fixtureDetail()})
	if m.(Model).detail.data != nil {
		t.Error("the result of the first view was applied to the second")
	}
	m, _ = m.Update(detailMsg{seq: mm.detailSeq, ref: refA, data: fixtureDetail()})
	if m.(Model).detail.data != nil {
		t.Error("a result for another PR was applied")
	}
	if !m.(Model).detailInFlight {
		t.Error("a dropped result cleared detailInFlight")
	}
	m, _ = m.Update(detailMsg{seq: mm.detailSeq, ref: mm.detail.ref, data: fixtureDetail()})
	if m.(Model).detail.data == nil {
		t.Error("the current result was dropped")
	}
}

// ---- narrow panes, resizes and tiny heights (plan 03-08 task 2) ----

// narrowDetail is fixtureDetail with two long names added: a check whose name
// does not fit 36 cells, and one whose line only fits without its source.
func narrowDetail() domain.Detail {
	d := fixtureDetail()
	d.Checks = append(d.Checks,
		domain.Check{Name: "a-very-long-check-name-that-never-ends-ok", Source: "ci-very-long-workflow.yml", State: domain.CheckFailing, Word: "timed out"},
	)
	domain.SortChecks(d.Checks)
	d.Counts, d.Read, d.Total = domain.CountChecks(d.Checks), len(d.Checks), len(d.Checks)
	return d
}

func TestDetailNarrow(t *testing.T) {
	open := func(w int) []string {
		m := press(runDetailOpen(t, detailModel(t, newDetailFake(narrowDetail()), Options{}, w, 40)), "e")
		return strings.Split(plain(m), "\n")
	}
	lineWithText := func(ls []string, text string) string {
		for _, l := range ls {
			if strings.Contains(l, text) {
				return l
			}
		}
		return ""
	}

	t.Run("36 cells keep only the glyph and the name", func(t *testing.T) {
		ls := open(36)
		for _, want := range []string{
			"  ✗ lint", "  ✗ unit", "  ⊘ integration", "  ● e2e", "  ✓ build", "  – preview",
			"  ✗ bob", "  … carol", "  ✓ alice", "  · dave", "  – erin", "  … 2 teams (names hidden)",
		} {
			found := false
			for _, l := range ls {
				if l == want {
					found = true
				}
			}
			if !found {
				t.Errorf("no line is exactly %q:\n%s", want, strings.Join(ls, "\n"))
			}
		}
		for _, word := range []string{"timed out", "failure", "ci.yml", "changes requested", "approved", "pending", "commented"} {
			for _, l := range ls { // the indented lines are the checks and the reviewers
				if strings.HasPrefix(l, "  ") && strings.Contains(l, word) {
					t.Errorf("the word %q is still drawn: %q", word, l)
				}
			}
		}
		long := lineWithText(ls, "a-very-long")
		if !strings.HasPrefix(long, "  ✗ a-very-long") || !strings.HasSuffix(long, "…") || lipgloss.Width(long) > 36 {
			t.Errorf("the long name should be cut with an ellipsis within 36 cells: %q", long)
		}
		for _, l := range ls {
			if lipgloss.Width(l) > 36 {
				t.Errorf("line is %d cells wide: %q", lipgloss.Width(l), l)
			}
		}
		if sum := lineWithText(ls, "CHECKS"); lipgloss.Width(sum) > 36 || !strings.HasPrefix(sum, "CHECKS  3 failing") {
			t.Errorf("the summary should be cut to 36 cells: %q", sum)
		}
	})
	t.Run("60 cells drop the source before the word", func(t *testing.T) {
		ls := open(60)
		long := lineWithText(ls, "a-very-long")
		if !strings.Contains(long, "timed out") || strings.Contains(long, "ci-very") {
			t.Errorf("the long line should keep the word and lose the source: %q", long)
		}
		if l := lineWithText(ls, "✗ unit"); strings.Contains(l, "ci.yml") || !strings.Contains(l, "failure") {
			t.Errorf("the source column should go for every line at once: %q", l)
		}
		for _, l := range ls {
			if lipgloss.Width(l) > 60 {
				t.Errorf("line is %d cells wide: %q", lipgloss.Width(l), l)
			}
		}
	})
	t.Run("120 cells keep the source", func(t *testing.T) {
		if l := lineWithText(open(120), "✗ unit"); !strings.Contains(l, "ci.yml") {
			t.Errorf("the source column is gone at 120 cells: %q", l)
		}
	})
}

func TestDetailResize(t *testing.T) {
	m := press(runDetailOpen(t, detailModel(t, newDetailFake(bigDetail()), Options{}, 120, 40)), "e")
	for range 60 {
		m = press(m, "j")
	}
	ref := m.(Model).detail.ref

	valid := func(step string, w, h int) {
		t.Helper()
		ls := strings.Split(plain(m), "\n")
		if len(ls) > h {
			t.Errorf("%s: %d lines, want at most %d", step, len(ls), h)
		}
		for _, l := range ls {
			if lipgloss.Width(l) > w {
				t.Errorf("%s: line is %d cells wide, want at most %d: %q", step, lipgloss.Width(l), w, l)
				break
			}
		}
		if last := ls[len(ls)-1]; !strings.Contains(last, "esc back") {
			t.Errorf("%s: the last line is %q, want the footer", step, last)
		}
		vp := m.(Model).detail.vp
		if limit := max(0, vp.TotalLineCount()-vp.Height()); vp.YOffset() > limit {
			t.Errorf("%s: offset %d is past the end (max %d)", step, vp.YOffset(), limit)
		}
	}
	for _, sz := range [][2]int{{60, 15}, {36, 12}, {120, 40}, {30, 6}} {
		m, _ = m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		for range 60 {
			m = press(m, "j") // to the bottom at this size
		}
		valid(itoa(sz[0])+"x"+itoa(sz[1]), sz[0], sz[1])
		if sz[0] < 40 {
			if v := plain(m); strings.Contains(v, "ci.yml") || strings.Contains(v, "success") {
				t.Errorf("%dx%d: the body was not laid out again at the new width:\n%s", sz[0], sz[1], v)
			}
		}
	}
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	valid("120x40 again", 120, 40)
	if v := plain(m); !strings.Contains(v, "ci.yml") {
		t.Errorf("the source column did not come back at 120 cells:\n%s", v)
	}
	m = press(m, "esc")
	if mm := m.(Model); mm.detail.open || mm.selected != ref {
		t.Errorf("open=%v selected=%v after esc, want the list with %v", mm.detail.open, mm.selected, ref)
	}
}

func TestDetailTinyHeights(t *testing.T) {
	for _, w := range []int{80, 36} {
		for h, want := range map[int][]string{
			1: {"prv · "},
			2: {"prv · ", "esc"},
			3: {"prv · ", "", "esc"}, // header, one body line (the title), footer
		} {
			ls := lines(stateModel(t, "detail", w, h).View().Content)
			if len(ls) != len(want) {
				t.Errorf("w=%d h=%d: %d lines, want %d: %q", w, h, len(ls), len(want), ls)
				continue
			}
			for i, part := range want {
				if !strings.Contains(ls[i], part) {
					t.Errorf("w=%d h=%d: line %d = %q, want it to hold %q", w, h, i, ls[i], part)
				}
			}
		}
	}
}

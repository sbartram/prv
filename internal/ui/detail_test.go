package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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
	const fold = "… +12 more passing"
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
	if strings.Contains(v, "more passing") {
		t.Errorf("expanded view still has the fold line:\n%s", v)
	}

	m = press(m, "e")
	if v := plain(m); !strings.Contains(v, fold) || strings.Contains(v, "pass-01") {
		t.Errorf("the second e should collapse again:\n%s", v)
	}

	t.Run("failing, cancelled and running checks are never folded", func(t *testing.T) {
		d := bigDetail()
		d.Checks = d.Checks[:4]
		if v := plain(open(d)); strings.Contains(v, "more passing") {
			t.Errorf("a fold line with nothing folded:\n%s", v)
		}
	})
	t.Run("an unknown check folds too", func(t *testing.T) {
		d := fixtureDetail()
		d.Checks = append(d.Checks, domain.Check{Name: "mystery", State: domain.CheckOther, Word: "unknown"})
		v := plain(open(d))
		if strings.Contains(v, "mystery") || !strings.Contains(v, "… +5 more passing") {
			t.Errorf("unknown check not folded:\n%s", v)
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

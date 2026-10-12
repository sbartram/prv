package domain

import (
	"slices"
	"testing"
	"time"
)

func TestClassifyCheck(t *testing.T) {
	tests := []struct {
		name                                string
		typename, status, conclusion, state string
		want                                CheckState
		word                                string
	}{
		{"failure", "CheckRun", "COMPLETED", "FAILURE", "", CheckFailing, "failure"},
		{"timed out", "CheckRun", "COMPLETED", "TIMED_OUT", "", CheckFailing, "timed out"},
		{"startup failure", "CheckRun", "COMPLETED", "STARTUP_FAILURE", "", CheckFailing, "startup failure"},
		{"action required", "CheckRun", "COMPLETED", "ACTION_REQUIRED", "", CheckFailing, "action required"},
		{"cancelled", "CheckRun", "COMPLETED", "CANCELLED", "", CheckCancelled, "cancelled"},
		{"success", "CheckRun", "COMPLETED", "SUCCESS", "", CheckPassing, "success"},
		{"skipped", "CheckRun", "COMPLETED", "SKIPPED", "", CheckNeutral, "skipped"},
		{"neutral", "CheckRun", "COMPLETED", "NEUTRAL", "", CheckNeutral, "neutral"},
		{"stale", "CheckRun", "COMPLETED", "STALE", "", CheckNeutral, "stale"},
		{"in progress", "CheckRun", "IN_PROGRESS", "", "", CheckRunning, "running"},
		{"queued", "CheckRun", "QUEUED", "", "", CheckRunning, "queued"},
		{"requested", "CheckRun", "REQUESTED", "", "", CheckRunning, "queued"},
		{"pending", "CheckRun", "PENDING", "", "", CheckRunning, "queued"},
		{"waiting", "CheckRun", "WAITING", "", "", CheckRunning, "waiting"},
		{"context error", "StatusContext", "", "", "ERROR", CheckFailing, "error"},
		{"context failure", "StatusContext", "", "", "FAILURE", CheckFailing, "failure"},
		{"context pending", "StatusContext", "", "", "PENDING", CheckRunning, "pending"},
		{"context expected", "StatusContext", "", "", "EXPECTED", CheckRunning, "expected"},
		{"context success", "StatusContext", "", "", "SUCCESS", CheckPassing, "success"},
		{"unknown status", "CheckRun", "FUTURE_VALUE", "FAILURE", "", CheckOther, "unknown"},
		{"unknown conclusion", "CheckRun", "COMPLETED", "FUTURE_VALUE", "", CheckOther, "unknown"},
		{"completed without conclusion", "CheckRun", "COMPLETED", "", "", CheckOther, "unknown"},
		{"unknown context state", "StatusContext", "", "", "FUTURE_VALUE", CheckOther, "unknown"},
		{"a context ignores check-run fields", "StatusContext", "COMPLETED", "FAILURE", "", CheckOther, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, word := ClassifyCheck(tt.typename, tt.status, tt.conclusion, tt.state)
			if got != tt.want || word != tt.word {
				t.Errorf("ClassifyCheck(%q, %q, %q, %q) = (%v, %q), want (%v, %q)",
					tt.typename, tt.status, tt.conclusion, tt.state, got, word, tt.want, tt.word)
			}
		})
	}
}

func TestLatestChecks(t *testing.T) {
	t8 := time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC)
	t9 := t8.Add(time.Hour)
	chk := func(name, wf string, ctx bool, st CheckState, at time.Time, id int64) Check {
		return Check{Name: name, Workflow: wf, Context: ctx, State: st, SuiteCreatedAt: at, ID: id}
	}
	tests := []struct {
		name string
		in   []Check
		want []Check
	}{
		{"nil", nil, []Check{}},
		{"newer suite wins, failing then passing",
			[]Check{chk("build", "ci", false, CheckFailing, t8, 1), chk("build", "ci", false, CheckPassing, t9, 2)},
			[]Check{chk("build", "ci", false, CheckPassing, t9, 2)}},
		{"newer suite wins whatever the input order",
			[]Check{chk("build", "ci", false, CheckPassing, t9, 2), chk("build", "ci", false, CheckFailing, t8, 1)},
			[]Check{chk("build", "ci", false, CheckPassing, t9, 2)}},
		{"same suite time: larger id wins",
			[]Check{chk("build", "ci", false, CheckFailing, t9, 5), chk("build", "ci", false, CheckPassing, t9, 6)},
			[]Check{chk("build", "ci", false, CheckPassing, t9, 6)}},
		{"full tie: the earlier one stays",
			[]Check{chk("build", "ci", false, CheckFailing, t9, 5), chk("build", "ci", false, CheckPassing, t9, 5)},
			[]Check{chk("build", "ci", false, CheckFailing, t9, 5)}},
		{"same name under two workflows is two checks",
			[]Check{chk("build", "ci", false, CheckFailing, t9, 1), chk("build", "release", false, CheckPassing, t9, 2)},
			[]Check{chk("build", "ci", false, CheckFailing, t9, 1), chk("build", "release", false, CheckPassing, t9, 2)}},
		{"a status context and a check run with one name are two checks",
			[]Check{chk("build", "", true, CheckFailing, time.Time{}, 0), chk("build", "", false, CheckPassing, t9, 2)},
			[]Check{chk("build", "", true, CheckFailing, time.Time{}, 0), chk("build", "", false, CheckPassing, t9, 2)}},
		{"output keeps the order of first appearance",
			[]Check{chk("b", "ci", false, CheckFailing, t8, 1), chk("a", "ci", false, CheckFailing, t9, 2), chk("b", "ci", false, CheckPassing, t9, 3)},
			[]Check{chk("b", "ci", false, CheckPassing, t9, 3), chk("a", "ci", false, CheckFailing, t9, 2)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := slices.Clone(tt.in)
			got := LatestChecks(tt.in)
			if !slices.Equal(got, tt.want) {
				t.Errorf("LatestChecks =\n%+v\nwant\n%+v", got, tt.want)
			}
			if !slices.Equal(tt.in, before) {
				t.Errorf("the input was modified: %+v, was %+v", tt.in, before)
			}
		})
	}
}

func TestSummarizeFailing(t *testing.T) {
	c := func(name string, st CheckState) Check { return Check{Name: name, State: st} }
	tests := []struct {
		name     string
		in       []Check
		complete bool
		want     FailingCheck
	}{
		{"three failing: first by name, two more",
			[]Check{c("lint", CheckFailing), c("test", CheckFailing), c("vet", CheckFailing)}, true,
			FailingCheck{Known: true, Name: "lint", More: 2}},
		{"first by name regardless of input order",
			[]Check{c("vet", CheckFailing), c("test", CheckFailing), c("lint", CheckFailing)}, true,
			FailingCheck{Known: true, Name: "lint", More: 2}},
		{"case-insensitive name order",
			[]Check{c("beta", CheckFailing), c("Alpha", CheckFailing)}, true,
			FailingCheck{Known: true, Name: "Alpha", More: 1}},
		{"one cancelled",
			[]Check{c("deploy", CheckCancelled)}, true,
			FailingCheck{Known: true, Name: "deploy", Cancelled: true}},
		{"two cancelled",
			[]Check{c("deploy", CheckCancelled), c("docs", CheckCancelled)}, true,
			FailingCheck{Known: true, Name: "deploy", Cancelled: true, More: 1}},
		{"a failing check is named before a cancelled one, which is not counted",
			[]Check{c("deploy", CheckCancelled), c("Lint", CheckFailing)}, true,
			FailingCheck{Known: true, Name: "Lint"}},
		{"passing, running and neutral checks are ignored",
			[]Check{c("a", CheckPassing), c("b", CheckRunning), c("c", CheckNeutral), c("d", CheckOther), c("z", CheckFailing)}, true,
			FailingCheck{Known: true, Name: "z"}},
		{"truncated page: no name", []Check{c("lint", CheckFailing)}, false, FailingCheck{}},
		{"nothing failing or cancelled", []Check{c("a", CheckPassing)}, true, FailingCheck{}},
		{"nil", nil, true, FailingCheck{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SummarizeFailing(tt.in, tt.complete); got != tt.want {
				t.Errorf("SummarizeFailing = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestSummarizeLatestRerun runs the two steps the github mapper runs, on the
// shapes the fixtures hold: superseded runs never reach a name or a count.
func TestSummarizeLatestRerun(t *testing.T) {
	t8 := time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC)
	t9 := t8.Add(time.Hour)
	chk := func(name, wf string, ctx bool, st CheckState, at time.Time, id int64) Check {
		return Check{Name: name, Workflow: wf, Context: ctx, State: st, SuiteCreatedAt: at, ID: id}
	}
	tests := []struct {
		name string
		in   []Check
		want FailingCheck
	}{
		{"cancelled older suite, one failure in the newer suite",
			[]Check{
				chk("Linux / test", "Code OSS", false, CheckCancelled, t8, 1), chk("Linux / lint", "Code OSS", false, CheckCancelled, t8, 2),
				chk("Linux / test", "Code OSS", false, CheckFailing, t9, 3), chk("Linux / lint", "Code OSS", false, CheckPassing, t9, 4),
			},
			FailingCheck{Known: true, Name: "Linux / test"}},
		{"a failure fixed by a newer passing run is not named",
			[]Check{chk("deploy", "release", false, CheckFailing, t8, 1), chk("deploy", "release", false, CheckPassing, t9, 2)},
			FailingCheck{}},
		{"cancelled older run, newer run passes: nothing to name",
			[]Check{chk("deploy", "release", false, CheckCancelled, t8, 1), chk("deploy", "release", false, CheckPassing, t9, 2)},
			FailingCheck{}},
		{"a failing status and a failing check run with one name are two failures",
			[]Check{chk("build", "", true, CheckFailing, time.Time{}, 0), chk("build", "ci", false, CheckFailing, t9, 2)},
			FailingCheck{Known: true, Name: "build", More: 1}},
		{"a failing status stays failing next to a passing check run of that name",
			[]Check{chk("build", "", true, CheckFailing, time.Time{}, 0), chk("build", "ci", false, CheckPassing, t9, 2)},
			FailingCheck{Known: true, Name: "build"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SummarizeFailing(LatestChecks(tt.in), true); got != tt.want {
				t.Errorf("SummarizeFailing(LatestChecks(...)) = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCheckGlyph(t *testing.T) {
	tests := []struct {
		state CheckState
		want  string
	}{
		{CheckFailing, "✗"},
		{CheckCancelled, "⊘"},
		{CheckRunning, "●"},
		{CheckPassing, "✓"},
		{CheckNeutral, "–"},
		{CheckOther, "?"},
	}
	for _, tt := range tests {
		if got := tt.state.Glyph(); got != tt.want {
			t.Errorf("CheckState(%d).Glyph() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

// detailList is the mixed list of the SortChecks and CountChecks tests.
func detailList() []Check {
	return []Check{
		{Name: "docs", State: CheckNeutral},
		{Name: "build", State: CheckPassing},
		{Name: "e2e", State: CheckRunning},
		{Name: "deploy", State: CheckCancelled},
		{Name: "Test", State: CheckFailing},
		{Name: "lint", State: CheckFailing},
		{Name: "x", State: CheckOther},
	}
}

func checkNames(checks []Check) []string {
	var out []string
	for _, c := range checks {
		out = append(out, c.Name)
	}
	return out
}

func TestSortChecks(t *testing.T) {
	want := []string{"lint", "Test", "deploy", "e2e", "build", "docs", "x"}
	in := detailList()
	SortChecks(in)
	if got := checkNames(in); !slices.Equal(got, want) {
		t.Errorf("order = %q, want %q", got, want)
	}
	rev := detailList()
	slices.Reverse(rev)
	SortChecks(rev)
	if got := checkNames(rev); !slices.Equal(got, want) {
		t.Errorf("reversed input order = %q, want %q", got, want)
	}

	t.Run("same name sorts by workflow", func(t *testing.T) {
		in := []Check{
			{Name: "build", Workflow: "b", State: CheckPassing},
			{Name: "Build", Workflow: "z", State: CheckPassing},
			{Name: "build", Workflow: "a", State: CheckPassing},
		}
		SortChecks(in)
		var got []string
		for _, c := range in {
			got = append(got, c.Name+"/"+c.Workflow)
		}
		want := []string{"build/a", "build/b", "Build/z"}
		if !slices.Equal(got, want) {
			t.Errorf("order = %q, want %q", got, want)
		}
	})
}

func TestCountChecks(t *testing.T) {
	want := CheckCounts{Failing: 2, Cancelled: 1, Running: 1, Passing: 1, Neutral: 2}
	if got := CountChecks(detailList()); got != want {
		t.Errorf("CountChecks = %+v, want %+v", got, want)
	}
	if got := CountChecks(nil); got != (CheckCounts{}) {
		t.Errorf("CountChecks(nil) = %+v, want zero", got)
	}
}

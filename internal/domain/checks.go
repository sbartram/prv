package domain

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// CheckState is the display class of one check (a check run or a commit
// status). The zero value is CheckOther: an unrecognised value is never
// failing and never passing.
type CheckState int

const (
	CheckOther CheckState = iota
	CheckFailing
	CheckCancelled
	CheckRunning
	CheckPassing
	CheckNeutral
)

// Check is one check run or commit status of a pull request, already mapped
// and sanitised.
type Check struct {
	Name     string // check-run name, or the status context name
	Workflow string // workflow name, else the check suite's app name, else ""
	Source   string // detail view only: the workflow file's base name, else the app name; "" for a status
	Context  bool   // true for a commit StatusContext
	State    CheckState
	Word     string // lowercase display word, e.g. "failure", "timed out"

	SuiteCreatedAt time.Time // when the check suite was created (orders re-runs)
	ID             int64     // check-run databaseId; zero for a status context
}

// ClassifyCheck maps GitHub's check values to a CheckState and a display word.
// typename is "StatusContext" for a commit status (which reads state); any
// other typename is a check run (which reads status, and conclusion once it is
// COMPLETED). The mapping is total: any value it does not know is
// (CheckOther, "unknown"), never failing and never passing.
func ClassifyCheck(typename, status, conclusion, state string) (CheckState, string) {
	if typename == "StatusContext" {
		switch state {
		case "ERROR":
			return CheckFailing, "error"
		case "FAILURE":
			return CheckFailing, "failure"
		case "PENDING":
			return CheckRunning, "pending"
		case "EXPECTED":
			return CheckRunning, "expected"
		case "SUCCESS":
			return CheckPassing, "success"
		}
		return CheckOther, "unknown"
	}
	switch status {
	case "IN_PROGRESS":
		return CheckRunning, "running"
	case "QUEUED", "REQUESTED", "PENDING":
		return CheckRunning, "queued"
	case "WAITING":
		return CheckRunning, "waiting"
	case "COMPLETED":
		switch conclusion {
		case "FAILURE":
			return CheckFailing, "failure"
		case "TIMED_OUT":
			return CheckFailing, "timed out"
		case "STARTUP_FAILURE":
			return CheckFailing, "startup failure"
		case "ACTION_REQUIRED":
			return CheckFailing, "action required"
		case "CANCELLED":
			return CheckCancelled, "cancelled"
		case "SUCCESS":
			return CheckPassing, "success"
		case "SKIPPED":
			return CheckNeutral, "skipped"
		case "NEUTRAL":
			return CheckNeutral, "neutral"
		case "STALE":
			return CheckNeutral, "stale"
		}
	}
	return CheckOther, "unknown"
}

type checkKey struct {
	context  bool
	workflow string
	name     string
}

// LatestChecks keeps one check per (Context, Workflow, Name): the one from the
// newest check suite (SuiteCreatedAt, then the larger ID). A job that failed
// and then passed in a newer run is therefore not failing (D-03). On a full
// tie the earlier check in the input stays. The output keeps the input order
// of each key's first appearance; the input is not modified.
func LatestChecks(checks []Check) []Check {
	idx := make(map[checkKey]int, len(checks))
	out := make([]Check, 0, len(checks))
	for _, c := range checks {
		k := checkKey{c.Context, c.Workflow, c.Name}
		i, ok := idx[k]
		if !ok {
			idx[k] = len(out)
			out = append(out, c)
			continue
		}
		if newer(c, out[i]) {
			out[i] = c
		}
	}
	return out
}

// newer reports whether a was created after b.
func newer(a, b Check) bool {
	if !a.SuiteCreatedAt.Equal(b.SuiteCreatedAt) {
		return a.SuiteCreatedAt.After(b.SuiteCreatedAt)
	}
	return a.ID > b.ID
}

// FailingCheck is the failing check named in a PR's CI cell (LIST-07). The
// zero value means "no name: show the plain badge".
type FailingCheck struct {
	Known     bool
	Name      string
	Cancelled bool // the named check is cancelled, and nothing else fails
	More      int  // how many other checks fail
}

// SummarizeFailing picks the check a failing row names from the de-duplicated
// checks (LatestChecks). complete is false when the contexts page was
// truncated: then no name is known, because a re-run on the missing page may
// have fixed it.
//
// A failing check is named before any cancelled one (D-04): the name is the
// failing check that sorts first by lower-cased name, and More counts the
// other failing checks. Cancelled checks are never added to that count, so the
// row agrees with a detail summary that shows failing and cancelled
// separately. When no check fails, the first cancelled check is named with
// Cancelled set, and More counts the other cancelled checks. Counts come from
// the de-duplicated checks, never from GitHub's per-state counts (D-02). No
// failing or cancelled check gives the zero value.
func SummarizeFailing(latest []Check, complete bool) FailingCheck {
	if !complete {
		return FailingCheck{}
	}
	if name, n := firstNamed(latest, CheckFailing); n > 0 {
		return FailingCheck{Known: true, Name: name, More: n - 1}
	}
	if name, n := firstNamed(latest, CheckCancelled); n > 0 {
		return FailingCheck{Known: true, Name: name, Cancelled: true, More: n - 1}
	}
	return FailingCheck{}
}

// firstNamed returns the first name by nameLess among the checks in state, and
// how many checks are in that state.
func firstNamed(checks []Check, state CheckState) (string, int) {
	var first string
	n := 0
	for _, c := range checks {
		if c.State != state {
			continue
		}
		if n == 0 || nameLess(c.Name, first) {
			first = c.Name
		}
		n++
	}
	return first, n
}

// nameLess orders names case-insensitively, then byte-wise, so the named check
// does not change between refreshes.
func nameLess(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	if la != lb {
		return la < lb
	}
	return a < b
}

// Glyph returns the one-cell glyph of the state.
func (s CheckState) Glyph() string {
	switch s {
	case CheckFailing:
		return "✗"
	case CheckCancelled:
		return "⊘"
	case CheckRunning:
		return "●"
	case CheckPassing:
		return "✓"
	case CheckNeutral:
		return "–"
	}
	return "?"
}

// rank is the detail view order (D-10): failing, cancelled, running, passing,
// then skipped or neutral, then unknown.
func (s CheckState) rank() int {
	switch s {
	case CheckFailing:
		return 0
	case CheckCancelled:
		return 1
	case CheckRunning:
		return 2
	case CheckPassing:
		return 3
	case CheckNeutral:
		return 4
	}
	return 5
}

// CheckCounts is how many checks are in each display class. Neutral holds
// skipped, neutral and stale checks. Other holds the checks of class CheckOther
// (unknown), which are never counted as skipped.
type CheckCounts struct {
	Failing, Cancelled, Running, Passing, Neutral, Other int
}

// CountChecks counts checks by class. Callers pass the de-duplicated checks
// (LatestChecks), so a superseded re-run is never counted (D-03).
func CountChecks(checks []Check) CheckCounts {
	var n CheckCounts
	for _, c := range checks {
		switch c.State {
		case CheckFailing:
			n.Failing++
		case CheckCancelled:
			n.Cancelled++
		case CheckRunning:
			n.Running++
		case CheckPassing:
			n.Passing++
		case CheckNeutral:
			n.Neutral++
		default:
			n.Other++
		}
	}
	return n
}

// SortChecks sorts checks in place into the detail view order (D-10): failing,
// cancelled, running, passing, skipped or neutral, unknown; within a class by
// lower-cased name, then workflow, then name. The same checks in any input
// order give the same output.
func SortChecks(checks []Check) {
	slices.SortStableFunc(checks, func(a, b Check) int {
		if c := cmp.Compare(a.State.rank(), b.State.rank()); c != 0 {
			return c
		}
		if c := cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Workflow, b.Workflow); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})
}

// Detail is the full detail of one pull request.
//
// Checks are de-duplicated (LatestChecks) and sorted (SortChecks). Counts are
// always CountChecks(Checks), the de-duplicated checks, never GitHub's own
// per-state counts, which include superseded runs (D-03).
//
// Truncated means that not every check run was read: paging stopped at the page
// cap or at an empty cursor, a later page failed, or fewer runs were read than
// GitHub reports. Read is the number of check runs read and Total the number
// GitHub reports, so the view can say "read N of M" (D-03, D-08).
//
// Partial means that GitHub returned a GraphQL error next to the data of some
// page, so a check, a reviewer or a review request may be missing. Partial says
// nothing about the run count (G-03-6, WR-05).
type Detail struct {
	Checks    []Check
	Reviewers []Reviewer
	Counts    CheckCounts
	Truncated bool
	Partial   bool
	Read      int
	Total     int
}

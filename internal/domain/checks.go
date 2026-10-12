package domain

import (
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
	Source   string // left empty here; the detail view fills it (plan 03-06)
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
// (CheckOther, "unknown").
func ClassifyCheck(typename, status, conclusion, state string) (CheckState, string) {
	if typename != "StatusContext" {
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
			case "SUCCESS":
				return CheckPassing, "success"
			}
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
// have fixed it. The name is the failing check that sorts first by lower-cased
// name, and More counts the other failing checks, never GitHub's per-state
// counts (D-02). No failing check gives the zero value.
func SummarizeFailing(latest []Check, complete bool) FailingCheck {
	if !complete {
		return FailingCheck{}
	}
	var first *Check
	n := 0
	for i := range latest {
		c := &latest[i]
		if c.State != CheckFailing {
			continue
		}
		n++
		if first == nil || nameLess(c.Name, first.Name) {
			first = c
		}
	}
	if first == nil {
		return FailingCheck{}
	}
	return FailingCheck{Known: true, Name: first.Name, More: n - 1}
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

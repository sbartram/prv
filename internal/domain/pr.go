// Package domain holds prv's pure types and derivations. It does no I/O.
package domain

import (
	"strconv"
	"time"
)

// DefaultClosedDays is the default closed/merged window, in days.
const DefaultClosedDays = 14

// State is the lifecycle state of a pull request.
type State int

// The zero value is StateOpen. An unrecognised remote value is StateUnknown.
const (
	StateOpen State = iota
	StateMerged
	StateClosed
	StateUnknown
)

// String returns the lowercase display name of the state.
func (s State) String() string {
	switch s {
	case StateOpen:
		return "open"
	case StateMerged:
		return "merged"
	case StateClosed:
		return "closed"
	}
	return "unknown"
}

// ParseState maps GitHub's PullRequestState enum to a State. The mapping is
// total: any other value is StateUnknown, never StateOpen.
func ParseState(s string) State {
	switch s {
	case "OPEN":
		return StateOpen
	case "MERGED":
		return StateMerged
	case "CLOSED":
		return StateClosed
	}
	return StateUnknown
}

// Review is the review status shown for a pull request. The zero value is the
// non-favourable ReviewNone.
type Review int

const (
	ReviewNone Review = iota
	ReviewApproved
	ReviewChangesRequested
	ReviewRequired
)

// CI is the check status shown for a pull request. The zero value is the
// non-favourable CINone ("no checks").
type CI int

const (
	CINone CI = iota
	CIPassing
	CIFailing
	CIRunning
	CIUnknown
)

// Ref identifies a pull request by repository and number.
type Ref struct {
	Repo   string // owner/repo
	Number int
}

// String returns "owner/repo#123".
func (r Ref) String() string {
	return r.Repo + "#" + strconv.Itoa(r.Number)
}

// Key returns the ref in the form used to compare refs: owner and repository
// in lower case (GitHub names are case-insensitive), number unchanged. It is the
// key for duplicates, saved entries and removal (D-03). RED stub.
func (r Ref) Key() Ref { return r }

// URL returns the browser URL of the pull request. RED stub.
func (r Ref) URL() string { return "" }

// MaxPRNumber is the largest GraphQL Int, the largest PR number prv accepts.
const MaxPRNumber = 2147483647

// RefError is a pasted PR reference that ParseRef refused. Reason is the text
// shown in the add prompt.
type RefError struct{ Reason string }

func (e *RefError) Error() string { return e.Reason }

// The reasons ParseRef gives (D-04).
const (
	ReasonOtherHost = "only github.com PRs are supported"
	ReasonIssue     = "that is an issue, not a PR"
	ReasonNotRef    = "not a PR reference"
)

// ParseRef reads a pasted PR reference. RED stub.
func ParseRef(s string) (Ref, error) { return Ref{}, &RefError{Reason: ReasonNotRef} }

// ValidRef reports whether r could name a pull request. RED stub.
func ValidRef(r Ref) bool { return false }

// UnavailKind says why a saved PR could not be loaded.
type UnavailKind int

// UnavailNone is the zero value: the PR loaded.
const (
	UnavailNone UnavailKind = iota
	UnavailNotFound
	UnavailSSO
	UnavailOther
)

// Unavail describes a saved PR that could not be loaded. Reason is the short
// text for rows and the add prompt (D-10); Help is the longer text for the
// detail view.
type Unavail struct {
	Kind   UnavailKind
	Reason string
	Help   string
}

// NotFound is the Unavail of a PR that GitHub reports as not found.
func NotFound() Unavail {
	return Unavail{Kind: UnavailNotFound, Reason: "not found (deleted, renamed, or no access)"}
}

// LoadFailed is the Unavail of a PR that failed to load for another reason.
func LoadFailed(short, help string) Unavail {
	return Unavail{Kind: UnavailOther, Reason: "could not load (" + short + ")", Help: help}
}

// PR is one watched pull request, already mapped and sanitised.
type PR struct {
	ID      string
	Ref     Ref
	Title   string
	URL     string
	Author  string
	IsDraft bool
	State   State
	Review  Review
	CI      CI
	// Failing is the failing check named in the CI cell (LIST-07); the zero
	// value shows the plain badge.
	Failing FailingCheck
	// Merge is the merge-readiness (LIST-06); the zero value is a blank cell.
	Merge Merge
	// Reviewers are the people and teams on the PR in D-06 order (LIST-04).
	Reviewers []Reviewer

	// Auto is true when discovery found the PR (the viewer authored it or is
	// assigned to it). Manual is true when watchlist.json holds it. A PR can be
	// both (WATCH-03).
	Auto, Manual bool
	// Watched is the saved entry's ref, set when Manual. It can differ from Ref
	// after a repository rename; removal goes by it.
	Watched Ref
	// Unavail is set when a saved PR could not be loaded; the zero value means
	// the PR loaded.
	Unavail Unavail

	CreatedAt time.Time
	UpdatedAt time.Time
	ClosedAt  time.Time
}

// RateLimit is the GraphQL rate limit data returned with a query.
type RateLimit struct {
	Limit     int
	Cost      int
	Remaining int
	ResetAt   time.Time
}

// Snapshot is the result of one fetch.
type Snapshot struct {
	Viewer    string
	PRs       []PR
	Truncated bool
	Warnings  []string
	RateLimit RateLimit
}

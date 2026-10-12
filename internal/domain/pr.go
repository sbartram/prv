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

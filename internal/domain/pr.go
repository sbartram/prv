// Package domain holds prv's pure types and derivations. It does no I/O.
package domain

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
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
// key for duplicates, saved entries and removal (D-03).
func (r Ref) Key() Ref { return Ref{Repo: strings.ToLower(r.Repo), Number: r.Number} }

// URL returns the browser URL of the pull request.
func (r Ref) URL() string { return "https://github.com/" + r.Repo + "/pull/" + strconv.Itoa(r.Number) }

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

// GitHub names: an owner has at most 39 characters and no leading or trailing
// hyphen; a repository name is up to 100 characters of letters, digits and . _ -.
// Every pattern is anchored, one per accepted form (D-04); RE2 runs in linear
// time, so a long paste cannot stall the UI.
const (
	ownerPat = `[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?`
	repoPat  = `[A-Za-z0-9._-]{1,100}`
	// ghPrefix is an optional scheme, an optional www. and the host github.com.
	ghPrefix = `^(?i:https?://)?(?i:www\.)?(?i:github\.com)/`
)

var (
	prURLRe = regexp.MustCompile(ghPrefix + `(` + ownerPat + `)/(` + repoPat + `)/pull/([0-9]{1,10})` +
		`(?:/(?:files|checks|commits(?:/[0-9a-fA-F]{4,40})?))?/?(?:[?#].*)?$`)
	shortRe     = regexp.MustCompile(`^(` + ownerPat + `)/(` + repoPat + `)#([0-9]{1,10})$`)
	issueRe     = regexp.MustCompile(ghPrefix + ownerPat + `/` + repoPat + `/issues/[0-9]+(?:[/?#].*)?$`)
	otherHostRe = regexp.MustCompile(`^(?:[A-Za-z][A-Za-z0-9+.-]*://)?([^/\s]+)/[^/\s]+/[^/\s]+/pull/[0-9]+`)
	ownerRe     = regexp.MustCompile(`^` + ownerPat + `$`)
	repoRe      = regexp.MustCompile(`^` + repoPat + `$`)
)

// ParseRef reads one pasted PR reference: a github.com PR URL (scheme and www.
// optional; /files, /checks, /commits or /commits/SHA, a query or a fragment may
// follow the number) or OWNER/REPO#N. Only the owner, repository and number are
// kept; the owner and repository keep the letter case that was typed. Every
// failure is a *RefError whose Reason says why (D-04).
func ParseRef(s string) (Ref, error) {
	s = strings.TrimSpace(s)
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return Ref{}, &RefError{Reason: ReasonNotRef}
		}
	}
	if issueRe.MatchString(s) {
		return Ref{}, &RefError{Reason: ReasonIssue}
	}
	m := prURLRe.FindStringSubmatch(s)
	if m == nil {
		m = shortRe.FindStringSubmatch(s)
	}
	if m == nil {
		if h := otherHostRe.FindStringSubmatch(s); h != nil &&
			!strings.EqualFold(h[1], "github.com") && !strings.EqualFold(h[1], "www.github.com") {
			return Ref{}, &RefError{Reason: ReasonOtherHost}
		}
		return Ref{}, &RefError{Reason: ReasonNotRef}
	}
	n, err := strconv.Atoi(m[3])
	if err != nil || n < 1 || n > MaxPRNumber || m[2] == "." || m[2] == ".." {
		return Ref{}, &RefError{Reason: ReasonNotRef}
	}
	return Ref{Repo: m[1] + "/" + m[2], Number: n}, nil
}

// ValidRef reports whether r could name a pull request: OWNER/REPO with GitHub's
// character rules and a number from 1 to MaxPRNumber. It re-checks refs that
// come from a file a person may have edited.
func ValidRef(r Ref) bool {
	owner, repo, ok := strings.Cut(r.Repo, "/")
	return ok && ownerRe.MatchString(owner) && repoRe.MatchString(repo) && repo != "." && repo != ".." &&
		r.Number >= 1 && r.Number <= MaxPRNumber
}

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

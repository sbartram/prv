package domain

import (
	"cmp"
	"slices"
	"strings"
)

// ReviewerState is one reviewer's state on a pull request. The zero value is
// ReviewerPending, the least favourable known state.
type ReviewerState int

const (
	ReviewerPending ReviewerState = iota
	ReviewerChangesRequested
	ReviewerApproved
	ReviewerCommented // detail view only
	ReviewerDismissed // detail view only
)

// Glyph returns the one-cell glyph of the state.
func (s ReviewerState) Glyph() string {
	switch s {
	case ReviewerChangesRequested:
		return "✗"
	case ReviewerApproved:
		return "✓"
	case ReviewerCommented:
		return "·"
	case ReviewerDismissed:
		return "–"
	}
	return "…"
}

// Word returns the display word of the state.
func (s ReviewerState) Word() string {
	switch s {
	case ReviewerChangesRequested:
		return "changes requested"
	case ReviewerApproved:
		return "approved"
	case ReviewerCommented:
		return "commented"
	case ReviewerDismissed:
		return "dismissed"
	}
	return "pending"
}

// rank is the display order (D-06): changes requested, pending, approved, then
// the detail-only commented and dismissed.
func (s ReviewerState) rank() int {
	switch s {
	case ReviewerChangesRequested:
		return 0
	case ReviewerPending:
		return 1
	case ReviewerApproved:
		return 2
	case ReviewerCommented:
		return 3
	}
	return 4
}

// ReviewerKind says what a Reviewer names.
type ReviewerKind int

const (
	ReviewerPerson ReviewerKind = iota // a user, bot or mannequin login
	ReviewerTeam                       // a team request with a visible slug
	// ReviewerHidden is a request whose reviewer GitHub returned as null,
	// normally a team the viewer cannot see (RESEARCH F6).
	ReviewerHidden
)

// Reviewer is one reviewer of a pull request.
type Reviewer struct {
	Name  string
	Kind  ReviewerKind
	State ReviewerState
}

// ReviewInput is one review. Strings are already cleaned by the github seam.
// Login is "" when the review's author is null.
type ReviewInput struct{ Login, State string }

// RequestInput is one review request. Typename is "" when GitHub returned the
// requested reviewer as null.
type RequestInput struct{ Typename, Name string }

// MergeReviewers merges review requests and reviews into the reviewers of one
// pull request, in display order.
//
// Every request is a pending reviewer. A reviewer who reviewed and was then
// re-requested stays in opinionated, so pending wins (RESEARCH F7); logins are
// compared case-insensitively and the requested spelling is kept. A request
// whose reviewer is null is a hidden pending reviewer and is never dropped. A
// review whose author is null is skipped. opinionated holds only approvals and
// change requests (list rows); latest adds commented and dismissed reviewers
// for the detail view and never overrides a reviewer already present.
func MergeReviewers(opinionated []ReviewInput, requested []RequestInput, latest []ReviewInput) []Reviewer {
	var out []Reviewer
	seen := map[string]bool{}
	add := func(r Reviewer) {
		if r.Kind != ReviewerHidden {
			seen[strings.ToLower(r.Name)] = true
		}
		out = append(out, r)
	}
	for _, q := range requested {
		kind, name := ReviewerPerson, q.Name
		switch q.Typename {
		case "":
			kind = ReviewerHidden
		case "Team", "EnterpriseTeam":
			kind = ReviewerTeam
		default:
			name = strings.TrimSuffix(name, "[bot]")
		}
		if name == "" {
			kind = ReviewerHidden
		}
		if kind != ReviewerHidden && seen[strings.ToLower(name)] {
			continue
		}
		if kind == ReviewerHidden {
			name = ""
		}
		add(Reviewer{Name: name, Kind: kind, State: ReviewerPending})
	}
	addReviews := func(in []ReviewInput, states ...string) {
		for _, r := range in {
			name := strings.TrimSuffix(r.Login, "[bot]")
			if name == "" || seen[strings.ToLower(name)] || !slices.Contains(states, r.State) {
				continue
			}
			add(Reviewer{Name: name, State: reviewState(r.State)})
		}
	}
	addReviews(opinionated, "APPROVED", "CHANGES_REQUESTED")
	addReviews(latest, "APPROVED", "CHANGES_REQUESTED", "COMMENTED", "DISMISSED")

	slices.SortStableFunc(out, func(a, b Reviewer) int {
		if c := cmp.Compare(a.State.rank(), b.State.rank()); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Kind, b.Kind); c != 0 {
			return c
		}
		if c := cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return out
}

// reviewState maps the review states MergeReviewers accepts.
func reviewState(s string) ReviewerState {
	switch s {
	case "APPROVED":
		return ReviewerApproved
	case "CHANGES_REQUESTED":
		return ReviewerChangesRequested
	case "COMMENTED":
		return ReviewerCommented
	}
	return ReviewerDismissed
}

package domain

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// InWindow reports whether a PR belongs in the list. Open and unknown-state PRs
// are always kept, and so is every saved (Manual) PR whatever its state or
// close date: the window only decides which discovery PRs are kept (WATCH-07,
// D-13). A merged or closed PR that is not saved is kept only when it closed at
// or after now minus closedDays days (the boundary is inclusive); one with no
// close time cannot be placed in the window and is dropped.
func InWindow(pr PR, now time.Time, closedDays int) bool {
	if pr.Manual {
		return true
	}
	switch pr.State {
	case StateMerged, StateClosed:
		if pr.ClosedAt.IsZero() {
			return false
		}
		return !pr.ClosedAt.Before(now.Add(-time.Duration(closedDays) * 24 * time.Hour))
	}
	return true
}

// Bucket returns the attention bucket of a PR (D-01); a lower number needs
// attention sooner:
//
//	0 unavailable: a saved PR GitHub could not return (D-11)
//	1 open, failing CI
//	2 open, changes requested
//	3 open, approved and CI passing (ready to merge)
//	4 open, CI running
//	5 every other open PR (this includes approved with no checks or an
//	  unknown CI state: "no checks" never counts as passing)
//	6 open draft
//	7 merged
//	8 closed
//	9 unknown state
//
// Lifecycle state is checked before the draft flag (D-04 as clarified on
// 2026-10-12): only an open draft sorts in bucket 6; a merged or closed draft
// sorts with merged or closed PRs.
func Bucket(pr PR) int {
	if pr.Unavail.Kind != UnavailNone {
		return 0
	}
	switch pr.State {
	case StateMerged:
		return 7
	case StateClosed:
		return 8
	case StateUnknown:
		return 9
	}
	switch {
	case pr.IsDraft:
		return 6
	case pr.CI == CIFailing:
		return 1
	case pr.Review == ReviewChangesRequested:
		return 2
	case pr.Review == ReviewApproved && pr.CI == CIPassing:
		return 3
	case pr.CI == CIRunning:
		return 4
	}
	return 5
}

// Sort orders PRs in place for display: by attention bucket (see Bucket), then
// by creation time newest first, then repository ascending, then PR number
// ascending as an integer (D-02). The last-update time is deliberately not a
// key: comments and pushes would swap rows between refreshes. Equal keys keep
// their input order. Unavailable rows (bucket 0) have no creation time: they go
// by ref, the repository compared without letter case, then the number (D-11).
func Sort(prs []PR) {
	slices.SortStableFunc(prs, func(a, b PR) int {
		ba, bb := Bucket(a), Bucket(b)
		if c := cmp.Compare(ba, bb); c != 0 {
			return c
		}
		if ba == 0 {
			if c := cmp.Compare(strings.ToLower(a.Ref.Repo), strings.ToLower(b.Ref.Repo)); c != 0 {
				return c
			}
			if c := cmp.Compare(a.Ref.Number, b.Ref.Number); c != 0 {
				return c
			}
			return cmp.Compare(a.Ref.Repo, b.Ref.Repo)
		}
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Ref.Repo, b.Ref.Repo); c != 0 {
			return c
		}
		return cmp.Compare(a.Ref.Number, b.Ref.Number)
	})
}

// StateSet is a set of the lifecycle states the user wants to see (D-10). The
// unknown state has no bit: a PR in an unknown state is never filtered out.
type StateSet uint8

// AllStates shows open, merged and closed PRs.
const AllStates = StateSet(1<<StateOpen | 1<<StateMerged | 1<<StateClosed)

// Has reports whether st is in the set.
func (s StateSet) Has(st State) bool { return s&(1<<uint(st)) != 0 }

// Toggle returns the set with st added when it was absent and removed when it
// was present.
func (s StateSet) Toggle(st State) StateSet { return s ^ (1 << uint(st)) }

// Visible returns the PRs the filters let through, in input order, as a new
// slice; prs is not changed. A PR passes when its state is in show and, if
// number is not empty, its PR number starts with the digits in number (so "12"
// keeps #12, #120 and #1234, never #312). A PR in an unknown state is never
// hidden by show (same rule as InWindow): it must not vanish silently. The
// same goes for an unavailable row, whose state is not known (D-13). The number
// filter still applies to both.
func Visible(prs []PR, show StateSet, number string) []PR {
	out := make([]PR, 0, len(prs))
	for _, pr := range prs {
		if pr.State != StateUnknown && pr.Unavail.Kind == UnavailNone && !show.Has(pr.State) {
			continue
		}
		if number != "" && !strings.HasPrefix(strconv.Itoa(pr.Ref.Number), number) {
			continue
		}
		out = append(out, pr)
	}
	return out
}

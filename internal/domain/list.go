package domain

import (
	"cmp"
	"slices"
	"time"
)

// InWindow reports whether a PR belongs in the list. Open and unknown-state PRs
// are always kept. A merged or closed PR is kept only when it closed at or
// after now minus closedDays days (the boundary is inclusive); one with no
// close time cannot be placed in the window and is dropped.
func InWindow(pr PR, now time.Time, closedDays int) bool {
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

func stateRank(s State) int {
	switch s {
	case StateOpen:
		return 0
	case StateMerged:
		return 1
	case StateClosed:
		return 2
	}
	return 3
}

// Sort orders PRs in place for display: Open, Merged, Closed, then unknown;
// within a group the most recently updated first, then repository ascending,
// then PR number ascending as an integer. Equal keys keep their input order.
func Sort(prs []PR) {
	slices.SortStableFunc(prs, func(a, b PR) int {
		if c := cmp.Compare(stateRank(a.State), stateRank(b.State)); c != 0 {
			return c
		}
		if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Ref.Repo, b.Ref.Repo); c != 0 {
			return c
		}
		return cmp.Compare(a.Ref.Number, b.Ref.Number)
	})
}

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

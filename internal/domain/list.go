package domain

import "time"

// InWindow reports whether a PR belongs in the list. Stub: replaced in GREEN.
func InWindow(pr PR, now time.Time, closedDays int) bool { return true }

// Sort orders PRs for display. Stub: replaced in GREEN.
func Sort(prs []PR) {}

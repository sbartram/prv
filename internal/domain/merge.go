package domain

// Merge is the merge-readiness shown for an open pull request. The zero value
// is MergeNone, a blank cell: merged, closed and unknown-state PRs.
type Merge int

const (
	MergeNone Merge = iota
	MergeUnknown
	MergeMergeable
	MergeBlocked
	MergeBehind
	MergeConflicts
)

// DeriveMerge is a RED-phase stub.
func DeriveMerge(st State, draft bool, status, mergeable string) Merge { return MergeNone }

// Badge is a RED-phase stub.
func (m Merge) Badge() string { return "" }

// Glyph is a RED-phase stub.
func (m Merge) Glyph() string { return "" }

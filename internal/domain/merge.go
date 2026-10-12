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

// DeriveMerge returns the merge-readiness of a pull request from its state,
// draft flag and GitHub's mergeStateStatus and mergeable values.
//
// State and draft are decided first (RESEARCH F4): a merged PR reports UNKNOWN,
// a closed PR keeps a stale BLOCKED or DIRTY, and a draft reports BLOCKED or
// DIRTY because the enum has no DRAFT value. Only then is the status read.
// CLEAN, HAS_HOOKS and UNSTABLE are mergeable (D-07: the CI column already
// shows a failing check). UNKNOWN is common and transient (F5); it is a dash,
// except that mergeable CONFLICTING means conflicts. An empty or unrecognised
// status is a dash, never mergeable, so a PR cannot look more ready than
// GitHub's data supports.
func DeriveMerge(st State, draft bool, status, mergeable string) Merge {
	if st != StateOpen {
		return MergeNone
	}
	if draft {
		return MergeUnknown
	}
	switch status {
	case "CLEAN", "HAS_HOOKS", "UNSTABLE":
		return MergeMergeable
	case "BLOCKED":
		return MergeBlocked
	case "BEHIND":
		return MergeBehind
	case "DIRTY":
		return MergeConflicts
	}
	if mergeable == "CONFLICTING" {
		return MergeConflicts
	}
	return MergeUnknown
}

// Badge returns the merge badge: a glyph plus a word, readable without colour.
// MergeUnknown is a bare dash and MergeNone is blank.
func (m Merge) Badge() string {
	switch m {
	case MergeMergeable:
		return "✓ mergeable"
	case MergeBlocked:
		return "⊘ blocked"
	case MergeBehind:
		return "↓ behind"
	case MergeConflicts:
		return "✗ conflicts"
	case MergeUnknown:
		return "–"
	}
	return ""
}

// Glyph returns the one-cell glyph at the start of the badge. Compact rows show
// it alone; TestMergeGlyphPrefix keeps it equal to the badge's first glyph.
// MergeNone is a space so a compact row keeps its columns.
func (m Merge) Glyph() string {
	switch m {
	case MergeMergeable:
		return "✓"
	case MergeBlocked:
		return "⊘"
	case MergeBehind:
		return "↓"
	case MergeConflicts:
		return "✗"
	case MergeUnknown:
		return "–"
	}
	return " "
}

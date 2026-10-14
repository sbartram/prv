package domain

// DeriveReview returns the review status shown for a pull request.
//
// GitHub's reviewDecision wins when it is set. It is null on repositories
// without a required-review rule even when approvals exist, so a null (or
// unknown) decision falls through to the opinionated review states: any
// CHANGES_REQUESTED wins over approvals whatever the order, else any APPROVED
// gives approved, else none. Derivation never yields ReviewRequired, so a null
// decision is never shown as "review required".
func DeriveReview(decision string, opinionated []string) Review {
	switch decision { // "" when reviewDecision is null
	case "APPROVED":
		return ReviewApproved
	case "CHANGES_REQUESTED":
		return ReviewChangesRequested
	case "REVIEW_REQUIRED":
		return ReviewRequired
	}
	var approved bool
	for _, s := range opinionated {
		switch s {
		case "CHANGES_REQUESTED":
			return ReviewChangesRequested
		case "APPROVED":
			approved = true
		}
	}
	if approved {
		return ReviewApproved
	}
	return ReviewNone
}

// ClassifyRollup maps statusCheckRollup.state to a CI value. nil (no checks)
// is CINone, never passing, and an unrecognised value is CIUnknown.
//
// The rollup precedence is GitHub's own: a rollup with a CANCELLED run or a
// FAILURE beside running checks is FAILURE (01-SPIKE.md, extra finding 3).
// prv shows it without recomputing it (D-14). "Cancelled is never red" applies
// to per-check display in Phase 3, not to this list badge.
func ClassifyRollup(state *string) CI {
	if state == nil {
		return CINone
	}
	switch *state {
	case "SUCCESS":
		return CIPassing
	case "FAILURE", "ERROR":
		return CIFailing
	case "PENDING", "EXPECTED":
		return CIRunning
	}
	return CIUnknown
}

// Badge returns the review badge: a glyph plus a word, readable without colour.
// "– none" is the badge for "no review status".
func (r Review) Badge() string {
	switch r {
	case ReviewApproved:
		return "✓ approved"
	case ReviewChangesRequested:
		return "✗ changes"
	case ReviewRequired:
		return "○ required"
	}
	return "– none"
}

// Badge returns the CI badge: a glyph plus a word, readable without colour.
func (c CI) Badge() string {
	switch c {
	case CIPassing:
		return "✓ passing"
	case CIFailing:
		return "✗ failing"
	case CIRunning:
		return "● running"
	case CIUnknown:
		return "? unknown"
	}
	return "– no checks"
}

// Glyph returns the one-cell glyph at the start of the review badge. Compact
// rows show it alone; TestGlyphPrefix keeps it equal to the badge's first
// glyph.
func (r Review) Glyph() string {
	switch r {
	case ReviewApproved:
		return "✓"
	case ReviewChangesRequested:
		return "✗"
	case ReviewRequired:
		return "○"
	}
	return "–"
}

// Glyph returns the one-cell glyph at the start of the CI badge.
func (c CI) Glyph() string {
	switch c {
	case CIPassing:
		return "✓"
	case CIFailing:
		return "✗"
	case CIRunning:
		return "●"
	case CIUnknown:
		return "?"
	}
	return "–"
}

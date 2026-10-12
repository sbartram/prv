package domain

// ReviewerState is one reviewer's state on a pull request.
type ReviewerState int

// The zero value is ReviewerPending, the least favourable known state.
const (
	ReviewerPending ReviewerState = iota
	ReviewerChangesRequested
	ReviewerApproved
	ReviewerCommented
	ReviewerDismissed
)

// ReviewerKind says what a reviewer entry names.
type ReviewerKind int

const (
	ReviewerPerson ReviewerKind = iota
	ReviewerTeam
	ReviewerHidden
)

// Reviewer is one reviewer of a pull request.
type Reviewer struct {
	Name  string
	Kind  ReviewerKind
	State ReviewerState
}

// ReviewInput is one review, already cleaned by the github seam.
type ReviewInput struct{ Login, State string }

// RequestInput is one review request, already cleaned by the github seam.
type RequestInput struct{ Typename, Name string }

// Glyph is a RED-phase stub.
func (s ReviewerState) Glyph() string { return "" }

// Word is a RED-phase stub.
func (s ReviewerState) Word() string { return "" }

// MergeReviewers is a RED-phase stub.
func MergeReviewers(opinionated []ReviewInput, requested []RequestInput, latest []ReviewInput) []Reviewer {
	return nil
}

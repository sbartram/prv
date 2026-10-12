package domain

// Stubs for the RED step: they compile and return the non-favourable zero
// values, so the tests fail on their assertions. Replaced in the GREEN commit.

// DeriveReview is not implemented yet.
func DeriveReview(decision string, opinionated []string) Review { return ReviewNone }

// ClassifyRollup is not implemented yet.
func ClassifyRollup(state *string) CI { return CINone }

// Badge is not implemented yet.
func (r Review) Badge() string { return "" }

// Badge is not implemented yet.
func (c CI) Badge() string { return "" }

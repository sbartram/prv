package domain

// MarkWatched flags the PRs that the saved refs name. RED stub.
func MarkWatched(prs []PR, saved []Ref) []Ref { return saved }

// Combine merges discovery results with looked-up saved PRs. RED stub.
func Combine(auto, looked []PR) []PR { return auto }

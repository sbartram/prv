package domain

// MarkWatched sets Manual and Watched on every PR in prs whose Ref has the same
// Key as a saved ref, and returns the saved refs that no PR matched, in input
// order and without repeats (compared by Key). prs is changed in place, as Sort
// does. A saved PR that discovery already returned costs no Lookup.
func MarkWatched(prs []PR, saved []Ref) []Ref {
	var rest []Ref
	seen := map[Ref]bool{}
	for _, s := range saved {
		k := s.Key()
		if seen[k] {
			continue
		}
		seen[k] = true
		matched := false
		for i := range prs {
			if prs[i].Ref.Key() == k {
				prs[i].Manual = true
				prs[i].Watched = s
				matched = true
			}
		}
		if !matched {
			rest = append(rest, s)
		}
	}
	return rest
}

// Combine merges the discovery results with the looked-up saved PRs into a new
// slice: auto in order, then each looked PR in order. A looked PR whose
// non-empty ID equals the ID of a PR already kept is folded into that PR, which
// becomes Manual with the looked PR's Watched ref, and is not appended: a
// renamed repository is one row (WATCH-03). A looked PR with an empty ID (an
// unavailable one) is always appended. Neither input is changed.
func Combine(auto, looked []PR) []PR {
	out := make([]PR, 0, len(auto)+len(looked))
	out = append(out, auto...)
	byID := map[string]int{}
	for i, pr := range out {
		if pr.ID != "" {
			byID[pr.ID] = i
		}
	}
	for _, pr := range looked {
		if pr.ID != "" {
			if i, ok := byID[pr.ID]; ok {
				out[i].Manual = true
				out[i].Watched = pr.Watched
				continue
			}
			byID[pr.ID] = len(out)
		}
		out = append(out, pr)
	}
	return out
}

// Reconcile is not implemented yet: it returns next and strikes unchanged.
func Reconcile(prev, next []PR, strikes map[Ref]int) ([]PR, map[Ref]int) { return next, strikes }

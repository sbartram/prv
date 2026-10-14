package domain

import "slices"

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

// Reconcile applies the D-12 rule to the rows of a successful refresh and
// returns the rows to show and the new strike counts. It is pure: it returns a
// new slice and a new map and changes neither input.
//
// A row of next that is unavailable with an "other" error (not NOT_FOUND, not
// SSO) is a strike for its saved ref (Watched.Key()). On the first strike a PR
// that prev shows as loaded keeps its last data, so one bad answer does not
// flicker the row; on the second strike in a row, or when there is no last data
// (the first launch), it stays unavailable. NOT_FOUND and SSO are real answers
// and show at once. A row that loads adds no key, so its count starts over, and
// a key whose row is gone is dropped. The rows are sorted again when a row was
// replaced by its last data, because that row's bucket changed; otherwise the
// order of next is kept as it is.
//
// A refresh that fails as a whole (network, 5xx, 401, rate limit) never reaches
// Reconcile: the caller keeps the rows and the counts as they are. Reconcile
// never touches the saved list: a PR that stays unavailable stays saved.
func Reconcile(prev, next []PR, strikes map[Ref]int) ([]PR, map[Ref]int) {
	out := make([]PR, 0, len(next))
	counts := map[Ref]int{}
	replaced := false
	for _, pr := range next {
		if pr.Unavail.Kind != UnavailOther {
			out = append(out, pr)
			continue
		}
		key := pr.Watched.Key()
		n := strikes[key] + 1
		counts[key] = n
		if n < 2 {
			if i := slices.IndexFunc(prev, func(p PR) bool {
				return p.Manual && p.Unavail.Kind == UnavailNone && p.Watched.Key() == key
			}); i >= 0 {
				kept := prev[i]
				kept.Auto, kept.Manual, kept.Watched = false, true, pr.Watched
				out = append(out, kept)
				replaced = true
				continue
			}
		}
		out = append(out, pr)
	}
	if replaced {
		Sort(out)
	}
	return out, counts
}

package domain

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDeriveReview(t *testing.T) {
	tests := []struct {
		name     string
		decision string
		states   []string
		want     Review
	}{
		{"null decision, approved", "", []string{"APPROVED"}, ReviewApproved},
		{"null decision, no reviews is none, never required", "", []string{}, ReviewNone},
		{"null decision, nil reviews", "", nil, ReviewNone},
		{"null decision, changes requested", "", []string{"CHANGES_REQUESTED"}, ReviewChangesRequested},
		{"review required", "REVIEW_REQUIRED", []string{}, ReviewRequired},
		{"approved decision", "APPROVED", []string{"APPROVED"}, ReviewApproved},
		{"changes decision beside approval", "CHANGES_REQUESTED", []string{"CHANGES_REQUESTED", "APPROVED"}, ReviewChangesRequested},
		{"null decision, changes after approval", "", []string{"APPROVED", "CHANGES_REQUESTED"}, ReviewChangesRequested},
		{"null decision, changes before approval", "", []string{"CHANGES_REQUESTED", "APPROVED"}, ReviewChangesRequested},
		{"unknown decision falls through to derivation", "BOGUS", []string{"APPROVED"}, ReviewApproved},
		{"unknown decision, no reviews", "BOGUS", nil, ReviewNone},
		{"approve then comment", "", []string{"APPROVED", "COMMENTED"}, ReviewApproved},
		{"comment then approve", "", []string{"COMMENTED", "APPROVED"}, ReviewApproved},
		{"only a comment", "", []string{"COMMENTED"}, ReviewNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeriveReview(tt.decision, tt.states); got != tt.want {
				t.Errorf("DeriveReview(%q, %v) = %v, want %v", tt.decision, tt.states, got, tt.want)
			}
		})
	}
}

// TestDeriveReviewInvariant: ReviewRequired is returned if and only if the
// decision is REVIEW_REQUIRED, and the result does not depend on the order of
// the opinionated review states.
func TestDeriveReviewInvariant(t *testing.T) {
	decisions := []string{"", "APPROVED", "CHANGES_REQUESTED", "REVIEW_REQUIRED", "BOGUS"}
	alphabet := []string{"APPROVED", "CHANGES_REQUESTED", "COMMENTED", "DISMISSED"}

	lists := [][]string{{}}
	for _, a := range alphabet {
		lists = append(lists, []string{a})
		for _, b := range alphabet {
			lists = append(lists, []string{a, b})
		}
	}
	for _, d := range decisions {
		for _, l := range lists {
			got := DeriveReview(d, l)
			if (got == ReviewRequired) != (d == "REVIEW_REQUIRED") {
				t.Errorf("DeriveReview(%q, %v) = %v: required iff decision is REVIEW_REQUIRED", d, l, got)
			}
			rev := slices.Clone(l)
			slices.Reverse(rev)
			if again := DeriveReview(d, rev); again != got {
				t.Errorf("DeriveReview(%q, %v) = %v but reversed %v = %v", d, l, got, rev, again)
			}
		}
	}
}

func TestClassifyRollup(t *testing.T) {
	s := func(v string) *string { return &v }
	tests := []struct {
		in   *string
		want CI
	}{
		{nil, CINone},
		{s("SUCCESS"), CIPassing},
		{s("FAILURE"), CIFailing},
		{s("ERROR"), CIFailing},
		{s("PENDING"), CIRunning},
		{s("EXPECTED"), CIRunning},
		{s("SOMETHING_NEW"), CIUnknown},
		{s(""), CIUnknown},
		{s("success"), CIUnknown},
	}
	for _, tt := range tests {
		name := "nil"
		if tt.in != nil {
			name = *tt.in
		}
		t.Run(name, func(t *testing.T) {
			if got := ClassifyRollup(tt.in); got != tt.want {
				t.Errorf("ClassifyRollup(%q) = %v, want %v", name, got, tt.want)
			}
		})
	}
}

func TestBadge(t *testing.T) {
	reviews := map[Review]string{
		ReviewApproved:         "✓ approved",
		ReviewChangesRequested: "✗ changes",
		ReviewRequired:         "○ required",
		ReviewNone:             "– none",
	}
	for r, want := range reviews {
		if got := r.Badge(); got != want {
			t.Errorf("Review(%d).Badge() = %q, want %q", r, got, want)
		}
	}
	cis := map[CI]string{
		CIPassing: "✓ passing",
		CIFailing: "✗ failing",
		CIRunning: "● running",
		CINone:    "– no checks",
		CIUnknown: "? unknown",
	}
	for c, want := range cis {
		if got := c.Badge(); got != want {
			t.Errorf("CI(%d).Badge() = %q, want %q", c, got, want)
		}
	}
}

// TestGlyphPrefix keeps the one-cell glyphs of the compact rows from drifting
// away from the badges (RESEARCH Pitfall 11).
func TestGlyphPrefix(t *testing.T) {
	reviews := map[Review]string{ReviewApproved: "✓", ReviewChangesRequested: "✗", ReviewRequired: "○", ReviewNone: "–"}
	for r, want := range reviews {
		if got := r.Glyph(); got != want {
			t.Errorf("Review(%d).Glyph() = %q, want %q", r, got, want)
		}
		if !strings.HasPrefix(r.Badge(), r.Glyph()) {
			t.Errorf("Review(%d).Badge() = %q does not start with Glyph() %q", r, r.Badge(), r.Glyph())
		}
	}
	cis := map[CI]string{CIPassing: "✓", CIFailing: "✗", CIRunning: "●", CINone: "–", CIUnknown: "?"}
	for c, want := range cis {
		if got := c.Glyph(); got != want {
			t.Errorf("CI(%d).Glyph() = %q, want %q", c, got, want)
		}
		if !strings.HasPrefix(c.Badge(), c.Glyph()) {
			t.Errorf("CI(%d).Badge() = %q does not start with Glyph() %q", c, c.Badge(), c.Glyph())
		}
	}
}

func TestClean(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"ansi, bidi and tab", "a\x1b[31mred\x1b[0m‮b\tc", "aredb c"},
		{"bidi isolates", "⁦x⁩", "x"},
		{"osc title sequence", "a\x1b]0;pwned\x07b", "ab"},
		{"c0 and c1 controls", "a\x00b\x7fc\u0085d", "abcd"},
		{"zwj family emoji unchanged", "👨‍👩‍👧", "👨‍👩‍👧"},
		{"plain text unchanged", "Fix token refresh", "Fix token refresh"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Clean(tt.in); got != tt.want {
				t.Errorf("Clean(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestInWindow(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	edge := now.Add(-14 * 24 * time.Hour)
	tests := []struct {
		name string
		pr   PR
		want bool
	}{
		{"open is always kept", PR{State: StateOpen}, true},
		{"open with ancient update is kept", PR{State: StateOpen, UpdatedAt: now.AddDate(-5, 0, 0)}, true},
		{"unknown state is kept", PR{State: StateUnknown}, true},
		{"merged exactly on the boundary is kept", PR{State: StateMerged, ClosedAt: edge}, true},
		{"merged one second earlier is dropped", PR{State: StateMerged, ClosedAt: edge.Add(-time.Second)}, false},
		{"closed exactly on the boundary is kept", PR{State: StateClosed, ClosedAt: edge}, true},
		{"closed one second earlier is dropped", PR{State: StateClosed, ClosedAt: edge.Add(-time.Second)}, false},
		{"closed yesterday is kept", PR{State: StateClosed, ClosedAt: now.Add(-24 * time.Hour)}, true},
		{"merged with zero ClosedAt is dropped", PR{State: StateMerged}, false},
		{"closed with zero ClosedAt is dropped", PR{State: StateClosed}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InWindow(tt.pr, now, 14); got != tt.want {
				t.Errorf("InWindow = %v, want %v", got, tt.want)
			}
		})
	}
	// The window length is a parameter.
	pr := PR{State: StateMerged, ClosedAt: now.Add(-20 * 24 * time.Hour)}
	if InWindow(pr, now, 14) {
		t.Error("20 days old should be outside a 14 day window")
	}
	if !InWindow(pr, now, 30) {
		t.Error("20 days old should be inside a 30 day window")
	}
}

func TestSort(t *testing.T) {
	t0 := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	mk := func(id string, created time.Time, repo string, n int) PR {
		return PR{ID: id, State: StateOpen, CreatedAt: created, Ref: Ref{Repo: repo, Number: n}}
	}
	ids := func(prs []PR) []string {
		var got []string
		for _, p := range prs {
			got = append(got, p.ID)
		}
		return got
	}
	t.Run("bucket order, then createdAt newest first inside a bucket", func(t *testing.T) {
		prs := []PR{
			{ID: "unknown", State: StateUnknown, Ref: Ref{Repo: "a/a", Number: 1}},
			{ID: "closed", State: StateClosed, Ref: Ref{Repo: "a/a", Number: 2}},
			{ID: "merged", State: StateMerged, Ref: Ref{Repo: "a/a", Number: 3}},
			{ID: "draft", State: StateOpen, IsDraft: true, CI: CIFailing, Ref: Ref{Repo: "a/a", Number: 4}},
			{ID: "plain", State: StateOpen, Ref: Ref{Repo: "a/a", Number: 5}},
			{ID: "running", State: StateOpen, CI: CIRunning, Ref: Ref{Repo: "a/a", Number: 6}},
			{ID: "ready", State: StateOpen, Review: ReviewApproved, CI: CIPassing, Ref: Ref{Repo: "a/a", Number: 7}},
			{ID: "changes", State: StateOpen, Review: ReviewChangesRequested, Ref: Ref{Repo: "a/a", Number: 8}},
			{ID: "fail-old", State: StateOpen, CI: CIFailing, CreatedAt: t0, Ref: Ref{Repo: "a/a", Number: 9}},
			{ID: "fail-new", State: StateOpen, CI: CIFailing, CreatedAt: t0.Add(time.Hour), Ref: Ref{Repo: "a/a", Number: 10}},
		}
		Sort(prs)
		want := []string{"fail-new", "fail-old", "changes", "ready", "running", "plain", "draft", "merged", "closed", "unknown"}
		if got := ids(prs); !slices.Equal(got, want) {
			t.Errorf("order = %v\nwant    %v", got, want)
		}
	})
	t.Run("same bucket and createdAt: repo, then number as an integer", func(t *testing.T) {
		prs := []PR{
			mk("b10", t0, "acme/b", 10), mk("a100", t0, "acme/a", 100), mk("a9", t0, "acme/a", 9), mk("a10", t0, "acme/a", 10),
		}
		Sort(prs)
		want := []string{"a9", "a10", "a100", "b10"}
		if got := ids(prs); !slices.Equal(got, want) {
			t.Errorf("order = %v, want %v", got, want)
		}
	})
	t.Run("createdAt compares at full precision", func(t *testing.T) {
		prs := []PR{mk("earlier", t0, "acme/a", 1), mk("later", t0.Add(time.Millisecond), "acme/z", 2)}
		Sort(prs)
		if prs[0].ID != "later" {
			t.Errorf("first = %s, want later (1 ms newer)", prs[0].ID)
		}
	})
	t.Run("permutation independent and blind to updatedAt", func(t *testing.T) {
		base := []PR{
			mk("a9", t0, "acme/a", 9),
			mk("a10", t0, "acme/a", 10),
			mk("b10", t0, "acme/b", 10),
			mk("newer", t0.Add(time.Hour), "acme/z", 1),
			{ID: "fail", State: StateOpen, CI: CIFailing, CreatedAt: t0.Add(-time.Hour), Ref: Ref{Repo: "acme/q", Number: 4}},
			{ID: "draft", State: StateOpen, IsDraft: true, CreatedAt: t0, Ref: Ref{Repo: "acme/a", Number: 1}},
			{ID: "merged", State: StateMerged, CreatedAt: t0, Ref: Ref{Repo: "acme/a", Number: 2}},
			{ID: "closed", State: StateClosed, CreatedAt: t0, Ref: Ref{Repo: "acme/a", Number: 3}},
		}
		perms := [][]int{
			{0, 1, 2, 3, 4, 5, 6, 7},
			{7, 6, 5, 4, 3, 2, 1, 0},
			{3, 1, 7, 0, 5, 2, 6, 4},
			{2, 0, 1, 4, 7, 3, 5, 6},
			{5, 3, 6, 1, 4, 0, 7, 2},
			{1, 2, 0, 6, 3, 7, 4, 5},
		}
		var first []string
		for pi, perm := range perms {
			prs := make([]PR, len(perm))
			for i, k := range perm {
				prs[i] = base[k]
				prs[i].UpdatedAt = t0.Add(time.Duration(i*pi) * time.Minute) // different per permutation
			}
			Sort(prs)
			got := ids(prs)
			if first == nil {
				first = got
				continue
			}
			if !slices.Equal(got, first) {
				t.Errorf("permutation %d = %v\nwant %v", pi, got, first)
			}
		}
		// Changing only updatedAt never moves a row.
		prs := slices.Clone(base)
		Sort(prs)
		before := ids(prs)
		for i := range prs {
			prs[i].UpdatedAt = t0.Add(time.Duration(len(prs)-i) * 24 * time.Hour)
		}
		Sort(prs)
		if got := ids(prs); !slices.Equal(got, before) {
			t.Errorf("after updatedAt changes = %v, want %v", got, before)
		}
	})
	t.Run("stable for fully equal keys", func(t *testing.T) {
		prs := []PR{mk("first", t0, "acme/a", 1), mk("second", t0, "acme/a", 1), mk("third", t0, "acme/a", 1)}
		Sort(prs)
		if got := ids(prs); !slices.Equal(got, []string{"first", "second", "third"}) {
			t.Errorf("equal keys reordered: %v", got)
		}
	})
	t.Run("empty, nil and single do not panic", func(t *testing.T) {
		Sort(nil)
		Sort([]PR{})
		one := []PR{mk("only", t0, "acme/a", 1)}
		Sort(one)
		if len(one) != 1 || one[0].ID != "only" {
			t.Errorf("single PR changed: %v", one)
		}
	})
}

func TestBucket(t *testing.T) {
	tests := []struct {
		name string
		pr   PR
		want int
	}{
		{"D-01 failing CI, approved", PR{State: StateOpen, CI: CIFailing, Review: ReviewApproved}, 1},
		{"D-01 failing CI, changes requested", PR{State: StateOpen, CI: CIFailing, Review: ReviewChangesRequested}, 1},
		{"D-01 changes requested, CI running", PR{State: StateOpen, Review: ReviewChangesRequested, CI: CIRunning}, 2},
		{"D-01 changes requested, CI passing", PR{State: StateOpen, Review: ReviewChangesRequested, CI: CIPassing}, 2},
		{"D-01 approved and passing is ready to merge", PR{State: StateOpen, Review: ReviewApproved, CI: CIPassing}, 3},
		{"D-01 approved, CI running", PR{State: StateOpen, Review: ReviewApproved, CI: CIRunning}, 4},
		{"D-01 review required, CI running", PR{State: StateOpen, Review: ReviewRequired, CI: CIRunning}, 4},
		{"A3 approved with no checks is not ready", PR{State: StateOpen, Review: ReviewApproved, CI: CINone}, 5},
		{"A3 approved with unknown CI is not ready", PR{State: StateOpen, Review: ReviewApproved, CI: CIUnknown}, 5},
		{"D-01 review required, CI passing", PR{State: StateOpen, Review: ReviewRequired, CI: CIPassing}, 5},
		{"D-01 no review, no checks", PR{State: StateOpen, Review: ReviewNone, CI: CINone}, 5},
		{"D-04 open draft with failing CI", PR{State: StateOpen, IsDraft: true, CI: CIFailing}, 6},
		{"D-04 open draft, approved and passing", PR{State: StateOpen, IsDraft: true, Review: ReviewApproved, CI: CIPassing}, 6},
		{"D-01 merged", PR{State: StateMerged}, 7},
		{"D-01 merged with failing CI", PR{State: StateMerged, CI: CIFailing}, 7},
		{"D-04 clarified: merged draft sorts as merged", PR{State: StateMerged, IsDraft: true}, 7},
		{"D-01 closed", PR{State: StateClosed}, 8},
		{"D-04 clarified: closed draft sorts as closed", PR{State: StateClosed, IsDraft: true}, 8},
		{"D-01 unknown state sorts last", PR{State: StateUnknown}, 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Bucket(tt.pr); got != tt.want {
				t.Errorf("Bucket = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStateSet(t *testing.T) {
	for _, st := range []State{StateOpen, StateMerged, StateClosed} {
		if !AllStates.Has(st) {
			t.Errorf("AllStates lacks %v", st)
		}
	}
	if AllStates.Has(StateUnknown) {
		t.Error("AllStates must not claim the unknown state")
	}
	s := AllStates.Toggle(StateMerged)
	if s.Has(StateMerged) || !s.Has(StateOpen) || !s.Has(StateClosed) {
		t.Errorf("Toggle(merged) = %b, want open and closed only", s)
	}
	if got := s.Toggle(StateMerged); got != AllStates {
		t.Errorf("toggling merged twice = %b, want AllStates %b", got, AllStates)
	}
}

func TestVisible(t *testing.T) {
	mk := func(n int, st State) PR { return PR{ID: strconv.Itoa(n), State: st, Ref: Ref{Repo: "a/b", Number: n}} }
	nums := func(prs []PR) []int {
		out := []int{}
		for _, p := range prs {
			out = append(out, p.Ref.Number)
		}
		return out
	}
	states := []PR{mk(1, StateOpen), mk(2, StateMerged), mk(3, StateClosed), mk(4, StateUnknown)}
	tests := []struct {
		name   string
		prs    []PR
		show   StateSet
		number string
		want   []int
	}{
		{"all states keep every PR in order", states, AllStates, "", []int{1, 2, 3, 4}},
		{"open only keeps open and unknown", states, 1 << StateOpen, "", []int{1, 4}},
		{"empty set keeps only unknown", states, 0, "", []int{4}},
		{"prefix 12 is adjacent-safe", []PR{mk(12, 0), mk(120, 0), mk(1234, 0), mk(312, 0), mk(21, 0)}, AllStates, "12", []int{12, 120, 1234}},
		{"leading zero matches nothing", []PR{mk(1, 0), mk(10, 0)}, AllStates, "01", []int{}},
		{"number filter and toggles combine", states, 1 << StateMerged, "2", []int{2}},
		{"nil input gives an empty result", nil, AllStates, "", []int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nums(Visible(tt.prs, tt.show, tt.number)); !slices.Equal(got, tt.want) {
				t.Errorf("Visible = %v, want %v", got, tt.want)
			}
		})
	}
	t.Run("input is not modified", func(t *testing.T) {
		in := slices.Clone(states)
		_ = Visible(in, 0, "9")
		if !slices.EqualFunc(in, states, func(a, b PR) bool { return reflect.DeepEqual(a, b) }) {
			t.Error("Visible changed its input")
		}
	})
}

func TestBucketUnavailable(t *testing.T) {
	gone := Ref{Repo: "acme/gone", Number: 5}
	for _, tt := range []struct {
		name string
		pr   PR
	}{
		{"not found", PR{Ref: gone, Unavail: NotFound()}},
		{"SSO", PR{Ref: gone, Unavail: SSORequired("acme")}},
		{"other error", PR{Ref: gone, Unavail: LoadFailed("rate limited", "")}},
		{"with failing CI", PR{Ref: gone, CI: CIFailing, Unavail: NotFound()}},
		{"with state merged", PR{Ref: gone, State: StateMerged, Unavail: NotFound()}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := Bucket(tt.pr); got != 0 {
				t.Errorf("Bucket = %d, want 0 (D-11)", got)
			}
		})
	}
	if got := Bucket(PR{State: StateOpen, CI: CIFailing}); got != 1 {
		t.Errorf("failing CI without Unavail: Bucket = %d, want 1", got)
	}
}

func TestSortUnavailableByRef(t *testing.T) {
	un := func(repo string, n int) PR {
		return PR{Ref: Ref{Repo: repo, Number: n}, Manual: true, Unavail: NotFound()}
	}
	prs := []PR{
		un("acme/web", 9),
		un("Acme/api", 10),
		un("acme/api", 2),
		{ID: "F", Ref: Ref{Repo: "acme/x", Number: 1}, State: StateOpen, CI: CIFailing},
	}
	Sort(prs)
	var got []string
	for _, p := range prs {
		got = append(got, p.Ref.String())
	}
	want := []string{"acme/api#2", "Acme/api#10", "acme/web#9", "acme/x#1"}
	if !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v (repo compared without letter case, then the number)", got, want)
	}
}

func TestVisibleUnavailable(t *testing.T) {
	un := func(n int) PR { return PR{Ref: Ref{Repo: "acme/gone", Number: n}, Manual: true, Unavail: NotFound()} }
	open := PR{ID: "O", Ref: Ref{Repo: "acme/api", Number: 5}, State: StateOpen}
	prs := []PR{un(5), un(6), open}

	nums := func(prs []PR) []string {
		out := []string{}
		for _, p := range prs {
			out = append(out, p.Ref.String())
		}
		return out
	}
	if got, want := nums(Visible(prs, 1<<StateMerged, "")), []string{"acme/gone#5", "acme/gone#6"}; !slices.Equal(got, want) {
		t.Errorf("only Merged shown: %v, want %v (an unavailable row has no known state, D-13)", got, want)
	}
	if got, want := nums(Visible(prs, 0, "")), []string{"acme/gone#5", "acme/gone#6"}; !slices.Equal(got, want) {
		t.Errorf("no state shown: %v, want %v", got, want)
	}
	if got, want := nums(Visible(prs, 1<<StateMerged, "5")), []string{"acme/gone#5"}; !slices.Equal(got, want) {
		t.Errorf("number 5: %v, want %v (the number filter matches it by number)", got, want)
	}
}

func TestInWindowManual(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	old := now.Add(-60 * 24 * time.Hour)
	tests := []struct {
		name string
		pr   PR
		want bool
	}{
		{"saved merged 60 days ago is kept", PR{State: StateMerged, ClosedAt: old, Manual: true}, true},
		{"saved closed 60 days ago is kept", PR{State: StateClosed, ClosedAt: old, Manual: true}, true},
		{"the same merged PR, not saved, is dropped", PR{State: StateMerged, ClosedAt: old}, false},
		{"saved with a zero ClosedAt is kept", PR{State: StateClosed, Manual: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InWindow(tt.pr, now, 14); got != tt.want {
				t.Errorf("InWindow = %v, want %v", got, tt.want)
			}
		})
	}
}

package domain

import (
	"slices"
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
	mk := func(id string, st State, upd time.Time, repo string, n int) PR {
		return PR{ID: id, State: st, UpdatedAt: upd, Ref: Ref{Repo: repo, Number: n}}
	}
	t.Run("state rank, then updated desc, then repo, then number as int", func(t *testing.T) {
		prs := []PR{
			mk("unknown", StateUnknown, t0.Add(10*time.Hour), "acme/z", 1),
			mk("closed", StateClosed, t0.Add(9*time.Hour), "acme/z", 1),
			mk("merged", StateMerged, t0.Add(8*time.Hour), "acme/z", 1),
			mk("open-old", StateOpen, t0.Add(-time.Hour), "acme/a", 1),
			mk("open-b10", StateOpen, t0, "acme/b", 10),
			mk("open-a10", StateOpen, t0, "acme/a", 10),
			mk("open-a9", StateOpen, t0, "acme/a", 9),
			mk("open-new", StateOpen, t0.Add(time.Hour), "acme/z", 99),
		}
		Sort(prs)
		var got []string
		for _, p := range prs {
			got = append(got, p.ID)
		}
		want := []string{"open-new", "open-a9", "open-a10", "open-b10", "open-old", "merged", "closed", "unknown"}
		if !slices.Equal(got, want) {
			t.Errorf("order = %v\nwant    %v", got, want)
		}
	})
	t.Run("merged ranks before closed even when the closed one is newer", func(t *testing.T) {
		prs := []PR{
			mk("closed-new", StateClosed, t0.Add(48*time.Hour), "acme/a", 1),
			mk("merged-old", StateMerged, t0, "acme/a", 2),
		}
		Sort(prs)
		if prs[0].ID != "merged-old" {
			t.Errorf("first = %s, want merged-old", prs[0].ID)
		}
	})
	t.Run("stable for equal keys", func(t *testing.T) {
		prs := []PR{
			{ID: "first", State: StateOpen, UpdatedAt: t0, Ref: Ref{Repo: "acme/a", Number: 1}},
			{ID: "second", State: StateOpen, UpdatedAt: t0, Ref: Ref{Repo: "acme/a", Number: 1}},
			{ID: "third", State: StateOpen, UpdatedAt: t0, Ref: Ref{Repo: "acme/a", Number: 1}},
		}
		Sort(prs)
		if prs[0].ID != "first" || prs[1].ID != "second" || prs[2].ID != "third" {
			t.Errorf("equal keys reordered: %v %v %v", prs[0].ID, prs[1].ID, prs[2].ID)
		}
	})
	t.Run("empty and nil do not panic", func(t *testing.T) {
		Sort(nil)
		Sort([]PR{})
	})
}

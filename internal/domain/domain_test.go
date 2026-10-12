package domain

import (
	"slices"
	"testing"
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

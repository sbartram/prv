package domain

import (
	"strings"
	"testing"
	"time"
)

func TestDeriveMerge(t *testing.T) {
	tests := []struct {
		name      string
		st        State
		draft     bool
		status    string
		mergeable string
		want      Merge
	}{
		{"clean", StateOpen, false, "CLEAN", "MERGEABLE", MergeMergeable},
		{"has hooks", StateOpen, false, "HAS_HOOKS", "MERGEABLE", MergeMergeable},
		{"unstable is mergeable, CI has its own column", StateOpen, false, "UNSTABLE", "MERGEABLE", MergeMergeable},
		{"blocked", StateOpen, false, "BLOCKED", "MERGEABLE", MergeBlocked},
		{"behind", StateOpen, false, "BEHIND", "MERGEABLE", MergeBehind},
		{"dirty", StateOpen, false, "DIRTY", "CONFLICTING", MergeConflicts},
		{"unknown with conflicting is conflicts", StateOpen, false, "UNKNOWN", "CONFLICTING", MergeConflicts},
		{"unknown with mergeable stays dash", StateOpen, false, "UNKNOWN", "MERGEABLE", MergeUnknown},
		{"unknown with unknown stays dash", StateOpen, false, "UNKNOWN", "UNKNOWN", MergeUnknown},
		{"empty values", StateOpen, false, "", "", MergeUnknown},
		{"unrecognised status is never mergeable", StateOpen, false, "FUTURE_VALUE", "MERGEABLE", MergeUnknown},
		{"draft blocked", StateOpen, true, "BLOCKED", "MERGEABLE", MergeUnknown},
		{"draft dirty", StateOpen, true, "DIRTY", "CONFLICTING", MergeUnknown},
		{"draft clean is still a dash", StateOpen, true, "CLEAN", "MERGEABLE", MergeUnknown},
		{"merged reporting unknown is blank", StateMerged, false, "UNKNOWN", "UNKNOWN", MergeNone},
		{"closed reporting blocked is blank", StateClosed, false, "BLOCKED", "MERGEABLE", MergeNone},
		{"closed reporting dirty is blank", StateClosed, false, "DIRTY", "CONFLICTING", MergeNone},
		{"unknown state is blank", StateUnknown, false, "CLEAN", "MERGEABLE", MergeNone},
		{"merged draft is blank", StateMerged, true, "CLEAN", "MERGEABLE", MergeNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeriveMerge(tt.st, tt.draft, tt.status, tt.mergeable); got != tt.want {
				t.Errorf("DeriveMerge(%v, %v, %q, %q) = %q, want %q",
					tt.st, tt.draft, tt.status, tt.mergeable, got.Badge(), tt.want.Badge())
			}
		})
	}
}

func TestMergeBadgeAndGlyph(t *testing.T) {
	tests := []struct {
		m     Merge
		badge string
		glyph string
	}{
		{MergeMergeable, "✓ mergeable", "✓"},
		{MergeBlocked, "⊘ blocked", "⊘"},
		{MergeBehind, "↓ behind", "↓"},
		{MergeConflicts, "✗ conflicts", "✗"},
		{MergeUnknown, "–", "–"},
		{MergeNone, "", " "},
	}
	for _, tt := range tests {
		if got := tt.m.Badge(); got != tt.badge {
			t.Errorf("Merge(%d).Badge() = %q, want %q", tt.m, got, tt.badge)
		}
		if got := tt.m.Glyph(); got != tt.glyph {
			t.Errorf("Merge(%d).Glyph() = %q, want %q", tt.m, got, tt.glyph)
		}
	}
}

// TestMergeGlyphPrefix keeps compact rows from drifting away from the badges
// (RESEARCH Pitfall 11). MergeNone is the blank cell and has no badge.
func TestMergeGlyphPrefix(t *testing.T) {
	for _, m := range []Merge{MergeUnknown, MergeMergeable, MergeBlocked, MergeBehind, MergeConflicts} {
		if !strings.HasPrefix(m.Badge(), m.Glyph()) || m.Glyph() == "" {
			t.Errorf("Merge(%d) badge %q does not start with glyph %q", m, m.Badge(), m.Glyph())
		}
	}
	if MergeNone.Badge() != "" {
		t.Errorf("MergeNone badge = %q, want blank", MergeNone.Badge())
	}
}

// TestMergeDoesNotSort: Merge never changes the attention order (D-01).
func TestMergeDoesNotSort(t *testing.T) {
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	mk := func(n int, m Merge) PR {
		return PR{Ref: Ref{Repo: "o/r", Number: n}, CreatedAt: base, CI: CIPassing, Review: ReviewApproved, Merge: m}
	}
	for _, m := range []Merge{MergeNone, MergeUnknown, MergeMergeable, MergeBlocked, MergeBehind, MergeConflicts} {
		if a, b := Bucket(mk(1, MergeNone)), Bucket(mk(1, m)); a != b {
			t.Errorf("Bucket differs for Merge %d: %d vs %d", m, a, b)
		}
	}
	prs := []PR{mk(1, MergeConflicts), mk(2, MergeMergeable), mk(3, MergeNone), mk(4, MergeBlocked)}
	Sort(prs)
	for i, pr := range prs {
		if pr.Ref.Number != i+1 {
			t.Errorf("position %d holds #%d: Merge changed the order", i, pr.Ref.Number)
		}
	}
}

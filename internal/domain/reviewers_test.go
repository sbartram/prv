package domain

import (
	"slices"
	"testing"
)

func TestMergeReviewers(t *testing.T) {
	approved := func(l string) ReviewInput { return ReviewInput{l, "APPROVED"} }
	changes := func(l string) ReviewInput { return ReviewInput{l, "CHANGES_REQUESTED"} }
	user := func(l string) RequestInput { return RequestInput{"User", l} }
	person := func(n string, s ReviewerState) Reviewer { return Reviewer{n, ReviewerPerson, s} }
	hidden := Reviewer{"", ReviewerHidden, ReviewerPending}

	tests := []struct {
		name        string
		opinionated []ReviewInput
		requested   []RequestInput
		latest      []ReviewInput
		want        []Reviewer
	}{
		{"no reviews and no requests", nil, nil, nil, nil},
		{"request and approval", []ReviewInput{approved("alice")}, []RequestInput{user("carol")}, nil,
			[]Reviewer{person("carol", ReviewerPending), person("alice", ReviewerApproved)}},
		{"re-requested reviewer is only pending, any letter case",
			[]ReviewInput{changes("bob"), approved("Alice")}, []RequestInput{user("alice")}, nil,
			[]Reviewer{person("bob", ReviewerChangesRequested), person("alice", ReviewerPending)}},
		{"re-requested after changes requested is pending",
			[]ReviewInput{changes("bob")}, []RequestInput{user("BOB")}, nil,
			[]Reviewer{person("BOB", ReviewerPending)}},
		{"null reviewers are hidden pending, never dropped",
			nil, []RequestInput{{}, {}, {"Team", "acme/core"}}, nil,
			[]Reviewer{{"acme/core", ReviewerTeam, ReviewerPending}, hidden, hidden}},
		{"bot loses its suffix", nil, []RequestInput{{"Bot", "renovate[bot]"}}, nil,
			[]Reviewer{person("renovate", ReviewerPending)}},
		{"bot review and bot request are one pending reviewer",
			[]ReviewInput{approved("renovate[bot]")}, []RequestInput{{"Bot", "renovate[bot]"}}, nil,
			[]Reviewer{person("renovate", ReviewerPending)}},
		{"mannequin is a person and an enterprise team is a team",
			nil, []RequestInput{{"Mannequin", "old-user"}, {"EnterpriseTeam", "ent/reviewers"}}, nil,
			[]Reviewer{person("old-user", ReviewerPending), {"ent/reviewers", ReviewerTeam, ReviewerPending}}},
		{"a request with a type but no name is hidden", nil, []RequestInput{{"User", ""}, {"Team", ""}}, nil,
			[]Reviewer{hidden, hidden}},
		{"a review with a null author is skipped",
			[]ReviewInput{{"", "APPROVED"}, approved("alice")}, nil, nil,
			[]Reviewer{person("alice", ReviewerApproved)}},
		{"opinionated states other than approved and changes are ignored",
			[]ReviewInput{{"dave", "COMMENTED"}, {"erin", "DISMISSED"}, {"fay", "PENDING"}, approved("alice")}, nil, nil,
			[]Reviewer{person("alice", ReviewerApproved)}},
		{"detail-only reviews add commented and dismissed, an existing reviewer stays",
			[]ReviewInput{approved("alice")}, nil,
			[]ReviewInput{{"dave", "COMMENTED"}, {"erin", "DISMISSED"}, {"alice", "COMMENTED"}},
			[]Reviewer{person("alice", ReviewerApproved), person("dave", ReviewerCommented), person("erin", ReviewerDismissed)}},
		{"detail-only reviews do not override a pending request",
			nil, []RequestInput{user("dave")}, []ReviewInput{{"dave", "COMMENTED"}},
			[]Reviewer{person("dave", ReviewerPending)}},
		{"detail-only review with a null author or an unknown state is skipped",
			nil, nil, []ReviewInput{{"", "COMMENTED"}, {"gus", "PENDING"}}, nil},
		{"order is changes, pending, approved, commented, dismissed",
			[]ReviewInput{approved("zed"), changes("yan")},
			[]RequestInput{user("xia")},
			[]ReviewInput{{"wes", "DISMISSED"}, {"vic", "COMMENTED"}},
			[]Reviewer{
				person("yan", ReviewerChangesRequested), person("xia", ReviewerPending),
				person("zed", ReviewerApproved), person("vic", ReviewerCommented), person("wes", ReviewerDismissed)}},
		{"within a state: person, team, hidden, then lower-cased name",
			nil, []RequestInput{{}, {"Team", "acme/a"}, user("Bob"), user("alice"), {"Team", "Acme/B"}}, nil,
			[]Reviewer{
				person("alice", ReviewerPending), person("Bob", ReviewerPending),
				{"acme/a", ReviewerTeam, ReviewerPending}, {"Acme/B", ReviewerTeam, ReviewerPending}, hidden}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeReviewers(tt.opinionated, tt.requested, tt.latest)
			if !slices.Equal(got, tt.want) {
				t.Errorf("MergeReviewers =\n%v\nwant\n%v", got, tt.want)
			}
			// The same inputs in reversed order give the same output.
			rev := func(in []ReviewInput) []ReviewInput { c := slices.Clone(in); slices.Reverse(c); return c }
			reqs := slices.Clone(tt.requested)
			slices.Reverse(reqs)
			if again := MergeReviewers(rev(tt.opinionated), reqs, rev(tt.latest)); !slices.Equal(again, tt.want) {
				t.Errorf("reversed inputs gave\n%v\nwant\n%v", again, tt.want)
			}
		})
	}
}

func TestReviewerGlyphs(t *testing.T) {
	tests := []struct {
		s     ReviewerState
		glyph string
		word  string
	}{
		{ReviewerChangesRequested, "✗", "changes requested"},
		{ReviewerPending, "…", "pending"},
		{ReviewerApproved, "✓", "approved"},
		{ReviewerCommented, "·", "commented"},
		{ReviewerDismissed, "–", "dismissed"},
	}
	for _, tt := range tests {
		if got := tt.s.Glyph(); got != tt.glyph {
			t.Errorf("state %d Glyph() = %q, want %q", tt.s, got, tt.glyph)
		}
		if got := tt.s.Word(); got != tt.word {
			t.Errorf("state %d Word() = %q, want %q", tt.s, got, tt.word)
		}
	}
}

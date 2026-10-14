package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/sbartram/prv/internal/domain"
)

// TestCICell: the CI cell names the failing check, keeps " (cancelled)" and
// " +N" whole, and shortens only the name when the cell is too narrow. Every
// width is measured in terminal cells.
func TestCICell(t *testing.T) {
	failing := func(f domain.FailingCheck) domain.PR {
		return domain.PR{CI: domain.CIFailing, Failing: f}
	}
	tests := []struct {
		name string
		pr   domain.PR
		w    int
		want string // exact text; empty means check the properties below instead
		// properties, used when want is empty
		exactW   int // the width must be exactly this (0: no check)
		maxW     int // the width must be at most this (0: no check)
		prefix   string
		suffix   string
		hasEllip bool
	}{
		{name: "name and count", pr: failing(domain.FailingCheck{Known: true, Name: "lint", More: 2}), w: 24, want: "✗ lint +2"},
		{name: "cancelled", pr: failing(domain.FailingCheck{Known: true, Name: "deploy", Cancelled: true}), w: 24, want: "✗ deploy (cancelled)"},
		{name: "cancelled and count", pr: failing(domain.FailingCheck{Known: true, Name: "deploy", Cancelled: true, More: 1}), w: 24, want: "✗ deploy (cancelled) +1"},
		{name: "unknown name shows the plain badge", pr: failing(domain.FailingCheck{}), w: 24, want: "✗ failing"},
		{name: "passing PR shows its badge", pr: domain.PR{CI: domain.CIPassing}, w: 24, want: "✓ passing"},
		{name: "unsized pane: the full text, uncut", pr: failing(domain.FailingCheck{Known: true, Name: "very-long-check-name-here", More: 2}), w: 0,
			want: "✗ very-long-check-name-here +2"},
		{name: "fits exactly", pr: failing(domain.FailingCheck{Known: true, Name: "lint", More: 2}), w: 9, want: "✗ lint +2"},
		{name: "long name is shortened, count stays",
			pr: failing(domain.FailingCheck{Known: true, Name: "very-long-check-name-here", More: 2}), w: 14,
			exactW: 14, prefix: "✗ very", suffix: "… +2", hasEllip: true},
		{name: "long cancelled name keeps (cancelled) and the count",
			pr: failing(domain.FailingCheck{Known: true, Name: "a-very-long-deploy-job-name", Cancelled: true, More: 3}), w: 24,
			exactW: 24, prefix: "✗ a-very", suffix: "… (cancelled) +3", hasEllip: true},
		{name: "CJK name",
			pr: failing(domain.FailingCheck{Known: true, Name: "数据迁移工具修复问题测试", More: 2}), w: 12,
			maxW: 12, prefix: "✗ 数", suffix: "… +2", hasEllip: true},
		{name: "emoji ZWJ name",
			pr: failing(domain.FailingCheck{Known: true, Name: "👨‍👩‍👧‍👦👨‍👩‍👧‍👦👨‍👩‍👧‍👦 family build", More: 1}), w: 10,
			maxW: 10, prefix: "✗ ", suffix: "… +1", hasEllip: true},
		{name: "no room for 2 cells of name: the plain badge cut to the width",
			pr: failing(domain.FailingCheck{Known: true, Name: "lint", More: 2}), w: 5, want: "✗ fa…"},
		{name: "one cell short of 2 cells of name",
			pr: failing(domain.FailingCheck{Known: true, Name: "a-long-name", More: 2}), w: 6, want: "✗ fai…"},
		{name: "exactly 2 cells of name",
			pr: failing(domain.FailingCheck{Known: true, Name: "a-long-name", More: 2}), w: 7,
			exactW: 7, prefix: "✗ ", suffix: "… +2", hasEllip: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ciCell(tt.pr, tt.w)
			if tt.want != "" {
				if got != tt.want {
					t.Fatalf("ciCell = %q, want %q", got, tt.want)
				}
			} else {
				if !strings.HasPrefix(got, tt.prefix) {
					t.Errorf("ciCell = %q, want prefix %q", got, tt.prefix)
				}
				if !strings.HasSuffix(got, tt.suffix) {
					t.Errorf("ciCell = %q, want suffix %q", got, tt.suffix)
				}
				if tt.hasEllip && !strings.Contains(got, "…") {
					t.Errorf("ciCell = %q, want a shortened name ending in …", got)
				}
				if tt.exactW > 0 && lipgloss.Width(got) != tt.exactW {
					t.Errorf("width(%q) = %d, want exactly %d", got, lipgloss.Width(got), tt.exactW)
				}
			}
			if tt.w > 0 && lipgloss.Width(got) > tt.w {
				t.Errorf("width(%q) = %d, want at most %d", got, lipgloss.Width(got), tt.w)
			}
			if tt.maxW > 0 && lipgloss.Width(got) > tt.maxW {
				t.Errorf("width(%q) = %d, want at most %d", got, lipgloss.Width(got), tt.maxW)
			}
		})
	}
}

// TestCICellCap: at the real column cap, no name makes the cell wider than 24
// cells, and a short suffix stays whole.
func TestCICellCap(t *testing.T) {
	long := strings.Repeat("数据", 40)
	for _, f := range []domain.FailingCheck{
		{Known: true, Name: long},
		{Known: true, Name: long, More: 12},
		{Known: true, Name: long, Cancelled: true, More: 12},
		{Known: true, Name: strings.Repeat("x", 500), More: 99},
	} {
		got := ciCell(domain.PR{CI: domain.CIFailing, Failing: f}, maxCIColW)
		if w := lipgloss.Width(got); w > maxCIColW {
			t.Errorf("width(%q) = %d, want at most %d", got, w, maxCIColW)
		}
		if f.Cancelled && !strings.Contains(got, "(cancelled)") {
			t.Errorf("ciCell = %q lost its (cancelled) suffix", got)
		}
	}
}

// TestReviewersCell: tokens are a glyph glued to a name in slice order, the
// viewer is "me", hidden requests make one token, and a cell that is too
// narrow ends in "+N" counting every reviewer not shown (LIST-04, D-06).
func TestReviewersCell(t *testing.T) {
	hidden := domain.Reviewer{Kind: domain.ReviewerHidden, State: domain.ReviewerPending}
	rs6 := []domain.Reviewer{
		{Name: "bob", State: domain.ReviewerChangesRequested},
		{Name: "carol", State: domain.ReviewerPending},
		hidden, hidden,
		{Name: "alice", State: domain.ReviewerApproved},
		{Name: "octo", State: domain.ReviewerApproved},
	}
	tests := []struct {
		name   string
		rs     []domain.Reviewer
		viewer string
		w      int
		want   string
	}{
		{"everything, unsized", rs6, "octo", 0, "✗bob …carol …2 teams ✓alice ✓me"},
		{"everything, exactly fitting", rs6, "octo", 31, "✗bob …carol …2 teams ✓alice ✓me"},
		{"one reviewer short", rs6, "octo", 30, "✗bob …carol …2 teams ✓alice +1"},
		{"narrow: the rest becomes +N", rs6, "octo", 16, "✗bob …carol +4"},
		{"no room for a token: the total", rs6, "octo", 3, "+6"},
		{"the viewer matches without case", rs6, "OCTO", 0, "✗bob …carol …2 teams ✓alice ✓me"},
		{"another viewer keeps the login", rs6, "dave", 0, "✗bob …carol …2 teams ✓alice ✓octo"},
		{"one hidden request", []domain.Reviewer{hidden}, "octo", 0, "…team"},
		{"three hidden requests", []domain.Reviewer{hidden, hidden, hidden}, "octo", 0, "…3 teams"},
		{"hidden token does not fit", []domain.Reviewer{hidden, hidden, hidden}, "octo", 5, "+3"},
		{"a visible team", []domain.Reviewer{{Name: "acme/core", Kind: domain.ReviewerTeam, State: domain.ReviewerPending}}, "octo", 0, "…@acme/core"},
		{"no reviewers", nil, "octo", 0, ""},
		{"no reviewers in a width", nil, "octo", 10, ""},
		{"CJK login counted in cells", []domain.Reviewer{
			{Name: "数据数据", State: domain.ReviewerApproved}, {Name: "bob", State: domain.ReviewerPending},
		}, "octo", 12, "✓数据数据 +1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reviewersCell(tt.rs, tt.viewer, tt.w)
			if got != tt.want {
				t.Errorf("reviewersCell(w=%d) = %q, want %q", tt.w, got, tt.want)
			}
			if tt.w > 0 && lipgloss.Width(got) > tt.w {
				t.Errorf("width(%q) = %d, want at most %d", got, lipgloss.Width(got), tt.w)
			}
		})
	}

	t.Run("never wider than the cell, never a silent loss", func(t *testing.T) {
		for w := 1; w <= 40; w++ {
			got := reviewersCell(rs6, "octo", w)
			if lipgloss.Width(got) > w {
				t.Errorf("w=%d: %q is %d cells wide", w, got, lipgloss.Width(got))
			}
			// Either all six reviewers are named, or the cell says how many are
			// not shown (or is cut to a bare ellipsis at 1 cell).
			if !strings.Contains(got, "+") && !strings.Contains(got, "✓me") && got != "…" {
				t.Errorf("w=%d: %q hides reviewers without a +N", w, got)
			}
		}
	})
}

func TestReviewersCount(t *testing.T) {
	r := domain.Reviewer{Name: "bob", State: domain.ReviewerApproved}
	for _, tt := range []struct {
		rs   []domain.Reviewer
		want string
	}{
		{nil, ""}, {[]domain.Reviewer{r}, "+1"}, {[]domain.Reviewer{r, r, r, r, r, r}, "+6"},
		{[]domain.Reviewer{{Kind: domain.ReviewerHidden}, {Kind: domain.ReviewerHidden}}, "+2"},
	} {
		if got := reviewersCount(tt.rs); got != tt.want {
			t.Errorf("reviewersCount(%d reviewers) = %q, want %q", len(tt.rs), got, tt.want)
		}
	}
}

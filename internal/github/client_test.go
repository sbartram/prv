package github

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sbartram/prv/internal/domain"
)

// fixtureTransport serves the named testdata files in order and records the
// request bodies.
type fixtureTransport struct {
	t      *testing.T
	files  []string
	calls  int
	bodies [][]byte
}

func (f *fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, _ := io.ReadAll(req.Body)
	f.bodies = append(f.bodies, raw)
	if f.calls >= len(f.files) {
		f.t.Fatalf("unexpected request %d; only %d fixtures", f.calls+1, len(f.files))
	}
	data, err := os.ReadFile(filepath.Join("testdata", f.files[f.calls]))
	if err != nil {
		f.t.Fatal(err)
	}
	f.calls++
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(data)),
		Request:    req,
	}, nil
}

func newFixtureClient(t *testing.T, files ...string) (*Client, *fixtureTransport) {
	t.Helper()
	tr := &fixtureTransport{t: t, files: files}
	c, err := NewClient(Options{
		Token:      "test-token-not-real",
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  tr,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c, tr
}

func fetchStatus(t *testing.T) (domain.Snapshot, map[int]domain.PR) {
	t.Helper()
	c, _ := newFixtureClient(t, "status.json")
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byNum := map[int]domain.PR{}
	for _, pr := range snap.PRs {
		byNum[pr.Ref.Number] = pr
	}
	return snap, byNum
}

func TestStatusMapping(t *testing.T) {
	_, prs := fetchStatus(t)
	tests := []struct {
		n      int
		review domain.Review
		ci     domain.CI
	}{
		{1, domain.ReviewApproved, domain.CINone},
		{2, domain.ReviewNone, domain.CINone},
		{3, domain.ReviewChangesRequested, domain.CINone},
		{4, domain.ReviewRequired, domain.CINone},
		{5, domain.ReviewApproved, domain.CINone},
		{6, domain.ReviewChangesRequested, domain.CINone},
		{7, domain.ReviewApproved, domain.CINone},
		{8, domain.ReviewRequired, domain.CINone},
		{9, domain.ReviewNone, domain.CINone},
		{10, domain.ReviewNone, domain.CIPassing},
		{11, domain.ReviewNone, domain.CIFailing},
		{12, domain.ReviewNone, domain.CIFailing},
		{13, domain.ReviewNone, domain.CIRunning},
		{14, domain.ReviewNone, domain.CIRunning},
		{15, domain.ReviewNone, domain.CIUnknown},
		{16, domain.ReviewChangesRequested, domain.CIFailing},
		{17, domain.ReviewApproved, domain.CINone},
		{19, domain.ReviewApproved, domain.CIPassing},
	}
	for _, tt := range tests {
		pr, ok := prs[tt.n]
		if !ok {
			t.Errorf("PR %d missing from snapshot", tt.n)
			continue
		}
		if pr.Review != tt.review {
			t.Errorf("PR %d review = %q, want %q", tt.n, pr.Review.Badge(), tt.review.Badge())
		}
		if pr.CI != tt.ci {
			t.Errorf("PR %d CI = %q, want %q", tt.n, pr.CI.Badge(), tt.ci.Badge())
		}
	}
	if pr := prs[8]; pr.State != domain.StateClosed {
		t.Errorf("PR 8 state = %v, want closed", pr.State)
	}
	if pr := prs[16]; !pr.IsDraft || pr.State != domain.StateOpen {
		t.Errorf("PR 16 draft/state = %v/%v, want draft and open", pr.IsDraft, pr.State)
	}
}

func TestCommentAfterApprove(t *testing.T) {
	_, prs := fetchStatus(t)
	for _, n := range []int{7, 19} {
		pr := prs[n]
		if pr.Review != domain.ReviewApproved {
			t.Errorf("PR %d review = %v, want approved", n, pr.Review)
		}
		if got := pr.Review.Badge(); got != "✓ approved" {
			t.Errorf("PR %d badge = %q, want %q", n, got, "✓ approved")
		}
	}
}

func TestCleanAtSeam(t *testing.T) {
	snap, prs := fetchStatus(t)
	pr := prs[18]
	if pr.Title != "eviltitle x" {
		t.Errorf("Title = %q, want %q", pr.Title, "eviltitle x")
	}
	if pr.Author != "badguy" {
		t.Errorf("Author = %q, want %q", pr.Author, "badguy")
	}
	if pr.Ref.Repo != "acme/repo" {
		t.Errorf("Repo = %q, want %q", pr.Ref.Repo, "acme/repo")
	}
	all := []string{snap.Viewer}
	for _, p := range snap.PRs {
		all = append(all, p.Title, p.Author, p.Ref.Repo)
	}
	for _, s := range all {
		for _, bad := range []string{"\x1b", "\a", "‮"} {
			if strings.Contains(s, bad) {
				t.Errorf("mapped string %q still contains %q", s, bad)
			}
		}
	}
}

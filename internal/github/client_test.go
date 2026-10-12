package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"

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

// noUnsafe reports why a mapped string is unsafe to print, or "" if it is fine.
func noUnsafe(s string) string {
	for _, r := range s {
		switch {
		case unicode.IsControl(r):
			return fmt.Sprintf("control character %U", r)
		case r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069:
			return fmt.Sprintf("bidi character %U", r)
		}
	}
	return ""
}

// TestCleanAtSeam sends the hostile PR 18 through the real client. go-gh's own
// response sanitiser already turns ESC and BEL into visible "^[" and "^G"
// before the mapper sees them, so this test asserts the outcome only: no
// control or bidi character reaches the screen strings.
func TestCleanAtSeam(t *testing.T) {
	snap, prs := fetchStatus(t)
	pr := prs[18]
	if !strings.Contains(pr.Title, "title x") {
		t.Errorf("Title = %q, want the tab turned into a space", pr.Title)
	}
	all := []string{snap.Viewer}
	for _, p := range snap.PRs {
		all = append(all, p.Title, p.Author, p.Ref.Repo)
	}
	for _, s := range all {
		if why := noUnsafe(s); why != "" {
			t.Errorf("mapped string %q has a %s", s, why)
		}
	}
}

// TestMapperCleansRemoteStrings decodes the fixture directly, bypassing go-gh's
// sanitiser, so the mapper's own domain.Clean calls are what get tested.
func TestMapperCleansRemoteStrings(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "status.json"))
	if err != nil {
		t.Fatal(err)
	}
	var env struct {
		Data wireResp `json:"data"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Data.Search.Nodes[17].Title, "\x1b") {
		t.Fatal("fixture PR 18 should still hold a raw ESC before mapping")
	}
	snap := mapResponse(env.Data)
	var pr domain.PR
	for _, p := range snap.PRs {
		if p.Ref.Number == 18 {
			pr = p
		}
	}
	if pr.Title != "eviltitle x" {
		t.Errorf("Title = %q, want %q", pr.Title, "eviltitle x")
	}
	if pr.Author != "badguy" {
		t.Errorf("Author = %q, want %q", pr.Author, "badguy")
	}
	if pr.Ref.Repo != "acme/repo" {
		t.Errorf("Repo = %q, want %q", pr.Ref.Repo, "acme/repo")
	}
	for _, p := range snap.PRs {
		for _, s := range []string{p.Title, p.Author, p.Ref.Repo} {
			if why := noUnsafe(s); why != "" {
				t.Errorf("mapped string %q has a %s", s, why)
			}
		}
	}
}

func refs(snap domain.Snapshot) []string {
	var out []string
	for _, pr := range snap.PRs {
		out = append(out, pr.Ref.String())
	}
	return out
}

// TestMembership: authored and assigned PRs interleave by the display order
// only; duplicates, Issue nodes, null nodes and out-of-window PRs are handled.
func TestMembership(t *testing.T) {
	c, _ := newFixtureClient(t, "search_page1.json", "search_page2.json")
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"acme/web#77", "acme/api#481", "deps/bot#1", "acme/a#9", "acme/a#10", "acme/b#10",
		"acme/old#3", "acme/one#7", "acme/api#500", "acme/two#7", "acme/api#503",
	}
	if got := refs(snap); !slices.Equal(got, want) {
		t.Errorf("refs =\n%v\nwant\n%v", got, want)
	}
	if snap.Viewer != "octo" {
		t.Errorf("Viewer = %q, want octo", snap.Viewer)
	}
	byRef := map[string]domain.PR{}
	for _, pr := range snap.PRs {
		byRef[pr.Ref.String()] = pr
	}
	if got := byRef["acme/web#77"].Author; got != "hubot" {
		t.Errorf("assigned PR author = %q, want hubot", got)
	}
	if pr := byRef["acme/old#3"]; pr.Author != "ghost" || pr.Title != "" {
		t.Errorf("acme/old#3 author/title = %q/%q, want ghost and empty", pr.Author, pr.Title)
	}
	if got := byRef["deps/bot#1"].Author; got != "dependabot" {
		t.Errorf("deps/bot#1 author = %q, want dependabot", got)
	}
	// Same number in two repositories gives two rows.
	if _, ok := byRef["acme/one#7"]; !ok {
		t.Error("acme/one#7 missing")
	}
	if _, ok := byRef["acme/two#7"]; !ok {
		t.Error("acme/two#7 missing")
	}
	// The duplicate PR_B (page 1 and page 2) appears once.
	n := 0
	for _, pr := range snap.PRs {
		if pr.ID == "PR_B" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("PR_B appears %d times, want 1", n)
	}
	for _, dropped := range []string{"acme/api#501", "acme/api#502"} {
		if _, ok := byRef[dropped]; ok {
			t.Errorf("%s is outside the window and must be dropped", dropped)
		}
	}
	if pr := byRef["acme/api#503"]; pr.State != domain.StateUnknown {
		t.Errorf("acme/api#503 state = %v, want unknown", pr.State)
	}
	if snap.Truncated {
		t.Error("two pages ending in hasNextPage false must not be truncated")
	}
}

func TestPagination(t *testing.T) {
	c, tr := newFixtureClient(t, "search_page1.json", "search_page2.json")
	if _, err := c.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(tr.bodies) != 2 {
		t.Fatalf("requests = %d, want 2", len(tr.bodies))
	}
	after := func(i int) any {
		var payload struct {
			Variables map[string]any `json:"variables"`
		}
		if err := json.Unmarshal(tr.bodies[i], &payload); err != nil {
			t.Fatal(err)
		}
		return payload.Variables["after"]
	}
	if got := after(0); got != nil {
		t.Errorf("first request after = %v, want none", got)
	}
	if got := after(1); got != "CUR1" {
		t.Errorf("second request after = %v, want CUR1", got)
	}
}

// TestTruncatedAfterFivePages is the accepted 500-result cap: a transport that
// always says another page exists gets exactly five requests, and the snapshot
// says it is truncated.
func TestTruncatedAfterFivePages(t *testing.T) {
	files := []string{"search_page1.json", "search_page1.json", "search_page1.json", "search_page1.json", "search_page1.json"}
	c, tr := newFixtureClient(t, files...)
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if tr.calls != 5 || len(tr.bodies) != 5 {
		t.Errorf("requests = %d, want exactly 5", len(tr.bodies))
	}
	if !snap.Truncated {
		t.Error("Truncated = false, want true after five pages with more remaining")
	}
}

func TestRateLimitDecoded(t *testing.T) {
	c, _ := newFixtureClient(t, "search_page1.json", "search_page2.json")
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rl := snap.RateLimit
	if rl.Cost != 2 || rl.Remaining != 4989 {
		t.Errorf("RateLimit cost/remaining = %d/%d, want 2/4989", rl.Cost, rl.Remaining)
	}
	if want := time.Date(2026, 10, 11, 13, 0, 0, 0, time.UTC); !rl.ResetAt.Equal(want) {
		t.Errorf("ResetAt = %v, want %v", rl.ResetAt, want)
	}
}

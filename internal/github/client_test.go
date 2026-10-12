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

// failingNoneBody answers a FailingChecks request with no nodes. It has no
// rateLimit block on purpose, so the cost and Limit assertions of the discovery
// tests stay valid.
const failingNoneBody = `{"data":{"nodes":[]}}`

// opName returns the GraphQL operation name of a request body ("Watch",
// "FailingChecks"), or "" when the body is not a query.
func opName(body []byte) string {
	var payload struct {
		Query string `json:"query"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return ""
	}
	rest, ok := strings.CutPrefix(strings.TrimSpace(payload.Query), "query ")
	if !ok {
		return ""
	}
	if i := strings.IndexAny(rest, "( {"); i >= 0 {
		rest = rest[:i]
	}
	return rest
}

// fixtureTransport routes by operation. "Watch" requests are served from files
// in order (only they count in calls and bodies); "FailingChecks" requests are
// served from failing in order, then with failingNoneBody.
type fixtureTransport struct {
	t          *testing.T
	files      []string
	calls      int
	bodies     [][]byte
	failing    []string
	failBodies [][]byte
}

func (f *fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, _ := io.ReadAll(req.Body)
	var data []byte
	switch op := opName(raw); op {
	case "Watch":
		f.bodies = append(f.bodies, raw)
		if f.calls >= len(f.files) {
			f.t.Fatalf("unexpected request %d; only %d fixtures", f.calls+1, len(f.files))
		}
		var err error
		data, err = os.ReadFile(filepath.Join("testdata", f.files[f.calls]))
		if err != nil {
			f.t.Fatal(err)
		}
		f.calls++
	case "FailingChecks":
		n := len(f.failBodies)
		f.failBodies = append(f.failBodies, raw)
		data = []byte(failingNoneBody)
		if n < len(f.failing) {
			var err error
			data, err = os.ReadFile(filepath.Join("testdata", f.failing[n]))
			if err != nil {
				f.t.Fatal(err)
			}
		}
	default:
		f.t.Fatalf("unexpected operation %q", op)
	}
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
	// go-gh shows the ESC as a visible "^[" (see above); the bidi override is gone.
	if got := reviewerNames(pr); !slices.Equal(got, []string{"req^[[2Juser", "rev^[[31miewer"}) {
		t.Errorf("PR 18 reviewer names = %q, want [req^[[2Juser rev^[[31miewer]", got)
	}
	all := []string{snap.Viewer}
	for _, p := range snap.PRs {
		all = append(all, p.Title, p.Author, p.Ref.Repo)
		all = append(all, reviewerNames(p)...)
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
	if got := reviewerNames(pr); !slices.Equal(got, []string{"requser", "reviewer"}) {
		t.Errorf("Reviewer names = %q, want [requser reviewer]", got)
	}
	for _, p := range snap.PRs {
		for _, s := range append([]string{p.Title, p.Author, p.Ref.Repo}, reviewerNames(p)...) {
			if why := noUnsafe(s); why != "" {
				t.Errorf("mapped string %q has a %s", s, why)
			}
		}
	}
}

// reviewerNames returns the names of a PR's reviewers, in order.
func reviewerNames(pr domain.PR) []string {
	var out []string
	for _, r := range pr.Reviewers {
		out = append(out, r.Name)
	}
	return out
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
		"acme/api#481", "deps/bot#1", "acme/web#77", "acme/a#9", "acme/a#10", "acme/b#10",
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
	// createdAt comes from the query and is kept at full precision.
	if got, want := byRef["acme/api#481"].CreatedAt, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("acme/api#481 CreatedAt = %v, want %v", got, want)
	}
	if got, want := byRef["deps/bot#1"].CreatedAt, time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("deps/bot#1 CreatedAt = %v, want %v", got, want)
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

// TestFetchAttentionOrder: Fetch hands rows back in D-01 bucket order, and the
// discovery query asks for createdAt (the D-02 tie-break key).
func TestFetchAttentionOrder(t *testing.T) {
	c, tr := newFixtureClient(t, "status.json")
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.PRs) < 2 {
		t.Fatalf("fixture gave %d PRs, want several", len(snap.PRs))
	}
	prev := 0
	for _, pr := range snap.PRs {
		b := domain.Bucket(pr)
		if b < prev {
			t.Errorf("%s is in bucket %d after bucket %d: order is not attention order", pr.Ref, b, prev)
		}
		prev = b
	}
	if len(tr.bodies) == 0 || !strings.Contains(string(tr.bodies[0]), "createdAt") {
		t.Error("the discovery query must request createdAt")
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
	c, tr := newFixtureClient(t, "search_page1.json", "search_page2.json")
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rl := snap.RateLimit
	if rl.Limit != 5000 || rl.Cost != 2 || rl.Remaining != 4989 {
		t.Errorf("RateLimit limit/cost/remaining = %d/%d/%d, want 5000/2/4989", rl.Limit, rl.Cost, rl.Remaining)
	}
	if want := time.Date(2026, 10, 11, 13, 0, 0, 0, time.UTC); !rl.ResetAt.Equal(want) {
		t.Errorf("ResetAt = %v, want %v", rl.ResetAt, want)
	}
	if len(tr.bodies) == 0 || !strings.Contains(string(tr.bodies[0]), "rateLimit { limit cost remaining resetAt }") {
		t.Error("the discovery query must request rateLimit { limit cost remaining resetAt }")
	}
	// status.json carries no limit field on purpose: it decodes as 0.
	snap, _ = fetchStatus(t)
	if snap.RateLimit.Limit != 0 {
		t.Errorf("Limit without a limit field = %d, want 0", snap.RateLimit.Limit)
	}
}

// TestMergeMapping: Fetch maps mergeStateStatus and mergeable through
// DeriveMerge, deciding state and draft first (D-07, RESEARCH F4/F5).
func TestMergeMapping(t *testing.T) {
	c, tr := newFixtureClient(t, "blockers.json")
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]domain.Merge{
		1: domain.MergeMergeable, 2: domain.MergeMergeable, 3: domain.MergeMergeable,
		4: domain.MergeBlocked, 5: domain.MergeBehind, 6: domain.MergeConflicts,
		7: domain.MergeUnknown, 8: domain.MergeConflicts, 9: domain.MergeUnknown,
		10: domain.MergeNone, 11: domain.MergeNone, 12: domain.MergeUnknown,
	}
	if len(snap.PRs) != len(want) {
		t.Fatalf("PRs = %d, want %d", len(snap.PRs), len(want))
	}
	for _, pr := range snap.PRs {
		if got := pr.Merge; got != want[pr.Ref.Number] {
			t.Errorf("PR %d Merge = %q, want %q", pr.Ref.Number, got.Badge(), want[pr.Ref.Number].Badge())
		}
	}
	if len(tr.failBodies) != 0 {
		t.Errorf("FailingChecks requests = %d, want 0 (no failing rollup in the fixture)", len(tr.failBodies))
	}
	for _, field := range []string{"mergeStateStatus mergeable"} {
		if !strings.Contains(string(tr.bodies[0]), field) {
			t.Errorf("the discovery query must request %q", field)
		}
	}
}

// TestReviewerMapping: Fetch merges latestOpinionatedReviews and reviewRequests
// into D-06 order, pending wins over an old review, and a null requested
// reviewer is a hidden pending reviewer, never dropped.
func TestReviewerMapping(t *testing.T) {
	c, tr := newFixtureClient(t, "blockers.json")
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	pending := func(n string) domain.Reviewer { return domain.Reviewer{Name: n} }
	hidden := domain.Reviewer{Kind: domain.ReviewerHidden}
	want := map[int][]domain.Reviewer{
		1: {pending("carol"), {Name: "alice", State: domain.ReviewerApproved}},
		2: {{Name: "bob", State: domain.ReviewerChangesRequested}, pending("alice")},
		3: {{Name: "acme/core", Kind: domain.ReviewerTeam}, hidden, hidden},
		4: {pending("renovate")},
		5: nil,
		6: {pending("old-user"), {Name: "ent/reviewers", Kind: domain.ReviewerTeam}},
		7: {{Name: "octo", State: domain.ReviewerApproved}},
	}
	for _, pr := range snap.PRs {
		if got := pr.Reviewers; !slices.Equal(got, want[pr.Ref.Number]) {
			t.Errorf("PR %d Reviewers =\n%v\nwant\n%v", pr.Ref.Number, got, want[pr.Ref.Number])
		}
	}
	body := string(tr.bodies[0])
	for _, field := range []string{"reviewRequests(first: 20)", "combinedSlug", "requestedReviewer", "__typename"} {
		if !strings.Contains(body, field) {
			t.Errorf("the discovery query must request %q", field)
		}
	}
}

// TestNoFailingNoRequest: a refresh in which no listed PR has a FAILURE or
// ERROR rollup sends no FailingChecks request (D-01).
func TestNoFailingNoRequest(t *testing.T) {
	c, tr := newFixtureClient(t, "search_page2.json")
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.PRs) == 0 {
		t.Fatal("no PRs: the test would prove nothing")
	}
	if len(tr.failBodies) != 0 {
		t.Errorf("FailingChecks requests = %d, want 0", len(tr.failBodies))
	}
	for _, pr := range snap.PRs {
		if pr.Failing != (domain.FailingCheck{}) {
			t.Errorf("%s Failing = %+v, want the zero value", pr.Ref, pr.Failing)
		}
	}
}

// TestFailingNames: step 2 names the failing check of every failing PR, from
// the de-duplicated contexts, with one request carrying all three ids.
func TestFailingNames(t *testing.T) {
	c, tr := newFixtureClient(t, "status.json")
	tr.failing = []string{"failing_contexts.json"}
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]domain.FailingCheck{
		11: {Known: true, Name: "lint", More: 2},           // build failed, then passed in a newer suite
		12: {Known: true, Name: "ci/jenkins"},              // a commit status
		16: {Known: true, Name: "deploy", Cancelled: true}, // only a cancelled job
	}
	for _, pr := range snap.PRs {
		if got := pr.Failing; got != want[pr.Ref.Number] {
			t.Errorf("PR %d Failing = %+v, want %+v", pr.Ref.Number, got, want[pr.Ref.Number])
		}
	}
	if len(tr.failBodies) != 1 {
		t.Fatalf("FailingChecks requests = %d, want 1", len(tr.failBodies))
	}
	var payload struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	if err := json.Unmarshal(tr.failBodies[0], &payload); err != nil {
		t.Fatal(err)
	}
	ids, _ := payload.Variables["ids"].([]any)
	if !slices.Equal(ids, []any{"PR_11", "PR_12", "PR_16"}) {
		t.Errorf("variables.ids = %v, want [PR_11 PR_12 PR_16]", ids)
	}
	for _, id := range []string{"PR_11", "PR_12", "PR_16"} {
		if strings.Contains(payload.Query, id) {
			t.Errorf("query text contains %s: ids must travel only as a variable", id)
		}
	}
	// The row's colour and bucket still come from the rollup: PR 16 is failing.
	for _, pr := range snap.PRs {
		if pr.Ref.Number == 16 && pr.CI != domain.CIFailing {
			t.Errorf("PR 16 CI = %v, want failing even when only a cancelled check is named", pr.CI)
		}
	}
	// Cost is summed over both queries (1 + 1); limit and remaining come from
	// the step-2 response, because status.json carries no limit.
	if snap.RateLimit.Cost != 2 || snap.RateLimit.Limit != 5000 || snap.RateLimit.Remaining != 4987 {
		t.Errorf("RateLimit = %+v, want cost 2, limit 5000, remaining 4987 from the step-2 response", snap.RateLimit)
	}
}

// TestFailingCleansNames calls mapContext directly, bypassing go-gh's own
// sanitiser, so the mapper's domain.Clean calls are what get tested.
func TestFailingCleansNames(t *testing.T) {
	run := mapContext(&wireContext{
		Typename: "CheckRun", Name: "li\x1b[31mnt‮", Status: "COMPLETED", Conclusion: "FAILURE", DatabaseID: 3,
		CheckSuite: &wireSuite{
			CreatedAt: time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC),
			App:       &struct{ Name string }{"App\x1b[1m‭"},
			WorkflowRun: &struct {
				Workflow *struct{ Name string }
			}{Workflow: &struct{ Name string }{"c\x1b[0mi⁦"}},
		},
	})
	if run.Name != "lint" || run.Workflow != "ci" {
		t.Errorf("check run Name/Workflow = %q/%q, want lint/ci", run.Name, run.Workflow)
	}
	appOnly := mapContext(&wireContext{
		Typename: "CheckRun", Name: "x", Status: "COMPLETED", Conclusion: "SUCCESS",
		CheckSuite: &wireSuite{App: &struct{ Name string }{"App\x1b[1m‭"}},
	})
	if appOnly.Workflow != "App" {
		t.Errorf("Workflow without a workflow run = %q, want the app name %q", appOnly.Workflow, "App")
	}
	status := mapContext(&wireContext{Typename: "StatusContext", Context: "ci/je\x1b[2Jnkins‮", State: "ERROR"})
	if status.Name != "ci/jenkins" || !status.Context || status.State != domain.CheckFailing {
		t.Errorf("status context = %+v, want Name ci/jenkins, Context true, failing", status)
	}
	for _, s := range []string{run.Name, run.Workflow, appOnly.Workflow, status.Name} {
		if why := noUnsafe(s); why != "" {
			t.Errorf("mapped string %q has a %s", s, why)
		}
	}
}

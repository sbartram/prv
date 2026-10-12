package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/sbartram/prv/internal/domain"
)

// fetchDetailFixture runs FetchDetail for PR_11 over the given Detail fixtures.
func fetchDetailFixture(t *testing.T, files ...string) (domain.Detail, error, *fixtureTransport) {
	t.Helper()
	c, tr := newFixtureClient(t)
	tr.detail = files
	d, err := c.FetchDetail(context.Background(), "PR_11")
	return d, err, tr
}

// requestVariables decodes the variables of one recorded request body.
func requestVariables(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var payload struct {
		Variables map[string]any `json:"variables"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Variables
}

func TestFetchDetail(t *testing.T) {
	d, err, tr := fetchDetailFixture(t, "detail_basic.json")
	if err != nil {
		t.Fatalf("FetchDetail: %v", err)
	}
	if len(tr.detailBodies) != 1 {
		t.Fatalf("Detail requests = %d, want 1", len(tr.detailBodies))
	}
	if vars := requestVariables(t, tr.detailBodies[0]); len(vars) != 1 || vars["id"] != "PR_11" {
		t.Errorf("variables = %v, want only id PR_11 (no after)", vars)
	}

	wantChecks := []string{"lint", "test (ubuntu)", "deploy", "e2e", "build", "ci/circleci", "codecov/patch", "docs"}
	var gotChecks []string
	sources := map[string]string{}
	for _, c := range d.Checks {
		gotChecks = append(gotChecks, c.Name)
		sources[c.Name] = c.Source
	}
	if !slices.Equal(gotChecks, wantChecks) {
		t.Errorf("checks = %q, want %q", gotChecks, wantChecks)
	}
	for _, c := range d.Checks {
		if c.Name == "build" && c.State != domain.CheckPassing {
			t.Errorf("build state = %v, want passing: the newer run replaces the failure", c.State)
		}
	}
	wantSources := map[string]string{
		"lint": "ci.yml", "test (ubuntu)": "ci.yml", "deploy": "release.yml", "e2e": "ci.yml",
		"build": "ci.yml", "docs": "ci.yml", "codecov/patch": "Codecov", "ci/circleci": "",
	}
	for name, want := range wantSources {
		if sources[name] != want {
			t.Errorf("Source of %q = %q, want %q", name, sources[name], want)
		}
	}

	type rev struct {
		name  string
		state domain.ReviewerState
	}
	wantRev := []rev{
		{"bob", domain.ReviewerChangesRequested},
		{"carol", domain.ReviewerPending},
		{"", domain.ReviewerPending},
		{"", domain.ReviewerPending},
		{"alice", domain.ReviewerApproved},
		{"dave", domain.ReviewerCommented},
		{"erin", domain.ReviewerDismissed},
	}
	var gotRev []rev
	for _, r := range d.Reviewers {
		gotRev = append(gotRev, rev{r.Name, r.State})
	}
	if !slices.Equal(gotRev, wantRev) {
		t.Errorf("reviewers = %v, want %v", gotRev, wantRev)
	}

	if want := (domain.CheckCounts{Failing: 2, Cancelled: 1, Running: 1, Passing: 3, Neutral: 1}); d.Counts != want {
		t.Errorf("Counts = %+v, want %+v", d.Counts, want)
	}
	if d.Truncated || d.Read != 9 || d.Total != 9 {
		t.Errorf("Truncated/Read/Total = %v/%d/%d, want false/9/9", d.Truncated, d.Read, d.Total)
	}
}

func TestFetchDetailNotFound(t *testing.T) {
	for name, body := range map[string]string{
		"null node":          `{"data":{"node":null}}`,
		"not a pull request": `{"data":{"node":{"__typename":"Issue","id":"PR_11"}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			c := clientWith(t, rtFunc(func(req *http.Request) (*http.Response, error) {
				return jsonResp(req, 200, []byte(body)), nil
			}))
			_, err := c.FetchDetail(context.Background(), "PR_11")
			var ge *Error
			if !errors.As(err, &ge) || ge.Kind != KindGraphQL {
				t.Fatalf("err = %v, want *Error of KindGraphQL", err)
			}
			if ge.Message != "this pull request could not be loaded" {
				t.Errorf("Message = %q", ge.Message)
			}
		})
	}
}

func TestFetchDetailRateLimited(t *testing.T) {
	rt := headerTransport(200, rateLimitedBody, http.Header{"X-Ratelimit-Remaining": {"0"}})
	_, err := clientWith(t, rt).FetchDetail(context.Background(), "PR_11")
	var ge *Error
	if !errors.As(err, &ge) || ge.Kind != KindRateLimited {
		t.Fatalf("err = %v, want KindRateLimited", err)
	}
}

// hostileDetail holds a hostile name in every remote string of a Detail node:
// ESC, BEL, tab and a bidi override.
const hostileDetail = `{"data":{"node":{
  "__typename":"PullRequest","id":"PR_11",
  "latestOpinionatedReviews":{"nodes":[{"state":"APPROVED","author":{"__typename":"User","login":"rev\u202eiewer\u001b[2J"}}]},
  "latestReviews":{"nodes":[{"state":"COMMENTED","author":{"__typename":"User","login":"com\u0007menter"}}]},
  "reviewRequests":{"nodes":[{"requestedReviewer":{"__typename":"Team","combinedSlug":"acme/te\u202eam\u001b[31m"}}]},
  "statusCheckRollup":{"state":"SUCCESS","contexts":{"totalCount":3,"pageInfo":{"hasNextPage":false,"endCursor":""},"nodes":[
    {"__typename":"CheckRun","name":"bu\u001b[31mild\u202e\tx","status":"COMPLETED","conclusion":"SUCCESS","databaseId":1,
     "checkSuite":{"createdAt":"2026-10-11T08:00:00Z","app":{"name":"app\u202e"},
       "workflowRun":{"workflow":{"name":"wf\u0007"},"file":{"path":".github/workflows/e\u202evil\u001b[31m.yml"}}}},
    {"__typename":"CheckRun","name":"app-check","status":"COMPLETED","conclusion":"SUCCESS","databaseId":2,
     "checkSuite":{"createdAt":"2026-10-11T08:00:00Z","app":{"name":"Co\u202edecov\u001b[0m"},"workflowRun":null}},
    {"__typename":"StatusContext","context":"ci/\u202ecircle\u0007","state":"SUCCESS"}
  ]}}
}}}`

// TestFetchDetailCleans decodes hostile JSON directly, bypassing go-gh's own
// sanitiser (which would already hide an ESC), so the mapper's domain.Clean
// calls are what get tested.
func TestFetchDetailCleans(t *testing.T) {
	var env struct {
		Data wireDetailResp `json:"data"`
	}
	if err := json.Unmarshal([]byte(hostileDetail), &env); err != nil {
		t.Fatal(err)
	}
	node := env.Data.Node
	if node == nil || node.StatusCheckRollup == nil {
		t.Fatal("hostile fixture did not decode")
	}
	var checks []domain.Check
	for _, c := range node.StatusCheckRollup.Contexts.Nodes {
		checks = append(checks, mapContext(c))
	}
	reviewers := detailReviewers(node)

	var all []string
	for _, c := range checks {
		all = append(all, c.Name, c.Workflow, c.Source)
	}
	for _, r := range reviewers {
		all = append(all, r.Name)
	}
	if len(checks) != 3 || len(reviewers) != 3 {
		t.Fatalf("checks/reviewers = %d/%d, want 3/3", len(checks), len(reviewers))
	}
	for _, s := range all {
		if why := noUnsafe(s); why != "" {
			t.Errorf("mapped string %q has a %s", s, why)
		}
	}
	if checks[0].Source != "evil.yml" {
		t.Errorf("workflow file Source = %q, want the cleaned base name", checks[0].Source)
	}
	if checks[1].Source != "Codecov" {
		t.Errorf("app Source = %q, want the cleaned app name", checks[1].Source)
	}
}

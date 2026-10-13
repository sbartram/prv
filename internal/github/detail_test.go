package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// builtPage returns a Detail response body with n unique passing check runs
// named "<prefix>-<i>", plus the extra contexts given.
func builtPage(t *testing.T, prefix string, n, total int, hasNext bool, cursor string, extra ...map[string]any) []byte {
	t.Helper()
	suite := map[string]any{
		"createdAt":   "2026-10-11T08:00:00Z",
		"app":         map[string]any{"name": "GitHub Actions"},
		"workflowRun": map[string]any{"workflow": map[string]any{"name": "ci"}, "file": map[string]any{"path": ".github/workflows/ci.yml"}},
	}
	var nodes []map[string]any
	for i := 0; i < n-len(extra); i++ {
		nodes = append(nodes, map[string]any{
			"__typename": "CheckRun", "name": fmt.Sprintf("%s-%03d", prefix, i), "status": "COMPLETED",
			"conclusion": "SUCCESS", "databaseId": 1000 + i, "checkSuite": suite,
		})
	}
	nodes = append(nodes, extra...)
	body, err := json.Marshal(map[string]any{"data": map[string]any{"node": map[string]any{
		"__typename": "PullRequest", "id": "PR_11",
		"latestOpinionatedReviews": map[string]any{"nodes": []any{}},
		"latestReviews":            map[string]any{"nodes": []any{}},
		"reviewRequests":           map[string]any{"nodes": []any{}},
		"statusCheckRollup": map[string]any{"state": "FAILURE", "contexts": map[string]any{
			"totalCount": total,
			"pageInfo":   map[string]any{"hasNextPage": hasNext, "endCursor": cursor},
			"nodes":      nodes,
		}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// dupRun is a check run named "dup" from the check suite created at created.
func dupRun(conclusion, created string, id int) map[string]any {
	return map[string]any{
		"__typename": "CheckRun", "name": "dup", "status": "COMPLETED", "conclusion": conclusion, "databaseId": id,
		"checkSuite": map[string]any{
			"createdAt":   created,
			"app":         map[string]any{"name": "GitHub Actions"},
			"workflowRun": map[string]any{"workflow": map[string]any{"name": "ci"}, "file": map[string]any{"path": ".github/workflows/ci.yml"}},
		},
	}
}

// pagedTransport serves Detail page n (1-based) from answer and records the
// "after" variable of every request.
func pagedTransport(t *testing.T, afters *[]string, answer func(req *http.Request, n int) *http.Response) http.RoundTripper {
	return rtFunc(func(req *http.Request) (*http.Response, error) {
		raw, err := readBody(req)
		if err != nil {
			t.Fatal(err)
		}
		after, _ := requestVariables(t, raw)["after"].(string)
		*afters = append(*afters, after)
		return answer(req, len(*afters)), nil
	})
}

func TestFetchDetailPaging(t *testing.T) {
	t.Run("two pages", func(t *testing.T) {
		d, err, tr := fetchDetailFixture(t, "detail_page1.json", "detail_page2.json")
		if err != nil {
			t.Fatal(err)
		}
		if len(tr.detailBodies) != 2 {
			t.Fatalf("Detail requests = %d, want 2", len(tr.detailBodies))
		}
		if v := requestVariables(t, tr.detailBodies[0]); v["after"] != nil {
			t.Errorf("first request variables = %v, want no after", v)
		}
		if v := requestVariables(t, tr.detailBodies[1]); v["after"] != "CUR1" || v["id"] != "PR_11" {
			t.Errorf("second request variables = %v, want id PR_11 and after CUR1", v)
		}
		builds := 0
		for _, c := range d.Checks {
			if c.Name == "build" {
				builds++
				if c.State != domain.CheckPassing {
					t.Errorf("build state = %v, want passing: the newer run on page 2 wins", c.State)
				}
			}
		}
		if builds != 1 || len(d.Checks) != 149 {
			t.Errorf("builds/checks = %d/%d, want 1/149", builds, len(d.Checks))
		}
		if d.Counts.Failing != 0 || d.Counts.Passing != 149 {
			t.Errorf("Counts = %+v, want 0 failing and 149 passing", d.Counts)
		}
		if d.Truncated || d.Read != 150 || d.Total != 150 {
			t.Errorf("Truncated/Read/Total = %v/%d/%d, want false/150/150", d.Truncated, d.Read, d.Total)
		}
	})

	t.Run("five page cap", func(t *testing.T) {
		var afters []string
		c := clientWith(t, pagedTransport(t, &afters, func(req *http.Request, n int) *http.Response {
			var extra []map[string]any
			switch n {
			case 1:
				extra = append(extra, dupRun("FAILURE", "2026-10-11T07:00:00Z", 7))
			case 5:
				extra = append(extra, dupRun("SUCCESS", "2026-10-11T09:00:00Z", 9))
			}
			return jsonResp(req, 200, builtPage(t, fmt.Sprintf("p%d", n), 100, 731, true, fmt.Sprintf("CUR%d", n), extra...))
		}))
		d, err := c.FetchDetail(context.Background(), "PR_11")
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"", "CUR1", "CUR2", "CUR3", "CUR4"}; !slices.Equal(afters, want) {
			t.Errorf("after cursors = %q, want %q (exactly five requests)", afters, want)
		}
		if !d.Truncated || d.Read != 500 || d.Total != 731 {
			t.Errorf("Truncated/Read/Total = %v/%d/%d, want true/500/731", d.Truncated, d.Read, d.Total)
		}
		if d.Counts != domain.CountChecks(d.Checks) {
			t.Errorf("Counts = %+v, want CountChecks(Checks) = %+v", d.Counts, domain.CountChecks(d.Checks))
		}
		if d.Counts.Failing != 0 || len(d.Checks) != 499 {
			t.Errorf("failing/checks = %d/%d, want 0/499: the failure on page 1 is superseded on page 5", d.Counts.Failing, len(d.Checks))
		}
	})

	t.Run("later page fails with 502", func(t *testing.T) {
		var afters []string
		page1 := fixtureBytes(t, "detail_page1.json")
		c := clientWith(t, pagedTransport(t, &afters, func(req *http.Request, n int) *http.Response {
			if n == 1 {
				return jsonResp(req, 200, page1)
			}
			return jsonResp(req, 502, []byte(`{"message":"Bad Gateway"}`))
		}))
		d, err := c.FetchDetail(context.Background(), "PR_11")
		if err != nil {
			t.Fatalf("err = %v, want nil: the earlier page is kept", err)
		}
		if len(afters) != 2 {
			t.Errorf("requests = %d, want 2 (paging stops at the failure)", len(afters))
		}
		if !d.Truncated || d.Read != 100 || d.Total != 150 {
			t.Errorf("Truncated/Read/Total = %v/%d/%d, want true/100/150", d.Truncated, d.Read, d.Total)
		}
		if len(d.Checks) != 100 || d.Counts.Failing != 1 {
			t.Errorf("checks/failing = %d/%d, want 100/1 from page 1 alone", len(d.Checks), d.Counts.Failing)
		}
	})

	t.Run("later page rate limited", func(t *testing.T) {
		var afters []string
		page1 := fixtureBytes(t, "detail_page1.json")
		c := clientWith(t, pagedTransport(t, &afters, func(req *http.Request, n int) *http.Response {
			if n == 1 {
				return jsonResp(req, 200, page1)
			}
			resp := jsonResp(req, 200, []byte(rateLimitedBody))
			resp.Header.Set("X-RateLimit-Remaining", "0")
			return resp
		}))
		_, err := c.FetchDetail(context.Background(), "PR_11")
		var ge *Error
		if !errors.As(err, &ge) || ge.Kind != KindRateLimited {
			t.Fatalf("err = %v, want KindRateLimited", err)
		}
	})
}

// TestFetchDetailPartial: a Detail read that GraphQL cut short is marked
// Truncated, so the summary says how much was read (G-03-6-WR02, D-03, D-08).
func TestFetchDetailPartial(t *testing.T) {
	t.Run("a null context node", func(t *testing.T) {
		c := clientWith(t, rtFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResp(req, 200, builtPage(t, "p", 3, 3, false, "", nil)), nil
		}))
		d, err := c.FetchDetail(context.Background(), "PR_11")
		if err != nil {
			t.Fatalf("FetchDetail: %v", err)
		}
		if len(d.Checks) != 2 || d.Read != 2 || d.Total != 3 || !d.Truncated {
			t.Errorf("checks/Read/Total/Truncated = %d/%d/%d/%v, want 2/2/3/true: the null node is a run that was not read",
				len(d.Checks), d.Read, d.Total, d.Truncated)
		}
	})
}

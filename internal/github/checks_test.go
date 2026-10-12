package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/sbartram/prv/internal/domain"
)

// readBody returns the request body and puts it back, so a wrapped transport
// can read it again.
func readBody(req *http.Request) ([]byte, error) {
	raw, err := io.ReadAll(req.Body)
	req.Body = io.NopCloser(bytes.NewReader(raw))
	return raw, err
}

// fetchWith runs one fetch over status.json (PR_11, PR_12 and PR_16 fail) with
// the given FailingChecks fixtures.
func fetchWith(t *testing.T, failing ...string) (domain.Snapshot, map[int]domain.PR, *fixtureTransport) {
	t.Helper()
	c, tr := newFixtureClient(t, "status.json")
	tr.failing = failing
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	byNum := map[int]domain.PR{}
	for _, pr := range snap.PRs {
		byNum[pr.Ref.Number] = pr
	}
	return snap, byNum, tr
}

// TestFailingRerunFixture is the vscode shape (RESEARCH F1): the older suite's
// cancelled jobs were superseded by a newer suite, so the row reads one failing
// check with no "+N", although the raw contexts hold 7 failing-or-cancelled runs.
func TestFailingRerunFixture(t *testing.T) {
	_, prs, _ := fetchWith(t, "failing_rerun.json")
	want := domain.FailingCheck{Known: true, Name: "Linux / test"}
	if got := prs[11].Failing; got != want {
		t.Errorf("PR 11 Failing = %+v, want %+v", got, want)
	}
	// PR 16's only failure was superseded by a passing re-run: no name, even
	// though the discovery rollup still reads FAILURE.
	if got := prs[16].Failing; got != (domain.FailingCheck{}) {
		t.Errorf("PR 16 Failing = %+v, want unnamed: its failed run was superseded", got)
	}

	// Guard the fixture itself: the raw contexts really hold 7 failing or
	// cancelled runs, so the "+0" above is the de-duplication at work.
	var env struct {
		Data wireFailResp `json:"data"`
	}
	if err := json.Unmarshal(fixtureBytes(t, "failing_rerun.json"), &env); err != nil {
		t.Fatal(err)
	}
	raw := 0
	for _, n := range env.Data.Nodes[0].StatusCheckRollup.Contexts.Nodes {
		switch mapContext(n).State {
		case domain.CheckFailing, domain.CheckCancelled:
			raw++
		}
	}
	if raw != 7 {
		t.Errorf("raw failing-or-cancelled runs in the fixture = %d, want 7", raw)
	}
}

// TestFailingTruncatedUnnamed: a contexts page that says hasNextPage never
// gives a name, because a re-run on the missing page may have fixed the check.
func TestFailingTruncatedUnnamed(t *testing.T) {
	_, prs, _ := fetchWith(t, "failing_truncated.json")
	if got := prs[11].Failing; got != (domain.FailingCheck{}) {
		t.Errorf("PR 11 Failing = %+v, want the zero value for a truncated page", got)
	}
	if got := ciCellText(prs[11]); got != "✗ failing" {
		t.Errorf("PR 11 CI badge = %q, want %q", got, "✗ failing")
	}
}

// ciCellText is the plain badge a row shows when no name is known.
func ciCellText(pr domain.PR) string {
	if pr.Failing.Known {
		return "✗ " + pr.Failing.Name
	}
	return pr.CI.Badge()
}

// TestFailingPartialErrors: GitHub nulls one node and reports it in errors[]
// next to data. The node that loaded is named, the others stay unnamed, and the
// discovery rows stay on screen with one warning.
func TestFailingPartialErrors(t *testing.T) {
	snap, prs, _ := fetchWith(t, "failing_partial.json")
	if got := prs[11].Failing; got != (domain.FailingCheck{}) {
		t.Errorf("PR 11 Failing = %+v, want unnamed (its node was null)", got)
	}
	if got := prs[16].Failing; got != (domain.FailingCheck{}) {
		t.Errorf("PR 16 Failing = %+v, want unnamed (not in the response)", got)
	}
	if got, want := prs[12].Failing, (domain.FailingCheck{Known: true, Name: "ci/jenkins"}); got != want {
		t.Errorf("PR 12 Failing = %+v, want %+v", got, want)
	}
	if !slices.Contains(snap.Warnings, "Check names could not be loaded") {
		t.Errorf("Warnings = %q, want it to contain %q", snap.Warnings, "Check names could not be loaded")
	}
	if len(snap.PRs) < 17 {
		t.Errorf("PRs = %d: the discovery rows must stay on screen", len(snap.PRs))
	}
}

// manyBodies builds discovery pages for n open PRs (ids PR_1..PR_n, every
// rollup FAILURE): pages of 100, hasNextPage true until the last.
func manyBodies(n int) [][]byte {
	var pages [][]byte
	for start := 1; start <= n; start += 100 {
		end := min(start+99, n)
		var nodes []string
		for i := start; i <= end; i++ {
			nodes = append(nodes, fmt.Sprintf(
				`{"__typename":"PullRequest","id":"PR_%[1]d","number":%[1]d,"title":"t%[1]d","url":"https://github.com/acme/many/pull/%[1]d",`+
					`"isDraft":false,"state":"OPEN","closedAt":null,"mergedAt":null,"createdAt":"2026-10-10T00:00:00Z","updatedAt":"2026-10-11T08:00:00Z",`+
					`"repository":{"nameWithOwner":"acme/many"},"author":{"login":"octo"},"reviewDecision":null,`+
					`"latestOpinionatedReviews":{"nodes":[]},"statusCheckRollup":{"state":"FAILURE"}}`, i))
		}
		more := end < n
		pages = append(pages, fmt.Appendf(nil,
			`{"data":{"viewer":{"login":"octo"},"rateLimit":{"limit":5000,"cost":1,"remaining":4990,"resetAt":"2026-10-11T13:00:00Z"},`+
				`"search":{"issueCount":%d,"pageInfo":{"hasNextPage":%t,"endCursor":"C%d"},"nodes":[%s]}}}`,
			n, more, end, strings.Join(nodes, ",")))
	}
	return pages
}

// failingFor answers a FailingChecks request: every requested id gets one
// FAILURE context named "lint". The ids of each request are appended to *got.
// remaining is the rateLimit.remaining of the first answer; each later answer
// reports one less. A negative remaining leaves the rateLimit block out.
func failingFor(got *[][]string, remaining int) func(body []byte) []byte {
	return func(body []byte) []byte {
		var payload struct {
			Variables struct {
				IDs []string `json:"ids"`
			} `json:"variables"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			panic(err)
		}
		ids := payload.Variables.IDs
		*got = append(*got, ids)
		var nodes []string
		for i, id := range ids {
			nodes = append(nodes, fmt.Sprintf(
				`{"__typename":"PullRequest","id":%q,"statusCheckRollup":{"state":"FAILURE","contexts":{"totalCount":1,"pageInfo":{"hasNextPage":false},`+
					`"nodes":[{"__typename":"CheckRun","name":"lint","status":"COMPLETED","conclusion":"FAILURE","databaseId":%d,`+
					`"checkSuite":{"createdAt":"2026-10-11T09:00:00Z","app":{"name":"GitHub Actions"},"workflowRun":{"workflow":{"name":"ci"}}}}]}}}`,
				id, i+1))
		}
		rl := ""
		if remaining >= 0 {
			rl = fmt.Sprintf(`"rateLimit":{"limit":5000,"cost":1,"remaining":%d,"resetAt":"2026-10-11T14:00:00Z"},`, remaining-(len(*got)-1))
		}
		return fmt.Appendf(nil, `{"data":{%s"nodes":[%s]}}`, rl, strings.Join(nodes, ","))
	}
}

// manyTransport serves manyBodies for "Watch" and answer(body) for
// "FailingChecks"; a nil answer means HTTP 502.
func manyTransport(t *testing.T, pages [][]byte, answer func([]byte) []byte) http.RoundTripper {
	t.Helper()
	watch := 0
	return rtFunc(func(req *http.Request) (*http.Response, error) {
		raw, _ := readBody(req)
		switch op := opName(raw); op {
		case "Watch":
			if watch >= len(pages) {
				t.Fatalf("unexpected discovery request %d", watch+1)
			}
			watch++
			return jsonResp(req, 200, pages[watch-1]), nil
		case "FailingChecks":
			if answer == nil {
				return jsonResp(req, 502, []byte(`{"message":"Bad Gateway sk-upstream-detail"}`)), nil
			}
			return jsonResp(req, 200, answer(raw)), nil
		default:
			t.Fatalf("unexpected operation %q", op)
			return nil, nil
		}
	})
}

// TestFailingChunks: every failing PR is looked up, 25 ids per request, in
// discovery order, and every one is named (D-01, LIST-07).
func TestFailingChunks(t *testing.T) {
	tests := []struct {
		name   string
		n      int
		chunks []int
	}{
		{"30 failing PRs, one page", 30, []int{25, 5}},
		{"120 failing PRs, two pages", 120, []int{25, 25, 25, 25, 20}},
		{"exactly one full chunk", 25, []int{25}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got [][]string
			rt := manyTransport(t, manyBodies(tt.n), failingFor(&got, 4980))
			snap, err := clientWith(t, rt).Fetch(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			var sizes []int
			var flat []string
			for _, ids := range got {
				sizes = append(sizes, len(ids))
				flat = append(flat, ids...)
			}
			if !slices.Equal(sizes, tt.chunks) {
				t.Errorf("FailingChecks request sizes = %v, want %v", sizes, tt.chunks)
			}
			for i, id := range flat {
				if want := fmt.Sprintf("PR_%d", i+1); id != want {
					t.Fatalf("id %d = %s, want %s: ids must follow discovery order", i, id, want)
				}
			}
			if len(snap.PRs) != tt.n {
				t.Fatalf("PRs = %d, want %d", len(snap.PRs), tt.n)
			}
			for _, pr := range snap.PRs {
				if want := (domain.FailingCheck{Known: true, Name: "lint"}); pr.Failing != want {
					t.Errorf("%s Failing = %+v, want %+v", pr.Ref, pr.Failing, want)
				}
			}
		})
	}
}

// TestFailingStepRateLimited: a rate-limited FailingChecks request fails the
// whole refresh like a rate-limited discovery page (02-07): the error comes
// back with an empty snapshot, so the previous rows stay and the model pauses.
func TestFailingStepRateLimited(t *testing.T) {
	status := fixtureBytes(t, "status.json")
	rt := rtFunc(func(req *http.Request) (*http.Response, error) {
		raw, _ := readBody(req)
		if opName(raw) == "FailingChecks" {
			resp := jsonResp(req, 200, []byte(rateLimitedBody))
			resp.Header.Set("X-RateLimit-Remaining", "0")
			return resp, nil
		}
		return jsonResp(req, 200, status), nil
	})
	snap, err := clientWith(t, rt).Fetch(context.Background())
	var ge *Error
	if !errors.As(err, &ge) || ge.Kind != KindRateLimited {
		t.Fatalf("err = %v, want *Error of KindRateLimited", err)
	}
	if len(snap.PRs) != 0 || len(snap.Warnings) != 0 || snap.Truncated {
		t.Errorf("snapshot = %+v, want empty", snap)
	}
}

// TestFailingStepServerError: an HTTP 502 on the first of two chunks leaves the
// discovery rows, names nothing, adds one fixed warning and sends no later chunk.
func TestFailingStepServerError(t *testing.T) {
	rt := manyTransport(t, manyBodies(30), nil)
	var failBodies int
	counting := rtFunc(func(req *http.Request) (*http.Response, error) {
		raw, _ := readBody(req)
		if opName(raw) == "FailingChecks" {
			failBodies++
		}
		return rt.RoundTrip(req)
	})
	snap, err := clientWith(t, counting).Fetch(context.Background())
	if err != nil {
		t.Fatalf("err = %v, want nil: the discovery rows stay", err)
	}
	if failBodies != 1 {
		t.Errorf("FailingChecks requests = %d, want exactly 1 (the second chunk is not sent)", failBodies)
	}
	if len(snap.PRs) != 30 {
		t.Errorf("PRs = %d, want all 30 discovery rows", len(snap.PRs))
	}
	for _, pr := range snap.PRs {
		if pr.Failing != (domain.FailingCheck{}) {
			t.Errorf("%s Failing = %+v, want unnamed", pr.Ref, pr.Failing)
		}
	}
	if len(snap.Warnings) != 1 || snap.Warnings[0] != "Check names could not be loaded" {
		t.Fatalf("Warnings = %q, want exactly the fixed warning", snap.Warnings)
	}
	for _, leak := range []string{secretToken, "NEVER-SHOW", "Bad Gateway", "sk-upstream-detail", "502"} {
		if strings.Contains(snap.Warnings[0], leak) {
			t.Errorf("warning %q contains %q", snap.Warnings[0], leak)
		}
	}
}

// TestFailingStepCanceled: a cancelled FailingChecks request means prv is
// quitting; the cancellation comes back unchanged.
func TestFailingStepCanceled(t *testing.T) {
	status := fixtureBytes(t, "status.json")
	rt := rtFunc(func(req *http.Request) (*http.Response, error) {
		raw, _ := readBody(req)
		if opName(raw) == "FailingChecks" {
			return nil, context.Canceled
		}
		return jsonResp(req, 200, status), nil
	})
	snap, err := clientWith(t, rt).Fetch(context.Background())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want it to wrap context.Canceled", err)
	}
	var ge *Error
	if errors.As(err, &ge) {
		t.Errorf("a cancelled fetch must not be classified as %v", ge.Kind)
	}
	if len(snap.PRs) != 0 {
		t.Errorf("PRs = %d, want an empty snapshot", len(snap.PRs))
	}
}

// TestFailingCostSummed: Cost adds the discovery pages and every FailingChecks
// request; Limit, Remaining and ResetAt come from the last response that
// carried a rate limit (RFSH-05).
func TestFailingCostSummed(t *testing.T) {
	t.Run("status.json has no limit: step 2 supplies it", func(t *testing.T) {
		snap, _, _ := fetchWith(t, "failing_contexts.json")
		rl := snap.RateLimit
		if rl.Cost != 2 || rl.Limit != 5000 || rl.Remaining != 4987 {
			t.Errorf("RateLimit = %+v, want cost 2, limit 5000, remaining 4987", rl)
		}
	})
	t.Run("two pages and five chunks", func(t *testing.T) {
		var got [][]string
		rt := manyTransport(t, manyBodies(120), failingFor(&got, 4980))
		snap, err := clientWith(t, rt).Fetch(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		rl := snap.RateLimit
		// 2 discovery pages + 5 chunks, cost 1 each; the last chunk reports 4980-4.
		if rl.Cost != 7 || rl.Limit != 5000 || rl.Remaining != 4976 {
			t.Errorf("RateLimit = %+v, want cost 7, limit 5000, remaining 4976", rl)
		}
		if got := rl.ResetAt.UTC().Hour(); got != 14 {
			t.Errorf("ResetAt = %v, want the last response's 14:00 reset", rl.ResetAt)
		}
	})
	t.Run("a step-2 answer without a rateLimit block keeps the discovery limit", func(t *testing.T) {
		var got [][]string
		rt := manyTransport(t, manyBodies(30), failingFor(&got, -1))
		snap, err := clientWith(t, rt).Fetch(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		rl := snap.RateLimit
		if rl.Cost != 1 || rl.Limit != 5000 || rl.Remaining != 4990 {
			t.Errorf("RateLimit = %+v, want cost 1, limit 5000, remaining 4990 from discovery", rl)
		}
	})
}

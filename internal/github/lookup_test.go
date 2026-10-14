package github

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/sbartram/prv/internal/domain"
)

// fixedRefs is a WatchSource that returns a fixed list.
type fixedRefs []domain.Ref

func (f fixedRefs) Load() ([]domain.Ref, []string, error) { return slices.Clone(f), nil, nil }

// watchClient builds a client over rt whose saved list is refs.
func watchClient(t *testing.T, rt http.RoundTripper, refs []domain.Ref) *Client {
	t.Helper()
	c, err := NewClient(Options{
		Token:      secretToken,
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rt,
		Watch:      fixedRefs(refs),
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// mixedRefs are the nine refs that lookup_mixed.json answers, in alias order.
func mixedRefs() []domain.Ref {
	return []domain.Ref{
		{Repo: "acme/api", Number: 1}, {Repo: "ghost-org/nope", Number: 2}, {Repo: "acme/api", Number: 99999999},
		{Repo: "samlcorp/app", Number: 3}, {Repo: "samlcorp/web", Number: 4}, {Repo: "acme/private", Number: 5},
		{Repo: "acme/api", Number: 6}, {Repo: "acme/api", Number: 7}, {Repo: "acme/api", Number: 8},
	}
}

// lookupRequest splits a request body into its query text and variables.
func lookupRequest(t *testing.T, raw []byte) (string, map[string]any) {
	t.Helper()
	var p struct {
		Query     string
		Variables map[string]any
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	return p.Query, p.Variables
}

// lookupPRNode is the JSON of a loaded pull request in a Lookup answer.
func lookupPRNode(title, nameWithOwner, id string, number int, rollup string) map[string]any {
	return map[string]any{"pullRequest": map[string]any{
		"id": id, "number": number, "title": title,
		"url":       "https://github.com/" + nameWithOwner + "/pull/" + strconv.Itoa(number),
		"isDraft":   false,
		"state":     "OPEN",
		"createdAt": "2026-10-05T00:00:00Z", "closedAt": nil, "mergedAt": nil, "updatedAt": "2026-10-11T11:00:00Z",
		"repository":               map[string]any{"nameWithOwner": nameWithOwner},
		"author":                   map[string]any{"login": "hubot"},
		"reviewDecision":           nil,
		"mergeStateStatus":         "CLEAN",
		"mergeable":                "MERGEABLE",
		"latestOpinionatedReviews": map[string]any{"nodes": []any{}},
		"reviewRequests":           map[string]any{"nodes": []any{}},
		"statusCheckRollup":        map[string]any{"state": rollup},
	}}
}

// lookupBody encodes a Lookup answer: a rateLimit block with cost 1, the
// aliases and errors[] when errs is not empty.
func lookupBody(t *testing.T, aliases map[string]any, errs []map[string]any) []byte {
	t.Helper()
	data := map[string]any{"rateLimit": map[string]any{
		"limit": 5000, "cost": 1, "remaining": 4980, "resetAt": "2026-10-11T13:00:00Z"}}
	for k, v := range aliases {
		data[k] = v
	}
	body := map[string]any{"data": data}
	if len(errs) > 0 {
		body["errors"] = errs
	}
	out, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestBuildLookupVariablesOnly(t *testing.T) {
	q := buildLookup(3)
	for _, want := range []string{
		"$o0: String!", "$n0: String!", "$p0: Int!", "$o2: String!", "$n2: String!", "$p2: Int!",
		"r0: repository(owner: $o0, name: $n0)", "r1: repository(owner: $o1, name: $n1)", "r2: repository(owner: $o2, name: $n2)",
		"pullRequest(number: $p2)",
	} {
		if !strings.Contains(q, want) {
			t.Errorf("buildLookup(3) lacks %q:\n%s", want, q)
		}
	}
	if strings.Contains(q, "r3:") || strings.Contains(q, "$o3") {
		t.Errorf("buildLookup(3) holds a fourth alias:\n%s", q)
	}

	var raw []byte
	c := clientWith(t, rtFunc(func(req *http.Request) (*http.Response, error) {
		raw, _ = io.ReadAll(req.Body)
		return jsonResp(req, 200, lookupBody(t, map[string]any{"r0": nil}, nil)), nil
	}))
	var snap domain.Snapshot
	if _, err := c.lookupRefs(context.Background(), []domain.Ref{{Repo: "zzowner/qqrepo", Number: 987654}}, &snap); err != nil {
		t.Fatal(err)
	}
	query, vars := lookupRequest(t, raw)
	for _, bad := range []string{"zzowner", "qqrepo", "987654"} {
		if strings.Contains(query, bad) {
			t.Errorf("the Lookup query text holds %q: owner, repo and number travel only as variables", bad)
		}
	}
	if len(vars) != 3 || vars["o0"] != "zzowner" || vars["n0"] != "qqrepo" || vars["p0"] != float64(987654) {
		t.Errorf("variables = %v, want {o0: zzowner, n0: qqrepo, p0: 987654}", vars)
	}
}

func TestLookupChunks(t *testing.T) {
	var sizes []int
	c := clientWith(t, rtFunc(func(req *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(req.Body)
		if op := opName(raw); op != "Lookup" {
			t.Errorf("unexpected operation %q", op)
		}
		_, vars := lookupRequest(t, raw)
		aliases := map[string]any{}
		n := 0
		for k := range vars {
			if strings.HasPrefix(k, "p") {
				aliases["r"+k[1:]] = nil
				n++
			}
		}
		sizes = append(sizes, n)
		return jsonResp(req, 200, lookupBody(t, aliases, nil)), nil
	}))
	var refs []domain.Ref
	for i := 0; i < 26; i++ {
		refs = append(refs, domain.Ref{Repo: "acme/r" + strconv.Itoa(i), Number: i + 1})
	}
	var snap domain.Snapshot
	prs, err := c.lookupRefs(context.Background(), refs, &snap)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sizes, []int{25, 1}) {
		t.Errorf("Lookup requests carried %v aliases, want [25 1]", sizes)
	}
	if len(prs) != 26 {
		t.Fatalf("got %d PRs, want 26", len(prs))
	}
	for i, pr := range prs {
		if pr.Ref != refs[i] {
			t.Errorf("PR %d = %v, want %v (ref order)", i, pr.Ref, refs[i])
		}
	}
	if snap.RateLimit.Cost != 2 {
		t.Errorf("rate limit cost = %d, want 2 (one per request)", snap.RateLimit.Cost)
	}
}

func TestLookupReasons(t *testing.T) {
	c, tr := newFixtureClient(t)
	tr.lookup = []string{"lookup_mixed.json"}
	refs := mixedRefs()
	var snap domain.Snapshot
	prs, err := c.lookupRefs(context.Background(), refs, &snap)
	if err != nil {
		t.Fatal(err)
	}
	if len(prs) != len(refs) {
		t.Fatalf("got %d PRs, want %d", len(prs), len(refs))
	}
	if len(tr.lookupBodies) != 1 {
		t.Errorf("Lookup requests = %d, want 1", len(tr.lookupBodies))
	}

	notFound := domain.NotFound()
	sso := domain.SSORequired("samlcorp")
	want := map[int]domain.Unavail{
		1: notFound,
		2: notFound,
		3: sso,
		4: {Kind: domain.UnavailSSO, Reason: sso.Reason, Help: sso.Help},
		5: domain.LoadFailed("forbidden", "Resource not accessible by personal access token"),
		6: domain.LoadFailed("service unavailable", "Something went wrong"),
		8: domain.LoadFailed("no data", ""),
	}
	for i, pr := range prs {
		if pr.Ref != refs[i] || pr.Watched != refs[i] || !pr.Manual {
			t.Errorf("PR %d: Ref/Watched/Manual = %v/%v/%v, want %v/%v/true", i, pr.Ref, pr.Watched, pr.Manual, refs[i], refs[i])
		}
		u, bad := want[i]
		if !bad {
			if pr.Unavail.Kind != domain.UnavailNone || pr.ID == "" {
				t.Errorf("PR %d (%v) should have loaded: ID %q, unavailable %+v", i, refs[i], pr.ID, pr.Unavail)
			}
			continue
		}
		if pr.Unavail != u {
			t.Errorf("PR %d (%v) unavailable = %+v, want %+v", i, refs[i], pr.Unavail, u)
		}
		if pr.ID != "" || pr.URL != refs[i].URL() {
			t.Errorf("PR %d (%v): ID %q, URL %q, want empty ID and %q", i, refs[i], pr.ID, pr.URL, refs[i].URL())
		}
	}
	if prs[0].ID != "PR_M1" || prs[7].ID != "PR_M7" {
		t.Errorf("loaded PR IDs = %q, %q, want PR_M1, PR_M7", prs[0].ID, prs[7].ID)
	}
	// r4 and r3 differ only in the signal: saml_failure true versus the message.
	if want := []string{"Some data could not be loaded: Resource not accessible by integration"}; !slices.Equal(snap.Warnings, want) {
		t.Errorf("warnings = %q, want %q", snap.Warnings, want)
	}
	if snap.RateLimit.Cost != 1 || snap.RateLimit.Remaining != 4980 {
		t.Errorf("rate limit = %+v, want cost 1 and remaining 4980", snap.RateLimit)
	}
}

func TestLookupRequestFailure(t *testing.T) {
	ref := []domain.Ref{{Repo: "acme/api", Number: 1}}
	tests := []struct {
		name string
		rt   rtFunc
		ctx  func() context.Context
		kind ErrorKind
		is   error
	}{
		{"HTTP 502", func(req *http.Request) (*http.Response, error) {
			return jsonResp(req, 502, []byte(`{"message":"bad gateway"}`)), nil
		}, context.Background, KindServer, nil},
		{"rate limited error", func(req *http.Request) (*http.Response, error) {
			return jsonResp(req, 200, []byte(rateLimitedBody)), nil
		}, context.Background, KindRateLimited, nil},
		{"error without data", func(req *http.Request) (*http.Response, error) {
			return jsonResp(req, 200, []byte(`{"data":null,"errors":[{"type":"FORBIDDEN","path":["r0"],"message":"nope"}]}`)), nil
		}, context.Background, KindGraphQL, nil},
		{"cancelled context", func(req *http.Request) (*http.Response, error) {
			return nil, req.Context().Err()
		}, func() context.Context {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx
		}, -1, context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := clientWith(t, tt.rt)
			var snap domain.Snapshot
			prs, err := c.lookupRefs(tt.ctx(), ref, &snap)
			if err == nil {
				t.Fatalf("lookupRefs returned PRs %v and no error, want an error", prs)
			}
			if prs != nil {
				t.Errorf("lookupRefs returned %d PRs with the error, want none", len(prs))
			}
			if tt.is != nil {
				if !errors.Is(err, tt.is) {
					t.Errorf("error = %v, want errors.Is %v", err, tt.is)
				}
				return
			}
			var ce *Error
			if !errors.As(err, &ce) || ce.Kind != tt.kind {
				t.Errorf("error = %v, want *Error of kind %d", err, tt.kind)
			}
		})
	}
}

func TestLookupCleans(t *testing.T) {
	const hostile = "bad\x1b[31m\u202e text"
	long := strings.Repeat("x", 300)
	c := clientWith(t, rtFunc(func(req *http.Request) (*http.Response, error) {
		body := lookupBody(t, map[string]any{
			"r0": lookupPRNode("title "+hostile, "acme/ap\x1b[31mi\u202e", "PR_H", 1, "SUCCESS"),
			"r1": nil,
			"r2": nil,
		}, []map[string]any{
			{"type": "WEIRD\x1b[31m_TYPE\u202e", "path": []any{"r1"}, "message": hostile},
			{"path": []any{"r2"}, "message": long},
		})
		return jsonResp(req, 200, body), nil
	}))
	refs := []domain.Ref{{Repo: "acme/api", Number: 1}, {Repo: "acme/b", Number: 2}, {Repo: "acme/c", Number: 3}}
	var snap domain.Snapshot
	prs, err := c.lookupRefs(context.Background(), refs, &snap)
	if err != nil {
		t.Fatal(err)
	}
	for name, s := range map[string]string{
		"title": prs[0].Title, "ref": prs[0].Ref.Repo,
		"reason 1": prs[1].Unavail.Reason, "help 1": prs[1].Unavail.Help,
		"reason 2": prs[2].Unavail.Reason, "help 2": prs[2].Unavail.Help,
	} {
		if why := noUnsafe(s); why != "" {
			t.Errorf("%s %q holds a %s", name, s, why)
		}
	}
	if !strings.HasPrefix(prs[1].Unavail.Reason, "could not load (weird") {
		t.Errorf("reason 1 = %q, want it to name the error type", prs[1].Unavail.Reason)
	}
	if want := "could not load (" + strings.Repeat("x", 40) + ")"; prs[2].Unavail.Reason != want {
		t.Errorf("reason 2 = %q, want the message cut to 40 runes", prs[2].Unavail.Reason)
	}
	if n := utf8.RuneCountInString(prs[2].Unavail.Help); n != 200 {
		t.Errorf("help 2 has %d runes, want 200", n)
	}
}

const singleFailingBody = `{"data":{"rateLimit":{"limit":5000,"cost":1,"remaining":4979,"resetAt":"2026-10-11T13:00:00Z"},
"nodes":[{"__typename":"PullRequest","id":"PR_F","statusCheckRollup":{"state":"FAILURE","contexts":{
"totalCount":1,"pageInfo":{"hasNextPage":false},"nodes":[{"__typename":"CheckRun","name":"lint",
"status":"COMPLETED","conclusion":"FAILURE","databaseId":22,"checkSuite":{"createdAt":"2026-10-11T09:00:00Z",
"app":{"name":"GitHub Actions"},"workflowRun":{"workflow":{"name":"ci"}}}}]}}}]}}`

func TestLookupSingle(t *testing.T) {
	t.Run("failing PR names its failing check", func(t *testing.T) {
		var failing int
		c := clientWith(t, rtFunc(func(req *http.Request) (*http.Response, error) {
			raw, _ := io.ReadAll(req.Body)
			switch opName(raw) {
			case "Lookup":
				return jsonResp(req, 200, lookupBody(t, map[string]any{
					"r0": lookupPRNode("Broken", "acme/api", "PR_F", 9, "FAILURE")}, nil)), nil
			case "FailingChecks":
				failing++
				return jsonResp(req, 200, []byte(singleFailingBody)), nil
			}
			t.Errorf("unexpected operation %q", opName(raw))
			return nil, errors.New("unexpected")
		}))
		pr, err := c.Lookup(context.Background(), domain.Ref{Repo: "acme/api", Number: 9})
		if err != nil {
			t.Fatal(err)
		}
		if failing != 1 {
			t.Errorf("FailingChecks requests = %d, want 1", failing)
		}
		if !pr.Failing.Known || pr.Failing.Name != "lint" {
			t.Errorf("Failing = %+v, want known and named lint", pr.Failing)
		}
	})
	t.Run("not found PR sends no FailingChecks", func(t *testing.T) {
		var ops []string
		c := clientWith(t, rtFunc(func(req *http.Request) (*http.Response, error) {
			raw, _ := io.ReadAll(req.Body)
			ops = append(ops, opName(raw))
			return jsonResp(req, 200, lookupBody(t, map[string]any{"r0": nil}, []map[string]any{
				{"type": "NOT_FOUND", "path": []any{"r0"}, "message": "Could not resolve to a Repository."}})), nil
		}))
		ref := domain.Ref{Repo: "ghost/nope", Number: 2}
		pr, err := c.Lookup(context.Background(), ref)
		if err != nil {
			t.Fatal(err)
		}
		if pr.Unavail != domain.NotFound() || pr.Ref != ref {
			t.Errorf("Lookup = %+v, want the NotFound PR for %v", pr, ref)
		}
		if !slices.Equal(ops, []string{"Lookup"}) {
			t.Errorf("requests = %v, want only Lookup", ops)
		}
	})
}

func TestFetchWatchedRequestFailure(t *testing.T) {
	c := watchClient(t, rtFunc(func(req *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(req.Body)
		switch opName(raw) {
		case "Watch":
			return jsonResp(req, 200, fixtureBytes(t, "search_page2.json")), nil
		case "FailingChecks":
			return jsonResp(req, 200, []byte(failingNoneBody)), nil
		case "Lookup":
			return jsonResp(req, 502, []byte(`{"message":"bad gateway"}`)), nil
		}
		t.Errorf("unexpected operation %q", opName(raw))
		return nil, errors.New("unexpected")
	}), []domain.Ref{{Repo: "acme/api", Number: 1}})
	snap, err := c.Fetch(context.Background())
	var ce *Error
	if !errors.As(err, &ce) || ce.Kind != KindServer {
		t.Fatalf("Fetch error = %v, want *Error KindServer", err)
	}
	if len(snap.PRs) != 0 || len(snap.Warnings) != 0 {
		t.Errorf("snapshot = %+v, want it empty so the model keeps its last rows", snap)
	}
}

func TestFetchUnavailableRows(t *testing.T) {
	tr := &fixtureTransport{t: t, files: []string{"search_page2.json"}, lookup: []string{"lookup_mixed.json"}}
	c := watchClient(t, tr, mixedRefs())
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var loaded, unavailable, auto int
	ids := map[string]bool{}
	for _, pr := range snap.PRs {
		switch {
		case pr.Unavail.Kind != domain.UnavailNone:
			unavailable++
		case pr.Manual:
			loaded++
			ids[pr.ID] = true
		case pr.Auto:
			auto++
		}
	}
	if auto != 4 || loaded != 2 || unavailable != 7 || !ids["PR_M1"] || !ids["PR_M7"] {
		t.Errorf("rows: %d discovered, %d loaded saved (%v), %d unavailable; want 4, 2 (PR_M1, PR_M7), 7", auto, loaded, ids, unavailable)
	}
	if want := "Some data could not be loaded: Resource not accessible by integration"; !slices.Contains(snap.Warnings, want) {
		t.Errorf("warnings = %q, want %q", snap.Warnings, want)
	}
	if len(tr.lookupBodies) != 1 {
		t.Errorf("Lookup requests = %d, want 1", len(tr.lookupBodies))
	}
}

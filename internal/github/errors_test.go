package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const secretToken = "test-token-NEVER-SHOW-7f3a"

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResp(req *http.Request, status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
		Request:    req,
	}
}

func fixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// withErrors returns a fixture body with an errors[] member added.
func withErrors(t *testing.T, name, errorsJSON string) []byte {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(fixtureBytes(t, name), &m); err != nil {
		t.Fatal(err)
	}
	m["errors"] = json.RawMessage(errorsJSON)
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// answerFailingNone answers a FailingChecks request with failingNoneBody
// without calling rt, so a test about discovery paging need not know about the
// second step.
func answerFailingNone(rt http.RoundTripper) http.RoundTripper {
	return rtFunc(func(req *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(raw))
		if opName(raw) == "FailingChecks" {
			return jsonResp(req, 200, []byte(failingNoneBody)), nil
		}
		return rt.RoundTrip(req)
	})
}

func clientWith(t *testing.T, rt http.RoundTripper) *Client {
	t.Helper()
	c, err := NewClient(Options{
		Token:      secretToken,
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rt,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func statusTransport(status int, message string) http.RoundTripper {
	return rtFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := json.Marshal(map[string]string{"message": message})
		return jsonResp(req, status, body), nil
	})
}

func hintOf(err error) string {
	var h interface{ Hint() string }
	if errors.As(err, &h) {
		return h.Hint()
	}
	return ""
}

func TestClassify(t *testing.T) {
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	dns := rtFunc(func(*http.Request) (*http.Response, error) {
		return nil, &net.DNSError{Err: "no such host", Name: "api.github.com", IsNotFound: true}
	})
	tests := []struct {
		name   string
		rt     http.RoundTripper
		ctx    context.Context
		kind   ErrorKind
		status int
		hints  []string
	}{
		{"401", statusTransport(401, "Bad credentials"), context.Background(), KindAuth, 401, []string{"gh auth login", "GH_TOKEN", "GITHUB_TOKEN"}},
		{"403", statusTransport(403, "Forbidden"), context.Background(), KindForbidden, 403, []string{"rate limit", "SSO"}},
		{"429", statusTransport(429, "Too many requests"), context.Background(), KindRateLimited, 429, []string{"rate limit", "resumes on its own"}},
		{"500", statusTransport(500, "Server Error"), context.Background(), KindServer, 500, []string{"GitHub is having problems"}},
		{"502", statusTransport(502, "Bad Gateway"), context.Background(), KindServer, 502, []string{"GitHub is having problems"}},
		{"404", statusTransport(404, "Not Found"), context.Background(), KindUnknown, 404, []string{"gh auth status"}},
		{"dns failure", dns, context.Background(), KindNetwork, 0, []string{"Check your connection"}},
		{"expired deadline", dns, expired, KindNetwork, 0, []string{"Check your connection"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := clientWith(t, tt.rt).Fetch(tt.ctx)
			var ge *Error
			if !errors.As(err, &ge) {
				t.Fatalf("err = %v (%T), want *Error", err, err)
			}
			if ge.Kind != tt.kind || ge.Status != tt.status {
				t.Errorf("kind/status = %v/%d, want %v/%d", ge.Kind, ge.Status, tt.kind, tt.status)
			}
			for _, h := range tt.hints {
				if !strings.Contains(ge.Hint(), h) {
					t.Errorf("Hint() = %q, want it to contain %q", ge.Hint(), h)
				}
			}
			if !strings.HasPrefix(ge.Error(), "GitHub request failed: ") {
				t.Errorf("Error() = %q", ge.Error())
			}
		})
	}
	t.Run("401 message and no silent fallback", func(t *testing.T) {
		_, err := clientWith(t, statusTransport(401, "Bad credentials")).Fetch(context.Background())
		if got, want := err.Error(), "GitHub request failed: HTTP 401: Bad credentials"; got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
		want := "GitHub rejected the token. Run `gh auth login`. If GH_TOKEN or GITHUB_TOKEN is set, it overrides the gh login: fix or unset it."
		if got := hintOf(err); got != want {
			t.Errorf("Hint() = %q, want %q", got, want)
		}
	})
	t.Run("context canceled is passed through for quit", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		honour := rtFunc(func(req *http.Request) (*http.Response, error) { return nil, req.Context().Err() })
		_, err := clientWith(t, honour).Fetch(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v, want it to wrap context.Canceled", err)
		}
		var ge *Error
		if errors.As(err, &ge) {
			t.Errorf("a canceled fetch must not be classified as %v", ge.Kind)
		}
	})
	t.Run("hostile API message is cleaned", func(t *testing.T) {
		_, err := clientWith(t, statusTransport(401, "Bad\x1b[31m credentials‮")).Fetch(context.Background())
		if why := noUnsafe(err.Error()); why != "" {
			t.Errorf("Error() %q has a %s", err.Error(), why)
		}
	})
}

func TestPartialData(t *testing.T) {
	body := withErrors(t, "search_page2.json", `[{"message":"Resource protected by organization SAML enforcement","type":"FORBIDDEN","path":["search","nodes",3]}]`)
	c := clientWith(t, rtFunc(func(req *http.Request) (*http.Response, error) { return jsonResp(req, 200, body), nil }))
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("err = %v, want nil: partial data is shown", err)
	}
	if len(snap.PRs) == 0 {
		t.Fatal("no PRs: the rows that loaded must be kept")
	}
	if len(snap.Warnings) != 1 || !strings.HasPrefix(snap.Warnings[0], "Some data could not be loaded") {
		t.Errorf("Warnings = %q, want one starting 'Some data could not be loaded'", snap.Warnings)
	}
}

func TestGraphQLErrorNoData(t *testing.T) {
	body := []byte(`{"data":null,"errors":[{"message":"Something went wrong"}]}`)
	c := clientWith(t, rtFunc(func(req *http.Request) (*http.Response, error) { return jsonResp(req, 200, body), nil }))
	snap, err := c.Fetch(context.Background())
	var ge *Error
	if !errors.As(err, &ge) || ge.Kind != KindGraphQL {
		t.Fatalf("err = %v, want *Error of KindGraphQL", err)
	}
	if len(snap.PRs) != 0 {
		t.Errorf("PRs = %d, want 0", len(snap.PRs))
	}
}

func TestLaterPageFailure(t *testing.T) {
	calls := 0
	page1 := fixtureBytes(t, "search_page1.json")
	c := clientWith(t, answerFailingNone(rtFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return jsonResp(req, 200, page1), nil
		}
		return jsonResp(req, 502, []byte(`{"message":"Bad Gateway"}`)), nil
	})))
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("err = %v, want nil: earlier pages stay on screen", err)
	}
	if calls != 2 {
		t.Errorf("requests = %d, want 2 (paging stops at the failure)", calls)
	}
	if !snap.Truncated {
		t.Error("Truncated = false, want true")
	}
	if len(snap.Warnings) != 1 || snap.Warnings[0] != "Only part of the list could be loaded" {
		t.Errorf("Warnings = %q", snap.Warnings)
	}
	found := false
	for _, pr := range snap.PRs {
		if pr.Ref.String() == "acme/web#77" {
			found = true
		}
	}
	if !found {
		t.Errorf("page 1 PRs missing: %v", refs(snap))
	}
}

func TestLaterPageRateLimit(t *testing.T) {
	calls := 0
	page1 := fixtureBytes(t, "search_page1.json") // hasNextPage is true
	c := clientWith(t, rtFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return jsonResp(req, 200, page1), nil
		}
		resp := jsonResp(req, 200, []byte(rateLimitedBody))
		resp.Header.Set("X-RateLimit-Remaining", "0")
		return resp, nil
	}))
	snap, err := c.Fetch(context.Background())
	var ge *Error
	if !errors.As(err, &ge) || ge.Kind != KindRateLimited {
		t.Fatalf("err = %v, want KindRateLimited", err)
	}
	if len(snap.PRs) != 0 || snap.Truncated || len(snap.Warnings) != 0 {
		t.Errorf("snapshot = %+v, want empty: the previous data stays on screen", snap)
	}
	if calls != 2 {
		t.Errorf("requests = %d, want 2", calls)
	}
}

// TestTokenNeverAppears runs every failure path with a recognisable token and
// checks that no message, hint or warning contains it, even when the
// transport's own error text carries the Authorization header.
func TestTokenNeverAppears(t *testing.T) {
	leaky := rtFunc(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("round trip failed with header %q", req.Header.Get("Authorization"))
	})
	page1 := fixtureBytes(t, "search_page1.json")
	partial := withErrors(t, "search_page2.json", `[{"message":"partial failure"}]`)
	second := 0
	scenarios := map[string]http.RoundTripper{
		"401":              statusTransport(401, "Bad credentials"),
		"403":              statusTransport(403, "Forbidden"),
		"429":              statusTransport(429, "Too many requests"),
		"200 rate limited": headerTransport(200, rateLimitedBody, http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {"1791721800"}}),
		"403 retry-after":  headerTransport(403, `{"message":"secondary rate limit"}`, http.Header{"Retry-After": {"120"}}),
		"500":              statusTransport(500, "Server Error"),
		"404":              statusTransport(404, "Not Found"),
		"leaky":            leaky,
		"dns":              rtFunc(func(*http.Request) (*http.Response, error) { return nil, &net.DNSError{Err: "no such host"} }),
		"graphql none": rtFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResp(req, 200, []byte(`{"data":null,"errors":[{"message":"boom"}]}`)), nil
		}),
		"partial": rtFunc(func(req *http.Request) (*http.Response, error) { return jsonResp(req, 200, partial), nil }),
		"later page": rtFunc(func(req *http.Request) (*http.Response, error) {
			second++
			if second == 1 {
				return jsonResp(req, 200, page1), nil
			}
			return jsonResp(req, 500, []byte(`{"message":"Server Error"}`)), nil
		}),
	}
	for name, rt := range scenarios {
		t.Run(name, func(t *testing.T) {
			snap, err := clientWith(t, rt).Fetch(context.Background())
			var texts []string
			if err != nil {
				texts = append(texts, err.Error(), hintOf(err))
			}
			texts = append(texts, snap.Warnings...)
			for _, s := range texts {
				if strings.Contains(s, secretToken) || strings.Contains(s, "NEVER-SHOW") {
					t.Errorf("text %q contains the token", s)
				}
			}
		})
	}
}

func TestShortReason(t *testing.T) {
	tests := []struct {
		kind ErrorKind
		want string
	}{
		{KindAuth, "auth"},
		{KindForbidden, "forbidden"},
		{KindServer, "server"},
		{KindNetwork, "network"},
		{KindGraphQL, "GitHub error"},
		{KindUnknown, "error"},
		{KindRateLimited, "rate limit"},
	}
	for _, tt := range tests {
		e := &Error{Kind: tt.kind, Status: 401, Message: "HTTP 401: " + secretToken}
		got := e.Short()
		if got != tt.want {
			t.Errorf("Short() for kind %d = %q, want %q", tt.kind, got, tt.want)
		}
		if strings.Contains(got, secretToken) || strings.Contains(got, "HTTP") {
			t.Errorf("Short() for kind %d leaks the message: %q", tt.kind, got)
		}
	}
}

const rateLimitedBody = `{"data":null,"errors":[{"type":"RATE_LIMITED","message":"API rate limit exceeded"}]}`

// headerTransport answers every request with the given status, body and headers.
func headerTransport(status int, body string, h http.Header) http.RoundTripper {
	return rtFunc(func(req *http.Request) (*http.Response, error) {
		resp := jsonResp(req, status, []byte(body))
		for k, v := range h {
			resp.Header[k] = v
		}
		return resp, nil
	})
}

func TestRateLimitDetection(t *testing.T) {
	reset := time.Date(2026, 10, 11, 12, 30, 0, 0, time.UTC)
	epoch := fmt.Sprint(reset.Unix())
	noType := `{"data":null,"errors":[{"message":"API rate limit exceeded"}]}`
	plainErr := `{"data":null,"errors":[{"message":"Something went wrong"}]}`
	tests := []struct {
		name       string
		status     int
		body       string
		hdr        http.Header
		kind       ErrorKind
		retryAfter time.Duration
		resetAt    time.Time
		backoff    bool
	}{
		{"200 typed with reset", 200, rateLimitedBody, http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {epoch}}, KindRateLimited, 0, reset, true},
		{"200 header only", 200, noType, http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {epoch}}, KindRateLimited, 0, reset, true},
		{"200 type only", 200, rateLimitedBody, nil, KindRateLimited, 0, time.Time{}, true},
		{"200 neither signal", 200, plainErr, nil, KindGraphQL, 0, time.Time{}, false},
		{"200 retry-after", 200, plainErr, http.Header{"Retry-After": {"30"}}, KindRateLimited, 30 * time.Second, time.Time{}, true},
		{"403 retry-after", 403, `{"message":"secondary"}`, http.Header{"Retry-After": {"120"}}, KindRateLimited, 120 * time.Second, time.Time{}, true},
		{"403 remaining 0", 403, `{"message":"limit"}`, http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {epoch}}, KindRateLimited, 0, reset, true},
		{"429 bare", 429, `{"message":"slow down"}`, nil, KindRateLimited, 0, time.Time{}, true},
		{"403 SSO stays forbidden", 403, `{"message":"SAML"}`, nil, KindForbidden, 0, time.Time{}, false},
		{"403 remaining 4999 stays forbidden", 403, `{"message":"SAML"}`, http.Header{"X-Ratelimit-Remaining": {"4999"}}, KindForbidden, 0, time.Time{}, false},
		{"retry-after as date", 429, `{"message":"x"}`, http.Header{"Retry-After": {"Wed, 21 Oct 2015 07:28:00 GMT"}}, KindRateLimited, 0, time.Time{}, true},
		{"retry-after negative", 429, `{"message":"x"}`, http.Header{"Retry-After": {"-5"}}, KindRateLimited, 0, time.Time{}, true},
		{"reset not a number", 429, `{"message":"x"}`, http.Header{"X-Ratelimit-Reset": {"abc"}}, KindRateLimited, 0, time.Time{}, true},
		{"reset zero", 429, `{"message":"x"}`, http.Header{"X-Ratelimit-Reset": {"0"}}, KindRateLimited, 0, time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := clientWith(t, headerTransport(tt.status, tt.body, tt.hdr)).Fetch(context.Background())
			var ge *Error
			if !errors.As(err, &ge) {
				t.Fatalf("err = %v (%T), want *Error", err, err)
			}
			if ge.Kind != tt.kind {
				t.Fatalf("Kind = %v, want %v", ge.Kind, tt.kind)
			}
			if ge.RetryAfter != tt.retryAfter || !ge.ResetAt.Equal(tt.resetAt) {
				t.Errorf("RetryAfter/ResetAt = %v/%v, want %v/%v", ge.RetryAfter, ge.ResetAt, tt.retryAfter, tt.resetAt)
			}
			ra, at, ok := ge.Backoff()
			if ok != tt.backoff || ra != tt.retryAfter || !at.Equal(tt.resetAt) {
				t.Errorf("Backoff() = %v, %v, %v", ra, at, ok)
			}
			if tt.kind == KindRateLimited {
				if ge.Short() != "rate limit" {
					t.Errorf("Short() = %q", ge.Short())
				}
				if tt.status == 200 && ge.Message != "GitHub API rate limit exceeded" {
					t.Errorf("Message = %q", ge.Message)
				}
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := map[string]time.Duration{
		"30": 30 * time.Second, " 120 ": 120 * time.Second, "0": 0, "": 0, "-5": 0, "1.5": 0,
		"Wed, 21 Oct 2015 07:28:00 GMT": 0, "abc": 0,
	}
	for in, want := range tests {
		if got := parseRetryAfter(in); got != want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseReset(t *testing.T) {
	want := time.Date(2026, 10, 11, 12, 30, 0, 0, time.UTC)
	if got := parseReset(fmt.Sprint(want.Unix())); !got.Equal(want) {
		t.Errorf("parseReset(valid) = %v, want %v", got, want)
	}
	for _, in := range []string{"", "abc", "0", "-1", "1.5"} {
		if got := parseReset(in); !got.IsZero() {
			t.Errorf("parseReset(%q) = %v, want zero", in, got)
		}
	}
}

func TestCaptureHeadersOnly(t *testing.T) {
	if n := reflect.TypeOf(rateHeaders{}).NumField(); n != 3 {
		t.Fatalf("rateHeaders has %d fields, want exactly 3", n)
	}
	ct := &captureTransport{base: rtFunc(func(req *http.Request) (*http.Response, error) {
		resp := jsonResp(req, 200, []byte("{}"))
		resp.Header.Set("Retry-After", "7")
		resp.Header.Set("X-RateLimit-Remaining", "0")
		resp.Header.Set("X-RateLimit-Reset", "1791721800")
		resp.Header.Set("Authorization", "token "+secretToken)
		resp.Header.Set("X-Other", secretToken)
		return resp, nil
	})}
	req, _ := http.NewRequest("POST", "https://api.github.com/graphql", nil)
	req.Header.Set("Authorization", "token "+secretToken)
	if _, err := ct.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	got := ct.latest()
	if got != (rateHeaders{retryAfter: "7", remaining: "0", reset: "1791721800"}) {
		t.Errorf("latest() = %+v", got)
	}
	if s := fmt.Sprintf("%+v", got); strings.Contains(s, secretToken) || strings.Contains(s, "NEVER-SHOW") {
		t.Errorf("captured record contains the token: %s", s)
	}
	// A transport error clears the record.
	ct.base = rtFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("boom") })
	if _, err := ct.RoundTrip(req); err == nil {
		t.Fatal("want error")
	}
	if ct.latest() != (rateHeaders{}) {
		t.Errorf("latest() after a failed request = %+v, want zero", ct.latest())
	}
}

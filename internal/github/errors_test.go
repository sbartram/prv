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
		{"429", statusTransport(429, "Too many requests"), context.Background(), KindForbidden, 429, []string{"rate limit", "SSO"}},
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
	c := clientWith(t, rtFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return jsonResp(req, 200, page1), nil
		}
		return jsonResp(req, 502, []byte(`{"message":"Bad Gateway"}`)), nil
	}))
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
		"401":   statusTransport(401, "Bad credentials"),
		"403":   statusTransport(403, "Forbidden"),
		"429":   statusTransport(429, "Too many requests"),
		"500":   statusTransport(500, "Server Error"),
		"404":   statusTransport(404, "Not Found"),
		"leaky": leaky,
		"dns":   rtFunc(func(*http.Request) (*http.Response, error) { return nil, &net.DNSError{Err: "no such host"} }),
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

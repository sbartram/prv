package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/cli/go-gh/v2/pkg/auth"

	"github.com/sbartram/prv/internal/github"
	"github.com/sbartram/prv/internal/ui"
)

// useConfig points prv at a config file in a temp dir so no test reads the
// developer's real config. An empty body leaves the file absent.
func useConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "prv", "config.toml")
	if body != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	orig := configPath
	configPath = func() (string, error) { return path, nil }
	t.Cleanup(func() { configPath = orig })
	return path
}

func TestVersionFlag(t *testing.T) {
	useConfig(t, "")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !regexp.MustCompile(`^prv \S+\n$`).MatchString(stdout.String()) {
		t.Errorf("stdout = %q, want one line 'prv <version>'", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestNoToken(t *testing.T) {
	useConfig(t, "")
	orig := tokenForHost
	tokenForHost = func(string) (string, string) { return "", "default" }
	t.Cleanup(func() { tokenForHost = orig })

	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	want := "prv needs a GitHub token. Run `gh auth login`, or set GH_TOKEN / GITHUB_TOKEN.\n"
	if stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestHiddenClosedDaysFlag(t *testing.T) {
	useConfig(t, "")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-h"}, &stdout, &stderr); code != 0 {
		t.Errorf("-h exit code = %d, want 0", code)
	}
	if strings.Contains(stderr.String(), "closed-days") {
		t.Errorf("usage mentions the hidden flag: %q", stderr.String())
	}
	stderr.Reset()
	if code := run([]string{"--closed-days", "0"}, &stdout, &stderr); code != 2 {
		t.Errorf("--closed-days 0 exit code = %d, want 2", code)
	}
}

func TestBrokenConfigExitsBeforeToken(t *testing.T) {
	path := useConfig(t, `refresh_interval = "soon"`)
	orig := tokenForHost
	tokenForHost = func(string) (string, string) {
		t.Error("tokenForHost was called with a broken config")
		return "", ""
	}
	t.Cleanup(func() { tokenForHost = orig })

	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !strings.HasPrefix(stderr.String(), "prv: ") || !strings.Contains(stderr.String(), path) {
		t.Errorf("stderr = %q, want a line starting 'prv: ' that contains %q", stderr.String(), path)
	}
}

func TestVersionIgnoresBrokenConfig(t *testing.T) {
	useConfig(t, `refresh_interval = "soon"`)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
}

// countingTransport serves the fixture body and counts requests. It is safe
// for concurrent use.
type countingTransport struct{ n atomic.Int64 }

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.n.Add(1)
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(fixtureHead + fixtureNodes + fixtureTail)),
		Request:    req,
	}, nil
}

// TestRefreshEndToEnd runs a real tea.Program over the real github client and
// checks that prv keeps re-fetching and that quitting returns cleanly.
func TestRefreshEndToEnd(t *testing.T) {
	ct := &countingTransport{}
	client, err := github.NewClient(github.Options{Token: "test-token-not-real", ClosedDays: 14, Transport: ct})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	p := tea.NewProgram(
		ui.New(ctx, client, ui.Options{ClosedDays: 14, Interval: 20 * time.Millisecond}),
		tea.WithContext(ctx), tea.WithInput(nil), tea.WithOutput(io.Discard),
	)
	done := make(chan error, 1)
	go func() {
		_, err := p.Run()
		done <- err
	}()

	deadline := time.After(5 * time.Second)
	for ct.n.Load() < 3 {
		select {
		case err := <-done:
			t.Fatalf("program ended early with %v after %d requests", err, ct.n.Load())
		case <-deadline:
			t.Fatalf("only %d requests in 5 s, want at least 3", ct.n.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	p.Quit()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("program did not return within 5 s of Quit")
	}
}

// runCmd runs cmd, recurses into tea.BatchMsg, and passes every non-nil
// message to Update once. It never runs commands that Update returns.
func runCmd(m tea.Model, cmd tea.Cmd) tea.Model {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			m = runCmd(m, c)
		}
		return m
	}
	if msg == nil {
		return m
	}
	m, _ = m.Update(msg)
	return m
}

func send(m tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.Update(msg)
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
	r := []rune(s)[0]
	return tea.KeyPressMsg{Code: r, Text: s}
}

// recorder is a fake http.RoundTripper serving one canned GraphQL body.
type recorder struct {
	body  string
	url   string
	auth  string
	query string
	vars  map[string]any
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.url = req.URL.String()
	r.auth = req.Header.Get("Authorization")
	raw, _ := io.ReadAll(req.Body)
	var payload struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	_ = json.Unmarshal(raw, &payload)
	r.query, r.vars = payload.Query, payload.Variables
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(r.body)),
		Request:    req,
	}, nil
}

const fixtureHead = `{"data":{"viewer":{"login":"octo"},"rateLimit":{"cost":1,"remaining":4999,"resetAt":"2026-10-11T13:00:00Z"},` +
	`"search":{"issueCount":2,"pageInfo":{"hasNextPage":false,"endCursor":"c"},"nodes":[`

const fixtureTail = `]}}}`

const fixtureNodes = `
{"__typename":"PullRequest","id":"PR_1","number":481,"title":"Fix token refresh","url":"https://github.com/acme/api/pull/481",
 "isDraft":true,"state":"OPEN","closedAt":null,"mergedAt":null,"updatedAt":"2026-10-11T08:00:00Z",
 "repository":{"nameWithOwner":"acme/api"},"author":{"login":"octo"},
 "reviewDecision":null,"latestOpinionatedReviews":{"nodes":[]},"statusCheckRollup":null},
{"__typename":"PullRequest","id":"PR_2","number":12,"title":"Bump deps","url":"https://github.com/acme/web/pull/12",
 "isDraft":false,"state":"MERGED","closedAt":"2026-10-10T09:00:00Z","mergedAt":"2026-10-10T09:00:00Z","updatedAt":"2026-10-10T09:00:00Z",
 "repository":{"nameWithOwner":"acme/web"},"author":{"login":"hubot"},
 "reviewDecision":"APPROVED","latestOpinionatedReviews":{"nodes":[]},"statusCheckRollup":{"state":"SUCCESS"}}`

func fixtureModel(t *testing.T, body string) (tea.Model, *recorder) {
	t.Helper()
	rec := &recorder{body: body}
	client, err := github.NewClient(github.Options{
		Token:      "test-token-not-real",
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rec,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = ui.New(context.Background(), client, ui.Options{ClosedDays: 14})
	m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = runCmd(m, m.Init())
	return m, rec
}

func plain(m tea.Model) string { return ansi.Strip(m.View().Content) }

func lineWith(view, needle string) string {
	for _, l := range strings.Split(view, "\n") {
		if strings.Contains(l, needle) {
			return l
		}
	}
	return ""
}

func TestEndToEndFixture(t *testing.T) {
	m, rec := fixtureModel(t, fixtureHead+fixtureNodes+fixtureTail)

	view := plain(m)
	for _, want := range []string{"@octo", "acme/api#481", "[draft] Fix token refresh", "me", "hubot", "merged", "– none", "– no checks", "✓ approved", "✓ passing"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
	row1 := func() string { return lineWith(plain(m), "acme/api#481") }
	row2 := func() string { return lineWith(plain(m), "acme/web#12") }
	if !strings.HasPrefix(row1(), ">") || strings.HasPrefix(row2(), ">") {
		t.Fatalf("marker should start on row 1:\n%s", plain(m))
	}

	m, _ = send(m, key("k"))
	if !strings.HasPrefix(row1(), ">") {
		t.Error("k at the first row must keep the marker on row 1")
	}
	m, _ = send(m, key("j"))
	if !strings.HasPrefix(row2(), ">") || strings.HasPrefix(row1(), ">") {
		t.Error("j must move the marker to row 2")
	}
	m, _ = send(m, key("j"))
	if !strings.HasPrefix(row2(), ">") {
		t.Error("j at the last row must keep the marker on row 2")
	}

	_, cmd := send(m, key("q"))
	if cmd == nil {
		t.Fatal("q returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("q command message is not tea.QuitMsg")
	}

	if rec.url != "https://api.github.com/graphql" {
		t.Errorf("request URL = %q", rec.url)
	}
	if rec.auth != "token test-token-not-real" {
		t.Errorf("Authorization header = %q", rec.auth)
	}
	if !strings.Contains(rec.query, "ISSUE_ADVANCED") {
		t.Error("query does not contain ISSUE_ADVANCED")
	}
	const wantQ = "is:pr (author:@me OR assignee:@me) (is:open OR closed:>=2026-09-26) sort:updated-desc"
	if got := rec.vars["q"]; got != wantQ {
		t.Errorf("variables.q = %q, want %q", got, wantQ)
	}

	t.Run("zero rows", func(t *testing.T) {
		m, _ := fixtureModel(t, fixtureHead+fixtureTail)
		for _, k := range []string{"j", "k", "down", "up"} {
			m, _ = send(m, key(k))
		}
		if v := plain(m); !strings.Contains(v, "@octo") {
			t.Errorf("zero-row view lost the header:\n%s", v)
		}
	})
}

func TestLiveEndToEnd(t *testing.T) {
	if os.Getenv("PRV_LIVE") != "1" {
		t.Skip("set PRV_LIVE=1 to run against api.github.com")
	}
	token, _ := auth.TokenForHost("github.com")
	if token == "" {
		t.Fatal("no GitHub token found: run `gh auth login` or set GH_TOKEN")
	}
	// 300 days: the account has no PRs in the default 14-day window.
	client, err := github.NewClient(github.Options{Token: token, ClosedDays: 300})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = ui.New(context.Background(), client, ui.Options{ClosedDays: 300})
	m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = runCmd(m, m.Init())

	view := plain(m)
	if strings.Contains(view, "Could not load your PRs") {
		t.Fatalf("live fetch failed:\n%s", view)
	}
	if !regexp.MustCompile(`prv · @\S+ · [0-9]+ PRs`).MatchString(view) {
		t.Errorf("view has no list header:\n%s", view)
	}
}

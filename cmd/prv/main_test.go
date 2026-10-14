package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/cli/go-gh/v2/pkg/auth"

	"github.com/sbartram/prv/internal/config"
	"github.com/sbartram/prv/internal/domain"
	"github.com/sbartram/prv/internal/github"
	"github.com/sbartram/prv/internal/ui"
	"github.com/sbartram/prv/internal/watchlist"
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

	// WR-01: the upper bound is 365, and the message names both bounds.
	stderr.Reset()
	if code := run([]string{"--closed-days", "366"}, &stdout, &stderr); code != 2 {
		t.Errorf("--closed-days 366 exit code = %d, want 2", code)
	}
	if want := "prv: --closed-days must be between 1 and 365\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}

	// 365 passes validation: with no token the run stops at the token check.
	orig := tokenForHost
	tokenForHost = func(string) (string, string) { return "", "default" }
	t.Cleanup(func() { tokenForHost = orig })
	stderr.Reset()
	if code := run([]string{"--closed-days", "365"}, &stdout, &stderr); code != 1 {
		t.Errorf("--closed-days 365 exit code = %d, want 1 (validation passed, no token)", code)
	}
}

func TestEffectiveClosedDays(t *testing.T) {
	parse := func(args ...string) (*flag.FlagSet, int) {
		fs := flag.NewFlagSet("prv", flag.ContinueOnError)
		days := fs.Int("closed-days", domain.DefaultClosedDays, "")
		if err := fs.Parse(args); err != nil {
			t.Fatal(err)
		}
		return fs, *days
	}
	cases := []struct {
		name string
		args []string
		cfg  int
		want int
	}{
		{"flag beats config", []string{"--closed-days", "30"}, 20, 30},
		{"config beats default", nil, 20, 20},
		{"default config", nil, domain.DefaultClosedDays, 14},
		{"flag equal to the default still beats config", []string{"--closed-days", "14"}, 20, 14},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs, days := parse(c.args...)
			cfg := config.Default()
			cfg.ClosedDays = c.cfg
			if got := effectiveClosedDays(fs, days, cfg); got != c.want {
				t.Errorf("effectiveClosedDays = %d, want %d", got, c.want)
			}
		})
	}
}

func TestConfigErrorFormat(t *testing.T) {
	cases := []struct {
		name, body, linePrefix string
	}{
		{"wrong type", `closed_days = "fourteen"`, " line 1: "},
		{"unknown show value", "show = [\"open\", \"opne\"]", " line 1: "},
		{"integer interval", `refresh_interval = 60`, " line 1: "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := useConfig(t, c.body)
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
			if want := "prv: " + path + c.linePrefix; !strings.HasPrefix(stderr.String(), want) {
				t.Errorf("stderr = %q, want a prefix %q", stderr.String(), want)
			}
		})
	}
}

func TestDisplayPath(t *testing.T) {
	cases := []struct{ path, home, want string }{
		{"/home/u/.config/prv/config.toml", "/home/u", "~/.config/prv/config.toml"},
		{"/etc/prv/config.toml", "/home/u", "/etc/prv/config.toml"},
		{"/home/user2/x.toml", "/home/u", "/home/user2/x.toml"},
		{"/home/u/x.toml", "", "/home/u/x.toml"},
		{"", "/home/u", "none"},
	}
	for _, c := range cases {
		if got := displayPath(c.path, c.home); got != c.want {
			t.Errorf("displayPath(%q, %q) = %q, want %q", c.path, c.home, got, c.want)
		}
	}
}

// TestClosedWindowFromConfig follows closed_days = 3 from the file to the
// search date and the exact window filter (FILT-02).
func TestClosedWindowFromConfig(t *testing.T) {
	path := useConfig(t, "closed_days = 3\n")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	fs := flag.NewFlagSet("prv", flag.ContinueOnError)
	days := fs.Int("closed-days", domain.DefaultClosedDays, "")
	if err := fs.Parse(nil); err != nil {
		t.Fatal(err)
	}
	eff := effectiveClosedDays(fs, *days, cfg)
	if eff != 3 {
		t.Fatalf("effectiveClosedDays = %d, want 3", eff)
	}

	old := `,
{"__typename":"PullRequest","id":"PR_3","number":7,"title":"Old news","url":"https://github.com/acme/web/pull/7",
 "isDraft":false,"state":"CLOSED","closedAt":"2026-10-06T09:00:00Z","mergedAt":null,"updatedAt":"2026-10-06T09:00:00Z",
 "repository":{"nameWithOwner":"acme/web"},"author":{"login":"hubot"},
 "reviewDecision":null,"latestOpinionatedReviews":{"nodes":[]},"statusCheckRollup":null}`
	rec := &recorder{body: fixtureHead + fixtureNodes + old + fixtureTail}
	client, err := github.NewClient(github.Options{
		Token:      "test-token-not-real",
		ClosedDays: eff,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rec,
	})
	if err != nil {
		t.Fatal(err)
	}
	snap, err := client.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if q, _ := rec.vars["q"].(string); !strings.Contains(q, "closed:>=2026-10-07") {
		t.Errorf("search q = %q, want it to contain closed:>=2026-10-07", q)
	}
	for _, pr := range snap.PRs {
		if pr.Ref.Number == 7 {
			t.Errorf("a PR closed 5 days ago is listed with closed_days = 3")
		}
	}
	if len(snap.PRs) != 2 {
		t.Errorf("got %d PRs, want the 2 inside the window", len(snap.PRs))
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
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
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

const fixtureHead = `{"data":{"viewer":{"login":"octo"},"rateLimit":{"limit":5000,"cost":1,"remaining":4999,"resetAt":"2026-10-11T13:00:00Z"},` +
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
	if !strings.Contains(view, "api 4999") {
		t.Errorf("view missing the budget 'api 4999':\n%s", view)
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

// rateLimitedTransport serves the fixture on request 1 and an exhausted
// primary-limit response (HTTP 200 with errors) on every later request.
type rateLimitedTransport struct {
	n     atomic.Int64
	reset time.Time
}

func (r *rateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	h := http.Header{"Content-Type": []string{"application/json"}}
	body := fixtureHead + fixtureNodes + fixtureTail
	if r.n.Add(1) > 1 {
		body = `{"data":null,"errors":[{"type":"RATE_LIMITED","message":"API rate limit exceeded"}]}`
		h.Set("X-RateLimit-Remaining", "0")
		h.Set("X-RateLimit-Reset", strconv.FormatInt(r.reset.Unix(), 10))
	}
	return &http.Response{StatusCode: 200, Header: h, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
}

// TestRateLimitedEndToEnd drives the real client through a rate-limited
// refresh: the failure and the pause show in the header, the rows stay, and r
// makes no further request.
func TestRateLimitedEndToEnd(t *testing.T) {
	fixed := func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) }
	rt := &rateLimitedTransport{reset: time.Date(2026, 10, 11, 12, 30, 0, 0, time.UTC)}
	client, err := github.NewClient(github.Options{Token: "test-token-not-real", ClosedDays: 14, Now: fixed, Transport: rt})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = ui.New(context.Background(), client, ui.Options{ClosedDays: 14, Now: fixed})
	m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = runCmd(m, m.Init())
	if !strings.Contains(plain(m), "acme/api#481") {
		t.Fatalf("first fetch did not list the fixture:\n%s", plain(m))
	}

	m, cmd := send(m, key("r"))
	m = runCmd(m, cmd)
	view := plain(m)
	for _, want := range []string{"✗ refresh failed 12:00 (rate limit)", "paused until 12:30 (rate limit)", "acme/api#481", "acme/web#12"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
	if n := rt.n.Load(); n != 2 {
		t.Fatalf("requests = %d, want 2", n)
	}

	m, _ = send(m, key("r")) // refused: its command is not run
	if n := rt.n.Load(); n != 2 {
		t.Errorf("requests after the refused r = %d, want 2", n)
	}
	if v := plain(m); !strings.Contains(v, "rate limited — resumes 12:30") {
		t.Errorf("footer missing the refusal:\n%s", v)
	}
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

// routedTransport answers by GraphQL operation: "query Watch" gets the
// discovery body, "query FailingChecks" the step-2 body (D-01) and "query
// Detail" the detail body. It records every FailingChecks and Detail request,
// and the order of all operations.
type routedTransport struct {
	t            *testing.T
	watch        string
	failing      string
	detail       string
	lookup       string
	lookupStatus int // 0 means 200; any other value answers Lookup with that HTTP status
	failVars     []map[string]any
	failQuerys   []string
	detailVars   []map[string]any
	lookupVars   []map[string]any
	lookupQuerys []string
	ops          []string
}

// opIndexes returns the positions of op in the order of all requests.
func (r *routedTransport) opIndexes(op string) []int {
	var out []int
	for i, o := range r.ops {
		if o == op {
			out = append(out, i)
		}
	}
	return out
}

func (r *routedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, _ := io.ReadAll(req.Body)
	var payload struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	_ = json.Unmarshal(raw, &payload)
	body := ""
	switch {
	case strings.HasPrefix(payload.Query, "query Watch"):
		body = r.watch
		r.ops = append(r.ops, "Watch")
	case strings.HasPrefix(payload.Query, "query FailingChecks"):
		body = r.failing
		r.failVars = append(r.failVars, payload.Variables)
		r.failQuerys = append(r.failQuerys, payload.Query)
		r.ops = append(r.ops, "FailingChecks")
	case strings.HasPrefix(payload.Query, "query Detail"):
		body = r.detail
		r.detailVars = append(r.detailVars, payload.Variables)
		r.ops = append(r.ops, "Detail")
	case strings.HasPrefix(payload.Query, "query Lookup"):
		body = r.lookup
		r.lookupVars = append(r.lookupVars, payload.Variables)
		r.lookupQuerys = append(r.lookupQuerys, payload.Query)
		r.ops = append(r.ops, "Lookup")
	default:
		r.t.Errorf("unexpected operation: %.40q", payload.Query)
	}
	status := 200
	if r.lookupStatus != 0 && strings.HasPrefix(payload.Query, "query Lookup") {
		status, body = r.lookupStatus, `{"message":"Bad Gateway"}`
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

const failingWatchBody = fixtureHead + `
{"__typename":"PullRequest","id":"PR_7","number":7,"title":"Add retries","url":"https://github.com/acme/api/pull/7",
 "isDraft":false,"state":"OPEN","closedAt":null,"mergedAt":null,"updatedAt":"2026-10-11T08:00:00Z",
 "repository":{"nameWithOwner":"acme/api"},"author":{"login":"octo"},
 "reviewDecision":null,"latestOpinionatedReviews":{"nodes":[]},"statusCheckRollup":{"state":"FAILURE"}},
{"__typename":"PullRequest","id":"PR_8","number":8,"title":"Docs","url":"https://github.com/acme/web/pull/8",
 "isDraft":false,"state":"OPEN","closedAt":null,"mergedAt":null,"updatedAt":"2026-10-11T07:00:00Z",
 "repository":{"nameWithOwner":"acme/web"},"author":{"login":"octo"},
 "reviewDecision":null,"latestOpinionatedReviews":{"nodes":[]},"statusCheckRollup":{"state":"SUCCESS"}}` + fixtureTail

// ctxRun builds one CheckRun context in the GitHub Actions workflow "ci".
func ctxRun(id int, name, conclusion, suiteAt string) string {
	return `{"__typename":"CheckRun","name":"` + name + `","status":"COMPLETED","conclusion":"` + conclusion +
		`","databaseId":` + strconv.Itoa(id) + `,"checkSuite":{"createdAt":"` + suiteAt +
		`","app":{"name":"GitHub Actions"},"workflowRun":{"workflow":{"name":"ci"}}}}`
}

// failingStepBody is PR_7 with lint, test (ubuntu) and vet failing in the newer
// suite, and build failing in an older suite but passing in the newer one: the
// superseded build run must neither be named nor counted (D-03).
var failingStepBody = `{"data":{"rateLimit":{"limit":5000,"cost":1,"remaining":4987,"resetAt":"2026-10-11T13:00:00Z"},"nodes":[` +
	`{"__typename":"PullRequest","id":"PR_7","statusCheckRollup":{"state":"FAILURE","contexts":{"totalCount":5,"pageInfo":{"hasNextPage":false,"endCursor":"x"},"nodes":[` +
	strings.Join([]string{
		ctxRun(4, "build", "FAILURE", "2026-10-11T08:00:00Z"),
		ctxRun(11, "lint", "FAILURE", "2026-10-11T09:00:00Z"),
		ctxRun(12, "test (ubuntu)", "FAILURE", "2026-10-11T09:00:00Z"),
		ctxRun(13, "vet", "FAILURE", "2026-10-11T09:00:00Z"),
		ctxRun(14, "build", "SUCCESS", "2026-10-11T09:00:00Z"),
	}, ",") + `]}}}]}}`

// TestFailingNameEndToEnd is the phase tracer: a real github.Client over a
// routed transport and the real ui.Model name the failing check in the CI cell
// of the failing PR, with one FailingChecks request for the failing PRs only.
func TestFailingNameEndToEnd(t *testing.T) {
	rt := &routedTransport{t: t, watch: failingWatchBody, failing: failingStepBody}
	client, err := github.NewClient(github.Options{
		Token:      "test-token-not-real",
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rt,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = ui.New(context.Background(), client, ui.Options{ClosedDays: 14})
	m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = runCmd(m, m.Init())

	view := plain(m)
	if got := lineWith(view, "acme/api#7"); !strings.Contains(got, "✗ lint +2") {
		t.Errorf("acme/api#7 row = %q, want it to contain %q\n%s", got, "✗ lint +2", view)
	}
	if got := lineWith(view, "acme/web#8"); !strings.Contains(got, "✓ passing") {
		t.Errorf("acme/web#8 row = %q, want it to contain %q", got, "✓ passing")
	}
	if len(rt.failVars) != 1 {
		t.Fatalf("FailingChecks requests = %d, want exactly 1", len(rt.failVars))
	}
	ids, _ := rt.failVars[0]["ids"].([]any)
	if len(ids) != 1 || ids[0] != "PR_7" {
		t.Errorf("variables.ids = %v, want [PR_7]", rt.failVars[0]["ids"])
	}
	if strings.Contains(rt.failQuerys[0], "PR_7") {
		t.Error("the FailingChecks query text must not contain a PR id: ids travel only as a variable")
	}
}

// detailBody is the Detail answer for PR_7: one failing check and one reviewer.
var detailBody = `{"data":{"rateLimit":{"limit":5000,"cost":1,"remaining":4986,"resetAt":"2026-10-11T13:00:00Z"},"node":` +
	`{"__typename":"PullRequest","id":"PR_7",` +
	`"latestOpinionatedReviews":{"nodes":[{"state":"APPROVED","author":{"__typename":"User","login":"alice"}}]},` +
	`"latestReviews":{"nodes":[{"state":"APPROVED","author":{"__typename":"User","login":"alice"}}]},` +
	`"reviewRequests":{"nodes":[]},` +
	`"statusCheckRollup":{"state":"FAILURE","contexts":{"totalCount":1,"pageInfo":{"hasNextPage":false,"endCursor":"x"},"nodes":[` +
	ctxRun(11, "lint", "FAILURE", "2026-10-11T09:00:00Z") + `]}}}}}`

// partialDetailBody is the Detail answer for PR_7 when GitHub nulls one check
// node: totalCount 2, the lint failure and a null, with the error that explains
// the null next to the data (G-03-6-WR02).
var partialDetailBody = `{"data":{"rateLimit":{"limit":5000,"cost":1,"remaining":4986,"resetAt":"2026-10-11T13:00:00Z"},"node":` +
	`{"__typename":"PullRequest","id":"PR_7",` +
	`"latestOpinionatedReviews":{"nodes":[]},"latestReviews":{"nodes":[]},"reviewRequests":{"nodes":[]},` +
	`"statusCheckRollup":{"state":"FAILURE","contexts":{"totalCount":2,"pageInfo":{"hasNextPage":false,"endCursor":"x"},"nodes":[` +
	ctxRun(11, "lint", "FAILURE", "2026-10-11T09:00:00Z") + `,null]}}}},` +
	`"errors":[{"type":"FORBIDDEN","path":["node","statusCheckRollup","contexts","nodes",1],` +
	`"message":"Resource not accessible by integration"}]}`

// keptErrorDetailBody is the Detail answer for PR_7 when GitHub keeps an error
// about a review request next to complete check data: one run, totalCount 1.
var keptErrorDetailBody = `{"data":{"rateLimit":{"limit":5000,"cost":1,"remaining":4986,"resetAt":"2026-10-11T13:00:00Z"},"node":` +
	`{"__typename":"PullRequest","id":"PR_7",` +
	`"latestOpinionatedReviews":{"nodes":[]},"latestReviews":{"nodes":[]},"reviewRequests":{"nodes":[{"requestedReviewer":null}]},` +
	`"statusCheckRollup":{"state":"FAILURE","contexts":{"totalCount":1,"pageInfo":{"hasNextPage":false,"endCursor":"x"},"nodes":[` +
	ctxRun(11, "lint", "FAILURE", "2026-10-11T09:00:00Z") + `]}}}},` +
	`"errors":[{"type":"FORBIDDEN","path":["node","reviewRequests","nodes",0,"requestedReviewer"],` +
	`"message":"Resource not accessible by integration"}]}`

// keptErrorNoChecksDetailBody is the same answer for a PR with no
// statusCheckRollup (G-03-6, WR-05 case 2).
var keptErrorNoChecksDetailBody = `{"data":{"rateLimit":{"limit":5000,"cost":1,"remaining":4986,"resetAt":"2026-10-11T13:00:00Z"},"node":` +
	`{"__typename":"PullRequest","id":"PR_7",` +
	`"latestOpinionatedReviews":{"nodes":[]},"latestReviews":{"nodes":[]},"reviewRequests":{"nodes":[{"requestedReviewer":null}]},` +
	`"statusCheckRollup":null}},` +
	`"errors":[{"type":"FORBIDDEN","path":["node","reviewRequests","nodes",0,"requestedReviewer"],` +
	`"message":"Resource not accessible by integration"}]}`

// step runs cmd and passes every message it yields to Update once, like
// runCmd, and returns the commands Update returned. A command that takes longer
// than 250 ms is a timer (the refresh tick) and is dropped, so no test waits
// for a 60 s tick.
func step(m tea.Model, cmd tea.Cmd) (tea.Model, []tea.Cmd) {
	if cmd == nil {
		return m, nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-ch:
	case <-time.After(250 * time.Millisecond):
		return m, nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var next []tea.Cmd
		for _, c := range batch {
			var n []tea.Cmd
			m, n = step(m, c)
			next = append(next, n...)
		}
		return m, next
	}
	if msg == nil {
		return m, nil
	}
	m, n := m.Update(msg)
	return m, []tea.Cmd{n}
}

// settle runs a Cmd and then the Cmds its messages made, once: for a refresh
// that is the fetch and then the detail fetch that follows its result.
func settle(m tea.Model, cmd tea.Cmd) tea.Model {
	m, next := step(m, cmd)
	for _, c := range next {
		m, _ = step(m, c)
	}
	return m
}

// TestDetailEndToEnd runs the real github.Client under the real ui.Model: enter
// sends one Detail request for the selected PR, the next refresh sends the
// discovery query and then one more Detail request, and after esc no Detail
// request goes out.
func TestDetailEndToEnd(t *testing.T) {
	rt := &routedTransport{t: t, watch: failingWatchBody, failing: failingStepBody, detail: detailBody}
	client, err := github.NewClient(github.Options{
		Token:      "test-token-not-real",
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rt,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = ui.New(context.Background(), client, ui.Options{ClosedDays: 14})
	m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = runCmd(m, m.Init())

	// enter: one Detail request, for the id of the selected PR.
	m, cmd := send(m, key("enter"))
	m = settle(m, cmd)
	if len(rt.detailVars) != 1 || rt.detailVars[0]["id"] != "PR_7" {
		t.Fatalf("Detail requests = %v, want exactly one with id PR_7", rt.detailVars)
	}
	if v := plain(m); !strings.Contains(v, "CHECKS") || !strings.Contains(v, "✗ lint") || !strings.Contains(v, "alice") {
		t.Errorf("the detail view is not showing the answer:\n%s", v)
	}

	// r: the discovery query again, then one more Detail request.
	m, cmd = send(m, key("r"))
	m = settle(m, cmd)
	watches, details := rt.opIndexes("Watch"), rt.opIndexes("Detail")
	if len(watches) != 2 || len(details) != 2 {
		t.Fatalf("after a refresh: %d Watch and %d Detail requests (%v), want 2 and 2", len(watches), len(details), rt.ops)
	}
	if details[1] < watches[1] {
		t.Errorf("the second Detail request went out before the second Watch request: %v", rt.ops)
	}
	if rt.detailVars[1]["id"] != "PR_7" {
		t.Errorf("second Detail id = %v, want PR_7", rt.detailVars[1]["id"])
	}

	// esc: a refresh makes no Detail request.
	m = press(m, "esc")
	m, cmd = send(m, key("r"))
	_ = settle(m, cmd)
	if got := len(rt.opIndexes("Detail")); got != 2 {
		t.Errorf("Detail requests = %d after esc and a refresh, want still 2 (%v)", got, rt.ops)
	}
	if got := len(rt.opIndexes("Watch")); got != 3 {
		t.Errorf("Watch requests = %d, want 3 (%v)", got, rt.ops)
	}
}

// TestDetailPartialEndToEnd: through the real client and model, a Detail answer
// with a nulled check node and an errors[] member shows how much was read.
func TestDetailPartialEndToEnd(t *testing.T) {
	rt := &routedTransport{t: t, watch: failingWatchBody, failing: failingStepBody, detail: partialDetailBody}
	client, err := github.NewClient(github.Options{
		Token:      "test-token-not-real",
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rt,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = ui.New(context.Background(), client, ui.Options{ClosedDays: 14})
	m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = runCmd(m, m.Init())

	m, cmd := send(m, key("enter"))
	m = settle(m, cmd)
	if len(rt.detailVars) != 1 {
		t.Fatalf("Detail requests = %v, want exactly one", rt.detailVars)
	}
	if v := plain(m); !strings.Contains(v, "✗ lint") || !strings.Contains(v, "read 1 of 2 runs") ||
		!strings.Contains(v, "some details could not be loaded") {
		t.Errorf("the detail view must show the lint failure, \"read 1 of 2 runs\" and the partial notice:\n%s", v)
	}
}

// TestDetailKeptErrorEndToEnd: through the real client and model, a GraphQL
// error kept next to the data shows the notice and never a run count, both
// with every run read and with no checks at all (G-03-6, WR-05).
func TestDetailKeptErrorEndToEnd(t *testing.T) {
	const notice = "some details could not be loaded"
	for _, tc := range []struct {
		name string
		body string
		want []string
	}{
		{"every run read", keptErrorDetailBody, []string{"✗ lint", notice}},
		{"no checks", keptErrorNoChecksDetailBody, []string{"CHECKS  no checks", notice}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := &routedTransport{t: t, watch: failingWatchBody, failing: failingStepBody, detail: tc.body}
			client, err := github.NewClient(github.Options{
				Token:      "test-token-not-real",
				ClosedDays: 14,
				Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
				Transport:  rt,
			})
			if err != nil {
				t.Fatal(err)
			}
			var m tea.Model = ui.New(context.Background(), client, ui.Options{ClosedDays: 14})
			m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
			m = runCmd(m, m.Init())

			m, cmd := send(m, key("enter"))
			m = settle(m, cmd)
			if len(rt.detailVars) != 1 {
				t.Fatalf("Detail requests = %v, want exactly one", rt.detailVars)
			}
			v := plain(m)
			for _, w := range tc.want {
				if !strings.Contains(v, w) {
					t.Errorf("the detail view must contain %q:\n%s", w, v)
				}
			}
			if strings.Contains(v, " runs") {
				t.Errorf("the detail view must not show a run count:\n%s", v)
			}
		})
	}
}

func press(m tea.Model, keys ...string) tea.Model {
	for _, k := range keys {
		m, _ = send(m, key(k))
	}
	return m
}

// lookupToolsPR is the Lookup answer for octo/tools#42: open, approved, passing.
const lookupToolsPR = `{"pullRequest":{"id":"PR_42","number":42,"title":"Speed up the build","url":"https://github.com/octo/tools/pull/42",` +
	`"isDraft":false,"state":"OPEN","createdAt":"2026-10-10T08:00:00Z","closedAt":null,"mergedAt":null,"updatedAt":"2026-10-11T08:00:00Z",` +
	`"repository":{"nameWithOwner":"octo/tools"},"author":{"login":"alice"},"reviewDecision":"APPROVED",` +
	`"mergeStateStatus":"CLEAN","mergeable":"MERGEABLE","latestOpinionatedReviews":{"nodes":[]},` +
	`"reviewRequests":{"nodes":[]},"statusCheckRollup":{"state":"SUCCESS"}}}`

const lookupRate = `"rateLimit":{"limit":5000,"cost":1,"remaining":4985,"resetAt":"2026-10-11T13:00:00Z"}`

// lookupOKBody is the Lookup answer for octo/tools#42: open, approved, passing.
const lookupOKBody = `{"data":{` + lookupRate + `,"r0":` + lookupToolsPR + `}}`

// lookupGoneBody answers a Lookup of two refs: r0 is octo/tools#42 and r1 is
// null with a NOT_FOUND error at ["r1"] (acme/gone#5 does not exist).
const lookupGoneBody = `{"data":{` + lookupRate + `,"r0":` + lookupToolsPR + `,"r1":null},` +
	`"errors":[{"type":"NOT_FOUND","path":["r1"],"message":"Could not resolve to a Repository with the name 'acme/gone'."}]}`

// drain runs cmd and then, for at most 6 rounds, the Cmds that the messages
// made. step drops the Cmds that take longer than 250 ms, so a refresh tick never
// blocks it.
func drain(m tea.Model, cmd tea.Cmd) tea.Model {
	cmds := []tea.Cmd{cmd}
	for round := 0; round < 6 && len(cmds) > 0; round++ {
		var next []tea.Cmd
		for _, c := range cmds {
			var n []tea.Cmd
			m, n = step(m, c)
			next = append(next, n...)
		}
		cmds = next
	}
	return m
}

// addViaPrompt adds a PR the way the user does: a opens the prompt, the text is
// pasted, enter checks and saves it.
func addViaPrompt(t *testing.T, m tea.Model, text string) tea.Model {
	t.Helper()
	m, _ = send(m, key("a"))
	m, _ = send(m, tea.PasteMsg{Content: text})
	m, cmd := send(m, key("enter"))
	return drain(m, cmd)
}

// TestAddEndToEnd is the phase tracer: a real github.Client, a real
// watchlist.Store in a temp dir and the real ui.Model. Pasting a PR link checks
// the PR with one variables-only Lookup request, saves it and lists it with its
// full status, the + marker and the cursor on it.
func TestAddEndToEnd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
	store := watchlist.New(path)
	rt := &routedTransport{t: t, watch: fixtureHead + fixtureNodes + fixtureTail, lookup: lookupOKBody}
	client, err := github.NewClient(github.Options{
		Token:      "test-token-not-real",
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rt,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = ui.New(context.Background(), client, ui.Options{ClosedDays: 14, Watchlist: store})
	m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = runCmd(m, m.Init())

	m = addViaPrompt(t, m, "https://github.com/octo/tools/pull/42/files\n")

	if len(rt.lookupVars) != 1 {
		t.Fatalf("Lookup requests = %d, want exactly 1 (%v)", len(rt.lookupVars), rt.ops)
	}
	v := rt.lookupVars[0]
	if v["o0"] != "octo" || v["n0"] != "tools" || v["p0"] != float64(42) || len(v) != 3 {
		t.Errorf("Lookup variables = %v, want {o0: octo, n0: tools, p0: 42}", v)
	}
	for _, bad := range []string{"octo", "tools", "42"} {
		if strings.Contains(rt.lookupQuerys[0], bad) {
			t.Errorf("the Lookup query text holds %q: owner, repo and number travel only as variables", bad)
		}
	}

	view := plain(m)
	row := lineWith(view, "octo/tools#42")
	if !strings.HasPrefix(row, ">") {
		t.Errorf("the cursor is not on the new row:\n%s", view)
	}
	for _, want := range []string{"+ octo/tools#42", "✓ approved", "✓ passing"} {
		if !strings.Contains(row, want) {
			t.Errorf("row = %q, want it to contain %q", row, want)
		}
	}
	if strings.Contains(view, "add PR:") {
		t.Errorf("the prompt is still open:\n%s", view)
	}

	refs, _, err := store.Load()
	if err != nil || len(refs) != 1 || refs[0] != (domain.Ref{Repo: "octo/tools", Number: 42}) {
		t.Errorf("store.Load() = %v, %v, want [octo/tools#42]", refs, err)
	}
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("watchlist.json mode = %v, want 0600", fi.Mode().Perm())
		}
	}
}

// addTestClient is a github.Client over the routed fixtures for the add tests.
func addTestClient(t *testing.T) *github.Client {
	t.Helper()
	rt := &routedTransport{t: t, watch: fixtureHead + fixtureNodes + fixtureTail, lookup: lookupOKBody}
	client, err := github.NewClient(github.Options{
		Token:      "test-token-not-real",
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rt,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// TestAddOverCorruptEndToEnd: adding a PR over an unparseable watchlist.json
// starts a new list, keeps the old bytes in a backup and says so in the footer
// (D-04): the real store, the real ui.Model.
func TestAddOverCorruptEndToEnd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	const broken = "{not json"
	if err := os.WriteFile(path, []byte(broken), 0o600); err != nil {
		t.Fatal(err)
	}
	store := watchlist.New(path)
	var m tea.Model = ui.New(context.Background(), addTestClient(t), ui.Options{ClosedDays: 14, Watchlist: store})
	m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = runCmd(m, m.Init())

	m = addViaPrompt(t, m, "octo/tools#42")

	backups, err := filepath.Glob(path + ".corrupt-*")
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups = %v, %v, want exactly one", backups, err)
	}
	got, err := os.ReadFile(backups[0])
	if err != nil || string(got) != broken {
		t.Errorf("backup holds %q, %v, want %q", got, err, broken)
	}

	view := plain(m)
	if row := lineWith(view, "+ octo/tools#42"); !strings.HasPrefix(row, ">") {
		t.Errorf("the new row is not listed with the cursor on it:\n%s", view)
	}
	lines := strings.Split(view, "\n")
	last := lines[len(lines)-1]
	if want := "previous list saved to " + filepath.Base(backups[0]); !strings.Contains(last, want) {
		t.Errorf("last line = %q, want it to contain %q", last, want)
	}
	refs, _, err := store.Load()
	if err != nil || len(refs) != 1 || refs[0] != (domain.Ref{Repo: "octo/tools", Number: 42}) {
		t.Errorf("store.Load() = %v, %v, want [octo/tools#42]", refs, err)
	}
}

// TestAddVersionMismatchEndToEnd: adding a PR over a watchlist.json that is a JSON
// object whose version is not 1, or that has no version, changes nothing on disk
// and says why in the prompt (D-04, user-confirmed 2026-10-13): the real store,
// the real ui.Model.
func TestAddVersionMismatchEndToEnd(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"version2", `{"version": 2, "prs": [{"repo":"a/b","number":1}]}`,
			"✗ could not save (watchlist.json is version 2; this prv writes version 1)"},
		{"noversion", `{"prs": [{"repo":"a/b","number":1}]}`,
			"✗ could not save (watchlist.json has no version number; this prv writes version 1)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(c.body), 0o600); err != nil {
				t.Fatal(err)
			}
			var m tea.Model = ui.New(context.Background(), addTestClient(t), ui.Options{ClosedDays: 14, Watchlist: watchlist.New(path)})
			m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
			m = runCmd(m, m.Init())

			m = addViaPrompt(t, m, "octo/tools#42")

			view := plain(m)
			lines := strings.Split(view, "\n")
			last := lines[len(lines)-1]
			if !strings.HasPrefix(last, "add PR:") || !strings.Contains(last, c.want) {
				t.Errorf("last line = %q, want the prompt with %q", last, c.want)
			}
			if strings.Contains(view, "previous list saved to") {
				t.Errorf("a refused save shows the started-over notice:\n%s", view)
			}
			if row := lineWith(view, "+ octo/tools#42"); row != "" {
				t.Errorf("octo/tools#42 is listed although the save was refused: %q", row)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != c.body {
				t.Errorf("watchlist.json = %q, %v, want it unchanged: %q", got, err, c.body)
			}
			if backups, _ := filepath.Glob(path + ".corrupt-*"); len(backups) != 0 {
				t.Errorf("backups = %v, want none", backups)
			}
		})
	}
}

// TestRestartEndToEnd: a PR added in one session is listed again after a restart,
// with its full status and the + marker. A saved PR that discovery already found
// is one row and costs no Lookup; only the other saved PR is looked up.
func TestRestartEndToEnd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
	now := func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) }

	// Session 1: add octo/tools#42 through the prompt; no saved list is read.
	rt1 := &routedTransport{t: t, watch: fixtureHead + fixtureNodes + fixtureTail, lookup: lookupOKBody}
	c1, err := github.NewClient(github.Options{Token: "test-token-not-real", ClosedDays: 14, Now: now, Transport: rt1})
	if err != nil {
		t.Fatal(err)
	}
	var m1 tea.Model = ui.New(context.Background(), c1, ui.Options{ClosedDays: 14, Watchlist: watchlist.New(path)})
	m1, _ = send(m1, tea.WindowSizeMsg{Width: 120, Height: 40})
	m1 = runCmd(m1, m1.Init())
	_ = addViaPrompt(t, m1, "octo/tools#42")

	// Another saved PR that discovery returns, saved in another letter case.
	if _, err := watchlist.New(path).Add(domain.Ref{Repo: "ACME/API", Number: 481}); err != nil {
		t.Fatal(err)
	}

	// Session 2: a new client, a new model, the same file.
	rt2 := &routedTransport{t: t, watch: fixtureHead + fixtureNodes + fixtureTail, lookup: lookupOKBody}
	c2, err := github.NewClient(github.Options{Token: "test-token-not-real", ClosedDays: 14, Now: now, Transport: rt2, Watch: watchlist.New(path)})
	if err != nil {
		t.Fatal(err)
	}
	var m2 tea.Model = ui.New(context.Background(), c2, ui.Options{ClosedDays: 14, Watchlist: watchlist.New(path)})
	m2, _ = send(m2, tea.WindowSizeMsg{Width: 120, Height: 40})
	m2 = runCmd(m2, m2.Init())

	if want := []string{"Watch", "Lookup"}; !slices.Equal(rt2.ops, want) {
		t.Fatalf("requests = %v, want %v", rt2.ops, want)
	}
	if v := rt2.lookupVars[0]; v["o0"] != "octo" || v["n0"] != "tools" || v["p0"] != float64(42) {
		t.Errorf("Lookup variables = %v, want {o0: octo, n0: tools, p0: 42}", v)
	} else if _, ok := v["o1"]; ok {
		t.Errorf("Lookup asked for a second ref although discovery returned acme/api#481: %v", v)
	}

	view := plain(m2)
	if row := lineWith(view, "octo/tools#42"); !strings.Contains(row, "+ octo/tools#42") || !strings.Contains(row, "✓ passing") {
		t.Errorf("octo/tools#42 row = %q, want the + marker and ✓ passing\n%s", row, view)
	}
	if n := strings.Count(view, "acme/api#481"); n != 1 {
		t.Errorf("acme/api#481 is on %d lines, want exactly 1\n%s", n, view)
	}
	if row := lineWith(view, "acme/api#481"); !strings.Contains(row, "+ acme/api#481") {
		t.Errorf("acme/api#481 row = %q, want the + marker", row)
	}
}

// TestWatchWarnings: watch-list problems found at startup become warnings
// ("watch list: ..."); a missing file has none, and the copy of a corrupt file
// is made here without changing the file.
func TestWatchWarnings(t *testing.T) {
	write := func(t *testing.T, body string) *watchlist.Store {
		t.Helper()
		path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return watchlist.New(path)
	}

	missing := watchlist.New(filepath.Join(t.TempDir(), "prv", "watchlist.json"))
	if got := watchWarnings(missing); len(got) != 0 {
		t.Errorf("missing file: warnings = %v, want none", got)
	}

	const bad = "{not json"
	corrupt := write(t, bad)
	got := watchWarnings(corrupt)
	if len(got) != 1 || !strings.HasPrefix(got[0], "watch list: ") || !strings.Contains(got[0], "corrupt-") {
		t.Fatalf("corrupt file: warnings = %v, want one 'watch list: ...' naming the corrupt- copy", got)
	}
	if data, _ := os.ReadFile(corrupt.Path()); string(data) != bad {
		t.Errorf("startup changed the corrupt file: %q", data)
	}
	if backups, _ := filepath.Glob(corrupt.Path() + ".corrupt-*"); len(backups) != 1 {
		t.Errorf("backups = %v, want one", backups)
	}

	skipped := write(t, `{"version":1,"prs":[{"repo":"a/a","number":1},{"repo":"bad","number":1}]}`)
	want := []string{"watch list: watchlist.json: 1 entry skipped (not a PR reference)"}
	if got := watchWarnings(skipped); !slices.Equal(got, want) {
		t.Errorf("skipped entry: warnings = %v, want %v", got, want)
	}
}

// TestUnavailableEndToEnd: a saved PR that GitHub answers NOT_FOUND is a row at
// the top with its reason, and the other PRs still show every column.
func TestUnavailableEndToEnd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
	store := watchlist.New(path)
	for _, ref := range []domain.Ref{{Repo: "octo/tools", Number: 42}, {Repo: "acme/gone", Number: 5}} {
		if _, err := store.Add(ref); err != nil {
			t.Fatal(err)
		}
	}
	rt := &routedTransport{t: t, watch: fixtureHead + fixtureNodes + fixtureTail, lookup: lookupGoneBody}
	client, err := github.NewClient(github.Options{
		Token:      "test-token-not-real",
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rt,
		Watch:      store,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = ui.New(context.Background(), client, ui.Options{ClosedDays: 14, Watchlist: store})
	m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = runCmd(m, m.Init())

	view := plain(m)
	ls := strings.Split(view, "\n")
	if len(ls) < 3 || !strings.Contains(ls[2], "acme/gone#5") ||
		!strings.Contains(ls[2], "⚠ unavailable: not found (deleted, renamed, or no access)") {
		t.Errorf("the first row under the column header is not the unavailable acme/gone#5:\n%s", view)
	}
	if row := lineWith(view, "octo/tools#42"); !strings.Contains(row, "+ octo/tools#42") || !strings.Contains(row, "✓ passing") {
		t.Errorf("octo/tools#42 row = %q, want the + marker and ✓ passing", row)
	}
	for _, want := range []string{"acme/api#481", "acme/web#12"} {
		if !strings.Contains(view, want) {
			t.Errorf("the view lacks %s:\n%s", want, view)
		}
	}
	if !strings.Contains(ls[0], "1 unavailable") {
		t.Errorf("header = %q, want '1 unavailable'", ls[0])
	}
}

// lookupClosedBody answers a Lookup of octo/old#7: closed on 2026-08-01, more
// than two months before the test clock and far outside a 14 day window.
const lookupClosedBody = `{"data":{` + lookupRate + `,` +
	`"r0":{"pullRequest":{"id":"PR_7","number":7,"title":"Dropped idea","url":"https://github.com/octo/old/pull/7",` +
	`"isDraft":false,"state":"CLOSED","createdAt":"2026-07-01T08:00:00Z","closedAt":"2026-08-01T00:00:00Z","mergedAt":null,` +
	`"updatedAt":"2026-08-01T00:00:00Z","repository":{"nameWithOwner":"octo/old"},"author":{"login":"alice"},` +
	`"reviewDecision":null,"mergeStateStatus":"UNKNOWN","mergeable":"UNKNOWN","latestOpinionatedReviews":{"nodes":[]},` +
	`"reviewRequests":{"nodes":[]},"statusCheckRollup":null}}}}`

// TestClosedSavedPREndToEnd: a saved PR that closed long ago stays listed, one
// row however often the list is refreshed, and a refresh that fails as a whole
// keeps the row and the saved entry.
func TestClosedSavedPREndToEnd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
	store := watchlist.New(path)
	if _, err := store.Add(domain.Ref{Repo: "octo/old", Number: 7}); err != nil {
		t.Fatal(err)
	}
	rt := &routedTransport{t: t, watch: fixtureHead + fixtureNodes + fixtureTail, lookup: lookupClosedBody}
	client, err := github.NewClient(github.Options{
		Token:      "test-token-not-real",
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rt,
		Watch:      store,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = ui.New(context.Background(), client, ui.Options{ClosedDays: 14, Watchlist: store})
	m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = runCmd(m, m.Init())

	view := plain(m)
	if row := lineWith(view, "octo/old#7"); !strings.Contains(row, "+ octo/old#7") || !strings.Contains(row, "closed") {
		t.Errorf("octo/old#7 row = %q, want the + marker and 'closed'\n%s", row, view)
	}
	refresh := func() {
		var cmd tea.Cmd
		m, cmd = send(m, key("r"))
		m = drain(m, cmd)
	}
	refresh()
	refresh()
	if n := strings.Count(plain(m), "octo/old#7"); n != 1 {
		t.Errorf("octo/old#7 is on %d lines after two refreshes, want exactly 1\n%s", n, plain(m))
	}

	rt.lookupStatus = 502
	refresh()
	view = plain(m)
	if row := lineWith(view, "octo/old#7"); !strings.Contains(row, "closed") || strings.Contains(row, "unavailable") {
		t.Errorf("after a failed refresh the row = %q, want the last data kept\n%s", row, view)
	}
	if !strings.Contains(strings.Split(view, "\n")[0], "refresh failed") {
		t.Errorf("header = %q, want 'refresh failed'", strings.Split(view, "\n")[0])
	}
	if refs, _, err := store.Load(); err != nil || len(refs) != 1 || refs[0] != (domain.Ref{Repo: "octo/old", Number: 7}) {
		t.Errorf("store.Load() = %v, %v, want [octo/old#7]", refs, err)
	}
}

// TestRemoveEndToEnd: d then y takes a saved PR off the list and out of
// watchlist.json, with no request to GitHub.
func TestRemoveEndToEnd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
	store := watchlist.New(path)
	if _, err := store.Add(domain.Ref{Repo: "octo/tools", Number: 42}); err != nil {
		t.Fatal(err)
	}
	rt := &routedTransport{t: t, watch: fixtureHead + fixtureNodes + fixtureTail, lookup: lookupOKBody}
	client, err := github.NewClient(github.Options{
		Token:      "test-token-not-real",
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rt,
		Watch:      store,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = ui.New(context.Background(), client, ui.Options{ClosedDays: 14, Watchlist: store})
	m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = runCmd(m, m.Init())

	view := plain(m)
	if row := lineWith(view, "octo/tools#42"); !strings.HasPrefix(row, ">") || !strings.Contains(row, "+ octo/tools#42") {
		t.Fatalf("octo/tools#42 is not the selected saved row:\n%s", view)
	}
	before := len(rt.ops)

	m, _ = send(m, key("d"))
	ls := strings.Split(plain(m), "\n")
	if got, want := ls[len(ls)-1], "remove octo/tools#42 from watch list? y/n"; got != want {
		t.Errorf("last line = %q, want %q", got, want)
	}
	m, cmd := send(m, key("y"))
	m = drain(m, cmd)

	view = plain(m)
	ls = strings.Split(view, "\n")
	for _, l := range ls[:len(ls)-1] {
		if strings.Contains(l, "octo/tools#42") {
			t.Errorf("octo/tools#42 is still listed:\n%s", view)
		}
	}
	if last := ls[len(ls)-1]; !strings.Contains(last, "removed octo/tools#42") {
		t.Errorf("footer = %q, want it to hold 'removed octo/tools#42'", last)
	}
	if refs, _, err := store.Load(); err != nil || len(refs) != 0 {
		t.Errorf("store.Load() = %v, %v, want no ref", refs, err)
	}
	if n := len(rt.ops) - before; n != 0 {
		t.Errorf("the removal sent %d GraphQL requests (%v), want none", n, rt.ops[before:])
	}
}

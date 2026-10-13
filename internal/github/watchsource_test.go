package github

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/sbartram/prv/internal/domain"
	"github.com/sbartram/prv/internal/watchlist"
)

// wsLookupBody answers a Lookup request: for every o<i>/n<i>/p<i> in vars, alias
// r<i> is a loaded OPEN PR with that owner, repo and number, plus a rateLimit
// block with cost 1.
func wsLookupBody(t *testing.T, vars map[string]any) []byte {
	t.Helper()
	aliases := map[string]any{}
	for k, v := range vars {
		if len(k) < 2 || k[0] != 'o' {
			continue
		}
		i := k[1:]
		owner, _ := v.(string)
		repo, _ := vars["n"+i].(string)
		num, _ := vars["p"+i].(float64)
		full := owner + "/" + repo
		aliases["r"+i] = lookupPRNode("watched", full, "W_"+owner+"_"+repo+"_"+strconv.Itoa(int(num)), int(num), "SUCCESS")
	}
	return lookupBody(t, aliases, nil)
}

// wsTransport answers "Watch" with search_page2.json, "FailingChecks" with no
// nodes and "Lookup" with wsLookupBody. It records the variables of every
// Lookup request.
func wsTransport(t *testing.T) (http.RoundTripper, *[]map[string]any) {
	t.Helper()
	var lookups []map[string]any
	rt := rtFunc(func(req *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(req.Body)
		switch op := opName(raw); op {
		case "Watch":
			return jsonResp(req, 200, fixtureBytes(t, "search_page2.json")), nil
		case "FailingChecks":
			return jsonResp(req, 200, []byte(failingNoneBody)), nil
		case "Lookup":
			_, vars := lookupRequest(t, raw)
			lookups = append(lookups, vars)
			return jsonResp(req, 200, wsLookupBody(t, vars)), nil
		default:
			t.Errorf("unexpected operation %q", op)
			return jsonResp(req, 500, []byte(`{}`)), nil
		}
	})
	return rt, &lookups
}

// wsClient builds a client over rt. src is the saved list, or nil for none.
func wsClient(t *testing.T, rt http.RoundTripper, src WatchSource) *Client {
	t.Helper()
	opts := Options{
		Token:      secretToken,
		ClosedDays: 14,
		Now:        func() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) },
		Transport:  rt,
	}
	if src != nil { // never put a nil *Store in the interface
		opts.Watch = src
	}
	c, err := NewClient(opts)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// wsStoreAt returns two stores on one path: two prv panes.
func wsStoreAt(t *testing.T) (pane1, pane2 *watchlist.Store, path string) {
	t.Helper()
	path = filepath.Join(t.TempDir(), "prv", "watchlist.json")
	return watchlist.New(path), watchlist.New(path), path
}

// wsFind returns the PR with ref r (by key) in snap.
func wsFind(snap domain.Snapshot, r domain.Ref) (domain.PR, bool) {
	for _, pr := range snap.PRs {
		if pr.Ref.Key() == r.Key() {
			return pr, true
		}
	}
	return domain.PR{}, false
}

func TestFetchRereadsWatchList(t *testing.T) {
	s1, s2, _ := wsStoreAt(t)
	a := domain.Ref{Repo: "octo/a", Number: 1}
	b := domain.Ref{Repo: "octo/b", Number: 2}
	if _, err := s1.Add(a); err != nil {
		t.Fatal(err)
	}
	rt, lookups := wsTransport(t)
	c := wsClient(t, rt, s1)

	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if pr, ok := wsFind(snap, a); !ok || !pr.Manual {
		t.Errorf("Fetch 1: octo/a#1 = %+v, %v, want a Manual row", pr, ok)
	}

	if _, err := s2.Add(b); err != nil { // another pane adds
		t.Fatal(err)
	}
	snap, err = c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []domain.Ref{a, b} {
		if pr, ok := wsFind(snap, r); !ok || !pr.Manual {
			t.Errorf("Fetch 2: %v = %+v, %v, want a Manual row", r, pr, ok)
		}
	}

	if _, err := s2.Remove(a); err != nil { // another pane removes
		t.Fatal(err)
	}
	snap, err = c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := wsFind(snap, a); ok {
		t.Error("Fetch 3: octo/a#1 is still listed after the other pane removed it")
	}
	if _, ok := wsFind(snap, b); !ok {
		t.Error("Fetch 3: octo/b#2 is missing")
	}
	last := (*lookups)[len(*lookups)-1]
	if len(last) != 3 || last["o0"] != "octo" || last["n0"] != "b" || last["p0"] != float64(2) {
		t.Errorf("Fetch 3 Lookup variables = %v, want only {o0: octo, n0: b, p0: 2}", last)
	}
}

func TestFetchKeepsRefsOnReadFailure(t *testing.T) {
	s1, _, path := wsStoreAt(t)
	a := domain.Ref{Repo: "octo/a", Number: 1}
	if _, err := s1.Add(a); err != nil {
		t.Fatal(err)
	}
	rt, lookups := wsTransport(t)
	c := wsClient(t, rt, s1)
	if _, err := c.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}

	const broken = "{broken"
	if err := os.WriteFile(path, []byte(broken), 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := wsFind(snap, a); !ok {
		t.Error("a broken file emptied the list: octo/a#1 is gone")
	}
	if !slices.Contains(snap.Warnings, "Watch list could not be read") {
		t.Errorf("warnings = %v, want Watch list could not be read", snap.Warnings)
	}
	if len(*lookups) != 2 {
		t.Errorf("Lookup requests = %d, want 2 (looked up again from the kept list)", len(*lookups))
	}
	if got, _ := os.ReadFile(path); string(got) != broken {
		t.Errorf("the refresh changed the broken file: %q", got)
	}
	if backups, _ := filepath.Glob(path + ".corrupt-*"); len(backups) != 1 {
		t.Errorf("backups = %v, want one", backups)
	}
}

func TestFetchWithoutWatchSource(t *testing.T) {
	rt, lookups := wsTransport(t)
	c := wsClient(t, rt, nil)
	if _, err := c.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(*lookups) != 0 {
		t.Errorf("Lookup requests = %d, want none without a watch source", len(*lookups))
	}
}

func TestFetchSavedFoundByDiscovery(t *testing.T) {
	s1, _, _ := wsStoreAt(t)
	saved := domain.Ref{Repo: "ACME/WEB", Number: 77}
	if _, err := s1.Add(saved); err != nil {
		t.Fatal(err)
	}
	rt, lookups := wsTransport(t)
	snap, err := wsClient(t, rt, s1).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(*lookups) != 0 {
		t.Errorf("Lookup requests = %d, want none: discovery already returned acme/web#77", len(*lookups))
	}
	pr, ok := wsFind(snap, domain.Ref{Repo: "acme/web", Number: 77})
	if !ok || !pr.Auto || !pr.Manual || pr.Watched != saved {
		t.Errorf("acme/web#77 = %+v, %v, want Auto and Manual with Watched %v", pr, ok, saved)
	}
}

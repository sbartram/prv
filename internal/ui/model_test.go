package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sbartram/prv/internal/domain"
)

type fakeFetcher struct {
	snap     domain.Snapshot
	err      error
	panicVal any
	calls    int
}

func (f *fakeFetcher) Fetch(context.Context) (domain.Snapshot, error) {
	f.calls++
	if f.panicVal != nil {
		panic(f.panicVal)
	}
	return f.snap, f.err
}

// hintErr is an error that carries a hint, like github.Error does.
type hintErr struct{ msg, hint string }

func (e *hintErr) Error() string { return e.msg }
func (e *hintErr) Hint() string  { return e.hint }

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

func key(s string) tea.KeyPressMsg {
	if s == "ctrl+c" {
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

func plain(m tea.Model) string { return ansi.Strip(m.View().Content) }

func newModel(f Fetcher, closedDays int) tea.Model {
	var m tea.Model = New(context.Background(), f, Options{ClosedDays: closedDays})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return m
}

func onePR() domain.Snapshot {
	return domain.Snapshot{Viewer: "octo", PRs: []domain.PR{{
		ID: "PR_1", Ref: domain.Ref{Repo: "acme/api", Number: 1}, Title: "A change", Author: "octo",
		UpdatedAt: time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC),
	}}}
}

// modelIn builds a model in each of the four states.
func modelIn(t *testing.T, state string) tea.Model {
	t.Helper()
	switch state {
	case "loading":
		return newModel(&fakeFetcher{}, 14)
	case "list":
		m := newModel(&fakeFetcher{}, 14)
		m, _ = m.Update(fetchedMsg{snap: onePR()})
		return m
	case "empty":
		m := newModel(&fakeFetcher{}, 14)
		m, _ = m.Update(fetchedMsg{snap: domain.Snapshot{Viewer: "octo"}})
		return m
	case "error":
		m := newModel(&fakeFetcher{}, 14)
		m, _ = m.Update(fetchedMsg{err: errors.New("boom")})
		return m
	}
	t.Fatalf("unknown state %q", state)
	return nil
}

func TestLoadingView(t *testing.T) {
	m := newModel(&fakeFetcher{}, 14)
	if v := plain(m); !strings.Contains(v, "Fetching your PRs…") {
		t.Errorf("loading view = %q, want the fetching message", v)
	}
}

func TestQuitInEveryState(t *testing.T) {
	for _, state := range []string{"loading", "list", "empty", "error"} {
		for _, k := range []string{"q", "ctrl+c"} {
			t.Run(state+"/"+k, func(t *testing.T) {
				_, cmd := modelIn(t, state).Update(key(k))
				if cmd == nil {
					t.Fatal("no command returned")
				}
				if _, ok := cmd().(tea.QuitMsg); !ok {
					t.Error("command did not produce tea.QuitMsg")
				}
			})
		}
	}
}

func TestRefreshKey(t *testing.T) {
	// build returns a sized model in the named state and the fetcher behind it.
	build := func(state string) (tea.Model, *fakeFetcher) {
		f := &fakeFetcher{snap: onePR()}
		var m tea.Model = New(context.Background(), f, Options{ClosedDays: 14})
		m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		switch state {
		case "list":
			m, _ = m.Update(fetchedMsg{snap: onePR()})
		case "empty":
			m, _ = m.Update(fetchedMsg{snap: domain.Snapshot{Viewer: "octo"}})
		case "error":
			m, _ = m.Update(fetchedMsg{err: errors.New("boom")})
		}
		return m, f
	}

	t.Run("loading: the first fetch is running, r does nothing", func(t *testing.T) {
		m, f := build("loading")
		m2, cmd := m.Update(key("r"))
		if cmd != nil {
			t.Error("r while the first fetch runs returned a command")
		}
		if f.calls != 0 {
			t.Errorf("fetch calls = %d, want 0", f.calls)
		}
		if got := m2.(Model).gen; got != 0 {
			t.Errorf("generation = %d, want 0", got)
		}
	})
	for _, state := range []string{"list", "empty"} {
		t.Run(state+": one fetch, merged while it runs", func(t *testing.T) {
			m, f := build(state)
			m, cmd := m.Update(key("r"))
			if cmd == nil {
				t.Fatalf("r in the %s state returned no command", state)
			}
			if got := m.(Model).gen; got != 1 {
				t.Errorf("generation after r = %d, want 1", got)
			}
			if _, again := m.Update(key("r")); again != nil {
				t.Error("r while a refresh runs returned a command")
			}
			m = runCmd(m, cmd)
			if f.calls != 1 {
				t.Errorf("fetch calls = %d, want exactly 1", f.calls)
			}
			if v := plain(m); !strings.Contains(v, "acme/api#1") {
				t.Errorf("after the refresh the screen is %q, want the list", v)
			}
		})
	}
	t.Run("error retries once", func(t *testing.T) {
		m, f := build("error")
		m, cmd := m.Update(key("r"))
		if cmd == nil {
			t.Fatal("r in the error state returned no command")
		}
		if v := plain(m); !strings.Contains(v, "Fetching your PRs…") {
			t.Errorf("after r the screen is %q, want the spinner", v)
		}
		// A second r while the retry runs does nothing.
		if _, again := m.Update(key("r")); again != nil {
			t.Error("r while a retry is running returned a command")
		}
		m = runCmd(m, cmd)
		if f.calls != 1 {
			t.Errorf("fetch calls = %d, want exactly 1", f.calls)
		}
		if v := plain(m); !strings.Contains(v, "acme/api#1") {
			t.Errorf("after the retry the screen is %q, want the list", v)
		}
	})
	t.Run("retry that fails again returns to the panel", func(t *testing.T) {
		f := &fakeFetcher{err: errors.New("still broken")}
		var m tea.Model = New(context.Background(), f, Options{ClosedDays: 14})
		m, _ = m.Update(fetchedMsg{err: errors.New("boom")})
		m, cmd := m.Update(key("r"))
		m = runCmd(m, cmd)
		if v := plain(m); !strings.Contains(v, "Could not load your PRs") || !strings.Contains(v, "still broken") {
			t.Errorf("screen = %q", v)
		}
	})
	t.Run("j and k still move", func(t *testing.T) {
		snap := domain.Snapshot{Viewer: "octo", PRs: []domain.PR{
			{ID: "1", Ref: domain.Ref{Repo: "a/b", Number: 1}, Title: "one"},
			{ID: "2", Ref: domain.Ref{Repo: "a/b", Number: 2}, Title: "two"},
		}}
		var m tea.Model = New(context.Background(), &fakeFetcher{}, Options{ClosedDays: 14})
		m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		m, _ = m.Update(fetchedMsg{snap: snap})
		marked := func() string {
			for _, l := range strings.Split(plain(m), "\n") {
				if strings.HasPrefix(l, ">") {
					return l
				}
			}
			return ""
		}
		for _, k := range []string{"j", "down"} {
			m, _ = m.Update(key("k"))
			m, _ = m.Update(key(k))
			if !strings.Contains(marked(), "a/b#2") {
				t.Errorf("after %s the marker is on %q, want a/b#2", k, marked())
			}
		}
		for _, k := range []string{"k", "up"} {
			m, _ = m.Update(key("j"))
			if k == "up" {
				m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
			} else {
				m, _ = m.Update(key(k))
			}
			if !strings.Contains(marked(), "a/b#1") {
				t.Errorf("after %s the marker is on %q, want a/b#1", k, marked())
			}
		}
	})
}

func TestEmptyStateNamesAccount(t *testing.T) {
	for _, days := range []int{14, 30} {
		t.Run(fmt.Sprint(days), func(t *testing.T) {
			m := newModel(&fakeFetcher{}, days)
			m, _ = m.Update(fetchedMsg{snap: domain.Snapshot{Viewer: "octo"}})
			want := fmt.Sprintf("No open PRs authored by or assigned to @octo, and none closed in the last %d days.", days)
			v := plain(m)
			if !strings.Contains(v, want) {
				t.Errorf("view = %q, want it to contain %q", v, want)
			}
			if !strings.Contains(v, "q quit") {
				t.Errorf("view = %q, want the q quit footer", v)
			}
			if strings.Contains(v, "r retry") {
				t.Errorf("the empty state must not offer retry: %q", v)
			}
		})
	}
}

func TestErrorPanelShowsHint(t *testing.T) {
	t.Run("with hint", func(t *testing.T) {
		m := newModel(&fakeFetcher{}, 14)
		err := fmt.Errorf("wrapped: %w", &hintErr{msg: "GitHub request failed: HTTP 401", hint: "Run `gh auth login`."})
		m, _ = m.Update(fetchedMsg{err: err})
		v := plain(m)
		for _, want := range []string{"Could not load your PRs", "GitHub request failed: HTTP 401", "Run `gh auth login`.", "r retry · q quit"} {
			if !strings.Contains(v, want) {
				t.Errorf("view missing %q:\n%s", want, v)
			}
		}
	})
	t.Run("without hint", func(t *testing.T) {
		m := newModel(&fakeFetcher{}, 14)
		m, _ = m.Update(fetchedMsg{err: errors.New("plain failure")})
		v := plain(m)
		if !strings.Contains(v, "plain failure") || !strings.Contains(v, "r retry · q quit") {
			t.Errorf("view = %q", v)
		}
		if strings.Contains(v, "gh auth") {
			t.Errorf("an error without Hint() must show no hint line:\n%s", v)
		}
		if !strings.Contains(v, "retrying at ") {
			t.Errorf("panel does not say when it retries:\n%s", v)
		}
		// Exactly: title, blank, message, blank, retry line, blank, footer.
		if got := strings.Count(strings.TrimRight(v, "\n"), "\n"); got != 6 {
			t.Errorf("panel has %d line breaks, want 6:\n%s", got, v)
		}
	})
}

func TestFetchPanicShowsErrorPanel(t *testing.T) {
	var m tea.Model = New(context.Background(), &fakeFetcher{panicVal: "secret-panic-value"}, Options{ClosedDays: 14})
	m = runCmd(m, m.Init())
	v := plain(m)
	if !strings.Contains(v, "Could not load your PRs") || !strings.Contains(v, "unexpected internal error while loading PRs") {
		t.Errorf("view = %q, want the generic error panel", v)
	}
	if strings.Contains(v, "secret-panic-value") {
		t.Errorf("view shows the panic value:\n%s", v)
	}
}

func TestFooterNotes(t *testing.T) {
	list := func(truncated bool, warnings []string) string {
		snap := onePR()
		snap.Truncated = truncated
		snap.Warnings = warnings
		m := newModel(&fakeFetcher{}, 14)
		m, _ = m.Update(fetchedMsg{snap: snap})
		return plain(m)
	}
	if v := list(true, nil); !strings.Contains(v, "list truncated") {
		t.Errorf("truncated list has no 'list truncated' note:\n%s", v)
	}
	v := list(false, nil)
	if strings.Contains(v, "list truncated") || strings.Contains(v, "some data could not be loaded") {
		t.Errorf("a complete list must carry no notes:\n%s", v)
	}
	if !strings.Contains(v, "j/k move · q quit") {
		t.Errorf("list footer missing:\n%s", v)
	}
	if v := list(false, []string{"Some data could not be loaded: x"}); !strings.Contains(v, "some data could not be loaded") {
		t.Errorf("warnings produce no note:\n%s", v)
	}
	if v := list(true, []string{"Only part of the list could be loaded"}); !strings.Contains(v, "list truncated") || !strings.Contains(v, "some data could not be loaded") {
		t.Errorf("both notes expected:\n%s", v)
	}
}

func TestCanceledFetchIgnored(t *testing.T) {
	for _, err := range []error{context.Canceled, fmt.Errorf("fetching PRs: %w", context.Canceled)} {
		m := newModel(&fakeFetcher{}, 14)
		m, cmd := m.Update(fetchedMsg{err: err})
		if cmd != nil {
			t.Error("a canceled fetch returned a command")
		}
		v := plain(m)
		if strings.Contains(v, "Could not load your PRs") || !strings.Contains(v, "Fetching your PRs…") {
			t.Errorf("a canceled fetch must leave the screen alone, got:\n%s", v)
		}
	}
}

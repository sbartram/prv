package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sbartram/prv/internal/domain"
)

// fakeOpener records the URLs it is asked to open and can fail or panic.
type fakeOpener struct {
	urls     []string
	err      error
	panicVal any
}

func (o *fakeOpener) Open(url string) error {
	o.urls = append(o.urls, url)
	if o.panicVal != nil {
		panic(o.panicVal)
	}
	return o.err
}

const prURL = "https://github.com/acme/api/pull/1"

// openSnap has one PR, acme/api#1, with the given URL.
func openSnap(url string) domain.Snapshot {
	s := onePR()
	s.PRs[0].URL = url
	return s
}

// openModel builds a sized model on the list screen with o as its opener.
func openModel(url string, o Opener) tea.Model {
	return filterModelOpts(120, 40, openSnap(url), Options{ClosedDays: 14, Now: fixedNow, Opener: o})
}

func TestSafeURL(t *testing.T) {
	for _, tt := range []struct {
		name string
		raw  string
		want string // "" means refused
	}{
		{"github pull request", "https://github.com/acme/api/pull/1", "https://github.com/acme/api/pull/1"},
		{"host in any letter case", "https://GitHub.com/acme/api/pull/1", "https://GitHub.com/acme/api/pull/1"},
		{"github.com as a prefix of the host", "https://github.com.evil.com/x", ""},
		{"github.com as user info", "https://github.com@evil.com/x", ""},
		{"plain http", "http://github.com/x", ""},
		{"explicit port", "https://github.com:444/x", ""},
		{"leading space", " https://github.com/x", ""},
		{"javascript scheme", "javascript:alert(1)", ""},
		{"empty", "", ""},
		{"gist subdomain", "https://gist.github.com/x", ""},
		{"github.com only in the path", "https://evil.com/github.com", ""},
		{"escape character", "https://github.com/x\x1b[31m", ""},
		{"space in the path", "https://github.com/a b", ""},
		{"newline in the path", "https://github.com/a\nb", ""},
		{"user info with a password", "https://user:pw@github.com/x", ""},
		{"file scheme", "file:///etc/passwd", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := safeURL(tt.raw)
			if tt.want == "" {
				if ok || got != "" {
					t.Errorf("safeURL(%q) = %q, %v, want a refusal", tt.raw, got, ok)
				}
				return
			}
			if !ok || got != tt.want {
				t.Errorf("safeURL(%q) = %q, %v, want %q, true", tt.raw, got, ok, tt.want)
			}
		})
	}
}

func TestOpenKey(t *testing.T) {
	o := &fakeOpener{}
	m := openModel(prURL, o)
	m, cmd := m.Update(key("o"))
	if cmd == nil {
		t.Fatal("o returned no command")
	}
	if len(o.urls) != 0 {
		t.Fatalf("the opener ran inside Update: %v", o.urls)
	}
	msg := cmd()
	if len(o.urls) != 1 || o.urls[0] != prURL {
		t.Errorf("opener calls = %v, want one call with %q", o.urls, prURL)
	}
	m, _ = m.Update(msg)
	if got := lastLine(m); strings.Contains(got, "not opened") || strings.Contains(got, "could not") {
		t.Errorf("a successful open left a message in the footer: %q", got)
	}
}

func TestOpenRefusesForeignURL(t *testing.T) {
	for _, url := range []string{"https://evil.example/x", "https://github.com.evil.com/x", "", "http://github.com/x"} {
		o := &fakeOpener{}
		m := openModel(url, o)
		m, cmd := m.Update(key("o"))
		if cmd == nil {
			t.Errorf("%q: no command returned, want the flash timer", url)
		} else {
			cmd() // the timer; it must not reach the opener
		}
		if len(o.urls) != 0 {
			t.Errorf("%q: the opener was called with %v", url, o.urls)
		}
		if got := lastLine(m); !strings.Contains(got, "not a github.com link — not opened") {
			t.Errorf("%q: footer = %q, want the refusal message", url, got)
		}
	}
}

func TestOpenFailureFlash(t *testing.T) {
	o := &fakeOpener{err: errors.New("exec: \"xdg-open\": executable file not found")}
	m := openModel(prURL, o)
	m, cmd := m.Update(key("o"))
	m, _ = m.Update(cmd())
	got := lastLine(m)
	if !strings.Contains(got, "could not open the browser") {
		t.Errorf("footer = %q, want the failure message", got)
	}
	if strings.Contains(got, "xdg-open") {
		t.Errorf("footer shows the launcher error text: %q", got)
	}
}

func TestOpenPanicRecovered(t *testing.T) {
	o := &fakeOpener{panicVal: "secret-panic-value"}
	m := openModel(prURL, o)
	m, cmd := m.Update(key("o"))
	if cmd == nil {
		t.Fatal("o returned no command")
	}
	msg := cmd() // must not panic
	om, ok := msg.(openedMsg)
	if !ok || om.err == nil {
		t.Fatalf("cmd() = %#v, want an openedMsg with an error", msg)
	}
	if strings.Contains(om.err.Error(), "secret-panic-value") {
		t.Errorf("the error carries the panic value: %v", om.err)
	}
	m, _ = m.Update(msg)
	got := plain(m)
	if !strings.Contains(lastLine(m), "could not open the browser") {
		t.Errorf("footer = %q, want the failure message", lastLine(m))
	}
	if strings.Contains(got, "secret-panic-value") {
		t.Errorf("the screen shows the panic value:\n%s", got)
	}
}

func TestOpenNilOpener(t *testing.T) {
	m := openModel(prURL, nil)
	m, _ = m.Update(key("o"))
	if got := lastLine(m); !strings.Contains(got, "no browser available") {
		t.Errorf("footer = %q, want %q", got, "no browser available")
	}
}

func TestOpenIgnoredOutsideList(t *testing.T) {
	t.Run("no rows", func(t *testing.T) {
		o := &fakeOpener{}
		var m tea.Model = filterModelOpts(120, 40, domain.Snapshot{Viewer: "octo"}, Options{ClosedDays: 14, Opener: o})
		if _, cmd := m.Update(key("o")); cmd != nil {
			t.Error("o on the empty screen returned a command")
		}
		if len(o.urls) != 0 {
			t.Errorf("opener calls = %v", o.urls)
		}
	})
	t.Run("filtered empty", func(t *testing.T) {
		o := &fakeOpener{}
		m := press(openModel(prURL, o), "1", "2", "3")
		if _, cmd := m.Update(key("o")); cmd != nil {
			t.Error("o with every row filtered out returned a command")
		}
	})
	t.Run("number prompt", func(t *testing.T) {
		o := &fakeOpener{}
		m := press(openModel(prURL, o), "/", "1")
		m, cmd := m.Update(key("o"))
		if cmd != nil {
			t.Error("o in the prompt returned a command")
		}
		mm := m.(Model)
		if !mm.prompting || mm.numFilter != "1" {
			t.Errorf("prompt changed: prompting = %v, filter = %q, want true and %q", mm.prompting, mm.numFilter, "1")
		}
		if len(o.urls) != 0 {
			t.Errorf("opener calls = %v", o.urls)
		}
	})
	t.Run("help overlay", func(t *testing.T) {
		o := &fakeOpener{}
		m := press(openModel(prURL, o), "?")
		m, cmd := m.Update(key("o"))
		if cmd != nil {
			t.Error("o in the overlay returned a command")
		}
		if m.(Model).showHelp {
			t.Error("o did not close the overlay")
		}
		if len(o.urls) != 0 {
			t.Errorf("opener calls = %v", o.urls)
		}
	})
}

func TestOpenCanceledIgnored(t *testing.T) {
	m := openModel(prURL, &fakeOpener{})
	before := plain(m)
	m, cmd := m.Update(openedMsg{err: context.Canceled})
	if cmd != nil {
		t.Error("a canceled open returned a command")
	}
	if after := plain(m); after != before {
		t.Errorf("a canceled open changed the screen:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if m.(Model).flash != "" {
		t.Errorf("flash = %q, want none", m.(Model).flash)
	}
}

func TestOpenHint(t *testing.T) {
	if got := lastLine(filterModel(120, 40, promptSnap())); !strings.Contains(got, "o open") {
		t.Errorf("footer = %q, want the 'o open' hint", got)
	}
}

package ui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/sbartram/prv/internal/domain"
)

const testConfigPath = "~/.config/prv/config.toml"

// helpModel builds a sized model that has loaded snap. Unset options get the
// values the help tests share: 14 closed days and the fixed clock.
func helpModel(t *testing.T, w, h int, snap domain.Snapshot, opts Options) Model {
	t.Helper()
	if opts.ClosedDays == 0 {
		opts.ClosedDays = 14
	}
	if opts.Now == nil {
		opts.Now = fixedNow
	}
	var m tea.Model = New(t.Context(), &fakeFetcher{}, opts)
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m, _ = m.Update(fetchedMsg{snap: snap})
	return m.(Model)
}

func TestHelpOverlay(t *testing.T) {
	const marker = "press any key to return"
	open := func(t *testing.T) tea.Model {
		t.Helper()
		var m tea.Model = helpModel(t, 120, 40, fixtureThirty(), Options{})
		m, cmd := m.Update(key("?"))
		if cmd != nil {
			t.Error("opening the overlay returned a command")
		}
		if !strings.Contains(plain(m), marker) {
			t.Fatalf("? did not open the overlay:\n%s", plain(m))
		}
		return m
	}

	t.Run("any key closes it", func(t *testing.T) {
		for name, k := range map[string]tea.KeyPressMsg{
			"x": key("x"), "esc": special(tea.KeyEscape), "enter": special(tea.KeyEnter), "?": key("?"),
		} {
			m, cmd := open(t).Update(k)
			if cmd != nil {
				t.Errorf("%s: closing the overlay returned a command", name)
			}
			if strings.Contains(plain(m), marker) {
				t.Errorf("%s: the overlay is still shown", name)
			}
			if !strings.Contains(plain(m), "acme/api#1") {
				t.Errorf("%s: the list is not back:\n%s", name, plain(m))
			}
		}
	})
	t.Run("q closes it and does not quit", func(t *testing.T) {
		m, cmd := open(t).Update(key("q"))
		if cmd != nil {
			t.Error("q in the overlay returned a command, want none (no quit)")
		}
		if strings.Contains(plain(m), marker) {
			t.Error("q left the overlay open")
		}
	})
	t.Run("ctrl+c quits", func(t *testing.T) {
		_, cmd := open(t).Update(key("ctrl+c"))
		if cmd == nil {
			t.Fatal("no command returned")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Error("ctrl+c did not produce tea.QuitMsg")
		}
	})
	t.Run("the key is consumed", func(t *testing.T) {
		// "2" would toggle merged PRs; with the overlay open it only closes it.
		m := open(t)
		before := m.(Model).show
		m, _ = m.Update(key("2"))
		if m.(Model).show != before {
			t.Error("a key pressed in the overlay acted on the list")
		}
	})
	t.Run("works in every state", func(t *testing.T) {
		for _, state := range []string{"loading", "error", "empty"} {
			m, _ := stateModel(t, state, 120, 40).Update(key("?"))
			if !strings.Contains(plain(m), marker) {
				t.Errorf("%s: ? did not open the overlay", state)
			}
		}
	})
}

func TestHelpContent(t *testing.T) {
	m := helpModel(t, 120, 40, fixtureThirty(), Options{ConfigPath: testConfigPath})
	v := plain(press(m, "?"))
	for _, want := range []string{
		"press any key to return",
		"✓ approved", "✗ changes", "○ required", "– none",
		"✓ passing", "✗ failing", "● running", "– no checks", "? unknown",
		"[draft]",
		"config ~/.config/prv/config.toml · every 60s · 14 days",
		"api 4812 / 5000, resets 15:02",
	} {
		if !strings.Contains(v, want) {
			t.Errorf("overlay missing %q:\n%s", want, v)
		}
	}

	t.Run("checks legend and the expand key", func(t *testing.T) {
		var legend string
		for _, l := range strings.Split(v, "\n") {
			if strings.HasPrefix(strings.TrimSpace(l), "checks") {
				legend = l
			}
		}
		for _, want := range []string{"✗ failing", "⊘ cancelled", "● running", "✓ passing", "– skipped"} {
			if !strings.Contains(legend, want) {
				t.Errorf("checks legend %q missing %q", legend, want)
			}
		}
		// The key table pads each key to its column, so match the pair.
		if !regexp.MustCompile(`\be +expand\b`).MatchString(v) {
			t.Errorf("overlay missing the 'e expand' binding:\n%s", v)
		}
	})
	t.Run("limit not known", func(t *testing.T) {
		snap := fixtureThirty()
		snap.RateLimit = domain.RateLimit{}
		v := plain(press(helpModel(t, 120, 40, snap, Options{}), "?"))
		if !strings.Contains(v, "api limit not known yet") {
			t.Errorf("overlay missing the unknown-limit line:\n%s", v)
		}
		if strings.Contains(v, "api 0 /") {
			t.Errorf("overlay shows a made-up budget:\n%s", v)
		}
	})
	t.Run("interval and window from options", func(t *testing.T) {
		m := helpModel(t, 120, 40, fixtureThirty(), Options{Interval: 2 * time.Minute, ClosedDays: 3, ConfigPath: "none"})
		if v := plain(press(m, "?")); !strings.Contains(v, "config none · every 2m · 3 days") {
			t.Errorf("overlay settings line wrong:\n%s", v)
		}
	})
}

func TestHelpFromKeymap(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {120, 40}} {
		m := helpModel(t, sz[0], sz[1], fixtureThirty(), Options{})
		v := plain(press(m, "?"))
		for _, col := range m.keys.FullHelp() {
			for _, b := range col {
				if !strings.Contains(v, b.Help().Key) || !strings.Contains(v, b.Help().Desc) {
					t.Errorf("%dx%d: overlay missing binding %q %q:\n%s", sz[0], sz[1], b.Help().Key, b.Help().Desc, v)
				}
			}
		}
	}
	// The footer hints draw from the same map: each short binding is a
	// binding of that map, not a separate string.
	k := defaultKeys()
	if len(k.ShortHelp()) == 0 || len(k.FullHelp()) == 0 {
		t.Fatal("keymap has no help")
	}
	if !k.Help.Enabled() {
		t.Error("the Help binding is not enabled")
	}
}

func TestFmtInterval(t *testing.T) {
	for _, tt := range []struct {
		d    time.Duration
		want string
	}{
		{60 * time.Second, "60s"},
		{90 * time.Second, "90s"},
		{2 * time.Minute, "2m"},
		{15 * time.Minute, "15m"},
		{15 * time.Second, "15s"},
		{150 * time.Second, "150s"},
		{1500 * time.Millisecond, "1.5s"},
	} {
		if got := fmtInterval(tt.d); got != tt.want {
			t.Errorf("fmtInterval(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestHelpWarningsAndCleanPath(t *testing.T) {
	t.Run("warnings stay listed after the footer window", func(t *testing.T) {
		clk := &testClock{}
		clk.set(12, 0, 0)
		warnings := []string{`config: unknown key "x"`, `config: unknown key "y"`}
		m := helpModel(t, 120, 40, fixtureThirty(), Options{Now: clk.now, Warnings: warnings, ConfigPath: testConfigPath})
		clk.set(12, 1, 0) // 60 s later: the 8 s footer window is over
		if footer := lastLine(m); strings.Contains(footer, "unknown key") {
			t.Fatalf("the footer still shows the warning after 8 s: %q", footer)
		}
		v := plain(press(m, "?"))
		for _, w := range warnings {
			if !strings.Contains(v, w) {
				t.Errorf("overlay missing warning %q:\n%s", w, v)
			}
		}
	})
	t.Run("escape sequences are removed", func(t *testing.T) {
		m := helpModel(t, 120, 40, fixtureThirty(), Options{
			ConfigPath: "~/.config/\x1b[31mprv/config.toml",
			Warnings:   []string{"config: bad \x1b]0;owned\x07value"},
		})
		v := ansi.Strip(press(m, "?").(Model).View().Content)
		if strings.ContainsRune(v, 0x1b) || strings.Contains(v, "[31m") || strings.Contains(v, "owned") {
			t.Errorf("overlay holds an escape sequence from the path or a warning: %q", v)
		}
		if !strings.Contains(v, "~/.config/prv/config.toml") {
			t.Errorf("overlay lost the cleaned path:\n%s", v)
		}
	})
}

func TestRefreshBehindHelp(t *testing.T) {
	f := &fakeFetcher{snap: onePR()}
	m, rec := recModel(f, 0, nil)
	m = runCmd(m, m.Init())
	if f.calls != 1 || len(rec.calls) != 1 {
		t.Fatalf("setup: fetches = %d, timers = %d, want 1 and 1", f.calls, len(rec.calls))
	}

	m = press(m, "?")
	fresh := onePR()
	fresh.PRs[0].Title = "Fresh title after refresh"
	f.snap = fresh

	m, cmd := m.Update(rec.calls[0].msg) // the tick fires behind the overlay
	if cmd == nil {
		t.Fatal("the tick started no fetch while the overlay was open")
	}
	m = runCmd(m, cmd)
	if f.calls != 2 {
		t.Errorf("fetches = %d, want 2", f.calls)
	}
	if v := plain(m); !strings.Contains(v, "press any key to return") {
		t.Errorf("the overlay closed when the refresh result arrived:\n%s", v)
	}
	if len(rec.calls) != 2 {
		t.Errorf("armed timers = %d, want 2 (the chain continues behind the overlay)", len(rec.calls))
	}

	m = press(m, "x")
	if v := plain(m); !strings.Contains(v, "Fresh title after refresh") {
		t.Errorf("the list does not show the new data:\n%s", v)
	}
}

func TestHelpFits(t *testing.T) {
	warnings := []string{
		`config: unknown key "alpha"`,
		`config: unknown key "beta"`,
		`config: unknown key "gamma" — this one is deliberately long so that the line has to wrap at 80 columns`,
	}
	for _, sz := range [][2]int{{80, 24}, {120, 40}} {
		for _, withWarnings := range []bool{false, true} {
			name := itoa(sz[0]) + "x" + itoa(sz[1])
			if withWarnings {
				name += "_warnings"
			}
			t.Run(name, func(t *testing.T) {
				opts := Options{ConfigPath: testConfigPath}
				if withWarnings {
					opts.Warnings = warnings
				}
				m := press(helpModel(t, sz[0], sz[1], fixtureThirty(), opts), "?")
				ls := lines(m.View().Content)
				if len(ls) > sz[1] {
					t.Errorf("%d lines, want at most %d", len(ls), sz[1])
				}
				for _, l := range ls {
					if lipgloss.Width(l) > sz[0] {
						t.Errorf("line is %d cells wide, want at most %d: %q", lipgloss.Width(l), sz[0], l)
					}
				}
				if last := ls[len(ls)-1]; last != "press any key to return" {
					t.Errorf("last line = %q, want the return hint", last)
				}
			})
		}
	}
	t.Run("many warnings are cut, the hint stays", func(t *testing.T) {
		var many []string
		for i := range 40 {
			many = append(many, "config: unknown key "+itoa(i))
		}
		m := press(helpModel(t, 80, 24, fixtureThirty(), Options{Warnings: many}), "?")
		ls := lines(m.View().Content)
		if len(ls) > 24 {
			t.Errorf("%d lines, want at most 24", len(ls))
		}
		if last := ls[len(ls)-1]; last != "press any key to return" {
			t.Errorf("last line = %q, want the return hint", last)
		}
	})
	t.Run("not sized does not panic", func(t *testing.T) {
		var m tea.Model = New(t.Context(), &fakeFetcher{}, Options{})
		m, _ = m.Update(key("?"))
		if !strings.Contains(plain(m), "press any key to return") {
			t.Error("unsized overlay is missing the hint")
		}
	})
}

func TestHelpAnsi16(t *testing.T) {
	m := press(helpModel(t, 80, 24, fixtureThirty(), Options{Warnings: []string{"config: unknown key x"}}), "?")
	raw := m.View().Content
	if rgbFg.MatchString(raw) || rgbBg.MatchString(raw) {
		t.Errorf("overlay holds a 24-bit colour sequence: %q", raw)
	}
	if strings.Contains(raw, "38;5;") || strings.Contains(raw, "48;5;") {
		t.Errorf("overlay holds a 256-colour sequence: %q", raw)
	}
	if !strings.Contains(raw, "\x1b[") {
		t.Error("overlay holds no styling at all")
	}
}

func TestHelpHint(t *testing.T) {
	if got := lastLine(filterModel(120, 40, promptSnap())); !strings.Contains(got, "? help") {
		t.Errorf("footer = %q, want the '? help' hint", got)
	}
}

func TestHelpGolden(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {120, 40}} {
		t.Run(itoa(sz[0])+"x"+itoa(sz[1]), func(t *testing.T) {
			m := press(helpModel(t, sz[0], sz[1], fixtureThirty(), Options{ConfigPath: testConfigPath}), "?")
			golden.RequireEqual(t, ansi.Strip(m.View().Content))
		})
	}
}

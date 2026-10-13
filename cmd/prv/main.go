// Command prv is a terminal UI for monitoring GitHub pull requests.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/browser"

	"github.com/sbartram/prv/internal/config"
	"github.com/sbartram/prv/internal/domain"
	"github.com/sbartram/prv/internal/github"
	"github.com/sbartram/prv/internal/ui"
	"github.com/sbartram/prv/internal/watchlist"
)

// version is overridable: -ldflags "-X main.version=1.2.3".
var version = "dev"

// tokenForHost is a variable so tests can replace it.
var tokenForHost = auth.TokenForHost

// configPath is a variable so tests can replace it.
var configPath = config.Path

// watchlistPath is a variable so tests can replace it.
var watchlistPath = watchlist.Path

const noTokenMessage = "prv needs a GitHub token. Run `gh auth login`, or set GH_TOKEN / GITHUB_TOKEN."

// browserOpener opens URLs through go-gh's browser, which honours GH_BROWSER,
// the gh config and BROWSER like `gh pr view --web` does. ui only sees the
// ui.Opener interface and checks every URL before it gets here.
type browserOpener struct{ b *browser.Browser }

var _ ui.Opener = browserOpener{}

// The github client supplies the detail view's data and the add prompt's check.
var (
	_ ui.DetailFetcher = (*github.Client)(nil)
	_ ui.Looker        = (*github.Client)(nil)
)

// The watch list store is what the client reads and the add prompt writes.
var (
	_ ui.Watchlist       = (*watchlist.Store)(nil)
	_ github.WatchSource = (*watchlist.Store)(nil)
)

func (o browserOpener) Open(url string) error { return o.b.Browse(url) }

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// buildVersion prefers the -X value, then the module version stamped by
// `go install`, then "dev".
func buildVersion() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return version
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("prv", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprintln(stderr, "usage: prv [--version]") }
	showVersion := fs.Bool("version", false, "print the version and exit")
	closedDays := fs.Int("closed-days", domain.DefaultClosedDays, "") // hidden: not in usage
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, "prv "+buildVersion())
		return 0
	}
	if *closedDays < config.MinClosedDays || *closedDays > config.MaxClosedDays {
		fmt.Fprintln(stderr, "prv: --closed-days must be between 1 and 365")
		return 2
	}

	// A broken config stops prv before the token lookup and the TUI. When the
	// path itself cannot be found there is no config to read: keep the defaults.
	// The file is read once, here; edits while prv runs need a restart.
	cfg := config.Default()
	shownPath := ""
	if path, err := configPath(); err == nil {
		cfg, err = config.Load(path)
		if err != nil {
			fmt.Fprintln(stderr, "prv: "+err.Error())
			return 2
		}
		shownPath = path
	}
	home, _ := os.UserHomeDir() // an error leaves home empty: no substitution
	days := effectiveClosedDays(fs, *closedDays, cfg)

	// The token goes only into github.Options.Token. It is never printed,
	// logged or written. Missing means exactly the empty string.
	token, _ := tokenForHost("github.com")
	if token == "" {
		fmt.Fprintln(stderr, noTokenMessage)
		return 1
	}
	ghOpts := github.Options{Token: token, ClosedDays: days}
	uiOpts := ui.Options{
		Opener:     browserOpener{b: browser.New("", io.Discard, io.Discard)}, // io.Discard keeps a launcher from writing over the TUI
		ClosedDays: days,
		Interval:   cfg.Interval,
		Show:       cfg.Show,
		Warnings:   cfg.Warnings,
		ConfigPath: displayPath(shownPath, home),
	}
	// The store is a nil interface when the path cannot be found: never assign a
	// nil *watchlist.Store to either field.
	if path, err := watchlistPath(); err == nil {
		s := watchlist.New(path)
		ghOpts.Watch = s
		uiOpts.Watchlist = s
	}
	client, err := github.NewClient(ghOpts)
	if err != nil {
		fmt.Fprintln(stderr, "prv: "+err.Error())
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := tea.NewProgram(ui.New(ctx, client, uiOpts), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(stderr, "prv: "+err.Error())
		return 1
	}
	return 0
}

// effectiveClosedDays is the closed window in days: the --closed-days flag
// when it was given, else closed_days from the config, else the default (D-16).
func effectiveClosedDays(fs *flag.FlagSet, flagValue int, cfg config.Config) int {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "closed-days" {
			set = true
		}
	})
	if set {
		return flagValue
	}
	return cfg.ClosedDays
}

// displayPath shortens a path under home to "~/...". An empty path is "none".
func displayPath(path, home string) string {
	if path == "" {
		return "none"
	}
	if home != "" && strings.HasPrefix(path, home+"/") {
		return "~" + path[len(home):]
	}
	return path
}

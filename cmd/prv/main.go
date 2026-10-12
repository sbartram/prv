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

	"github.com/sbartram/prv/internal/config"
	"github.com/sbartram/prv/internal/domain"
	"github.com/sbartram/prv/internal/github"
	"github.com/sbartram/prv/internal/ui"
)

// version is overridable: -ldflags "-X main.version=1.2.3".
var version = "dev"

// tokenForHost is a variable so tests can replace it.
var tokenForHost = auth.TokenForHost

// configPath is a variable so tests can replace it.
var configPath = config.Path

const noTokenMessage = "prv needs a GitHub token. Run `gh auth login`, or set GH_TOKEN / GITHUB_TOKEN."

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
	client, err := github.NewClient(github.Options{Token: token, ClosedDays: days})
	if err != nil {
		fmt.Fprintln(stderr, "prv: "+err.Error())
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := tea.NewProgram(ui.New(ctx, client, ui.Options{
		ClosedDays: days,
		Interval:   cfg.Interval,
		Show:       cfg.Show,
		Warnings:   cfg.Warnings,
		ConfigPath: displayPath(shownPath, home),
	}), tea.WithContext(ctx))
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

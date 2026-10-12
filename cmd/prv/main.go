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

	tea "charm.land/bubbletea/v2"
	"github.com/cli/go-gh/v2/pkg/auth"

	"github.com/sbartram/prv/internal/domain"
	"github.com/sbartram/prv/internal/github"
	"github.com/sbartram/prv/internal/ui"
)

// version is overridable: -ldflags "-X main.version=1.2.3".
var version = "dev"

// tokenForHost is a variable so tests can replace it.
var tokenForHost = auth.TokenForHost

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
	if *closedDays < 1 {
		fmt.Fprintln(stderr, "prv: --closed-days must be at least 1")
		return 2
	}

	// The token goes only into github.Options.Token. It is never printed,
	// logged or written. Missing means exactly the empty string.
	token, _ := tokenForHost("github.com")
	if token == "" {
		fmt.Fprintln(stderr, noTokenMessage)
		return 1
	}
	client, err := github.NewClient(github.Options{Token: token, ClosedDays: *closedDays})
	if err != nil {
		fmt.Fprintln(stderr, "prv: "+err.Error())
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := tea.NewProgram(ui.New(ctx, client, ui.Options{ClosedDays: *closedDays}), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(stderr, "prv: "+err.Error())
		return 1
	}
	return 0
}

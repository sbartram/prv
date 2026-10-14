package ui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestShowRateWarning prints prv's real list screen with a low API budget (999
// of 5000, below 20% of the hourly limit) at 80 and 40 columns, ANSI kept, so a
// person can judge the warning in their own terminal and multiplexer pane
// (05-UAT test 2). A live low budget would need about 4,000 points spent, so
// this renders the real model with fixture data instead.
//
// Run it with:
//
//	PRV_SHOW=1 go test ./internal/ui -run '^TestShowRateWarning$' -count=1 -v
//
// -v streams the output; -count=1 skips the test cache. The test is offline and
// uses fixture data only. Without PRV_SHOW=1 it skips and prints nothing.
func TestShowRateWarning(t *testing.T) {
	if os.Getenv("PRV_SHOW") != "1" {
		t.Skip("set PRV_SHOW=1 to print the low-budget rate warning as prv renders it (05-UAT test 2)")
	}
	for _, w := range []int{80, 40} {
		snap := fixtureSix()
		snap.RateLimit = withRate(5000, 999).RateLimit // 999*5 < 5000: below 20%
		content := newTestModel(t, snap, w, 12).View().Content
		ls := strings.Split(content, "\n")

		if !strings.Contains(ls[0], "\x1b[33mapi 999/5000") || !strings.Contains(ansi.Strip(ls[0]), "api 999/5000 · resets 15:02") {
			t.Errorf("%d columns: header %q lacks the yellow warning", w, ls[0])
		}
		if last := ls[len(ls)-1]; !strings.Contains(last, "\x1b[2mapi 999") {
			t.Errorf("%d columns: last line %q lacks the faint api segment", w, last)
		}

		fmt.Fprintf(os.Stdout, "prv at %d columns, API budget 999 of 5000 (below 20%%):\n", w)
		fmt.Fprintln(os.Stdout, content)
		fmt.Fprintln(os.Stdout)
	}
}

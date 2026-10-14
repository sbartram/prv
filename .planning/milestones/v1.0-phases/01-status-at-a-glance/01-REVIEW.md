---
phase: 01-status-at-a-glance
reviewed: 2026-10-11T00:00:00Z
depth: standard
files_reviewed: 20
files_reviewed_list:
  - cmd/prv/main.go
  - cmd/prv/main_test.go
  - go.mod
  - internal/domain/clean.go
  - internal/domain/domain_test.go
  - internal/domain/list.go
  - internal/domain/pr.go
  - internal/domain/status.go
  - internal/github/client.go
  - internal/github/client_test.go
  - internal/github/errors.go
  - internal/github/errors_test.go
  - internal/ui/layout.go
  - internal/ui/model.go
  - internal/ui/model_test.go
  - internal/ui/styles.go
  - internal/ui/view.go
  - internal/ui/view_test.go
  - README.md
  - .gitignore
findings:
  critical: 0
  warning: 3
  info: 4
  total: 7
status: issues_found
---

# Phase 1: Code Review Report

**Reviewed:** 2026-10-11
**Depth:** standard
**Files Reviewed:** 20
**Status:** issues_found

## Summary

The code builds, `go vet` is clean and all tests pass. The token handling is sound: it is passed only through `Options.Token`, never printed, and `LogIgnoreEnv` is set. Search input goes through a GraphQL variable. Remote strings are cleaned before display, and the panic path in `fetchCmd` is safe. I found no security vulnerabilities and no crashes.

The defects are correctness gaps in how partial or degraded results are shown to the user. The tool's core value is an at-a-glance view of the user's PRs, so a degraded list must not look like a complete one. There is also one unbounded CLI input that corrupts the closed-window logic. The first defect below was confirmed with a throwaway test, which I deleted.

## Warnings

### WR-01: `--closed-days` has no upper bound; large values overflow and silently drop closed PRs

**File:** `cmd/prv/main.go:57`, `internal/domain/list.go:19`, `internal/github/client.go:86`
**Issue:** `main` only rejects values below 1. `InWindow` computes `time.Duration(closedDays) * 24 * time.Hour`, which overflows `int64` nanoseconds above about 106,751 days. `BuildSearch` calls `AddDate` with the same value, which produces nonsense dates. I confirmed both effects:
- With `--closed-days 200000` the search contains `closed:>=1479-03-13`, and `InWindow` returns `false` for a PR that closed one hour ago. Every merged and closed PR is dropped.
- With `--closed-days 1000000` the search string contains `closed:>=-0712-11-13`, which is invalid.
- With `1<<40` the overflow flips the other way and `InWindow` returns `true` for everything.

The flag is hidden, but it is still user-reachable, and the result is a wrong list with no warning.
**Fix:** Cap the value in `run`, for example:
```go
const maxClosedDays = 3650
if *closedDays < 1 || *closedDays > maxClosedDays {
    fmt.Fprintf(stderr, "prv: --closed-days must be between 1 and %d\n", maxClosedDays)
    return 2
}
```
Add a boundary test for the upper limit.

### WR-02: Empty-state screen claims "No open PRs" even when the fetch was partial or truncated

**File:** `internal/ui/view.go:28-31`, `internal/ui/model.go:137-140`, `internal/github/client.go:159-171`
**Issue:** `Fetch` deliberately returns a snapshot with `Warnings` set when page 1 comes back with a GraphQL error plus partial data. That includes the case where `viewer.login` is present and `search.nodes` is empty, for example when SAML enforcement blocks the results. `Update` then sees `len(PRs) == 0` and enters `stateEmpty`. The empty screen prints "No open PRs authored by or assigned to @x, and none closed in the last N days." and discards `Warnings` and `Truncated`. The user is told with confidence that nothing is wrong when the load actually failed. The list screen at least shows "some data could not be loaded"; the empty screen shows nothing.
**Fix:** In `render()`, for `stateEmpty` append a notice when `len(m.snap.Warnings) > 0` or `m.snap.Truncated`, and change the wording to "No PRs could be listed". Alternatively, in `Fetch`, treat "warnings and zero PRs" as a `KindGraphQL` error so the error panel with its retry hint is shown. Add a test for this combination.

### WR-03: Footer puts the "incomplete list" indicators last, so `cut` drops them first on narrow panes

**File:** `internal/ui/view.go:141-147`, `internal/ui/view.go:152-154`
**Issue:** The footer is built as `"j/k move · q quit"` followed by `" · list truncated"` and `" · some data could not be loaded"`. `cut` truncates from the right. At 80 columns the full footer is 70 cells, so it fits. In a narrow multiplexer pane of 60 columns or fewer, which the project explicitly targets, the safety indicators are the first thing removed, and the list looks complete. This repeats the WR-02 problem on the list screen.
**Fix:** Put the status indicators first and the key hints last, for example `"list truncated · some data could not be loaded · j/k move · q quit"`. Alternatively, render the indicators on the header line. Add a view test at width 40 with `Truncated` set.

## Info

### IN-01: Dead data: `wireNode.MergedAt` and the text of `Snapshot.Warnings`

**File:** `internal/github/client.go:119`, `internal/github/client.go:164-170`, `internal/ui/view.go:145-147`
**Issue:** `MergedAt` is requested and decoded but never read, because `domain.PR` has no such field. The cleaned GraphQL error message added to `Warnings` is also never shown. The UI only tests `len(Warnings) > 0` and prints a fixed string, so the cleaning work and the detailed text are wasted.
**Fix:** Drop `mergedAt` from the query and the wire struct if it is not needed yet. Show the first warning in the footer, or reduce `Warnings` to a bool.

### IN-02: Extra positional arguments are silently ignored

**File:** `cmd/prv/main.go:46-52`
**Issue:** A mistyped flag is rejected, but stray positional arguments such as `prv https://github.com/o/r/pull/1` start the TUI as if nothing was passed. Users who expect URL arguments to work get no feedback.
**Fix:** After `fs.Parse`, check `if fs.NArg() > 0` and print the usage with exit code 2.

### IN-03: `Clean` leaves implicit bidi marks and Unicode line/paragraph separators

**File:** `internal/domain/clean.go:20-22`
**Issue:** The function strips the embedding, override and isolate characters. It keeps U+200E and U+200F (LRM and RLM), U+061C (ALM), and U+2028 and U+2029, which are not `unicode.IsControl`. Implicit marks can still reorder adjacent text in a PR title or author name. The doc comment limits the scope to overrides and isolates, so this is deliberate, but the risk remains for a display-spoofing hardening function.
**Fix:** Add `r == 0x200E || r == 0x200F || r == 0x061C || r == 0x2028 || r == 0x2029` to the removal case, and add test rows.

### IN-04: `classify` reports every `net.Error` as "could not reach api.github.com"

**File:** `internal/github/errors.go:98-100`
**Issue:** `*url.Error` implements `net.Error`, so TLS certificate failures, proxy errors and similar problems are also reported as a connectivity failure. The hint then says "Check your connection", which misleads someone behind a TLS-intercepting proxy. A zero-value `Error` is also `KindAuth` because the enum starts at 0, so a forgotten `Kind` produces an auth message.
**Fix:** Optionally detect `x509.UnknownAuthorityError` and similar types and give a specific message. Start the enum at `KindUnknown = iota` so the zero value is the neutral case.

---

_Reviewed: 2026-10-11_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

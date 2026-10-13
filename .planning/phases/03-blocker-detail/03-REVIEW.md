---
phase: 03-blocker-detail
reviewed: 2026-10-12T00:00:00Z
depth: standard
files_reviewed: 21
files_reviewed_list:
  - cmd/prv/main_test.go
  - internal/domain/backoff.go
  - internal/domain/backoff_test.go
  - internal/domain/checks.go
  - internal/domain/checks_test.go
  - internal/github/checks.go
  - internal/github/detail_test.go
  - internal/ui/detail.go
  - internal/ui/detail_test.go
  - internal/ui/testdata/TestGolden/detail-error_0x0.golden
  - internal/ui/testdata/TestGolden/detail-error_120x40.golden
  - internal/ui/testdata/TestGolden/detail-error_60x15.golden
  - internal/ui/testdata/TestGolden/detail-error_80x24.golden
  - internal/ui/testdata/TestGolden/detail-gone_0x0.golden
  - internal/ui/testdata/TestGolden/detail-gone_120x40.golden
  - internal/ui/testdata/TestGolden/detail-gone_60x15.golden
  - internal/ui/testdata/TestGolden/detail-gone_80x24.golden
  - internal/ui/testdata/TestGolden/detail_0x0.golden
  - internal/ui/testdata/TestGolden/detail_120x40.golden
  - internal/ui/testdata/TestGolden/detail_60x15.golden
  - internal/ui/testdata/TestGolden/detail_80x24.golden
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
status: issues_found
---

# Phase 3: Code Review Report (re-review after gap plans 03-09 and 03-10)

**Reviewed:** 2026-10-12
**Depth:** standard (incremental: `git diff 2a7ca32`, plus the surrounding code the diff touches)
**Files Reviewed:** 21
**Status:** issues_found

## Summary

`go vet ./...` and `go test ./...` pass on the current tree.

Resolution of the prior findings:

| Prior finding | Result | Basis |
|---------------|--------|-------|
| WR-01 detail rate-limit pause bypasses the clamp | **Resolved** | `pauseForRateLimit` (`internal/ui/detail.go:146-163`) now computes `now + domain.ServerWait(...)`. `ServerWait` (`internal/domain/backoff.go:93`) clamps Retry-After to `[MinServerWait, MaxServerWait]` and sends a reset time through `resetWait`, which cuts the span before adding the 1 s margin, so a year-9999 reset cannot wrap. A rate limit with no timing now gets `RateLimitFloor`, so there is no "no pause" path. `TestServerWait` covers hostile and saturating inputs. |
| WR-02 `FetchDetail` undercounts on partial data | **Resolved in data, incomplete in display (see WR-05)** | `detailPage` returns a `partial` bool when it keeps data next to a GraphQL error. `FetchDetail` sets `Truncated` on it, and also when `Read < Total`. `TestFetchDetailPartial` covers three cases (null node, page-1 error, page-2 error), and `TestDetailPartialEndToEnd` covers the real client path. The flag is not shown correctly (WR-05). |
| WR-04 unknown checks folded as "passing" and counted as "skipped" | **Resolved** | `CheckCounts.Other` was added and `CountChecks` no longer sends `CheckOther` to `Neutral`. `checkLines` always lists unknown checks. The fold line now reads "more passing or skipped". I found no other consumer of `CheckCounts.Neutral`. |

Carried forward, not re-derived: WR-03 (deferred by the user), and IN-01 to IN-05 (open, see `03-REVIEW-DISPOSITION.md`).

## Warnings

### WR-05: A partial detail read is shown as "read N of N runs", or hides "no checks"

**File:** `internal/ui/detail.go:308-310` (cause: `internal/github/checks.go:331-335`)
**Issue:** The WR-02 fix reuses `Detail.Truncated` for two different things. One is "fewer check runs were read than exist". The other is "GitHub returned an error next to the data, so a check, reviewer or review request may be missing". The only place the flag reaches the screen is the text `read <Read> of <Total> runs`. This gives wrong output in two cases:

1. The error is about a reviewer or review request (the exact case in `TestFetchDetailPartial/"page 1 kept a GraphQL error"`, with path `reviewRequests.nodes.0.requestedReviewer`). All checks were read, so `Read == Total` and the user sees "read 3 of 3 runs". That text says the read was complete, but the reviewer list may be missing people. The user gets no clue that the reviewers are the incomplete part. The test checks only the struct fields (`Truncated == true`), not what is rendered.
2. The PR has no `statusCheckRollup` and a partial error arrives on page 1. `FetchDetail` takes the `r == nil` break with `Truncated == true` and `Read == Total == 0`. In `checksSummary`, `segs` then holds one segment, "read 0 of 0 runs". The `len(segs) == 0` branch that returns "no checks" is skipped, so the line reads `CHECKS  read 0 of 0 runs`.

The user is told "partial" in words that say "complete", which is the failure WR-02 meant to remove.
**Fix:** Add a separate signal so each cause gets its own text. Show "read N of M runs" only when `Read < Total` or paging was cut. For an error that was kept, add a distinct faint segment, for example "some details could not be loaded". Show it under the CHECKS line or the reviewers section. Example for `checksSummary`:
```go
// domain.Detail: add `Partial bool` (a GraphQL error was kept next to the data).
// github/checks.go: on `partial`, set d.Partial = true instead of d.Truncated = true.
// ui/detail.go:
if d.data.Truncated && d.data.Read < d.data.Total { // or paging was cut
    segs = append(segs, faintStyle.Render("read "+strconv.Itoa(d.data.Read)+" of "+strconv.Itoa(d.data.Total)+" runs"))
}
if d.data.Partial {
    segs = append(segs, faintStyle.Render("some details could not be loaded"))
}
```
Add a UI test that renders a `Detail{Partial: true, Read: 3, Total: 3}` and a `Detail{Partial: true}` with no checks, and asserts the text.

## Info

### IN-06: `ServerWait` and `NextDelay` agree only by coincidence when the error has no timing, and `ServerWait` ignores the known rate-limit reset

**File:** `internal/domain/backoff.go:93-101` (compare `:57-66`), test `internal/domain/backoff_test.go:140-145`
**Issue:** For a rate-limited error with no Retry-After and no reset, `NextDelay` waits `max(RateLimitFloor, failureWait(interval, failures))`. It also falls back to the snapshot's `rl.ResetAt` when the error has none. `ServerWait` returns a fixed `RateLimitFloor` and has no `rl` fallback. The doc comment says it "uses the order and bounds of NextDelay", but that is not fully true. `TestServerWait` checks `got == NextDelay(...)` only with `interval = 1 min`, `failures = 1` and `RateLimit{}`, where `failureWait` happens to equal `RateLimitFloor`. A different interval or a known `rl.ResetAt` would make the two paths disagree, and the test would not show it. In practice a detail request that is rate limited without timing pauses for 1 minute even when the list snapshot says the limit resets in 40 minutes. The next detail fetch then probably hits the limit again.
**Fix:** Either pass the snapshot's `ResetAt` into the detail path (`ServerWait(now, after, firstNonZero(reset, m.snap.RateLimit.ResetAt))`), or soften the doc comment and the test comment so they do not claim parity. Vary the interval and failures in the equivalence test so it does not rely on 1 min.

### IN-07: The help legend does not name the new "unknown" check class

**File:** `internal/ui/help.go:58-60` (summary line: `internal/ui/detail.go:307`)
**Issue:** The CHECKS summary can now print "N unknown" in its own style (WR-04). The legend lists only failing, cancelled, running, passing and skipped. The `?` help therefore never explains the new word or its glyph. This is the same gap as the "commented and dismissed glyphs" in the open IN-05, with one more entry.
**Fix:** Add `{domain.CheckOther, "unknown"}` to the `checks` slice in `help.go`. This changes the help golden files, so regenerate them.

---

_Reviewed: 2026-10-12_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

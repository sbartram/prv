---
phase: 03-blocker-detail
reviewed: 2026-10-13T00:00:00Z
depth: standard
files_reviewed: 11
files_reviewed_list:
  - cmd/prv/main_test.go
  - internal/domain/checks.go
  - internal/github/checks.go
  - internal/github/detail_test.go
  - internal/ui/detail.go
  - internal/ui/detail_test.go
  - internal/ui/testdata/TestGolden/detail-partial_0x0.golden
  - internal/ui/testdata/TestGolden/detail-partial_120x40.golden
  - internal/ui/testdata/TestGolden/detail-partial_60x15.golden
  - internal/ui/testdata/TestGolden/detail-partial_80x24.golden
  - internal/ui/view_test.go
findings:
  critical: 0
  warning: 1
  info: 9
  total: 10
status: issues_found
---

# Phase 3: Code Review Report (re-review after gap plan 03-11)

**Reviewed:** 2026-10-13
**Depth:** standard (incremental: `git diff 156a819..HEAD` over the 11 files, plus the code around the diff)
**Files Reviewed:** 11
**Status:** issues_found

## Summary

`go vet ./...` and `go test ./...` pass on the current tree.

**WR-05 is resolved.** `Detail.Truncated` and the new `Detail.Partial` now carry separate meanings:

- `FetchDetail` (`internal/github/checks.go:331-338`) sets `Partial` when a page kept a GraphQL error. It no longer sets `Truncated` for that case.
- `Truncated` is set only when runs were not read: a later page failed (`:330`), the page cap or an empty cursor stopped paging (`:359`), or `Read < Total` (`:367`).
- `checksSummary` (`internal/ui/detail.go:313`) shows "read N of M runs" only for `Truncated`.
- `detailNotes` (`:274`) shows the faint line "some details could not be loaded" for `Partial`. That line sits above the blank line and the CHECKS section.

I traced both original WR-05 cases:

1. Error on a reviewer or review request, every run read. `Truncated=false`, `Partial=true`. The view shows the notice and no run count.
2. No `statusCheckRollup` and a kept error. `Truncated=false`, `Partial=true`, `Read=Total=0`. The summary has no segments and returns "no checks", with the notice above it.

The only readers of `Detail.Truncated` are `checksSummary` (`internal/ui/detail.go:313`) and the writes in `FetchDetail`. `snap.Truncated` in `view.go` and `client.go` is a different field on `Snapshot` and is not affected. No reader misses the `Partial` case.

The note line goes through `detailLines`, which cuts every line to the width (`ansi.Truncate`). Its 32 cells fit the 60 and 36 cell cases that are tested. It is drawn from `m.detail.data`, so it clears when a later fetch succeeds with no kept error, and stays while older data is shown after a failed refresh (this is consistent with the D-11 note). The tests are real: `TestFetchDetailPartial` asserts both flags per case, `TestDetailPartialNotice` renders all four Truncated/Partial combinations plus the compact pane and the notice-above-CHECKS order, and `TestDetailKeptErrorEndToEnd` goes through the real client and model. I found no defect in the diff. Two small new observations follow (IN-08, IN-09).

Carried forward unchanged from the earlier reviews (not re-derived; line numbers are from the review that raised them and may have moved): WR-03 (deferred by the user) and IN-01 to IN-07. Prior WR-01, WR-02 and WR-04 stay resolved.

## Resolved

| Finding | Result | Basis |
|---------|--------|-------|
| WR-05 partial detail read shown as "read N of N runs" or hiding "no checks" | **Resolved** | `Partial` split from `Truncated`; see Summary. Pinned by `TestFetchDetailPartial` (4 subtests), `TestDetailPartialNotice`, `TestDetailKeptErrorEndToEnd` and the `detail-partial` goldens. |

## Warnings

### WR-03: Shared "latest response headers" is now racy, and detail requests are invisible to the budget

**Status:** deferred by the user (UAT 2026-10-12). Still present.

**File:** `internal/github/ratelimit.go:25-47`, `internal/github/checks.go:284`, `internal/github/client.go:218`
**Issue:** `captureTransport` keeps one `last rateHeaders` for the whole client, and `classify(err, c.capture.latest())` reads it after the request returns. This was safe while exactly one request was ever in flight; the `Fetch` doc comment still claims "one single-flight fetch". This phase added `FetchDetail`, which the UI starts as its own Cmd and which routinely overlaps `Fetch` (open detail during a refresh; `refreshDetail` runs while the failing-check query could still be running). When both are in flight, an HTTP-200 GraphQL error from one request is classified with the headers of the other. Consequences:
- A real rate limit with no `RATE_LIMITED` type can be treated as a generic GraphQL error, so no pause happens.
- Or `remaining: "0"` from the other response turns an ordinary GraphQL error into a rate-limited one, so the app pauses wrongly.

Separately, `wireDetailResp.RateLimit` is requested in the query but never read. The detail requests (up to 5 pages every refresh while the view is open) are never added to `m.snap.RateLimit`, so the footer "api N" and the `NextDelay` reserve check undercount real usage. The project constraint is that auto-refresh stays inside the rate limits.
**Fix:** Capture the headers per request, not per client, for example by wrapping the context:
```go
type ctxKey struct{}
func (t *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
    resp, err := t.base.RoundTrip(r)
    if slot, ok := r.Context().Value(ctxKey{}).(*rateHeaders); ok && resp != nil {
        *slot = rateHeaders{ /* the three headers */ }
    }
    return resp, err
}
```
Each caller creates a `*rateHeaders`, passes it in the context, and hands it to `classify`. For the budget, have `FetchDetail` return the response's `RateLimit` (or fold it into a client-level atomic) so the UI can update `snap.RateLimit.Remaining`.

## Info

### IN-01: Cached viewport lines go stale for time-dependent summary text

**File:** `internal/ui/detail.go:176-186,283-296`
**Issue:** The body is rendered into the viewport only in `syncDetail` (open, size, refreshDetail, applyDetail, expand). `checksSummary` reads `m.now()` and `m.pausedUntil`. A pause started by a failed list refresh or a rate limit does not trigger a sync. A detail still loading therefore says "loading…" instead of "paused until HH:MM". An old "paused until" text stays on screen after the pause ends until the next successful refresh.
**Fix:** Call `m.syncDetail()` (when `m.detail.open`) in the `fetchedMsg` error branch and in the `redrawMsg` handler, or compute the summary line at render time instead of caching it.

### IN-02: A rate-limited detail result that arrives after esc is dropped without pausing

**File:** `internal/ui/detail.go:126-132`
**Issue:** `applyDetail` returns early when the view is closed or the sequence is stale, before `pauseForRateLimit` runs. A rate-limit error from a fetch the user already left is thrown away. The next list refresh then runs into the limit again.
**Fix:** Call `m.pauseForRateLimit(msg.err)` before the `seq`/`ref` check when `msg.err` is non-nil. Only the `detailInFlight` and `detail.err` updates need the seq guard.

### IN-03: "loading…" can show forever for a PR that left the list before its first fetch

**File:** `internal/ui/detail.go:54-57,283-296`
**Issue:** `startDetailFetch` returns nil while `detail.gone`. If the user opens a PR during a rate-limit pause (no fetch starts) and the PR is absent from the next refresh, `data` and `err` both stay nil and the CHECKS line says "loading…" with no further attempt. The "no longer in list" note is the only hint.
**Fix:** In `checksSummary`, when `d.gone && d.data == nil`, return a fixed text such as "not loaded (no longer in list)".

### IN-04: Detail query re-requests reviewers on every page, and review states are parsed twice

**File:** `internal/github/checks.go:209-235,315-326`, `internal/github/client.go:318-324`
**Issue:** Each page of `FetchDetail` sends the whole `detailQuery`, including the three reviewer connections of up to 50 nodes each. Only page 1's reviewers are used. With up to 5 pages every refresh this spends request budget for nothing. In `mapResponse`, `reviewStates` is built by a loop that duplicates what `reviewInputs` now does for the same nodes.
**Fix:** Use a second, smaller query for pages after the first (contexts only), or put the reviewer selection behind an `@include(if: $first)` variable. Build `reviewStates` from `reviewInputs` output.

### IN-05: Detail view is line-scroll only; help legend omits the commented and dismissed glyphs

**File:** `internal/ui/detail.go:113-116`, `internal/ui/help.go:55-56`
**Issue:** The detail body scrolls only with `j`/`k`/arrows, one line at a time. A PR with several hundred checks (the cap is 500, and `e` expands them) has no page, home or end keys. The help overlay explains `✗ … ✓` for reviewers but not `·` (commented) or `–` (dismissed), which the detail view draws.
**Fix:** Bind `pgup`, `pgdown`, `g` and `G` to `vp.PageUp`, `vp.PageDown`, `GotoTop` and `GotoBottom` in `detailKey`, and add the two glyphs to the legend line.

### IN-06: `ServerWait` and `NextDelay` agree only by coincidence when the error has no timing, and `ServerWait` ignores the known rate-limit reset

**File:** `internal/domain/backoff.go:93-101` (compare `:57-66`), test `internal/domain/backoff_test.go:140-145`
**Issue:** For a rate-limited error with no Retry-After and no reset, `NextDelay` waits `max(RateLimitFloor, failureWait(interval, failures))`. It also falls back to the snapshot's `rl.ResetAt` when the error has none. `ServerWait` returns a fixed `RateLimitFloor` and has no `rl` fallback. The doc comment says it "uses the order and bounds of NextDelay", but that is not fully true. `TestServerWait` checks `got == NextDelay(...)` only with `interval = 1 min`, `failures = 1` and `RateLimit{}`, where `failureWait` happens to equal `RateLimitFloor`. A different interval or a known `rl.ResetAt` would make the two paths disagree, and the test would not show it. In practice a detail request that is rate limited without timing pauses for 1 minute even when the list snapshot says the limit resets in 40 minutes. The next detail fetch then probably hits the limit again.
**Fix:** Either pass the snapshot's `ResetAt` into the detail path (`ServerWait(now, after, firstNonZero(reset, m.snap.RateLimit.ResetAt))`), or soften the doc comment and the test comment so they do not claim parity. Vary the interval and failures in the equivalence test so it does not rely on 1 min.

### IN-07: The help legend does not name the new "unknown" check class

**File:** `internal/ui/help.go:58-60` (summary line: `internal/ui/detail.go:307`)
**Issue:** The CHECKS summary can now print "N unknown" in its own style (WR-04). The legend lists only failing, cancelled, running, passing and skipped. The `?` help therefore never explains the new word or its glyph. This is the same gap as the "commented and dismissed glyphs" in the open IN-05, with one more entry.
**Fix:** Add `{domain.CheckOther, "unknown"}` to the `checks` slice in `help.go`. This changes the help golden files, so regenerate them.

### IN-08: A kept error on a later page sets `Partial` even though that page's reviewer data is never used

**File:** `internal/github/checks.go:333-338` (cause: the shared query, IN-04)
**Issue:** `Partial` is set from any page. The detail query re-sends the reviewer connections (`latestOpinionatedReviews`, `latestReviews`, `reviewRequests`) on every page, but `FetchDetail` reads reviewers from page 1 only (`:339-341`). If page 2 to 5 keeps a GraphQL error whose path is under `reviewRequests` or a review connection, the error cannot make anything on screen incomplete, yet `Partial` is set and the user sees "some details could not be loaded" for a complete view. The new test "a kept error on page 2" uses a reviewer path (`keptError`), and asserts `Partial == true`, so the test pins this false positive. The notice is a false alarm, not a hidden failure, and it only happens with more than 100 check runs. That makes it low severity.
**Fix:** Cheapest: accept it and say so in the `Partial` doc comment. Better: fix IN-04 (no reviewer fields after page 1), after which an error on a later page can only concern checks. If the error path is parsed, set `Partial` for a later page only when the path touches `statusCheckRollup`. Update the page-2 test to match.

### IN-09: The partial notice is untested at widths below its own length, and its refresh behaviour is untested

**File:** `internal/ui/detail_test.go:570-664` (notice: `internal/ui/detail.go:274-276`)
**Issue:** `TestDetailPartialNotice` covers widths 120 and 36. The notice is 32 cells and `detailLines` cuts with `ansi.Truncate(l, width, "")` and no ellipsis. In a pane under 32 cells it is cut mid-word ("some details could not be l"), with no test that says this is intended. The tests also never check that the notice goes away after a later successful fetch whose `Detail` has `Partial=false`, or that it stays while a failed refresh keeps the older data. Both follow from the code today, but nothing would catch a regression, for example someone merging a new result into the old one.
**Fix:** Add a short model-level test: open a detail with `Partial=true`, apply a `detailMsg` with `Partial=false` and assert the notice is gone; apply a `detailMsg` with an error and assert the notice is still there. Optionally shorten the text to "details incomplete" (18 cells) so it survives the narrowest supported pane.

---

_Reviewed: 2026-10-13_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

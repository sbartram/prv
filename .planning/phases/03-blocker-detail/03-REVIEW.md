---
phase: 03-blocker-detail
reviewed: 2026-10-12T00:00:00Z
depth: standard
files_reviewed: 31
files_reviewed_list:
  - cmd/prv/main.go
  - cmd/prv/main_test.go
  - internal/domain/checks.go
  - internal/domain/checks_test.go
  - internal/domain/domain_test.go
  - internal/domain/merge.go
  - internal/domain/merge_test.go
  - internal/domain/pr.go
  - internal/domain/reviewers.go
  - internal/domain/reviewers_test.go
  - internal/domain/status.go
  - internal/github/checks.go
  - internal/github/checks_test.go
  - internal/github/client.go
  - internal/github/client_test.go
  - internal/github/detail_test.go
  - internal/github/errors_test.go
  - internal/ui/cells.go
  - internal/ui/cells_test.go
  - internal/ui/detail.go
  - internal/ui/detail_test.go
  - internal/ui/help.go
  - internal/ui/help_test.go
  - internal/ui/keys.go
  - internal/ui/layout.go
  - internal/ui/layout_test.go
  - internal/ui/model.go
  - internal/ui/refresh.go
  - internal/ui/styles.go
  - internal/ui/view.go
  - internal/ui/view_test.go
findings:
  critical: 0
  warning: 4
  info: 5
  total: 9
status: issues_found
---

# Phase 3: Code Review Report

**Reviewed:** 2026-10-12
**Depth:** standard
**Files Reviewed:** 31
**Status:** issues_found

## Summary

Reviewed the phase 3 diff (`034f67d..HEAD`) of the detail view, the detail and failing-check GraphQL queries, the new domain derivations (checks, reviewers, merge) and the layout and cell work. `go vet ./...` and `go test -race ./...` pass.

Sanitising is consistent: every remote string (check names, workflow names, source paths, logins, team slugs) goes through `domain.Clean` at the github seam. IDs and cursors travel only as GraphQL variables. The URL opened from the detail view goes through `safeURL`. Error text on the detail screen comes from `Short()` only. I found no injection or secret-handling problem.

The defects are about robustness. The detail view is the first feature that runs a second request concurrently with the list refresh, and several earlier single-flight assumptions no longer hold. The main concerns are:

- A rate-limit pause path that skips the clamp every other pause path uses.
- A silent undercount of checks on partial GraphQL data, on the screen whose job is to show blockers.
- A shared "latest headers" capture that two overlapping requests now race on.
- A fold label that hides unknown-state checks as "passing".

## Warnings

### WR-01: Detail rate-limit pause bypasses the server-wait clamp

**File:** `internal/ui/detail.go:148-164`
**Issue:** `pauseForRateLimit` sets `m.pausedUntil` straight from `reset` or `now + after`. The list path goes through `domain.NextDelay`, which clamps every server-supplied delay to `[MinServerWait, MaxServerWait]` (1 s to 65 min). The comment on `MaxServerWait` says a hostile or broken value must not "park it for days", and `parseRetryAfter` explicitly saturates at 2^31 s and leaves the clamp to the caller. The detail path has no clamp. A rate-limited detail response with `Retry-After: 2147483647` or a far-future `X-RateLimit-Reset` therefore pauses all refreshes for years. The tick handler re-arms to `pausedUntil`, `manualRefresh` refuses with a flash, and `startDetailFetch` refuses. Only a restart recovers. A reset time in the past or `reset == now` also yields a zero or negative pause, so the rate-limited detail error is retried at once on the next refresh.
**Fix:** Reuse the domain clamp instead of a second, weaker copy of the policy. For example, export a helper from `domain` and use it here:
```go
// domain/backoff.go
func ServerWait(now time.Time, retryAfter time.Duration, reset time.Time) (time.Time, bool) {
    switch {
    case retryAfter > 0:
        return now.Add(clamp(retryAfter)), true
    case reset.After(now):
        return now.Add(resetWait(reset.Sub(now))), true
    }
    return time.Time{}, false
}
```
Then in `pauseForRateLimit`, call `domain.ServerWait(m.now(), after, reset)` and keep the "never moves earlier" rule. Add a test with `Retry-After: 2147483647` that asserts the pause is at most 65 minutes.

### WR-02: FetchDetail silently undercounts checks and reviewers on partial data

**File:** `internal/github/checks.go:296-346`
**Issue:** The code accepts partial GraphQL data (`detailPage` keeps the node on a `GraphQLError`). The wire comment says "GraphQL nulls a list element when a nested field errors". Null context nodes are then skipped (`if ctxNode != nil`) and `d.Read` is not incremented. Null reviews and requests are skipped in `reviewInputs` and `requestInputs`, with no signal either. `Truncated` stays false unless a later page fails or the page cap is hit. The result is a detail view with Read less than Total, or an error swallowed on page 1, that shows exact-looking counts ("2 failing · 10 passing") with no "read N of M runs" note. A failing check on a nulled node is invisible on the screen whose purpose is to show blockers. `fetchFailing` has the same gap, but it at least adds a warning for the error. `FetchDetail` does not. No test covers a null context node.
**Fix:** After the loop, mark the result as incomplete when anything was dropped:
```go
if d.Read < d.Total {
    d.Truncated = true // nodes were nulled or never read
}
```
Also set `d.Truncated = true` when a page returned a `GraphQLError` that was kept. Add a `detail_test.go` case with a `null` entry in `contexts.nodes` and assert `Truncated` and the "read N of M runs" text.

### WR-03: Shared "latest response headers" is now racy, and detail requests are invisible to the budget

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

### WR-04: Unknown and neutral checks are folded away under the label "more passing" and counted as "skipped"

**File:** `internal/ui/detail.go:308,329-336,392`, `internal/domain/checks.go:225-247`
**Issue:** With `expanded == false`, only failing, cancelled and running checks are listed. Every other state is folded into the faint line `… +N more passing`. That includes `CheckNeutral` (skipped, neutral, stale) and `CheckOther` (an unrecognised status or conclusion, which `ClassifyCheck` documents as "never failing and never passing"). So a check whose status prv does not understand is hidden under a line that says it passed. The summary has the matching problem: `CountChecks` puts `CheckOther` into `Neutral`, and the summary prints that count as "skipped". The `ClassifyCheck` doc promises unknown is never shown as passing, and the display breaks that promise for the core "is anything blocking me" question.
**Fix:** Either list `CheckOther` checks in the default (unfolded) view, or give the fold an accurate label. Minimal change:
```go
case domain.CheckFailing, domain.CheckCancelled, domain.CheckRunning, domain.CheckOther:
    shown = append(shown, c)
```
and word the fold line "+N more passing or skipped". Keep unknown out of the "skipped" count by adding a separate `Other` field to `CheckCounts`.

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

---

_Reviewed: 2026-10-12_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

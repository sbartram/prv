---
status: complete
phase: 03-blocker-detail
source: [03-VERIFICATION.md]
started: 2026-10-13T00:03:11Z
updated: 2026-10-13T01:14:29.959Z
---

## Current Test

[testing complete]

## Tests

### 1. Run `prv` in a real terminal, open the detail (enter) of a PR with a large CI matrix. Check failing checks sit on top, passing ones fold into '... +N more passing', e expands and collapses, j/k scroll, esc returns to the same PR.
expected: Layout is readable; the selected PR is unchanged after esc.
result: skipped
reason: "Deferred follow-up: defer UI tests until we have multiple PRs in multiple states to test with"

### 2. Leave the detail open across a refresh (press r), then drag the pane narrow (about 30 columns) and wide again.
expected: The 'updated' time and the checks refresh; the body re-lays out; the footer stays on the last line; esc returns to the same PR. The list drops columns in the D-13 order and compact rows appear below 40 columns.
result: skipped
reason: "Deferred follow-up: defer UI tests until we have multiple PRs in multiple states to test with"

### 3. Run against a PR that has a visible team review request (an org account).
expected: The row shows '...@org/slug' and the detail shows '@org/slug  pending'; a hidden team shows as '...team'.
result: skipped
reason: "Deferred follow-up: defer UI tests until we have multiple PRs in multiple states to test with"

### 4. Compare the MERGE column with the github.com merge box for several real PRs (mergeable, blocked, behind, conflicts, draft).
expected: Each row agrees with GitHub's own merge state.
result: skipped
reason: "Deferred follow-up: defer UI tests until we have multiple PRs in multiple states to test with"

### 5. Check the glyph widths of the check, merge, reviewer and compact-row glyphs in the user's font (the check, cancelled, behind, blocked and dismissed glyphs).
expected: Every glyph is one cell wide; no column shifts.
result: skipped
reason: "Deferred follow-up: defer UI tests until we have multiple PRs in multiple states to test with"

### 6. Decide the disposition of WR-01..WR-04 in 03-REVIEW-DISPOSITION.md (all still `open`): fix before ship, defer, or skip.
expected: Each warning has a recorded disposition.
result: issue
reported: "fix 1, 2, 4; defer 3"
severity: major

## Summary

total: 6
passed: 0
issues: 1
pending: 0
skipped: 5
blocked: 0

## Gaps

- gap_id: G-03-6-WR01
  truth: "A rate-limited detail response pauses refreshes only within the same [MinServerWait, MaxServerWait] clamp the list path uses (WR-01)"
  status: failed
  reason: "User reported: fix 1, 2, 4; defer 3 (WR-01 marked fix before ship)"
  severity: major
  test: 6
  root_cause: "internal/ui/detail.go pauseForRateLimit sets m.pausedUntil straight from reset or now+after, bypassing domain.NextDelay's clamp; a huge Retry-After or far-future reset pauses for years, a past reset gives a zero/negative pause"
  artifacts:
    - path: "internal/ui/detail.go"
      issue: "pauseForRateLimit (lines ~148-164) has no clamp"
    - path: "internal/domain/backoff.go"
      issue: "clamp policy not exported for reuse"
  missing:
    - "Export a domain helper (e.g. ServerWait) that applies the existing clamp and use it in pauseForRateLimit"
    - "Tests for huge Retry-After, far-future reset, and past/now reset"
  debug_session: ".planning/phases/03-blocker-detail/03-REVIEW.md#wr-01"
- gap_id: G-03-6-WR02
  truth: "The detail view marks its counts as partial ('read N of M runs') whenever GraphQL nulled nodes or a page returned a kept GraphQLError (WR-02)"
  status: failed
  reason: "User reported: fix 1, 2, 4; defer 3 (WR-02 marked fix before ship)"
  severity: major
  test: 6
  root_cause: "internal/github/checks.go FetchDetail skips nil context/review/request nodes without incrementing Read or setting Truncated, and keeps page-1 GraphQLError data with no signal"
  artifacts:
    - path: "internal/github/checks.go"
      issue: "FetchDetail (lines ~296-346) never sets Truncated for dropped nodes or kept GraphQLError"
  missing:
    - "Set d.Truncated = true when d.Read < d.Total and when a page's GraphQLError was kept"
    - "detail_test.go case with a null entry in contexts.nodes asserting Truncated and the 'read N of M runs' text"
  debug_session: ".planning/phases/03-blocker-detail/03-REVIEW.md#wr-02"
- gap_id: G-03-6-WR04
  truth: "Unknown (CheckOther) checks are never folded under '+N more passing' nor counted as 'skipped' (WR-04)"
  status: failed
  reason: "User reported: fix 1, 2, 4; defer 3 (WR-04 marked fix before ship)"
  severity: major
  test: 6
  root_cause: "internal/ui/detail.go lists only failing/cancelled/running when collapsed, folding CheckNeutral and CheckOther into '... +N more passing'; internal/domain/checks.go CountChecks puts CheckOther into Neutral, printed as 'skipped'"
  artifacts:
    - path: "internal/ui/detail.go"
      issue: "collapsed filter (~308, 329-336) and fold label (~392)"
    - path: "internal/domain/checks.go"
      issue: "CountChecks (~225-247) merges CheckOther into Neutral"
  missing:
    - "Show CheckOther in the collapsed list"
    - "Reword fold line to '+N more passing or skipped'"
    - "Add a separate Other field to CheckCounts so unknown is not counted as skipped; update summary text and golden files"
  debug_session: ".planning/phases/03-blocker-detail/03-REVIEW.md#wr-04"

## Deferred Follow-Ups

- test: 1
  idea: "defer UI tests until we have multiple PRs in multiple states to test with"
  deferred_at: 2026-10-12
- test: 2
  idea: "defer UI tests until we have multiple PRs in multiple states to test with"
  deferred_at: 2026-10-12
- test: 3
  idea: "defer UI tests until we have multiple PRs in multiple states to test with"
  deferred_at: 2026-10-12
- test: 4
  idea: "defer UI tests until we have multiple PRs in multiple states to test with"
  deferred_at: 2026-10-12
- test: 5
  idea: "defer UI tests until we have multiple PRs in multiple states to test with"
  deferred_at: 2026-10-12

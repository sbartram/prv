---
phase: 03-blocker-detail
plan: 09
subsystem: github-detail-fetch
tags: [go, github-graphql, detail-view, partial-data, gap-closure, tdd]

requires:
  - phase: 03-blocker-detail
    provides: "plan 03-06 FetchDetail and detailPage; plan 03-07 checksSummary that renders 'read N of M runs' when Detail.Truncated is set"
provides:
  - "FetchDetail marks Detail.Truncated when fewer check runs were read than GitHub reported (null context nodes)"
  - "FetchDetail marks Detail.Truncated when a page's GraphQL error was kept, also when every check node arrived"
  - "detailPage returns (node, partial bool, error); the bool is true only for a kept GraphQL error"
  - "End-to-end proof through the real github.Client and ui.Model that a nulled node reads 'read 1 of 2 runs' on screen"
affects: [phase-03-verification, 03-10]

plan_head_before: cfb3eec08c7b24b9537479c4df908cb420394a93
plan_head_after: c992ad5c1b41c0bc63f7f275cae21fbdc8589976

actuals:
  tokens: 2500
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "A GraphQL error kept next to data is reduced to a bool at the seam; its message never reaches the screen or a log"
    - "Partial reads are marked via the existing Truncated flag; the UI needed no change"

key-files:
  created: []
  modified:
    - internal/github/checks.go
    - internal/github/detail_test.go
    - cmd/prv/main_test.go

key-decisions:
  - "Reuse Detail.Truncated for both causes (Read < Total and kept GraphQL error) instead of a new field, so the UI renders 'read N of M runs' unchanged"
  - "A kept error on a complete read shows 'read 3 of 3 runs': the counts are marked as possibly incomplete whenever GitHub reports an error next to the data"

requirements-completed: [DETL-01, DETL-02]

coverage:
  - id: D1
    description: "FetchDetail sets Truncated when a null context node leaves Read below Total"
    requirement: DETL-01
    verification:
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetailPartial/a_null_context_node"
        status: pass
    human_judgment: false
  - id: D2
    description: "FetchDetail sets Truncated when a page's GraphQL error was kept, on page 1 and on a later page, even with every node present"
    requirement: DETL-02
    verification:
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetailPartial/page_1_kept_a_GraphQL_error"
        status: pass
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetailPartial/a_later_page_kept_a_GraphQL_error"
        status: pass
    human_judgment: false
  - id: D3
    description: "The detail screen shows 'read 1 of 2 runs' for a Detail answer with a nulled node and an errors[] member, through the real client and model"
    requirement: DETL-01
    verification:
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestDetailPartialEndToEnd"
        status: pass
    human_judgment: false
  - id: D4
    description: "Complete reads, rate limits, cancels and a missing node behave as before"
    verification:
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetail, TestFetchDetailPaging, TestFetchDetailRateLimited, TestFetchDetailNotFound"
        status: pass
    human_judgment: false

duration: 3 min
completed: 2026-10-13
status: complete
---

# Phase 3 Plan 09: Detail view marks partial data (gap G-03-6-WR02) Summary

**FetchDetail now sets Truncated when a null check node leaves Read below Total or when a page's GraphQL error was kept, so the detail screen says "read N of M runs" instead of showing exact-looking counts on partial data.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-10-13T01:35:14Z
- **Completed:** 2026-10-13T01:37:04Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- A null context node in a Detail response no longer vanishes silently: `FetchDetail` sets `Truncated` when `d.Read < d.Total`, and the CHECKS summary shows "read 1 of 2 runs" (proved end to end through `github.NewClient` and `ui.New`).
- `detailPage` now returns `(*wireDetailNode, bool, error)`. The bool is true only when it keeps a node next to a GraphQL error. `FetchDetail` sets `Truncated` from it, so a nulled reviewer or review request is no longer hidden when every check node arrived.
- Error rules are unchanged: a rate limit, a cancelled context and any page-1 failure are still returned; a later failing page still keeps the earlier pages. No UI code, `classify`, `captureTransport` or rate-limit code was touched (WR-03 stays deferred).

## Task Commits

1. **Task 1: null check node marks the read as partial**
   - RED `71c8869` (test) - `TestFetchDetailPartial/a_null_context_node` and `TestDetailPartialEndToEnd`
   - GREEN `c8f1198` (feat) - `Read < Total` sets `Truncated`
2. **Task 2: kept GraphQL error marks the read as partial**
   - RED `9d95d68` (test) - two kept-error subtests and the `withErrorsBody` helper
   - GREEN `c992ad5` (feat) - `detailPage` partial flag, `FetchDetail` uses it

**Plan metadata:** committed separately (docs: complete plan)

## TDD Gate Compliance

- **RED, Task 1:** `TestFetchDetailPartial/a_null_context_node` failed with "checks/Read/Total/Truncated = 2/2/3/false, want 2/2/3/true". `TestDetailPartialEndToEnd` failed because the view showed "1 failing" with no "read 1 of 2 runs". Semantic assessment: both failed on the planned assertion (Truncated false, text missing), not on a decode or setup error.
- **RED, Task 2:** both new subtests failed on "Truncated false, want true" with Read/Total already correct (3/3 and 150/150). Semantic assessment: they fail for the planned reason; a kept error alone was not marked.
- **GREEN:** all of the above pass with `-race`; `TestFetchDetail` and `TestFetchDetailPaging` (complete reads) still report `Truncated` false.
- No REFACTOR commit was needed.

## Files Created/Modified

- `internal/github/checks.go` - `FetchDetail` sets `Truncated` on `Read < Total` and on a kept error; `detailPage` returns a partial flag; doc comments updated
- `internal/github/detail_test.go` - `TestFetchDetailPartial` (3 subtests) and `withErrorsBody` helper
- `cmd/prv/main_test.go` - `partialDetailBody` and `TestDetailPartialEndToEnd`

## Decisions Made

- Reuse the existing `Truncated` flag for both causes so the UI renders "read N of M runs" with no change.
- A kept error on an otherwise complete read shows "read 3 of 3 runs"; this follows the gap's truth (counts are marked possibly incomplete whenever GitHub reports an error beside the data).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. `go vet ./...` and `go test ./... -count=1 -race` pass; `go.mod` and `go.sum` are unchanged.

## Known Stubs

None.

## Threat Flags

None. The kept GraphQL error is reduced to a bool (T-03-25); no new module (T-03-SC); T-03-24 is mitigated by the three `TestFetchDetailPartial` cases and `TestDetailPartialEndToEnd`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Gap G-03-6-WR02 is closed. Plan 03-10 is the next plan in the phase.

## Self-Check: PASSED

- FOUND: internal/github/checks.go, internal/github/detail_test.go, cmd/prv/main_test.go
- FOUND commits: 71c8869, c8f1198, 9d95d68, c992ad5

---
*Phase: 03-blocker-detail*
*Completed: 2026-10-13*

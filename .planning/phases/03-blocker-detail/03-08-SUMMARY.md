---
phase: 03-blocker-detail
plan: 08
subsystem: ui-detail-view
tags: [go, bubbletea-v2, detail-view, refresh, rate-limit, narrow-panes, goldens, tdd]

requires:
  - phase: 03-blocker-detail
    provides: "plan 03-07 detail view (detailSeq, detailInFlight, startDetailFetch, applyDetail, syncDetail, viewport); plan 03-06 FetchDetail; Phase 2 pausedUntil, backoffer and the tick re-arm"
provides:
  - "The open detail refreshes after every successful list refresh as its own Cmd (one Detail query, none while closed, in flight, paused or gone)"
  - "A PR that leaves the rows keeps its last data under a faint 'no longer in list' line; fetching stops; it comes back and is selected again when the refresh returns it"
  - "A failed detail refresh keeps the data and adds '✗ refresh failed HH:MM (REASON) · details from HH:MM:SS'; a rate-limited detail error extends the refresh pause"
  - "Compact detail lines below 40 cells (glyph and name only)"
  - "Goldens for detail-loading, detail-error, detail-gone and detail-scrolled at 60x15, 80x24, 120x40 and 0x0"
  - "End-to-end test of the Detail request counts through the real github.Client"
affects: [phase-03-verification]

plan_head_before: 68d3e968d79a25021443294d9017254a00cc3a56
plan_head_after: 90511af6fd7dbb7c9aa9ef5f0c95f4188ac7901c

actuals:
  tokens: 13700
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Detail follows the list: the fetchedMsg success branch batches refreshDetail() after schedule(); the detail fetch is its own sequenced Cmd, never part of the list fetch"
    - "The open PR is looked up by ref in the rows on screen; the view keeps its own copy and never reads the row at the cursor"
    - "An active rate-limit pause never shrinks: schedule() keeps a still-running pause that a detail error extended"
    - "Compact lines come from one helper (glyphLine) for checks and reviewers"
    - "End-to-end tests step through Cmds with a 250 ms cut-off so the 60 s refresh tick is dropped, not awaited"

key-files:
  created:
    - internal/ui/testdata/TestGolden/detail-loading_0x0.golden
    - internal/ui/testdata/TestGolden/detail-loading_60x15.golden
    - internal/ui/testdata/TestGolden/detail-loading_80x24.golden
    - internal/ui/testdata/TestGolden/detail-loading_120x40.golden
    - internal/ui/testdata/TestGolden/detail-error_0x0.golden
    - internal/ui/testdata/TestGolden/detail-error_60x15.golden
    - internal/ui/testdata/TestGolden/detail-error_80x24.golden
    - internal/ui/testdata/TestGolden/detail-error_120x40.golden
    - internal/ui/testdata/TestGolden/detail-gone_0x0.golden
    - internal/ui/testdata/TestGolden/detail-gone_60x15.golden
    - internal/ui/testdata/TestGolden/detail-gone_80x24.golden
    - internal/ui/testdata/TestGolden/detail-gone_120x40.golden
    - internal/ui/testdata/TestGolden/detail-scrolled_0x0.golden
    - internal/ui/testdata/TestGolden/detail-scrolled_60x15.golden
    - internal/ui/testdata/TestGolden/detail-scrolled_80x24.golden
    - internal/ui/testdata/TestGolden/detail-scrolled_120x40.golden
  modified:
    - internal/ui/detail.go
    - internal/ui/model.go
    - internal/ui/detail_test.go
    - internal/ui/layout_test.go
    - internal/ui/view_test.go
    - cmd/prv/main_test.go

key-decisions:
  - "D-12 'the list' is the rows on screen after the filters, so the note and the esc landing spot agree (carried from the plan)"
  - "A rate-limited detail error extends m.pausedUntil directly and arms no tick; the pending tick re-arms itself to the end of the pause (RESEARCH A6)"
  - "schedule() keeps a pause that is still running when a list result arrives, so a list result cannot end a pause the detail error set"
  - "A gone PR is not refetched; while gone, the view shows the last data and o still opens the detail PR's own URL"

patterns-established:
  - "refreshDetail() is the one place that syncs a detail view with the list: fresh PR copy, selection follows the ref, gone flag, viewport lines, next fetch"
  - "detailNotes() holds the lines under the title (gone, failed refresh); the CHECKS line holds loading, pause and first-load error"

requirements-completed: [DETL-03, NAV-05]

coverage:
  - id: D1
    description: "While the detail view is open, each successful refresh starts one FetchDetail for the open PR as its own Cmd; none when closed, in flight, paused, gone or after a failed refresh"
    requirement: DETL-03
    verification:
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailRefreshes"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailNoFetchWhenClosed"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailNoOverlap"
        status: pass
      - kind: integration
        ref: "cmd/prv/main_test.go#TestDetailEndToEnd"
        status: pass
    human_judgment: false
  - id: D2
    description: "Rate limits: a rate-limited detail error extends the pause to its reset time or to now plus Retry-After and never earlier; no request goes out during a pause; enter during a pause shows 'CHECKS  paused until HH:MM (rate limit)'"
    requirement: DETL-03
    verification:
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailPausedNoRequest"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailRateLimitExtendsPause"
        status: pass
    human_judgment: false
  - id: D3
    description: "A failed detail refresh keeps the last data, scroll offset and expand state and adds the failure line under the title; the next success removes it"
    requirement: DETL-03
    verification:
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailErrorKeepsData"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailKeepsScrollAndExpand"
        status: pass
    human_judgment: false
  - id: D4
    description: "A PR that leaves the list: the view stays open with its last data and 'no longer in list', fetching stops, o opens the detail PR's URL; esc lands on the nearest surviving neighbour; a PR that comes back is selected again; a pure reorder keeps the same PR selected and on screen"
    requirement: DETL-03
    verification:
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailGone"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailGoneOpensDetailURL"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailEscAfterReorder"
        status: pass
    human_judgment: false
  - id: D5
    description: "A stale or other-PR detail result is dropped, also after esc and enter on another PR"
    requirement: DETL-03
    verification:
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailStaleAfterSwitch"
        status: pass
    human_judgment: false
  - id: D6
    description: "Below 40 cells the detail keeps glyph and name only; from 40 cells the Source column drops before names shrink; a resize lays the body out again, keeps the offset inside the content and the footer on the last line; heights 1 and 2 show header and header+footer"
    requirement: NAV-05
    verification:
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailNarrow"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailResize"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailTinyHeights"
        status: pass
    human_judgment: false
  - id: D7
    description: "For widths 1..200 at heights 3, 15, 50 and heights 1..50 at widths 30, 60, 120 the detail (loaded, expanded, gone, error) never panics, draws no line wider than the pane and no more lines than the pane; goldens pin every detail state at 60x15, 80x24, 120x40 and 0x0"
    requirement: NAV-05
    verification:
      - kind: unit
        ref: "internal/ui/layout_test.go#TestNeverOverflowsDetail"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestGolden"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestViewFitsScreen"
        status: pass
    human_judgment: false
  - id: D8
    description: "In a real terminal: leave the detail open across a refresh (the updated time changes and the checks update), drag the pane narrow and wide again (the body re-lays out, the footer stays at the bottom), press esc and see the same PR selected"
    requirement: NAV-05
    verification: []
    human_judgment: true
    rationale: "The plan's own human-check: how the live refresh and the resize feel in a real terminal cannot be asserted by unit tests; the verifier harvests it at end of phase"

duration: 6min
completed: 2026-10-12
status: complete
---

# Phase 3 Plan 08: The open detail stays current and honest, and fits any pane Summary

**The open detail view refetches after every successful list refresh (one Detail query, none when closed, in flight, paused or gone), says "no longer in list" and keeps its last data when its PR leaves the rows, shows a failure line on a failed refresh, extends the rate-limit pause on a rate-limited detail error, drops to glyph-and-name lines below 40 cells, and has goldens for all four new states**

## Performance

- **Duration:** 6 min
- **Started:** 2026-10-12T23:45:38Z
- **Completed:** 2026-10-12T23:52:03Z (last task commit; metadata commits follow)
- **Tasks:** 2 (each as a RED and a GREEN commit)
- **Files modified:** 22 (5 Go files plus `cmd/prv/main_test.go`, 16 new goldens)

## Accomplishments

- `refreshDetail()` runs from the `fetchedMsg` success branch while the detail is open. It looks the open ref up in the rows on screen, takes the fresh PR copy, moves `selected` and `cursor` to it (so esc returns to the right PR after a reorder, or after the PR left and came back), sets or clears `gone`, re-lays the viewport and returns the next detail fetch as its own Cmd.
- `startDetailFetch()` now refuses while a fetch is in flight, the PR is gone, or `m.now()` is before `pausedUntil`. A rate-limited detail error moves `pausedUntil` to its reset time, or to now plus Retry-After, never earlier, and arms no tick.
- Lines under the title: faint `no longer in list`, and `✗ refresh failed 14:33 (network) · details from 14:32:05` while older data is shown. The CHECKS line reads `paused until HH:MM (rate limit)` before the first result during a pause. Data, scroll offset and expand state survive a refresh; the viewport clamps the offset when the content shrinks.
- Below `compactBelow` (40 cells) check and reviewer lines are `  GLYPH NAME`, the name cut with `…`; from 40 cells the 03-07 rule stays (Source drops first, then names shrink).
- `TestDetailEndToEnd` runs the real `github.Client` over the routed transport: enter sends one Detail request with id `PR_7`, `r` sends Watch then Detail, and after esc a refresh sends no Detail request.
- 16 new goldens: `detail-loading`, `detail-error`, `detail-gone`, `detail-scrolled` at 60x15, 80x24, 120x40 and 0x0. No existing golden changed.

## Task Commits

Each task was committed atomically as a RED and a GREEN commit:

1. **Task 1: The open detail refreshes with the list, never shows a stale or wrong PR, and stays honest when its PR leaves the list**
   - RED `c93be96` (test)
   - GREEN `cffd00a` (feat)
2. **Task 2: The detail view fits any pane and survives resizes; every detail state has goldens**
   - RED `b623508` (test)
   - GREEN `90511af` (feat)

**Plan metadata:** recorded in the docs commits that follow this file.

## TDD Gate Compliance

The plan type is `execute` with `tdd="true"` tasks, so there is no plan-level gate. Each task followed RED then GREEN. No REFACTOR commit was needed.

- **RED, Task 1 (`c93be96`):** 11 new `TestDetail*` tests plus `TestDetailEndToEnd`. The commit carries a type-only stub (`detailState.gone`, `errAt`) so the package compiles. Failures name the planned behaviour: "FetchDetail ids = [PR_3] after a refresh, want [PR_3 PR_3]", "enter during a pause started a request", "pausedUntil = 0001-01-01 ..., want 2026-10-11 14:42:05", "view missing '✗ refresh failed 14:33 (network)...'", "no longer in list", "after a refresh: 2 Watch and 1 Detail requests". `TestDetailNoFetchWhenClosed`, `TestDetailGoneOpensDetailURL`, `TestDetailEscAfterReorder`, `TestDetailStaleAfterSwitch` and the "esc lands on the nearest neighbour" subtest passed at RED by design: they are guards for behaviour that must stay true once refreshDetail exists. Semantic assessment: every failure is an assertion on the planned behaviour, none is a compile, load or fixture fault.
- **RED, Task 2 (`b623508`):** `TestDetailNarrow` failed with "no line is exactly '  ✗ lint'" and "the word 'timed out' is still drawn"; `TestDetailResize` failed with "36x12: the body was not laid out again at the new width"; the 16 new golden subtests failed on the missing files. `TestDetailTinyHeights`, `TestNeverOverflowsDetail`, `TestViewFitsScreen` and `TestResizeKeepsSelection` passed at RED because 03-07 already handles those cases; they pin them. Semantic assessment: the failing tests fail on the planned assertions.
- `gsd_run check tdd-red-evidence` supports TAP, JUnit, swift-testing and unittest reports; plain `go test` output is none of these, so the classifier was not run. The semantic assessment above was done by reading the failure text of each run.
- **GREEN:** `cffd00a` and `90511af`; `go vet ./...` and `go test ./... -count=1 -race` pass.

## Files Created/Modified

- `internal/ui/detail.go` - `gone`/`errAt`, `refreshDetail`, `pauseForRateLimit`, `errReason`, `detailNotes`, pause line, `glyphLine`, `appendFold`, compact check and reviewer lines
- `internal/ui/model.go` - `fetchedMsg` success branch batches `refreshDetail()`; `schedule()` keeps a running pause
- `internal/ui/detail_test.go` - `urlSnap`, `liveDetail`, `openRow`, `refresh` helpers and 14 new tests
- `internal/ui/layout_test.go` - `TestNeverOverflowsDetail`
- `internal/ui/view_test.go` - `stateModel` states `detail-loading`, `detail-error`, `detail-gone`, `detail-scrolled`; added to `TestGolden` and `TestViewFitsScreen`
- `cmd/prv/main_test.go` - Detail route and `ops` log in `routedTransport`, `detailBody`, `step`/`settle`/`press` helpers, enter/esc in `key`, `TestDetailEndToEnd`
- `internal/ui/testdata/TestGolden/detail-{loading,error,gone,scrolled}_{0x0,60x15,80x24,120x40}.golden` - new

## Decisions Made

- See `key-decisions`. The plan's flagged assumptions were taken as written: "the list" is the rows on screen, the rate-limited detail error arms no second tick, and a gone PR is not refetched.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical] A list result could end a pause that a detail error had set**
- **Found during:** Task 1 (`TestDetailPausedNoRequest`, `TestDetailRateLimitExtendsPause`)
- **Issue:** `schedule()` assigned `m.pausedUntil = d.PausedUntil` on every list result. A list fetch already in flight when a rate-limited detail error extended the pause would wipe the pause when its result landed, so a request could go out before the reset time. The plan truth says no request of any kind goes out before then.
- **Fix:** `schedule()` keeps the previous `pausedUntil` when it is still in the future and later than the new one. Existing flows never have an active pause at that point, so only the new case is affected.
- **Files modified:** internal/ui/model.go
- **Verification:** `TestDetailRateLimitExtendsPause/a_refresh_result_does_not_end_the_pause`; the full suite passes.
- **Committed in:** `cffd00a`

**2. [Plan variation, minor] A separate `TestNeverOverflowsDetail` instead of extending `TestNeverOverflows`**
- The plan says to extend `TestNeverOverflows`. A second test with the same loops keeps the list property test unchanged and names the state in each failure. The plan's `-run` pattern `TestNeverOverflows` matches both, so the verify command runs it.

**3. [Plan variation, minor] Test helpers in `cmd/prv/main_test.go`**
- `key("enter")` and `key("esc")` were missing from the test helper (they returned `e`); they are added. The refresh in `TestDetailEndToEnd` uses the `r` key and a `step`/`settle` helper that drops any Cmd slower than 250 ms (the 60 s refresh tick), because `tickMsg` is unexported and `runCmd` does not run the Cmds that Update returns.

**4. [Plan variation, minor] Small refactors under the plan's files**
- The failure-reason lookup moved from `checksSummary` into `errReason`, and the fold line into `appendFold`, because the new note line and the compact branch need the same code. No behaviour change; existing goldens are unchanged.

---

**Total deviations:** 4 (1 missing-critical fix, 3 minor plan variations)
**Impact on plan:** the schedule guard was needed for the plan's own no-request-during-a-pause truth. No scope creep.

## Issues Encountered

- A first draft of the stale-test guard text assumed `faintStyle` lines would be matched with the status line; `TestDetailNarrow` first flagged the title status line `✓ approved` as a leaked word. The test now checks only indented lines (checks and reviewers).
- The detail footer at 60 cells still cuts to `j/k scroll …` (03-07 behaviour, unchanged).

## Known Stubs

None.

## Threat Flags

None. The only outbound actions are the existing read-only Detail query (now once per successful refresh while the view is open) and `o`, which opens `detail.pr.URL` through `safeURL`. `detail.go` never reads the list row at the cursor (grep gate passes). No module was added (`git diff --exit-code go.mod go.sum` is clean).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 3 has no plans left after this one; ready for phase verification. Open for the verifier: the human-check of this plan (leave the detail open across a refresh, resize, esc) and the live run of the Detail query from 03-06 and 03-07.

## Self-Check: PASSED

- Files: `internal/ui/detail.go`, `internal/ui/detail_test.go`, `cmd/prv/main_test.go` and the 16 new golden files exist.
- Commits `c93be96`, `cffd00a`, `b623508`, `90511af` are ancestors of HEAD; `git rev-list --count` from the ledger base gives 4.
- `go vet ./...` and `go test ./... -count=1 -race` pass; the 12 Task 1 tests and the Task 2 tests report `--- PASS`; `git diff --exit-code go.mod go.sum` is clean; the golden hygiene check prints nothing; the grep gates (`m.rows[m.cursor]`, `LightDark|AdaptiveColor|#hex`) pass.

---
*Phase: 03-blocker-detail*
*Completed: 2026-10-12*

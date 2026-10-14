---
phase: 03-blocker-detail
plan: 02
subsystem: github-checks
tags: [go, github-graphql, checks, rate-limit, bubbletea-v2, lipgloss, ansi]

requires:
  - phase: 03-blocker-detail
    provides: "plan 03-01 tracer: FailingChecks nodes(ids:) query, domain.LatestChecks / SummarizeFailing, ciCell"
provides:
  - "Step-2 robustness tests for the failing-check name: re-runs, truncated pages, partial GraphQL errors, chunking of every failing PR, rate limit, server error, cancel, summed cost"
  - "Three step-2 fixtures: failing_rerun.json (vscode shape), failing_truncated.json, failing_partial.json"
  - "ciCell shortens only the check name (display width, CJK and ZWJ safe) and keeps ' (cancelled)' and ' +N'"
affects: [03-03, 03-06, 03-07]

plan_head_before: 830cbf617c0b178cc2e917413fa414fdd82279c2
plan_head_after: a14d0e9670b2cbb3527c8faf7a996615c25120bd

actuals:
  tokens: 8000
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Generated GraphQL page bodies in tests (manyBodies / failingFor) to exercise chunking at 30 and 120 PRs without large checked-in fixtures"
    - "Suffix-preserving cell cut: shorten the variable part with ansi.Truncate, keep fixed suffixes whole, fall back to the cut plain badge under 2 cells of name"

key-files:
  created:
    - internal/github/checks_test.go
    - internal/github/testdata/failing_rerun.json
    - internal/github/testdata/failing_truncated.json
    - internal/github/testdata/failing_partial.json
    - internal/ui/cells_test.go
  modified:
    - internal/domain/checks_test.go
    - internal/ui/cells.go

key-decisions:
  - "No production gap in fetchFailing / mapContext / domain: the 03-01 implementation already met every Task 1 behavior, so Task 1 is a tests-and-fixtures commit; each test was proven to bite by mutating checks.go (drop LatestChecks, ignore hasNextPage, continue after a failed chunk) and restoring it"
  - "failing_rerun.json also holds PR_16 (a FAILURE run superseded by a passing re-run): with the plan's PR_11 shape alone, dropping LatestChecks changed nothing because cancelled runs are never counted in '+N'"
  - "ciCell keeps the plain-badge fallback (cut to the width) when fewer than 2 cells of name fit, so a very narrow pane never shows a name stub"

patterns-established:
  - "Step-2 tests build discovery and FailingChecks bodies in code and route by opName through an rtFunc"
  - "A test that depends on a fixture property (7 raw failing-or-cancelled runs) asserts that property on the fixture itself"

requirements-completed: [LIST-07]

coverage:
  - id: D1
    description: "A failing PR's row names one failing check on re-run-heavy data: superseded cancelled or failed runs add no '+N' and are never named"
    requirement: LIST-07
    verification:
      - kind: unit
        ref: "internal/github/checks_test.go#TestFailingRerunFixture"
        status: pass
      - kind: unit
        ref: "internal/domain/checks_test.go#TestSummarizeLatestRerun"
        status: pass
    human_judgment: false
  - id: D2
    description: "A truncated contexts page, a null or missing node, or a failed step-2 query leaves the plain failing badge; no name is ever guessed"
    requirement: LIST-07
    verification:
      - kind: unit
        ref: "internal/github/checks_test.go#TestFailingTruncatedUnnamed"
        status: pass
      - kind: unit
        ref: "internal/github/checks_test.go#TestFailingPartialErrors"
        status: pass
      - kind: unit
        ref: "internal/github/checks_test.go#TestFailingStepServerError"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every failing PR is looked up in read-only chunks of 25 (30 -> 25+5, 120 -> 25,25,25,25,20), none when nothing fails; cost is summed over all requests"
    requirement: LIST-07
    verification:
      - kind: unit
        ref: "internal/github/checks_test.go#TestFailingChunks"
        status: pass
      - kind: unit
        ref: "internal/github/checks_test.go#TestFailingCostSummed"
        status: pass
      - kind: other
        ref: "! grep -rnE 'mutation *[A-Z{(]' internal/github --include='*.go'"
        status: pass
    human_judgment: false
  - id: D4
    description: "A rate-limited or cancelled step-2 request fails the whole refresh like a discovery page (empty snapshot, error returned)"
    requirement: LIST-07
    verification:
      - kind: unit
        ref: "internal/github/checks_test.go#TestFailingStepRateLimited"
        status: pass
      - kind: unit
        ref: "internal/github/checks_test.go#TestFailingStepCanceled"
        status: pass
    human_judgment: false
  - id: D5
    description: "A long, CJK or emoji check name is shortened inside the 24-cell CI cell while ' (cancelled)' and ' +N' stay whole"
    requirement: LIST-07
    verification:
      - kind: unit
        ref: "internal/ui/cells_test.go#TestCICell"
        status: pass
      - kind: unit
        ref: "internal/ui/cells_test.go#TestCICellCap"
        status: pass
    human_judgment: false

duration: 3 min
completed: 2026-10-12
status: complete
---

# Phase 03 Plan 02: Failing-check names stay right on messy real data and in a narrow CI cell Summary

**Failing-check names proven correct on superseded re-runs, truncated pages, partial GraphQL errors, 120-PR chunking, rate limits and cancel, plus a suffix-preserving CI cell cut that is safe for CJK and ZWJ emoji names.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-10-12T23:00:07Z
- **Completed:** 2026-10-12T23:03:28Z (code); metadata commit follows
- **Tasks:** 2
- **Files modified:** 7 (5 created, 2 modified)

## Accomplishments
- Task 1 pins the step-2 behavior with 8 named github tests, 3 fixtures and a domain composition test. The 03-01 code already satisfied all of it, so no production change was needed. Mutation checks confirm the tests fail when the behavior breaks.
- Task 2 replaces the 03-01 fallback in `ciCell`: only the name is cut (`ansi.Truncate`, display cells), `" (cancelled)"` and `" +N"` stay, and under 2 cells of name the plain badge is cut to the width.

## Task Commits

1. **Task 1: Names stay right on messy real data** - `24d3164` (test). No GREEN commit: behavior was already implemented in 03-01 (see TDD Gate Compliance).
2. **Task 2: Suffix-preserving CI cell cut** - `b0792dd` (test, RED), `a14d0e9` (feat, GREEN)

**Plan metadata:** recorded in the following docs commit (`docs(03-02): complete ...`).

## Files Created/Modified
- `internal/github/checks_test.go` - re-run, truncated, partial, chunking, rate-limit, 502, cancel and cost tests; helpers `manyBodies`, `failingFor`, `manyTransport`, `readBody`
- `internal/github/testdata/failing_rerun.json` - vscode-shaped re-runs for PR_11 (7 raw failing-or-cancelled runs, one latest failure) and a superseded failure for PR_16
- `internal/github/testdata/failing_truncated.json` - PR_11 with `hasNextPage: true`
- `internal/github/testdata/failing_partial.json` - `nodes: [null, PR_12]` with a NOT_FOUND error at `nodes[0]`
- `internal/domain/checks_test.go` - `TestSummarizeLatestRerun` (LatestChecks + SummarizeFailing on fixture shapes, status context vs check run with one name)
- `internal/ui/cells.go` - name-only cut with `ansi.Truncate`
- `internal/ui/cells_test.go` - `TestCICell`, `TestCICellCap`

## Decisions Made
- Left `fetchFailing` / `mapContext` / domain code untouched: all plan behaviors already held. Proof by mutation (see Deviations), not by assumption.
- Added a second PR (PR_16, failed run superseded by a passing run) to `failing_rerun.json` so the fixture detects a missing de-duplication.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Test gap] vscode-shaped fixture alone did not detect a missing LatestChecks**
- **Found during:** Task 1 (mutation check of the new tests)
- **Issue:** With only the plan's PR_11 shape, removing `LatestChecks` still produced "Linux / test" with no "+N", because cancelled runs are never added to the failing count (03-01 decision). The test could not catch the regression it exists for.
- **Fix:** Added PR_16 to `failing_rerun.json` (older FAILURE "deploy", newer SUCCESS "deploy") and asserted PR_16 stays unnamed; added the 7-raw-runs guard on the fixture and `TestSummarizeLatestRerun` in the domain package.
- **Files modified:** `internal/github/testdata/failing_rerun.json`, `internal/github/checks_test.go`, `internal/domain/checks_test.go`
- **Verification:** with `LatestChecks` removed, `TestFailingRerunFixture` and `TestFailingNames` fail; restored, all pass.
- **Committed in:** `24d3164`

---

**Total deviations:** 1 auto-fixed (1 test gap)
**Impact on plan:** Strengthens a plan-required test; no scope change.

## TDD Gate Compliance

- **Task 1 (RED):** the tests passed on first run, which the fail-fast rule treats as unexpected GREEN. Investigated: the plan describes robustness of behavior 03-01 already built (the plan text expects only "gap fixes"). Mutation runs (drop `LatestChecks`; set `complete = true`; `continue` instead of `return nil` after a failed chunk) each failed the matching test, so the tests are not vacuous. There is no `feat(03-02)` commit for Task 1 because there was nothing to implement.
- **Task 2 (RED):** `go test ./internal/ui -run TestCICell` failed on the planned assertions (`ciCell = "✗ failing"` where a shortened name with its suffix was expected; 5 subtests plus `TestCICellCap`). Semantic assessment: the target tests executed and failed on the planned width and suffix assertions, not on compile, load or fixture errors. `gsd_run check tdd-red-evidence` was not run because it has no adapter for `go test` output.
- **GREEN:** `a14d0e9`, all `TestCICell*` pass. No REFACTOR commit.

## Issues Encountered
None. `.planning/config.json` carries a pre-existing uncommitted change that was left alone and never staged.

## Known Stubs
None.

## Threat Flags
None. No new endpoint, auth path or file access; the FailingChecks query stays read-only and no GraphQL write document exists in `internal/github`.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `ciCell(pr, w)` keeps its signature; plan 03-03 (narrow list layout) can call it unchanged.
- No blockers.

## Self-Check: PASSED

- Files exist: checks_test.go, the three fixtures, cells_test.go, cells.go, domain/checks_test.go.
- Commits `24d3164`, `b0792dd`, `a14d0e9` are ancestors of HEAD.
- `go vet ./...` and `go test ./... -count=1 -race` pass; `git diff --exit-code go.mod go.sum` exits 0; no `mutation` document in `internal/github`.

---
*Phase: 03-blocker-detail*
*Completed: 2026-10-12*

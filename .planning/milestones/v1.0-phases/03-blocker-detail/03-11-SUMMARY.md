---
phase: 03-blocker-detail
plan: 11
subsystem: detail-view
tags: [go, bubbletea-v2, detail-view, partial-data, gap-closure, tracer, tdd]

requires:
  - phase: 03-blocker-detail
    provides: "plan 03-09 FetchDetail partial-read handling (kept GraphQL error, null context nodes, Read/Total); plan 03-10 detail view with CheckCounts.Other"
provides:
  - "domain.Detail.Partial: a GraphQL error was kept next to the data. Detail.Truncated now means only 'not every check run was read'"
  - "FetchDetail sets Partial (not Truncated) on a kept GraphQL error"
  - "detailNotes shows the faint 'some details could not be loaded' line under the title when the detail data is Partial"
  - "'read N of M runs' shows only when runs were not read; 'CHECKS  no checks' stays exact"
  - "detail-partial golden state at 0x0, 60x15, 80x24 and 120x40"
affects: [phase-03-verification, 03-REVIEW-DISPOSITION]

plan_head_before: a2f111f4ba129180bd9212e216f70b7ead89e7d0
plan_head_after: b1fb1420a3dae76fbee5fbdd821eb432b8203260

actuals:
  tokens: 5600
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "One flag per cause: Truncated = runs not read (shown as 'read N of M runs' on the CHECKS line); Partial = GraphQL error kept (shown as a separate faint notice under the title)"
    - "Tests that arrive after the fix get their RED evidence from named mutation runs"

key-files:
  created:
    - internal/ui/testdata/TestGolden/detail-partial_0x0.golden
    - internal/ui/testdata/TestGolden/detail-partial_60x15.golden
    - internal/ui/testdata/TestGolden/detail-partial_80x24.golden
    - internal/ui/testdata/TestGolden/detail-partial_120x40.golden
  modified:
    - internal/domain/checks.go
    - internal/github/checks.go
    - internal/github/detail_test.go
    - internal/ui/detail.go
    - internal/ui/detail_test.go
    - internal/ui/view_test.go
    - cmd/prv/main_test.go

key-decisions:
  - "Truncated means only 'check runs were not read'; Partial means 'a GraphQL error was kept next to the data'. This replaces the 03-09 decision to reuse Truncated for both causes"
  - "The Partial notice goes in detailNotes, on its own faint line under the title, not on the CHECKS line: the kept error can concern reviewers or review requests, a separate line cannot be cut off in a narrow pane, and 'no checks' stays exactly 'CHECKS  no checks'"
  - "checksSummary keeps the condition d.data.Truncated unchanged and gets no Read < Total guard; after the change Truncated is set only for runs not read"
  - "The kept error message never reaches the screen or a log: detailPage still reduces it to a bool and the notice text is fixed"

requirements-completed: [DETL-01, DETL-02]

coverage:
  - id: D1
    description: "A kept GraphQL error next to complete check data (Read == Total) shows 'some details could not be loaded' under the title and never 'read N of N runs' (WR-05 case 1)"
    requirement: DETL-01
    verification:
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetailPartial/page_1_kept_a_GraphQL_error"
        status: pass
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestDetailKeptErrorEndToEnd/every_run_read"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailPartialNotice/every_run_read"
        status: pass
    human_judgment: false
  - id: D2
    description: "A PR with no statusCheckRollup and a kept GraphQL error on page 1 reads 'CHECKS  no checks' with the notice, never 'read 0 of 0 runs' (WR-05 case 2)"
    requirement: DETL-02
    verification:
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetailPartial/no_rollup_and_a_kept_GraphQL_error"
        status: pass
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestDetailKeptErrorEndToEnd/no_checks"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailPartialNotice/no_checks"
        status: pass
    human_judgment: false
  - id: D3
    description: "'read N of M runs' still shows whenever runs were not read: page cap, failed later page, or null context nodes (end to end 'read 1 of 2 runs')"
    requirement: DETL-01
    verification:
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetailPaging"
        status: pass
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestDetailPartialEndToEnd"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailTruncatedSummary"
        status: pass
    human_judgment: false
  - id: D4
    description: "The partial detail view has reviewed golden files at all four sizes and no existing golden changed"
    requirement: DETL-02
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestGolden/detail-partial_*"
        status: pass
    human_judgment: false

duration: 5min
completed: 2026-10-13
status: complete
---

# Phase 03 Plan 11: Each partial-read cause gets its own honest text in the detail view (gap G-03-6, review WR-05) Summary

**`Detail.Partial` splits a kept GraphQL error from a short check-run read, so the detail view shows a faint "some details could not be loaded" line for the first and "read N of M runs" only for the second.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-10-13T02:21:34Z
- **Completed:** 2026-10-13T02:26Z
- **Tasks:** 2
- **Files modified:** 11 (7 modified, 4 new goldens)

## Accomplishments

- Closed UAT gap G-03-6 (review finding WR-05). A kept reviewer error with every run read no longer shows "read 3 of 3 runs". A PR with no checks and a kept error shows "CHECKS  no checks", not "read 0 of 0 runs".
- Both WR-05 cases are proven end to end through the real `github.Client` and `ui.Model` (TestDetailKeptErrorEndToEnd), at the client level (TestFetchDetailPartial, 4 subtests) and at the model level (TestDetailPartialNotice, 6 cases).
- A short read is unchanged: page cap, failed later page and null nodes still show "read N of M runs" (D-03, D-08).
- `detail-partial` golden state added at 0x0, 60x15, 80x24 and 120x40; the notice sits on its own line above CHECKS, the CHECKS line has no run count and the footer is the last line.

## Task Commits

1. **Task 1: A kept GraphQL error reaches the screen as "some details could not be loaded", never as a run count (tracer)** - `e14b0f7` (fix)
2. **Task 2: Every Truncated/Partial combination has pinned rendered output, and the partial detail has goldens at every size** - `b1fb142` (test)

**Plan metadata:** committed as `docs(03-11)` after this file.

### Task 1 RED evidence

Tests were written first. Before the production change, `go test ./internal/github ./cmd/prv -run 'TestFetchDetailPartial|TestDetailKeptErrorEndToEnd|TestDetailPartialEndToEnd'` failed for the expected reasons only:
- TestFetchDetailPartial/page_1_kept_a_GraphQL_error: `Truncated/Partial = true/false, want false/true`
- TestFetchDetailPartial/a_later_page_kept_a_GraphQL_error: same
- TestFetchDetailPartial/no_rollup_and_a_kept_GraphQL_error: same
- TestDetailPartialEndToEnd: notice missing
- TestDetailKeptErrorEndToEnd/every_run_read: "read 1 of 1 runs" shown, notice missing
- TestDetailKeptErrorEndToEnd/no_checks: "CHECKS  read 0 of 0 runs" shown, notice missing

TestFetchDetailPartial/a_null_context_node passed before and after (Truncated true, Partial false). The tracer gate (automated verify, end-of-phase mode) was re-run and passed: all 7 top-level and the kept-error subtests pass.

### Task 2 RED evidence (mutation runs)

Task 2 changes no production code, so its tests could not fail before a change. Each was seen failing under at least one named mutation; production files were restored with `git restore` after each run.

- **Mutation A** (delete the Partial note in `detailNotes`): `go test ./internal/ui -count=1 -run 'TestDetailPartialNotice|TestGolden$'` failed:
  TestDetailPartialNotice/every_run_read, /no_checks, /Truncated_and_Partial, /compact_pane; TestGolden/detail-partial_60x15, _80x24, _120x40, _0x0.
- **Mutation B** (note shown whenever `m.detail.data != nil`): same command failed:
  TestDetailPartialNotice/Truncated_only, /neither; and, as expected, the existing detail, detail-expanded, detail-error, detail-gone and detail-scrolled goldens at all four sizes.
- **Mutation C** (the `if partial` branch sets `Truncated` again): `go test ./internal/github ./cmd/prv -count=1 -run 'TestFetchDetailPartial|TestDetailKeptErrorEndToEnd|TestDetailPartialEndToEnd'` failed:
  TestFetchDetailPartial/page_1_kept_a_GraphQL_error, /a_later_page_kept_a_GraphQL_error, /no_rollup_and_a_kept_GraphQL_error; TestDetailPartialEndToEnd; TestDetailKeptErrorEndToEnd/every_run_read, /no_checks.

All 6 TestDetailPartialNotice cases and all 4 detail-partial goldens appear as failing under A or B.

## Files Created/Modified

- `internal/domain/checks.go` - `Detail.Partial` field; doc comment gives Truncated and Partial one meaning each
- `internal/github/checks.go` - FetchDetail sets `Partial` on a kept error; doc comment names both causes; other three `Truncated = true` assignments unchanged
- `internal/ui/detail.go` - `detailNotes` adds the faint notice after "no longer in list" and before the refresh-failed warning; comments updated
- `internal/github/detail_test.go` - TestFetchDetailPartial asserts Truncated and Partial separately; new "no rollup" subtest
- `cmd/prv/main_test.go` - `keptErrorDetailBody`, `keptErrorNoChecksDetailBody`, `TestDetailKeptErrorEndToEnd`; TestDetailPartialEndToEnd also wants the notice
- `internal/ui/detail_test.go` - `TestDetailPartialNotice`
- `internal/ui/view_test.go` - `detail-partial` golden state
- `internal/ui/testdata/TestGolden/detail-partial_{0x0,60x15,80x24,120x40}.golden` - new

## Decisions Made

- `Truncated` means only that check runs were not read; `Partial` means a GraphQL error was kept next to the data. This replaces STATE.md decision "[03-09] Reuse Detail.Truncated for both partial-read causes".
- The notice is its own faint line under the title (in `detailNotes`), not on the CHECKS line. Reasons: the kept error can be about reviewers or review requests, a separate line cannot be cut off at the end of a long CHECKS line in a narrow pane, and "no checks" stays exactly "CHECKS  no checks".
- No `Read < Total` guard was added to `checksSummary`. After this change Truncated is set only when runs were not read. A cut or failure with Read == Total needs GitHub to report hasNextPage true while totalCount equals the runs already read; that GitHub inconsistency is not guarded.
- The kept error message never reaches the screen or a log (T-03-30, unchanged from 03-09).

## Deviations from Plan

None - plan executed exactly as written.

During Task 1 a Python-based text replacement for the `if partial` branch of FetchDetail did not match (a tab-count mismatch in the old comment). The tests still failed with Partial false, which exposed the miss immediately, and the branch was then edited directly. This was a tooling slip and no code was left in a wrong state.

**Total deviations:** 0.

## Issues Encountered

None. Out of scope and untouched, per plan: IN-06, IN-07, IN-01 to IN-05, WR-03. `03-REVIEW-DISPOSITION.md` (WR-05 to fixed) is left for the verify/review step.

## Known Stubs

None. No placeholder data or unwired values were added.

## Threat Flags

None. No new network endpoint, auth path or trust-boundary file access. Threat register: T-03-29 mitigated (Partial flag and separate notice, covered by the tests above), T-03-30 mitigated (fixed notice string, error reduced to a bool), T-03-SC mitigated (`git diff --exit-code go.mod go.sum` exits 0).

## User Setup Required

None - no external service configuration required.

## Verification Results

- `go vet ./... && go test ./... -count=1 -race`: all five packages ok (cmd/prv, config, domain, github, ui).
- `go test ./internal/github ./cmd/prv -count=1 -race -v -run 'TestFetchDetailPartial|TestFetchDetail$|TestFetchDetailPaging|TestFetchDetailNotFound|TestDetailKeptErrorEndToEnd|TestDetailPartialEndToEnd|TestDetailEndToEnd'`: 7 top-level PASS lines, 4 TestFetchDetailPartial subtests, 2 TestDetailKeptErrorEndToEnd subtests.
- `go test ./internal/ui -count=1 -race -v -run 'TestDetailPartialNotice|TestDetailTruncatedSummary|TestDetailEmpty|TestGolden$'`: 6 TestDetailPartialNotice and 4 TestGolden/detail-partial_ subtests pass.
- Acceptance greps: `Partial bool` in domain, `d.Partial = true` in github, 3 non-comment `Truncated = true` in github/checks.go, notice string in detail.go and in main_test.go (2), 4 new goldens each containing the notice, `"detail-partial"` twice in view_test.go, no existing golden changed, `go.mod`/`go.sum` unchanged, production files clean after the mutation runs.

## Next Phase Readiness

G-03-6 is closed in code. The verify/review step should mark WR-05 fixed in `03-REVIEW-DISPOSITION.md` and re-run the UAT item for the detail partial-read text.

## Self-Check: PASSED

- Created files exist: the four `detail-partial_*.golden` files (checked with `ls`, 4 found).
- Commits exist on the branch: `e14b0f7`, `b1fb142` (both between `plan_head_before` and `plan_head_after`; `git rev-list --count` gives 2).

---
*Phase: 03-blocker-detail*
*Completed: 2026-10-13*

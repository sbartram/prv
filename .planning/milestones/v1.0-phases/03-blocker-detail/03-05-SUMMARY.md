---
phase: 03-blocker-detail
plan: 05
subsystem: ui-list
tags: [go, bubbletea-v2, lipgloss-v2, reviewers, merge-state, layout, tdd]

requires:
  - phase: 03-blocker-detail
    provides: "plan 03-03 layoutColumns, dropSteps, compact rows; plan 03-04 PR.Merge and PR.Reviewers"
provides:
  - "MERGE column (✓ mergeable, ⊘ blocked, ↓ behind, ✗ conflicts, – for unknown or draft, blank for merged and closed) with mergeStyle in ANSI-16"
  - "REVIEWERS column (✗bob …carol ✓alice) with reviewersCell: viewer as me, hidden teams as one token, +N for every reviewer not shown, 22 cells at most"
  - "D-13 drop order AUTHOR, STATE, REVIEWERS folded to REVS, REVIEWERS dropped, MERGE, TITLE; layout.collapsed and sizes.collapsed"
  - "Compact rows with review, merge and CI glyphs (> web#7 ✗⊘✗)"
  - "? overlay legend lines for the merge badges and the reviewer glyphs"
affects: [03-06, 03-07, 03-08]

plan_head_before: 98c7b8e667eeb1cc1155cd7bac5f91a562caf387
plan_head_after: e396b5b40d4edbee77e4beadc584bffc42f6889e

actuals:
  tokens: 11000
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Collapse step in the drop table: a dropStep with collapse true swaps a column to its count-only width before a later step drops it"
    - "Cells built twice per row: once uncut to size the column (capped at maxReviewersW), once at the layout width"
    - "No per-token colour inside a cell: the REVIEWERS glyphs carry state, so the selected-row reverse style never wraps nested escape codes"

key-files:
  created: []
  modified:
    - internal/ui/cells.go
    - internal/ui/cells_test.go
    - internal/ui/layout.go
    - internal/ui/layout_test.go
    - internal/ui/view.go
    - internal/ui/view_test.go
    - internal/ui/styles.go
    - internal/ui/help.go
    - internal/ui/testdata/TestGolden/list_0x0.golden
    - internal/ui/testdata/TestGolden/list_60x15.golden
    - internal/ui/testdata/TestGolden/list_80x24.golden
    - internal/ui/testdata/TestGolden/list_120x40.golden
    - internal/ui/testdata/TestGolden/prompt_0x0.golden
    - internal/ui/testdata/TestGolden/prompt_60x15.golden
    - internal/ui/testdata/TestGolden/prompt_80x24.golden
    - internal/ui/testdata/TestGolden/prompt_120x40.golden
    - internal/ui/testdata/TestGoldenCompact.golden
    - internal/ui/testdata/TestGoldenListColor.golden
    - internal/ui/testdata/TestHelpGolden/80x24.golden
    - internal/ui/testdata/TestHelpGolden/120x40.golden

key-decisions:
  - "The reviewers legend in the ? overlay is two lines, not one, because the one-line text is 82 cells and wrapped mid-token ('+N' / 'more') at 80 cells"
  - "renderList builds its cell and header tables as [colCount]string arrays indexed by column constant, so inserting a column constant cannot shift another column's data"
  - "A reviewer name too long for the whole cell is not cut: reviewersCell falls back to '+N', as the plan's algorithm says, so the count is always shown"
  - "REVS collapsed width is the widest '+N' over all rows (at least 4), so the count column is as narrow as the data allows"

patterns-established:
  - "TDD stubs: the RED commit carries compiling stubs (mergeStyle; reviewersCell, reviewersCount, colReviewers hidden in layoutColumns) so the tests fail on assertions"

requirements-completed: [LIST-04, LIST-06, NAV-05]

coverage:
  - id: D1
    description: "Every open row shows whether the PR can merge: ✓ mergeable, ⊘ blocked, ↓ behind, ✗ conflicts, – for unknown or draft; merged and closed rows stay blank; colours are ANSI-16 only"
    requirement: LIST-06
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestMergeColumn"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestMergeColours"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestBadgesReadWithoutColour"
        status: pass
    human_judgment: true
    rationale: "Whether MERGE agrees with the github.com merge box on real PRs, and whether ↓ and ⊘ are one cell wide in the user's font (RESEARCH A5), needs a person in a real terminal"
  - id: D2
    description: "Each row shows who is reviewing: glyph glued to login in changes requested, pending, approved order; the viewer as me; hidden team requests as one token; the rest as +N; an empty cell for a PR with no reviewers"
    requirement: LIST-04
    verification:
      - kind: unit
        ref: "internal/ui/cells_test.go#TestReviewersCell"
        status: pass
      - kind: unit
        ref: "internal/ui/cells_test.go#TestReviewersCount"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestReviewersColumn"
        status: pass
    human_judgment: true
    rationale: "A visible team ('…@org/slug', RESEARCH A1) and the 22-cell cap (A7) could only be checked on fixtures; the plan's human-check on real PRs is still open"
  - id: D3
    description: "Adding MERGE and REVIEWERS never reorders the list: Bucket and Sort read neither field"
    requirement: LIST-06
    verification:
      - kind: unit
        ref: "internal/domain/merge_test.go#TestMergeDoesNotSort"
        status: pass
    human_judgment: false
  - id: D4
    description: "Narrowing drops AUTHOR, then STATE, folds REVIEWERS to a REVS count, drops REVIEWERS, then MERGE, then TITLE; PR, REVIEW and CI stay; compact rows show three glyphs; no width 1..200 or height 1..50 overflows"
    requirement: NAV-05
    verification:
      - kind: unit
        ref: "internal/ui/layout_test.go#TestLayoutColumns"
        status: pass
      - kind: unit
        ref: "internal/ui/layout_test.go#TestDropOrder"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestReviewersCollapse"
        status: pass
      - kind: unit
        ref: "internal/ui/layout_test.go#TestCompactRows"
        status: pass
      - kind: unit
        ref: "internal/ui/layout_test.go#TestNeverOverflows"
        status: pass
    human_judgment: false
  - id: D5
    description: "The ? overlay lists the merge badges and the reviewer glyphs with their words; list, prompt, colour, compact and help goldens show both columns"
    requirement: LIST-04
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestHelpReviewersLegend"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestGolden"
        status: pass
      - kind: unit
        ref: "internal/ui/help_test.go#TestHelpGolden"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-10-12
status: complete
---

# Phase 3 Plan 05: Each row shows who is reviewing and whether the PR can merge Summary

**MERGE and REVIEWERS columns in the PR list (`✗bob …carol ✓alice`, `⊘ blocked`), folding REVIEWERS to a REVS count before dropping it in the D-13 order, with 22-cell cap, hidden-team token and `+N` for anyone not shown**

## Performance

- **Duration:** about 8 min
- **Started:** 2026-10-12T23:17Z (approximate; the start time was not captured)
- **Completed:** 2026-10-12T23:25Z
- **Tasks:** 2 (each as a RED and a GREEN commit)
- **Files modified:** 20 (8 Go files, 12 golden files)

## Accomplishments

- MERGE sits between REVIEW and CI: mergeable green, blocked and behind yellow, conflicts red, a dash faint for unknown and draft, blank for merged and closed. It drops after STATE and before TITLE, and compact rows carry its glyph (`> web#7 ––●`).
- REVIEWERS sits between REVIEW and MERGE. `reviewersCell` writes one token per reviewer (`✗bob …carol …2 teams ✓alice ✓me`), measured in terminal cells. When the cell is full it keeps the longest run of tokens that fits with ` +K`, so no reviewer is hidden without a count; with no room it shows `+N` for all of them.
- The layout gained a collapse step. At the width where REVIEWERS no longer fits, the column becomes `REVS` with a `+N` per row; one step narrower it drops. `TestDropOrder` sweeps 120 down to 40 cells and checks the whole D-13 sequence, including that `REVIEWERS` and `REVS` both appear.
- The `?` overlay lists the merge badges and the reviewer glyphs. All twelve affected golden files were regenerated with `-update`, reviewed line by line, and no other golden changed.

## Task Commits

1. **Task 1: MERGE column**
   - RED `b42fc17` (test): TestMergeColumn, TestMergeColours, TestDropOrder, TestCompactRows, TestBadgesReadWithoutColour, TestTitleTruncatesFirst failed on their assertions
   - GREEN `f29919d` (feat): colMerge, mergeStyle, compact merge glyph, help line, goldens
2. **Task 2: REVIEWERS column**
   - RED `b98924e` (test): TestReviewersCell, TestReviewersCount, TestReviewersColumn, TestReviewersCollapse, TestHelpReviewersLegend, TestLayoutColumns, TestDropOrder, TestTitleTruncatesFirst failed on their assertions
   - GREEN `e396b5b` (feat): reviewersCell, reviewersCount, colReviewers, collapse step, help lines, goldens

**Plan metadata:** the commit that adds this file (docs: complete plan).

## TDD Gate Compliance

Both tasks have a `test(03-05)` commit before the `feat(03-05)` commit. No REFACTOR commit was needed.

- **RED evidence, Task 1:** `go test -run 'TestMergeColumn|TestMergeColours|...'` exited non-zero. TestMergeColumn failed because the header lacked MERGE; TestMergeColours on `mergeStyle` foreground; TestDropOrder with `MERGE dropped while AUTHOR is still shown`; TestCompactRows with `row of api#102 = "  api#102 ✗✗", want "  api#102 ✗⊘✗"`. Semantic assessment: each failure is on the planned assertion; the `mergeStyle` stub only let the package compile.
- **RED evidence, Task 2:** the stubs returned `""` and hid `colReviewers` in `layoutColumns`, so the tests built and ran. TestReviewersCell failed with `reviewersCell(w=0) = "", want "✗bob …carol …2 teams ✓alice ✓me"`; TestLayoutColumns, TestDropOrder and TestTitleTruncatesFirst on their expected column sets; TestReviewersCollapse with `full=false REVS=false`. Semantic assessment: all on the planned assertions. TestTruncateBoundary also failed in RED, because its sizes now include the REVIEWERS column that the stub hid; that is a planned-behaviour assertion too.
- `gsd_run check tdd-red-evidence` was not run: the plan type is `execute`, not `tdd`, and Go's `go test` output is not a report format that classifier accepts.
- A first draft of TestReviewersColumn panicked (`slice bounds out of range [:-1]`) on the stub, because it sliced the header at a missing `REVIEWERS`. I fixed the test to fail with a message before the RED commit; this was a test bug, not a RED result.

## Files Created/Modified

- `internal/ui/cells.go` - `reviewersCell`, `reviewersCount`
- `internal/ui/layout.go` - `colReviewers`, `colMerge`, `maxReviewersW`, `sizes.collapsed`, `layout.collapsed`, `dropSteps` with the collapse step, collapse handling in `layoutColumns`
- `internal/ui/view.go` - header and cells for both columns, REVS header, compact merge glyph, `[colCount]string` tables
- `internal/ui/styles.go` - `mergeStyle`
- `internal/ui/help.go` - merge line and two reviewers lines
- `internal/ui/cells_test.go`, `layout_test.go`, `view_test.go` - tests listed in the plan, plus TestReviewersCount, TestReviewersColumn, TestReviewersCollapse, TestHelpReviewersLegend
- `internal/ui/testdata/**` - regenerated goldens, reviewed

## Decisions Made

See `key-decisions` above. The plan was followed; the decisions fill gaps it left open (help line length, array-indexed tables, long single names, REVS width).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Existing tests indexed columns by position**
- **Found during:** Task 1 (GREEN) and Task 2 (RED)
- **Issue:** `TestTruncateBoundary` built its `sizes` from 7-element rows indexed by position. Inserting `colMerge` and `colReviewers` shifted CI onto the wrong column. Its "TITLE floor of 10" assertion also moved: with MERGE and REVIEWERS present, AUTHOR, STATE, REVIEWERS and MERGE are all gone only at 44 cells, not 50.
- **Fix:** the rows are now `[colCount]string` with the two new cells; the floor assertion uses 44 cells. `testSizes()` in layout_test.go gained MERGE, REVIEWERS and a collapsed width, and every width in `TestLayoutColumns` was recomputed (shown in the test table).
- **Files modified:** internal/ui/view_test.go, internal/ui/layout_test.go
- **Committed in:** f29919d, b98924e

**2. [Rule 1 - Layout] Reviewers legend split over two lines**
- **Found during:** Task 2 (GREEN, golden review)
- **Issue:** the one-line legend from the plan is 82 cells; at 80 cells it wrapped between `+N` and `more`.
- **Fix:** two lines, with `…team name hidden  +N more` indented under the first.
- **Files modified:** internal/ui/help.go
- **Committed in:** e396b5b

**3. [Rule 1 - Bug in own test] Compact-row test widths**
- **Found during:** Task 1 (RED)
- **Issue:** with a third glyph, the old cut-ref widths (12, 11, 10, 8, 7) no longer all leave room for a left-cut ref with `…` and `#1234`.
- **Fix:** widths 13, 12, 10, 9, 8; `#1234` is required from 12.
- **Files modified:** internal/ui/layout_test.go
- **Committed in:** b42fc17

---

**Total deviations:** 3 auto-fixed (1 blocking, 2 own-test or layout corrections)
**Impact on plan:** no change to behaviour or scope.

## Issues Encountered

- At 60x15 the list golden did not change in Task 2: there REVIEWERS cannot fit even as `REVS` (62 cells needed with a 10-cell TITLE), so it is already dropped and the output equals Task 1's. The `REVS` stage shows in `prompt_60x15` (one filtered row, narrower columns) and in `TestReviewersCollapse`.

## Known Stubs

None. The RED-stage stubs (`mergeStyle`, `reviewersCell`, `reviewersCount`, the `colReviewers` hide line) were replaced in the GREEN commits.

## Threat Flags

None. T-03-19 is mitigated by the 22-cell cap, the collapse and drop steps, `lipgloss.Width` measurement, TestReviewersCell and TestNeverOverflows. T-03-20 by glyph-plus-word values, no colour inside the REVIEWERS cell, TestBadgesReadWithoutColour and TestMergeColours. T-03-SC: `git diff --exit-code go.mod go.sum` exits 0.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 03-06 to 03-08 can reuse `reviewersCell`'s token rules for the detail view; `dropSteps` now supports collapse steps.
- Open for a person (the plan's human-check): run prv on real PRs (or the throwaway sbartram/prv#1) and compare REVIEWERS and MERGE with the github.com merge box. Carried assumptions: A1 (team request shows `…@org/slug`; a user outside the org sees `…team`), A5 (`↓` and `⊘` one cell wide; fallback `⇣`), A7 (the 22-cell cap).

## Self-Check: PASSED

- Files: cells.go, layout.go, view.go, styles.go, help.go and the three test files exist and hold the planned identifiers (`func reviewersCell`, `maxReviewersW = 22`, `REVS`, `func mergeStyle`, `colMerge`, `Merge.Glyph()`).
- Commits `b42fc17`, `f29919d`, `b98924e`, `e396b5b` are ancestors of HEAD.
- `go vet ./...` and `go test ./... -count=1 -race` pass; `git diff --exit-code go.mod go.sum` exits 0; the golden-hygiene `git status` check prints nothing; no LightDark, AdaptiveColor or hex colour in non-test UI code.

---
*Phase: 03-blocker-detail*
*Completed: 2026-10-12*

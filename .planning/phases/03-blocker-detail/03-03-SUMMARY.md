---
phase: 03-blocker-detail
plan: 03
subsystem: ui-layout
tags: [go, bubbletea-v2, lipgloss-v2, layout, narrow-panes, tdd]

requires:
  - phase: 03-blocker-detail
    provides: "plan 03-01 ciCell and the column constants; plan 03-02 suffix-preserving ciCell cut"
provides:
  - "Pure layoutColumns with the D-13 drop table (AUTHOR, STATE, TITLE) and the D-14 compact threshold (40 cells)"
  - "trimLeft: PR ref cut from the left so its '#number' stays"
  - "Compact rows ('> web#7 ✗✗') with a 'widen for columns' hint on the column-header line"
  - "Review.Glyph() and CI.Glyph(), pinned to the first glyph of Badge()"
  - "Small-height rule: column-header line, then footer, then header line are dropped, so the selected row is always on screen"
  - "60x15 goldens for all six screens plus one compact golden"
affects: [03-05, 03-07, 03-08]

plan_head_before: 99ef1d8c77b0f853ca920b6fbdeb55e011f40dff
plan_head_after: 7a5d1df1361b5fbb8100c71a633170860bcf2274

actuals:
  tokens: 8000
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Data-driven column layout: sizes{natural,min} in, layout{compact,show,width} out; dropSteps is the single list plan 03-05 extends"
    - "Compact row styled segment by segment (ref, review glyph, CI glyph), each segment inheriting selectedStyle, so no style wraps text that holds escape codes"
    - "stack() composes header, column header, rows and footer and drops lines for tiny heights"

key-files:
  created:
    - internal/ui/layout_test.go
    - internal/ui/testdata/TestGolden/loading_60x15.golden
    - internal/ui/testdata/TestGolden/error_60x15.golden
    - internal/ui/testdata/TestGolden/empty_60x15.golden
    - internal/ui/testdata/TestGolden/list_60x15.golden
    - internal/ui/testdata/TestGolden/filtered_60x15.golden
    - internal/ui/testdata/TestGolden/prompt_60x15.golden
    - internal/ui/testdata/TestGoldenCompact.golden
  modified:
    - internal/ui/layout.go
    - internal/ui/view.go
    - internal/ui/view_test.go
    - internal/domain/status.go
    - internal/domain/domain_test.go

key-decisions:
  - "The compact hint sits on the column-header line ('  PR  widen for columns'), not in the header line, so the header segments that filter_test.go asserts at 36 cells do not move"
  - "A compact row whose ref gets under 1 cell drops the ref and draws only the marker and the glyphs, so REVIEW and CI state stay visible down to 4 cells"
  - "columnWidths was removed; layoutColumns reproduces its widths at every width where TITLE fits, so the existing 80x24, 120x40 and 0x0 goldens are byte-identical"
  - "Narrow-width cases of TestLayoutColumns use 40 cells and up, because layoutColumns now returns compact below 40"

patterns-established:
  - "Property test with recover per case: TestNeverOverflows fails a case with its size in the message instead of aborting the run"
  - "Golden hygiene for new sizes: -update, then git status shows only new files and no modified golden"

requirements-completed: [NAV-05]

coverage:
  - id: D1
    description: "As the pane narrows, TITLE shrinks to 10 cells, then AUTHOR, STATE and TITLE drop in that order; PR, REVIEW and CI stay at every width of 40 cells or more, and columns never come back at a smaller width"
    requirement: NAV-05
    verification:
      - kind: unit
        ref: "internal/ui/layout_test.go#TestLayoutColumns"
        status: pass
      - kind: unit
        ref: "internal/ui/layout_test.go#TestDropOrder"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestTitleTruncatesFirst"
        status: pass
    human_judgment: false
  - id: D2
    description: "Below 40 cells the list shows compact rows with the short ref and the REVIEW and CI glyphs, a 'widen for columns' hint, and j/k, o, 1/2/3 and / keep working"
    requirement: NAV-05
    verification:
      - kind: unit
        ref: "internal/ui/layout_test.go#TestCompactRows"
        status: pass
      - kind: unit
        ref: "internal/ui/layout_test.go#TestCompactHeaderHint"
        status: pass
      - kind: unit
        ref: "internal/ui/layout_test.go#TestCompactNavigation"
        status: pass
      - kind: unit
        ref: "internal/domain/domain_test.go#TestGlyphPrefix"
        status: pass
    human_judgment: true
    rationale: "The plan's human-check is dragging a real terminal pane from wide to about 30 columns and back; tests cover the rendered rows but not how the drag feels"
  - id: D3
    description: "After any resize the selected PR stays selected and on screen; below 4 lines the list drops the column-header line, the footer, then the header line"
    requirement: NAV-05
    verification:
      - kind: unit
        ref: "internal/ui/layout_test.go#TestResizeKeepsSelection"
        status: pass
      - kind: unit
        ref: "internal/ui/layout_test.go#TestTinyHeights"
        status: pass
    human_judgment: false
  - id: D4
    description: "No width 1..200 at heights 3, 15 and 50, and no height 1..50 at widths 30, 60 and 120, makes the list panic or draw a line wider or more lines than the pane, prompt closed or open"
    requirement: NAV-05
    verification:
      - kind: unit
        ref: "internal/ui/layout_test.go#TestNeverOverflows"
        status: pass
    human_judgment: false
  - id: D5
    description: "Golden files for the six screens at 60x15 and one compact golden; the 80x24, 120x40 and 0x0 files do not change"
    requirement: NAV-05
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestGolden"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestGoldenCompact"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestViewFitsScreen"
        status: pass
    human_judgment: false

duration: 5min
completed: 2026-10-12
status: complete
---

# Phase 3 Plan 03: The list stays readable in a narrow pane and after a resize Summary

**Pure layoutColumns drops AUTHOR, STATE and TITLE in the D-13 order, then compact `> web#7 ✗✗` rows below 40 cells, with a property test over 1500 sizes proving no overflow**

## Performance

- **Duration:** 5 min
- **Started:** 2026-10-12T23:06:03Z
- **Completed:** 2026-10-12T23:10:49Z
- **Tasks:** 2 (each as a RED and a GREEN commit)
- **Files modified:** 13 (5 Go files, 8 golden files)

## Accomplishments
- `layoutColumns(sizes, total)` replaces `columnWidths`: TITLE shrinks to 10 cells, then AUTHOR, STATE and TITLE drop; when PR, REVIEW and CI still do not fit, CI falls back to its bare badge and the PR ref is cut from the left with `…` so its number stays. A pane of 0x0 shows every column uncut and is never compact.
- Below 40 cells the list shows compact rows (marker, short ref, review glyph, CI glyph) with `  PR  widen for columns` on the column-header line. Selected, dimmed and coloured styling is applied per segment.
- `stack()` handles tiny heights: below 4 lines it drops the column-header line, then the footer, then the header line, so the selected row always stays.
- `TestNeverOverflows` renders the list for widths 1..200 at heights 3, 15, 50 and heights 1..50 at widths 30, 60, 120, prompt closed and open: no panic, no wide line, no extra line.
- Goldens: six new 60x15 files and `TestGoldenCompact.golden`. The existing 80x24, 120x40 and 0x0 files are byte-identical.

## Task Commits

1. **Task 1: narrow pane drops columns in the D-13 order**
   - RED `8bf89a7` (test) - TestLayoutColumns, TestDropOrder, TestTrimLeft fail on assertions against compiling stubs
   - GREEN `aa4747e` (feat) - layoutColumns, trimLeft, renderList on shown columns, 60x15 goldens
2. **Task 2: compact rows, tiny heights, no overflow**
   - RED `b466c5a` (test) - TestGlyphPrefix, TestLayoutCompact, TestCompactRows, TestCompactHeaderHint, TestCompactNavigation, TestTinyHeights, TestNeverOverflows fail on assertions
   - GREEN `7a5d1df` (feat) - Glyph methods, compact threshold, renderCompact, stack, compact golden

**Plan metadata:** committed with this SUMMARY (docs: complete plan)

## TDD Gate Compliance

Both tasks have a `test(03-03)` commit before the `feat(03-03)` commit. No REFACTOR commit was needed.

- **RED evidence, Task 1:** `go test -run 'TestLayoutColumns|TestDropOrder|TestTrimLeft'` exited non-zero; every TestLayoutColumns case failed on its shown-columns or width assertion, TestDropOrder failed with `w=52: column header ... lacks CI` (the old layout cut the CI header), and TestTrimLeft failed because the stub returned the string uncut. Semantic assessment: each failure is on the planned assertion for the intended reason, not a load or syntax fault.
- **RED evidence, Task 2:** TestGlyphPrefix failed with `Review(1).Glyph() = "", want "✓"`; TestCompactRows with `row of api#101 = "> …me/api#101 ✓ approved ✓ passing  ", want "> api#101 ✓✓"`; TestTinyHeights and TestNeverOverflows with `4 lines, want at most 3`; TestGoldenCompact with a missing golden file. Semantic assessment: all on the planned assertions.
- `TestResizeKeepsSelection` already passed at RED, since selection and scrolling logic existed before this plan. It stays as a regression guard for the compact path.
- `gsd_run check tdd-red-evidence` was not run: the plan type is `execute`, not `tdd`, and Go's `go test` output is not one of the report formats that classifier accepts.

## Files Created/Modified
- `internal/ui/layout.go` - `sizes`, `layout`, `dropStep`, `dropSteps`, `layoutColumns`, `trimLeft`, `minPRColW`, `compactBelow`; `columnWidths` removed
- `internal/ui/view.go` - `renderList` draws only shown columns; `stack`, `shortRef`, `renderCompact`, `compactRow`
- `internal/domain/status.go` - `Review.Glyph()` and `CI.Glyph()`
- `internal/ui/layout_test.go` - new: TestLayoutColumns, TestDropOrder, TestTrimLeft, TestLayoutCompact, TestCompactRows, TestCompactHeaderHint, TestCompactNavigation, TestResizeKeepsSelection, TestTinyHeights, TestNeverOverflows
- `internal/ui/view_test.go` - 60x15 added to TestGolden and TestViewFitsScreen; TestGoldenCompact; TestTruncateBoundary ported to layoutColumns
- `internal/domain/domain_test.go` - TestGlyphPrefix
- `internal/ui/testdata/TestGolden/*_60x15.golden`, `internal/ui/testdata/TestGoldenCompact.golden` - regenerated output, reviewed

## Decisions Made
- The "widen for columns" hint is on the column-header line, as the plan's flagged assumption said; the header line is unchanged.
- A compact row with no room for a ref cell shows the marker and both glyphs only, so approved/blocked/failing state stays visible down to 4 cells.
- The legend for the glyphs stays in the `?` overlay (plan 03-05 adds the merge glyph).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] RED commits carry compiling stubs**
- **Found during:** Task 1 and Task 2 (RED)
- **Issue:** Go cannot run a test that refers to an undefined function; a build failure is not a valid RED.
- **Fix:** The RED commits include stubs (`layoutColumns`, `trimLeft`, `Review.Glyph`, `CI.Glyph`) so the tests compile and fail on assertions. The GREEN commits replace the stubs.
- **Files modified:** internal/ui/layout.go, internal/domain/status.go
- **Committed in:** 8bf89a7, b466c5a

**2. [Rule 3 - Blocking] TestTruncateBoundary called the removed columnWidths**
- **Found during:** Task 1 (GREEN)
- **Issue:** The plan removes `columnWidths`, but `TestTruncateBoundary` in view_test.go called it. Its "TITLE floor of 10 at 30 cells" assertion no longer holds, because a 30-cell pane is now compact and a narrow pane drops TITLE instead of keeping a stub.
- **Fix:** The test now builds `sizes` from its cells and calls `layoutColumns`: the 80-cell sum, the cut boundary and the unsized 200-cell title are unchanged; the floor assertion moved to 50 cells, where AUTHOR is gone and TITLE sits on 10.
- **Files modified:** internal/ui/view_test.go
- **Committed in:** aa4747e

**3. [Rule 1 - Bug in own test] TestLayoutColumns cases below 40 cells**
- **Found during:** Task 2 (GREEN)
- **Issue:** Two Task 1 table cases used totals 35 and 34, which Task 2's compact threshold now routes to compact rows.
- **Fix:** Both cases use a wider PR column (30 cells) at totals 53 and 40, so they still test the exact-fit and PR-narrowing paths at 40 cells and up.
- **Files modified:** internal/ui/layout_test.go
- **Committed in:** 7a5d1df

---

**Total deviations:** 3 auto-fixed (2 blocking, 1 own-test correction)
**Impact on plan:** No change to behavior or scope; the first is a Go constraint on TDD, the others follow from the plan's own removal and threshold.

## Issues Encountered
- `TestNeverOverflows` takes about 11 s under `-race` (1500 renders); about 1 s without it. Left as is.

## Known Stubs

None. The RED-stage stubs were replaced in the GREEN commits.

## Threat Flags

None. The plan's threats are mitigated as listed: T-03-06 by floors in `layoutColumns`, a final `m.cut` per line and `TestNeverOverflows`; T-03-07 by the D-13 order and `TestDropOrder`/`TestCompactRows`; T-03-SC by `git diff --exit-code go.mod go.sum` (clean).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `dropSteps` and `sizes` are ready for plan 03-05 to insert REVIEWERS and MERGE before `colTitle`.
- The detail view (plans 03-07 and 03-08) can reuse `trimLeft` and the `stack` height rule pattern.
- Open item for the verifier: the plan's human-check (drag a real pane from wide to about 30 columns and back) has not been done by a person.

## Self-Check: PASSED

- Files: layout.go, layout_test.go, view.go, status.go, TestGoldenCompact.golden and the six `*_60x15.golden` files exist.
- Commits `8bf89a7`, `aa4747e`, `b466c5a`, `7a5d1df` are ancestors of HEAD.
- `go vet ./...` and `go test ./... -count=1 -race` pass; `git diff --exit-code go.mod go.sum` is clean; no existing golden changed.

---
*Phase: 03-blocker-detail*
*Completed: 2026-10-12*

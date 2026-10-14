---
phase: 01-status-at-a-glance
plan: 03
subsystem: ui
tags: [go, bubbletea-v2, lipgloss-v2, layout, golden-tests]

requires:
  - phase: 01-status-at-a-glance
    provides: "01-02 trustworthy list and honest states: sorted, windowed PR list; loading, error and empty screens"
provides:
  - "ANSI-16 badge styles (green, red, yellow, faint), dim rule, header and selection styles"
  - "columnWidths: explicit cell-width columns with TITLE as the flexible column (floor 10), truncateCell via ansi.Truncate"
  - "Cursor-following scroll window (Model.offset, visibleRows) with no table Height or YOffset"
  - "Word-wrapped message screens and width-cut list lines so nothing exceeds the pane"
  - "Golden snapshots: loading, error, empty, list at 80x24, 120x40, 0x0 plus a raw coloured list at 120x40"
affects: [01-04, phase-02, phase-03]

actuals:
  tokens: 6126
  tasks: 2
  commits: 4

plan_head_before: c77f8b96f2e253f43542bbc64be32ab2cab348c0
plan_head_after: 95231fcf23eda79bfc6d5f744849ae15796362a0
commits: 4

tech-stack:
  added:
    - "github.com/charmbracelet/x/exp/golden v0.1.0 (test-only; brings github.com/aymanbagabas/go-udiff v0.4.1)"
  patterns:
    - "Style functions return lipgloss.Style built from lipgloss.Green/Red/Yellow plus Faint, Bold, Reverse; Style.Inherit layers dim and selection on top of the badge colour"
    - "Column widths are computed over every row (header and all PRs) so columns do not jump while scrolling; cells are pre-truncated, then the table is given Width per column"
    - "Only the visible slice of rows goes to the lipgloss table; StyleFunc maps the table row back with offset + row"
    - "Message screens are wrapped with lipgloss Width then right-trimmed so goldens carry no trailing padding"

key-files:
  created:
    - internal/ui/styles.go
    - internal/ui/layout.go
    - internal/ui/view_test.go
    - internal/ui/testdata/TestGoldenListColor.golden
    - internal/ui/testdata/TestGolden/ (12 golden files)
  modified:
    - internal/ui/view.go
    - internal/ui/model.go
    - go.mod
    - go.sum

key-decisions:
  - "TITLE width is max(10, min(widest title cell, space left)); the floor is applied last, so a pane too narrow for the fixed columns keeps TITLE at 10 and the line is cut at the right edge (NAV-05 owns column dropping)"
  - "The blank lines between the header, table and footer were removed so the view is header + column-header row + rows + footer, matching the plan's h minus 3 row budget"
  - "Offset is also clamped to len(PRs) minus visible rows, so enlarging the window never leaves blank space under the list"

patterns-established:
  - "RED commits compile against stubs and fail on assertions; golden files are generated in the GREEN commit, so the golden subtests are not part of the RED evidence"

requirements-completed: [LIST-01, LIST-02, LIST-03, LIST-05, NAV-01]

coverage:
  - id: D1
    description: "Review and CI badges use the terminal's ANSI-16 palette: approved and passing green, changes and failing red, required and running yellow, none, no checks and unknown faint; the coloured golden holds no 24-bit sequences"
    requirement: "LIST-03"
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestReviewAndCIColours"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestStylesUseAnsi16"
        status: pass
      - kind: golden
        ref: "internal/ui/testdata/TestGoldenListColor.golden"
        status: pass
    human_judgment: true
    rationale: "Readability on a light and a dark theme and inside the multiplexer pane needs the end-of-phase human check."
  - id: D2
    description: "Every badge and the [draft] tag read correctly with colour stripped; a draft keeps STATE open"
    requirement: "LIST-03"
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestBadgesReadWithoutColour"
        status: pass
    human_judgment: false
  - id: D3
    description: "Drafts, merged and closed rows are dimmed as a whole; open non-draft and unknown-state rows are not; no section headers or dividers"
    requirement: "LIST-05"
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestDimmedRule"
        status: pass
      - kind: golden
        ref: "internal/ui/testdata/TestGoldenListColor.golden"
        status: pass
    human_judgment: false
  - id: D4
    description: "The selected row shows > and reverse video, never colour alone; exactly one row is marked and j moves the marker"
    requirement: "NAV-01"
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestSelectionNotColourOnly"
        status: pass
    human_judgment: false
  - id: D5
    description: "At 80 and 120 columns every column header is whole and only TITLE is cut, ending in …; a title exactly as wide as the column is not cut, one cell wider is; TITLE never gets fewer than 10 cells"
    requirement: "LIST-02"
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestTitleTruncatesFirst"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestTruncateBoundary"
        status: pass
    human_judgment: false
  - id: D6
    description: "Widths are in terminal cells: a 12-character CJK title takes 24 cells and is cut on a character boundary; no line is wider than the pane, including a ZWJ emoji title"
    requirement: "LIST-02"
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestCellWidth"
        status: pass
    human_judgment: true
    rationale: "Ambiguous-width glyphs (the badge glyphs) may draw two cells wide in a CJK-locale terminal; this cannot be tested in code (RESEARCH A7)."
  - id: D7
    description: "With more PRs than fit, moving the cursor past the last visible row scrolls so the selected row stays visible, in displayed order; moving back scrolls back"
    requirement: "NAV-01"
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestScrollFollowsCursor"
        status: pass
    human_judgment: false
  - id: D8
    description: "At 80x24 and 120x40 no line is wider than the terminal and the view has at most h lines in every state; at 0x0 the view renders without panicking"
    requirement: "LIST-01"
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestViewFitsScreen"
        status: pass
    human_judgment: false
  - id: D9
    description: "Golden snapshots lock the loading, error, empty and list screens at 80x24, 120x40 and 0x0"
    requirement: "LIST-01"
    verification:
      - kind: golden
        ref: "internal/ui/view_test.go#TestGolden (12 subtests)"
        status: pass
    human_judgment: false
  - id: D10
    description: "Real data at --closed-days 300 in a real terminal pane: dimmed merged and closed rows, colours on light and dark themes, 80-column pane, navigation, terminal restored after q, empty-state sentence"
    requirement: "LIST-03"
    verification: []
    human_judgment: true
    rationale: "Plan verify block human-check, steps 1 to 6; recorded for end-of-phase UAT and not run by this executor."

duration: 12min
completed: 2026-10-12
status: complete
---

# Phase 1 Plan 3: Readable Rows Summary

**The PR list now reads at a glance: ANSI-16 green/red/yellow/faint badges that also read without colour, dimmed draft/merged/closed rows, a `>` plus reverse-video selection, cell-width column layout where only TITLE is cut with `…`, cursor-following scrolling, and 13 golden files locking every screen at 80x24, 120x40 and 0x0**

## Performance

- **Duration:** 12 min (2026-10-12T00:32Z to 00:43Z, after reading plan and context)
- **Tasks:** 2 (both TDD, RED then GREEN)
- **Files:** 4 source files created or rewritten in `internal/ui`, 13 golden files, `go.mod` and `go.sum`

## Accomplishments

- Colour (Task 1): `reviewStyle` and `ciStyle` use `lipgloss.Green`, `Red` and `Yellow` with `Faint` for none, no checks and unknown. `dimmed` is true for drafts, merged and closed PRs. The table `StyleFunc` makes a dimmed row Faint in every cell while REVIEW and CI keep their colour on top (`Faint;Green` renders as `2;32`). The cursor row adds Reverse and keeps `>`. No 24-bit, 256-colour, hex, `LightDark` or `AdaptiveColor` constructs exist in `internal/ui`.
- Layout (Task 2): `columnWidths` measures with `lipgloss.Width` over the header and every row; PR is capped at 40, AUTHOR at 16, TITLE takes the rest with a floor of 10 and a ceiling of its widest cell. `truncateCell` uses `ansi.Truncate(s, w, "…")` only when the cell is wider. A 12-character CJK title measures 24 cells and is cut on a character boundary.
- Scrolling: `Model.offset` and `visibleRows()` (h minus 3) with `clampOffset` after every cursor move, size message and new data. Only the visible slice goes to the table; the table's own `Height` and `YOffset` are not used.
- Fit: list lines are cut to the pane width, and loading, error and empty screens are word-wrapped to it. Before the first size message nothing is wrapped or cut.
- Tests: 5 task 1 tests, 5 task 2 tests, `TestGolden` with 12 subtests, and `TestReviewAndCIColours`. `go vet ./...`, `go test ./... -count=1` and `-race` pass.

## Task Commits

1. **Task 1 (TDD): status reads at a glance**
   - RED: `d637005` (test) - stub styles, colour, dim, selection and ANSI-16 tests, golden module added
   - GREEN: `b9883d5` (feat) - styles, `StyleFunc`, coloured golden
2. **Task 2 (TDD): the list fits the pane**
   - RED: `0dd93f4` (test) - layout stubs, `offset` field, truncation, width, scroll and fit tests
   - GREEN: `95231fc` (feat) - layout, scrolling, wrapping, 12 goldens, regenerated colour golden

**Plan metadata:** committed separately as `docs(01-03): complete readable rows plan`.

## TDD Gate Compliance

- **RED (Task 1):** `d637005`. Everything compiled against stub styles. `TestReviewAndCIColours`, `TestDimmedRule` (draft, merged, closed), `TestSelectionNotColourOnly` and `TestStylesUseAnsi16` failed on assertions. `TestBadgesReadWithoutColour` already passed because the 01-02 view already printed the badge text; it stays as a guard. `TestGoldenListColor` was not part of RED (no golden file until GREEN).
- **RED (Task 2):** `0dd93f4`. `TestTitleTruncatesFirst`, `TestTruncateBoundary` (TITLE width 0), `TestCellWidth` (uncut CJK title, 182-cell lines), `TestScrollFollowsCursor` and `TestViewFitsScreen` (error, empty and list subtests) failed on assertions. `TestGolden` was not part of RED for the same reason as above.
- **Semantic assessment:** each target test failed for the intended reason (behaviour absent), not on a build or fixture error. `gsd_run check tdd-red-evidence` does not parse Go test output, so there is no machine-classifier record. The plan is `type: execute` with two `tdd="true"` tasks, so the plan-level gate does not apply.
- **GREEN:** `b9883d5` and `95231fc`. All listed tests pass.
- **REFACTOR:** none needed.

## Files Created/Modified

- `internal/ui/styles.go` - ANSI-16 badge styles, `dimmed`, `headerStyle`, `selectedStyle`, `faintStyle`
- `internal/ui/layout.go` - column constants, `columnWidths`, `truncateCell`, `visibleRows`, `clampOffset`
- `internal/ui/view.go` - width-aware `renderList` with the table `StyleFunc`, `wrap` and `cut` helpers
- `internal/ui/model.go` - `offset` field and clamping in `Update`
- `internal/ui/view_test.go` - fixtures `fixtureSix` and `fixtureThirty`, helpers, 11 test functions
- `internal/ui/testdata/TestGoldenListColor.golden`, `internal/ui/testdata/TestGolden/*.golden` (12)
- `go.mod`, `go.sum` - `github.com/charmbracelet/x/exp/golden v0.1.0` (still `go 1.26.0`, no toolchain line)

## Decisions Made

- TITLE width is `max(10, min(widest title, space left))`. The floor is applied last, so in a pane narrower than the fixed columns TITLE keeps 10 cells and the line is cut at the right edge. Column dropping is NAV-05, Phase 3.
- The two blank lines around the table were removed, so the view is the `prv · @login · N PRs` line, the column-header row, the rows and the footer. This matches the plan's "h minus 3" row budget.
- `clampOffset` also caps the offset at `len(PRs) - visibleRows()`, so enlarging the window never leaves empty space below the list.

## Deviations from Plan

### Auto-fixed Issues

None - plan executed as written.

Two small choices the plan left open, not deviations: the layout stubs and the `offset` field were added in the Task 2 RED commit so the tests compile, and golden files were generated in the GREEN commits.

## Issues Encountered

- A BSD `sed -i` call used to add test imports failed; the imports were added with a short Python edit instead. No code was affected.
- The Task 1 coloured golden had to be regenerated in Task 2 because the layout changed (no blank lines, one-cell gaps). It was re-checked for `38;2;` and `48;2;` sequences and has none.

## Authentication Gates

None.

## Known Stubs

None. `layout.go` stubs existed only in the Task 2 RED commit.

## Threat Flags

None. All new surface is in the plan's threat register: T-01-14 (cell widths and `TestCellWidth`, `TestViewFitsScreen`), T-01-15 (glyph plus word badges, `>` marker, `TestBadgesReadWithoutColour`, `TestSelectionNotColourOnly`), T-01-SC (golden v0.1.0 pinned, go-udiff v0.4.1 verified by `go.sum`, test-only).

## User Setup Required

None. The end-of-phase human check from the plan's verify block is recorded for UAT: build with `go build -o /tmp/prv ./cmd/prv`, then (1) `/tmp/prv --closed-days 300` lists real merged PRs with merged and closed rows dimmed and badges readable; (2) colours read on a light and a dark theme and inside the multiplexer pane; (3) at about 80 columns every column header stays whole and only TITLE is cut with `…`; (4) j, k and the arrow keys move `>` and it stays visible; (5) q returns to the shell with the screen restored; (6) `/tmp/prv` with the default window shows the empty-state sentence naming the account and 14 days.

## Next Phase Readiness

- 01-04 (and later phases) can rely on the table layout, `offset` scrolling and the golden harness (`stateModel`, `newTestModel`, `fixtureSix`, `fixtureThirty`).
- Open items carried forward: RESEARCH A7 (ambiguous-width badge glyphs in CJK-locale terminals) is untestable in code; panes narrower than the fixed columns plus 10 cut lines at the right edge until NAV-05; the human check above has not been run.

## Self-Check: PASSED

- Created files exist: `internal/ui/styles.go`, `internal/ui/layout.go`, `internal/ui/view_test.go`, `internal/ui/testdata/TestGoldenListColor.golden`, and 12 files under `internal/ui/testdata/TestGolden/`.
- Commits `d637005`, `b9883d5`, `0dd93f4`, `95231fc` are ancestors of HEAD; RED precedes GREEN in both tasks.
- `go vet ./...`, `go test ./... -count=1` and `go test ./... -race -count=1` pass; `gofmt -l .` is empty.
- Acceptance greps: `lipgloss.Green`, `Red`, `Yellow`, `Faint(true)`, `Reverse(true)` present in `styles.go`; no `LightDark`, `AdaptiveColor` or hex colour in `internal/ui`; no `38;2;` or `48;2;` in the colour golden; `go 1.26.0` with no toolchain line; `ansi.Truncate` in `layout.go`; no `len(...)` width maths and no table `Height` or `YOffset` calls.

---
*Phase: 01-status-at-a-glance*
*Completed: 2026-10-12*

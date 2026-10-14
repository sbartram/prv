---
phase: 04-curated-watch-list
plan: 07
subsystem: ui-discoverability
tags: [go, bubbletea-v2, detail-view, help, golden, tdd]
status: complete

requires:
  - phase: 04-curated-watch-list
    provides: "04-05 unavailable rows and unavailSnap; 04-06 d/y remove, Add and Remove key bindings"
provides:
  - "enter on an unavailable row opens a reason-only detail: reason, Help text (SSO hint), 'saved in your watch list' line; no Detail request, also on r"
  - "o opens the saved ref's github.com page from the list row and from the detail"
  - "Footer hints: a add after o open, d remove after it when the selected row is saved"
  - "Empty screen hint ' · press a to add any PR'; ? overlay keys a and d plus legend lines for the + marker and the unavailable row"
  - "21 new goldens (five states at four sizes, compact watch) and 14 regenerated ones"
affects: []

plan_head_before: 9082987358f0a95419de65a5a898e91375b1690a
plan_head_after: 4c3c1963e04ac63f3d563b37a3898b24f5c9397f

actuals:
  tokens: 5600
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "detailLines branches on Unavail.Kind and skips the title, CHECKS and REVIEWERS sections"
    - "listHints() builds the list footer hints from the keymap and inserts Remove only for a saved row"

key-files:
  created:
    - internal/ui/testdata/TestGolden/watch_*.golden
    - internal/ui/testdata/TestGolden/add-checking_*.golden
    - internal/ui/testdata/TestGolden/add-error_*.golden
    - internal/ui/testdata/TestGolden/confirm_*.golden
    - internal/ui/testdata/TestGolden/watch-detail_*.golden
    - internal/ui/testdata/TestGoldenCompactWatch.golden
  modified:
    - internal/ui/detail.go
    - internal/ui/detail_test.go
    - internal/ui/view.go
    - internal/ui/keys.go
    - internal/ui/help.go
    - internal/ui/help_test.go
    - internal/ui/view_test.go
    - internal/ui/unavail_test.go

key-decisions:
  - "The unavailable detail drops the title and status line (an unavailable row has no title or state) and wraps Reason and Help to the pane instead of cutting them, so the SSO hint with 'gh auth refresh' stays whole at 60 columns"
  - "The d remove hint keys off the row's Manual flag, the same test askRemove uses"
  - "unavailSnap rows gained the URL the github seam sets (Ref.URL()) so o can be tested"

requirements-completed: [LIST-09, WATCH-06, WATCH-07]

coverage:
  - id: D1
    description: "enter on an unavailable row explains why in full, offers o to the PR on github.com, and sends no Detail request; a PR that comes back starts its normal fetch"
    requirement: LIST-09
    verification:
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailUnavailable"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailUnavailableRecovers"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailUnavailableNarrow"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestOpenUnavailableRow"
        status: pass
    human_judgment: false
  - id: D2
    description: "a add and d remove are in the footer, the empty screen and the ? overlay; every new screen is pinned by a golden at 0x0, 60x15, 80x24 and 120x40"
    requirement: WATCH-06
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestFooterAddRemoveHints"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestEmptyAddHint"
        status: pass
      - kind: unit
        ref: "internal/ui/help_test.go#TestHelpWatchLegend"
        status: pass
      - kind: golden
        ref: "internal/ui/view_test.go#TestGolden"
        status: pass
    human_judgment: false
  - id: D3
    description: "The warning glyph takes one cell and the + marker reads clearly in the user's terminal and multiplexer pane"
    requirement: LIST-09
    verification: []
    human_judgment: true
    rationale: "Cell width of U+26A0 depends on the terminal font (RESEARCH A5); no test can assert it. See the human-check in the plan."

duration: 20 min
completed: 2026-10-13
---

# Phase 4 Plan 07: Discoverable and readable — the unavailable detail, the hints, the help legend and the goldens Summary

**enter on an unavailable saved PR now shows why (reason, SSO hint, how to remove it) without asking GitHub again, a add and d remove appear in the footer, the empty screen and the ? overlay, and every new screen is pinned by golden files.**

## Accomplishments

- **Unavailable detail (LIST-09, D-10, D-11).** `startDetailFetch` returns nil for a row with `Unavail.Kind != UnavailNone` (so also on `r`). `detailLines` draws `detailNotes()` plus `unavailLines()` for such a row: `⚠ unavailable: REASON`, the Help text, and `saved in your watch list · esc, then d to remove it`. `o` opens the URL set from the saved ref through the existing `safeURL` gate. When a refresh brings the PR back, `refreshDetail` copies the loaded row and the normal fetch starts (`TestDetailUnavailableRecovers`).
- **Hints (D-15).** `ShortHelp` is Move, Refresh, Enter, Open, Add, Filter, States, Help, Quit. `listHints()` inserts Remove after Add when the selected row is saved. The detail hints are unchanged. `FullHelp` first group gains Add and Remove.
- **Empty screen and overlay (D-05, D-09, D-17).** The empty sentence ends with ` · press a to add any PR` (not on the truncated or warning variant). The overlay legend gains `+ added by you; d removes it` and `⚠ unavailable: a saved PR GitHub could not return; enter shows why`.
- **Goldens.** 21 new files: `watch`, `add-checking`, `add-error`, `confirm` and `watch-detail` at 0x0, 60x15, 80x24 and 120x40, plus `TestGoldenCompactWatch`. 14 existing goldens regenerated: `empty` (4), `filtered` (4), `list` 0x0/80x24/120x40, `TestGoldenListColor`, and the two help goldens.

## Task Commits

1. **Task 1** — RED `95b23bd` `test(04-07): add failing tests for the detail view of an unavailable row`; GREEN `4f407a2` `feat(04-07): explain an unavailable saved PR in the detail view`
2. **Task 2** — RED `5cdac90` `test(04-07): add failing tests for the a/d hints, the empty hint and the help legend`; GREEN `4c3c196` `feat(04-07): show a add and d remove in the footer, empty screen and help, with goldens`

## TDD Gate Compliance

| Task | RED commit | GREEN commit |
| ---- | ---------- | ------------ |
| 1 | 95b23bd | 4f407a2 |
| 2 | 5cdac90 | 4c3c196 |

- **RED evidence, task 1.** `TestDetailUnavailable` failed on `detail lacks "⚠ unavailable: SSO: authorize your token for samlcorp"` (the detail still drew CHECKS and REVIEWERS). The package compiled.
- **RED evidence, task 2.** `TestFooterAddRemoveHints` failed on the footer lacking `a add`; `TestEmptyAddHint` on the missing sentence; `TestHelpWatchLegend` on the missing legend lines and keys; the five new golden states failed on missing files.
- **Semantic assessment.** Each RED failure names the planned assertion; none is a compile or import failure. `gsd_run check tdd-red-evidence` was not run (plain `go test` output, plan `type: execute`).

## Golden review

Every changed existing golden differs only as planned: `list`, `filtered` and `TestGoldenListColor` change in the footer line only (1 line each); `empty` changes in the sentence and the footer (2 lines each); the help goldens change in the key table and legend. `list_60x15`, `TestGoldenCompact` and every other golden are byte-identical. No path outside the expected set changed. `watch` at 120x40 shows `a add · d remove` (cursor on a saved row).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The SSO hint was cut at the pane edge**
- **Found during:** Task 2, golden review of `watch-detail_80x24`
- **Issue:** The plan says detail lines are cut to width, so at 80 columns the Help text ended in `gh aut` and the user could not read the fix. This defeats D-11 on any pane under about 100 cells.
- **Fix:** `unavailLines` wraps the reason and the Help text with `m.wrap` (already used by the message screens). Lines stay within the pane. At width 0 nothing is wrapped. `TestDetailUnavailableNarrow` now also asserts that `gh auth refresh` is present at 60 and 40 columns.
- **Files modified:** `internal/ui/detail.go`, `internal/ui/detail_test.go`
- **Commit:** 4c3c196

**2. [Rule 3 - Blocking] The fixture rows had no URL**
- **Found during:** Task 1
- **Issue:** `unavailSnap()` built unavailable rows without the `URL` that the github seam sets, so `o` could not be tested.
- **Fix:** Added `URL: "https://github.com/.../pull/N"` to the two rows in `unavail_test.go`.
- **Commit:** 95b23bd

**3. [Plan reading] No title or state line in the unavailable detail**
- The plan says the detail holds only the ref, the reason, the Help and the remove line. An unavailable row has no title or state, so `detailTitle` (which printed `open · – none`) is skipped for it.

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking), 1 plan reading. **Impact:** none on scope.

## Issues Encountered

None. One test I wrote at first ran the refresh Cmd with real timers (60 s); the helper model now replaces `after` with a no-op.

## Known Stubs

None.

## Threat Flags

None. No new module, endpoint or file access (`git diff --exit-code go.mod go.sum` exits 0). T-04-26 is covered by `TestOpenUnavailableRow` and `TestDetailUnavailable` (the URL comes from the saved ref and passes `safeURL`), T-04-27 by the wrap plus the existing truncation, and T-04-28 by `TestDetailUnavailable` (zero FetchDetail calls, also on `r`).

## Verification

- `go vet ./... && go test ./... -count=1 -race`: all six packages pass without `-update`.
- Task 2 acceptance greps and golden-set checks: 21 new goldens exist; no unexpected path under `internal/ui/testdata`; list, filtered and colour goldens change one line each; `a add`, `press a to add any PR` and `added by you` are in the expected goldens.
- Human check still open (see D3): run `prv` in the real terminal and multiplexer pane with a saved and an unavailable saved PR, confirm the ⚠ glyph is one cell wide and the `+` marker reads clearly at 80 and 40 columns. Record the result in the phase UAT.

## Next Phase Readiness

Last plan of Phase 4. Ready for phase verification and the UAT human-check above.

## Self-Check: PASSED

- Commits 95b23bd, 4f407a2, 5cdac90 and 4c3c196 are ancestors of HEAD.
- The 21 new golden files exist; `internal/ui/detail.go`, `view.go`, `keys.go` and `help.go` carry the changes named above.

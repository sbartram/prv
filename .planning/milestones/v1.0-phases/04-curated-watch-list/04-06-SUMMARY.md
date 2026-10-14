---
phase: 04-curated-watch-list
plan: 06
subsystem: ui-watchlist
tags: [go, bubbletea-v2, confirm-prompt, watchlist, tdd]
status: complete

requires:
  - phase: 04-curated-watch-list
    provides: "04-01 Watchlist interface; 04-03 markMutation and the epoch rule; 04-04 saved list; 04-05 unavailable rows and unavailSnap"
provides:
  - "d on a + row opens the confirm line 'remove REF from watch list? y/n'; y removes through Watchlist.Remove(row.Watched) in a Cmd"
  - "Locked rule: in the confirm line only y removes, only ctrl+c quits, every other key (q included) cancels"
  - "D-07 unpin: a row that is saved and also authored/assigned stays listed without its + marker"
  - "D-08 note for a discovery-only row; a and d do nothing in the detail view"
  - "applyRemoved: idempotent when the entry is already gone, keeps the row on a failed write, shows the empty screen after the last row, and calls markMutation so a stale refresh is discarded"
affects: [04-07]

plan_head_before: 313f164cd432699640bacb59fbff03512060dfbf
plan_head_after: 488a5b6cf3e32ce942eb137e95804051b918b1e2

actuals:
  tokens: 5900
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Confirm line is a Model sub-state (confirmState) checked in Update right after the add prompt; confirmKey always closes it first, then decides"
    - "Remove Cmd mirrors saveCmd: closure over its arguments, panic wrapped in a fixed message"

key-files:
  created:
    - internal/ui/remove.go
    - internal/ui/remove_test.go
  modified:
    - internal/ui/model.go
    - internal/ui/keys.go
    - internal/ui/view.go
    - cmd/prv/main_test.go

key-decisions:
  - "confirmKey closes the confirm before it looks at the key, so no path (a refresh, a lookup failure, an unrelated key) can leave it open or remove anything but on y"
  - "The footer 'd remove' hint is not added; plan 04-07 adds it with the goldens (no golden file changed here)"
  - "Reused the existing toolsRef from unavail_test.go instead of declaring a second one"
  - "m.strikes entry of a removed ref is left alone: the stale count is harmless and the plan did not ask for the change"

requirements-completed: [WATCH-06]

coverage:
  - id: D1
    description: "d then y takes a PR the user added off the list and out of watchlist.json with no GitHub request; the footer says removed REF and the cursor moves to the row below"
    requirement: WATCH-06
    verification:
      - kind: unit
        ref: "internal/ui/remove_test.go#TestRemoveSaved"
        status: pass
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestRemoveEndToEnd"
        status: pass
      - kind: unit
        ref: "internal/ui/remove_test.go#TestRemoveCursorNeighbour"
        status: pass
    human_judgment: false
  - id: D2
    description: "Only y removes: q, n, esc, a, d, j, enter and ? cancel the confirm line without removing, ctrl+c quits; a discovery-only row and the detail view cannot start a removal"
    requirement: WATCH-06
    verification:
      - kind: unit
        ref: "internal/ui/remove_test.go#TestRemoveConfirmCancels"
        status: pass
      - kind: unit
        ref: "internal/ui/remove_test.go#TestRemoveAutoOnly"
        status: pass
      - kind: unit
        ref: "internal/ui/remove_test.go#TestRemoveNotInDetail"
        status: pass
    human_judgment: false
  - id: D3
    description: "Unpin of a saved row that discovery also lists, removal of an unavailable row, an entry already gone, a failed write, the last row, and a refresh that raced the removal"
    requirement: WATCH-06
    verification:
      - kind: unit
        ref: "internal/ui/remove_test.go#TestRemoveUnpin"
        status: pass
      - kind: unit
        ref: "internal/ui/remove_test.go#TestRemoveUnavailable"
        status: pass
      - kind: unit
        ref: "internal/ui/remove_test.go#TestRemoveAlreadyGone"
        status: pass
      - kind: unit
        ref: "internal/ui/remove_test.go#TestRemoveError"
        status: pass
      - kind: unit
        ref: "internal/ui/remove_test.go#TestRemoveLastRow"
        status: pass
      - kind: unit
        ref: "internal/ui/remove_test.go#TestRemoveStaleFetch"
        status: pass
    human_judgment: false

duration: 10 min
completed: 2026-10-13
---

# Phase 4 Plan 06: Remove a PR I added — d, then y Summary

**Pressing d on a + row asks "remove REF from watch list? y/n"; only y removes it (through Watchlist.Remove in a Cmd), q and every other key cancel, and a PR that discovery also lists is unpinned instead of dropped.**

## Performance

- **Duration:** about 10 min
- **Tasks:** 2 (both TDD: RED commit, then GREEN commit)
- **Files:** 6 changed (2 new)

## Accomplishments

- **Remove path (WATCH-06, D-06).** `askRemove` opens `confirmState` for a saved row. `confirmKey` copies the row, closes the confirm, and returns `removeCmd(m.watchlist, pr.Watched)` on `y`, `tea.Quit` on `ctrl+c`, and nil for any other key. `applyRemoved` calls `markMutation`, drops the saved row, rebuilds (the cursor moves to the nearest row, looking below first), and flashes `removed REF`. `TestRemoveEndToEnd` drives the real client, store and model: the row is gone, `store.Load()` returns no ref, and `rt.ops` is unchanged.
- **Locked rule (2026-10-12).** `TestRemoveConfirmCancels` runs q, n, esc, a, d, j, enter and ? through an open confirm: each closes it, returns a nil Cmd, calls Remove zero times, and does not open help, the add prompt or the detail. `ctrl+c` returns `tea.QuitMsg`.
- **Unpin (D-07) and discovery-only rows (D-08).** A row with both `Auto` and `Manual` stays with `Manual=false`, `Watched` cleared, and the flash `unpinned — still listed (authored/assigned)`. `d` on a discovery-only row flashes `REF is authored/assigned — only + rows can be removed` and opens nothing. `a` and `d` in the detail view fall through `detailKey` and change nothing.
- **Edges.** An entry already gone from the file (Remove returns false) removes the row without an error. A write error keeps the row and flashes `could not update the watch list`. Removing the last row shows the empty screen. A refresh that started before the removal is discarded and a new one starts (`TestRemoveStaleFetch`). An unavailable row is removed like any saved row and the header count drops.

## Task Commits

1. **Task 1** — RED `23c9576` `test(04-06): add failing tests for removing a saved PR with d then y`; GREEN `5f87251` `feat(04-06): remove a saved PR with d then y`
2. **Task 2** — RED `57b7236` `test(04-06): add failing tests for the confirm rules, unpin and removal edge cases`; GREEN `488a5b6` `feat(04-06): unpin a saved PR that discovery also lists`

**Plan metadata:** the docs commit that follows this file.

## TDD Gate Compliance

| Task | RED commit | GREEN commit |
| ---- | ---------- | ------------ |
| 1 | 23c9576 | 5f87251 |
| 2 | 57b7236 | 488a5b6 |

- **RED evidence, task 1.** `TestRemoveSaved`: `last line = "j/k move · r refresh · …", want "remove octo/tools#42 from watch list? y/n"` and `y returned no Cmd`. `TestRemoveEndToEnd`: same last-line failure, the row still listed, the footer lacking `removed octo/tools#42`, and `store.Load() = [octo/tools#42]`. Both fail on the planned assertions; the package compiled (no new symbols in the tests).
- **RED evidence, task 2.** Only `TestRemoveUnpin` failed at RED, on `the row is not listed once` (the Task 1 code dropped the row). The other nine tests passed at RED because the Task 1 implementation already covers them (the confirm rule, the discovery-only note, error and already-gone paths, the epoch rule, the last row, the cursor). They stay as guards for those rules, as in 04-05.
- **Semantic assessment.** Each RED failure names the planned assertion for the intended reason; none is a compile, import, fixture or zero-test failure. `gsd_run check tdd-red-evidence` was not run: `go test` plain output is not a supported report format, and the plan is `type: execute`.

## Deviations from Plan

None - plan executed exactly as written. (In Task 2 `model.go` needed no change beyond Task 1: the confirm branch sits after the help overlay, and `?` cannot reach the help overlay because an open confirm consumes it first.)

## Issues Encountered

None.

## Known Stubs

None.

## Threat Flags

None. No new module, endpoint or file access (`git diff --exit-code go.mod go.sum` exits 0). T-04-23, T-04-24 and T-04-25 are covered by `TestRemoveConfirmCancels`, `TestRemoveAutoOnly`, `TestRemoveError` and `TestRemoveStaleFetch`.

## Verification

- `go vet ./... && go test ./... -count=1 -race`: all six packages pass.
- The 11 `TestRemove*` tests of Task 2 all report `--- PASS` (11 lines); `TestRemoveEndToEnd` passes in `cmd/prv`.
- `git diff --exit-code go.mod go.sum internal/ui/testdata` exits 0.
- Acceptance greps: `WithKeys("d")`, `WithKeys("y")`, `from watch list? y/n`, `m.confirm.open`, `unpinned — still listed (authored/assigned)`, `only + rows can be removed`, `func TestRemoveConfirmCancels` all print a line.

## Next Phase Readiness

Ready for 04-07. It still has to add the `d remove` hint to the footer and the `?` legend, regenerate the goldens, and handle the detail view of unavailable rows (D-11). WATCH-06 is declared only by this plan, so it is complete.

## Self-Check: PASSED

- Created files `internal/ui/remove.go` and `internal/ui/remove_test.go`: found.
- Commits 23c9576, 5f87251, 57b7236 and 488a5b6 are ancestors of HEAD.

---
phase: 02-live-monitoring
plan: 02
subsystem: ui
tags: [go, bubbletea-v2, refresh, single-flight, keymap, selection]

requires:
  - phase: 02-live-monitoring
    provides: generation-stamped single-flight tick chain, failed refresh keeps rows (plan 02-01)
provides:
  - "keyMap (Up, Down, Move, Refresh, Quit) with ShortHelp/FullHelp; every list key matches through it"
  - "r refreshes from every screen and merges with a running refresh"
  - "Header 'updated HH:MM:SS' (last successful fetch) plus a spinner while a refresh runs"
  - "Selection tracked by domain.Ref with nearest-neighbour fallback; rows/cursor derived in rebuild()"
affects: [02-03 sort buckets (cursor follows moves), 02-04 footer hints from keyMap, 02-08 filtered rows, 02-10 help overlay and open]

actuals:
  tokens: 14500
  tasks: 3
  commits: 6
plan_head_before: c69b205ec1e182c74a08c8eacea985076e578c6b
plan_head_after: 3e4ed2476f505e5b502b97fb4381eff4995577c0

tech-stack:
  added: []
  patterns:
    - "selected domain.Ref is the source of truth; cursor is derived by rebuild() after every snapshot"
    - "bubbles key package is imported as bkey because the test helper key() owns that package-level name"
    - "Header clock time formatted through Model.clock in the zone of the injected clock, so goldens are machine independent"

key-files:
  created:
    - internal/ui/keys.go
    - internal/ui/selection_test.go
  modified:
    - internal/ui/model.go
    - internal/ui/layout.go
    - internal/ui/view.go
    - internal/ui/model_test.go
    - internal/ui/refresh_test.go
    - internal/ui/view_test.go
    - internal/ui/testdata/TestGolden/list_0x0.golden
    - internal/ui/testdata/TestGolden/list_80x24.golden
    - internal/ui/testdata/TestGolden/list_120x40.golden
    - internal/ui/testdata/TestGoldenListColor.golden

key-decisions:
  - "bubbles/key is imported as bkey in model.go and keys.go: a package-level test helper key() cannot coexist with an import named key"
  - "nearest(): at each distance look below first, then above; with no surviving old neighbour keep the old row index clamped to the new list"
  - "r increments the generation whenever it starts a fetch (all states), so the pending tick is orphaned"

patterns-established:
  - "Later plans add bindings to keyMap; footer (02-04) and help overlay (02-10) render from it"
  - "Failed refresh never calls rebuild(); rows and selection stay as they were"

requirements-completed: [RFSH-02, RFSH-03, RFSH-06]

coverage:
  - id: D1
    description: "r refreshes at once from list, empty and error screens, makes no second request while one runs, and orphans the pending tick"
    requirement: RFSH-02
    verification:
      - kind: unit
        ref: "internal/ui/model_test.go#TestRefreshKey"
        status: pass
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestManualRefreshOrphansTick"
        status: pass
    human_judgment: false
  - id: D2
    description: "Under -race a real tea.Program with a blocking fetcher, a 5 ms interval and r from 4 goroutines never has more than one fetch in flight and quits cleanly"
    requirement: RFSH-03
    verification:
      - kind: e2e
        ref: "internal/ui/refresh_test.go#TestNoOverlappingFetches"
        status: pass
    human_judgment: false
  - id: D3
    description: "Header shows updated HH:MM:SS from the last successful fetch only, with a spinner while a refresh runs; identical data changes only that time"
    requirement: RFSH-03
    verification:
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestUpdatedTimeShown"
        status: pass
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestUpdatedOnlyOnSuccess"
        status: pass
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestSpinnerWhileRefreshing"
        status: pass
      - kind: golden
        ref: "internal/ui/view_test.go#TestGolden"
        status: pass
    human_judgment: false
  - id: D4
    description: "The cursor follows the selected PR by owner/repo#n through reorders and swaps, survives an empty refresh, falls back to the nearest surviving neighbour, and the scroll window shows it"
    requirement: RFSH-06
    verification:
      - kind: unit
        ref: "internal/ui/selection_test.go#TestSelectionFollowsRef"
        status: pass
      - kind: unit
        ref: "internal/ui/selection_test.go#TestSelectionNearestNeighbour"
        status: pass
      - kind: unit
        ref: "internal/ui/selection_test.go#TestSelectionSurvivesEmpty"
        status: pass
      - kind: unit
        ref: "internal/ui/selection_test.go#TestSelectionAgedOut"
        status: pass
      - kind: unit
        ref: "internal/ui/selection_test.go#TestSelectionScrollsIntoView"
        status: pass
      - kind: unit
        ref: "internal/ui/selection_test.go#TestSelectionInvariant"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-10-12
status: complete
---

# Phase 2 Plan 02: r refresh, updated time, selection by ref Summary

**`r` refreshes from every screen without ever overlapping a running fetch, the header shows the clock time of the last successful update with a spinner while refreshing, and the cursor follows the selected PR by owner/repo#n instead of by row index.**

## Performance

- **Duration:** ~12 min
- **Completed:** 2026-10-12T15:35Z
- **Tasks:** 3 (all TDD: RED test commit, then GREEN feature commit)
- **Files modified:** 12 (2 created, 10 modified, 4 of them golden snapshots)

## Accomplishments
- `internal/ui/keys.go`: `keyMap{Up, Down, Move, Refresh, Quit}` with `defaultKeys`, `ShortHelp`, `FullHelp`. `Update` matches every key press with `Matches` against `m.keys`; no `msg.String()` switch is left.
- `r` returns no Cmd while a fetch runs; otherwise it bumps the generation (orphaning the pending tick), switches the error screen back to "Fetching your PRs…", and starts one fetch.
- `TestNoOverlappingFetches` runs a real `tea.Program` under `-race` (5 ms interval, 4 goroutines x 50 `r` presses, blocking fetcher) and asserts `maxInside == 1`; it also passed 5 repeated runs.
- Header reads `prv · @login · N PRs · updated 14:32:05`, plus the spinner frame while `inFlight`. `lastOK` is set only on success, so a failure never moves it. The spinner chain continues only while `inFlight` or loading.
- `selected domain.Ref` is the source of truth; `rebuild()` derives `rows` and `cursor` after each successful snapshot. When the selected PR leaves, `nearest()` picks the closest surviving neighbour (below first, then above at each distance); an empty refresh keeps `selected` so the PR is re-selected when it returns.
- `layout.go` and `view.go` read `m.rows`; goldens for list and colour changed only in the header line.

## Task Commits

1. **Task 1: r from any screen, one keymap**
   - RED `2515925` (test): `TestRefreshKey` list/empty and `TestManualRefreshOrphansTick` failed as expected
   - GREEN `8a9d1a2` (feat)
2. **Task 2: updated time and spinner**
   - RED `7805956` (test): three header tests failed
   - GREEN `34f5715` (feat), with the four regenerated goldens
3. **Task 3: selection by ref**
   - RED `a234689` (test): fails to build until `rows`, `selected`, `indexOf` exist
   - GREEN `3e4ed24` (feat)

**Plan metadata:** committed separately (docs: complete plan)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] bubbles `key` import clashes with the test helper `key()`**
- **Found during:** Task 1 (`go vet`)
- **Issue:** `model_test.go` declares a package-level helper `key(s string)`; Go rejects a file-scope import named `key` in the same package.
- **Fix:** import `charm.land/bubbles/v2/key` as `bkey` in `model.go` and `keys.go` (comment in `keys.go` explains why). The plan's greps for `key.Matches(` still match `bkey.Matches(`. Later plans that import the key package in this package must use the same alias, or rename the test helper.
- **Files modified:** internal/ui/keys.go, internal/ui/model.go
- **Commit:** 8a9d1a2

**2. [Rule 1 - Test expectation] Two nearest-neighbour test expectations corrected before GREEN**
- **Found during:** Task 3, before the RED commit
- **Issue:** I first wrote "two rows away" expecting a/b#1 and "nothing survives" expecting a/b#8. The plan's algorithm gives a/b#5 (below wins at equal distance) and a/b#9 (old index 2 clamped to the 2-row list).
- **Fix:** cases renamed and expectations set to the plan's algorithm.
- **Commit:** a234689

**3. [Process] Task 1 test file replaced via a scripted splice**
- `TestRetryOnlyInError` was replaced by `TestRefreshKey` using a shell splice and a Python edit script instead of the Edit tool; result verified by `grep -c TestRetryOnlyInError` = 0 and the full suite.

**Total deviations:** 1 blocking (import alias), 1 test expectation, 1 process note. **Impact:** none on behaviour.

## Issues Encountered
None. All commits used explicit pathspecs; the pre-staged `.serena` files and modified `.planning/config.json` were left out of every commit.

## Known Stubs
None. `failedAt` and `nextAt` are still set but not rendered; plan 02-04 consumes them (as stated in 02-01).

## Threat Flags
None beyond the plan's threat model. T-02-23 (r and overlapping fetches), T-02-04 (freshness label), T-02-12 (selection after refresh) are mitigated and covered by `TestRefreshKey`, `TestManualRefreshOrphansTick`, `TestNoOverlappingFetches`, `TestUpdatedOnlyOnSuccess`, `TestSelectionFollowsRef`, `TestSelectionInvariant`. No new modules (go.mod and go.sum unchanged).

## Next Phase Readiness
Ready for 02-03 (sort buckets; the cursor already follows moved PRs), 02-04 (footer hints from `keyMap`, failure marker), 02-08 and 02-10. `go build ./...`, `go vet ./...` and `go test ./... -race` pass.

## Self-Check: PASSED

`internal/ui/keys.go` and `internal/ui/selection_test.go` exist; commits `2515925`, `8a9d1a2`, `7805956`, `34f5715`, `a234689`, `3e4ed24` are ancestors of HEAD; all three tasks' acceptance greps pass; `go vet ./... && go test ./... -count=1 -race` pass.

---
*Phase: 02-live-monitoring*
*Completed: 2026-10-12*

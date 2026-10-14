---
phase: 05-address-tech-debt-watch-list-locking-live-smoke
plan: 02
subsystem: ui-add-prompt
tags: [go, bubbletea-v2, add-prompt, error-reasons, golden, tdd]

requires:
  - phase: 05-address-tech-debt-watch-list-locking-live-smoke
    provides: "watchlist.AddRef/RemoveRef with Result{Changed, StartedOver}, ErrBusy, ErrFull, *VersionError (plan 05-01)"
provides:
  - "ui.Watchlist on AddRef/RemoveRef; the backup name from Result.StartedOver reaches the footer as 'previous list saved to watchlist.json.corrupt-XXXXXXXX' for warnFor, ahead of other text"
  - "addState.saving: esc, typing and paste are ignored once the save is dispatched; the prompt reads 'saving REF SPINNER' with no cancel hint; the spinner keeps ticking"
  - "saveReason: ErrFull, ErrBusy, VersionError, permission denied, disk full or write failed; no path or OS text on screen"
  - "Phase 4 review dispositions recorded: 5 fixed, 3 deferred, open 0"
affects: [05-verify, v1 milestone audit]

plan_head_before: fc83b315b05ef36f8dec3899f0da3e8ef0408f59
plan_head_after: 589e38cedee6e8b9f83d0fdafd3e83ee25679a90

actuals:
  tokens: 8900
  tasks: 3
  commits: 7

tech-stack:
  added: []
  patterns:
    - "ui imports the leaf package internal/watchlist for Result and the sentinel errors (still never internal/github)"
    - "A notice leads the footer text and uses warnFor; text without a notice keeps flashFor"
    - "Error text on screen comes from errors.Is/As classes mapped to fixed phrases, never err.Error()"

key-files:
  created:
    - internal/ui/testdata/TestGolden/add-saving_0x0.golden
    - internal/ui/testdata/TestGolden/add-saving_60x15.golden
    - internal/ui/testdata/TestGolden/add-saving_80x24.golden
    - internal/ui/testdata/TestGolden/add-saving_120x40.golden
  modified:
    - internal/ui/add.go
    - internal/ui/remove.go
    - internal/ui/model.go
    - internal/ui/add_test.go
    - internal/ui/remove_test.go
    - internal/ui/view_test.go
    - cmd/prv/main_test.go
    - .planning/phases/04-curated-watch-list/04-REVIEW-DISPOSITION.md

key-decisions:
  - "ui imports internal/watchlist (leaf package, no cycle) instead of adding Short() methods to the store errors: less code, reversible"
  - "The started-over notice uses warnFor (8 s) and comes first in the flash, so a narrow pane cuts the other text first"
  - "jumpTo takes the flash duration, so the duplicate path keeps flashFor and the saved path can use warnFor"
  - "saveReason shows a VersionError's own text (already fixed wording, passed through domain.Clean); every other error maps to a fixed phrase"

requirements-completed: [WATCH-04, WATCH-05, WATCH-06]

coverage:
  - id: D1
    description: "An add over an unparseable watchlist.json flashes 'previous list saved to <real backup name>' for 8 s, ahead of other footer text; the real store keeps the old bytes in the backup"
    requirement: WATCH-06
    verification:
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestAddOverCorruptEndToEnd"
        status: pass
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddStartedOverNotice, internal/ui/remove_test.go#TestRemoveStartedOverNotice"
        status: pass
    human_judgment: false
  - id: D2
    description: "Once the save is dispatched, esc, typed keys and paste change nothing, ctrl+c still quits, the prompt reads 'saving REF' with no cancel hint and the spinner keeps turning; esc during the lookup still cancels"
    requirement: WATCH-04
    verification:
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddEscIgnoredWhileSaving, TestAddSavingLine, TestAddSpinnerWhileSaving, TestAddEscDropsLateResult"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestGolden/add-saving_{60x15,80x24,120x40,0x0}"
        status: pass
    human_judgment: false
  - id: D3
    description: "A failed add or removal names a short fixed reason (full list, another prv writing, version, permission, disk full, write failed) and never a path or control character"
    requirement: WATCH-05
    verification:
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddSaveReasons, internal/ui/remove_test.go#TestRemoveReasons, TestAddSaveError, TestRemoveError"
        status: pass
    human_judgment: false
  - id: D4
    description: "An add over a JSON object whose version is not 1, or that has no version, leaves watchlist.json byte-identical and shows the reason in the prompt with no started-over notice"
    requirement: WATCH-06
    verification:
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestAddVersionMismatchEndToEnd (subtests version2, noversion)"
        status: pass
    human_judgment: false
  - id: D5
    description: "04-REVIEW-DISPOSITION.md records WR-01, WR-02, WR-03, WR-05, IN-02 fixed and WR-04, IN-01, IN-03 deferred with reasons; open 0"
    verification:
      - kind: other
        ref: "awk disposition check from 05-02-PLAN.md Task 3 verify, plus grep '^open: 0$'"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-10-14
status: complete
---

# Phase 05 Plan 02: The add prompt never lies Summary

**The add prompt now flashes the real backup name when an add replaced a broken watchlist.json, ignores esc and typing once a save has started (with a "saving REF" line and a live spinner), and gives every failed add or removal a fixed, path-free reason.**

## Performance

- **Duration:** about 6 min of wall clock
- **Started:** 2026-10-14T03:42:57Z
- **Completed:** 2026-10-14T03:49Z
- **Tasks:** 3 (TDD: 3 RED commits, 3 GREEN commits, 1 docs commit)
- **Files modified:** 12 (498 insertions, 48 deletions), 4 of them new goldens

## Accomplishments

- WR-03 (UI side): `Result.StartedOver` travels through `saveCmd`/`removeCmd` into `addSavedMsg.note` / `removedMsg.note`. The footer reads `previous list saved to watchlist.json.corrupt-XXXXXXXX` for `warnFor`, followed by ` · added REF · hidden by filters`, ` · already watching REF` or ` · removed REF` when those apply. Proven end to end with a real store over `{not json`.
- WR-05: `addState.saving` is separate from `checking`. `addKey` returns nil for everything except ctrl+c while saving, `addPaste` returns early, `addLine` shows `saving REF SPINNER` with no cancel hint, and the `spinner.TickMsg` guard includes `m.add.saving`. Esc during the lookup still cancels and drops late results.
- IN-02: `saveReason` maps `*VersionError`, `ErrBusy`, `fs.ErrPermission`, `syscall.ENOSPC` and everything else to fixed phrases through `domain.Clean`; `ErrFull` gets its own wording. Removals flash `could not update (REASON)`. The generic texts are gone from code and tests.
- A version-2 file and a file with no version key leave `watchlist.json` byte-identical and show the reason in the prompt, with no started-over notice (real store, real ui).
- 04-REVIEW-DISPOSITION.md: WR-01, WR-02, WR-03, WR-05, IN-02 fixed; WR-04, IN-01, IN-03 deferred with the CONTEXT reason; open 0.

## Task Commits

1. **Task 1 (tracer): started-over notice** - `8dd2167` (test), `a721f90` (feat)
2. **Task 2: saving state** - `90facac` (test), `ed82555` (feat)
3. **Task 3: specific reasons and dispositions** - `a490900` (test), `35510df` (feat), `589e38c` (docs)

**Plan metadata:** the docs commit that holds this file.

## Files Created/Modified

- `internal/ui/add.go` - `Watchlist` on `AddRef`/`RemoveRef`, `addState.saving`, `addSavedMsg.note`, `startedOverNotice`, `joinNotice`, `saveReason`, the saving prompt line, `jumpTo` with a duration
- `internal/ui/remove.go` - `RemoveRef`, `removedMsg.note`, `could not update (REASON)`
- `internal/ui/model.go` - the spinner keeps ticking while `m.add.saving`
- `internal/ui/add_test.go`, `remove_test.go`, `view_test.go` - `memWatch` on `AddRef`/`RemoveRef` with `note`/`removeNote`; the new tests; the `add-saving` golden state
- `internal/ui/testdata/TestGolden/add-saving_*.golden` - four new goldens; no other golden changed
- `cmd/prv/main_test.go` - `addTestClient`, `TestAddOverCorruptEndToEnd`, `TestAddVersionMismatchEndToEnd`
- `.planning/phases/04-curated-watch-list/04-REVIEW-DISPOSITION.md` - dispositions

## Decisions Made

- Followed the plan's flagged choices: ui imports `internal/watchlist`; the notice is first and uses `warnFor`; the removal notice is tested with the fake only, because the real store reports none for a removal.
- `jumpTo` gained a duration argument rather than a second function, so the duplicate path keeps `flashFor`.

## Deviations from Plan

None - plan executed exactly as written, with one test-only correction.

- **Test assertion dropped (own test, not a plan rule):** my first `TestAddVersionMismatchEndToEnd` also asserted that no `.corrupt-*` backup exists. The plan only asks for byte-identical file contents. The store (plan 05-01, documented on `read()`) copies a version-mismatch file to a backup when it reads it and leaves the original untouched, so the assertion contradicted existing, documented store behaviour. I removed it in `35510df`. See "Issues Encountered".

**Total deviations:** 0 auto-fixed.

## Issues Encountered

- `watchlist.read()` makes a `.corrupt-HASH8` backup copy for a version-mismatch file as well as for an unparseable one (05-01 design; the original file is not touched, no data is lost). The plan's wording "changes nothing on disk" is true for `watchlist.json`; a harmless extra backup file can appear next to it. If that is not wanted, the fix is in `internal/watchlist` (skip the backup when `ve != nil`), not in the UI. Not changed here, out of scope for 05-02.

## TDD Gate Compliance

- **RED (Task 1), `8dd2167`:** `ui.Watchlist` moved to `AddRef`/`RemoveRef` as a compile stub (the notice is not carried yet). `TestAddStartedOverNotice` (3 subtests) failed with `flash = "", want "previous list saved to ..."` and wrong `flashUntil`; `TestRemoveStartedOverNotice` failed with `flash = "removed octo/tools#42"`; `TestAddOverCorruptEndToEnd` failed with `last line = "j/k move ... api 4999", want it to contain "previous list saved to watchlist.json.corrupt-92072df3"`. Semantic assessment: each target ran and failed on the planned assertion; no load or fixture fault.
- **GREEN (Task 1), `a721f90`.**
- **RED (Task 2), `90facac`:** `addState.saving` as a compile stub. `TestAddEscIgnoredWhileSaving` (`saving = false, checking = true`), `TestAddSavingLine` (`checking` text, no `saving`), `TestAddSpinnerWhileSaving` (setup assertion `saving = false`), and `TestGolden/add-saving_*` (golden files missing). Semantic assessment: the saving behaviour was absent, so each failed on the planned assertion; the golden failures are "file missing", which is the expected RED for a new golden. The spinner test fails on its setup assertion because with `checking` still true the tick would also continue; the real tick assertion becomes meaningful only in GREEN.
- **GREEN (Task 2), `ed82555`.**
- **RED (Task 3), `a490900`:** `TestAddSaveReasons` (7 subtests), `TestRemoveReasons` (6), `TestAddSaveError`, `TestRemoveError` failed with `errText = "could not save the watch list", want "could not save (another prv is writing the list)"` and the flash equivalents; `TestAddVersionMismatchEndToEnd` failed with the generic prompt text. One extra failure in that RED (the backups-none assertion) was my test's mistake and was removed in GREEN.
- **GREEN (Task 3), `35510df`.** No REFACTOR commit was needed.
- The `gsd_run check tdd-red-evidence` classifier was not run: Go's default output is not one of its supported report formats, and `workflow.tdd_mode` is not set. The RED runs were inspected by hand as above.

## Verification

- `go vet ./... && go test ./... -count=1 -race`: all packages pass.
- `go test ./internal/ui -run '^TestGolden$/^add-saving' -update` created exactly the 4 new goldens; `git status --short internal/ui/testdata` listed only those; none contains `esc cancel`; all contain `saving octo/new#7`.
- `git diff --exit-code <commit that added 05-01-SUMMARY.md> -- go.mod go.sum`: exit 0.
- Disposition check (awk + `open: 0`): pass.
- `grep -rn 'could not save the watch list\|could not update the watch list' internal/ui/*.go cmd/prv/*.go`: no output.
- `grep -n '"github.com/sbartram/prv/internal/github"' internal/ui/*.go`: no output.

## Known Stubs

None.

## Threat Flags

None. T-05-11 (paths on screen), T-05-12 (esc during save) and T-05-13 (silent data loss) are mitigated as planned and each has a test. No new network, auth or file surface.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

All three Phase 5 plans have summaries once this one is committed. Phase 5 is ready for verification (`/gsd-verify-work`), including the manual checks in `05-UAT.md` from plan 05-03.

## Self-Check: PASSED

- FOUND: internal/ui/testdata/TestGolden/add-saving_0x0.golden, add-saving_60x15.golden, add-saving_80x24.golden, add-saving_120x40.golden
- FOUND: internal/ui/add.go, remove.go, model.go, add_test.go, remove_test.go, view_test.go, cmd/prv/main_test.go, 04-REVIEW-DISPOSITION.md (all in `git diff --stat` of the plan range)
- FOUND commits (7, `git rev-list --count` from the ledger base): 8dd2167, a721f90, 90facac, ed82555, a490900, 35510df, 589e38c

---
*Phase: 05-address-tech-debt-watch-list-locking-live-smoke*
*Completed: 2026-10-14*

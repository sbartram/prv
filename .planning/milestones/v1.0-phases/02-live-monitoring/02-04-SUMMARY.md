---
phase: 02-live-monitoring
plan: 04
subsystem: ui
tags: [go, bubbletea-v2, header, footer, bubbles-help, ansi16]

requires:
  - phase: 02-live-monitoring
    provides: keyMap with ShortHelp, failedAt/nextAt/lastOK fields, updated-time header, spinner (plans 02-01, 02-02)
provides:
  - "(*github.Error).Short(): fixed one-or-two-word reason per kind (D-07)"
  - "fit(): priority header composer; the failure marker is dropped last (Pitfall 10)"
  - "Header failure marker '✗ refresh failed HH:MM (<reason>) · data from HH:MM:SS' in ANSI yellow"
  - "Error panel 'retrying at HH:MM' / 'retrying now…' line (D-09)"
  - "footer(): integrity notes first, key hints from the keymap via bubbles/help in ANSI-16 styles (D-17, WR-03)"
  - "Empty screen with header and footer; 'No PRs could be listed for @login.' over a partial load (WR-02)"
affects: [02-05 rate warning takes header prio 6, 02-08 filter segment takes header prio 3, 02-10 help overlay and more footer bindings]

actuals:
  tokens: 5500
  tasks: 2
  commits: 4
plan_head_before: 94f099a36854d7109add7402db2d81a266ef9699
plan_head_after: 724709ad7dce1f0f2513705bc65d62ac0c5eeb39

tech-stack:
  added: []
  patterns:
    - "Header is a list of {text, prio} segments joined by fit(); lowest prio drops first, a single remaining segment is cut with an ellipsis"
    - "ui finds Short() through a local shorter interface (like hinter), so it never imports internal/github"
    - "help.Styles built by helpStyles() from plain and faint styles; help.DefaultStyles is never used (hex colours)"

key-files:
  created: []
  modified:
    - internal/github/errors.go
    - internal/github/errors_test.go
    - internal/ui/styles.go
    - internal/ui/view.go
    - internal/ui/view_test.go
    - internal/ui/model.go
    - internal/ui/model_test.go
    - internal/ui/refresh_test.go
    - internal/ui/testdata/TestGolden/error_0x0.golden
    - internal/ui/testdata/TestGolden/error_80x24.golden
    - internal/ui/testdata/TestGolden/error_120x40.golden
    - internal/ui/testdata/TestGolden/empty_0x0.golden
    - internal/ui/testdata/TestGolden/empty_80x24.golden
    - internal/ui/testdata/TestGolden/empty_120x40.golden
    - internal/ui/testdata/TestGolden/list_0x0.golden
    - internal/ui/testdata/TestGolden/list_80x24.golden
    - internal/ui/testdata/TestGolden/list_120x40.golden
    - internal/ui/testdata/TestGoldenListColor.golden

key-decisions:
  - "Header priorities: prv 1, @login 2, N PRs 4, updated 5, failure marker 7; 3 and 6 stay free for plans 02-08 and 02-05"
  - "The spinner is appended outside warnStyle, so the yellow covers only the marker text"
  - "Footer with notes: the notes are cut with an ellipsis when wider than the pane and hints need at least 8 cells; notes are never dropped"
  - "Error panel layout: retry line sits between the hint block and 'r retry · q quit', each separated by a blank line"

patterns-established:
  - "Later header segments go through fit() with a prio; the footer renders hints only from m.keys.ShortHelp()"

requirements-completed: [RFSH-04, RFSH-03]

coverage:
  - id: D1
    description: "A failed refresh keeps the rows undimmed and marks the header with a short reason and the time of the data on screen; the next success clears the marker; the marker carries ANSI yellow"
    requirement: RFSH-04
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestFailureMarker"
        status: pass
      - kind: unit
        ref: "internal/github/errors_test.go#TestShortReason"
        status: pass
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestUpdatedOnlyOnSuccess"
        status: pass
    human_judgment: false
  - id: D2
    description: "At widths 40, 60, 80 and 120 no header line is wider than the pane and the failure marker is the last segment kept"
    requirement: RFSH-04
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestHeaderPriority"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestFitDropsLowestPriorityFirst"
        status: pass
    human_judgment: false
  - id: D3
    description: "The first-fetch error panel says 'retrying at HH:MM', or 'retrying now…' while the automatic retry runs, and keeps 'r retry · q quit'"
    requirement: RFSH-04
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestRetryingAt"
        status: pass
      - kind: golden
        ref: "internal/ui/view_test.go#TestGolden"
        status: pass
    human_judgment: false
  - id: D4
    description: "Integrity notes come first in the footer and outlive key hints on narrow panes; hints come from the keymap in ANSI-16 styles"
    requirement: RFSH-03
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestFooterNotesFirst"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestStylesUseAnsi16"
        status: pass
      - kind: golden
        ref: "internal/ui/view_test.go#TestGoldenListColor"
        status: pass
    human_judgment: false
  - id: D5
    description: "The empty screen says 'No PRs could be listed' and shows the notes over a partial load; with no notes it keeps the Phase 1 sentence"
    requirement: RFSH-03
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestEmptyStateShowsNotes"
        status: pass
      - kind: unit
        ref: "internal/ui/model_test.go#TestEmptyStateNamesAccount"
        status: pass
    human_judgment: false
  - id: D6
    description: "In a real 80-column terminal the failure marker, its removal on recovery and the 40-column footer read correctly"
    requirement: RFSH-04
    verification: []
    human_judgment: true
    rationale: "The plan's human-check (Wi-Fi off and on, pane resize in a live terminal) is recorded for end-of-phase UAT; unit tests cover the strings and widths but not the real-terminal look"

duration: 3min
completed: 2026-10-12
status: complete
---

# Phase 2 Plan 04: Failure marker, retry line, notes-first footer Summary

**A failed refresh now keeps the rows and marks the header "✗ refresh failed HH:MM (reason) · data from HH:MM:SS" in ANSI yellow through a priority-based header composer, the error panel says when it retries, and the footer puts integrity notes before key hints taken from the keymap.**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-10-12T15:46:46Z
- **Completed:** 2026-10-12T15:49:37Z (code); SUMMARY and state follow
- **Tasks:** 2 (both TDD: RED test commit, then GREEN feature commit)
- **Files modified:** 18 (9 of them golden snapshots)

## Accomplishments
- `(*github.Error).Short()` returns a fixed string per kind (auth, forbidden, server, network, GitHub error, error); `TestShortReason` checks a token-bearing message never leaks.
- `fit()` and `Model.header()`: segments `prv`(1) `@login`(2) `N PRs`(4) status(5 or 7). At 60 columns `prv` and `@octo` drop first, at 40 the marker alone is cut with an ellipsis and still starts with "✗ refresh failed".
- `shorter` interface in `ui` finds `Short()` through `errors.As`; an error without it shows "(error)". `ui` still does not import `internal/github`.
- Error panel adds "retrying at HH:MM" (from `nextAt`) or "retrying now… <spinner>" while the automatic retry runs.
- `footer()`: notes ("list truncated", "some data could not be loaded") first; hints from `help.ShortHelpView(m.keys.ShortHelp())` with `helpStyles()` (plain keys, faint descriptions and separators, no hex). Notes wider than the pane are cut with "…"; hints need at least 8 cells.
- Empty screen renders header, wrapped message, blank line, footer; "No PRs could be listed for @login." when the snapshot is truncated or has warnings.

## Task Commits

1. **Task 1: header marker, retrying line, priority fitting**
   - RED `10dfdb0` (test): build failed on missing `segment` and `Short`
   - GREEN `865fa6c` (feat), with the three error goldens
2. **Task 2: notes-first footer, keymap hints, honest empty screen**
   - RED `9426d8d` (test): build failed on missing `footer`
   - GREEN `724709a` (feat), with six list/empty goldens and the colour golden

**Plan metadata:** committed separately (docs: complete plan)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Test expectation] TestUpdatedOnlyOnSuccess and TestFailedRefreshKeepsData asserted the old header**
- **Found during:** Task 1 (full test run)
- **Issue:** Both tests, from plans 02-01/02-02, required the screen after a failed refresh to be byte-identical to the screen before, and the header to still say "updated". The D-07 marker deliberately changes the header line. The plan named `TestUpdatedOnlyOnSuccess` in its verify list but did not list `refresh_test.go` in `files_modified`.
- **Fix:** Both tests now compare every line after the header (rows unchanged) and `TestUpdatedOnlyOnSuccess` asserts "data from 14:32:05" and no "updated". The prohibition (no stale data shown as fresh) is still covered, and `TestFailureMarker` covers the marker itself.
- **Files modified:** internal/ui/refresh_test.go
- **Committed in:** 10dfdb0 (assertions) and 865fa6c (second edit with the feature)

**2. [Rule 1 - Test expectation] My fit() width table was off by one**
- **Found during:** Task 1, before the GREEN commit
- **Issue:** `TestFitDropsLowestPriorityFirst` used widths 13/12/8/7 where the joined strings measure 14/13/9/8.
- **Fix:** Corrected the table; no code change.
- **Committed in:** 865fa6c

---

**Total deviations:** 2 auto-fixed (2 test expectations). **Impact:** none on behaviour; one extra file (`refresh_test.go`) touched beyond `files_modified`.

## Issues Encountered
None. All commits used explicit pathspecs; the staged `.serena` files and modified `.planning/config.json` were left out.

## Known Stubs
None.

## Threat Flags
None beyond the plan's threat model. T-02-24 (stale data presented as fresh) is mitigated by the marker and `TestFailureMarker`/`TestHeaderPriority`; T-02-05 by fixed `Short()` strings and `TestShortReason`; T-02-27 by `TestFooterNotesFirst` and `TestEmptyStateShowsNotes`; T-02-SC: no new modules (`go.mod` and `go.sum` unchanged).

## Next Phase Readiness
Ready for 02-05 (rate warning at header prio 6) and 02-08 (filter segment at prio 3); 02-10 can add bindings to `keyMap.ShortHelp()` between "r refresh" and "q quit". The end-of-phase human check from Task 2 (real terminal, Wi-Fi off/on, 40-column pane) is recorded for UAT. `go vet ./...`, `go build ./...` and `go test ./... -race` pass.

## Self-Check: PASSED

Modified files exist; commits `10dfdb0`, `865fa6c`, `9426d8d`, `724709a` are ancestors of HEAD; all acceptance greps of both tasks pass (Short, shorter, fit, footer, ShortHelpView, chromeLines = 3, golden contents, golden hygiene empty); no hex or adaptive colour in `internal/ui`; `go vet ./... && go test ./... -count=1 -race` pass.

---
*Phase: 02-live-monitoring*
*Completed: 2026-10-12*

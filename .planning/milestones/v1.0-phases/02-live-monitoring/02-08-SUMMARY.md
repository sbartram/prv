---
phase: 02-live-monitoring
plan: 08
subsystem: ui
tags: [go, filters, bubbletea-v2, prompt]

requires:
  - phase: 02-live-monitoring
    provides: "selection by ref with nearest-neighbour fallback and rebuild() (02-02); Sort/Bucket attention order (02-03); header segment priorities with prio 3 reserved, footer notes and key hints from keyMap.ShortHelp() (02-04)"
provides:
  - "domain.StateSet (Has, Toggle), domain.AllStates and the pure domain.Visible(prs, show, number) filter"
  - "keys 1/2/3 toggle Open/Merged/Closed; header segment '[open merged closed]' (prio 3) with '#digits' appended while a number filter is set"
  - "count '<visible> of <total> PRs' when filters hide PRs; filtered-empty screen 'No PRs match: ...' with the hidden count"
  - "'/' digits-only prompt (hand-rolled buffer, 7 digits, bracketed paste keeps only ASCII digits); enter keeps, esc clears; the prompt replaces the footer line"
  - "ui.Options.Show for plan 02-09 to set from the config key show"
affects: [02-09, 02-10]

actuals:
  tokens: 9000
  tasks: 2
  commits: 2
plan_head_before: 8e0647bca6f7bda3ba6ee3af4760cb055ded83a3
plan_head_after: bf0656b204e7bf4cdca003fa63be1bb9258b2f21

tech-stack:
  added: []
  patterns:
    - "Filters live in rebuild(): rows = domain.Visible(snap.PRs, show, numFilter), so refresh results and toggles share one path and selection follows the ref"
    - "bottom() is the one last line of every screen with rows or a filter: the prompt while it is open, otherwise the footer"
    - "Prompt keys are handled before the list key switch, so q, 1/2/3 and ? never reach list handlers while the prompt is open"

key-files:
  created:
    - internal/ui/filter_test.go
    - internal/ui/testdata/TestGolden/filtered_0x0.golden
    - internal/ui/testdata/TestGolden/filtered_80x24.golden
    - internal/ui/testdata/TestGolden/filtered_120x40.golden
    - internal/ui/testdata/TestGolden/prompt_0x0.golden
    - internal/ui/testdata/TestGolden/prompt_80x24.golden
    - internal/ui/testdata/TestGolden/prompt_120x40.golden
  modified:
    - internal/domain/list.go
    - internal/domain/domain_test.go
    - internal/ui/keys.go
    - internal/ui/model.go
    - internal/ui/view.go
    - internal/ui/view_test.go

key-decisions:
  - "A PR in an unknown state is never hidden by a state toggle (RESEARCH A4); the number filter still applies to it"
  - "The prompt opens only on the list screen (stateList); toggles work in every state because they only change session state"
  - "bottom() serves the empty screen too, so a refresh that returns zero PRs while the prompt is open cannot strand an invisible prompt"
  - "Toggling a state back restores rows, order and header exactly; the selection returns only if it never had to move (a hidden selected PR moves to its nearest visible neighbour, per D-03)"

patterns-established:
  - "Golden snapshots are regenerated with one -update run and reviewed by diff; only footer hints and the header filter segment changed in existing goldens"

requirements-completed: [FILT-01, FILT-03]

coverage:
  - id: D1
    description: "Keys 1/2/3 toggle Open/Merged/Closed in any combination; the header shows the state indicator and '<visible> of <total> PRs'; toggling twice restores the screen; unknown-state PRs are never hidden"
    requirement: FILT-01
    verification:
      - kind: unit
        ref: "internal/ui/filter_test.go#TestStateToggles"
        status: pass
      - kind: unit
        ref: "internal/ui/filter_test.go#TestToggleTwiceRestores"
        status: pass
      - kind: unit
        ref: "internal/ui/filter_test.go#TestUnknownAlwaysVisible"
        status: pass
      - kind: unit
        ref: "internal/ui/filter_test.go#TestHiddenCount"
        status: pass
      - kind: unit
        ref: "internal/domain/domain_test.go#TestVisible"
        status: pass
    human_judgment: false
  - id: D2
    description: "A filter that hides every PR shows 'No PRs match: ... N hidden by filters' with integrity notes, and a toggle or refresh keeps the filter on new data"
    requirement: FILT-01
    verification:
      - kind: unit
        ref: "internal/ui/filter_test.go#TestFilteredEmpty"
        status: pass
      - kind: unit
        ref: "internal/ui/filter_test.go#TestToggleSurvivesRefresh"
        status: pass
      - kind: unit
        ref: "internal/ui/filter_test.go#TestSelectionAfterToggle"
        status: pass
    human_judgment: false
  - id: D3
    description: "'/' opens a digits-only prompt that narrows the list live by PR-number prefix; enter keeps, esc clears; paste keeps only digits; the prompt costs no list row"
    requirement: FILT-03
    verification:
      - kind: unit
        ref: "internal/ui/filter_test.go#TestNumberPrompt"
        status: pass
      - kind: unit
        ref: "internal/ui/filter_test.go#TestPrefixMatch"
        status: pass
      - kind: unit
        ref: "internal/ui/filter_test.go#TestPromptKeepsOrder"
        status: pass
      - kind: unit
        ref: "internal/ui/filter_test.go#TestRefreshWhilePrompting"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestGolden"
        status: pass
    human_judgment: false
  - id: D4
    description: "Toggles, prompt, paste and Esc feel right in a real terminal with real PRs"
    requirement: FILT-03
    verification: []
    human_judgment: true
    rationale: "Live typing, bracketed paste of a PR URL and visual clarity of the header indicator cannot be judged by a unit test; recorded for end-of-phase UAT"

duration: 14min
completed: 2026-10-12
status: complete
---

# Phase 02 Plan 08: Narrow the list by state and PR number Summary

**Keys 1/2/3 toggle Open/Merged/Closed with a "[open merged ·]" header indicator and "N of M PRs" count, and "/" opens a digits-only prompt that narrows the list by PR-number prefix, with a filtered-empty screen that says how many PRs the filters hide.**

## Performance

- **Duration:** 14 min
- **Tasks:** 2
- **Files modified:** 20 (6 new goldens, 1 new test file)

## Accomplishments

- `domain.StateSet` and `domain.Visible` are pure and tested apart from the UI: unknown-state PRs pass every toggle, a number prefix matches `12` to `#12`, `#120` and `#1234` but not `#312`, and the input slice is never modified.
- `rebuild()` derives `m.rows` from the snapshot, the state set and the number filter, so a toggle, a typed digit and a refresh result all use one path; selection still follows the PR ref with the nearest-neighbour fallback.
- The header shows one slot per state (name when visible, `·` when hidden), `#digits` after it for a kept number filter, and `<visible> of <total> PRs` whenever something is hidden. The filtered-empty screen is distinct from the Phase 1 empty sentence and still carries truncation and partial-data notes (WR-02).
- The prompt is a hand-rolled buffer capped at 7 digits. While it is open `q`, `?`, `1`/`2`/`3` and other keys do not act as list keys; `ctrl+c` still quits. A paste keeps only ASCII digits (T-02-15). The prompt replaces the footer, so the row budget stays height minus 3.

## Task Commits

1. **Task 1: toggles, header indicator, hidden count, filtered-empty screen** - `c230ec2` (feat)
2. **Task 2: "/" digits-only number prompt** - `bf0656b` (feat)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Existing empty-screen test asserted the old header**
- **Found during:** Task 1
- **Issue:** `TestEmptyStateShowsNotes` required the header to start with `prv · @octo · 0 PRs`; the plan puts the filter segment between the login and the count.
- **Fix:** updated the expected prefix to `prv · @octo · [open merged closed] · 0 PRs`.
- **Files modified:** internal/ui/view_test.go
- **Commit:** c230ec2

**2. [Rule 1 - Bug] Own test expected the selection to return after toggling it away**
- **Found during:** Task 1
- **Issue:** `TestToggleTwiceRestores` selected the first PR, which the "1" toggle hides, so the selection legitimately moves to its neighbour (D-03).
- **Fix:** the test selects the unknown-state PR, which no toggle hides; the plan's "restores rows, order, selection and header" holds for a selection that did not have to move.
- **Files modified:** internal/ui/filter_test.go
- **Commit:** c230ec2

**3. [Rule 2 - Missing critical] The prompt also renders on the zero-PR empty screen**
- **Found during:** Task 2
- **Issue:** a refresh that returns zero PRs while the prompt is open would switch to the empty screen and leave a prompt that swallows keys but is not drawn.
- **Fix:** the empty screen and the filtered-empty screen use `bottom()` as well; the prompt opens only from the list screen.
- **Files modified:** internal/ui/view.go, internal/ui/model.go
- **Commit:** bf0656b

**Total deviations:** 3 auto-fixed (2 bugs in tests, 1 missing-critical guard). **Impact:** none on scope; no new modules, go.mod and go.sum unchanged.

## Verification

- `go vet ./...` and `go test ./... -count=1 -race`: pass.
- Task 1 verify run: 14 top-level PASS lines; Task 2 verify run: all 11 `TestNumberPrompt` subtests pass.
- Golden diffs reviewed: existing list/empty/filtered goldens changed only in the header filter segment and footer hints; loading and error snapshots unchanged; only `.golden` files changed under `internal/ui/testdata`.

## Known Stubs

None.

## Threat Flags

None. T-02-14 (filter transparency) is mitigated by the header indicator, the "of" count, the D-13 message and the unknown-state rule; T-02-15 (paste) by the digits-only buffer capped at 7.

## Issues Encountered

None.

## User Setup Required

None.

## Next Phase Readiness

Ready for 02-09, which sets `ui.Options.Show` from the config key `show`. The end-of-phase human check (press 2 and 3, type digits after "/", paste a PR URL, Enter, j/k, Esc) is recorded for UAT.

## Self-Check: PASSED

- Created files exist: internal/ui/filter_test.go, filtered_* and prompt_* goldens.
- Commits `c230ec2` and `bf0656b` are on the branch.
- Acceptance greps (Visible signature, domain.Visible call, ToggleOpen, maxFilterDigits = 7, tea.PasteMsg, "enter keep · esc clear") all hit.

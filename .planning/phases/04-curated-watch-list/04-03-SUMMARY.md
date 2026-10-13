---
phase: 04-curated-watch-list
plan: 03
subsystem: ui
tags: [go, bubbletea-v2, prompt, paste, duplicates, race, tdd]

requires:
  - phase: 04-curated-watch-list
    provides: add prompt, Looker, Watchlist, addState, Lookup, saved-PR merge (plan 04-01); ParseRef reasons and SSO Unavail (plan 04-02)
provides:
  - "Duplicate adds jump to the row: found by Ref.Key() of the row's Ref or Watched ref before any request, and by node ID after the Lookup; nothing is looked up twice or saved"
  - "'hidden by filters' wording for a duplicate or a new PR that the 1/2/3 or / filters hide; the selection does not move"
  - "Fetch epochs: fetchedMsg.epoch, Model.fetchEpoch/mutEpoch, markMutation(); a refresh that started before an add is discarded and re-run, or the pending tick is armed to the end of a rate-limit pause"
  - "Rate-limit pause in the prompt: enter during a pause shows 'rate limited — resumes HH:MM'; a rate-limited Lookup moves the pause"
  - "SSO help in the prompt line; prompt line that fits any pane width (hint first, then the tail of the text behind '…')"
  - "Test fakes for plans 04-05..04-07: addFetcher, memWatch, addModel, submit, drainUI, freshResult"
affects: [04-04, 04-05, 04-06, 04-07]

plan_head_before: ae0a3cc36a3a7dca8b917d18ee7f883aa8c0ff37
plan_head_after: 7752d90f8e288c3b7f7d3e5bf6217e3323c3ac81

actuals:
  tokens: 8300
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Epoch stamp on every fetch: results older than the last local change are discarded and re-run (no I/O in Update)"
    - "Prompt line fitted by priority: cursor and status text outrank the typed text, which shows its tail"
    - "Duplicate checks by Key (ref) before the request and by node ID after it"

key-files:
  created:
    - internal/ui/add_test.go
  modified:
    - internal/ui/add.go
    - internal/ui/model.go
    - internal/ui/refresh.go

key-decisions:
  - "A fetch result is discarded by epoch, not by generation: gen already orphans ticks, while the epoch only says whether the fetch began before the last local watch-list change"
  - "A save that finds the PR already listed marks that row Manual with the saved ref, instead of waiting for the next refresh to show the + marker"
  - "A visible, newly added PR gets no flash (the row and cursor are the feedback); only the hidden and already-watching outcomes speak"
  - "The pause check runs before parsing, as the plan says, so a typo during a pause shows the pause text"

patterns-established:
  - "freshResult(m, snap, err) stamps a fetchedMsg with the model's gen and mutEpoch for tests that send a refresh result after a local change"
  - "drainUI(m, cmd) delivers Cmds and the Cmds they return, skipping spinner ticks, at most 5 rounds"

requirements-completed: [WATCH-04]

coverage:
  - id: D1
    description: "Pasting a PR that is already listed (discovered, saved, unavailable, any letter case, or the same node ID under a new name) does not create a second row: no Lookup, no save, the cursor moves to the row, or the footer says it is hidden by filters"
    requirement: WATCH-04
    verification:
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddDuplicateByRef"
        status: pass
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddDuplicateHidden"
        status: pass
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddDuplicateByID"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every add mistake is explained inline with the typed text kept and nothing saved: parser reason, Lookup reason with SSO help, could not check (SHORT), rate limited with the resume time, could not save the watch list"
    requirement: WATCH-04
    verification:
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddLookupErrors"
        status: pass
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddParseErrorKeepsText"
        status: pass
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddDuringPause"
        status: pass
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddSaveError"
        status: pass
    human_judgment: false
  - id: D3
    description: "A refresh that began before an add can never make the new row vanish: it is discarded and re-run, or the tick waits out a rate-limit pause"
    requirement: WATCH-04
    verification:
      - kind: unit
        ref: "internal/ui/add_test.go#TestStaleFetchAfterAdd"
        status: pass
      - kind: unit
        ref: "internal/ui/add_test.go#TestFetchWithoutEpochApplies"
        status: pass
    human_judgment: false
  - id: D4
    description: "Typing and pasting are safe (escapes stripped, two lines refused, 256 runes, q a d 1 / : typed, ctrl+c quits), esc drops late results, and the prompt line fits panes of 120, 60, 40 and 20 cells"
    requirement: WATCH-04
    verification:
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddPasteSanitised"
        status: pass
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddTyping"
        status: pass
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddEscDropsLateResult"
        status: pass
      - kind: unit
        ref: "internal/ui/add_test.go#TestAddPromptWidths"
        status: pass
    human_judgment: false
  - id: D5
    description: "The prompt line reads well in a real narrow terminal pane (how the cut tail with '…' and the long SSO help line look)"
    verification: []
    human_judgment: true
    rationale: "Tests assert width, cursor and error text, not the look; the prompt goldens are plan 04-07"

duration: 25min
completed: 2026-10-13
status: complete
---

# Phase 04 Plan 03: The add prompt is forgiving - typos are explained, duplicates jump to the row, and a refresh can never undo an add Summary

**Pasting a PR that is already listed moves the cursor to it (or says the filters hide it) and saves nothing; every add mistake is explained in the prompt line with the typed text kept; and an epoch on every fetch discards any refresh that began before an add, so a new row never disappears.**

## Performance

- **Duration:** about 25 min
- **Completed:** 2026-10-13
- **Tasks:** 2
- **Files:** 1 created, 3 modified

## Accomplishments

- **Task 1 (duplicates and the stale refresh):** `findListed` matches a pasted or looked-up PR to a row by node ID, by `Ref.Key()` of the row's `Ref`, or by `Ref.Key()` of a saved row's `Watched` ref (so unavailable saved rows count). `submitAdd` checks by key before any request; `applyAddChecked` checks by key and node ID after the Lookup (a renamed repository). `jumpTo` selects the row or, when the filters hide it, flashes `... · hidden by filters` and leaves the selection. `applyAddSaved` lists the PR at once, handles `Add` returning `(false, nil)` ("already watching"), and flashes "added REF · hidden by filters" for a hidden new row. `fetchedMsg` carries an `epoch`; `markMutation()` sets `mutEpoch = fetchEpoch + 1`; a successful result with a lower epoch is discarded and a new fetch starts, or, during a rate-limit pause, the one pending tick is armed to the end of the pause.
- **Task 2 (input and errors):** enter during a pause shows `rate limited — resumes HH:MM` and sends nothing; a rate-limited Lookup calls `pauseForRateLimit`; an SSO Unavail shows its Reason and Help. `addLine` drops the input hint first, then shows the tail of the typed text behind `…` (or nothing when even one cell is too much), so the cursor, the checking text and the error always outrank the text. The behavior list for `a`, typing, paste cleaning, esc and late results, and the save error was already right from plan 04-01; the new tests pin it.
- No golden file, `go.mod` or `go.sum` change (`git diff --exit-code internal/ui/testdata go.mod go.sum` is clean).

## TDD Gate Compliance

Both tasks ran RED then GREEN. A Go test that does not compile is not a valid RED, so task 1's RED commit carries compile-only stubs (the `epoch` field, the `fetchEpoch` and `mutEpoch` fields and an empty `markMutation`); task 2 needed none. Every target test then failed on its assertion for the planned behavior (for example `looked [ACME/API#1], saved [ACME/API#1]; a duplicate sends no Lookup and saves nothing`, `fetchEpoch=0, want 1`, `last line has no cursor`, `want the pause error`). `workflow.tdd_mode` is off, the plan type is `execute`, and the `tdd-red-evidence` classifier has no Go report adapter, so the classifier was not run; the semantic assessment is: each failing test executed and failed on the planned assertion, not on a compile, setup or unrelated fault.

Some tests passed already at RED because plan 04-01 had built that behavior: `TestAddFromEmpty`, `TestFetchWithoutEpochApplies`, `TestAddOnlyOnListScreens`, `TestAddNotAvailable`, `TestAddTyping`, `TestAddPasteSanitised`, `TestAddParseErrorKeepsText`, `TestAddEscDropsLateResult`, `TestAddSaveError`. They are regression guards for the plan's behavior list, not RED evidence; the plan itself says "check that openAdd, addKey and addPaste already give the behavior list".

| Gate | Commit |
|------|--------|
| RED (task 1) | 51386f9 test(04-03): add failing tests for duplicate adds and the stale-refresh epoch |
| GREEN (task 1) | d8a080f feat(04-03): jump to a PR that is already listed and never let an old refresh undo an add |
| RED (task 2) | c4ac22b test(04-03): add failing tests for the add prompt input, errors, pause and widths |
| GREEN (task 2) | 7752d90 feat(04-03): explain every add mistake in the prompt line and fit it to any pane |

No REFACTOR commit was needed.

## Task Commits

1. Task 1 RED: `51386f9`
2. Task 1 GREEN: `d8a080f`
3. Task 2 RED: `c4ac22b`
4. Task 2 GREEN: `7752d90`

## Deviations from Plan

**1. [Rule 3 - Blocking] Test helper renamed to avoid a clash**
- **Found during:** Task 2 RED
- **Issue:** `internal/ui/filter_test.go` already declares `lastLine(m tea.Model) string` with the same meaning as the helper this plan's tests needed, so a second declaration did not compile.
- **Fix:** The new tests use the existing `lastLine`; no second helper was added.
- **Files modified:** internal/ui/add_test.go
- **Commit:** c4ac22b

**2. [Rule 1 - Bug] A save that finds the PR already listed marks it saved**
- **Found during:** Task 1 GREEN
- **Issue:** The plan says "insert the PR as today unless findListed finds it". If a refresh listed the PR between the Lookup and the save, skipping the insert would leave the row without its `+` marker and its `Watched` ref until the next refresh.
- **Fix:** In that case the existing row gets `Manual = true` and `Watched` set, and is the row that is selected or named in the flash.
- **Files modified:** internal/ui/add.go
- **Commit:** d8a080f

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug). **Impact:** none on scope; both stay inside the plan's files.

## Known Limits (owned by later plans, not stubs)

- The `a` hint is not in the footer or the help overlay yet, and the prompt has no golden file: plan 04-07.
- At 120 cells a real SSO reason plus its Help (about 115 cells) leaves no room for the typed text and the closing " · esc cancel" is cut. The reason and the start of the Help stay visible; the full text is in the detail view of a saved row (plan 04-05).
- The `+` marker of a row marked Manual by a save that found it already listed relies on the existing renderer; no new render code.

## Known Stubs

None. No placeholder data flows to the UI.

## Threat Flags

None beyond the plan's threat model. T-04-11 (terminal escapes) is covered by `TestAddPasteSanitised`; T-04-12 (256-rune cap, line cut to the pane) by `TestAddTyping` and `TestAddPromptWidths`; T-04-13 (stale list) by `TestStaleFetchAfterAdd`; T-04-14 (silent save) by sequence numbers in `TestAddEscDropsLateResult` and by `TestAddSaveError`. T-04-SC: no new modules.

## Verification

- `go vet ./... && go test ./... -count=1 -race`: all packages pass.
- Task 1 named tests: 8 of 8 PASS. Task 2 named tests: 10 of 10 PASS.
- All acceptance greps print their line; `git diff --exit-code internal/ui/testdata go.mod go.sum` exits 0; `gofmt -l .` is empty.

## Self-Check: PASSED

- Files: internal/ui/add.go, internal/ui/add_test.go, internal/ui/model.go, internal/ui/refresh.go found.
- Commits 51386f9, d8a080f, c4ac22b, 7752d90 found on the branch.

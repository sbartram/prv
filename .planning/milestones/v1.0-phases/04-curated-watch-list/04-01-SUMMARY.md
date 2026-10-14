---
phase: 04-curated-watch-list
plan: 01
subsystem: watch-list
tags: [go, github-graphql, aliased-lookup, watchlist, tracer, bubbletea-v2, tdd]

requires:
  - phase: 01-status-at-a-glance
    provides: discovery query, mapNode-able wireNode, fetchFailing, Sort
  - phase: 02-live-monitoring
    provides: cursor-by-ref rebuild, prompt line pattern, single-flight fetch
provides:
  - "Add prompt (a): paste or type a github.com PR URL or owner/repo#n, one variables-only Lookup before save, row listed with full status and the + marker"
  - "internal/watchlist: versioned JSON store with reload-apply-write Add/Remove and an atomic temp+rename write"
  - "Every refresh reads watchlist.json inside the fetch Cmd, skips saved PRs that discovery already returned, looks up the rest in chunks of 25"
  - "domain: ParseRef, ValidRef, Ref.Key/URL, RefError, Unavail, PR.Auto/Manual/Watched/Unavail, MarkWatched, Combine"
affects: [04-02, 04-03, 04-04, 04-05, 04-06, 04-07]

plan_head_before: 85c56b12b2f8a0f7e074d52436b4cefcffe79e4f
plan_head_after: 72f8688c54d667ba28b28e21d1fd88e41cd6b2c6

actuals:
  tokens: 20000
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Provenance as row properties (Auto, Manual) merged by key, then by node ID, instead of a second list"
    - "Saved entry ref (PR.Watched) kept apart from the canonical display ref (PR.Ref)"
    - "Aliased GraphQL lookup: alias index in the document, owner/repo/number only as variables"
    - "Panic-wrapped sequence-stamped Cmds for lookup and save (copy of fetchDetail)"

key-files:
  created:
    - internal/watchlist/watchlist.go
    - internal/watchlist/watchlist_test.go
    - internal/github/lookup.go
    - internal/ui/add.go
    - internal/domain/watch.go
    - internal/domain/watch_test.go
    - internal/domain/ref_test.go
  modified:
    - internal/domain/pr.go
    - internal/github/client.go
    - internal/ui/model.go
    - internal/ui/keys.go
    - internal/ui/view.go
    - cmd/prv/main.go
    - cmd/prv/main_test.go

key-decisions:
  - "Promote provenance to row properties (PR.Auto, PR.Manual) and keep the saved ref in PR.Watched apart from PR.Ref, so a repo rename cannot split a PR into two rows or make removal miss its entry"
  - "The saved entry is the canonical ref GitHub returned, not the pasted letter case"
  - "A Lookup that fails as a whole fails the whole refresh (D-12 row 1); an unreadable watchlist.json reuses the last list read and adds the warning 'Watch list could not be read', never an empty list"
  - "ParseRef, ValidRef and the unavailable type live in pr.go next to Ref and PR to keep the tracer at 9 files"
  - "The a key is not in ShortHelp/FullHelp yet; plan 04-07 adds the hints and regenerates the goldens once"

patterns-established:
  - "Store keeps no state: every call re-reads the file (D-16), never overwrites a file it cannot parse"
  - "Add prompt replaces the footer in bottom(), so it costs no list row"

requirements-completed: [WATCH-04, WATCH-05]

coverage:
  - id: D1
    description: "Pasting a github.com PR link into the add prompt checks the PR with one variables-only Lookup, saves it atomically and lists it with full status, the + marker and the cursor on it"
    requirement: WATCH-04
    verification:
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestAddEndToEnd"
        status: pass
      - kind: unit
        ref: "internal/domain/ref_test.go#TestParseRef"
        status: pass
    human_judgment: false
  - id: D2
    description: "Saved PRs are listed again after a restart, once each, with full status and the + marker; a saved PR that discovery returned costs no Lookup"
    requirement: WATCH-05
    verification:
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestRestartEndToEnd"
        status: pass
      - kind: unit
        ref: "internal/domain/watch_test.go#TestEverySavedRefOnce"
        status: pass
    human_judgment: false
  - id: D3
    description: "watchlist.json is created 0600 in a 0700 directory, written by temp file and rename, and an unreadable file is never overwritten"
    requirement: WATCH-05
    verification:
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestAddCreatesFile"
        status: pass
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestUnreadableFileIsNeverOverwritten"
        status: pass
    human_judgment: false
  - id: D4
    description: "The add prompt reads well in a real terminal (placement, caret, checking spinner, paste feel)"
    verification: []
    human_judgment: true
    rationale: "No golden or test asserts the look of the prompt line in a live terminal; the golden regeneration is plan 04-07"

duration: 40min
completed: 2026-10-13
status: complete
---

# Phase 04 Plan 01: Tracer - paste a PR link, it is saved, and it comes back after a restart Summary

**Paste a github.com PR link after pressing `a`: prv checks it with one variables-only aliased GraphQL Lookup, saves the canonical ref to `watchlist.json` with an atomic temp-file rename, lists it with full status and a `+` marker, and every later refresh (and restart) merges the saved list with discovery by key and node ID.**

## Performance

- **Duration:** about 40 min
- **Completed:** 2026-10-13
- **Tasks:** 2 (tracer, then restart)
- **Files:** 7 created, 7 modified

## Accomplishments

- **Task 1 (tracer):** `a` opens a one-line prompt in place of the footer. Typed text and `tea.PasteMsg` go through `domain.Clean` (CR and LF become spaces) with a 256-rune cap. `enter` runs `domain.ParseRef`, then `Client.Lookup`, then `watchlist.Store.Add`, then the row joins the list with the cursor on it. `TestAddEndToEnd` drives the real client, a real store in a temp dir and the real model, and asserts exactly one Lookup with variables `{o0: octo, n0: tools, p0: 42}` and none of the pasted values in the query text.
- **Task 2:** `Fetch` sets `Auto` on discovered PRs, reads the saved list through `WatchSource.Load` on every refresh, marks matches (case-insensitive owner/repo) with `MarkWatched`, looks up only the rest in chunks of 25, and merges with `Combine` (key, then node ID). `cmd/prv` builds the store and passes it as `github.Options.Watch` and `ui.Options.Watchlist`. `TestRestartEndToEnd` proves a second client and model on the same file list the saved PR again with exactly `["Watch", "Lookup"]` requests.
- `prFields` and `mapNode` were extracted from the discovery query and mapper so a looked-up PR maps exactly like a discovered one. No existing test assertion or golden file changed (`git diff` of `go.mod`, `go.sum` and `internal/ui/testdata` is empty).

## TDD Gate Compliance

Both tasks ran RED then GREEN. A Go test that does not compile is not a valid RED, so each RED commit carries compile stubs only; every new test then failed on its assertion for the planned behavior (for example `Lookup requests = 0, want exactly 1`, `rest = [...], want [octo/tools#42]`). `workflow.tdd_mode` is off in this project, so the `tdd-red-evidence` classifier (which has no Go report adapter) was not applicable.

| Gate | Commit |
|------|--------|
| RED (task 1) | cb3c149 test(04-01): add failing tests for paste-to-add tracer |
| GREEN (task 1) | b8926d3 feat(04-01): paste a PR link to check, save and list it (tracer) |
| RED (task 2) | 297531b test(04-01): add failing tests for restart and merge of saved PRs |
| GREEN (task 2) | 72f8688 feat(04-01): list saved PRs again after a restart, once each |

No REFACTOR commit was needed. Tracer feedback gate: `workflow.human_verify_mode` is `end-of-phase` and the tracer `<verify>` is automated only, so it was re-run after the commit (all four named tests and the full `-race` suite passed) and expansion went on without a checkpoint.

## Task Commits

1. Task 1 RED: `cb3c149`
2. Task 1 GREEN: `b8926d3`
3. Task 2 RED: `297531b`
4. Task 2 GREEN: `72f8688`

## Deviations from Plan

**1. [Rule 2 - Missing test coverage] Extra test files beyond the plan's file list**
- **Found during:** Task 1 RED
- **Issue:** The task's `<behavior>` list names ParseRef, Ref.Key and Store behavior, but its `<files>` list has no test file for them. A TDD RED needs them.
- **Fix:** Added `internal/domain/ref_test.go` (ParseRef table, Key/URL, ValidRef) and `internal/watchlist/watchlist_test.go` (Path, create modes, duplicate no-op, order, missing and unreadable file, Remove, invalid entries kept).
- **Files modified:** internal/domain/ref_test.go, internal/watchlist/watchlist_test.go
- **Commit:** cb3c149

No other deviations. No auth gates.

## Known Limits (owned by later plans, not stubs)

- A saved PR that closed outside the closed-days window is still dropped by the window filter in `Fetch` (plan 04-05 adds the exemption).
- Adding a PR that discovery already lists does not mark the existing row `Manual` until the next refresh; duplicate, hidden-row and stale-fetch rules are plan 04-03.
- Lookup reasons are two: "not found (deleted, renamed, or no access)" when GitHub sent `NOT_FOUND` for the alias, else "could not load (GitHub error)". SSO and short error text are plan 04-02.
- `Store.Load` warnings and a backup copy of a corrupt file are not shown yet (plan 04-04). An unreadable file is only refused, never overwritten.
- The `a` hint is not in the footer or help overlay (plan 04-07).

## Known Stubs

None. No placeholder data flows to the UI.

## Threat Flags

None beyond the plan's threat model. T-04-01 to T-04-05 are mitigated as planned: variables-only Lookup (asserted in `TestAddEndToEnd`), `domain.Clean` plus a 256-rune cap on typed and pasted text, `mapNode` cleaning at the seam, a file of refs only with 0700/0600 modes, and read-before-write with temp-file rename. T-04-SC: no new modules.

## Verification

- `go vet ./... && go test ./... -count=1 -race`: all packages pass.
- `TestAddEndToEnd`, `TestRestartEndToEnd`, `TestMarkWatched`, `TestCombine`, `TestEverySavedRefOnce`, `TestEndToEndFixture`, `TestFailingNameEndToEnd`, `TestDetailEndToEnd`: PASS.
- All Task 1 and Task 2 acceptance greps print their line; `grep` for `internal/github` or `internal/watchlist` imports in `internal/ui` prints nothing; `gofmt -l .` is empty.

## Self-Check: PASSED

- Files: internal/watchlist/watchlist.go, internal/github/lookup.go, internal/ui/add.go, internal/domain/watch.go, internal/domain/watch_test.go, internal/domain/ref_test.go, internal/watchlist/watchlist_test.go found.
- Commits cb3c149, b8926d3, 297531b, 72f8688 found on the branch.

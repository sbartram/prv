---
phase: 04-curated-watch-list
plan: 04
subsystem: watch-list
tags: [go, json-store, atomic-write, backup, live-sync, startup-warnings, tdd]

requires:
  - phase: 04-curated-watch-list
    provides: "Plan 04-01 store (Add/Remove/Load, atomic write) and Fetch savedRefs with lastRefs"
provides:
  - "watchlist.json that no bad file can empty: hash-named O_EXCL backup of a corrupt, unknown-version or over-1-MiB file, reads that never write, Add over a backed-up file starts fresh"
  - "Cap of 500 watched PRs (Load keeps the first 500 with a warning, Add refuses a 501st), singular/plural skip warning"
  - "Rename seam on Store; a failed write leaves the old file byte-identical and no .watchlist-*.tmp"
  - "watchWarnings: watch-list problems shown once at startup in the footer and ? overlay after config warnings"
  - "Fetch tests with a real Store proving two-pane live sync and a kept list on read failure"
affects: [04-05, 04-06, 04-07]

plan_head_before: a9d9c82e0b6988b3a9ad2cc0a6add915a01c6f5e
plan_head_after: 563900095cc7b549ac92102246260e5917b729bc

actuals:
  tokens: 12000
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Read path copies a bad file aside once (O_EXCL, name from SHA-256) and never rewrites it"
    - "Write path = read with backup, apply one change, temp file in the same dir, rename through a test seam"
    - "Startup problems become ui.Options.Warnings like config warnings; prv never exits for them"

key-files:
  created:
    - internal/github/watchsource_test.go
  modified:
    - internal/watchlist/watchlist.go
    - internal/watchlist/watchlist_test.go
    - cmd/prv/main.go
    - cmd/prv/main_test.go

key-decisions:
  - "The backup hash is the SHA-256 of what was read (first 1 MiB + 1 byte for a huge file) while the copy streams the whole file from a fresh open"
  - "A partial backup is removed on a copy error so an O_EXCL hit always means a complete copy"
  - "internal/github/client.go needed no change: savedRefs already kept lastRefs on any Load error, ErrCorrupt included"
  - "Fetch tests reuse lookupPRNode, lookupBody and lookupRequest from lookup_test.go (already committed by 04-02) for the Lookup answer"

requirements-completed: [WATCH-05, WATCH-06]

coverage:
  - id: D1
    description: "A corrupt, unknown-version or oversized watchlist.json is copied byte for byte to watchlist.json.corrupt-HASH8 once, never rewritten by a read, and Add over it starts a fresh version-1 file"
    requirement: WATCH-05
    verification:
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestCorruptBackup"
        status: pass
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestUnknownVersion"
        status: pass
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestTooLarge"
        status: pass
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestAddOverCorrupt"
        status: pass
    human_judgment: false
  - id: D2
    description: "Hand-edited entries that are not PR references stay in the file and are skipped with one warning; at most 500 PRs are watched"
    requirement: WATCH-05
    verification:
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestBadEntriesKept"
        status: pass
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestMaxEntries"
        status: pass
    human_judgment: false
  - id: D3
    description: "Writes are all-or-nothing and safe across two panes: a failed rename leaves the old bytes and no temp file; interleaved Add and Remove from two stores lose nothing; removing a missing ref does not write"
    requirement: WATCH-06
    verification:
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestNoTempLeftovers"
        status: pass
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestTwoStoresInterleave"
        status: pass
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestRemoveMissingNoWrite"
        status: pass
    human_judgment: false
  - id: D4
    description: "Each refresh re-reads the file: another pane's add and remove show up on the next refresh, a broken file mid-session keeps the last list and adds the warning, a saved PR that discovery found costs no Lookup"
    requirement: WATCH-05
    verification:
      - kind: integration
        ref: "internal/github/watchsource_test.go#TestFetchRereadsWatchList"
        status: pass
      - kind: integration
        ref: "internal/github/watchsource_test.go#TestFetchKeepsRefsOnReadFailure"
        status: pass
      - kind: integration
        ref: "internal/github/watchsource_test.go#TestFetchSavedFoundByDiscovery"
        status: pass
    human_judgment: false
  - id: D5
    description: "Startup watch-list problems (corrupt file with its backup name, unreadable file, skipped entries, the cap) are collected once as 'watch list: ...' warnings"
    requirement: WATCH-05
    verification:
      - kind: unit
        ref: "cmd/prv/main_test.go#TestWatchWarnings"
        status: pass
    human_judgment: false
  - id: D6
    description: "The startup warnings read well in the footer and the ? overlay of a real terminal"
    verification: []
    human_judgment: true
    rationale: "Text placement and wording in a live pane are not asserted by a test; the Options.Warnings display path itself is the 8 s footer and help list from the config warnings and is unchanged"

duration: 25min
completed: 2026-10-13
status: complete
---

# Phase 04 Plan 04: The saved list survives anything Summary

**watchlist.json now survives a corrupt, hand-edited, oversized or unreadable file and two prv panes: a bad file is copied once to `watchlist.json.corrupt-HASH8` (O_EXCL, SHA-256 name) and never rewritten by a read, entries cap at 500, writes go through a rename seam that cleans its temp file, and startup problems show once as `watch list: ...` warnings.**

## Performance

- **Duration:** about 25 min
- **Completed:** 2026-10-13
- **Tasks:** 2 (both TDD)
- **Files:** 1 created, 4 modified (5 touched in 775 inserted / 45 removed lines)

## Accomplishments

- **Store hardening (Task 1).** `read()` returns a `*CorruptError` (wraps `ErrCorrupt`, names the backup) for bad JSON, a version other than 1, or a file over 1 MiB, after `backup()` copies the whole file with `O_EXCL` and mode 0600. A copy failure is a plain `could not back up watchlist.json: ...` error. `Add` and `Remove` start a fresh version-1 file only after a successful backup; any other read error (permission denied) writes nothing. `Load` keeps the first 500 valid entries (`MaxEntries`), warns "only the first 500 PRs are watched", and uses the singular "1 entry skipped". `Add` refuses a 501st with "the watch list is full (500 PRs)". `Remove` of a missing ref writes nothing (bytes and mtime kept).
- **Live sync proven (Task 2).** Four Fetch tests with real stores in `t.TempDir()`: another pane's add and remove appear on the next refresh with Lookup variables holding only the remaining ref; a file overwritten with `{broken` keeps the last list, adds "Watch list could not be read", leaves the file as it is and makes one backup; no source means no Lookup; a saved `ACME/WEB#77` that discovery returned costs no Lookup and is Auto and Manual with Watched `{ACME/WEB, 77}`.
- **Startup warnings.** `watchWarnings(store)` calls `Load` once (making the backup before the TUI starts) and returns `watch list: ...` lines. `run()` sets `ui.Options.Warnings` to the config warnings followed by these. prv never exits for a watch-list problem.

## TDD Gate Compliance

| Gate | Commit |
|------|--------|
| RED (task 1) | ac5a491 test(04-04): add failing tests for backup, caps and two-pane safety of the store |
| GREEN (task 1) | cf8128b feat(04-04): a broken watchlist.json is copied aside once and never loses the saved list |
| RED (task 2) | 1e523db test(04-04): add failing test for startup watch-list warnings and live-sync fetch tests |
| GREEN (task 2) | 5639000 feat(04-04): show watch-list problems once at startup and carry on |

RED evidence (semantic assessment): as in 04-01, a Go test that does not compile is not a valid RED, so each RED commit carries compile stubs only (`ErrCorrupt`, `CorruptError`, `MaxEntries`, `Store.rename`; a `watchWarnings` that returns nil). Every new test then failed on its planned assertion: Task 1 `Load error = watchlist.json is not a valid watch list, want one wrapping ErrCorrupt`, `1 entries skipped` vs "1 entry", `Load = 501 refs, want 500`, `Add returned no error when the rename failed`, `Add = false, ... not a valid watch list, want true`; Task 2 `corrupt file: warnings = [], want one 'watch list: ...'`. The `tdd-red-evidence` classifier has no Go report adapter and `workflow.tdd_mode` is off. Note on the Task 2 Fetch tests: they already pass at the RED commit because Task 1 produced the behavior they check (the backup made by `Load`) and `savedRefs` from 04-01 already kept `lastRefs`; they are characterization tests of the Task 1 and 04-01 behavior, and `TestFetchKeepsRefsOnReadFailure` would have failed before Task 1. No REFACTOR commit.

## Task Commits

1. Task 1 RED: `ac5a491`
2. Task 1 GREEN: `cf8128b`
3. Task 2 RED: `1e523db`
4. Task 2 GREEN: `5639000`

## Files Created/Modified

- `internal/watchlist/watchlist.go` - ErrCorrupt, CorruptError, MaxEntries, read/backup/validRefs/readForWrite, rename seam, package doc
- `internal/watchlist/watchlist_test.go` - the 15 planned store tests plus helpers prefixed `wl`
- `internal/github/watchsource_test.go` (new) - `wsTransport`, `wsLookupBody`, `wsClient`, `wsStoreAt`, `wsFind` and the four Fetch tests
- `cmd/prv/main.go` - `watchWarnings` and the Warnings wiring in `run()`
- `cmd/prv/main_test.go` - `TestWatchWarnings`

## Decisions Made

- Backup name hash is the SHA-256 of the bytes read (for a file over 1 MiB, its first 1 MiB + 1 byte), while the copy streams the whole file from a fresh open, so a large file is saved in full.
- A partial backup is deleted when the copy fails, so an existing `corrupt-HASH8` file always means a complete copy.
- `internal/github/client.go` was not changed: `savedRefs` already keeps `lastRefs` on any Load error (including `ErrCorrupt`) and never replaces it with nil. The plan allowed this ("only where a test shows a gap"); no test showed one.
- At startup a corrupt file gives an empty saved list plus the startup warning; the per-refresh "Watch list could not be read" snapshot warning also shows while the file stays unreadable. This matches the plan's flagged assumption.

## Deviations from Plan

**1. [Rule 1 - Bug/obsolete test] Removed 04-01 `TestUnreadableFileIsNeverOverwritten`**
- **Found during:** Task 1 GREEN
- **Issue:** The 04-01 test asserted that `Add` over a junk (invalid JSON) file returns an error and leaves the file untouched. This plan's locked behavior (D-16, `TestAddOverCorrupt`) is the opposite for a corrupt file: copy it to a backup, then start a fresh file. The old test contradicts the new contract.
- **Fix:** Deleted it. Its two real guarantees are kept by `TestAddOverCorrupt` (the old bytes live on in the backup) and `TestUnreadableNotOverwritten` (a file prv cannot read is never written over, no backup made).
- **Files modified:** internal/watchlist/watchlist_test.go
- **Committed in:** cf8128b

**2. [Rule 3 - Blocking] Fetch tests reuse existing Lookup test helpers**
- **Found during:** Task 2
- **Issue:** The plan has `wsLookupBody` build the Lookup answer node by node. `lookupPRNode`, `lookupBody` and `lookupRequest` in `lookup_test.go` (committed by 04-02 before this plan ran) build the same JSON.
- **Fix:** `wsLookupBody` calls them instead of repeating the structure. The `ws` helper names the plan requires are all present, and no name collides.
- **Files modified:** internal/github/watchsource_test.go
- **Committed in:** 1e523db

**Total deviations:** 2 auto-fixed (1 obsolete test, 1 helper reuse). **Impact:** none on behavior; no scope creep.

## Issues Encountered

None.

## Known Stubs

None. No placeholder data flows to the UI.

## Threat Flags

None beyond the plan's threat model. T-04-15 (1 MiB cap, ValidRef, 500 cap), T-04-16 (O_EXCL backup, reads never write, refusal over unreadable files), T-04-17 (0700/0600, refs only, `TestRoundTrip` key check) and T-04-19 (path only from XDG or home) are mitigated as planned; T-04-18 (millisecond lost-update window between re-read and rename) is accepted as planned. T-04-SC: `git diff --exit-code go.mod go.sum` exits 0; `crypto/sha256` and `encoding/hex` are stdlib.

## Requirements

`WATCH-05` (saved PRs survive restart and a broken file) is fully delivered with 04-01. `WATCH-06` (remove a manually added PR) is delivered here only on the store side (idempotent, pane-safe `Remove`); the remove key and its confirmation are plans 04-06 and 04-07, so it is not complete yet.

## Verification

- `go vet ./... && go test ./... -count=1 -race`: all packages pass.
- The Task 1 named run prints 16 `--- PASS: Test` lines (the plan's 14 minimum); the Task 2 named run prints 5.
- `git diff --exit-code go.mod go.sum`: exit 0. `gofmt -l cmd internal`: empty.
- Task 1 greps (`var ErrCorrupt`, `MaxEntries = 500`, `O_EXCL`, `sha256`, `func TestTwoStoresInterleave`) and Task 2 greps (`func watchWarnings(s *watchlist.Store) []string`, 2 uses of `watchWarnings(`, `func wsLookupBody`) all print their lines.

## Self-Check: PASSED

- Files found: internal/github/watchsource_test.go, internal/watchlist/watchlist.go, internal/watchlist/watchlist_test.go, cmd/prv/main.go, cmd/prv/main_test.go.
- Commits ac5a491, cf8128b, 1e523db, 5639000 found on the branch.

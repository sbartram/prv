---
phase: 05-address-tech-debt-watch-list-locking-live-smoke
verified: 2026-10-14T04:30:00Z
status: human_needed
score: 5/5 must-haves verified
covered_files:
  - ".planning/phases/05-address-tech-debt-watch-list-locking-live-smoke/05-01-PLAN.md"
  - ".planning/phases/05-address-tech-debt-watch-list-locking-live-smoke/05-01-SUMMARY.md"
  - ".planning/phases/05-address-tech-debt-watch-list-locking-live-smoke/05-02-PLAN.md"
  - ".planning/phases/05-address-tech-debt-watch-list-locking-live-smoke/05-02-SUMMARY.md"
  - ".planning/phases/05-address-tech-debt-watch-list-locking-live-smoke/05-03-PLAN.md"
  - ".planning/phases/05-address-tech-debt-watch-list-locking-live-smoke/05-03-SUMMARY.md"
  - "README.md"
  - "cmd/prv/main_test.go"
  - "go.mod"
  - "internal/github/live_test.go"
  - "internal/ui/add.go"
  - "internal/ui/model.go"
  - "internal/ui/remove.go"
  - "internal/ui/show_test.go"
  - "internal/watchlist/lock.go"
  - "internal/watchlist/watchlist.go"
covered_digest: "v3:sha256:1728a32eb0225e014298070edd03670eec53205eb1722b721fe65df1eeddb4f6"
behavior_unverified: 0
overrides_applied: 0
warnings:
  - id: WR-01
    note: "Live test skips on expired CI data before Review/State/Merge assertions run. Matches the D-13 plan text; latent weakness (earliest expiry about Oct 2027), not a goal failure."
  - id: WR-02
    note: "RemoveRef over an unparseable file returns Result{} (as 05-01-PLAN specifies) and the UI flashes 'removed'. Edge case; not a goal failure."
human_verification:
  - test: "05-UAT #1: failure marker and recovery at 80 and 40 columns (Wi-Fi off, wait one refresh, on again)"
    expected: "Amber 'refresh failed HH:MM (network) · data from HH:MM:SS' marker, rows stay, header returns to 'updated ...', footer readable at 40 columns"
    why_human: "Real network failure and real terminal rendering"
  - test: "05-UAT #2: PRV_SHOW=1 go test ./internal/ui -run '^TestShowRateWarning$' -count=1 -v, in the usual terminal and in tmux/herdr"
    expected: "Yellow api warning legible as a warning; faint 'api 999' footer quiet but legible"
    why_human: "Colour and legibility are visual"
  - test: "05-UAT #3: keys 1/2/3, / filter prompt, paste digits and text"
    expected: "Indicator toggles; prompt takes digits only; list narrows; esc clears"
    why_human: "Real terminal key and paste behaviour"
  - test: "05-UAT #4: o opens a PR in a real browser; BROWSER=nope shows 'could not open the browser'"
    expected: "Browser opens; bad launcher flashes the message and does not corrupt the screen"
    why_human: "External browser integration"
  - test: "05-UAT #5: ? help overlay with and without a config warning, and in a short pane"
    expected: "Overlay shows keys, legend, settings, api budget, warnings; any key returns; q does not quit"
    why_human: "Visual layout in a real terminal"
  - test: "05-UAT #6: config.toml edits (misspelt key, bad value, show/closed_days)"
    expected: "Unknown-key footer note for about 8 s; exit status 2 on a bad value; header follows the file"
    why_human: "Real file edit and process restart"
  - test: "05-UAT #7: REVIEW and CI badges of added public PRs compared with github.com"
    expected: "Badges match github.com; no checks is never passing; approved without a review rule is never required"
    why_human: "Comparing with the live web page by eye; open-PR CI states drift and are not pinned"
  - test: "05-UAT #8: MERGE column of open PRs (mergeable, blocked, behind, conflicts, draft) compared with the github.com merge box"
    expected: "Each MERGE cell agrees with the merge box; merged/closed rows blank"
    why_human: "Open-PR merge state drifts, so it cannot be pinned in the live test"
  - test: "Two real prv panes (macOS and Linux) add and remove at the same time, once with a held lock"
    expected: "Every change survives; a writer that waits over 3 s shows 'could not save (another prv is writing the list)'; the other pane keeps refreshing"
    why_human: "Real multi-instance, long-running behaviour; flock semantics on Linux filesystems were not run (cross-built only)"
---

# Phase 5: Address tech debt (watch-list locking, live smoke) Verification Report

**Phase Goal:** Close the v1 tech debt from the milestone audit: two prv instances can write the watch list at the same time without losing a change, a broken or newer-version watchlist.json is never lost silently, the add prompt never lies about a save, and prv's state, review, CI and MERGE mapping is checked against real github.com (an opt-in live test plus a manual UAT checklist). No new user-facing features.
**Verified:** 2026-10-14
**Status:** human_needed
**Re-verification:** No, initial verification

## Goal Achievement

The code evidence is complete: every roadmap success criterion is met in the codebase and by tests I ran myself. No gaps. The status is `human_needed` only because the phase deliberately ships a manual checklist (05-UAT.md, 8 items, all pending) and because multi-instance and Linux flock behaviour were not exercised by hand.

### Observable Truths (ROADMAP success criteria are the contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Two prv panes adding/removing at once keep every change; 8 real processes each add a different PR and all 8 survive; a writer waits at most 3 s then says another prv is writing; refreshes never wait | VERIFIED | `internal/watchlist/lock.go` takes `unix.Flock(LOCK_EX\|LOCK_NB)` on `watchlist.json.lock`, polls every 10 ms for `defaultLockWait` = 3 s, then returns `ErrBusy`. `AddRef` and `RemoveRef` call `s.lock()` before `readForWrite` and defer unlock past `write`. `Load` and `read` take no lock. `TestMultiProcessAdd`, `TestLockBusy`, `TestLoadWhileLocked` and `TestLockFileKept` pass (3 runs). Mutation proof: with the lock a no-op via `go test -overlay`, `TestMultiProcessAdd` fails 3 of 3 runs ("Load = 1 refs ..., want 8"). |
| 2 | A watchlist.json with another version is never written over (add/remove refuse with "watchlist.json is version N; this prv writes version 1"); an unparseable or oversized file is backed up completely before it gets its final name, and the add that replaces it flashes "previous list saved to watchlist.json.corrupt-XXXXXXXX" | VERIFIED | `classify` reads the version before decoding entries; a non-1, missing or null version returns a `*VersionError`, and `readForWrite` returns it without a write path. `backup` writes a `.corrupt-*.tmp` file, syncs and closes it, then `os.Link`s it to `.corrupt-HASH8` (no clobber, `fs.ErrExist` counts as done). `AddRef` returns `Result{Changed, StartedOver}`. `startedOverNotice` in `ui/add.go` builds "previous list saved to ..." with `warnFor`. Tests: `TestAddOverCorruptEndToEnd` and `TestAddVersionMismatchEndToEnd` (real store plus real ui, in `cmd/prv/main_test.go`), `TestAddStartedOverNotice`, and the watchlist unit tests, all pass. |
| 3 | Once a save starts, esc does not cancel it; the prompt shows "saving ..." without "esc cancel" | VERIFIED | `addKey` returns nil for every key but ctrl+c when `m.add.saving`; `addPaste` ignores paste while saving; `model.go` keeps the spinner ticking (`m.add.saving`). `saving` is cleared on every exit of `applyAddSaved`. `TestAddEscIgnoredWhileSaving`, `TestAddSavingLine`, `TestAddSpinnerWhileSaving`, `TestAddEscDropsLateResult` and the 4 `add-saving_*.golden` files exist and pass. |
| 4 | A failed add or remove says why (list full, another prv writing, another version, permission denied, disk full) and never shows a path or raw OS text | VERIFIED | `saveReason` maps `VersionError`, `ErrBusy`, `fs.ErrPermission`, `ENOSPC`, default "write failed", then `domain.Clean`. `applyAddSaved` handles `ErrFull` with "; remove a PR first"; `applyRemoved` flashes "could not update (REASON)". `TestAddSaveReasons` passes. |
| 5 | `PRV_LIVE=1 go test ./internal/github -run TestLive -count=1 -v` checks state, review, CI and MERGE for pinned public PRs while plain `go test ./...` stays offline and green; 05-UAT.md holds the manual checks | VERIFIED | I ran it against real github.com: `--- PASS: TestLive` with all 11 pins passing (10 mapped values plus the not-found pin), no skips. Without `PRV_LIVE` it prints `--- SKIP: TestLive` before any token lookup. `go build`, `go vet`, `go test ./...` are green. `TestReviewVerdict` proves the drift/bug classification offline. 05-UAT.md exists with 8 pending checks. README has the Development section with the exact command and the macOS/Linux-only note. |

**Score:** 5/5 truths verified (0 behavior-unverified)

### Plan-level must-haves (spot checks beyond the roadmap)

| Must-have | Status | Evidence |
|-----------|--------|----------|
| D-08: `golang.org/x/sys v0.47.0` is in the direct require block; the module builds for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 | VERIFIED | `go.mod` line 13 is in the first `require` block. All four `GOOS/GOARCH` builds ok. `go mod tidy -diff` is clean. `gofmt -l` is empty. |
| D-11: lock file is `watchlist.json.lock`, mode 0600, directory 0700, never deleted | VERIFIED | `lock.go` `OpenFile(..., 0o600)` and `MkdirAll(dir, 0o700)`; `TestLockFileKept` passes. |
| Version 0, null, non-integer and absent version are all refused (no start-over) | VERIFIED | `classify` sets `VersionError{Missing: true}` for absent/null/non-integer and `Found: v` otherwise; `read` copies to a backup and sets `CorruptError.Version`. |
| D-16: `TestShowRateWarning` is silent by default and prints the low-budget screen with `PRV_SHOW=1` | VERIFIED | Ran with `PRV_SHOW=1`: printed "api 999/5000 · resets 15:02" in the yellow header at 80 columns. |
| 04-REVIEW-DISPOSITION.md records WR-01/02/03/05 and IN-02 as fixed, WR-04/IN-01/IN-03 as deferred, open 0 | VERIFIED | File shows exactly this, `open: 0`. |
| 05-UAT.md lists exactly 8 pending manual checks | VERIFIED | Summary block shows total 8, pending 8. |

### Code review warnings weighed against the goal

- **WR-02 (RemoveRef over an unparseable file returns `Result{}`, UI flashes "removed", the row can come back):** Confirmed in `watchlist.go` (`RemoveRef` returns `Result{}, nil` when nothing was removed) and `ui/remove.go` (`applyRemoved` removes the row on any non-error result). `05-01-PLAN.md` specifies exactly this behaviour ("writes nothing and returns Result{} with no notice"), and `watchlist_test.go:452` pins it, so it matches the plan and does not fail a must-have. It does not break the goal. The goal's "add prompt never lies about a save" concerns add, which is correct. "Never lost silently" holds: the broken file is untouched and a backup exists. The path needs a saved row to be on screen while the file becomes unparseable (a hand edit while prv runs). The remaining flaw is a misleading "removed" flash in that rare case. **WARNING, not a blocker.** Suggest a follow-up: return the `CorruptError` from `RemoveRef`, or surface the backup name.
- **WR-01 (live test skips on expired CI data before Review/State/Merge assertions):** Confirmed at `live_test.go` lines 254-262: the `Skipf` for "GitHub expired the check data" comes before the `Review`, `State` and `Merge` assertions. This also follows the plan's D-13 text, which makes an expired CI badge a skip. It does not undermine a must-have: today all 11 pins pass with no skip (I ran the live test). The weakness is latent: from about Oct 2027, when pins 14327 and 14440 expire, their Review/State assertions would be skipped with a re-pin message. **WARNING, not a blocker.** Suggest a follow-up: move the expiry skip after the Review/State/Merge error checks.
- **Info items (IN-01..IN-04):** none touches a must-have. IN-01 (directory sync after link) is a low-probability durability gap. IN-02 is that `afterRead` is wired only into `AddRef`. IN-03 and IN-04 are test hygiene.
- **Bookkeeping:** `05-REVIEW-DISPOSITION.md` still records all 6 findings as `open` (never triaged). Triage them (fixed/deferred) before closing the phase.

### Requirements Coverage

Phase-level requirement IDs are null (ROADMAP: "TBD", no new requirements; tech debt). I cross-referenced the IDs from the PLAN frontmatter. REQUIREMENTS.md has no Phase 5 mappings, so there are no orphaned requirements.

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| WATCH-04 | 05-02 | Add a PR by URL; invalid or inaccessible PRs show an error and are not saved | SATISFIED | Add flow, saving state and specific failure reasons; tests pass |
| WATCH-05 | 05-01, 05-02 | Manually added PRs are saved locally and reappear after restart | SATISFIED | Locked, atomic, crash-safe store; multi-process test |
| WATCH-06 | 05-01, 05-02 | Remove a manually added PR | SATISFIED | `RemoveRef` under the lock; "could not update (REASON)" (see WR-02 for the one edge case) |
| LIST-02 | 05-03 | State: Open, Merged, Closed | SATISFIED | Live test pins State for 10 PRs (passed live). Manual check is UAT #7. |
| LIST-03 | 05-03 | Review status; repos without required reviews never show "required" by mistake | SATISFIED | Live pin komoot/photon#254 passed live |
| LIST-05 | 05-03 | One CI badge; no checks is never passing | SATISFIED | CI pins (none, passing, running, failing) passed live |
| LIST-06 | 05-03 | Merge-readiness | SATISFIED (merged/closed blank only) | Live test asserts blank MERGE for merged/closed PRs; open-PR states go to UAT #8 |
| LIST-09 | 05-03 | Unavailable row with a reason | SATISFIED | Not-found pin `cli/cli#2147483647` passed live |
| RFSH-04 | 05-03 | Failure indicator keeps last data | NEEDS HUMAN | UAT #1 |
| RFSH-05 | 05-03 | Rate limit shown | NEEDS HUMAN | UAT #2 (render helper verified) |
| FILT-01 | 05-03 | Toggle Open/Merged/Closed | NEEDS HUMAN | UAT #3 |
| FILT-03 | 05-03 | Type a number to narrow the list | NEEDS HUMAN | UAT #3 |
| NAV-02 | 05-03 | `o` opens the PR | NEEDS HUMAN | UAT #4 |
| NAV-03 | 05-03 | `?` shows key bindings | NEEDS HUMAN | UAT #5 |
| CONF-01 | 05-03 | Settings read from config.toml | NEEDS HUMAN | UAT #6 |

Every ID in the PLAN frontmatter exists in REQUIREMENTS.md and is accounted for. The "NEEDS HUMAN" rows are re-verifications that the phase intentionally leaves as the 05-UAT checklist. They were `Complete` in Phases 2 and 4, so the status does not regress.

### Anti-Patterns Found

No TBD, FIXME or XXX markers in the phase files (`XXXXXXXX` in backup-name text is not a marker). No stubs, hollow props or static returns on the paths checked. The UI result flows from the real store (`Result.Changed` and `Result.StartedOver` copied into the messages).

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full offline suite | `go build ./... && go vet ./... && go test ./...` | all packages ok | PASS |
| Lock and multi-process | `go test ./internal/watchlist -run 'TestMultiProcessAdd\|TestLock\|TestLoadWhileLocked' -count=3` | pass | PASS |
| Lock proof (mutation) | same test with a no-op `lock()` via `-overlay` | fails 3 of 3 ("Load = 1 refs ..., want 8") | PASS |
| UI and end-to-end | `go test ./cmd/prv ./internal/ui -run 'AddOverCorrupt\|AddVersionMismatch\|Saving\|StartedOver\|SaveReasons\|EscDrops'` | pass | PASS |
| Live test, opt-out | `go test ./internal/github -run '^TestLive$' -v` | SKIP, no network | PASS |
| Live test, opt-in | `PRV_LIVE=1 go test ./internal/github -run '^TestLive$' -count=1 -v` | PASS, 11 of 11 pins, no skips | PASS |
| Cross-build | `GOOS/GOARCH` linux and darwin, amd64 and arm64 | all ok | PASS |

### Human Verification Required

See the `human_verification` list in the frontmatter: 05-UAT.md items 1-8, plus one multi-instance check of two real panes on macOS and Linux. Items 1-6 re-verify the Phase 2 UI requirements. Items 7-8 compare badges and MERGE with github.com by eye.

### Gaps Summary

No gaps. The phase goal is met in code: the lock, the crash-safe backup, the version refusal, the saving state, the specific failure reasons and the live test all exist, are wired end to end, and were exercised by tests I ran (including a mutation check of the lock and a real run of the live test against github.com). Two code-review warnings (WR-01 and WR-02) remain open. Neither is a must-have failure, and both are suggested follow-ups. The remaining work is human: the 8 pending UAT items.

---

_Verified: 2026-10-14_
_Verifier: Claude (gsd-verifier)_

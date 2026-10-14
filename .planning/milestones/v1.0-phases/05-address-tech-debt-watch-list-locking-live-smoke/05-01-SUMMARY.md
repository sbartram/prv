---
phase: 05-address-tech-debt-watch-list-locking-live-smoke
plan: 01
subsystem: watchlist-store
tags: [go, flock, file-lock, multi-process-test, crash-safe-backup, json-store]

requires:
  - phase: 04-curated-watch-list
    provides: "watchlist.Store with Add/Remove, read/backup/write, ui.Watchlist"
provides:
  - "Exclusive flock on watchlist.json.lock around the whole read-modify-write (ErrBusy after 3 s)"
  - "Crash-safe backup: temp file, Sync, no-clobber os.Link; name = SHA-256 of the bytes copied"
  - "Version classification: version != 1 (0 and absent included) is a *VersionError, never written over"
  - "AddRef/RemoveRef returning Result{Changed, StartedOver}; Add/Remove kept as wrappers"
  - "ErrFull sentinel, CorruptError.Version"
affects: [05-02 UI notice and error wording, 05-03 live smoke]

plan_head_before: a9285cd36a74a06d362f8eebb69b41685d5126b7
plan_head_after: 5a6e045006ff0e07c7c62ebd21ff5ac8b84fcd40

actuals:
  tokens: 9000
  tasks: 2
  commits: 4

tech-stack:
  added: ["golang.org/x/sys v0.47.0 (moved from indirect to direct; go.sum unchanged)"]
  patterns:
    - "Advisory flock on a sibling lock file, polled with LOCK_NB against a deadline"
    - "Test binary re-executed as child processes behind a file start barrier"
    - "Unexported test hook (Store.afterRead) to hold the critical section open"
    - "go test -overlay mutation run: prove a test fails when the fix is removed, without touching the tree"

key-files:
  created:
    - internal/watchlist/lock.go
    - internal/watchlist/lock_test.go
  modified:
    - internal/watchlist/watchlist.go
    - internal/watchlist/watchlist_test.go
    - go.mod

key-decisions:
  - "x/sys/unix.Flock (no build tags, macOS and Linux only) over syscall; only the go.mod line moves"
  - "Lock wait 3 s, poll 10 ms; reads never lock"
  - "Version is classified from a map[string]json.RawMessage before entries are decoded, so a future version with undecodable entries is still a mismatch"
  - "Backup link failures other than EEXIST fail closed: no copy under the final name, no write"

requirements-completed: [WATCH-05, WATCH-06]

coverage:
  - id: D1
    description: "Eight real prv processes adding at once all keep their entry; the same test fails 5 of 5 runs with the lock removed"
    requirement: WATCH-05
    verification:
      - kind: integration
        ref: "internal/watchlist/lock_test.go#TestMultiProcessAdd"
        status: pass
    human_judgment: false
  - id: D2
    description: "Writers wait at most 3 s then get ErrBusy with the file unchanged; Load never waits; the 0600 lock file is kept"
    requirement: WATCH-05
    verification:
      - kind: unit
        ref: "internal/watchlist/lock_test.go#TestLockBusy, TestLoadWhileLocked, TestLockFileKept"
        status: pass
    human_judgment: false
  - id: D3
    description: "Backup is published only when complete and its name matches its bytes (also over 1 MiB); a link failure writes nothing"
    requirement: WATCH-06
    verification:
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestBackupPublishedComplete, TestBackupLinkFails, TestTooLarge"
        status: pass
    human_judgment: false
  - id: D4
    description: "A JSON object whose version is not 1 (2, 0, null, string, absent) is refused byte-identical; unparseable or oversized files start over and AddRef reports the backup name"
    requirement: WATCH-06
    verification:
      - kind: unit
        ref: "internal/watchlist/watchlist_test.go#TestVersionMismatchRefused, TestNonObjectStartsOver, TestAddOverCorrupt, TestRemoveOverCorrupt"
        status: pass
    human_judgment: false
  - id: D5
    description: "Linux behaviour of flock polling and link(2) (run only on darwin/arm64 here)"
    verification: []
    human_judgment: true
    rationale: "Cross-compiled and vetted for linux/amd64 and linux/arm64 but not executed on a Linux host (RESEARCH A7)"

duration: 4min
completed: 2026-10-14
status: complete
---

# Phase 05 Plan 01: Watch list locking and honest bad-file handling Summary

**Exclusive flock on watchlist.json.lock around every add and remove (proved by 8 real processes and a mutation run), a crash-safe hash-named backup, and a version-mismatch refusal that never overwrites a v0, v2 or version-less file.**

## Performance

- **Duration:** about 4 min of wall clock (2026-10-14T03:31Z to 03:35Z)
- **Tasks:** 2 (TDD: 4 commits)
- **Files modified:** 5 (703 insertions, 80 deletions)

## Accomplishments

- WR-01: `Store.lock()` takes `LOCK_EX|LOCK_NB` on `watchlist.json.lock`, polls every 10 ms for up to 3 s, then returns `ErrBusy`. Add and Remove hold it from before `readForWrite` until after the rename. Load and read never lock.
- WR-02: `backup(data, rest)` writes a `.corrupt-*.tmp` file, syncs, closes, then publishes it with `os.Link` (seam `Store.link`). The name is the first 8 hex digits of the SHA-256 of the bytes copied, including the tail of an oversized file via the same open handle. The path is never re-opened.
- WR-03: `classify` decodes into `map[string]json.RawMessage` and reads `version` first. Not 1 (including 0, null, string, absent) gives a `*VersionError`, carried in `CorruptError.Version`; `readForWrite` returns it and nothing is written. Non-JSON, non-object, oversized and undecodable version-1 files start over after the backup.
- `AddRef`/`RemoveRef` return `Result{Changed, StartedOver}`; `Add`/`Remove` stay as `(bool, error)` wrappers, so `ui.Watchlist` and the other call sites compile unchanged. `ErrFull` is the IN-02 sentinel.

## Mutation proof (D-07)

With both `s.lock()` calls replaced by a no-op in a `go test -overlay` copy (working tree untouched), `TestMultiProcessAdd -count=5 -race` failed 5 of 5 runs:

```
Load = 1 refs [o/r#4], want 8 (o/r#1..o/r#8)
Load = 1 refs [o/r#2], want 8 (o/r#1..o/r#8)
Load = 1 refs [o/r#1], want 8 (o/r#1..o/r#8)
Load = 1 refs [o/r#8], want 8 (o/r#1..o/r#8)
Load = 1 refs [o/r#6], want 8 (o/r#1..o/r#8)
```

(An earlier run of the same mutation, after Task 1, gave `[o/r#8] [o/r#8] [o/r#7] [o/r#3] [o/r#8]`, also 5 of 5.) The real code passes `TestMultiProcessAdd` with `-race` (about 1.5 s).

## TDD Gate Compliance

- **RED (Task 1), `fc94a57`:** tests for the lock plus compile stubs (`lock()` returns a no-op). Failed on the planned assertions: `TestMultiProcessAdd` "Load = 1 refs [o/r#6], want 8", `TestLockBusy` "want ErrBusy", `TestLockFileKept` and `TestAddCreatesFile` (no lock file), `TestPermissions`. Semantic assessment: each target test ran and failed on the intended assertion; no load, import or fixture fault. `TestLoadWhileLocked` passed already, as it should for a no-op lock.
- **GREEN (Task 1), `4a6c1ed`:** real flock in `lock.go`, `go.mod` promotion.
- **RED (Task 2), `802a6bc`:** tests plus API stubs (`Result`, `VersionError`, `ErrFull`, `AddRef`/`RemoveRef` delegating to the old code, `link` field). Failed: `TestTooLarge` (name hash did not match the backup's bytes), `TestAddOverCorrupt` (no StartedOver), `TestBackupPublishedComplete`, `TestBackupLinkFails` (old code wrote a fresh file and made the backup), all 7 `TestVersionMismatchRefused` subtests, all 5 `TestNonObjectStartsOver` subtests, `TestMaxEntries` (no `ErrFull`). `TestRemoveOverCorrupt` and `TestAddRefRemoveRefResult` passed against the stubs because the old behaviour already matched.
- **GREEN (Task 2), `5a6e045`:** the implementation. No REFACTOR commit was needed.
- The `gsd_run check tdd-red-evidence` classifier was not run: `workflow.tdd_mode` is not set in `.planning/config.json`, and Go's default output is not one of its supported report formats. The RED runs were inspected by hand as described above.

## Task Commits

1. **Task 1 (tracer): flock on watchlist.json.lock** - `fc94a57` (test), `4a6c1ed` (feat)
2. **Task 2: crash-safe backup, version refusal, AddRef/RemoveRef** - `802a6bc` (test), `5a6e045` (feat)

**Plan metadata:** the docs commit that holds this file.

## Files Created/Modified

- `internal/watchlist/lock.go` - `ErrBusy`, `defaultLockWait`, `lockPoll`, `Store.lock`
- `internal/watchlist/lock_test.go` - `TestHelperAdd`, `TestMultiProcessAdd`, `TestLockBusy`, `TestLoadWhileLocked`, `TestLockFileKept`
- `internal/watchlist/watchlist.go` - `Result`, `AddRef`, `RemoveRef`, `VersionError`, `ErrFull`, `CorruptError.Version`, `classify`, new `backup`, `Store.link`, `Store.lockWait`, `Store.afterRead`
- `internal/watchlist/watchlist_test.go` - backup, version and result tests; lock file expectations in `TestAddCreatesFile` and `TestPermissions`
- `go.mod` - `golang.org/x/sys v0.47.0` moved to the direct block (go.sum identical to `0f4babd`)

## Decisions Made

- Followed the plan as written: x/sys/unix, 3 s / 10 ms, version classified before entries are decoded, fail-closed backup.
- `TestBackupPublishedComplete` asserts the "final name does not exist yet" condition only on the first `link` call. Its second `Load` legitimately meets the existing name, which also exercises the `fs.ErrExist` "already done" path (`calls == 2`).

## Deviations from Plan

None - plan executed exactly as written. The TDD RED commits carry small compile stubs in non-test files (the plan's commit-per-phase rule needs the tests to compile to fail on an assertion rather than on a build error); both stubs were replaced in the GREEN commits.

## Issues Encountered

None. One test bug of my own (the "final name absent" assertion on the second Load) was fixed before the GREEN commit.

## Verification

- `go vet ./... && go test ./... -count=1 -race`: all packages pass.
- `GOOS=linux GOARCH=amd64 go vet ./internal/watchlist`; `go build ./...` for linux/arm64, darwin/amd64, darwin/arm64: pass.
- `git diff --exit-code 0f4babd -- go.sum`: exit 0. `go.mod` diff is only the x/sys line moving.
- Not run: Linux host execution (RESEARCH A7). If a Linux host is available, run `go test ./internal/watchlist -race -count=1` there.

## Known Stubs

None.

## Threat Flags

None. New files (`watchlist.json.lock`, `.corrupt-*.tmp`) are all in the existing 0700 data directory and covered by T-05-05.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan 05-02 can switch `ui.Watchlist` to `AddRef`/`RemoveRef`, show `Result.StartedOver`, and map `ErrBusy`, `ErrFull` and `*VersionError` to short messages. `Add`/`Remove` remain until then.

## Self-Check: PASSED

- FOUND: internal/watchlist/lock.go, lock_test.go, watchlist.go, watchlist_test.go
- FOUND commits: fc94a57, 4a6c1ed, 802a6bc, 5a6e045

---
*Phase: 05-address-tech-debt-watch-list-locking-live-smoke*
*Completed: 2026-10-14*

---
phase: 05-address-tech-debt-watch-list-locking-live-smoke
reviewed: 2026-10-13T00:00:00Z
depth: standard
files_reviewed: 19
files_reviewed_list:
  - README.md
  - cmd/prv/main_test.go
  - go.mod
  - internal/github/live_test.go
  - internal/ui/add.go
  - internal/ui/add_test.go
  - internal/ui/model.go
  - internal/ui/remove.go
  - internal/ui/remove_test.go
  - internal/ui/show_test.go
  - internal/ui/testdata/TestGolden/add-saving_0x0.golden
  - internal/ui/testdata/TestGolden/add-saving_120x40.golden
  - internal/ui/testdata/TestGolden/add-saving_60x15.golden
  - internal/ui/testdata/TestGolden/add-saving_80x24.golden
  - internal/ui/view_test.go
  - internal/watchlist/lock.go
  - internal/watchlist/lock_test.go
  - internal/watchlist/watchlist.go
  - internal/watchlist/watchlist_test.go
findings:
  critical: 0
  warning: 2
  info: 4
  total: 6
status: issues_found
---

# Phase 5: Code Review Report

**Reviewed:** 2026-10-13
**Depth:** standard
**Files Reviewed:** 19
**Status:** issues_found

## Summary

I reviewed the flock-based watch-list lock, the backup and version-mismatch changes in the store, the new `saving` state in the add prompt, the specific save and removal reasons, the opt-in live test, and the supporting tests, goldens and README.

I found no critical defects. I checked these points:
- The lock is correct. `flock` is taken on a sibling file, polled with `LOCK_NB`, and dropped on `Close`. The lock file is never deleted. `EWOULDBLOCK` and `EINTR` are matched correctly.
- The store holds the lock across the whole read-modify-write in both `AddRef` and `RemoveRef`.
- The backup is built from the bytes that were hashed and is published with a no-clobber `os.Link`.
- A version mismatch never reaches the start-over path.
- esc, typing and paste are ignored in the `saving` state, and the spinner keeps running because of the `model.go` change.
- `saving` is cleared on every exit from `applyAddSaved`.

Verification run:
- `go build`, `go vet`, `gofmt -l` and `go mod tidy -diff` are clean.
- `go test ./...` passes.
- `go test -race ./internal/watchlist ./internal/ui` passes.
- `TestMultiProcessAdd` passed 20 runs in a row.
- `GOOS=windows go build` fails in `lock.go`, as D-08 and the README state.

The warnings below concern one removal path that reports success without writing, and a skip in the live test that can hide a real regression.

## Warnings

### WR-01: Live test skips on expired CI data before it reports a Review or State regression on the same pin

**File:** `internal/github/live_test.go:254-275`
**Issue:** The code runs every skip before it records any error (`t.Skipf` at line 262 for expired check data, line 258 for drift, line 255 for a reopened PR). A pin with `pin.ci != CINone` whose check data has expired (`pr.CI == CINone`) is skipped for the whole subtest. The pins are 17465, 20036, 14327 and 14440, and the file says 14327 and 14440 expire around October 2027. When that happens, the `State`, `Merge` and `Review` assertions for that pin never run. If `verdictBug` was computed at line 251, it is dropped too. A prv regression in the review or state mapping stays hidden behind a "re-pin" message, and the run reports a skip, which people read as acceptable. This weakens the main purpose of the test.
**Fix:** Do not skip before checking the fields that expiry cannot change. Report the Review, State and Merge errors first. Treat only the CI comparison as skippable:
```go
if v == verdictBug { t.Errorf(...) }
if pr.State != pin.state { t.Errorf(...) }
if pr.Merge != domain.MergeNone { t.Errorf(...) }
if pin.ci != domain.CINone && pr.CI == domain.CINone {
    t.Logf("pin %s: GitHub expired the check data; re-pin", pin.ref) // or Skip as the last statement
} else if pr.CI != pin.ci {
    t.Errorf(...)
}
if t.Failed() { return }
```
Keep the drift skip, because there the Review value itself is unknowable. Move the "check data expired" skip so it comes after the error checks.

### WR-02: Removing from an unparseable file reports success, writes nothing, and the "started over" notice cannot fire

**File:** `internal/watchlist/watchlist.go:289-307`, `internal/ui/remove.go:80-106`
**Issue:** When `watchlist.json` is unparseable and the user removes a saved PR, `RemoveRef` takes the `readForWrite` start-over branch. That branch returns an empty shape and a backup name. `kept` equals `shape.PRs`, both empty, so it returns `Result{}, nil` at line 301. This has three consequences:
1. The UI treats `Changed == false` as "already gone". `applyRemoved` still drops the row and flashes `removed …`, although the file is still broken and the PR was never written anywhere. The next refresh calls `savedRefs`, which fails to `Load` and falls back to `c.lastRefs`. That list still holds the removed PR, so the row comes back with no explanation.
2. The `Result{Changed: true, StartedOver: …}` return at line 306 can never carry a backup. `StartedOver` is only non-empty when the shape was replaced by an empty list, and `kept < len` needs entries. The D-04 notice in `removeCmd`/`applyRemoved` (`note`, `startedOverNotice`) is therefore dead in production. `TestRemoveStartedOverNotice` only exercises a fake. Its comment says "The real store reports none today", which confirms this.
3. A side effect of backing up the file is that the user is told nothing about it.
**Fix:** Pick one of two options:
- Make `RemoveRef` fail on an unparseable file, because there is nothing to remove from. For example, return the `*CorruptError` (with the backup name) instead of `Result{}, nil`. `saveReason` can then say "watch list is damaged; saved to …".
- Or keep the "nothing to remove" result, and set `StartedOver: startedOver(backup)` on the early return so the UI can show the backup path. Then delete the `note` plumbing in `removeCmd` if you decide removal never reports it.

## Info

### IN-01: A backup is published without syncing the directory, and an existing backup of the same 8-digit name counts as complete

**File:** `internal/watchlist/watchlist.go:187-195`
**Issue:** `backup` syncs the temp file and then calls `link`, but never syncs the directory. A power loss can leave the later `rename` of the new `watchlist.json` on disk without the link, so the old bytes would be gone. This depends on the filesystem's journal ordering, so the risk is low. Also, `errors.Is(err, fs.ErrExist)` treats any existing `watchlist.json.corrupt-HASH8` as done without comparing contents. The name holds only 32 bits of the hash, so a different file that collides would silently lose its backup. The chance is very small.
**Fix:** After a successful link, open the directory and `Sync()` it. On `fs.ErrExist`, compare the size and bytes of the existing file with `data` (or use more hex digits) before treating it as done.

### IN-02: Test hook `afterRead` lives in the production `Store` and is wired only into `AddRef`

**File:** `internal/watchlist/watchlist.go:132-135,282-284`
**Issue:** The hook is a test seam in the production type. `RemoveRef` has no equivalent, so the multi-process proof covers only add. The same seam pattern is already used for `rename` and `link`, so this is a consistency note rather than a defect.
**Fix:** Add the same call after `readForWrite` in `RemoveRef`, or document that remove is covered by `TestLockBusy` only.

### IN-03: `sync` local variable shadows the imported package in the helper test

**File:** `internal/watchlist/lock_test.go:28`
**Issue:** `sync := os.Getenv("PRV_WATCHLIST_HELPER_SYNC")` shadows the `sync` package that the same file imports for `sync.WaitGroup`. It compiles, but a later edit that uses `sync.` inside `TestHelperAdd` would fail with a confusing error.
**Fix:** Rename the variable to `syncDir`.

### IN-04: Windows branches in tests cannot run

**File:** `internal/watchlist/lock_test.go:224`, `internal/watchlist/watchlist_test.go:61,476,644,694`
**Issue:** The `runtime.GOOS != "windows"` guards are dead code. The package no longer builds on Windows, because `lock.go` has no build tag or stub (D-08, documented in the README).
**Fix:** Remove the guards, or leave them if a later Windows port is expected. If `lock.go` gets a `//go:build unix` tag, make that choice now so that `GOOS=windows go build ./...` fails with a clearer message than `undefined: unix.Flock`.

---

_Reviewed: 2026-10-13_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

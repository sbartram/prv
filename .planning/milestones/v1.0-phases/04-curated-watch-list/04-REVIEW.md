---
phase: 04-curated-watch-list
reviewed: 2026-10-13T00:00:00Z
depth: standard
files_reviewed: 31
files_reviewed_list:
  - cmd/prv/main.go
  - cmd/prv/main_test.go
  - internal/domain/domain_test.go
  - internal/domain/list.go
  - internal/domain/pr.go
  - internal/domain/ref_test.go
  - internal/domain/watch.go
  - internal/domain/watch_test.go
  - internal/github/client.go
  - internal/github/client_test.go
  - internal/github/lookup.go
  - internal/github/lookup_test.go
  - internal/github/testdata/lookup_mixed.json
  - internal/github/watchsource_test.go
  - internal/ui/add.go
  - internal/ui/add_test.go
  - internal/ui/detail.go
  - internal/ui/detail_test.go
  - internal/ui/help.go
  - internal/ui/help_test.go
  - internal/ui/keys.go
  - internal/ui/model.go
  - internal/ui/refresh.go
  - internal/ui/remove.go
  - internal/ui/remove_test.go
  - internal/ui/styles.go
  - internal/ui/unavail_test.go
  - internal/ui/view.go
  - internal/ui/view_test.go
  - internal/watchlist/watchlist.go
  - internal/watchlist/watchlist_test.go
findings:
  critical: 0
  warning: 5
  info: 3
  total: 8
status: issues_found
---

# Phase 4: Code Review Report

**Reviewed:** 2026-10-13
**Depth:** standard
**Files Reviewed:** 31
**Status:** issues_found

## Summary

I reviewed the Phase 4 changes (watch-list store, aliased GraphQL Lookup, add and remove prompts, fetch-epoch handling, unavailable rows). `go vet` is clean, and the tests pass under `-race` for `internal/watchlist`, `internal/github`, `internal/domain`, `internal/ui` and `cmd/prv`.

Parts that hold up under scrutiny:
- **Query construction:** `buildLookup` formats only the integer alias index into the query text. Owner, repository and number always travel as variables.
- **Error attribution:** `lookupRefs` attributes errors to an alias by `path[0]`. A whole-request failure returns before any row is built.
- **Epoch handling:** `markMutation` sets `mutEpoch = fetchEpoch+1`. That discards an in-flight fetch and never discards the next one. I traced the Init, manual refresh and rate-limit-pause paths.
- **Snapshot aliasing:** Updates to `m.snap.PRs` clone the slice before mutating, so copies of the `Model` do not share state.
- **Input handling:** `ParseRef`, `ValidRef` and `Clean` are sound. I found no injection or terminal-escape path from GitHub text or pasted text.

The defects cluster in the watch-list store. Its stated guarantees (two prv instances stay in sync, a hand edit is never deleted, a backup exists before anything is replaced) are weaker than the documentation says. The store has no cross-process locking and its backup step is not crash-safe. It also drops unknown JSON fields on every write and replaces a version-mismatched file without a prompt. In the UI, esc during the save step does not stop the save.

No critical issues were found.

## Warnings

### WR-01: Add/Remove is an unlocked read-modify-write; concurrent prv processes lose updates

**File:** `internal/watchlist/watchlist.go:223-267` (Add, Remove), `:271-307` (write)
**Issue:** `Add` and `Remove` read the file (`readForWrite`), modify it in memory, then rename a temp file over it. Nothing serialises the cycle across processes. The rename is atomic, but the cycle is not. Two prv panes can both read version N. Pane A then adds X and pane B adds Y. Whichever renames second silently deletes the other pane's entry, and the user sees "added" in both panes.

The package doc and D-16 promise that two instances stay in sync. `TestTwoStoresInterleave` only interleaves the steps sequentially, so it cannot catch this. The window is short, but this is the one data-loss path in the store.

**Fix:** Hold an advisory lock on a sibling lock file for the whole read-modify-write.
```go
func (s *Store) lock() (unlock func(), err error) {
    if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil { return nil, err }
    lf, err := os.OpenFile(s.path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
    if err != nil { return nil, err }
    if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil { lf.Close(); return nil, err }
    return func() { syscall.Flock(int(lf.Fd()), syscall.LOCK_UN); lf.Close() }, nil
}
// Add and Remove: unlock, err := s.lock(); if err != nil {...}; defer unlock()
```
Add a test that runs N goroutines, each with its own `Store`, calling `Add`, and asserts that all N entries survive. Reads can stay lock-free, because the rename is atomic.

### WR-02: Backup is not crash-safe, and a partial backup is treated as complete

**File:** `internal/watchlist/watchlist.go:133-164` (backup), `:211-217` (readForWrite)
**Issue:** `backup` creates the final-named file with `O_EXCL` and then copies into it. If prv is killed or the machine loses power between the create and the finished `io.Copy`, a truncated `watchlist.json.corrupt-HASH8` remains. The deferred cleanup only runs on a Go error, not on a crash.

On the next run `OpenFile` returns `ErrExist`, so `backup` returns success without checking the copy ("once"). `readForWrite` then treats the corrupt file as safely backed up, and the next `Add` or `Remove` overwrites the only complete copy of the user's bytes.

There is a second flaw. The name is the hash of `data`, but the copy re-opens `s.path` instead of writing `data`. If another process atomically replaces the file between `read` and `backup`, the backup holds different bytes than its name says. `readForWrite` then proceeds as if the old content were saved.

**Fix:** Write the copy to a temp file in the same directory, `Sync` it, then publish it with a no-clobber link, so the final name only ever appears complete. Write the bytes that were hashed (`data`), not a re-read. For the over-1-MiB case, hash and copy through a single open handle.
```go
tmp, _ := os.CreateTemp(dir, ".corrupt-*.tmp")   // write data, Sync, Close
if err := os.Link(tmp.Name(), name); err != nil && !errors.Is(err, fs.ErrExist) { ... }
os.Remove(tmp.Name())
```

### WR-03: Add/Remove silently replaces a corrupt or newer-version file, and the user is not told in the add flow

**File:** `internal/watchlist/watchlist.go:211-217`, `internal/ui/add.go:225-248`, `cmd/prv/main.go:136`
**Issue:** `readForWrite` converts every `ErrCorrupt` into an empty list. That covers invalid JSON, a file over 1 MiB, and `Version != 1`. The first `Add` or `Remove` therefore replaces the whole file with a one-entry list. A `version: 2` file written by a newer prv is "downgraded" and clobbered by an older one.

On the next refresh `savedRefs` reads the new file. Every previously saved PR vanishes from the list. The only explanation was the startup warning, which expires after 8 seconds (`warnFor`) and is shown once. The add confirmation just says "added". The path to the backup is not shown at the moment of loss.

**Fix:**
- For a version mismatch, refuse to write and return an error such as "watchlist.json is version N; this prv writes version 1". Only an unparseable file should start over.
- When starting over, have `Add` and `Remove` return a notice (for example a `CorruptError` flag in the result). The UI then flashes "previous list saved to watchlist.json.corrupt-XXXXXXXX".

### WR-04: Round-tripping through typed structs drops unknown fields, and one wrongly typed entry makes the whole file "corrupt"

**File:** `internal/watchlist/watchlist.go:67-76` (fileShape/entry), `:117`, `:239`, `:290`
**Issue:** The package doc says "a hand edit is never deleted by an Add" and "Entries that are not PR references stay in the file". Two things break this:
1. `fileShape` and `entry` list only `version`, `prs`, `repo`, `number` and `added_at`. Any other key (a hand-written `"note"`, a future version-1 field, top-level metadata) is discarded on the next `Add` or `Remove`. A `null` element in `prs` is rewritten as `{"repo":"","number":0,"added_at":"0001-01-01T00:00:00Z"}`.
2. `json.Unmarshal` into `[]entry` fails the whole file if one entry has `"number": "12"` or `"added_at": ""`. The file is then classed as corrupt (WR-03 path), instead of that one entry being skipped as the doc states.

**Fix:** Decode `prs` as `[]json.RawMessage`. Parse each element on its own for `Load`, and skip those that fail. Write back the raw bytes of the untouched elements and marshal only the new entry. Preserve unknown top-level keys the same way (`map[string]json.RawMessage`).

### WR-05: Esc while a save is in flight does not cancel the save and gives no feedback

**File:** `internal/ui/add.go:104-111` (addKey), `:187-221` (applyAddChecked), `:225-228` (applyAddSaved)
**Issue:** The prompt shows "esc cancel" while `checking` is true. After the Lookup succeeds, `applyAddChecked` dispatches `saveCmd`, which writes the file. Esc at that point sets `m.add = addState{}` and bumps `addSeq`, so `applyAddSaved` drops the result. But the `Add` still runs and the entry is persisted.

The user cancelled, saw the prompt close, and the PR then appears on a later refresh, or never visibly if nothing triggers it. `markMutation` is also skipped, so an in-flight fetch is not discarded. Cancel during the lookup step is clean. Cancel during the save step lies.

**Fix:** Ignore esc once the save has been dispatched. Track a `saving` flag, and make `addKey` return nil for Cancel while it is set. Alternatively, handle a dropped `addSavedMsg` that has `added == true` by still merging the PR. Choose one so the UI and the file agree.

## Info

### IN-01: Lookup and Fetch share one "latest response headers" slot, so classification can use the other request's headers

**File:** `internal/github/lookup.go:136`, `internal/github/client.go:265`, `internal/github/ratelimit.go:20-52`
**Issue:** The `Lookup` doc says it may run alongside `Fetch` and that the shared header capture is "safe for concurrent use". It is free of data races, but `capture.latest()` returns whichever response finished last. `classify(err, c.capture.latest())` can therefore read `Retry-After` or `X-RateLimit-Remaining` from the other request. A HTTP-200 GraphQL error, such as NOT_FOUND on an add, could be classed as rate-limited. A real rate-limit error could miss its headers. Both requests spend the same quota, so the effect is rare and mostly self-correcting.

**Fix:** Capture headers per request. For example, attach a header sink to the request context that `RoundTrip` fills, and have `classify` read from it. Or correct the doc comment so it does not claim more than it delivers.

### IN-02: Add failure reasons are collapsed into one generic message

**File:** `internal/ui/add.go:229-232`
**Issue:** `Store.Add` returns distinct errors: "the watch list is full (500 PRs)", permission errors, and an unreadable file. The UI shows only "could not save the watch list". The user cannot tell "list full" (needs `d`) from "disk or permission problem". `applyRemoved` has the same shape ("could not update the watch list").

**Fix:** Show a cleaned short reason, for example `"could not save (" + domain.Clean(shortErr(msg.err)) + ")"`. At least special-case the full-list error with a sentinel such as `watchlist.ErrFull`.

### IN-03: Parse errors are hidden during a rate-limit pause

**File:** `internal/ui/add.go:159-174`
**Issue:** `submitAdd` checks the pause first and returns "rate limited — resumes HH:MM" before it parses the text. A pasted issue URL or a non-GitHub host gets the rate-limit message instead of its own reason. The user fixes the wrong thing once the pause ends. No request would be sent for a bad ref anyway.

**Fix:** Run `ParseRef` and the duplicate check first. Apply the pause check only immediately before the `lookupCmd` is built.

---

_Reviewed: 2026-10-13_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

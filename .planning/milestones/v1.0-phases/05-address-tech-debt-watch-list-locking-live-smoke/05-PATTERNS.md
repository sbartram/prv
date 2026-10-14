# Phase 5: Address tech debt: watch-list locking + live smoke - Pattern Map

**Mapped:** 2026-10-13
**Files analyzed:** 13 (3 new code, 7 modified code/test, 3 docs)
**Analogs found:** 12 / 13

All analog paths below were confirmed git-tracked (`git ls-files`). No gitignored mirror paths are used. 05-RESEARCH.md already holds tested skeletons for lock.go, backup, the re-exec test and the live test; this file points at the real code each one slots into. Line numbers are from the working tree at mapping time.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/watchlist/lock.go` (new) | utility (file lock) | file-I/O | `internal/watchlist/watchlist.go` `write()` (lines ~300-330: MkdirAll + 0600 file) | role-match (no flock code exists) |
| `internal/watchlist/watchlist.go` (mod: Add/Remove lock, read(), backup(f), readForWrite, ErrFull, VersionError, Result/AddRef/RemoveRef) | service (store) | file-I/O CRUD | itself (current `read`/`backup`/`readForWrite`/`Add`/`Remove`) | exact |
| `internal/watchlist/lock_test.go` (new: multi-process, timeout, backup crash) | test | file-I/O, multi-process | `watchlist_test.go` `TestTwoStoresInterleave` (524-545) + `wlStore` helper (177-181) | role-match |
| `internal/watchlist/watchlist_test.go` (mod) | test | file-I/O | itself (`TestAddCreatesFile` 41-78, `TestNoTempLeftovers` 502-522, `wlCheckBackup` 334-352) | exact |
| `internal/ui/add.go` (mod: `saving`, addKey order, saveReason, notice) | component (Update state machine) | request-response / event-driven | itself | exact |
| `internal/ui/remove.go` (mod: RemoveRef, specific reason) | component | request-response | itself + `add.go` `applyAddSaved` | exact |
| `internal/ui/model.go` (mod: spinner.TickMsg condition, line ~320) | component | event-driven | itself | exact |
| `internal/ui/add_test.go`, `remove_test.go`, `view_test.go` goldens (mod) | test | event-driven | `TestAddEscDropsLateResult` (650-683), `TestAddSaveError` (686-700), `memWatch` (41-58) | exact |
| `internal/github/live_test.go` (new, `package github_test`, PRV_LIVE=1) | test | request-response (live) | `internal/github/watchsource_test.go` (stub WatchSource, Options{Token, Watch}) | role-match |
| `go.mod` | config | n/a | n/a (`go mod tidy` moves `golang.org/x/sys` to direct) | n/a |
| `05-UAT.md` (new) | docs | n/a | `.planning/phases/02-live-monitoring/02-UAT.md`, `04-curated-watch-list/04-UAT.md` | role-match (use as format) |
| `04-REVIEW-DISPOSITION.md` (mod) | docs | n/a | itself | exact |
| `README.md` (mod: live-test run note, Windows note) | docs | n/a | itself | exact |
| `ROADMAP.md` goal line | docs | n/a | itself | exact |

## Pattern Assignments

### `internal/watchlist/lock.go` (utility, file-I/O) - NEW

**Analog:** `internal/watchlist/watchlist.go` `write()` for dir/permission conventions; the flock body is the tested skeleton in 05-RESEARCH.md "Pattern 1" (copy it whole; it was built and run `-race`). No flock precedent exists in the repo.

**Directory + mode convention to match** (watchlist.go `write`, ~lines 305-312):
```go
dir := filepath.Dir(s.path)
if err := os.MkdirAll(dir, 0o700); err != nil {
    return fmt.Errorf("creating %s: %w", dir, err)
}
tmp, err := os.CreateTemp(dir, ".watchlist-*.tmp") // mode 0600
```

**Core:** `lock()` opens `s.path+".lock"` with `O_CREATE|O_RDWR, 0o600`, polls `unix.Flock(fd, LOCK_EX|LOCK_NB)` every 10 ms until `s.lockWait` (field, default 3 s) and returns `func(){ f.Close() }`. Errors: `ErrBusy` sentinel (text `watch list is busy (another prv is writing it)`). Never delete the lock file (D-11). Keep the `*os.File` captured in the closure. Add `lockWait time.Duration` to `Store` and set it in `New`.

**Usage in Add/Remove**, placed before `readForWrite`, after the `ValidRef` early return in Add:
```go
unlock, err := s.lock()
if err != nil { return false, err }
defer unlock()
```

---

### `internal/watchlist/watchlist.go` (service, file-I/O CRUD) - MODIFY

**Analog:** itself. Edit points:

**Typed error to extend** (lines 40-44, 31-38):
```go
type CorruptError struct {
	Backup  string
	Problem string
}
func (e *CorruptError) Error() string {
	return "watchlist.json " + e.Problem + " (a copy is in " + filepath.Base(e.Backup) + ")"
}
func (e *CorruptError) Unwrap() error { return ErrCorrupt }
```
Add `Found int` (non-zero only for a real version mismatch) so `Load` still returns `*CorruptError` wrapping `ErrCorrupt` (required by `wlCheckBackup`). Add sibling `VersionError{Found int}` with the D-04 text, and `ErrFull = fmt.Errorf("the watch list is full (%d PRs)", MaxEntries)` (text must stay identical for `TestMaxEntries`, watchlist_test.go:480). Give the new error types a `Short() string` method if the UI reuses the `shorter` interface (see Shared Patterns).

**read() switch to change** (WR-03, currently lines ~116-125): the existing
```go
case len(data) > maxFileSize:
    problem = "is larger than 1 MiB"
case json.Unmarshal(data, &shape) != nil || shape.Version != Version:
    problem = "is not a valid watch list"
```
becomes the three-way probe from RESEARCH Pattern 3 (version probe `*int` first, oversized/unparseable start over, `probe.Version != nil && != Version` sets `Found`). Then `name, err := s.backup(f)` instead of `s.backup(data)`.

**backup()** (current lines ~133-164 use `O_EXCL` then re-open `s.path`, the WR-02 flaw). Replace with the tested `backup(f *os.File)` in RESEARCH Pattern 2: Seek 0, `CreateTemp(dir, ".corrupt-*.tmp")`, `io.MultiWriter(tmp, sha256)`, `Sync`, `Close`, `os.Link` with `errors.Is(err, fs.ErrExist)` tolerated, deferred `Remove(tmp)`. Fail closed on other Link errors (existing wrapper text `could not back up watchlist.json: %w` stays).

**readForWrite()** (current lines ~208-216):
```go
shape, err := s.read()
if errors.Is(err, ErrCorrupt) {
    return fileShape{Version: Version}, nil
}
return shape, err
```
Change to the RESEARCH Pattern 3 form: `errors.As(err,&ce)`; `ce.Found != 0` returns `&VersionError{}`; else returns empty shape plus `ce.Backup` as a second return value.

**Result/AddRef/RemoveRef** (RESEARCH Pattern 4): keep `Add`/`Remove` `(bool, error)` as thin wrappers (about 41 call sites in `watchlist_test.go`, `cmd/prv/main_test.go`, `internal/github/watchsource_test.go`, plus `var _ ui.Watchlist = (*watchlist.Store)(nil)` at `cmd/prv/main.go:54`). `StartedOver = filepath.Base(backup)`, set only when the file was actually written (Remove over a corrupt file writes nothing, so no notice). Replace the inline `fmt.Errorf("the watch list is full ...")` at the `len(refs) >= MaxEntries` check with `ErrFull`.

**Update the package doc comment** (lines 1-10): it says "A file prv cannot use (not valid JSON, another version, over 1 MiB) is first copied ... only then may an Add start a fresh file". Reword: a version mismatch is never written over.

---

### `internal/watchlist/lock_test.go` (test, multi-process) - NEW

**Analog:** `watchlist_test.go` `TestTwoStoresInterleave` (stays unchanged, D-07) and `wlStore` (lines 177-181):
```go
func wlStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
	return New(path), path
}
```
Style: stdlib `testing` only, no testify, `t.Helper()`, `t.TempDir()`. Copy the tested `TestHelperAdd` / `TestMultiProcessAdd` from RESEARCH Pattern 5 verbatim (uses `os.Executable()`, `-test.run=^TestHelperAdd$`, env var for path and number, skip when the env var is empty). Add a timeout test: `unlock, _ := s1.lock()`, `s2 := New(path); s2.lockWait = 50*time.Millisecond`, assert `errors.Is(err, ErrBusy)` and file bytes unchanged. Add a backup-failure test asserting no `.corrupt-*.tmp` leftover.

### `internal/watchlist/watchlist_test.go` (test) - MODIFY
Existing assertions to change (RESEARCH "Existing tests the changes touch"): `TestAddCreatesFile` file count 1 -> 2 (add `watchlist.json.lock`), `TestPermissions` also checks the lock is 0600, `TestNoTempLeftovers` (502-522) also globs `.corrupt-*.tmp`, `TestAddOverCorrupt` (415-434) asserts `StartedOver`, `TestUnknownVersion` (368) gains Add/Remove refusal with unchanged bytes. Reuse `wlCheckBackup` (334-352) and `corruptName` regexp.

---

### `internal/ui/add.go` (component, event-driven) - MODIFY

**Analog:** itself. Excerpts of the exact places:

**State** (lines 35-40): add `saving bool` next to `open, checking bool`.

**addKey order** (lines 104-125). WR-05 root cause: Cancel is matched before `checking`. Insert after the ctrl+c case, before Cancel:
```go
case msg.String() == "ctrl+c":
    return tea.Quit
case m.add.saving:   // NEW: save dispatched, esc and typing are ignored
    return nil
case bkey.Matches(msg, m.pkeys.Cancel):
    m.add = addState{}
    m.addSeq++ // a result that is still on its way is dropped
    return nil
case m.add.checking:
    return nil
```
Esc during `checking` (the Lookup step) must still cancel (`TestAddEscDropsLateResult`).

**applyAddChecked** end (line ~219): `m.add.checking = true` becomes `m.add.saving = true` (line ~191 already set `checking = false`). Keep `return saveCmd(m.watchlist, pr, m.addSeq)` so `addSeq` still drops stale results.

**applyAddSaved error branch** (lines ~229-233):
```go
if msg.err != nil {
    m.add.checking = false
    m.add.errText = "could not save the watch list"
    return nil
}
```
Set `m.add.saving = false`; `errText = "could not save (" + saveReason(msg.err) + ")"`. Mirror the existing wording shape in `applyAddChecked`: `m.add.errText = "could not check (" + errReason(msg.err) + ")"`. Success branches: when `msg.note != ""` show `m.setFlash("previous list saved to "+msg.note, warnFor)` (8 s, model.go:42) instead of the plain-return branch (lines ~254-257) and the "hidden by filters" flash (see Pitfall 8 in RESEARCH: notice must outrank the added text on narrow panes).

**addLine** (lines 303-334): add a `case m.add.saving:` after the `errText` case:
```go
case m.add.checking:
    suffix = "   checking " + m.add.ref.String() + " " + m.spin.View()
```
-> new `case m.add.saving: suffix = "   saving " + m.add.ref.String() + " " + m.spin.View()` (no "esc cancel").

**addPaste** (~149-155): guard becomes `if m.add.checking || m.add.saving`.

**Message and Cmd shapes** (lines 52-57, 74-84): `addSavedMsg` gains `note string`; `saveCmd` calls `w.AddRef(pr.Watched)` and keeps the `recover()` wrapper:
```go
defer func() {
    if recover() != nil {
        msg = addSavedMsg{seq: seq, pr: pr, err: errors.New("unexpected internal error while saving the watch list")}
    }
}()
```
**Interface** (lines 25-28): `Watchlist` switches to `AddRef`/`RemoveRef` returning `watchlist.Result`-like values. To avoid importing `internal/watchlist` into `ui` for the result type, define a small `ui`-side struct or keep `StartedOver` as a second return; planner's choice.

### `internal/ui/remove.go` (component) - MODIFY
**Analog:** `add.go` for the reason helper. `removedMsg` (18-26) stays; `removeCmd` (31-41) calls `RemoveRef`. `applyRemoved` line ~80: replace
```go
return m.setFlash("could not update the watch list", flashFor)
```
with `"could not update (" + saveReason(msg.err) + ")"`.

### `internal/ui/model.go` (line ~320)
```go
case spinner.TickMsg:
    if m.inFlight || m.state == stateLoading || m.add.checking {
```
Add `|| m.add.saving` or the spinner freezes during the save.

### UI tests
**Fake to update** (`add_test.go` 41-58): `memWatch` with `Add`/`Remove` recording refs and returning configured values; add `AddRef`/`RemoveRef` with a configurable `note`. **Pattern for the new esc-during-saving test:** copy `TestAddEscDropsLateResult` (650-683): `m, first := submit(m, "octo/tools#42")`, `drainUI`, then feed `addCheckedMsg` to reach `saving` and assert `special(tea.KeyEscape)` leaves `addOpen(m)` true and the late `addSavedMsg` applies. **Update** `TestAddSaveError` (686-700, expects `✗ could not save the watch list`) and `TestRemoveError` (remove_test.go:293-310). Goldens: new `add-saving` state in view_test.go (479-488); regenerate with the golden package update flag (confirm the flag on first run).

---

### `internal/github/live_test.go` (test, live request-response) - NEW

**Analog:** `internal/github/watchsource_test.go` for building a `Client` with a `Watch` source (there it is `package github`; the live test is `package github_test` per RESEARCH Pattern 6, so it needs only exported API). Exported shapes: `Options{Token, ClosedDays, Now, Transport, Watch}` (client.go:28-34), `Watch WatchSource` with `Load() ([]domain.Ref, []string, error)` (client.go:20-23), `(*Client).Fetch(ctx)` (client.go:255).

Core (from RESEARCH, run once live):
```go
type pinSource []domain.Ref
func (p pinSource) Load() ([]domain.Ref, []string, error) { return p, nil, nil }
// gate: if os.Getenv("PRV_LIVE") != "1" { t.Skip(...) } before any network or gh call
// tok, _ := auth.TokenForHost("github.com"); empty -> t.Fatal
```
Pin table with expected State/Review/CI/Merge and durable-vs-perishable flags is in 05-RESEARCH.md "Live PR pins" (11 pins; do NOT pin `prometheus/prometheus#19994`, it is an open draft). CI-perishable pins must `t.Skipf` with a re-pin message when the rollup became `CINone`. Never log the token. Run line for the doc comment and README: `PRV_LIVE=1 go test ./internal/github -run TestLive -count=1 -v`.

### `05-UAT.md` - NEW
**Analog:** `.planning/phases/02-live-monitoring/02-UAT.md` and `04-curated-watch-list/04-UAT.md` for the checklist format (read them when writing; not excerpted here). Scope per D-16: six Phase 2 UI checks plus badges/MERGE by eye. Exclude all D-17 items.

## Shared Patterns

### Short, cleaned error reasons (IN-02 / D-06)
**Source:** `internal/ui/detail.go:166-174` and `internal/ui/view.go:254-256`
```go
func errReason(err error) string {
	var sh shorter
	if errors.As(err, &sh) {
		return sh.Short()
	}
	return "error"
}
type shorter interface{ Short() string }
```
**Apply to:** `saveReason` in add.go/remove.go. Either give `ErrBusy`/`VersionError`/`ErrFull` a `Short()` and reuse `errReason`, or have `saveReason` use `errors.Is` on the sentinels (adds a `ui` -> `watchlist` import, which today only `cmd/prv` has). Never show `err.Error()` (store errors carry full paths). Fallbacks: `fs.ErrPermission` -> `permission denied`, `syscall.ENOSPC` -> `disk full`, else a fixed word. Run any leftover text through `domain.Clean`.

### No I/O in Update; all file ops in tea.Cmd with recover
**Source:** `internal/ui/add.go` `saveCmd`, `lookupCmd`; `internal/ui/remove.go` `removeCmd`. A blocking lock wait runs inside these Cmds only.

### Sequence-number staleness
**Source:** `add.go` `applyAddSaved` start: `if !m.add.open || msg.seq != m.addSeq { return nil }`. The new `saving` state must keep this.

### Transient footer messages
**Source:** `internal/ui/refresh.go:95` `setFlash(text, duration)`; `flashFor` = 4 s (refresh.go:21), `warnFor` = 8 s (model.go:42).

### File permission and atomic-write conventions
**Source:** watchlist.go `write()` (dir 0700, files 0600, temp + Sync + rename). Lock file and `.corrupt-*.tmp` follow the same modes.

### Test style
Stdlib `testing` only, `t.TempDir()`, `t.Helper()`, no testify; golden files for UI via `golden.RequireEqual`.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/watchlist/lock.go` flock body | utility | file-I/O | No flock or cross-process code exists; use the tested skeleton in 05-RESEARCH.md Pattern 1 |
| Re-exec multi-process test | test | multi-process | No helper-process test exists; use RESEARCH Pattern 5 |

## Metadata

**Analog search scope:** `internal/watchlist`, `internal/ui`, `internal/github`, `.planning/phases/04-curated-watch-list`
**Files scanned:** watchlist.go, watchlist_test.go (partial), add.go, remove.go, model.go (320), detail.go, view.go, add_test.go (partial), watchsource_test.go (head), client.go (Options)
**Tracked-source check:** `git ls-files` confirmed for internal/watchlist, internal/ui/add.go, internal/ui/remove.go, internal/github/*, and the Phase 4 planning files.
**Pattern extraction date:** 2026-10-13

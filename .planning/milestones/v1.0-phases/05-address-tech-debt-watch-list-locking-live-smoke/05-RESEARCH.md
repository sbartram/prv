# Phase 5: Address tech debt: watch-list locking + live smoke - Research

**Researched:** 2026-10-13
**Domain:** Go file locking (flock), crash-safe backup (temp + hard link), multi-process Go tests, Bubble Tea v2 add-prompt state, opt-in live GitHub GraphQL test
**Confidence:** HIGH for code shapes, flock/link semantics on macOS (run locally) and live PR values (queried 2026-10-13). MEDIUM for Linux behaviour (cross-compiled and man-page cited, not run) and for how long the pinned CI values stay true (see "Perishable CI data").

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** In scope: **WR-01** (locking), **WR-02** (backup crash-safety), **WR-03** (no silent clobber), **WR-05** (esc during save) and **IN-02** (specific save errors). All come from `.planning/phases/04-curated-watch-list/04-REVIEW.md`. Out of scope: WR-04 (unknown JSON keys and per-entry tolerant decode) and G4 (add path drops partial-data warnings). See Deferred.
- **D-02 (WR-01):** Add and Remove hold an exclusive advisory lock for the whole read-modify-write cycle (`readForWrite` → modify → `write`).
- **D-03 (WR-02):** Write the backup to a temp file in the same directory and `Sync` it. Then publish it under the final `watchlist.json.corrupt-HASH8` name with a no-clobber `os.Link`, so a partial backup can never appear under the final name. Copy the bytes that were hashed (`data`), not a re-read of `s.path`. For files over 1 MiB, hash and copy through one open handle.
- **D-04 (WR-03):** For a **version mismatch** (`version != 1`), Add and Remove **refuse to write** and return an error such as `watchlist.json is version N; this prv writes version 1`. Only an **unparseable or oversized** file may start over. When it does, Add and Remove report it to the UI, and the UI flashes the backup path, e.g. `previous list saved to watchlist.json.corrupt-XXXXXXXX`. The user must not lose saved PRs with only an 8-second startup warning as the explanation. — **Reversibility:** reversible
- **D-05 (WR-05):** Once the save is dispatched, esc no longer cancels. Add a `saving` state that is separate from `checking`. Today `applyAddChecked` reuses `m.add.checking = true` for the save, and `addKey` matches Cancel before it checks `checking`. While saving, Cancel is ignored, and the prompt should not offer `esc cancel`. The UI and the file must agree.
- **D-06 (IN-02):** Add and remove failures show a specific, cleaned short reason instead of `could not save the watch list` / `could not update the watch list`. The full-list case at least gets a sentinel (e.g. `watchlist.ErrFull`) and its own wording. The new WR-03 version-mismatch and lock-timeout errors (D-09) go through this same path. Clean every OS or path text with `domain.Clean`.
- **D-07:** The proof of WR-01 is a **multi-process test**: the test binary re-execs itself as N child processes, each calls `Store.Add` for a distinct ref on the same file, and the test asserts that all N entries survive. A goroutine-only test is not enough. The existing `TestTwoStoresInterleave` stays, but it does not count as proof.
- **D-08:** **Targets are macOS and Linux only.** Use `flock` directly (`golang.org/x/sys/unix` is already an indirect dependency; promote it, or use `syscall`). No Windows variant and no build-tagged stub. — **Reversibility:** reversible — a Windows port later adds one `_windows.go` lock file.
- **D-09:** **Block with a timeout.** Add and Remove wait for the lock up to a few seconds (exact value is Claude's discretion; a real hold lasts milliseconds). `flock` has no timeout, so poll with `LOCK_EX|LOCK_NB` until a deadline. On timeout, return an error (e.g. `watch list is busy (another prv is writing it)`), and show it through the D-06 path. This already runs in a `tea.Cmd`, never in Update.
- **D-10:** **Reads stay lock-free.** `Load` (every refresh tick) and the read inside `read()` take no lock. The atomic temp+rename write means a reader always sees a whole file.
- **D-11:** The lock file is the sibling **`watchlist.json.lock`**, mode 0600, created on demand in the same directory (`MkdirAll` 0700 as today). It is **never deleted**, because deleting a lock file races with other lockers.
- **D-12:** **Both forms.** (a) An automated live test that is **opt-in only**: it skips unless an env var such as `PRV_LIVE=1` is set. `go test ./...` must stay offline and green without it. (b) A manual UAT checklist (`05-UAT.md`, run through `/gsd-verify-work`).
- **D-13:** The automated test targets **pinned, stable public PRs** in large public repos (long-merged or closed, so their state does not drift; e.g. `cli/cli`, `prometheus/prometheus#19994`). It is read-only and asserts exact values only on fields that cannot change after merge or close. The research must choose concrete PRs and say which fields are stable for them. Open-PR-only states such as CONFLICTING mergeability or BEHIND drift, so either avoid exact assertions on them or note them as unverifiable with stable fixtures.
- **D-14:** **Local only. No CI.** There is no `.github/workflows` today, and this phase does not add one. Document how to run the live test (env var, token from `gh`) where a developer will find it (test file doc comment and/or README).
- **D-15:** Automated live test scope: **state / review decision / CI rollup / MERGE mapping** on the pinned PRs, through the real client path (`github.Client` Fetch or Lookup → domain mapping). This covers the data level of Phase 1 UAT #2 and Phase 3 UAT #4.
- **D-16:** Manual UAT scope:
  - **Phase 2 UI set** (the 6 doable checks from `02-UAT.md`): the failure marker at 80 and 40 columns, the api footer and rate-warning legibility, the filter keys (1/2/3, `/`) and paste, `o` opening a real browser, the `?` overlay, and live `config.toml` edits reflected in the header.
  - **Badges + MERGE by eye:** compare a handful of added PRs side by side with github.com (P1 #2, P3 #4, visual level).
- **D-17:** **Stay deferred** (do not plan): SAML/SSO reason (P4 #2, no SSO org), live rate-limit pause (P2 #3, already covered by the simulated-hour and e2e rate-limit tests), live `assignee:@me` (P1 #1, no assigned PR), and ambiguous-width/CJK glyphs (P1 #5). These were not selected either, so they also stay deferred: team review requests (P3 #3), large-CI-matrix detail data and interaction (P3 #1), the bad-token 401 flow (P1 #4), and light/dark theme plus narrow pane.

### Claude's Discretion

- The exact lock timeout and poll interval (D-09), and the exact error wording within D-04, D-06 and D-09.
- How Add and Remove report the "started over" notice (a result struct, a `*CorruptError` value or a flag) and how the UI flashes it (reuse `setFlash`).
- `x/sys/unix` vs `syscall` for flock.
- The live-test env var name, package placement (e.g. `internal/github`), and the helper-process mechanism for D-07 (`os.Args`/env-var re-exec pattern).
- How `05-UAT.md` words each manual check.

### Deferred Ideas (OUT OF SCOPE)

- **WR-04:** keep unknown JSON keys and decode each entry tolerantly, as `[]json.RawMessage`.
- **G4:** the add path drops partial-data warnings.
- **G1 / Phase 3 WR-03 / Phase 4 IN-01:** per-request rate-limit header capture. **G2:** detail and lookup cost missing from "api N". Both were deferred earlier by the user.
- **IN-03:** parse errors are hidden during a rate-limit pause.
- **G8:** `01-REVIEW-DISPOSITION.md` still says WR-01..03 are open, but they were fixed in Phase 2. This is docs only, and can be fixed as a quick task.
- **Phase 2 IN-01..04** (a dead MergedAt payload, a dead Config.Path, the 32-bit `1 << 31` build break, positional args silently ignored), and the README not documenting the config.toml keys.
- **A CI workflow** (`.github/workflows`), including a scheduled live test. No CI exists yet.
- **Windows support** for the lock.
- **Live checks still deferred:** see D-17.
</user_constraints>

<phase_requirements>
## Phase Requirements

No requirement IDs are mapped to this phase (tech-debt phase). It protects existing requirements:

| ID | Description | Research Support |
|----|-------------|------------------|
| WATCH-05 / WATCH-06 | Watch-list store: add and remove entries, kept across instances | The flock design (section "Lock design"), crash-safe backup, version-mismatch refusal |
| RFSH-05 | Refresh reads the watch list every tick | D-10 keeps `Load` lock-free; the live test runs the real `Fetch` path |
</phase_requirements>

## Summary

The phase has three independent parts, and all three are small. (1) **Lock + backup** in `internal/watchlist`: one new file `lock.go` (about 40 lines, flock on `watchlist.json.lock`, poll with `LOCK_EX|LOCK_NB`), a rewritten `backup` that streams through the open handle into a temp file and publishes it with `os.Link`, and a way for `read()` to tell a version mismatch from an unparseable file. I built all of it in a scratch copy of the repo: the existing watchlist tests pass (one assertion needs `1` changed to `2` for the lock file), a re-exec multi-process test loses 7 of 8 entries without the lock (3 of 3 runs) and keeps all 8 with it (3 of 3 runs under `-race`, 1.1 s). (2) **UI** in `internal/ui/add.go` and `remove.go`: a `saving` flag, a specific-reason helper, and a flash for the "started over" notice. (3) **Live smoke**: an opt-in `PRV_LIVE=1` test that runs the real `Fetch` with a stub `WatchSource` that returns pinned PRs, plus a manual `05-UAT.md`.

**The most important finding for D-13:** `prometheus/prometheus#19994` is an **open draft PR** (state OPEN, isDraft true, reviewDecision REVIEW_REQUIRED, rollup FAILURE as of 2026-10-13). It drifts and must not be pinned. Also, **GitHub removes check-run data from old commits**: in `cli/cli`, every merged PR whose head commit is older than about 2025-08-28 now has `statusCheckRollup: null`, and the cut-off moves forward by one day per day (about 400 days). So a CI-state assertion on a check-run based PR will go stale. Only PRs whose rollup comes from legacy *commit statuses* (the `hashicorp/terraform` pins below) have survived for years. The pin table therefore marks each CI expectation as durable or perishable, and the test must turn "rollup disappeared" into a clear skip with a re-pin message, not a failure that looks like a prv bug.

**Primary recommendation:** Use `golang.org/x/sys/unix.Flock` (already linked into the binary through bubbletea, so promoting it only moves one line in `go.mod`); keep `Add`/`Remove` signatures at `(bool, error)` and add richer sibling methods for the "started over" notice (about 41 test call sites use the two-value form); implement the live test in the external package `github_test` with `Fetch` and a stub `WatchSource`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Cross-process write lock | Local file store (`internal/watchlist`) | OS (kernel flock) | The store owns the read-modify-write cycle; the kernel owns lock release when a process dies |
| Crash-safe corrupt-file backup | Local file store | OS (hard link) | Pure file-system work; no UI knowledge |
| Version-mismatch refusal, "started over" notice | Local file store (typed result/error) | UI (wording, flash) | Store decides *what happened*; UI decides *what to say* |
| Esc ignored while saving | UI (`internal/ui/add.go` state machine) | — | Pure Update logic, no I/O |
| Specific error reasons | UI (`saveReason` helper) | Store (sentinels) | Store exports typed errors; UI maps them to short, cleaned text |
| Live mapping check | GitHub client (`internal/github`) → domain | `gh` (token) | Exercises the real wire → domain path |

## Standard Stack

No new library. One module moves from indirect to direct.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `golang.org/x/sys/unix` | v0.47.0 (already in go.mod and go.sum; published 2026-06-30) | `unix.Flock`, `unix.LOCK_EX`, `unix.LOCK_NB`, `unix.EWOULDBLOCK`, `unix.EINTR` | The Go team's supported syscall package; the stdlib `syscall` package is frozen. `go mod why -m golang.org/x/sys` shows the binary already imports `golang.org/x/sys/unix` through `charm.land/bubbletea/v2`, so no new code is linked. [VERIFIED: `go mod why -m golang.org/x/sys` run in this session; go.mod line `golang.org/x/sys v0.47.0 // indirect`] |

`unix.Flock` exists for darwin (amd64, arm64) and linux. [VERIFIED: grep of `/Users/scottb/go/pkg/mod/golang.org/x/sys@v0.47.0/unix/zsyscall_darwin_arm64.go:1337` and `zsyscall_linux.go:837`]. `unix.EWOULDBLOCK` and `unix.EAGAIN` are the same value on darwin (`syscall.Errno(0x23)`) [VERIFIED: `zerrors_darwin_arm64.go:1630,1731`]; they are also the same on Linux, but I only read the darwin file this session. [CITED: man7.org/linux/man-pages/man2/flock.2.html says `EWOULDBLOCK` is the error for LOCK_NB on a held lock].

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| stdlib `os/exec`, `os.Executable`, `sync` | Go 1.27.2 local, `go 1.26.0` in go.mod | Re-exec helper-process test | D-07 test |
| `github.com/cli/go-gh/v2/pkg/auth` | v2.16.2 (already a direct dependency) | `auth.TokenForHost("github.com")` in the live test | D-12 live test |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `x/sys/unix.Flock` | `syscall.Flock` | Works on darwin and linux too. `syscall` is frozen and does not get new platforms; `x/sys` costs one `go.mod` line move. Either is allowed by D-08. Use `x/sys/unix`. |
| flock polling | `fcntl` byte-range locks (`unix.FcntlFlock`) | fcntl locks are per-process and are dropped when *any* fd for the file is closed, which is a known trap. flock is per open-file-description. Stay with flock. [CITED: man7.org flock(2) Notes] |

**Installation:**
```bash
cd /Users/scottb/dev/bartram/prv
go get golang.org/x/sys@v0.47.0   # no network needed: already in the module cache
go mod tidy                        # drops the "// indirect" comment, moves the line to the direct block
```
I ran the equivalent (`-mod=mod`) in a scratch copy: the only `go.mod` change is that `golang.org/x/sys v0.47.0` moves from the indirect block to the direct block. [VERIFIED: diff of go.mod in the scratch copy]. `go.sum` is unchanged.

**Version verification:** `go list -m -json golang.org/x/sys` returns `Version v0.47.0`, `Time 2026-06-30T17:07:31Z`, `GoVersion 1.25.0`. [VERIFIED: this session]

## Package Legitimacy Audit

`gsd_run query package-legitimacy check` supports only `npm|pypi|crates`, so the Go module cannot be run through the seam (the seam answers "Usage: ... --ecosystem <npm|pypi|crates>"). Substitute evidence:

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `golang.org/x/sys` | Go module proxy | Official Go project module | n/a | go.googlesource.com/sys (mirror github.com/golang/sys) | OK by inspection: already in `go.sum` with `h1:` hashes (`go.sum` lines 81-82) and already imported by `bubbletea` | Approved (no new download) |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
 prv pane A (tea.Cmd)                 prv pane B (tea.Cmd)              refresh tick (any pane)
 Store.Add(ref)                       Store.Add(ref2)                   Store.Load()  -- NO lock (D-10)
      |                                    |                                  |
      v                                    v                                  v
 lock(): open watchlist.json.lock  lock(): open watchlist.json.lock     read(): open watchlist.json
 poll flock(LOCK_EX|LOCK_NB) --+   poll flock(...) ---- EWOULDBLOCK      (always a whole file:
 10 ms sleep, deadline 3-5 s   |   sleep, retry ... until A closes fd     rename is atomic)
      | got it                 |        | deadline passed -> ErrBusy
      v                        |        v
 readForWrite()                |   (UI: "another prv is writing the list")
   read() -> ok / CorruptError |
     version != 1 -> VersionError -> return (nothing written)
     unparseable/oversized -> backup(f): stream f -> .corrupt-*.tmp, Sync,
                              os.Link(tmp, final)  (EEXIST = already done), remove tmp
                              -> start over with empty list, remember backup name
 modify in memory              |
 write(): temp + Sync + rename |
 unlock: f.Close() drops flock <+  (kernel also drops it if the process dies)
      |
      v
 Result{Changed, StartedOver: "watchlist.json.corrupt-XXXXXXXX"} -> addSavedMsg/removedMsg
      |
      v
 UI: saving flag blocks esc; applyAddSaved / applyRemoved -> setFlash(notice or specific reason)
```

### Recommended Project Structure
```
internal/watchlist/
├── watchlist.go          # modified: Add/Remove take the lock, read() classifies version vs unparseable, backup(f)
├── lock.go               # NEW: Store.lock(), ErrBusy, lockWait, lockPoll
├── watchlist_test.go     # modified: lock-file count, backup/version/start-over tests
└── lock_test.go          # NEW: multi-process test (D-07), timeout test, backup crash test
internal/ui/
├── add.go                # modified: saving flag, addKey order, saveReason, notice in applyAddSaved
├── remove.go             # modified: specific reason + notice in applyRemoved
└── model.go              # modified: spinner.TickMsg condition adds m.add.saving
internal/github/
└── live_test.go          # NEW: package github_test, opt-in PRV_LIVE=1
.planning/phases/05-.../05-UAT.md          # NEW: manual checklist
.planning/phases/04-curated-watch-list/04-REVIEW-DISPOSITION.md  # update dispositions
README.md                 # a "Development" note on running the live test (D-14)
```

### Pattern 1: Lock with deadline (flock on a sibling file)
**What:** Open `path+".lock"`, poll `flock(LOCK_EX|LOCK_NB)`, close the file to unlock.
**When to use:** Around the whole `readForWrite` → modify → `write` span of `Add` and `Remove`.
**Tested skeleton** (built and run in a scratch copy; passes with `-race` on darwin/arm64; cross-compiles for linux/amd64, linux/arm64, darwin/amd64):
```go
// internal/watchlist/lock.go
package watchlist

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

// ErrBusy: another prv holds the lock past the wait.
var ErrBusy = errors.New("watch list is busy (another prv is writing it)")

const (
	lockWait = 3 * time.Second  // a real hold lasts milliseconds
	lockPoll = 10 * time.Millisecond
)

func (s *Store) lock() (unlock func(), err error) {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil { // the dir may not exist on a first Add
		return nil, fmt.Errorf("creating %s: %w", dir, err)
	}
	f, err := os.OpenFile(s.path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}
	deadline := time.Now().Add(s.lockWait) // field, default lockWait, so a test can use 50 ms
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return func() { _ = f.Close() }, nil // closing the only fd drops the lock; never delete the file (D-11)
		}
		if err != unix.EWOULDBLOCK && err != unix.EINTR {
			f.Close()
			return nil, fmt.Errorf("locking watch list: %w", err)
		}
		if !time.Now().Before(deadline) {
			f.Close()
			return nil, ErrBusy
		}
		time.Sleep(lockPoll)
	}
}
```
Use in `Add` and `Remove`, **before** `readForWrite`:
```go
unlock, err := s.lock()
if err != nil { return false, err }
defer unlock()
```
Do not lock for the "not a PR reference" early return in `Add` (no I/O needed).

Facts the planner must keep:
- **Lock scope is the open file description.** Two `os.OpenFile` calls in one process conflict with each other, so a goroutine test with two Stores really contends. A forked/dup'ed fd shares the lock. [CITED: man7.org/linux/man-pages/man2/flock.2.html: "these file descriptors are treated independently by flock()", "may be denied by a lock that the calling process has already placed via another file descriptor"]
- **Release is automatic on process death** (the last fd closes), so no stale-lock cleanup is ever needed, which is why D-11 can keep the file forever. [CITED: same man page: lock released "when all such file descriptors have been closed"]
- **`os.OpenFile` sets `O_CLOEXEC`** (Go does this for every open), so the lock fd does not leak into a spawned browser or `gh` child. [ASSUMED: Go runtime behaviour from training; not re-checked this session]. Keep the lock file open only inside `Add`/`Remove`, which already makes this irrelevant.
- **Keep the `*os.File` reachable until unlock.** The returned closure captures `f`, so the finalizer cannot close the fd early. Do not store only the int fd.
- **EINTR with `LOCK_NB`:** a non-blocking flock returns immediately, so EINTR is not expected; handling it as "retry" costs nothing. [CITED: man page lists EINTR only for "While waiting to acquire a lock"]
- **EWOULDBLOCK == EAGAIN** on both OSes, so comparing with `unix.EWOULDBLOCK` is enough. Compare with `==`/`errors.Is`; `unix.Flock` returns a bare `syscall.Errno`.
- **NFS home directories:** Linux 2.6.12+ emulates flock with fcntl locks on NFS, and for that the file must be open for writing (the skeleton uses `O_RDWR`). [CITED: flock(2)]. Not a concern for the personal-tool scope.
- **Windows does not build** the `watchlist` package after this change (`undefined: unix.Flock`). This is the accepted D-08 trade-off. [VERIFIED: `GOOS=windows go build ./internal/watchlist` failed in the scratch copy.] Mention it in the README so nobody is surprised.

### Pattern 2: Crash-safe backup through the open handle
**What:** `read()` already has the file open. Pass that handle to `backup`, stream it into a temp file while hashing, `Sync`, then publish with `os.Link`.
**Why through the handle:** another process may atomically `rename` a new file over `s.path` after `read()` opened it. The open handle still points at the *old inode*, so copying from it gives the bytes that were parsed (and the file name is the hash of the bytes actually copied, so name and content always agree). Re-opening `s.path` (today's code, lines 149-154) is the WR-02 second flaw. This also satisfies "for files over 1 MiB, hash and copy through one open handle", because it never needs the 1 MiB prefix in memory. A side effect: for an oversized file the hash now covers the whole file, not the 1 MiB+1 prefix, so HASH8 changes for oversized files (the existing tests only check the `\.corrupt-[0-9a-f]{8}$` shape).
**Tested skeleton** (replaces `backup`, and `read()` calls `s.backup(f)` instead of `s.backup(data)`; all existing watchlist tests pass, including `TestTooLarge` which needs the full file copied):
```go
func (s *Store) backup(f *os.File) (name string, err error) {
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".corrupt-*.tmp") // mode 0600
	if err != nil {
		return "", err
	}
	defer func() {
		tmp.Close() // a second Close after the explicit one is harmless
		os.Remove(tmp.Name())
	}()
	h := sha256.New()
	if _, err = io.Copy(io.MultiWriter(tmp, h), f); err != nil {
		return "", err
	}
	if err = tmp.Sync(); err != nil {
		return "", err
	}
	if err = tmp.Close(); err != nil {
		return "", err
	}
	name = s.path + ".corrupt-" + hex.EncodeToString(h.Sum(nil))[:8]
	if err = os.Link(tmp.Name(), name); err != nil && !errors.Is(err, fs.ErrExist) {
		return "", err
	}
	return name, nil // exists already: same hash, same bytes
}
```
**`os.Link` semantics:** `link(2)` never overwrites: "If newpath exists, it will not be overwritten" and fails with `EEXIST`; Go wraps it in `*LinkError`, and `errors.Is(err, fs.ErrExist)` matches. [CITED: man7.org/linux/man-pages/man2/link.2.html]. The darwin path was exercised by the existing `wlCheckBackup` (second `Load` hits `EEXIST` and still reports the same name). [VERIFIED: scratch copy test run]. The temp file and the final name are separate directory entries for one inode, so `os.Remove(tmp)` leaves the complete final file (link count goes 2 → 1) and the mode stays 0600 (set by `CreateTemp`).

**Failure modes for `os.Link` (Linux man page):** `EPERM` = "The filesystem ... does not support the creation of hard links"; `EXDEV` = different filesystem (cannot happen, same directory); `EMLINK` = too many links to the inode (cannot happen, fresh temp file). [CITED: link(2)]. macOS filesystems without hard links (FAT/exFAT) return a different errno, which I did not observe. [ASSUMED]. **Recommended fallback: fail closed.** A non-`EEXIST` error returns `could not back up watchlist.json: ...`, `read()` already turns that into a plain error, and Add/Remove then write nothing. The file is never lost, and the only cost is that a corrupt list on such a filesystem needs a manual fix. Do not add a rename fallback (it would clobber).
**Optional hardening (Claude's discretion):** when `Link` returns `EEXIST`, compare `os.Stat(name).Size()` with the number of bytes copied (`io.Copy` returns it). A smaller file is a partial backup left by a *pre-Phase-5* prv (the old non-atomic code). The decision text does not ask for this; skip it unless it is one line.
**Crash residue:** a kill between `CreateTemp` and the deferred `Remove` leaves `.corrupt-*.tmp` in the data dir. It is harmless (never read) and mirrors the existing `.watchlist-*.tmp` behaviour. Extend `TestNoTempLeftovers` to glob `.corrupt-*.tmp` after a failing backup.

### Pattern 3: Tell a version mismatch from an unparseable file (WR-03)
`read()` today sets `problem = "is not a valid watch list"` for both cases (watchlist.go:117). Probe the version separately **before** decoding into `fileShape`, because a future version-2 file may have entries `fileShape` cannot decode, and `json.Unmarshal` would then fail and the file would look "unparseable" and be clobbered:
```go
var probe struct{ Version *int `json:"version"` }
switch {
case len(data) > maxFileSize:                       // oversized: may start over (D-04)
case json.Unmarshal(data, &probe) == nil && probe.Version != nil && *probe.Version != Version:
	// valid JSON, another version: Add/Remove must refuse
case json.Unmarshal(data, &shape) != nil || shape.Version != Version:  // unparseable (also {} or []): may start over
default:
	return shape, nil
}
```
Add a field to `CorruptError` (for example `Found int`, non-zero only for a real version mismatch) so `Load` keeps returning a `*CorruptError` that wraps `ErrCorrupt`. This matters because `TestUnknownVersion` calls `wlCheckBackup`, which requires `errors.As(err, &*CorruptError)` with a `Backup` and `errors.Is(err, ErrCorrupt)`. Then `readForWrite` becomes:
```go
shape, err := s.read()
var ce *CorruptError
if errors.As(err, &ce) {
	if ce.Found != 0 { return fileShape{}, "", &VersionError{Found: ce.Found} }  // refuse
	return fileShape{Version: Version}, ce.Backup, nil                           // start over, remember the backup
}
return shape, "", err
```
`VersionError.Error()` = `fmt.Sprintf("watchlist.json is version %d; this prv writes version %d", Found, Version)` (the D-04 text). Edge: a `{}` file has no `version` key (`probe.Version == nil`), so it counts as unparseable and starts over with a backup rather than refusing. That is the safer reading of D-04 (`version != 1` is about a file that *declares* another version). Flag it in the plan so the user can overrule.

**The "started over" notice must only be reported when the file was actually replaced.** `Remove` over a corrupt file finds nothing to remove, hits `len(kept) == len(shape.PRs)` and returns `false, nil` without writing, so the corrupt file stays in place and no notice is due. `Add` always writes after starting over.

### Pattern 4: How Add/Remove report the notice (recommended: keep the two-value methods)
There are about 41 test call sites of `Store.Add`/`Remove` with the `(bool, error)` form (`watchlist_test.go`, `cmd/prv/main_test.go`, `internal/github/watchsource_test.go`) plus `var _ ui.Watchlist = (*watchlist.Store)(nil)` in `cmd/prv/main.go:54`. Changing the signature would touch all of them for no behaviour gain.
**Recommended:** add two methods and make the old ones thin wrappers.
```go
// Result tells the caller what a write did.
type Result struct {
	Changed     bool   // the file was written
	StartedOver string // base name of the backup when an unparseable/oversized file was replaced; "" otherwise
}
func (s *Store) AddRef(ref domain.Ref) (Result, error)    // the real implementation
func (s *Store) RemoveRef(ref domain.Ref) (Result, error)
func (s *Store) Add(ref domain.Ref) (bool, error) { r, err := s.AddRef(ref); return r.Changed, err }
func (s *Store) Remove(ref domain.Ref) (bool, error) { r, err := s.RemoveRef(ref); return r.Changed, err }
```
Then `ui.Watchlist` becomes `AddRef`/`RemoveRef`; the only UI test fake is `memWatch` (`internal/ui/add_test.go:41-58`), so one fake changes. Names are the planner's choice. `StartedOver` should hold `filepath.Base(backup)` (the footer shows `watchlist.json.corrupt-XXXXXXXX`, not a home-directory path; `CorruptError.Error()` already uses `filepath.Base`).

### Pattern 5: Re-exec helper process test (D-07)
**Tested** (`go test ./internal/watchlist -race -count=1`, 3 runs, all pass, about 1.1 s for the test; the same test with the lock call removed failed 3 of 3 runs with `Load = 1 refs, want 8`, so it detects the bug):
```go
const helperEnv = "PRV_WATCHLIST_HELPER_PATH"

// TestHelperAdd is not a test: the parent re-execs the test binary to run it.
func TestHelperAdd(t *testing.T) {
	path := os.Getenv(helperEnv)
	if path == "" {
		t.Skip("helper process only")
	}
	n, _ := strconv.Atoi(os.Getenv("PRV_WATCHLIST_HELPER_N"))
	added, err := New(path).Add(domain.Ref{Repo: "o/r", Number: n})
	if err != nil || !added {
		fmt.Fprintf(os.Stderr, "Add = %v, %v\n", added, err)
		os.Exit(1)
	}
}

func TestMultiProcessAdd(t *testing.T) {
	const n = 8
	path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
	exe, err := os.Executable()
	if err != nil { t.Fatal(err) }
	var wg sync.WaitGroup
	outs, errs := make([]string, n), make([]error, n)   // each goroutine writes its own index: race-free
	for i := 1; i <= n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command(exe, "-test.run=^TestHelperAdd$")
			cmd.Env = append(os.Environ(), helperEnv+"="+path, "PRV_WATCHLIST_HELPER_N="+strconv.Itoa(i))
			b, err := cmd.CombinedOutput()
			outs[i-1], errs[i-1] = string(b), err
		}()
	}
	wg.Wait()
	// report each child error with its output, then:
	refs, _, err := New(path).Load()
	if err != nil || len(refs) != n { t.Fatalf("Load = %d refs, %v, want %d", len(refs), err, n) }
}
```
Points the plan must carry:
- Use `os.Executable()` (absolute path of the test binary), not `os.Args[0]`. Under `-race` the child is the same race-instrumented binary, so no extra setup is needed. [VERIFIED: ran with `-race`.]
- The helper **must skip when the env var is absent**, or a plain `go test` runs it as a test with an empty path.
- `i` is a per-iteration variable in Go 1.22+ (`go 1.26.0` in go.mod), so the closure is correct.
- Distinct Number per child (and distinct `Ref`), because `Add` of an existing key is a no-op.
- Optional: a `t.Context()`-based `exec.CommandContext` plus `cmd.WaitDelay` so a hung child cannot hang the suite; with the 3 s lock deadline a stuck child fails with `ErrBusy` anyway.
- `TestMain` dispatch (check env var, run the work, `os.Exit`) is an alternative that avoids the `-test.run` argument; I did not run it. The tested pattern above is enough.
- Add a second test for D-09: hold the lock in-process (`unlock, _ := s1.lock()`), build `s2 := New(path); s2.lockWait = 50*time.Millisecond`, assert `errors.Is(err, ErrBusy)` and that the file is unchanged. This needs `lockWait` to be a `Store` field (default `lockWait`).
- Optional for determinism: a "start gate" file the children poll for. Not needed: 8 simultaneous starts already fail without the lock every time.

### Pattern 6: Live test shape (D-12, D-15)
Place it in `internal/github/live_test.go` as **`package github_test`** (external package; `NewClient`, `Options{Token, Watch}`, `Fetch`, `Lookup` are exported, so no in-package helpers are needed, and the existing `package github` tests keep their own helpers). Gate it with `PRV_LIVE=1`; with the variable unset it must `t.Skip` before touching the network or `gh`. When `PRV_LIVE=1` and no token resolves, `t.Fatal` (the opt-in was explicit).

I ran this exact flow in a throwaway test (then deleted it): `NewClient(Options{Token: tok, Watch: stub})` → `Fetch` with 14 pinned refs returned all of them with `Manual: true` (saved PRs bypass the closed-days window: `domain.InWindow` returns true for `Manual`, `internal/domain/list.go:17-19`) and cost **4 points** in total (discovery + lookup + failing-checks). [VERIFIED: ran 2026-10-13, `rate cost=4 remaining=4753`.]
```go
type pinSource []domain.Ref
func (p pinSource) Load() ([]domain.Ref, []string, error) { return p, nil, nil }

func TestLive(t *testing.T) {
	if os.Getenv("PRV_LIVE") != "1" {
		t.Skip("set PRV_LIVE=1 to run against github.com (token from gh/GH_TOKEN); read-only, about 4 API points")
	}
	tok, _ := auth.TokenForHost("github.com")
	if tok == "" { t.Fatal("PRV_LIVE=1 but no GitHub token (run gh auth login or set GH_TOKEN)") }
	c, err := github.NewClient(github.Options{Token: tok, Watch: pinSource(refs)})
	...
	snap, err := c.Fetch(ctx)         // real path: discovery + lookupRefs + fetchFailing
	// index snap.PRs by Watched.Key(), compare each pin's want{State, Review, CI, Merge}
}
```
Run command (put in the file's doc comment and the README): `PRV_LIVE=1 go test ./internal/github -run TestLive -count=1 -v` (`-count=1` bypasses the test cache). Do not `t.Log` the token, headers or raw responses. The discovery half of `Fetch` lists the *developer's own* open PRs; never assert on those, only on the pinned refs.

### Anti-Patterns to Avoid
- **Locking `Load`/`read()`** (D-10 says lock-free). A reader never sees a partial file because writes rename.
- **Deleting the lock file** on unlock (races with a waiting locker that already opened it; D-11).
- **Blocking `flock` without a deadline** in a goroutine: cannot be cancelled.
- **Unlocking before `write()` returns** (the whole cycle, including the rename, must be inside the lock).
- **Asserting `updatedAt`, `title`, `author.login`, `mergeable`, `mergeStateStatus`, or raw `reviewDecision` text in the live test.** See the stability table.
- **Reusing `m.add.checking` for the save** (the WR-05 root cause).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-process mutual exclusion | PID files, `O_EXCL` lock files with a retry loop | `flock` via `unix.Flock` | The kernel releases the lock when the process dies; a PID/`O_EXCL` file stays behind after a crash and needs stale detection |
| Atomic "publish only when complete, never overwrite" | `O_EXCL` create then copy (today's bug), `Stat` then `Rename` | temp file + `Sync` + `os.Link` | `link(2)` is atomic and never overwrites |
| Short reasons for OS errors | Printing `err.Error()` (contains full paths and the OS text) | Map with `errors.Is(fs.ErrPermission)`, `errors.Is(syscall.ENOSPC)`, then fall back to a fixed word; run any leftover text through `domain.Clean` | Matches the project rule "the message of an error can hold a secret and is never shown" (`internal/ui/detail.go:166-173`) |
| Golden-file diffs | Hand-written screen compares | `github.com/charmbracelet/x/exp/golden` (`golden.RequireEqual`), already used | Update with the package's `-update` flag [ASSUMED: flag name from the golden package convention; confirm with `go test ./internal/ui -run TestGolden -update` on the first golden change] |

## Runtime State Inventory

Not a rename/refactor/migration phase in the sense of the protocol, but the store changes its on-disk side effects, so each category is answered:

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `~/.local/share/prv/watchlist.json` format is **unchanged** (version 1, same fields). New sibling files: `watchlist.json.lock` (empty, 0600) and transient `.corrupt-*.tmp`. Existing `watchlist.json.corrupt-HASH8` files from the old code stay valid; a partial one left by a crash of the old code would now count as "already backed up" (see optional size check). | none; code edit only |
| Live service config | None. | None, verified: the phase touches no external service config |
| OS-registered state | None (no launchd/systemd/cron entry names the watch list). | None |
| Secrets/env vars | New env var `PRV_LIVE` (opt-in test switch, not a secret). Live test reads the token only through `auth.TokenForHost`. | Document in README; never persist or print the token |
| Build artifacts | `go.mod` moves `golang.org/x/sys` to the direct block (via `go mod tidy`). The checked-in `prv` binary at the repo root is untracked build output (listed in `.gitignore` as `/prv`). | Run `go mod tidy`; commit `go.mod` only |

## Common Pitfalls

### Pitfall 1: Existing tests that the lock file breaks
**What goes wrong:** `TestAddCreatesFile` (watchlist_test.go:69-72) asserts the directory holds exactly one file; after the change it holds `watchlist.json` and `watchlist.json.lock`.
**How to avoid:** change the expected count to 2 and assert the names. `TestPermissions` should also check the lock file is 0600. [VERIFIED: failure reproduced in the scratch copy.]
**Warning signs:** `directory holds 2 files, want only watchlist.json`.

### Pitfall 2: A closed PR can be reopened; a merged one cannot
**What goes wrong:** a pinned CLOSED PR (not merged) flips to OPEN if a maintainer reopens it.
**How to avoid:** prefer MERGED pins for every assertion that is not specifically about "closed". Keep two CLOSED pins at most and make a state mismatch on them a skip with a re-pin message, not a prv failure.

### Pitfall 3: Perishable CI data (the big one for D-13)
**What goes wrong:** `statusCheckRollup` becomes `null` for old commits, so a pinned `CIPassing`/`CIFailing` turns into `CINone`.
**Evidence:** in `cli/cli`, merged PRs whose last commit is dated before 2025-08-28 all return `statusCheckRollup: null`; from that date on, every PR returns one. Today (2026-10-13) is 411 days later. [VERIFIED: queried 2026-10-13: 39 merged PRs between 2025-07-20 and 2025-08-28 all `None`; PRs merged 2025-08-29 onward show FAILURE/SUCCESS.] The rule looks like "check-run data is kept about 400 days". [ASSUMED: the 400-day figure is inferred from the cut-off date; no GitHub document was found this session.] `hashicorp/terraform` PRs from 2019-2020 that kept a rollup have only legacy `StatusContext` entries, never `CheckRun`. [VERIFIED: contexts of terraform#17465, #20036 are `Status SUCCESS`/`Status PENDING`.] Even so, most terraform PRs from the same years have a `null` rollup, so legacy statuses can also disappear. [VERIFIED: 2019-2020 monthly scan of terraform, rollup present for 3 of about 620 PRs.]
**How to avoid:** every CI expectation in the live test carries a `ciPerishable` flag. When the observed CI is `CINone` and the pin expects something else, `t.Skipf("pin %s: check data expired, re-pin (see README)")` instead of failing. Pins with `CINone` expectations are durable (the data is already gone). Put the date each CI pin was verified in the table row's comment.

### Pitfall 4: `reviewDecision` depends on the *current* branch protection, not on history
**What goes wrong:** `REVIEW_REQUIRED` and `APPROVED` raw values are computed from rules that can change; removing a rule turns them into `null`.
**Evidence:** `hashicorp/terraform#17465` is merged with two APPROVED reviews yet reports `REVIEW_REQUIRED`; `cli/cli#11048` is merged with one APPROVED review and reports `REVIEW_REQUIRED`. [VERIFIED: queried 2026-10-13.] [ASSUMED: the mechanism ("computed live from branch protection") is inferred from these observations.]
**How to avoid:** `DeriveReview` falls through to the latest opinionated reviews when the decision is null (`internal/domain/status.go:11-33`), so `APPROVED` and `CHANGES_REQUESTED` pins give the same prv result with or without a rule. Only the `REVIEW_REQUIRED` pins are rule-sensitive; mark them `soft`.

### Pitfall 5: `Fetch`'s discovery half is the developer's data
`Fetch` also runs the viewer's authored/assigned search. Ignore every result that is not a pinned ref. The pinned PRs bypass the closed-days window because they are `Manual`.

### Pitfall 6: `Remove` or `Add` creating files when nothing changes
`lock()` runs `MkdirAll` and creates the lock file even for a no-op `Remove`. Accept it (cheap, 0700/0600). Do not add a "skip the lock if the file is missing" shortcut: it reintroduces a race for the first writer.

### Pitfall 7: WR-05 wiring has four places, not one
`m.add.checking` is also read in `model.go:320` (the spinner tick) and in `addPaste` (`add.go:150`). If `saving` is a new flag, all of these need it, or the spinner freezes during the save and a paste can edit hidden text. See "UI code map".

### Pitfall 8: Flash length and duration for the backup notice
`flashFor` is 4 s (`refresh.go:21`) and `m.cut` truncates to the pane width. `previous list saved to watchlist.json.corrupt-XXXXXXXX` is 53 characters; combined with `added octo/tools#42 · ` it is about 75, which only fits at 80+ columns. The user must not lose this information, so show the notice instead of the plain "added ..." text on narrow panes, and consider a longer duration for this one flash (for example `warnFor`, 8 s, `model.go:42`) instead of `flashFor`.

### Pitfall 9: Version `0` / missing version
See Pattern 3: `{}` starts over (backup made), `{"version":2,...}` refuses. State this in the plan so a reviewer does not read it as a bug.

## Code Examples

### UI code map (exact places to change)

`internal/ui/add.go`
- `addState` (lines 35-40): add `saving bool`; keep `open`, `checking`.
- `addKey` (104-125): the order today is `ctrl+c` → `Cancel` → `m.add.checking` → `Accept`/`Delete`/default. Insert `case m.add.saving: return nil` **after** `ctrl+c` and **before** the Cancel case (ctrl+c still quits; the atomic rename means a quit mid-save never leaves a partial file, and the kernel drops the lock).
- `addPaste` (149-155): guard `if m.add.checking || m.add.saving`.
- `applyAddChecked` (187-221): line 219 `m.add.checking = true` becomes `m.add.checking = false; m.add.saving = true` (line 191 already sets `checking = false` first, so just replace line 219 with `m.add.saving = true`).
- `applyAddSaved` (225-260): on error, `m.add.saving = false` and set `errText` from the new `saveReason(err)`; on success `m.add = addState{}` already clears everything. The success branches return `nil` in the "row is visible" case (lines 254-257), so add the notice flash there too.
- `addLine` (303-334): add a `case m.add.saving:` between the error case and `checking`, text such as `"   saving " + ref + " " + spin`. It must **not** contain "esc cancel". Today's `checking` suffix also has no "esc cancel" hint, only the `errText` suffix does (line 309), and `errText` is empty while saving.
- `addSavedMsg` (52-57): add `note string` (the `StartedOver` base name).
- `saveCmd` (74-84): call the new `AddRef`, copy `Result.StartedOver` into the message.
- `closeAsDuplicate`, `openAdd`: no change. `Cancel` during `checking` (the Lookup step) stays valid: `TestAddEscDropsLateResult` (add_test.go:650-683) covers it and must keep passing.

`internal/ui/model.go:320`: `if m.inFlight || m.state == stateLoading || m.add.checking {` → add `|| m.add.saving`.

`internal/ui/remove.go`
- `removedMsg` (22-26): add `note string`; `removeCmd` (31-41): call `RemoveRef`.
- `applyRemoved` (75-105): line 80 `m.setFlash("could not update the watch list", flashFor)` → specific reason. Notice for removal can only be produced when the file was replaced, which `Remove` over a corrupt file never does (Pattern 3), so only the error path matters here.

`saveReason(err error) string` (new, in `add.go` or a new small file): order of checks, each returns a short fixed phrase; the fallback is `domain.Clean`-ed and cut to about 40 runes:
`errors.Is(err, watchlist.ErrFull)` → e.g. `list is full (500 PRs)`; `errors.Is(err, watchlist.ErrBusy)` → `another prv is writing the list`; `errors.As(err, &*watchlist.VersionError)` → the VersionError text; `errors.Is(err, fs.ErrPermission)` → `permission denied`; `errors.Is(err, syscall.ENOSPC)` → `disk full`; default → `write failed`. Prefix with `could not save (` / `could not update (`. Do **not** pass `err.Error()` through: store errors carry full paths (`reading /home/u/.local/share/...`). The `ui` package would import `internal/watchlist` for the sentinels; today only `cmd/prv` does. If the planner wants to avoid that import, export a `Short() string` method on the watchlist error types and reuse the existing `shorter` interface (`view.go:256`) through `errReason`; both work.

`ErrFull`: `TestMaxEntries` (watchlist_test.go:480) asserts the exact text `the watch list is full (500 PRs)`. Define `var ErrFull = fmt.Errorf("the watch list is full (%d PRs)", MaxEntries)` and return it directly so the text is unchanged.

### Existing tests the changes touch
| Test | File:line | What must change |
|------|-----------|------------------|
| `TestAddCreatesFile` | watchlist_test.go:41-78 | directory count 1 → 2 (lock file) |
| `TestAddOverCorrupt` | watchlist_test.go:415-434 | still valid (unparseable starts over); extend to assert `Result.StartedOver` |
| `TestUnknownVersion` | watchlist_test.go:368 | `Load` part stays; add `Add`/`Remove` refusal test with the file bytes unchanged |
| `TestNoTempLeftovers` | watchlist_test.go:502-522 | also glob `.corrupt-*.tmp` |
| `TestTwoStoresInterleave` | watchlist_test.go:524-545 | stays as is (D-07) |
| `TestAddSaveError` | add_test.go:686-700 | asserts `✗ could not save the watch list`; update to the new wording |
| `TestRemoveError` | remove_test.go:293-310 | asserts `could not update the watch list` twice (flash and last line); update |
| `TestAddEscDropsLateResult` | add_test.go:650-683 | esc during `checking` still cancels; keep. Add a new test: esc during `saving` is ignored, the result applies |
| `memWatch` | add_test.go:41-58 | add `AddRef`/`RemoveRef` returning the new `Result` |
| `TestGolden` states `add-checking`, `add-error` | view_test.go:479-488, 533 | goldens under `internal/ui/testdata/TestGolden/add-checking_*.golden` contain `checking acme/api#42 ⣾` and `✗ ... · esc cancel`; unchanged if the `checking` text is unchanged. A new `add-saving` state needs a model built by feeding `addCheckedMsg` (the existing `add-checking` case never runs the Lookup Cmd) and four new goldens |

To build a model in the `saving` state in a test: `m, cmd := submit(model, "octo/tools#42")`, run `cmd` to get the `addCheckedMsg`, call `m.Update(thatMsg)`; the returned model has `add.saving == true` and the returned Cmd is the unrun `saveCmd` (see `drainUI`/`deliverCmd`, add_test.go:85-120).

## Live PR pins (D-13)

All values below were queried on **2026-10-13** with `gh api graphql`, using the exact `prFields` selection from `internal/github/client.go:81-90`, and then mapped by running prv's own `Fetch` path (columns "prv shows"). [VERIFIED: live run, rate cost 4.]

**Rejected candidate:** `prometheus/prometheus#19994` is OPEN, `isDraft: true`, `reviewDecision: REVIEW_REQUIRED`, `mergeStateStatus: BLOCKED`, rollup `FAILURE`, updated 2026-10-12. Every one of those can change tomorrow. Do not pin it. [VERIFIED]

### Recommended pins

| # | Ref | Observed raw (2026-10-13) | prv shows (State / REVIEW / CI / MERGE) | What it proves | Stability |
|---|-----|---------------------------|------------------------------------------|----------------|-----------|
| 1 | `cli/cli#1` | MERGED 2019-10-04; `reviewDecision` null; no reviews; rollup null; mergeStateStatus `DIRTY`, mergeable `CONFLICTING` (stale) | merged / `– none` / `– no checks` / **blank** | MERGE is blank for a merged PR even though GitHub still reports conflicts (RESEARCH F4 in `merge.go`); null decision with no reviews is never "required"; no checks is never "passing" | State, review, CI, MERGE all durable |
| 2 | `cli/cli#6568` | MERGED 2022-11-04; `APPROVED`; one APPROVED review; rollup null | merged / `✓ approved` / `– no checks` / blank | decision `APPROVED` → approved | Durable (APPROVED also holds through the review fall-through) |
| 3 | `komoot/photon#254` | MERGED 2017-12-17; `reviewDecision` **null**; one APPROVED review (`lonvia`); rollup null; no branch protection (REST protection call returns 404) | merged / `✓ approved` / `– no checks` / blank | The Phase 1 UAT #2 case: approved on a repo with no required-review rule shows approved, never required | Durable. If the repo adds a rule the raw value becomes `APPROVED`; prv still shows approved |
| 4 | `cli/cli#8525` | CLOSED 2024-01-10 (not merged); `CHANGES_REQUESTED`; one CHANGES_REQUESTED review; rollup null; stale `BLOCKED`/`MERGEABLE` | closed / `✗ changes` / `– no checks` / blank | closed-not-merged state; changes requested; MERGE blank for closed | State durable unless reopened (Pitfall 2); review durable via fall-through |
| 5 | `cli/cli#11048` | MERGED 2025-06-24; `REVIEW_REQUIRED`; one APPROVED review; rollup null | merged / `○ required` / `– no checks` / blank | decision `REVIEW_REQUIRED` wins over the fall-through | **Soft**: changes if cli/cli removes its review rule (then prv shows approved) |
| 6 | `hashicorp/terraform#17465` | MERGED 2019-07-10; `REVIEW_REQUIRED`; two APPROVED reviews; rollup `SUCCESS` from one legacy `StatusContext`; stale `DIRTY`/`CONFLICTING` | merged / `○ required` / `✓ passing` / blank | CI passing from a rollup that survives (status API, no check runs); decision wins | CI durable so far (statuses, not check runs); review **soft** |
| 7 | `hashicorp/terraform#20036` | MERGED 2020-10-13; `REVIEW_REQUIRED`; one APPROVED review; rollup `PENDING` from one legacy `StatusContext` that never completed | merged / `○ required` / `● running` / blank | `PENDING` → running; CI `PENDING` on a merged PR is a stable oddity | CI durable (a status stuck in pending); review **soft** |
| 8 | `hashicorp/terraform#5` | MERGED 2014-07-01; `reviewDecision` null; no reviews; rollup null | merged / `– none` / `– no checks` / blank | oldest-style PR still maps | Durable |
| 9 | `cli/cli#14327` | MERGED 2026-09-03; `APPROVED`; rollup `FAILURE` (one check run: `govulncheck`, 25 contexts) | merged / `✓ approved` / `✗ failing` (with `Failing.Known`, name `govulncheck`) / blank | CI failing, and the second query (`fetchFailing`) names the check | **Perishable CI** (check runs; expect null from about Oct 2027). Do not assert the check name |
| 10 | `cli/cli#14440` | CLOSED 2026-09-14 (not merged); `CHANGES_REQUESTED`; rollup `SUCCESS`; 39 contexts | closed / `✗ changes` / `✓ passing` / blank | CI passing on a recent PR; closed | **Perishable CI**; reopen risk |
| 11 | `cli/cli#2147483647` | `repository.pullRequest` is null, GraphQL `NOT_FOUND` | unavailable row, `Unavail.Kind == domain.UnavailNotFound`, reason `not found (deleted, renamed, or no access)` | the unavailable-row path against the real API (Phase 4 D-10) | Durable (the number is the GraphQL `Int` maximum; no PR will ever have it) |

Optional extras: `hashicorp/terraform#18448` (CLOSED 2019-10-01, rollup `SUCCESS` status, `REVIEW_REQUIRED`, no reviews) as a second durable closed+passing pin; `cli/cli#13084` (MERGED 2026-04-15, rollup `FAILURE`, `CHANGES_REQUESTED`) as a second failing pin that expires sooner than #9.

The pins cover: MERGED, CLOSED, review `approved` / `changes requested` / `required` / `none`, CI `passing` / `failing` / `running` / `no checks`, MERGE blank, and a not-found row.

### Not coverable with stable fixtures
- **MERGE `mergeable`, `blocked`, `behind`, `conflicts`, draft (open PRs).** Only OPEN PRs get a MERGE badge (`DeriveMerge`: any non-open state returns `MergeNone`, `internal/domain/merge.go:24-27`). I looked for stable OPEN pins: `golang/go#27813` (open since 2018-09-22, `DIRTY`/`CONFLICTING`, prv shows `✗ conflicts`) is the only plausible one, and it drifts if the author rebases or a bot closes it. Do not make it an exact assertion; leave open-PR MERGE to the manual UAT (D-16 "badges + MERGE by eye"). If the user wants an automated tripwire, add it as a soft pin that `t.Skip`s when the state is no longer OPEN.
- **Open PR with live CI, `review requested` reviewers, drafts:** manual UAT.

### Which fields are safe for exact assertions
| Field | Safe? | Why |
|-------|-------|-----|
| `State` of a MERGED PR | yes | Merged is final |
| `State` of a CLOSED PR | mostly | Can be reopened |
| `Merge` of any merged/closed PR | yes | `DeriveMerge` returns `MergeNone` whatever GitHub's stale `mergeStateStatus`/`mergeable` say |
| `Review` derived from `APPROVED` / `CHANGES_REQUESTED` / null with reviews | yes | Holds with or without a rule |
| `Review == ReviewRequired` | soft | Depends on current branch protection |
| `CI == CINone` on pre-2025 PRs | yes | The data is already gone |
| `CI == CIPassing/CIFailing/CIRunning` | perishable except the terraform status-only pins | See Pitfall 3 |
| `Ref`, `URL`, `CreatedAt`, `ClosedAt` | yes (not needed) | Fixed once closed |
| `Title`, `Author`, `UpdatedAt`, `ID` | no | Editable, renameable or drifting; the legacy node IDs may be re-encoded |
| raw `mergeable`, `mergeStateStatus` | no | Not exposed in `domain.PR`; stale on closed PRs and may become `UNKNOWN` |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `O_EXCL` create + copy into the final name | temp + `Sync` + `os.Link` | this phase (D-03) | A partial backup can never carry the final name |
| `syscall.Flock` | `golang.org/x/sys/unix.Flock` | `syscall` frozen since Go 1.4 [ASSUMED] | One `go.mod` line moves |

**Deprecated/outdated:** nothing in this phase relies on a deprecated API.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | GitHub keeps check-run data about 400 days (inferred from a cut-off at 2025-08-28 observed 411 days before 2026-10-13) | Pitfall 3, pins 9-10 | The CI-perishable pins expire at a different date; the skip-with-message design still handles it |
| A2 | `reviewDecision` for closed/merged PRs is computed from the *current* branch-protection rules | Pitfall 4, pins 5-7 | If it is frozen, the "soft" flag is unnecessary but harmless |
| A3 | `os.OpenFile` always sets `O_CLOEXEC` so the lock fd does not leak into child processes | Pattern 1 | Low: the lock file is closed before `Add`/`Remove` returns |
| A4 | Hard links fail with an errno other than `EEXIST` on macOS FAT/exFAT | Pattern 2 | Low: the fail-closed rule handles any non-`EEXIST` error |
| A5 | golden `-update` flag name | Don't Hand-Roll table | Trivial: the executor will see the flag error and check |
| A6 | `syscall` package frozen / x/sys is the maintained replacement | State of the Art | None for the decision |
| A7 | Linux behaves like the macOS run for `flock` polling and `link` (only cross-compiled, plus the man pages) | Patterns 1, 2 | Low: both are POSIX-standard calls documented in the cited man pages. There is no Linux machine or Docker daemon here. The executor should run `go test ./internal/watchlist -race` on Linux if one is available |

## Open Questions (RESOLVED)

1. **Return shape for the "started over" notice**
   - What we know: ~41 test call sites use `(bool, error)`; the UI fake is one struct.
   - What's unclear: whether the user prefers a signature change over added methods.
   - Recommendation: add `AddRef`/`RemoveRef` returning `Result`, keep `Add`/`Remove` as wrappers (Pattern 4).
   - **RESOLVED:** as recommended (CONTEXT Claude's Discretion), adopted in plan 05-01.
2. **Lock timeout value**
   - Recommendation: 3 s total, 10 ms poll. The real hold is a few ms (one fsync). 3 s is "a few seconds" and keeps the spinner from looking stuck.
   - **RESOLVED:** 3 s, 10 ms poll, `Store.lockWait` for tests (plan 05-01).
3. **Version `0`/absent**
   - Recommendation: treat as unparseable (start over with a backup), not as a refusal (Pattern 3).
   - **RESOLVED (superseded by the user, 2026-10-13):** version 0 or an absent version is REFUSED with `VersionError`, per CONTEXT D-04 notes; a version-1 object whose body does not decode starts over with a backup. Pattern 3's edge note and Pitfall 9 are void on this point.
4. **Backup flash duration**
   - Recommendation: use `warnFor` (8 s) for this one flash (Pitfall 8).
   - **RESOLVED:** `warnFor`, adopted in plan 05-02.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | everything | yes | 1.27.2 darwin/arm64 (go.mod says `go 1.26.0`) | — |
| `golang.org/x/sys` v0.47.0 | flock | yes | in module cache and go.sum | — |
| `gh` CLI, logged in to github.com | live test token (`auth.TokenForHost`) | yes | 2.102.0, account `sbartram` (keyring) | `GH_TOKEN` env var |
| Network to api.github.com | live test | yes (queries ran) | rate limit 5000/h, 4998 left at last query | — |
| Linux host or Docker daemon | verifying flock/link on Linux | no | `docker` client present, daemon not running | Cross-compile (`GOOS=linux go build ./internal/watchlist` passes); rely on the cited man pages |
| golangci-lint / staticcheck | lint | no (`which golangci-lint` empty) | — | `go vet ./...` |

**Missing dependencies with no fallback:** none.

## Validation Architecture

Skipped: `workflow.nyquist_validation` is `false` in `.planning/config.json`.

For the planner's own use, the existing commands are `go test ./... -race -count=1` (all packages pass today; `internal/ui` takes about 26 s) and, for the new work, `go test ./internal/watchlist -race -count=1`, `go test ./internal/ui -run 'TestAdd|TestRemove|TestGolden' -count=1`, and `PRV_LIVE=1 go test ./internal/github -run TestLive -count=1 -v`.

## Security Domain

`security_enforcement` is on (ASVS level 1, block on high).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no (token comes from `gh`, unchanged) | — |
| V3 Session Management | no | — |
| V4 Access Control | yes (local files) | Data dir 0700, `watchlist.json`, the lock file and backups 0600 (`CreateTemp` gives 0600; `OpenFile(..., 0o600)` for the lock; a hard link keeps the inode's mode) |
| V5 Input Validation | yes | `domain.ParseRef`/`ValidRef` unchanged; all error text shown goes through fixed phrases or `domain.Clean` |
| V6 Cryptography | no (SHA-256 is only a file-name checksum) | — |
| V12 Files and Resources | yes | Temp + `Sync` + rename/link; no symlink following added (the lock path is `<own 0700 dir>/watchlist.json.lock`) |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Lost update by concurrent writers (WR-01) | Tampering | flock for the whole read-modify-write |
| Data loss from clobbering a newer-version file (WR-03) | Tampering / DoS | Refuse to write on version mismatch |
| Partial backup treated as a complete one (WR-02) | Tampering | Publish with `os.Link` only after `Sync` |
| Path or OS text in the footer (IN-02) | Information disclosure | Map to fixed phrases; `domain.Clean` for anything left |
| Lock held forever by a hung process | DoS | Deadline then `ErrBusy`; the kernel drops the lock if the holder dies |
| Token leaking through the live test | Information disclosure | Never log the token, headers or raw bodies; `LogIgnoreEnv: true` is already set in `NewClient`; run read-only queries only |
| Live test writing to GitHub | Tampering | The test calls only `Fetch`/`Lookup`, which send read-only queries (`buildLookup` doc: "The query is read-only") |

## Sources

### Primary (HIGH confidence)
- Local source read this session: `internal/watchlist/watchlist.go` (all), `internal/ui/add.go` (all), `internal/ui/remove.go` (all), `internal/ui/model.go:300-345`, `internal/ui/refresh.go:80-101`, `internal/ui/detail.go:150-200`, `internal/github/client.go`, `lookup.go`, `checks.go`, `internal/domain/merge.go`, `status.go`, `pr.go`, `list.go`, `go.mod`, the Phase 4 `04-REVIEW.md`, `04-REVIEW-DISPOSITION.md`, `02-UAT.md`.
- Live `gh api graphql` queries and a throwaway Go test calling prv's own `Fetch`/`Lookup`, 2026-10-13 (file deleted afterwards; `git status` shows only the pre-existing changes).
- A scratch copy of the repo (outside the project tree) where `lock.go`, the new `backup`, and the multi-process test were built and run with `-race`, with and without the lock.
- `/Users/scottb/go/pkg/mod/golang.org/x/sys@v0.47.0/unix/` (Flock and errno constants).

### Secondary (MEDIUM confidence)
- https://man7.org/linux/man-pages/man2/flock.2.html — LOCK_NB, EWOULDBLOCK, EINTR, open-file-description semantics, release on close, NFS.
- https://man7.org/linux/man-pages/man2/link.2.html — EEXIST, "not be overwritten", EPERM for no hard-link support, EXDEV, EMLINK.

### Tertiary (LOW confidence)
- None used. Items inferred from observation are listed in the Assumptions Log.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependency; module already linked.
- Architecture / code shapes: HIGH — prototyped and run against the existing tests.
- Pitfalls: HIGH for lock, backup, UI wiring; MEDIUM for the GitHub data-retention rule (observed, mechanism inferred).
- Live pins: HIGH for today's values; CI perishability explained and designed for.

**Research date:** 2026-10-13
**Valid until:** 2026-11-12 for the code shapes; re-verify the pins' CI values at execution time with the commands in the Live PR pins section (the check-run pins can expire at any point after the dates given).

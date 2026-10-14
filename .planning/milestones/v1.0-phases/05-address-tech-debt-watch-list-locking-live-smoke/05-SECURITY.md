---
phase: "05"
slug: "address-tech-debt-watch-list-locking-live-smoke"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-10-14"
---

# Phase 05 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| prv process ↔ prv process | Two or more prv instances (multiplexer panes) write the same watchlist.json | Watch-list refs (non-sensitive) |
| local file → prv | watchlist.json can be hand-edited, truncated, written by a newer prv, or replaced | Untrusted JSON bytes |
| prv → local file | prv writes watchlist.json, watchlist.json.lock, .corrupt-*.tmp, watchlist.json.corrupt-HASH8 in the 0700 data dir | Watch-list refs, backup copies |
| store error → screen | Store/OS errors (paths, OS messages, possibly hostile file names) reach the ui | Error text (may carry paths) |
| user keys → add prompt | esc, typed keys and paste arrive while a save is in flight | Key events |
| test process → api.github.com | Opt-in live test sends authenticated GraphQL queries | GitHub token (secret) |
| github.com → test output | Public PR data (titles, logins, check names) reaches the test | Untrusted remote strings |
| token store → test | auth.TokenForHost reads GH_TOKEN, GITHUB_TOKEN, hosts.yml or `gh auth token` | GitHub token (secret) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-05-01 | Tampering (lost update) | Store.AddRef / RemoveRef | high | mitigate | `internal/watchlist/watchlist.go:314,346` take `s.lock()` (exclusive `unix.Flock` on `.lock`, `lock.go:42`) around read-modify-write incl. rename; `TestMultiProcessAdd` (`lock_test.go:62`) passes | closed |
| T-05-02 | Denial of service | Store.lock | medium | mitigate | `LOCK_NB` poll every 10 ms, `defaultLockWait = 3s` then `ErrBusy` (`lock.go:15,19,42-53`); `Load` never locks; `TestLoadWhileLocked` passes | closed |
| T-05-03 | Tampering (partial backup) | Store.backup | medium | mitigate | `CreateTemp` → write → `Sync` → `Close` → no-clobber `s.link` to `corrupt-<sha256[:8]>`; non-EEXIST link error returns error (`watchlist.go:211-234`); `TestBackupPublishedComplete`, `TestBackupLinkFails` pass | closed |
| T-05-04 | Tampering / DoS (data loss) | Store.read / readForWrite | high | mitigate | `classify` sets `VersionError{Missing}`/`{Found}` (`watchlist.go:149-161`); `readForWrite` returns it and nothing is written (`:284-293`); only non-version corruption starts over after backup; `TestVersionMismatchRefused`, `TestNonObjectStartsOver`, `TestAddOverCorrupt` pass | closed |
| T-05-05 | Information disclosure | lock and backup files | low | mitigate | Lock opened `0o600` (`lock.go:36`), dir `0o700` (`lock.go:33`), backups via `os.CreateTemp` (0600, `watchlist.go:212`); `TestLockFileKept`, `TestPermissions` pass | closed |
| T-05-06 | Tampering (symlink at lock path) | Store.lock | low | accept | See AR-05-01 | closed |
| T-05-07 | Information disclosure | TestLive output | high | mitigate | Token only passed to `api.ClientOptions.AuthToken` and `github.Options.Token` (`live_test.go:195,205`); no log/error formats the token; `LogIgnoreEnv: true` in both clients (`live_test.go:195`, `client.go:63`) | closed |
| T-05-08 | Tampering | TestLive → github.com | medium | mitigate | Test calls only `Fetch` and the read-only `reviewDecision` query; no `mutation` in `internal/github/` | closed |
| T-05-09 | Denial of service (rate budget) | TestLive | low | accept | See AR-05-02 | closed |
| T-05-10 | Tampering (hostile remote strings) | pinned PR data | low | accept | See AR-05-03 | closed |
| T-05-11 | Information disclosure | applyAddSaved / applyRemoved | medium | mitigate | `saveReason` (`internal/ui/add.go:286-302`) maps errors to fixed phrases, only `VersionError` text is dynamic, all through `domain.Clean`; used by add (`:247`) and remove (`remove.go:81`); `ErrFull` uses its fixed message; `TestAddSaveReasons` passes | closed |
| T-05-12 | Repudiation (UI/file disagree) | addKey while saving | medium | mitigate | `case m.add.saving: return nil` ahead of Cancel/typing (`add.go:117-120`); `TestAddEscIgnoredWhileSaving` passes | closed |
| T-05-13 | Repudiation (silent data loss) | applyAddSaved notice | medium | mitigate | `startedOverNotice` → "previous list saved to …" for `warnFor` (`add.go:307-312`), joined first by `joinNotice`; `TestAddOverCorruptEndToEnd`, `TestAddVersionMismatchEndToEnd` pass | closed |
| T-05-14 | Information disclosure | TestShowRateWarning output | low | accept | See AR-05-04 | closed |
| T-05-SC | Tampering (supply chain) | Go module installs | high | mitigate | `git diff 0f4babd -- go.sum` empty; go.mod only moves `golang.org/x/sys v0.47.0` from `// indirect` to direct; no new modules across plans 05-01..03 | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-05-01 | T-05-06 | Lock path is inside the user's own 0700 data directory (XDG_DATA_HOME or ~/.local/share/prv); another account cannot plant a symlink there; no new path source added | plan 05-01 (user-approved plan) | 2026-10-14 |
| AR-05-02 | T-05-09 | Live test is opt-in (`PRV_LIVE=1`), ~4 points/run (+≤10 on Review mismatch) of 5,000/h; never runs in `go test ./...`; no CI schedule (D-14) | plan 05-03 (user-approved plan) | 2026-10-14 |
| AR-05-03 | T-05-10 | Remote text goes through existing `domain.Clean`; the test asserts enum values only and never prints titles or logins | plan 05-03 (user-approved plan) | 2026-10-14 |
| AR-05-04 | T-05-14 | Opt-in (`PRV_SHOW=1`); prints fixture data with a fixed clock; reads no token, file or network | plan 05-03 (user-approved plan) | 2026-10-14 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-14 | 15 | 15 | 0 | /gsd-secure-phase orchestrator (ASVS L1 grep-depth short-circuit, block_on high; cited tests re-run green) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-14

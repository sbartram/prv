---
phase: "04"
slug: "curated-watch-list"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-10-13"
---

# Phase 04 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| keyboard / paste → prv | Pasted or typed text can hold escape sequences, bidi characters, newlines or megabytes of data | Untrusted text that becomes owner, repo and number |
| prv → GitHub API | Validated owner, repo and number travel as GraphQL variables | Authenticated read-only GraphQL requests |
| GitHub API → prv | nameWithOwner, titles, logins and `errors[]` messages, types, paths and extensions are drawn on screen as data and reasons | Untrusted display text |
| prv ↔ local file | watchlist.json is written by prv and may be edited by hand, by another prv pane, or replaced by anything with the user's file access | Saved refs (repo, number, added_at) |
| keyboard → confirm line | A stray key must never remove data | Key presses |
| prv → browser | `o` opens a URL built from a saved ref | URL |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-04-01 | Tampering (query injection) | buildLookup / lookupVars | high | mitigate | Variables only; `TestAddEndToEnd` asserts no pasted value in the query text | closed |
| T-04-02 | Tampering / Spoofing (terminal escapes) | addKey / addPaste / addLine | high | mitigate | `domain.Clean` after CR/LF → space; `maxAddRunes = 256` (internal/ui/add.go:31) | closed |
| T-04-03 | Tampering / Spoofing (terminal escapes) | mapNode via lookupRefs | high | mitigate | Lookup reuses `mapNode`, which cleans every remote string at the github seam | closed |
| T-04-04 | Information disclosure | watchlist.json | low | mitigate | Refs only; dir 0o700, file via `os.CreateTemp` (0600) (internal/watchlist/watchlist.go:277-280) | closed |
| T-04-05 | Tampering (data loss) | Store.Add / Store.Remove | medium | mitigate | Read before write, temp file + rename; invalid entries kept | closed |
| T-04-06 | Tampering (query injection) | ParseRef / buildLookup | high | mitigate | Anchored regexps, `MaxPRNumber = 2147483647`; `TestBuildLookupVariablesOnly`, `TestParseRef` | closed |
| T-04-07 | Tampering / Spoofing (terminal escapes) | unavailFor Reason and Help | high | mitigate | Fixed strings, validated owner, cleaned and cut messages; `TestLookupCleans` | closed |
| T-04-08 | Information disclosure | SSO handling / captureTransport | high | mitigate | Only Retry-After / X-RateLimit-Remaining / X-RateLimit-Reset read (internal/github/ratelimit.go); SSO URL built from validated owner | closed |
| T-04-09 | Information disclosure | pasted URL query string and fragment | low | mitigate | ParseRef returns owner, repo, number only; `TestParseRef` `?token=abc#frag` row | closed |
| T-04-10 | Denial of service | one bad ref in a chunk | low | mitigate | Out-of-range refused before request; per-alias errors; `TestLookupReasons` | closed |
| T-04-11 | Tampering / Spoofing (terminal escapes) | addKey / addPaste | high | mitigate | `TestAddPasteSanitised` | closed |
| T-04-12 | Denial of service | addPaste | low | mitigate | 256-rune buffer cap; line cut to pane width | closed |
| T-04-13 | Tampering (integrity of what is shown) | fetchedMsg handler | low | mitigate | Epoch rule; `TestStaleFetchAfterAdd` | closed |
| T-04-14 | Repudiation (silent save) | applyAddChecked / applyAddSaved | medium | mitigate | Sequence numbers; `TestAddEscDropsLateResult`, `TestAddSaveError` | closed |
| T-04-15 | Tampering / Denial of service | Store.Load | medium | mitigate | `io.LimitReader` 1 MiB cap, `ValidRef`, 500-entry cap; `TestTooLarge`, `TestBadEntriesKept`, `TestMaxEntries` | closed |
| T-04-16 | Repudiation (data loss) | Store.read / backup | medium | mitigate | O_EXCL hash-named backup (watchlist.go:136); `TestCorruptBackup`, `TestUnreadableNotOverwritten` | closed |
| T-04-17 | Information disclosure | watchlist.json permissions | low | mitigate | `TestPermissions`, `TestRoundTrip` | closed |
| T-04-18 | Tampering (lost update) | two panes | low | accept | Reload-apply-write per change (`TestTwoStoresInterleave`); residual window accepted — see Accepted Risks Log | closed |
| T-04-19 | Tampering | path handling | low | mitigate | Path only from XDG_DATA_HOME or home; temp file in same dir via `os.CreateTemp` | closed |
| T-04-20 | Spoofing (misleading status) | Reconcile | medium | mitigate | Two-strike rule; `TestReconcile`, `TestTwoStrikeRule` | closed |
| T-04-21 | Tampering / Spoofing (terminal escapes) | unavailLine | high | mitigate | Reason from domain constructors; `ansi.Truncate` per styled segment | closed |
| T-04-22 | Repudiation (silent loss) | Visible / InWindow | medium | mitigate | `TestUnavailableFilters`, `TestInWindowManual` | closed |
| T-04-23 | Tampering (unintended removal) | confirmKey | medium | mitigate | Only `y` removes; `TestRemoveConfirmCancels`, `TestRemoveAutoOnly` | closed |
| T-04-24 | Repudiation | applyRemoved | low | mitigate | Every outcome in footer; failed write keeps row; `TestRemoveError` | closed |
| T-04-25 | Tampering (integrity of what is shown) | fetchedMsg epoch rule | low | mitigate | `markMutation` after removal; `TestRemoveStaleFetch` | closed |
| T-04-26 | Spoofing (attacker-chosen URL) | `o` on an unavailable row and its detail | medium | mitigate | URL from `Ref.URL` of a `ValidRef` ref, then `safeURL` (internal/ui/open.go:24); `TestOpenUnavailableRow` | closed |
| T-04-27 | Tampering / Spoofing (terminal escapes) | unavailLines | high | mitigate | Cleaned text; every detail line cut with `ansi.Truncate` (internal/ui/detail.go:237) | closed |
| T-04-28 | Denial of service (wasted requests) | startDetailFetch | low | mitigate | No Detail request for an unavailable row, also on `r`; `TestDetailUnavailable` | closed |
| T-04-SC | Tampering | Go module installs | high | mitigate | No new modules: `git diff 86c81a7^ HEAD -- go.mod go.sum` is empty | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-04-01 | T-04-18 | Two prv panes can lose an update in the millisecond window between re-read and rename. A lock file can go stale and flock is not portable (04-RESEARCH Alternatives); reload-apply-write per change keeps both panes' edits otherwise. | plan 04-04 (planner), confirmed at audit | 2026-10-13 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-13 | 29 | 29 | 0 | /gsd-secure-phase (L1 grep-depth; register authored at plan time) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-13

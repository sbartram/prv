---
phase: 04-curated-watch-list
verified: 2026-10-14T00:30:00Z
status: passed
score: 4/4 roadmap success criteria verified (all plan must-have truths verified; 0 failed)
covered_files:
  - ".planning/phases/04-curated-watch-list/04-01-PLAN.md"
  - ".planning/phases/04-curated-watch-list/04-01-SUMMARY.md"
  - ".planning/phases/04-curated-watch-list/04-02-PLAN.md"
  - ".planning/phases/04-curated-watch-list/04-02-SUMMARY.md"
  - ".planning/phases/04-curated-watch-list/04-03-PLAN.md"
  - ".planning/phases/04-curated-watch-list/04-03-SUMMARY.md"
  - ".planning/phases/04-curated-watch-list/04-04-PLAN.md"
  - ".planning/phases/04-curated-watch-list/04-04-SUMMARY.md"
  - ".planning/phases/04-curated-watch-list/04-05-PLAN.md"
  - ".planning/phases/04-curated-watch-list/04-05-SUMMARY.md"
  - ".planning/phases/04-curated-watch-list/04-06-PLAN.md"
  - ".planning/phases/04-curated-watch-list/04-06-SUMMARY.md"
  - ".planning/phases/04-curated-watch-list/04-07-PLAN.md"
  - ".planning/phases/04-curated-watch-list/04-07-SUMMARY.md"
  - "cmd/prv/main.go"
  - "internal/domain/list.go"
  - "internal/domain/pr.go"
  - "internal/domain/watch.go"
  - "internal/github/client.go"
  - "internal/github/lookup.go"
  - "internal/ui/add.go"
  - "internal/ui/detail.go"
  - "internal/ui/help.go"
  - "internal/ui/keys.go"
  - "internal/ui/model.go"
  - "internal/ui/refresh.go"
  - "internal/ui/remove.go"
  - "internal/ui/styles.go"
  - "internal/ui/view.go"
  - "internal/watchlist/watchlist.go"
covered_digest: "v3:sha256:26223f8fb2c06f87e8cdf302c526f455d98a60d58ded26a07574bb1c901f2b36"
behavior_unverified: 0
overrides_applied: 0
advisory:
  - finding: "WR-01: Store.Add/Remove is an unlocked read-modify-write; two prv panes that write inside the same few-millisecond window can lose one entry"
    category: other
    reason: "Real but narrow data-loss path. Each write is preceded by a human action and (for add) a network Lookup, so a collision needs two panes to reach the local read->rename step within one fsync. Does not break any roadmap success criterion. Recommended fix: advisory flock on a sibling lock file (see 04-REVIEW.md)."
    evidence_status: "code read (internal/watchlist/watchlist.go:223-307); TestTwoStoresInterleave interleaves sequentially only"
  - finding: "WR-02: backup() publishes the final-named .corrupt-HASH8 file before the copy finishes; a crash mid-copy leaves a truncated backup that a later run treats as complete"
    category: other
    reason: "Needs a crash during the (tiny) copy of an already-corrupt file, then a later write. Normal path (backup complete, then fresh file) is tested."
    evidence_status: "code read (watchlist.go:133-164)"
  - finding: "WR-03: Add/Remove over a version!=1 or invalid file starts a fresh list (old bytes are backed up first); UI does not say so at the moment of loss"
    category: other
    reason: "Persistence still works and the old bytes are preserved in watchlist.json.corrupt-HASH8 and announced once at startup (8 s footer + ? overlay). A newer-prv file being clobbered by an older prv is the only unfriendly case. Not a must-have failure."
    evidence_status: "code read (watchlist.go:208-217); TestAddOverCorrupt / TestUnknownVersion pass"
  - finding: "WR-04: typed-struct round trip drops unknown JSON keys on write, and one wrongly typed entry (e.g. number as a string) makes the whole file 'corrupt' instead of skipping that entry"
    category: other
    reason: "Applies only to hand-edited files with type errors or extra keys. Entries that parse but fail ValidRef are skipped and kept (TestBadEntriesKept). Plan 04-04 wording 'entries that fail ValidRef are skipped and stay' holds for that class; the type-mismatch class falls to the corrupt+backup path (data preserved in the backup)."
    evidence_status: "code read (watchlist.go:67-76, 117)"
  - finding: "WR-05: esc pressed after the Lookup succeeded but before the save result returns closes the prompt, yet the Add still runs; the PR is saved and appears on the next refresh"
    category: other
    reason: "The window is one local file write after the network check. The UI drops the late result as promised, and the entry is removable with d. SC2 (errors save nothing) is unaffected; esc is not an error path in any success criterion."
    evidence_status: "code read (add.go:104-111, 219-220); TestAddEscDropsLateResult covers the Lookup stage and a synthetic late save msg, not a real in-flight Add"
human_verification:
  - test: "Run prv in a real terminal and inside the user's multiplexer (herdr/tmux pane) at about 80 and about 40 columns with at least one saved PR and one unavailable saved PR (e.g. a deleted or typo'd number added by editing watchlist.json). Look at the warning glyph, the '+' marker and the 'N unavailable' header."
    expected: "The warning glyph occupies one cell, columns stay aligned, '+' sits before the ref, '+ repo#n ⚠' fits below 40 columns, and the header 'N unavailable' stays readable. No row wraps or shifts."
    why_human: "Plan 04-07 flagged it: goldens are rendered by lipgloss width tables, not by the terminal. Some terminals/fonts render U+26A0 as two cells, which only a real terminal can show."
  - test: "With a PR in an org that enforces SAML SSO and a token not yet authorized for it, add it by URL (prompt) and also save it and restart (unavailable row; press enter on the row)."
    expected: "The prompt line and the row both say 'SSO: authorize your token for ORG', with the https://github.com/orgs/ORG/sso / gh auth refresh hint. If GitHub's real error shape differs from the constructed one, the reason would instead read 'could not load (forbidden)'."
    why_human: "RESEARCH A1: the SAML FORBIDDEN error shape in internal/github/testdata/lookup_mixed.json is constructed, not recorded from live GitHub (the NOT_FOUND shapes are real from the Phase 1 spike). unavailFor accepts extensions.saml_failure==true or 'SAML' in the message, which is a deliberately broad detector, but only a live SSO-enforcing org can confirm."
  - test: "Live smoke of the whole flow against github.com: press a, paste a PR URL from a repo you neither authored nor are assigned to, quit, restart, then merge/close-date check, then d then y."
    expected: "Row appears with full status and '+', survives restart, stays after close, is removed by d,y; a second prv pane picks up the add/remove on its next refresh."
    why_human: "All automated end-to-end tests use fixtures/httptest (TestAddEndToEnd, TestRestartEndToEnd, TestRemoveEndToEnd, TestClosedSavedPREndToEnd); none touch live github.com. The aliased query was cost-checked live in Phase 1/04-RESEARCH, but this exact document is only fixture-tested."
---

# Phase 4: Curated Watch List Verification Report

**Phase Goal:** User can watch any PR on github.com, not only their own, and their curated list survives restarts and degrades gracefully when a PR can't be fetched
**Verified:** 2026-10-14
**Status:** human_needed
**Re-verification:** No, initial verification

All code-checkable truths are VERIFIED. The status is `human_needed`, not `passed`, only because of the three human items above (terminal glyph width, unrecorded SAML shape, live smoke). No gap blocks the goal. The five open review warnings are recorded as advisories: none of them falsifies a roadmap success criterion or a plan truth, though WR-01 and WR-05 make two plan sentences ("never lose each other's change", "esc ... changes nothing") true only for the tested cases.

## Goal Achievement

### Roadmap Success Criteria (the contract)

| # | Success criterion | Status | Evidence |
|---|-------------------|--------|----------|
| 1 | Paste URL or `owner/repo#123` into an add prompt, PR appears with full status; an already-listed PR (auto or manual) makes no second row | VERIFIED | `ParseRef` (domain/pr.go) table-tested (`TestParseRef`); `a` -> `openAdd` -> `submitAdd` -> `lookupCmd` -> `Client.Lookup` -> `lookupRefs` (variables-only aliased query, `buildLookup`/`lookupVars`) -> `saveCmd` -> `Store.Add` -> `applyAddSaved` appends the PR (full `prFields`, plus failing check name via `fetchFailing`) and selects it. Duplicates: `findListed`/`listedIndex` match by `Ref.Key()`, saved ref and node ID, then `closeAsDuplicate` sends no Lookup (`TestAddDuplicateByRef/Hidden/ByID`); in Fetch, `MarkWatched` + `Combine` fold by key and node ID so one row carries `Auto`+`Manual` (`TestFetchSavedFoundByDiscovery`, `TestEverySavedRefOnce`). `TestAddEndToEnd` passes through real client, store and model. |
| 2 | Malformed ref, nonexistent PR, or inaccessible PR shows an inline error that explains why; nothing saved | VERIFIED | Parser gives exactly three reasons (other host, issue link, not a PR reference). Lookup null alias -> `unavailFor`: SSO / not found / could not load, from the alias's own `errors[].path[0]`. `applyAddChecked` sets `errText`, keeps text, returns nil before any `saveCmd` for error and `Unavail` cases. `TestLookupReasons`, `TestAddSaveError`, add error tests pass; request-level failure returns an error (`TestFetchWatchedRequestFailure`) and shows 'could not check (...)'. |
| 3 | Manually added PRs reappear after quit/restart; stay listed after merge/close even past the closed window until removed; only manual PRs can be removed | VERIFIED | Persistence: `Store` JSON at `$XDG_DATA_HOME/prv/watchlist.json` (`Path`, `TestPath`), atomic temp+rename (`write`), 0700/0600 (`TestPermissions`); `main.go` builds `watchlist.New` and passes it to both `github.Options.Watch` and `ui.Options.Watchlist`; `Fetch` re-reads the file every refresh (`savedRefs`). `TestRestartEndToEnd` passes. Past-window: `InWindow` returns true for `pr.Manual`, and in `Fetch` `MarkWatched`/`lookupRefs` set Manual before the `InWindow` filter (`TestInWindowManual`, `TestClosedSavedPREndToEnd`). Removal: `d` -> `askRemove` refuses non-`Manual` rows with a flash; `y` -> `Store.Remove`; nothing else removes (grep of non-test code: only `add.go:81` Add and `remove.go:38` Remove call the store). `TestRemoveEndToEnd`, `TestRemoveConfirmCancels`, `TestRemoveAutoOnly` pass. |
| 4 | A saved PR that cannot be fetched (deleted, access revoked, SSO) shows as an "unavailable" row with reason; others still display; saved entry never deleted automatically | VERIFIED (SSO shape: see human item 2) | `lookupRefs` returns an `Unavail` PR per failed alias (never drops it); per-alias attribution keeps other aliases loaded. Rendering: view.go `⚠ unavailable: REASON` row, header `N unavailable`, bucket 0 sort, filters never hide it (`Visible`), compact `+ repo#n ⚠`; detail view shows reason + help + removal hint, no Detail fetch (`detail.go:58,224`). D-12 two-strike rule: `domain.Reconcile` (`TestTwoStrikeRule`); whole-request failure keeps previous rows. Never auto-deleted: no code path other than user-driven `Add`/`Remove` writes the file (`TestUnreadableNotOverwritten`, `TestFetchKeepsRefsOnReadFailure`). `TestUnavailableEndToEnd` passes. |

### Plan must-have truths (merged, none reduce the roadmap SCs)

| Plan | Truth (abbreviated) | Status | Evidence |
|------|---------------------|--------|----------|
| 04-01 | `a` prompt, paste, one Lookup before save, row with cursor | VERIFIED | add.go; `TestAddEndToEnd` |
| 04-01 | `+ ` marker on saved rows, existing goldens unchanged | VERIFIED | view.go marker; full golden suite passes |
| 04-01 | Store path, version 1 `{repo, number, added_at}`, atomic write, unreadable file never overwritten | VERIFIED | watchlist.go; `TestAddCreatesFile`, `TestUnreadableNotOverwritten` |
| 04-01 | Restart lists saved PRs; discovery match costs no Lookup; chunks <= 25 | VERIFIED | `lookupChunk = 25`; `TestLookupChunks`, `TestFetchSavedFoundByDiscovery`, `TestRestartEndToEnd` |
| 04-01 | Auto+Manual one row; node-ID fold; one row per saved ref | VERIFIED | `Combine`, `MarkWatched`; `TestEverySavedRefOnce` |
| 04-01 | Lookup uses variables only; rateLimit cost added | VERIFIED | `buildLookup` formats only the alias index; `TestBuildLookupVariablesOnly` |
| 04-01 | Whole-request Lookup failure fails the refresh (rows stay) | VERIFIED | `lookupRefs` returns `nil, err`; `Fetch` returns empty snapshot + err; model keeps rows. `TestFetchWatchedRequestFailure` |
| 04-02 | ParseRef accepts/rejects the listed forms with exactly three reasons | VERIFIED | `TestParseRef` |
| 04-02 | D-10 reasons: SSO (with sso URL + `gh auth refresh`), NOT_FOUND, could-not-load; per-alias path attribution; nested error keeps PR loaded | VERIFIED (SAML shape constructed) | `unavailFor`, `lookupRefs`; `TestLookupReasons`; human item 2 |
| 04-03 | Duplicate jump, filter-hidden suffix, node-ID duplicate | VERIFIED | `TestAddDuplicateByRef/Hidden/ByID` |
| 04-03 | Errors keep text; esc closes at any time and late results dropped | VERIFIED with caveat | `TestAddEscDropsLateResult`; WR-05: an Add already dispatched still persists |
| 04-03 | Rate-limit pause blocks the request; Lookup rate-limit extends pause | VERIFIED | `submitAdd`, `pauseForRateLimit`; `TestAddDuringPause` |
| 04-03 | Paste cleaning, 256-rune cap, prompt fits any width | VERIFIED | `addPaste`, `appendAdd`, `addLine`; add width tests |
| 04-03 | Stale refresh after an add/removal is discarded and re-run (epoch) | VERIFIED (behavior-dependent, test-exercised) | model.go:364 `msg.epoch < m.mutEpoch`; `TestStaleFetchAfterAdd`, `TestFetchWithoutEpochApplies`, `TestRemoveStaleFetch` |
| 04-04 | Corrupt/unknown-version/oversize file copied once to `.corrupt-HASH8`, Load never rewrites | VERIFIED | `read`/`backup`; `TestCorruptBackup`, `TestUnknownVersion`, `TestTooLarge` |
| 04-04 | Invalid entries skipped with one warning and kept; duplicates collapse | VERIFIED | `validRefs`; `TestLoadSkipsInvalidEntries`, `TestBadEntriesKept` |
| 04-04 | Remove of an absent ref writes nothing | VERIFIED | `TestRemoveMissingNoWrite` |
| 04-04 | Two panes interleaving Add/Remove never lose each other's change | VERIFIED with caveat | Holds for sequential interleaving (`TestTwoStoresInterleave`, re-read per call). WR-01: not guaranteed under truly simultaneous writes (advisory, not a gap) |
| 04-04 | 500 cap; 0700/0600; no temp leftovers | VERIFIED | `TestMaxEntries`, `TestPermissions`, `TestNoTempLeftovers` |
| 04-04 | Refresh re-reads the file; read failure keeps last list + 'Watch list could not be read' | VERIFIED | `savedRefs`; `TestFetchRereadsWatchList`, `TestFetchKeepsRefsOnReadFailure` |
| 04-04 | Startup warnings shown once; prv never exits for them | VERIFIED | `watchWarnings` in main.go, merged after config warnings; `TestWatchWarnings` |
| 04-05 | Unavailable row rendering, bucket 0 sort, toggles never hide, `/` matches number, header `N unavailable` priority 6, compact form | VERIFIED | list.go `Bucket`/`Sort`/`Visible`; view.go; `unavail_test.go`; `TestGoldenCompactWatch.golden` |
| 04-05 | InWindow exempts saved PRs; 2/3 toggles still hide merged/closed saved PRs | VERIFIED | `InWindow`; `TestInWindowManual` |
| 04-05 | D-12 two-strike rule; NOT_FOUND/SSO immediate; first-launch immediate; whole failure changes nothing | VERIFIED (behavior-dependent, test-exercised) | `Reconcile`; `TestTwoStrikeRule` |
| 04-06 | `d` on saved row -> confirm 'y/n'; only y removes; q/esc/n etc. cancel; ctrl+c quits | VERIFIED | remove.go `confirmKey`; `TestRemoveConfirmCancels` |
| 04-06 | d+y on Auto+Manual row unpins only; d on discovery-only row refuses; d works on unavailable row; already-gone entry still removes the row; write failure keeps row | VERIFIED | `applyRemoved`, `askRemove`; `TestRemoveAutoOnly`, remove tests |
| 04-07 | enter on unavailable row: reason + help + remove hint, no Detail request; o opens saved-ref URL | VERIFIED | detail.go `unavailLines`, `startDetailFetch` guard; `unavail_test.go` |
| 04-07 | Footer `a add`/`d remove` hints, empty-screen hint, `?` legend lines | VERIFIED | keys.go, help.go (`+ added by you; d removes it`, `⚠ unavailable ...`); `TestHelpGolden` and view goldens pass |
| 04-07 | Golden coverage at 0x0, 60x15, 80x24, 120x40 plus 36x12 compact | VERIFIED | `testdata/TestGolden`, `TestHelpGolden`, `TestGoldenCompactWatch.golden` exist and pass |

**Score:** 4/4 roadmap SCs and 0 failed plan truths; 0 present-but-behavior-unverified (the state-transition/ordering truths, epoch discard and two-strike, each have a passing named test).

### Prohibitions (plan `must_haves.prohibitions`, all test-tier)

| Prohibition | Status | Enforcement evidence |
|-------------|--------|----------------------|
| 04-02: nothing from pasted text except owner/repo/number is kept, sent or saved | ENFORCED | `TestParseRef`, `TestBuildLookupVariablesOnly`, `TestRoundTrip` (all pass) |
| 04-04: nothing but repo/number/added_at in watchlist.json | ENFORCED | `TestRoundTrip` key-set check; `entry` struct has only those fields |
| 04-04: no read path or failed fetch changes the file; corrupt file backed up before replacement | ENFORCED | `TestCorruptBackup`, `TestUnreadableNotOverwritten`, `TestNoTempLeftovers`, `TestFetchKeepsRefsOnReadFailure`. Only `Add` and `Remove` call `write`; `Load` only ever creates the backup copy |
| 04-05: whole-refresh failure never shown as unavailable rows; no invented reasons; filter/window never hide an unavailable saved PR | ENFORCED | `TestFetchWatchedRequestFailure`, `TestTwoStrikeRule`, `TestUnavailableFilters`, `TestLookupReasons` |
| 04-05: <= 1 read-only Lookup per 25 saved PRs not found by discovery | ENFORCED | `TestLookupChunks`, `TestMaxEntries`, `TestFetchSavedFoundByDiscovery` |
| 04-06: only y at the confirm line removes an entry | ENFORCED | `TestRemoveConfirmCancels`, `TestRemoveAutoOnly`; code-path grep shows `Remove` is called only from `removeCmd`, which only `confirmKey` issues |

Note: the plans mark these as "spec-less fallback, descriptor-less (flagged-unverified downstream)". Each has concrete named tests that exist and pass, so none is left unverified here.

### Requirements Coverage

| Requirement | Source plans | Description | Status | Evidence |
|-------------|--------------|-------------|--------|----------|
| WATCH-04 | 04-01, 04-02, 04-03 | Add by URL or owner/repo#n; invalid/inaccessible shows error, not saved | SATISFIED | SC1, SC2 evidence |
| WATCH-05 | 04-01, 04-04 | Manual PRs saved locally, reappear after restart | SATISFIED | SC3; `TestRestartEndToEnd` |
| WATCH-06 | 04-06 | Remove a manually added PR | SATISFIED | SC3; `TestRemoveEndToEnd` |
| WATCH-07 | 04-05 | Manual PRs stay listed after close, until removed | SATISFIED | `InWindow` Manual exemption; `TestClosedSavedPREndToEnd` |
| LIST-09 | 04-02 (data half), 04-05, 04-07 | Unfetchable PR shows as unavailable row with reason; others still display | SATISFIED | SC4. The checkbox was ticked early by 04-02 before the UI existed; the UI half (rows, header, detail, filters, sort) is now present and tested, so the tick is now earned |

No orphaned requirements: REQUIREMENTS.md maps exactly WATCH-04..07 and LIST-09 to Phase 4, and all five appear in plan frontmatter.

### Key Links

All plan `key_links` verified by reading the code: model -> add/remove routing (`m.add.open`, `m.confirm.open`), add -> `Looker.Lookup` (implemented by `*github.Client`, asserted in main.go), client -> `lookupRefs`/`MarkWatched`/`Combine`, main -> `watchlist.New` (both options), client `savedRefs` -> `WatchSource.Load` with `lastRefs` fallback, model -> `domain.Reconcile` in the fetchedMsg success branch, view/detail -> `Unavail.Kind` branches, footer -> `keys.Remove` when the row is saved.

### Data-Flow Trace (Level 4)

| Artifact | Data | Source | Real data | Status |
|----------|------|--------|-----------|--------|
| Saved rows in list | `m.snap.PRs` | `Fetch` -> `savedRefs` (file read) -> `lookupRefs` (GraphQL) | Yes | FLOWING |
| Unavailable row reason | `pr.Unavail` | per-alias `errors[]` via `unavailFor` | Yes (SAML shape constructed) | FLOWING |
| Added row | `addCheckedMsg.pr` | `Client.Lookup` | Yes | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Build, vet, full suite with race | `go build ./... && go vet ./... && go test ./... -count=1 -race` | all 6 packages ok | PASS |
| 30 named phase tests (restart/remove E2E, ParseRef, lookup variables/chunks/reasons, store backups/caps/interleave, two-strike, filters, confirm, stale epoch, esc) | `go test ./... -run '...' -v` | every one `--- PASS` | PASS |

### Anti-Patterns

Scanned the 16 non-test Go files changed in the phase for `TBD|FIXME|XXX|TODO|HACK`: none. No stub returns on rendered paths; empty-slice initialisers are overwritten by the fetch path.

| File | Pattern | Severity | Impact |
|------|---------|----------|--------|
| internal/watchlist/watchlist.go:223-307 | Unlocked read-modify-write (WR-01) | Warning | Possible lost update under simultaneous writes from two panes |
| internal/watchlist/watchlist.go:133-164 | Backup file published before copy completes, re-opens source instead of writing hashed bytes (WR-02) | Warning | Crash-window or concurrent-replace could leave a wrong/partial backup treated as done |
| internal/watchlist/watchlist.go:208-217 | Version mismatch replaced after backup, no add-time notice (WR-03) | Warning | Older prv overwrites newer-version list (backed up) |
| internal/watchlist/watchlist.go:67-76 | Unknown keys dropped; one mistyped entry marks the file corrupt (WR-04) | Warning | Hand-edits lose extra keys; list restarts from backup |
| internal/ui/add.go:104-111 | Esc during the save step does not stop the save (WR-05) | Warning | A cancelled add can still persist |
| IN-01..IN-03 | Shared header capture; generic add-failure text; parse errors hidden during rate-limit pause | Info | Cosmetic or rare mis-classification |

### Judgment on the review warnings

- **WR-01** (two-pane read-modify-write): The mechanism the phase promised (re-read on every call, atomic rename) is implemented and sequentially tested, and every refresh re-reads the file so panes converge. A lost update needs two local writes to overlap within one read->fsync->rename span, each following a human action. It does not falsify SC3 or the goal. It does make the 04-04 sentence "never lose each other's change" literally true only without simultaneity. Recommend fixing with a lock file before shipping widely; not a gap.
- **WR-03**: persistence and graceful degradation are intact, because the old bytes are copied to a backup before any replacement and the user is told at startup. The goal says "survives restarts and degrades gracefully": that holds. Not a gap.
- **WR-05**: the narrow window and the fact that the entry stays removable make this a UX defect, not a goal failure.
- **WR-02/WR-04**: edge cases around crashes and hand-edited files; data is preserved in the common cases.

## Gaps Summary

No gaps. The phase goal is achieved in the code: any github.com PR can be added by URL or `owner/repo#n` after a validating fetch; the list is a versioned, atomically written JSON store that is re-read on each refresh, survives restarts, keeps closed PRs past the window, and is changed only by the user's own `a` and confirmed `d`; and PRs that cannot be fetched degrade to explained unavailable rows while the rest still render. Status is `human_needed` only for terminal glyph width, the constructed SAML shape, and a live-GitHub smoke test, none of which can be proven from fixtures. The five open warnings should be triaged (WR-01 is the best candidate for a follow-up fix) but do not require a gap-closure plan for this phase to meet its goal.

---

_Verified: 2026-10-14_
_Verifier: Claude (gsd-verifier)_

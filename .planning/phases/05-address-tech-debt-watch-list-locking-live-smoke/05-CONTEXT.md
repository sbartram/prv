# Phase 5: Address tech debt: watch-list locking + live smoke - Context

**Gathered:** 2026-10-13
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase closes v1 tech debt that the milestone audit (`.planning/v1-MILESTONE-AUDIT.md`) recorded. It has two parts:

1. **Watch-list robustness.** Make the watch-list store safe when two prv instances write it at the same time (Phase 4 WR-01, integration G3). Fix the related store and add-flow defects WR-02, WR-03 and WR-05, and the error-message gap IN-02.
2. **Live smoke.** Check prv against real github.com in two ways. An opt-in automated live test asserts the state, review, CI and MERGE mapping for pinned public PRs. A manual UAT checklist covers the visual checks that Phases 1 and 2 deferred.

No new user-facing features. ROADMAP.md still shows the goal as "[To be planned]". The planner should replace it with the boundary above.

</domain>

<decisions>
## Implementation Decisions

### Watch-list fix scope
- **D-01:** In scope: **WR-01** (locking), **WR-02** (backup crash-safety), **WR-03** (no silent clobber), **WR-05** (esc during save) and **IN-02** (specific save errors). All come from `.planning/phases/04-curated-watch-list/04-REVIEW.md`. Out of scope: WR-04 (unknown JSON keys and per-entry tolerant decode) and G4 (add path drops partial-data warnings). See Deferred.
- **D-02 (WR-01):** Add and Remove hold an exclusive advisory lock for the whole read-modify-write cycle (`readForWrite` → modify → `write`).
- **D-03 (WR-02):** Write the backup to a temp file in the same directory and `Sync` it. Then publish it under the final `watchlist.json.corrupt-HASH8` name with a no-clobber `os.Link`, so a partial backup can never appear under the final name. Copy the bytes that were hashed (`data`), not a re-read of `s.path`. For files over 1 MiB, hash and copy through one open handle.
- **D-04 (WR-03):** For a **version mismatch** (`version != 1`), Add and Remove **refuse to write** and return an error such as `watchlist.json is version N; this prv writes version 1`. Only an **unparseable or oversized** file may start over. When it does, Add and Remove report it to the UI, and the UI flashes the backup path, e.g. `previous list saved to watchlist.json.corrupt-XXXXXXXX`. The user must not lose saved PRs with only an 8-second startup warning as the explanation. — **Reversibility:** reversible
  - **User confirmation (2026-10-13, plan revision, answer "Refuse"):** a watchlist.json that decodes as a JSON object whose `version` is not 1, including `"version": 0` and no `version` key (e.g. `{}`), is a version mismatch, so Add and Remove refuse to write; only non-JSON, non-object JSON (`[]`, `"x"`, `3`) or oversized files count as unparseable and may start over.
- **D-05 (WR-05):** Once the save is dispatched, esc no longer cancels. Add a `saving` state that is separate from `checking`. Today `applyAddChecked` reuses `m.add.checking = true` for the save, and `addKey` matches Cancel before it checks `checking`. While saving, Cancel is ignored, and the prompt should not offer `esc cancel`. The UI and the file must agree.
- **D-06 (IN-02):** Add and remove failures show a specific, cleaned short reason instead of `could not save the watch list` / `could not update the watch list`. The full-list case at least gets a sentinel (e.g. `watchlist.ErrFull`) and its own wording. The new WR-03 version-mismatch and lock-timeout errors (D-09) go through this same path. Clean every OS or path text with `domain.Clean`.
- **D-07:** The proof of WR-01 is a **multi-process test**: the test binary re-execs itself as N child processes, each calls `Store.Add` for a distinct ref on the same file, and the test asserts that all N entries survive. A goroutine-only test is not enough. The existing `TestTwoStoresInterleave` stays, but it does not count as proof.

### Lock behaviour & platforms
- **D-08:** **Targets are macOS and Linux only.** Use `flock` directly (`golang.org/x/sys/unix` is already an indirect dependency; promote it, or use `syscall`). No Windows variant and no build-tagged stub. — **Reversibility:** reversible — a Windows port later adds one `_windows.go` lock file.
- **D-09:** **Block with a timeout.** Add and Remove wait for the lock up to a few seconds (exact value is Claude's discretion; a real hold lasts milliseconds). `flock` has no timeout, so poll with `LOCK_EX|LOCK_NB` until a deadline. On timeout, return an error (e.g. `watch list is busy (another prv is writing it)`), and show it through the D-06 path. This already runs in a `tea.Cmd`, never in Update.
- **D-10:** **Reads stay lock-free.** `Load` (every refresh tick) and the read inside `read()` take no lock. The atomic temp+rename write means a reader always sees a whole file.
- **D-11:** The lock file is the sibling **`watchlist.json.lock`**, mode 0600, created on demand in the same directory (`MkdirAll` 0700 as today). It is **never deleted**, because deleting a lock file races with other lockers.

### Live smoke form
- **D-12:** **Both forms.** (a) An automated live test that is **opt-in only**: it skips unless an env var such as `PRV_LIVE=1` is set. `go test ./...` must stay offline and green without it. (b) A manual UAT checklist (`05-UAT.md`, run through `/gsd-verify-work`).
- **D-13:** The automated test targets **pinned, stable public PRs** in large public repos (long-merged or closed, so their state does not drift; e.g. `cli/cli`, `prometheus/prometheus#19994`). It is read-only and asserts exact values only on fields that cannot change after merge or close. The research must choose concrete PRs and say which fields are stable for them. Open-PR-only states such as CONFLICTING mergeability or BEHIND drift, so either avoid exact assertions on them or note them as unverifiable with stable fixtures.
- **D-14:** **Local only. No CI.** There is no `.github/workflows` today, and this phase does not add one. Document how to run the live test (env var, token from `gh`) where a developer will find it (test file doc comment and/or README).

### Which live checks
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

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Tech debt source
- `.planning/v1-MILESTONE-AUDIT.md` — the audit that created this phase. Integration findings G1–G8, the tech-debt list per phase, and the "Watch-list robustness" and "Live verification debt" assessment.
- `.planning/phases/04-curated-watch-list/04-REVIEW.md` — WR-01, WR-02, WR-03 and WR-05 (Warnings) and IN-02 (Info): the issue text and suggested fixes (the flock sketch, the temp+Link backup).
- `.planning/phases/04-curated-watch-list/04-REVIEW-DISPOSITION.md` — update the dispositions of the findings this phase fixes (`fixed`). Set WR-04 and IN-01/IN-03 to `deferred` with a reason, or leave them.
- `.planning/phases/04-curated-watch-list/04-CONTEXT.md` — D-16 (sync across instances: reload, apply one change, atomic write, re-read every tick) and the carried-forward store rules (corrupt file backed up, never auto-delete).

### Deferred UAT checks
- `.planning/phases/02-live-monitoring/02-UAT.md` — the 7 deferred UI checks. 6 come into the manual checklist (D-16). The live rate-limit pause stays deferred.
- `.planning/phases/01-status-at-a-glance/01-UAT.md` — #2 badges vs github.com is in scope. #1, #4 and #5 stay deferred.
- `.planning/phases/03-blocker-detail/03-UAT.md` — #4 MERGE vs merge box is in scope. #1 and #3 stay deferred.
- `.planning/phases/04-curated-watch-list/04-UAT.md` — #2 SAML stays deferred.

### Project
- `.planning/PROJECT.md`, `.planning/REQUIREMENTS.md` — WATCH-05/06 (the store requirements that WR-01 protects) and RFSH-05.
- `.claude/CLAUDE.md` (project) — stack constraints: go-gh GraphQL client, no `EnableCache`, `LogIgnoreEnv`, XDG paths, atomic JSON write.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/watchlist/watchlist.go`: `Store{path, now, rename}`, `read()`, `backup(data)`, `readForWrite()`, `Add`, `Remove`, `write()` (temp + Sync + rename). The lock wraps the `readForWrite`→`write` span in Add and Remove. `backup` (lines ~133-164) is the WR-02 target. `readForWrite` (~211) is the WR-03 target: today it turns every `ErrCorrupt` into an empty list.
- `CorruptError` / `ErrCorrupt` (watchlist.go:40-44): the existing typed error. Extend it, or add a sibling, to tell version mismatch from unparseable.
- `internal/watchlist/watchlist_test.go` (545 lines): `t.TempDir()` stores, `TestTwoStoresInterleave`, and the injectable `rename` hook.
- `internal/ui/add.go`: `addState`, `addKey` (Cancel is matched before `m.add.checking`), `applyAddChecked` (sets `checking = true` and dispatches `saveCmd`), and `applyAddSaved` (the generic `could not save the watch list`). These are the WR-05 and IN-02 targets. The remove path's `applyRemoved` has the same generic message.
- `internal/ui/refresh.go:95` `setFlash`: the transient footer message used for the D-04 backup notice.
- `domain.Clean`: sanitises any error text before display.
- `internal/github` client and `testdata/*.json` fixtures: the live test calls the real client instead of fixtures. `cmd/prv/main_test.go` already exists for binary-level tests.

### Established Patterns
- No I/O in Update or View. Every file operation runs in a `tea.Cmd` (Phase 4 D-16), so a blocking lock wait never freezes the UI.
- A sequence number (`addSeq`) drops stale results. The WR-05 fix must keep this, and must stop esc from abandoning a save that has already started.
- Stdlib `testing`, no testify. Golden files for the UI. The add-prompt goldens may need an update if the "saving" prompt text changes.

### Integration Points
- `cmd/prv/main.go` opens the store and passes it into `ui.Options`. The signatures stay the same unless the D-04 notice needs a richer return from `Add`/`Remove`.
- The UI's `watchlist` interface (what `saveCmd`/remove call) changes if Add/Remove return a notice.

</code_context>

<specifics>
## Specific Ideas

- Use the flock sketch in 04-REVIEW.md WR-01 as the starting shape: open `path+".lock"` with `O_CREATE|O_RDWR` and mode 0600, then flock. Add the D-09 timeout and keep the lock file (D-11).
- `prometheus/prometheus#19994` was already named in STATE.md as a candidate public PR for live checks.
- Throwaway test PR `sbartram/prv#1` is still open (STATE.md blocker). This phase does not use it (D-13).

</specifics>

<deferred>
## Deferred Ideas

- **WR-04:** keep unknown JSON keys and decode each entry tolerantly, as `[]json.RawMessage`.
- **G4:** the add path drops partial-data warnings.
- **G1 / Phase 3 WR-03 / Phase 4 IN-01:** per-request rate-limit header capture. **G2:** detail and lookup cost missing from "api N". Both were deferred earlier by the user.
- **IN-03:** parse errors are hidden during a rate-limit pause.
- **G8:** `01-REVIEW-DISPOSITION.md` still says WR-01..03 are open, but they were fixed in Phase 2. This is docs only, and can be fixed as a quick task.
- **Phase 2 IN-01..04** (a dead MergedAt payload, a dead Config.Path, the 32-bit `1 << 31` build break, positional args silently ignored), and the README not documenting the config.toml keys.
- **A CI workflow** (`.github/workflows`), including a scheduled live test. No CI exists yet.
- **Windows support** for the lock.
- **Live checks still deferred:** see D-17.

</deferred>

---

*Phase: 05-address-tech-debt-watch-list-locking-live-smoke*
*Context gathered: 2026-10-13*

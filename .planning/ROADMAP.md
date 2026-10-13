# Roadmap: prv

## Overview

prv is built as four vertical slices, and every phase ends with a working `prv` binary that does more than before. Phase 1 delivers the core value in its thinnest form: real auth, one live GraphQL fetch of the user's authored and assigned PRs, and a Bubble Tea v2 list showing state, review status, and CI. The phase opens with the research's live-verification spike so that the query is not frozen on guesses. Phase 2 makes prv a pane you can leave open all day: single-flight auto-refresh, rate-limit awareness, a stable attention sort, filters, and config. Phase 3 answers "why is it blocked?" with reviewers, merge-readiness, failing check names, and a per-PR detail view. Phase 4 widens the list from "my PRs" to "any PR I care about" with add by URL, removal, persistence, and per-PR unavailable rows.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Status at a Glance** - Live spike, then auth + one GraphQL fetch of authored/assigned PRs rendered as a list with state, review, and CI badges (completed 2026-10-11)
- [x] **Phase 2: Live Monitoring** - Single-flight auto-refresh, rate-limit backoff, stable attention sort, state/number filters, config file, open-in-browser, help (completed 2026-10-12)
- [ ] **Phase 3: Blocker Detail** - Reviewer names, merge-readiness, failing check name in rows, and a detail view of every check and reviewer
- [ ] **Phase 4: Curated Watch List** - Add any PR by URL or `owner/repo#n`, remove it, persist across restarts, and degrade to per-PR unavailable rows

## Phase Details

### Phase 1: Status at a Glance

**Goal**: User runs `prv` and, from live GitHub data, sees every PR they authored or are assigned to with its state, review status, and CI badge, without opening a browser
**Mode:** mvp
**Depends on**: Nothing (first phase)
**Requirements**: AUTH-01, AUTH-02, AUTH-03, CONF-02, WATCH-01, WATCH-02, WATCH-03, LIST-01, LIST-02, LIST-03, LIST-05, NAV-01, NAV-04
**Success Criteria** (what must be TRUE):
  1. Running `prv` with a token from `GH_TOKEN`, `GITHUB_TOKEN`, or the `gh` CLI login (in that order) shows the user's PRs from live GitHub data. With no token, prv exits with a clear message that says how to log in. The token never appears on screen, in logs, or in any file.
  2. The list contains every open PR the user authored or is assigned to, plus PRs merged or closed in the last 14 days. A PR that is both authored and assigned appears once. Each row shows repo, PR number, title, author, a draft marker, and Open / Merged / Closed state.
  3. Each row's review status (approved / changes requested / review required / no review status) matches github.com. On a repo without required reviews, an approved PR shows "approved" and an unreviewed PR shows "no review status", never "review required".
  4. Each row shows one CI badge (passing / failing / running / no checks) that matches github.com. A PR with no checks never shows "passing".
  5. User can move through the list with j/k and the arrow keys and quit with `q`, and the terminal is restored. `go install github.com/sbartram/prv/cmd/prv@latest` installs a working binary (after the repo is pushed), and `prv --version` prints its version.

**Early gate**: Run the live-verification spike (research/SUMMARY.md, "Live-Verification Spike", all 10 questions, via `gh api graphql`) before the discovery query, review derivation, and token resolution are frozen. Record the findings in the phase directory. Q1, Q4, Q5, and Q8 decide this phase: discovery shape (C2), the latestOpinionatedReviews fallback (C4), and token precedence. Q2, Q7, Q9, and Q10 feed the Phase 3 contexts decision (C3). Q3 and Q6 feed the Phase 4 per-alias error handling.
**Pitfalls first addressed here**: A null `reviewDecision` is derived from latestOpinionatedReviews and never maps to "review required". A null `statusCheckRollup` maps to "no checks". One total mapping covers CheckRun and StatusContext, with an unknown case, and CANCELLED/NEUTRAL/SKIPPED are never red. Recorded GraphQL fixtures cover these cases plus comment-after-approve. Read the raw GraphQL envelope (partial data plus `errors[]`). Dedupe by node ID. The 14-day closed window is a fixed default in the discovery query until Phase 2 makes it configurable. Strip control characters and ANSI from remote strings. No I/O in Update/View. Do not log the token (`LogIgnoreEnv`). Data is fetched once at launch; auto-refresh arrives in Phase 2.
**Plans**: 4/4 plans complete

Plans:
**Wave 1**
- [x] 01-01-PLAN.md — Walking skeleton: tracer (token, live ISSUE_ADVANCED search, Bubble Tea v2 list, j/k/q, --version, no-token exit) + review/CI badges and remote-string sanitising

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 01-02-PLAN.md — Right PRs once each in D-06 order (paging, dedupe, exact window) + loading/error/empty states with hints, r retry, token hygiene

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 01-03-PLAN.md — Readable rows: ANSI-16 badge colours, dimmed drafts/merged/closed, explicit widths (TITLE first), scrolling, golden snapshots

**Wave 4** *(blocked on Wave 3 completion)*
- [x] 01-04-PLAN.md — README + automated release gate (local go install, --version, hygiene) + user-gated go install @latest checkpoint

**UI hint**: yes

### Phase 2: Live Monitoring

**Goal**: User can leave prv open in a terminal pane all day and trust that it is current, stays inside rate limits, and lets them narrow the list and jump to a PR on github.com
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: RFSH-01, RFSH-02, RFSH-03, RFSH-04, RFSH-05, RFSH-06, CONF-01, LIST-08, FILT-01, FILT-02, FILT-03, NAV-02, NAV-03
**Success Criteria** (what must be TRUE):
  1. Left open, the list updates itself on the interval set in `~/.config/prv/config.toml` (default 60s; values below 15s are raised to 15s). Pressing `r` refreshes immediately. The screen shows the time of the last successful update. prv runs with defaults when the config file is missing.
  2. When a refresh fails, the last data stays on screen with an error indicator. The status bar shows the remaining API rate limit. When the limit is low or used up, prv backs off on its own and shows when it will resume. Pressing `r` repeatedly never starts overlapping refreshes.
  3. The list is sorted by need for attention (failing CI and changes requested first). The order does not shuffle between refreshes, and the selected PR stays selected after a refresh even when it moves.
  4. User can toggle Open / Merged / Closed visibility in any combination and type a PR number to narrow the list. Auto-found merged and closed PRs appear only if they closed within the configured window (default 14 days).
  5. User can press `o` to open the selected PR in the browser and `?` to see the key bindings screen.

**Pitfalls first addressed here**: Single-flight refresh with one tick chain that re-arms after each result. A generation counter drops stale ticks and results, and a manual `r` coalesces with an in-flight refresh. Selection is tracked by PR ref (owner/repo#n), not by row index, with a nearest-neighbour fallback. The sort is stable on a key that polling does not change. Every query includes `rateLimit { cost remaining resetAt }`. Backoff honours retry-after, then resetAt, then a bounded exponential. Cancel the context on quit. Wrap Cmds so that a panic is caught. Logs go to a file only. Open only `https://github.com` URLs, through go-gh `pkg/browser` in a `tea.Cmd`. Verify with a counting fake client over a simulated hour with `-race`.
**Plans**: 10/10 plans complete

Plans:
**Wave 1**
- [x] 02-01-PLAN.md — Tracer: auto-refresh on the config.toml refresh_interval through one generation-stamped tick chain (config → cmd → ui → github client); a failed refresh keeps the rows, a failed first fetch keeps the panel and retries (D-07, D-09)

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 02-02-PLAN.md — r refreshes from any screen, single-flight with a keymap; header "updated HH:MM:SS" with spinner; selection follows the PR by ref with a nearest-neighbour fallback (D-03, D-05)
- [x] 02-03-PLAN.md — Attention sort: Bucket (D-01, D-04 as clarified: merged/closed drafts by lifecycle) and createdAt-then-ref tie-break (D-02)

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 02-04-PLAN.md — Header/footer truth on any width: D-07 failure marker with short reason, D-09 "retrying at", priority composer, notes before keymap hints (WR-03), honest empty screen (WR-02)

**Wave 4** *(blocked on Wave 3 completion)*
- [x] 02-05-PLAN.md — Rate limit visible: limit in the query, faint "api N" footer, yellow <20% header warning (D-06)

**Wave 5** *(blocked on Wave 4 completion)*
- [x] 02-06-PLAN.md — Automatic backoff: pure NextDelay (retry-after, resetAt, bounded exponential, reserve, pacing), pause with r refusal (D-08, D-09), simulated-hour test

**Wave 6** *(blocked on Wave 5 completion)*
- [x] 02-07-PLAN.md — Real rate-limit responses: header-capturing transport, KindRateLimited for HTTP-200 primary and 403/429 secondary limits, later-page rule, end-to-end pause test

**Wave 7** *(blocked on Wave 6 completion)*
- [x] 02-08-PLAN.md — Open/Merged/Closed toggles (1/2/3) with header indicator and hidden count, PR-number prompt (/), filtered-empty message (D-10..D-13)

**Wave 8** *(blocked on Wave 7 completion)*
- [x] 02-09-PLAN.md — config.toml complete (closed_days, show, line-numbered errors, --closed-days precedence in 1..365), unknown-key warnings with a bounded 8 s lifetime that never mask other messages (D-14..D-16)

**Wave 9** *(blocked on Wave 8 completion)*
- [x] 02-10-PLAN.md — ? help overlay from the keymap with badges, settings, API budget and config warnings (D-17); o opens the selected PR on github.com only

**UI hint**: yes

### Phase 3: Blocker Detail

**Goal**: User can see, without leaving the terminal, who and what is holding each PR up: which reviewers, which failing check, and whether it can merge
**Mode:** mvp
**Depends on**: Phase 2
**Requirements**: LIST-04, LIST-06, LIST-07, DETL-01, DETL-02, DETL-03, NAV-05
**Success Criteria** (what must be TRUE):
  1. Each row shows which reviewers approved, which requested changes, and which are still pending, plus merge-readiness: mergeable, blocked, behind, or conflicts.
  2. The row for a PR with failing CI names one of its failing checks. (v1 limit accepted by the user on 2026-10-12, CONTEXT D-02: a PR whose checks fill more than one page of 100 contexts shows the plain failing badge; see LIST-07 in REQUIREMENTS.md.)
  3. Selecting a PR opens a detail view that lists every check/job with its status, failing first, and every reviewer with their review state. Returning to the list keeps the same PR selected, also when a refresh happened while the detail view was open.
  4. In a narrow pane, and after a terminal resize, the list and the detail view drop or truncate columns and stay readable and navigable.

**Gate**: Before planning, decide C3 (check contexts fetched eagerly for every PR, or lazily for the selected PR). Use the Phase 1 spike measurements (Q2, Q7, Q9, Q10). The decision must also cover LIST-07, which needs a failing check name for every failing row and not only for the selected PR.
**Pitfalls first addressed here**: Review-state edge cases: DISMISSED, a COMMENTED review that hides an approval, team and bot reviewers through union fragments with nil guards, and re-requests. The list badge always comes from the rollup state and counts, never from a truncated contexts page. Large matrices show "+N more". Re-run jobs are not double-counted. The detail view handles a PR that leaves the list during a refresh. Use display-width helpers. Add golden tests at 60x15, 80x24, 120x40, and 0x0.
**Plans**: 8/10 plans executed

Plans:
**Wave 1**
- [x] 03-01-PLAN.md — Tracer: a failing PR's row names its failing check (✗ lint +2) through the D-01 two-step fetch; every check kind classified, cancelled-only marker, '+N' counts only what fails, never a guessed name (D-01..D-04)

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 03-02-PLAN.md — Failing names on messy real data: re-runs, truncated pages, partial errors, every failing PR looked up in chunks of 25, rate limits; long and CJK names shortened, never the suffix (D-01..D-03)
- [x] 03-03-PLAN.md — Narrow panes for the list: pure layoutColumns with the D-13 drop order, compact rows below 40 cells, resize and never-overflow property test, 60x15 goldens (D-13, D-14)
- [x] 03-04-PLAN.md — Merge readiness and reviewers reach every PR through the github seam: DeriveMerge (state and draft first, UNKNOWN+CONFLICTING) and MergeReviewers (pending wins, hidden teams, bots) (D-06, D-07)

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 03-05-PLAN.md — MERGE and REVIEWERS columns in every row (+N, me, hidden teams), in the D-13 drop order and the compact glyphs, with help legends (D-05..D-07, D-13)
- [x] 03-06-PLAN.md — Detail data: one paged Detail query per PR with de-duplicated sorted checks, every reviewer incl. commented and dismissed, counts de-duplicated at every size with a partial read reported (D-01, D-03, D-08, D-10)

**Wave 4** *(blocked on Wave 3 completion)*
- [x] 03-07-PLAN.md — enter opens a full-screen detail: every check failing first, every reviewer, fold with e, honest summary ("read N of M runs" on a partial read); esc returns to the same PR (D-08..D-10)

**Wave 5** *(blocked on Wave 4 completion)*
- [x] 03-08-PLAN.md — The open detail refreshes with the list, drops stale results, respects pauses, says "no longer in list" when its PR leaves; fits any pane and every detail state has goldens (D-11, D-12, D-14)

**Wave 6** *(gap closure, UAT 2026-10-12: review warnings WR-01, WR-02, WR-04; WR-03 deferred)*
- [ ] 03-09-PLAN.md — FetchDetail marks a partial read (null nodes or a kept GraphQL error) so the detail says "read N of M runs" (G-03-6-WR02)
- [ ] 03-10-PLAN.md — Detail rate-limit pause clamped through domain.ServerWait (1 s to 65 min); unknown checks listed and counted as "unknown", fold line "more passing or skipped" (G-03-6-WR01, G-03-6-WR04)

**UI hint**: yes

### Phase 4: Curated Watch List

**Goal**: User can watch any PR on github.com, not only their own, and their curated list survives restarts and degrades gracefully when a PR can't be fetched
**Mode:** mvp
**Depends on**: Phase 2 (ordered after Phase 3; no hard dependency on it)
**Requirements**: WATCH-04, WATCH-05, WATCH-06, WATCH-07, LIST-09
**Success Criteria** (what must be TRUE):
  1. User can paste a github.com PR URL or `owner/repo#123` into an add prompt, and the PR appears on the list with full status. Pasting a PR that is already listed, whether auto-found or manually added, does not create a second row.
  2. Pasting a malformed reference, a PR that does not exist, or a PR the token cannot access shows an inline error that explains why, and nothing is saved.
  3. Manually added PRs reappear after the user quits and restarts prv. They stay listed after they merge or close, even past the closed window, until the user removes them. Only manually added PRs can be removed.
  4. When a saved PR later can't be fetched (deleted, access revoked, SSO not authorized), it shows as an "unavailable" row with the reason. The other PRs still display, and the saved entry is never deleted automatically.

**Pitfalls first addressed here**: Batched aliased lookups use GraphQL variables, never string interpolation, in chunks of 25-50 aliases. Parse the response envelope yourself and attribute `errors[].path[0]` to its alias, using the Phase 1 spike findings (Q3, Q6) for the NOT_FOUND and SAML FORBIDDEN shapes. Validate a PR by fetching it before you persist it. Support bracketed paste. The store holds PR refs only, as versioned JSON at `$XDG_DATA_HOME/prv/watchlist.json` with an atomic write, a tolerant load, and a backup of a corrupt file. Never remove manual entries automatically. Show an SSO hint when a PR is inaccessible.
**Plans**: TBD
**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Status at a Glance | 4/4 | Complete    | 2026-10-11 |
| 2. Live Monitoring | 10/10 | Complete    | 2026-10-12 |
| 3. Blocker Detail | 8/8 | In Progress | - |
| 4. Curated Watch List | 0/TBD | Not started | - |

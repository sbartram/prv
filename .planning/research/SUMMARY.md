# Project Research Summary

**Project:** prv
**Domain:** Read-only terminal UI (Go + Bubble Tea) that monitors a curated set of GitHub PRs (github.com only)
**Researched:** 2026-10-11
**Confidence:** MEDIUM-HIGH overall

## Executive Summary

prv is a watch-only TUI for GitHub PRs, in the niche of gh-dash and gh pr status but narrower: a persistent curated watch list (authored + assigned auto-included, any other PR added by URL) answering "which of my PRs are approved, blocked, or failing CI?" Experts build it as a layered Elm-architecture app: pure domain model in the middle, a GitHub client that maps wire JSON into it, a file store holding only PR refs, and a UI that never touches the network or wire types.

Recommended approach: Bubble Tea v2 (charm.land/*/v2), go-gh v2 for GraphQL, auth and browser, and one batched GraphQL request per refresh returning state, review decision, reviewers and CI rollup. Detail views render from the same snapshot. Refresh is single-flight, re-armed after each result, with a generation counter dropping stale results.

Main risks are in the data layer: (1) null reviewDecision and null statusCheckRollup are normal on unprotected repos and must not render as "review required" or "passing"; (2) GraphQL partial errors must degrade one row, never blank the list or auto-delete a manual entry; (3) search discovery has lag, a 1,000-result cap and silent SSO omissions; (4) the shared 5,000 points/hour budget needs a poll floor, rateLimit reporting and backoff. Mitigation: test domain mapping and the GraphQL layer with recorded fixtures, and run the live spike below before freezing the query.

## Key Findings

### Recommended Stack

- Bubble Tea v2.1.0, Bubbles v2.2.1, Lip Gloss v2.0.6 (charm.land/*/v2). Do not start on v1: View() returns tea.View, KeyPressMsg, v.AltScreen = true, no WithAltScreen. Upgrade the three together.
- go-gh v2.16.2: api.NewGraphQLClient with raw query strings and DoWithContext; auth.TokenForHost; pkg/browser (honours GH_BROWSER/BROWSER).
- Config: BurntSushi/toml v1.6.0 at $XDG_CONFIG_HOME/prv/config.toml (default ~/.config on macOS too). Watch list: versioned JSON at $XDG_DATA_HOME/prv/watchlist.json, atomic write.
- Release: GoReleaser v2.18.3, version: 2, homebrew_casks (not brews), CGO_ENABLED=0. Late phase.
- Constraints: never persist the token; no EnableCache for polling; LogIgnoreEnv: true; no stdout/stderr writes while TUI runs.

### Expected Features

- **Must have (v1):** auto-include authored and assigned, deduped, with provenance; add by URL or owner/repo#n with validation; persist and remove manual entries; list with repo, number, title, author, draft marker, state, review badge, one CI badge (passing/failing/running/none); reviewer names with approve/changes-requested; detail view with all checks (CheckRun and StatusContext), failing first, with link; state filter (any combination) and recent window for Merged/Closed; auto-refresh on configurable interval, manual r, last-updated indicator, keep data on error, visible rate-limit state, selection preserved by PR identity; open in browser (o), ? help, quit, resize; clear auth-failure message; config with defaults.
- **Should have (v1.x):** merge-readiness (mergeStateStatus), needs-attention sort, change highlight, text filter, failing check names inline, adaptive refresh, copy URL, startup arg owner/repo#n, rate remaining in footer.
- **Defer (v2+):** unresolved thread count, bell, grouping, --json, GHE, themes/mouse.
- **Anti-features:** write actions, diff viewer, CI logs, review-requested section, notifications, issues, arbitrary search sections, full history, webhooks, custom shell keybindings, OAuth login, checkout integration.

### Architecture Approach

Layers point down only. internal/domain is a leaf (PR, PRRef, ParseRef, rollup classification, sort/filter, Result, RateInfo). internal/github owns auth, query text, unexported wire structs and the single wire-to-domain mapper. internal/store persists PRRef values only, atomic write, tolerant load. internal/ui defines its own Fetcher and WatchStore interfaces and never imports github or store. cmd/prv wires everything. Config: interval floor 15s (clamped with warning), closed window default 14 days.

Patterns: re-arm tick after fetch completes, generation counter for stale ticks and results; cursor by owner/repo#n not index; stable sort (state group, repo, number), never by a poll-mutated field; errors keep last good snapshot; detail costs zero extra requests if contexts are in the snapshot (see C3).

### Critical Pitfalls

1. **Null reviewDecision treated as a state.** Fallback: reviewDecision if set, else derive from latestOpinionatedReviews (changes-requested, then approved, then none). Never map null to review required. Fixtures: null+approved, null+none, null+changes-requested.
2. **Review-state mistakes.** Use latestOpinionatedReviews, not reviews(first:N). Handle DISMISSED, COMMENTED hiding an approval, team/bot reviewers (union fragments, nil guards), re-requests.
3. **CI rollup.** Null rollup means no checks, not passing. Map CheckRun (status+conclusion) and StatusContext (state) with one total function and an unknown case. CANCELLED and NEUTRAL/SKIPPED must not be red. Badge from rollup state and counts, never from a truncated page.
4. **Batched GraphQL all-or-nothing.** Parse the envelope yourself, attribute errors[].path[0] to alias, render an unavailable row, keep last good data, never auto-remove manual entries. Use GraphQL variables. Chunk 25-50 aliases.
5. **Rate limits and overlapping refreshes.** Include rateLimit { cost remaining resetAt } in every query. Single-flight. Interval floor 15-30s. Backoff honouring retry-after, then reset, then bounded exponential. Show remaining and rate-limited-until in status bar.
6. **Search as state source.** Use search for discovery only where needed; always is:pr archived:false; compare issueCount with returned nodes for truncation banner; vanishing-PR grace; SSO hint when expected PRs are missing.
7. **Bubble Tea blocking and tick chains.** No I/O in Update/View. One tick chain. Manual r bumps generation and coalesces. Context cancelled on quit. Panic-safe Cmd wrapper. Logs to file only.
8. **Selection and sort.** Track selected PRRef; re-locate after refresh; nearest neighbour fallback. Do not rebuild list model. Filter lives in app model.
9. **Token handling.** Never log or persist token or put it in errors. Re-resolve once on 401. Document read-only fine-grained PAT permissions.
10. **Terminal safety.** Width/height 0 = loading. Display-width helpers. Strip control chars and ANSI from remote strings. Open only https://github.com URLs.

## Implications for Roadmap

### Phase 1: Domain model, auth, fixtures

**Rationale:** everything depends on domain types and mapping rules; auth gates live calls.
**Delivers:** domain (PRRef, ParseRef table tests, rollup classification, review fallback, sort/filter), auth with fail-fast message, cmd/prv skeleton with --version, recorded fixtures (null rollup, null reviewDecision, NOT_FOUND alias, truncated contexts, StatusContext-only, merged, dismissed, team reviewer).
**Avoids:** pitfalls 1, 2, 3, 11, 12, 13. Uses go-gh auth, regexp, net/url.

### Phase 2: GitHub data layer and store (gate: live spike)

**Rationale:** query shape and error model define the UI's real inputs.
**Delivers:** GraphQL client (raw envelope, per-alias errors, rateLimit capture, timeout), query builder (discovery plus aliased manual lookups via variables), wire-to-domain mapper, prv --dump, store (atomic JSON, version, corrupt-file backup), config (TOML defaults, interval floor, closed window). Addresses auto-include and dedupe, manual persistence, window at query time, per-item error rows.
**Avoids:** pitfalls 4, 5, 6, 12, 14, 15.

### Phase 3: TUI shell on fake data

**Rationale:** can start in parallel with end of Phase 2 via Fetcher interface.
**Delivers:** root model, list view (own cursor and scroll; lipgloss table with StyleFunc or custom rows), glyph+colour styles with NO_COLOR, key map and help, state filter chips, responsive column dropping, golden tests at 60x15, 80x24, 120x40, 0x0.
**Avoids:** 8, 9, 10, 16.

### Phase 4: Live refresh loop and status

**Rationale:** first user-visible milestone ("approved, blocked, failing at a glance").
**Delivers:** fetchCmd/tickCmd with generation counter and single-flight; manual refresh coalesced with min spacing; last-good snapshot with stale marker; status bar (updated time, spinner, rate remaining, backoff countdown); 401 re-resolve once; sleep-gap coalescing. Verify with counting fake client over simulated hour and -race.
**Avoids:** 5, 7, 8, 17.

### Phase 5: Detail view (gate: decide lazy vs eager contexts, C3)

**Delivers:** full-screen viewport with header, reviewer list with states, checks grouped failing/running/passing with skipped dimmed, workflow + job name, "+N more", check link, o to open via go-gh browser in a tea.Cmd, back navigation handling PR leaving list.
**Avoids:** 2, 3, 18.

### Phase 6: Watch management (gate: decide manual-vs-window, C5)

**Delivers:** add prompt (textinput, bracketed paste, validate by fetching before persist, inline NOT_FOUND/SSO error), remove for manual only, closed window for auto entries, "N hidden (closed > 14d)" count.
**Avoids:** 12, 14, 8.

### Phase 7: Polish and release

**Delivers:** narrow-terminal pass, --version, GoReleaser (version: 2, tar.gz, optional homebrew_casks), golangci-lint, CI snapshot build, 2+ hour soak test with mock counting points and injecting 403 retry-after, README with token scopes and precedence.

### Phase Ordering Rationale

Domain and auth first (pure, testable); data layer before UI (live verification where it matters); UI on fakes in parallel; live refresh before detail and add/remove (earliest core value, establishes single-flight pattern); release last.

### Research Flags

- **Needs spike or deeper research:** Phase 2 (discovery shape, partial-error shape, COMMENTED behaviour, SSO search behaviour, author:@me in search with open PRs, nodes(ids) limits); Phase 5 (context page cost for large matrices); Phase 4 (actual token precedence with GITHUB_TOKEN set).
- **Standard patterns, skip research-phase:** Phase 3 (Bubble Tea v2 list and layout), Phase 6 (textinput and file list), Phase 7 (GoReleaser v2, golangci-lint).

## Conflicts and Open Decisions

- **C1. Token precedence.** STACK and PITFALLS: auth.TokenForHost (env GH_TOKEN, GITHUB_TOKEN, then hosts.yml, then gh auth token), matching gh. ARCHITECTURE: implement PROJECT.md's gh-first order explicitly (gh auth token -h github.com, fall back to GITHUB_TOKEN). Default: follow the spec as written and document it; user to confirm (or update PROJECT.md if they prefer gh's convention).
- **C2. Discovery query.** ARCHITECTURE (live-verified): one ISSUE_ADVANCED search, (author:@me OR assignee:@me) (is:open OR closed:>=DATE); legacy ISSUE returned 0 silently. STACK: two searches deduped client-side, OR-combining untested. PITFALLS: viewer.pullRequests for authored plus search for assigned, discovery/hydration split. Default: ARCHITECTURE's single ISSUE_ADVANCED with four-alias fallback; in Phase 2, switch authored to viewer.pullRequests if authored results miss PRs; always dedupe by node ID.
- **C3. Check contexts eager vs lazy.** ARCHITECTURE: contexts(first:100) for every PR in the snapshot. STACK, PITFALLS, FEATURES: lazy for selected PR, list uses rollup state and counts. Default: lazy for selected PR; badge from state and counts; keep eager only if measured cost at ~25 PRs is cheap. Decide before Phase 5.
- **C4. Review fallback.** PITFALLS: latestOpinionatedReviews (ignores informational COMMENTED, to verify). FEATURES: latestReviews (a later comment can hide an approval). Default: latestOpinionatedReviews; verify in spike.
- **C5. Manual PRs vs closed window (user decision).** ARCHITECTURE and PITFALLS: manual PRs always shown until removed. PROJECT.md does not exempt them. Default: exempt; user must confirm before Phase 6.
- **Minor (defaults):** browser via go-gh pkg/browser in a tea.Cmd (STACK) rather than exec open (PITFALLS/ARCHITECTURE); head oid via commits(last:1) if change detection needs it, else statusCheckRollup shortcut; badge from rollup state with PITFALLS precedence for detail summary; non-required failures shown red with a marker.

## Live-Verification Spike (before freezing Phase 2)

1. Does latestOpinionatedReviews ignore a later COMMENTED review, keeping an earlier APPROVED? Compare with latestReviews.
2. After re-run of failed jobs, does rollup contexts show only the latest run or both?
3. How does the chosen client surface partial data with errors[]? Record exact type and extensions for NOT_FOUND and SAML FORBIDDEN.
4. Does GraphQL search silently omit PRs from SSO-unauthorized orgs or error? Does author:@me work in GraphQL search? Re-check with an account that has open PRs.
5. What does gh auth token return when GITHUB_TOKEN is set, on the installed gh version and with multiple hosts?
6. Is the nodes(ids:) maximum 100, and how do deleted PRs appear?
7. Real rateLimit.cost of the list query for about 25 PRs with 50-context pages.
8. Does ISSUE_ADVANCED return open PRs with (is:open OR closed:>=DATE) for an account with open PRs?
9. Does a 120-job matrix PR truncate contexts, and do totalCount and count fields match?
10. Is isRequired worth its query weight for v1?

## Confidence Assessment

| Area | Confidence | Notes |
|---|---|---|
| Stack | HIGH | Versions checked against module proxy and upgrade guides on 2026-10-11; go-gh behaviour read from source. teatest/v2 is MEDIUM (untagged). |
| Features | MEDIUM | Competitor features from READMEs and docs; gh-dash details from memory (MEDIUM-LOW). |
| Architecture | HIGH | GraphQL shapes and ISSUE_ADVANCED behaviour executed live; Bubble Tea v2 from upgrade guide. Disagreements recorded (C2, C3). |
| Pitfalls | MEDIUM-HIGH | Enums and rate-limit figures from official docs; many behaviours tagged [K] need the spike. |

**Overall confidence:** MEDIUM-HIGH

### Gaps to Address

- Discovery shape (C2): resolve in spike, keep behind an interface.
- Token precedence (C1): user decision, Phase 1.
- Manual vs window (C5): user decision, before Phase 6.
- Contexts lazy vs eager (C3): before Phase 5.
- Review derivation COMMENTED behaviour (C4): include comment-after-approve fixture.
- SSO search and partial-error shape: surface SSO hint and unavailable row so UI degrades.
- teatest/v2: optional; primary testing is Update/View tests.
- Rate budget on user's machine: tune in soak test; default 60s, floor 15s.

## Sources

### Primary (HIGH confidence)

- proxy.golang.org module metadata (bubbletea/v2 v2.1.0, bubbles/v2 v2.2.1, lipgloss/v2 v2.0.6, go-gh/v2 v2.16.2, BurntSushi/toml v1.6.0, x/exp/golden v0.1.0)
- Bubble Tea, Bubbles, Lip Gloss v2 upgrade guides
- go-gh v2.16.2 source (pkg/auth, pkg/api, pkg/browser)
- GitHub GraphQL rate-limit docs and public schema
- Live gh api graphql calls on 2026-10-11
- goreleaser deprecations page; kiroshi and ghzinga docs

### Secondary (MEDIUM confidence)

- gh-dash README and deps; ghui/phui README; gh pr status manual; teatest/v2 source at untagged pseudo-version

### Tertiary (LOW confidence, validate)

- Community note on latestReviews lag; [K]-tagged behaviours in PITFALLS.md (SAML error shape, NOT_FOUND masking, search lag and SSO omission, partial-data semantics, gh env-token precedence); re-run check dedupe

---
*Research completed: 2026-10-11*
*Ready for roadmap: once C1, C2, C3, C5 have a user decision or the recorded default is accepted*

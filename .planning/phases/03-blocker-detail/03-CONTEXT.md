# Phase 3: Blocker Detail - Context

**Gathered:** 2026-10-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Without leaving the terminal, the user can see who and what is holding each PR up. Each list row shows the reviewers (approved / changes requested / pending), merge-readiness (mergeable / blocked / behind / conflicts) and, for failing PRs, the name of a failing check. A full-screen detail view for the selected PR lists every check and every reviewer. The list and the detail view stay readable and navigable in narrow panes and after a resize.

Requirements: LIST-04, LIST-06, LIST-07, DETL-01, DETL-02, DETL-03, NAV-05.
Not in this phase: adding PRs by URL, removal, persistence, per-PR unavailable rows (Phase 4). No write actions.

</domain>

<decisions>
## Implementation Decisions

### Check fetching (C3 gate: resolved)
- **D-01:** Use a **two-step fetch**. The discovery query stays lean: the list badge still comes only from `statusCheckRollup.state` and per-state counts (Q9). In each refresh, after discovery, one batched `nodes(ids:)` query fetches check contexts **only for PRs whose rollup is FAILURE/ERROR**. This query feeds LIST-07. When the detail view opens, it fetches the selected PR's full checks and reviewers. The spike showed that every variant costs 1 point (Q7), so the choice rests on payload size and on the recorded default "contexts fetched lazily". `nodes(ids:)` has a maximum of 100 ids (Q6): chunk the ids if more than 100 PRs fail. — **Reversibility:** costly — the fetch pipeline and the refresh flow are built around two queries plus an on-demand detail query.
- **D-02:** A failing row's CI cell shows **one failing check name plus how many more fail**, for example `✗ lint +2`. The "+N" count comes from the **de-duplicated contexts** (D-03), not from `checkRunCountsByState` — research showed the counts include superseded runs (vscode#341132: +21 for one real failure). *(Revised 2026-10-12 at plan-phase.)* When the contexts page is truncated, the row shows no name and no +N; the badge still comes from the rollup. PRs with more than 100 contexts get no extra paging in v1.
- **D-03:** **Re-runs collapse by workflow + check name, and the latest run is kept.** A job that failed and then passed on a re-run is not counted or named as failing. This applies to the row name, the counts derived from contexts, and the detail view. (Spike Q2 was inconclusive; the research must confirm which fields order the runs, e.g. `startedAt`/`completedAt` or the check-suite order.)
- **D-04:** CANCELLED makes GitHub's rollup FAILURE (spike extra finding 3). When a PR is failing only because a check was cancelled, the row names that check with a marker: `✗ deploy (cancelled)`. When choosing the name, FAILURE/ERROR/TIMED_OUT/STARTUP_FAILURE/ACTION_REQUIRED checks are named before cancelled ones. The list badge does not change (still `failing`, per Phase 1 D-14).

### Row layout (LIST-04, LIST-06, LIST-07)
- **D-05:** Add new **REVIEWERS** and **MERGE** columns. The CI cell grows to carry the failing name. Each PR keeps one line. Column order: `PR TITLE AUTHOR STATE REVIEW REVIEWERS MERGE CI`. TITLE is still the flexible column and shrinks first (Phase 1 D-02).
- **D-06:** REVIEWERS shows a glyph and a login for each reviewer, in this order: changes-requested first, then pending, then approved. Example: `✗bob …carol ✓alice`. When the cell is full, the rest becomes `+N`. Logins follow the AUTHOR rule (the viewer shows as `me`). Review state comes from `latestOpinionatedReviews` (spike Q1). Pending reviewers come from `reviewRequests`. A reviewer who has reviewed and was then re-requested shows as pending.
- **D-07:** MERGE maps `mergeStateStatus`: CLEAN / HAS_HOOKS / **UNSTABLE → mergeable** (readiness answers "can I press merge?", and CI failure already shows in the CI column). BLOCKED → blocked, BEHIND → behind, DIRTY → conflicts. **UNKNOWN (still computing) → a neutral `–`**, never "mergeable" — except **UNKNOWN with `mergeable: CONFLICTING` → conflicts** *(added 2026-10-12 at plan-phase)*. `mergeStateStatus` has no DRAFT value, and merged/closed PRs report stale values, so derive merge state from PR state and `isDraft` first. DRAFT → `–` (the row is already dimmed and tagged). Merged and closed rows leave MERGE blank. Every value uses a glyph plus a word and reads correctly without colour (Phase 1 D-01).

### Detail view (DETL-01..03)
- **D-08:** The detail view is **full screen**. It replaces the list, like the `?` overlay, and keeps the header and footer. Top: ref, title, state · review · merge. Then **CHECKS** with a summary line (`2 failing · 1 running · 14 passing`, built from the counts, so it is correct past 100 contexts). Then **REVIEWERS**, each with its review state.
- **D-09:** Keys: **`enter` opens, `esc` goes back, `q` quits** (q means quit everywhere). `o` (open in browser) and `r` (refresh) work in the detail view. `j`/`k` scroll it. The footer hints and the `?` help come from the shared keymap (Phase 2 D-17).
- **D-10:** Check order: failing → cancelled → running → passing → skipped/neutral (faint). **Passing and skipped/neutral checks fold into one `… +N more passing` line, and a key expands it.** Failing, cancelled and running checks are always listed.
- **D-11:** While the detail view is open, its checks and reviewers **update on each refresh tick**. The detail fetch is a separate Cmd, sequenced after each successful refresh and guarded by a detail sequence number and a ref check *(clarified 2026-10-12: a literal "inside the same Cmd" cannot cover `enter` pressed during an in-flight refresh)*. It adds one small query per tick, and only while the view is open. The scroll position and the expand state are kept.
- **D-12:** If a refresh drops the PR from the list while its detail view is open, the view **stays open with the last data** and shows a faint `no longer in list` note. On `esc` the cursor goes to the nearest neighbour (Phase 2 D-03 / 02-02 rule). Returning from the detail view keeps the same PR selected by `Ref` (DETL-03), also after refreshes while the view was open.

### Narrow panes (NAV-05)
- **D-13:** As the width shrinks, TITLE first shrinks to its minimum. Then columns drop in this order: **AUTHOR → STATE → REVIEWERS → MERGE**. Before REVIEWERS drops, it is cut to `+N`. PR, REVIEW and CI stay to the end (core value: approved / blocked / failing). STATE can go early because dimming already marks merged and closed rows.
- **D-14:** **Below about 40 columns, use a compact mode**: each row is the ref plus glyphs only (e.g. `> web#7 ✗⊘✗`), and the header hints to widen the pane. The list stays navigable, and `enter`/`o` still work. The detail view truncates its columns in the same spirit. 0x0 keeps the established convention: it means unsized and renders everything uncut (as `list_0x0.golden`), and it never panics *(revised 2026-10-12 at plan-phase)*. Between about 40 and 50 columns, TITLE drops entirely after MERGE drops. Golden tests at 60x15, 80x24, 120x40 and 0x0 cover the list and the detail view (roadmap).

### Carried forward (do not re-litigate)
- The list CI badge always comes from the rollup `state` and counts, never from a truncated contexts page (roadmap pitfall, spike Q9).
- `isRequired` is dropped for v1 (spike Q10).
- Review state is derived from `latestOpinionatedReviews`, and `reviewDecision` wins when it is set (spike Q1, Phase 1).
- ANSI-16 palette, and every badge uses a glyph plus a word (Phase 1 D-01/D-05). Per-check display: CANCELLED, NEUTRAL and SKIPPED are never red.
- The selected `domain.Ref` is the source of truth, and `rebuild()` derives the rows and the cursor (02-02).
- Single-flight refresh, generation counter, backoff order, Cmds wrapped against panics, no I/O in Update/View (Phase 2).
- Clean all remote strings (check names, logins, team names) at the mapper seam with `domain.Clean` (Phase 1).
- Use display-width helpers (`lipgloss.Width`, `ansi.Truncate`), never byte or rune counts.

### Claude's Discretion
- Exact glyphs for MERGE (e.g. `✓ mergeable`, `⊘ blocked`, `↕ behind`, `✗ conflicts`) and for the reviewer states, within the ANSI-16 and glyph+word rules.
- How bot reviewers render (e.g. the login without `[bot]`). Team reviewers that come back as `requestedReviewer: null` (the common case for a viewer outside the org) count as pending and render as `…team` / `…2 teams` *(decided 2026-10-12)*. Use nil guards on the union fragments (roadmap pitfall: DISMISSED, team and bot reviewers, re-requests).
- What the detail view shows while its first fetch is loading, and how a failed detail fetch looks (keep the last data plus a marker, as in Phase 2 D-07).
- What the row shows if the failing-contexts query fails or comes back truncated: the badge stays `✗ failing` with no name. Never guess a name.
- Which key expands the collapsed passing checks.
- The exact column widths and caps for REVIEWERS / MERGE / CI, and the exact compact-mode threshold (about 40).
- Additions to the help legend for the new glyphs.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Live API evidence
- `.planning/phases/01-status-at-a-glance/01-SPIKE.md` — Q2 (re-runs not de-duplicated), Q6 (`nodes(ids:)` max 100, NOT_FOUND shape), Q7 (cost 1, payload sizes), Q9 (truncation, `checkRunCountsByState`/`statusContextCountsByState`), Q10 (drop `isRequired`), extra findings 3 (CANCELLED → FAILURE), 6 and 7 (opinionated reviews, bot reviewers); the proven discovery query.

### Scope and requirements
- `.planning/ROADMAP.md` §Phase 3 — goal, success criteria, C3 gate, pitfalls list, golden-test sizes.
- `.planning/REQUIREMENTS.md` — LIST-04, LIST-06, LIST-07, DETL-01..03, NAV-05. Technical Defaults ("check detail fetched lazily").
- `.planning/PROJECT.md` — core value and Key Decisions.

### Prior phase decisions
- `.planning/phases/01-status-at-a-glance/01-CONTEXT.md` — badge style, columns, CI mapping (D-14), palette.
- `.planning/phases/02-live-monitoring/02-CONTEXT.md` — attention sort, cursor-by-ref, refresh/backoff, footer and header priorities, help overlay.
- `CLAUDE.md` (project) — stack rules: Bubble Tea v2 APIs, `lipgloss/table`, no go-gh cache, GraphQL only.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/ui/layout.go` `columnWidths` / `truncateCell`: these already measure in cells with `lipgloss.Width` and `ansi.Truncate`. Extend them with the new columns and the D-13 drop order.
- `internal/ui/view.go` `fit(segs, sep, width)` and the segment priorities: the pattern for dropping by priority. Use it for the detail header/footer and maybe for the column drop.
- `internal/ui/help.go` `renderHelp`: the existing full-screen overlay. The detail view follows the same full-screen screen pattern (`showHelp` flag → a view state).
- `internal/ui/refresh.go` `fetch` / `beginFetch` / `arm`: the single-flight generation pattern that the failing-contexts query and the detail query must join.
- `internal/domain/status.go` `DeriveReview` / `ClassifyRollup` / `Badge()`: put the new merge-state and per-check classification next to them, as pure domain functions.
- `internal/github/client.go` `discoveryQuery` / `mapResponse`: the discovery query gains `mergeStateStatus`, `reviewRequests`, `checkRunCountsByState`/`statusContextCountsByState` (or the rollup counts) and the reviewer logins. Use wire structs plus mapping with `domain.Clean`.

### Established Patterns
- Pure domain (`internal/domain`, no I/O), mapping and sanitising at the github seam, and a ui model with no I/O in Update/View.
- Golden tests in `internal/ui/testdata/TestGolden/<screen>_<WxH>.golden` at 0x0, 80x24 and 120x40. Phase 3 adds 60x15 and the detail screens.
- `internal/github/testdata/*.json` fixtures for client tests. Add fixtures for failing contexts, re-runs, teams/bots, DISMISSED and truncated matrices.

### Integration Points
- `domain.PR` gains reviewer, merge-state and failing-check fields. `domain.Snapshot` stays the unit that a refresh hands to the UI.
- The `ui.Fetcher` interface may need a detail-fetch method for the selected PR (D-01, D-11).
- `internal/ui/model.go` `viewState`: add a detail state. `rebuild()` and `nearest()` handle D-12.
- `internal/ui/keys.go` keyMap: add `enter`, `esc` and the expand key, so that ShortHelp/FullHelp stay the single source.

</code_context>

<specifics>
## Specific Ideas

- Row preview the user chose:
  ```
    PR         TITLE          STATE REVIEW    REVIEWERS      MERGE      CI
  > acme/api#1 Fix login race open  ✓ approved ✓alice ✗bob …carol ✓ ready    ✓ passing
    acme/web#7 Bump deps      open  ✗ changes  ✗bob           ⊘ blocked  ✗ lint +2
  ```
  (The preview left out AUTHOR. AUTHOR stays in the full layout and is the first column to drop, per D-13. The merge word is "mergeable" per LIST-06; "ready" in the preview was illustrative.)
- Detail preview the user chose:
  ```
  prv · acme/web#7 · updated 14:32:05
  Bump deps                              open · ✗ changes · ⊘ blocked

  CHECKS  2 failing · 1 running · 14 passing
    ✗ lint             failure    ci.yml
    ✗ test (ubuntu)    failure    ci.yml
    ● e2e              running    e2e.yml
    ✓ build            success    ci.yml
    … +11 more passing

  REVIEWERS
    ✗ bob     changes requested
    … carol   pending
    ✓ alice   approved

  esc back · o open · r refresh · j/k scroll
  ```

</specifics>

<deferred>
## Deferred Ideas

None. The discussion stayed within the phase scope.

</deferred>

---

*Phase: 03-blocker-detail*
*Context gathered: 2026-10-12*

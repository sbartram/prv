# Phase 3: Blocker Detail - Research

**Researched:** 2026-10-12
**Domain:** GitHub GraphQL (reviewers, merge state, check contexts) plus Bubble Tea v2 / Lip Gloss v2 list layout and a full-screen detail view
**Confidence:** HIGH for API shapes (all probed live on 2026-10-12 with `gh api graphql`, read-only); MEDIUM for team-reviewer rendering (the `Team` branch could not be observed) and for layout thresholds (need goldens)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

### Check fetching (C3 gate: resolved)
- **D-01:** Use a **two-step fetch**. The discovery query stays lean: the list badge still comes only from `statusCheckRollup.state` and per-state counts (Q9). In each refresh, after discovery, one batched `nodes(ids:)` query fetches check contexts **only for PRs whose rollup is FAILURE/ERROR**. This query feeds LIST-07. When the detail view opens, it fetches the selected PR's full checks and reviewers. The spike showed that every variant costs 1 point (Q7), so the choice rests on payload size and on the recorded default "contexts fetched lazily". `nodes(ids:)` has a maximum of 100 ids (Q6): chunk the ids if more than 100 PRs fail. — **Reversibility:** costly — the fetch pipeline and the refresh flow are built around two queries plus an on-demand detail query.
- **D-02:** A failing row's CI cell shows **one failing check name plus how many more fail**, for example `✗ lint +2`. The "+N" count comes from `checkRunCountsByState` / `statusContextCountsByState`, so it stays correct when the contexts page is truncated.
- **D-03:** **Re-runs collapse by workflow + check name, and the latest run is kept.** A job that failed and then passed on a re-run is not counted or named as failing. This applies to the row name, the counts derived from contexts, and the detail view. (Spike Q2 was inconclusive; the research must confirm which fields order the runs, e.g. `startedAt`/`completedAt` or the check-suite order.)
- **D-04:** CANCELLED makes GitHub's rollup FAILURE (spike extra finding 3). When a PR is failing only because a check was cancelled, the row names that check with a marker: `✗ deploy (cancelled)`. When choosing the name, FAILURE/ERROR/TIMED_OUT/STARTUP_FAILURE/ACTION_REQUIRED checks are named before cancelled ones. The list badge does not change (still `failing`, per Phase 1 D-14).

### Row layout (LIST-04, LIST-06, LIST-07)
- **D-05:** Add new **REVIEWERS** and **MERGE** columns. The CI cell grows to carry the failing name. Each PR keeps one line. Column order: `PR TITLE AUTHOR STATE REVIEW REVIEWERS MERGE CI`. TITLE is still the flexible column and shrinks first (Phase 1 D-02).
- **D-06:** REVIEWERS shows a glyph and a login for each reviewer, in this order: changes-requested first, then pending, then approved. Example: `✗bob …carol ✓alice`. When the cell is full, the rest becomes `+N`. Logins follow the AUTHOR rule (the viewer shows as `me`). Review state comes from `latestOpinionatedReviews` (spike Q1). Pending reviewers come from `reviewRequests`. A reviewer who has reviewed and was then re-requested shows as pending.
- **D-07:** MERGE maps `mergeStateStatus`: CLEAN / HAS_HOOKS / **UNSTABLE → mergeable** (readiness answers "can I press merge?", and CI failure already shows in the CI column). BLOCKED → blocked, BEHIND → behind, DIRTY → conflicts. **UNKNOWN (still computing) → a neutral `–`**, never "mergeable". DRAFT → `–` (the row is already dimmed and tagged). Merged and closed rows leave MERGE blank. Every value uses a glyph plus a word and reads correctly without colour (Phase 1 D-01).

### Detail view (DETL-01..03)
- **D-08:** The detail view is **full screen**. It replaces the list, like the `?` overlay, and keeps the header and footer. Top: ref, title, state · review · merge. Then **CHECKS** with a summary line (`2 failing · 1 running · 14 passing`, built from the counts, so it is correct past 100 contexts). Then **REVIEWERS**, each with its review state.
- **D-09:** Keys: **`enter` opens, `esc` goes back, `q` quits** (q means quit everywhere). `o` (open in browser) and `r` (refresh) work in the detail view. `j`/`k` scroll it. The footer hints and the `?` help come from the shared keymap (Phase 2 D-17).
- **D-10:** Check order: failing → cancelled → running → passing → skipped/neutral (faint). **Passing and skipped/neutral checks fold into one `… +N more passing` line, and a key expands it.** Failing, cancelled and running checks are always listed.
- **D-11:** While the detail view is open, its checks and reviewers **update on each refresh tick**. The detail fetch runs inside the existing single-flight refresh. It adds one small query per tick, and only while the view is open. The scroll position and the expand state are kept.
- **D-12:** If a refresh drops the PR from the list while its detail view is open, the view **stays open with the last data** and shows a faint `no longer in list` note. On `esc` the cursor goes to the nearest neighbour (Phase 2 D-03 / 02-02 rule). Returning from the detail view keeps the same PR selected by `Ref` (DETL-03), also after refreshes while the view was open.

### Narrow panes (NAV-05)
- **D-13:** As the width shrinks, TITLE first shrinks to its minimum. Then columns drop in this order: **AUTHOR → STATE → REVIEWERS → MERGE**. Before REVIEWERS drops, it is cut to `+N`. PR, REVIEW and CI stay to the end (core value: approved / blocked / failing). STATE can go early because dimming already marks merged and closed rows.
- **D-14:** **Below about 40 columns, use a compact mode**: each row is the ref plus glyphs only (e.g. `> web#7 ✗⊘✗`), and the header hints to widen the pane. The list stays navigable, and `enter`/`o` still work. The detail view truncates its columns in the same spirit. 0x0 renders nothing and never panics. Golden tests at 60x15, 80x24, 120x40 and 0x0 cover the list and the detail view (roadmap).

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
- How team and bot reviewers render (e.g. `…@org/team`, the bot login without `[bot]`). Use nil guards on the union fragments (roadmap pitfall: DISMISSED, team and bot reviewers, re-requests).
- What the detail view shows while its first fetch is loading, and how a failed detail fetch looks (keep the last data plus a marker, as in Phase 2 D-07).
- What the row shows if the failing-contexts query fails or comes back truncated: the badge stays `✗ failing` with no name. Never guess a name.
- Which key expands the collapsed passing checks.
- The exact column widths and caps for REVIEWERS / MERGE / CI, and the exact compact-mode threshold (about 40).
- Additions to the help legend for the new glyphs.

### Deferred Ideas (OUT OF SCOPE)
None. The discussion stayed within the phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| LIST-04 | Each row shows reviewers who approved or requested changes, and reviewers still pending | Discovery query gains `reviewRequests` plus author logins on `latestOpinionatedReviews`; merge rule "pending wins" verified live (re-request keeps the old opinionated review); team requests arrive as `null` for non-members (verified) |
| LIST-06 | Each row shows merge-readiness: mergeable, blocked, behind, or conflicts | `mergeStateStatus` (+ `mergeable` as a tiebreak for UNKNOWN); draft PRs report BLOCKED/DIRTY and merged PRs report UNKNOWN, so state/draft checks must run before the status mapping |
| LIST-07 | For failing PRs, the row shows the name of a failing check | Step-2 `nodes(ids:)` query over failing PRs only; dedupe by (workflow, check name) keep newest; D-02's "+N from counts" is wrong on real data (see Finding F1) |
| DETL-01 | Detail view lists every check/job with status, failing first | `node(id:)` detail query with `contexts(first:100, after:)` paging, per-check classification table, grouped sort |
| DETL-02 | Detail view lists each reviewer and their review state | Detail query adds `latestReviews` (the only place COMMENTED and DISMISSED appear) merged under the list rule |
| DETL-03 | Returning from the detail view keeps the same PR selected | Detail keeps its own `ref` + last-PR copy; `m.selected`/`rebuild()` untouched while open; `o` must use the detail PR, not `rows[cursor]` |
| NAV-05 | Layout adapts to resizes and stays usable in narrow panes | Pure column-layout function with the D-13 drop order, compact mode below ~40, viewport resize in the detail view, golden sizes 60x15, 80x24, 120x40, 0x0 |
</phase_requirements>

## Summary

The phase is mostly assembly on top of Phase 1-2 seams, but live probing on 2026-10-12 found six facts that change how the CONTEXT decisions must be implemented. The most important: (1) GitHub's `checkRunCountsByState` and the contexts list are **not de-duplicated across workflow runs** (a PR with 66 raw contexts collapses to 35 after dedupe, and 21 stale CANCELLED runs vanish), so the "+N from counts" rule in D-02 would print `+21` for a PR with one real failure; the `+N` has to come from the de-duplicated contexts. (2) De-duplication is needed across check suites only: across 4,676 check runs there were **zero** same-suite duplicates, including 150 runs with `runAttempt > 1`, so "re-run" in practice means "a newer workflow run on the same commit". (3) `mergeStateStatus` is `UNKNOWN` on first read for many open PRs and resolves on the next query (6/6 resolved within 5 s), draft PRs report `BLOCKED`/`DIRTY` (there is no DRAFT enum value), and merged PRs report `UNKNOWN`. (4) Team review requests come back as `requestedReviewer: null` for a viewer who is not in the org (101 of 456 requests, 0 resolved Teams), so the nil guard is the main path, not an edge. (5) A re-requested reviewer stays in `latestOpinionatedReviews`, so "pending wins" must be an explicit dedupe step. (6) The discovery query grows from cost 1 to cost 2 per 100-PR page; the step-2 and detail queries cost 1 each.

Architecture: keep the fetch pipeline inside `github.Client.Fetch` (discovery, then one chunked `nodes(ids:)` query for failing PRs, all in the same single-flight Cmd), add an optional `ui.DetailFetcher` interface so existing `fakeFetcher` tests do not change, put every classification rule in pure `internal/domain` functions (merge state, reviewer merge, check classification and dedupe, failing-name selection), and implement narrow-pane behaviour as one pure `layoutColumns` function next to `columnWidths`. The detail view is a new `viewState` with its own `ref` and last-data copy; it must not share `rows[cursor]`.

**Primary recommendation:** Build bottom-up and test-first: domain rules (table tests with the live-derived fixtures) → github mapping/fixtures (route the fixture transport by query name) → pure column layout + cells + compact mode → detail view and fetch plumbing → regenerate and add goldens at 60x15, 80x24, 120x40, 0x0. Resolve Findings F1 (counts vs dedupe), F5 (0x0 convention) and Open Question 3 (TITLE below its minimum) before locking plans.

## Findings that amend or sharpen CONTEXT decisions

Every item below was measured live on 2026-10-12 with the user's `gh` token, read-only (about 150 points of 5,000 used). Probe scripts were run from the session scratchpad and are not committed.

| # | Finding | Evidence | Effect on a decision |
|---|---------|----------|----------------------|
| F1 | Raw contexts and `checkRunCountsByState` count **superseded runs**. | microsoft/vscode PR 341132: `totalCount` 66, `{'CANCELLED': 21, 'FAILURE': 1, 'IN_PROGRESS': 5, 'SUCCESS': 39}`; after dedupe by (workflow, name) keeping the newest suite: 35 contexts, `{'fail': 1, 'pass': 29, 'run': 5}`. The two workflow runs were `Code OSS` run 70752 (cancelled) and run 70756 (newer). [VERIFIED: live gh api graphql 2026-10-12] | D-02 says `+N` "comes from `checkRunCountsByState`". On real data that prints `+21`. **Use the de-duplicated contexts for `+N` whenever the contexts page is complete; when it is truncated show no name and no `+N`** (consistent with the discretion item "never guess"). D-03 already says counts derived from contexts are de-duplicated; this resolves the D-02/D-03 conflict in D-03's favour. |
| F2 | Duplicates occur **across suites only**. | 4,676 check runs scanned in cli/cli, microsoft/vscode, home-assistant/core, golang/go, charmbracelet/bubbletea, rust-lang/rust, facebook/react: 273 (workflow, name) groups with more than one run, **0** duplicates inside one check suite, 150 runs with `runAttempt` > 1 (all showing only the latest attempt). [VERIFIED: live] | Resolves Spike Q2. Key = (workflow name, check name); for non-Actions apps (`workflowRun` is `null`, 428 runs) use `checkSuite.app.name` as the "workflow". |
| F3 | Which field orders the runs. | Newest by check-run `databaseId` equals newest by `checkSuite.createdAt` in 272 of 273 groups (the one miss: three SKIPPED runs in the same second). Newest by `startedAt` disagreed in 3 and tied in 6 groups. [VERIFIED: live] | Order key = `(checkSuite.createdAt, checkRun.databaseId)`, larger wins. Do not use `startedAt` (ties, and null while queued). |
| F4 | `mergeStateStatus` per PR state. | Drafts: `BLOCKED`/`DIRTY` (no DRAFT value exists: enum is `DIRTY, UNKNOWN, BLOCKED, BEHIND, UNSTABLE, HAS_HOOKS, CLEAN`). Merged: `UNKNOWN`/`UNKNOWN`. Closed: stale `BLOCKED`/`DIRTY`. [VERIFIED: live introspection and queries] | D-07's "DRAFT → –" and "merged/closed blank" must be decided from `IsDraft`/`State` **before** reading the status. A mapper keyed on the string alone would show merged PRs as `–` and closed PRs as blocked. |
| F5 | `UNKNOWN` is common and transient. | home-assistant/core oldest-updated open PRs: 6 of 6 `UNKNOWN`/`UNKNOWN` at t0, all resolved (`DIRTY`/`CONFLICTING` x4, `BLOCKED`/`MERGEABLE` x2) at t+5 s and unchanged at t+60 s. microsoft/vscode: 58 of 79 open non-draft PRs `UNKNOWN`/`MERGEABLE` on one read, 4 `UNKNOWN`/`CONFLICTING`. [VERIFIED: live] | Show `–` and let the next tick resolve it (no special retry needed). Optional refinement: `UNKNOWN` + `mergeable == CONFLICTING` → conflicts (see Open Question 5). |
| F6 | Team reviewers are `null` for non-members; Bots resolve. | 456 review requests across 13 repos: `User` 353, `null` 101, `Bot` 2 (`copilot-pull-request-reviewer`), `Team` 0. REST confirms the null ones are teams (cli/cli#14044: `teams: ['code-reviewers']`, GraphQL `requestedReviewer: null`, `asCodeOwner: true`). [VERIFIED: live] The `Team`/`EnterpriseTeam` branch for org members could not be observed (account is in no org). | Nil `requestedReviewer` = "a reviewer we cannot name", counted as pending. Render it, never drop it. |
| F7 | Re-request keeps the old opinionated review. | cli/cli#14580: `babakks` is in both `reviewRequests` and `latestOpinionatedReviews` (`APPROVED`); cli/cli#13899: `BagToad` in both with `CHANGES_REQUESTED` while absent from `latestReviews`. [VERIFIED: live] | Merge rule: pending (in `reviewRequests`) overrides any opinionated state for the same login (case-insensitive). |
| F8 | `DISMISSED` and `COMMENTED` only exist in `latestReviews`. | 6 repos: `latestReviews` states `{COMMENTED 294, APPROVED 251, CHANGES_REQUESTED 42, DISMISSED 18}`; `latestOpinionatedReviews` states `{APPROVED 180, CHANGES_REQUESTED 52}`. Review authors: `User` 360, `Bot` 245, null 0. [VERIFIED: live] | List rows use only `latestOpinionatedReviews` + `reviewRequests`. The detail view adds `latestReviews` for reviewers not otherwise listed (COMMENTED/DISMISSED shown faint). |
| F9 | Query cost. | Baseline Phase-1 query, `first: 100`: cost 1. With `mergeStateStatus mergeable reviewRequests(first:20)` and author logins: **cost 2** (`nodeCount` 4100, 46 KB for 72 PRs). Adding `contexts(first:1){...}` to discovery: cost 3. `nodes(ids:)` x 30 failing PRs with `contexts(first:100)`: cost 1, 243 KB, 2.9 s. `node(id:)` detail with 66 contexts: cost 1, 27 KB. [VERIFIED: live] | Do **not** add contexts/counts to the discovery query (CONTEXT's code_context suggested it; step 2 makes it unnecessary). A steady tick costs about 2 + 1 + 1 = 4 points, about 240 per hour. |

## Architectural Responsibility Map

prv is a single-binary TUI, so the "tiers" are its packages.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Wire structs, GraphQL strings, chunking, partial-error policy | `internal/github` | — | Only package that knows the wire format (package doc, client.go:1-3) |
| Remote-string sanitising (`domain.Clean`) for check names, workflow names, logins, team slugs | `internal/github` mapper seam | `internal/domain` (the function) | Phase 1 rule: clean at the seam, once |
| Merge-state mapping, reviewer merge/ordering, per-check classification, dedupe, failing-name selection | `internal/domain` (pure) | — | Pure, table-testable, next to `DeriveReview`/`ClassifyRollup` |
| Column visibility/width algorithm, cell rendering (reviewers `+N`, CI name + `+N`), compact rows | `internal/ui` (pure helpers in `layout.go`) | `internal/domain` for glyph/word | Display-width concerns belong to ui |
| Detail view state (ref, last data, scroll, expand, gone flag), key handling, fetch scheduling | `internal/ui` `Model` | `internal/github` for `FetchDetail` | Elm loop; I/O only inside `tea.Cmd` |
| Selection by `Ref`, nearest-neighbour after a PR leaves | `internal/ui` `rebuild()`/`nearest()` | — | Existing 02-02 rule; detail must not disturb it |
| Rate-limit accounting for the extra queries | `internal/github` (sums `Cost`, last `Remaining`) | `internal/domain.NextDelay` unchanged | Phase 2 backoff already reads `Snapshot.RateLimit` |

## Standard Stack

No new modules are needed. Everything is already in `go.mod` (read this session):

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/cli/go-gh/v2` | v2.16.2 | GraphQL client `DoWithContext`; fills the response struct, then returns `*api.GraphQLError` on partial errors | Already the project's client [CITED: .claude/CLAUDE.md, 01-SPIKE.md Q3] |
| `charm.land/bubbletea/v2` | v2.1.0 | Elm loop, `tea.KeyPressMsg`, `tea.WindowSizeMsg` | go.mod |
| `charm.land/bubbles/v2` | v2.2.1 | `viewport` (detail scrolling), `key`, `help` | go.mod; viewport API read in the module cache this session |
| `charm.land/lipgloss/v2` (+ `table`) | v2.0.6 | Width helpers, table rendering | go.mod |
| `github.com/charmbracelet/x/ansi` | v0.11.9 | `ansi.Truncate`, `ansi.Strip` | go.mod |
| `github.com/charmbracelet/x/exp/golden` | v0.1.0 | Golden files (`-update` regenerates) | go.mod |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `viewport.Model` for detail scroll | Hand-rolled `scroll int` + pure `detailLines(width)` | Hand-rolled is ~20 lines and trivially testable; viewport matches CLAUDE.md ("viewport (scrollable detail view)") and clamps itself. **Use viewport**, driven programmatically with `ScrollDown(1)`/`ScrollUp(1)` from prv's keymap, not its own `KeyMap` (its defaults bind `f`, `b`, `d`, `u`, `h`, `l`, `space`, which would clash with future keys). |
| Per-PR `statusCheckRollup` paging for truncated PRs | Accept "no name" for truncated PRs | See Open Question 2. |
| `CheckSuite.checkRuns(filterBy:{checkType: LATEST})` per suite | Dedupe client-side | The filter exists (`CheckRunFilter`: `checkType`, `appId`, `checkName`, `status`, `statuses`, `conclusions` [VERIFIED: live introspection]) but it dedupes within a suite only, and F2 shows within-suite duplicates do not occur. Rejected. |

**Installation:** none. **Version verification:** versions read from `/Users/scottb/dev/bartram/prv/go.mod` (module list: `charm.land/bubbles/v2 v2.2.1`, `charm.land/bubbletea/v2 v2.1.0`, `charm.land/lipgloss/v2 v2.0.6`, `github.com/cli/go-gh/v2 v2.16.2`).

## Package Legitimacy Audit

No external packages are added in this phase. The `package-legitimacy` seam supports npm/pypi/crates only, so it was not run; nothing is installed.

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                tick / r / startup
                        │
                        ▼
        ui.Model.beginFetch (single-flight, gen counter)
                        │  tea.Cmd (panic-wrapped)
                        ▼
   github.Client.Fetch ───────────────────────────────────────────────┐
     1. discovery  search(ISSUE_ADVANCED, first:100) × ≤5 pages        │ cost ≈ 2/page
        + mergeStateStatus, mergeable, reviewRequests,                 │
          latestOpinionatedReviews{state author{login}}                │
     2. failing set = PRs with rollup FAILURE/ERROR                    │
        nodes(ids:[≤25]) { contexts(first:100) … }  per chunk          │ cost 1/chunk
        step-2 error ⇒ keep step-1 data, drop names, add warning;      │
        rate limit / cancel ⇒ whole fetch errors (Phase 2 rule)        │
     3. mapper (domain.Clean at the seam) → domain.Snapshot           ◄┘
                        │ fetchedMsg{gen, snap, err}
                        ▼
   ui.Update ── rebuild(): rows = Visible(snap.PRs), cursor from m.selected
        │
        ├─ list screen: layoutColumns(natural, width) → cells → lipgloss table
        │      compact (<~40 cols): "> web#7 ✗⊘✗"
        │
        └─ enter ─► detail state {ref, pr copy, gone, scroll, expand}
                     │ immediately: detailSeq++, Cmd FetchDetail(ctx, pr.ID)
                     │ on every successful fetchedMsg while open: another FetchDetail
                     ▼
              github.Client.FetchDetail  node(id:) { reviewers…, contexts(first:100, after) × ≤5 }
                     │ detailMsg{seq, ref, detail, err}   (dropped if seq or ref is stale)
                     ▼
              domain.Detail → viewport.SetContentLines → View()
```

### Recommended Project Structure
```
internal/domain/
├── pr.go          # PR gains: Merge, Reviewers, FailingCheck (+ Check/Detail types in checks.go)
├── status.go      # + Merge enum, DeriveMerge, Glyph() methods beside Badge()
├── reviewers.go   # Reviewer, ReviewerState, MergeReviewers (dedupe, pending wins, order)
├── checks.go      # Check, CheckState, ClassifyCheck, LatestChecks, SummarizeFailing, Counts
internal/github/
├── client.go      # discoveryQuery extended; Fetch calls fetchFailing after discovery
├── checks.go      # failingQuery, detailQuery, wire structs, FetchDetail
└── testdata/      # failing_contexts.json, detail_*.json, status_reviewers.json (see Testing Notes)
internal/ui/
├── layout.go      # columns table + layoutColumns (pure) replacing the fixed colCount math
├── cells.go       # reviewersCell, mergeCell, ciCell, compactRow (pure, width-aware)
├── detail.go      # detail state, detailLines(width), update/render helpers
├── keys.go        # + Enter, Back, Expand; DetailShortHelp via the same keyMap
└── model.go/view.go/refresh.go/help.go  # wiring
```

### Pattern 1: Pure domain classification (put beside `DeriveReview`)
**What:** Every rule from F1-F8 is a pure function with a table test.
**Existing code to match** (quoted verbatim from `internal/domain/status.go:42-55`, read this session):
```go
func ClassifyRollup(state *string) CI {
	if state == nil {
		return CINone
	}
	switch *state {
	case "SUCCESS":
		return CIPassing
	case "FAILURE", "ERROR":
		return CIFailing
	case "PENDING", "EXPECTED":
		return CIRunning
	}
	return CIUnknown
}
```
Failing-set selection in `Fetch` must reuse `pr.CI == domain.CIFailing` (which is exactly rollup FAILURE/ERROR) rather than re-testing the string.

**Merge mapping (recommended):**
```go
// Merge is the merge-readiness shown for an open PR. The zero value is MergeNone
// (blank cell): merged, closed and unknown-state PRs. (zero value = non-favourable, as for Review/CI)
type Merge int
const ( MergeNone Merge = iota; MergeUnknown; MergeMergeable; MergeBlocked; MergeBehind; MergeConflicts )

func DeriveMerge(st State, draft bool, status, mergeable string) Merge {
	if st != StateOpen { return MergeNone }       // F4: merged=UNKNOWN, closed=stale BLOCKED/DIRTY
	if draft { return MergeUnknown }               // F4: drafts report BLOCKED/DIRTY; there is no DRAFT value
	switch status {
	case "CLEAN", "HAS_HOOKS", "UNSTABLE": return MergeMergeable   // D-07
	case "BLOCKED": return MergeBlocked
	case "BEHIND":  return MergeBehind
	case "DIRTY":   return MergeConflicts
	}
	if mergeable == "CONFLICTING" { return MergeConflicts }        // Open Question 5 (optional)
	return MergeUnknown                                             // UNKNOWN or any new value: never "mergeable"
}
```
Badges: `✓ mergeable`, `⊘ blocked`, `↓ behind`, `✗ conflicts`, `–` (and blank). `⊘` is East-Asian-width N; `↓` is A (like the already-used `●`, `○`, `–`, `…`) [VERIFIED: python unicodedata]. `⇣` (N) is the narrow-safe alternative if the deferred ambiguous-width UAT fails. Add `Glyph()` methods and a test that `Badge()` starts with `Glyph()` so compact mode cannot drift from the badges.

**Reviewer merge (recommended):**
```go
type ReviewerState int // zero value: pick the least favourable known state (Pending), as Review/CI do
// Display order (D-06): ChangesRequested, Pending, Approved; the detail view adds Commented, Dismissed (faint, last).
type Reviewer struct{ Name string; Kind ReviewerKind; State ReviewerState } // Kind: User, Bot, Team, Hidden
func MergeReviewers(opinionated []OpinionatedReview, requested []Requested, latest []OpinionatedReview /*detail only*/) []Reviewer
```
Rules: key = lower-cased login; request wins over any review (F7); opinionated states other than APPROVED/CHANGES_REQUESTED are ignored (defensive); `latest` entries are added only for logins not already present (F8) as Commented/Dismissed; nil author on a review → skip (schema allows null; none observed); nil `requestedReviewer` → one `Hidden` pending entry per request, rendered together as `…team` (one) or `…N teams`; `Team`/`EnterpriseTeam` → name = `combinedSlug` (rendered `…@org/slug`); `Bot`/`Mannequin` → login. Within a group sort case-insensitively by name so rows do not reshuffle between refreshes (GitHub's order is not documented).

**Check classification (verified enums, live introspection 2026-10-12):** `CheckRun.status` ∈ `REQUESTED QUEUED IN_PROGRESS COMPLETED WAITING PENDING`; `CheckRun.conclusion` ∈ `ACTION_REQUIRED TIMED_OUT CANCELLED FAILURE SUCCESS NEUTRAL SKIPPED STARTUP_FAILURE STALE`; `StatusContext.state` ∈ `EXPECTED ERROR FAILURE PENDING SUCCESS`; `StatusCheckRollupContext` is a union of `CheckRun` and `StatusContext`.

| Input | CheckState | Display word | Colour |
|-------|-----------|--------------|--------|
| CheckRun status != COMPLETED (QUEUED, REQUESTED, PENDING → "queued"; IN_PROGRESS → "running"; WAITING → "waiting") | running | as left | yellow `●` |
| COMPLETED + FAILURE / TIMED_OUT / STARTUP_FAILURE / ACTION_REQUIRED | failing | "failure", "timed out", "startup failure", "action required" | red `✗` |
| COMPLETED + CANCELLED | cancelled | "cancelled" | **not red**: default foreground, `⊘` |
| COMPLETED + SUCCESS | passing | "success" | green `✓` |
| COMPLETED + SKIPPED / NEUTRAL / STALE | neutral | "skipped" / "neutral" / "stale" | faint `–` |
| StatusContext ERROR / FAILURE | failing | "error" / "failure" | red |
| StatusContext PENDING / EXPECTED | running | "pending" / "expected" | yellow |
| StatusContext SUCCESS | passing | "success" | green |
| any unrecognised value | neutral-unknown | "unknown" | faint `?` (total mapping; never red, never passing) |

**Dedupe + failing name:**
```go
// LatestChecks keeps, per (Workflow, Name), the check with the largest (SuiteCreatedAt, ID).
// Workflow = workflowRun.workflow.name, else checkSuite.app.name, else "" (StatusContext: Workflow "").
// SummarizeFailing returns the row's name: first failing-class check (sorted by name), else first
// cancelled (Cancelled=true); More = (failing+cancelled) - 1; Known=false when !complete or none found.
func SummarizeFailing(latest []Check, complete bool) FailingCheck
```
If the PR's rollup is failing but the de-duplicated set holds no failing or cancelled check (the run finished between the two queries), return `Known=false`: the badge stays `✗ failing` with no name.

### Pattern 2: Fetch pipeline stays inside `Client.Fetch`
- After the discovery loop and window filter, collect `ids` of PRs with `pr.CI == domain.CIFailing`, chunk by **25** (measured: 30 PRs took 2.9 s and 243 KB; the hard limit is 100 ids [CITED: 01-SPIKE.md Q6]; the client timeout is 20 s), cap total named PRs at 100 per refresh, run chunks sequentially.
- Policy for step-2 errors (mirrors the 02-07 rule): rate limit (`KindRateLimited`) or context cancellation → return the error for the **whole** fetch (previous data stays, model pauses); any other error (including a partial `*api.GraphQLError` or a NOT_FOUND `nodes[i]` null for a PR that vanished) → keep the discovery data, leave those rows unnamed, add one warning via the existing `addWarning` ("Some data could not be loaded" is already the footer note text; use a distinct snapshot warning such as "Check names could not be loaded").
- Sum `RateLimit.Cost` across all queries and take `Limit/Remaining/ResetAt` from the **last** response (same convention as the discovery loop: `snap.RateLimit.Cost += pageSnap.RateLimit.Cost`, client.go:200-203).
- Wire structs: `Nodes []*wireFailNode` (GraphQL nulls an element), `StatusCheckRollup *struct{...}` nullable, contexts nodes decoded through a tagged union struct with both `CheckRun` and `StatusContext` fields and a `Typename`.

### Pattern 3: Detail fetch as a separate, sequenced Cmd
- Add an optional interface so existing tests keep compiling: `type DetailFetcher interface { FetchDetail(ctx context.Context, id string) (domain.Detail, error) }`; `ui` type-asserts `m.fetcher.(DetailFetcher)`; `*github.Client` implements it; `fakeFetcher` is untouched (a second fake gains the method only in detail tests).
- Triggers: `enter` on the list (immediate), and the success branch of `fetchedMsg` while the detail view is open and no detail fetch is in flight. State: `detailSeq int`, `detailInFlight bool`. A `detailMsg` whose `seq != m.detailSeq` or whose `ref != m.detail.ref` is dropped (stale after `esc` + another `enter`).
- This is the one deliberate reading of D-11: user-visible behaviour is identical ("one small query per tick, only while open"), but the detail query is a second sequenced Cmd rather than part of the first Cmd, because an `enter` pressed during an in-flight refresh must not wait for the next tick (see Open Question 6).
- Pagination: `contexts(first:100, after: $after)` loop in `FetchDetail`, at most 5 pages (same cap as `maxPages`, client.go:144); `Detail.Truncated` set when the cap is hit. Summary line: from de-duplicated contexts when complete, otherwise from `checkRunCountsByState` + `statusContextCountsByState` with a trailing marker, because past the cap counts are the only complete data (D-08).
- Errors: keep the last `Detail`, set `detailErr`, show a faint `✗ could not refresh details` line (Phase 2 D-07 style). A rate-limit error from the detail Cmd should also set `pausedUntil` through the same `backoffer` interface used by `schedule`.

### Pattern 4: Pure column layout (NAV-05)
Replace the fixed `colCount` math with a column table (`id`, `header`, `natural width`, `min width`, `dropRank`) and one pure function returning visibility and widths. Existing helpers to keep, quoted from `internal/ui/layout.go:20-28`:
```go
const (
	columnGap    = 1  // spaces between columns
	minTitleW    = 10 // TITLE never gets fewer cells than this
	maxPRColW    = 40 // cap for the PR column
	maxAuthorCol = 16 // cap for the AUTHOR column
	// chromeLines are the lines around the rows: the "prv · @login" line, the
	// table's column-header row and the footer line.
	chromeLines = 3
)
```
Algorithm (all measurements via `lipgloss.Width`):
1. Natural widths: PR (cap 40), AUTHOR (cap 16), STATE, REVIEW (10), REVIEWERS (cap about 22), MERGE (11, from `✓ mergeable`), CI (cap about 24; name part is truncated first, `(cancelled)` marker and `+N` suffix are kept).
2. TITLE shrinks to `minTitleW`. If still too wide:
3. drop AUTHOR, then STATE (D-13).
4. REVIEWERS collapses to a count-only `+N` form (width about 4, header shortened), then is dropped.
5. drop MERGE.
6. If PR + REVIEW + CI (+ marker + gaps) still do not fit: see Open Question 3 (TITLE shrinking below its minimum is the gap in D-13), then CI name truncates to the bare badge.
7. If total width is below the compact threshold (about 40) → compact rows `> repo#N glyphs`; the header line carries a "widen pane" hint.
A width of `<= 0` means "not sized yet": show every column uncut, as `columnWidths` does today (see Open Question 4).

Cell rules to implement as pure functions:
- **REVIEWERS:** tokens `✗bob …carol ✓alice` (glyph glued to the login; viewer → `me`), greedy fill to the cell width, remainder `+N`; if even one token does not fit, show `+N` only.
- **CI:** failing with known name → `✗ lint +2`; cancelled-only → `✗ deploy (cancelled)` (+N if more); unknown → the plain badge. Truncate order: name text with `…`, never the suffix.
- **Check names are untrusted:** run through `domain.Clean` at the mapper, then `truncateCell` (cells, not bytes).

### Pattern 5: Detail state and DETL-03 / D-12
```go
type detailState struct {
	open     bool
	ref      domain.Ref
	pr       domain.PR      // last list data for this ref (header, review, merge)
	gone     bool           // faint "no longer in list"
	data     *domain.Detail // last successful detail fetch
	err      error
	expanded bool           // D-10 expand key
	vp       viewport.Model
}
```
- On every successful `fetchedMsg` while open: look up `ref` in `m.snap.PRs` (not `m.rows`; filters cannot change while detail is open, but rows can be empty). Found → refresh `pr`, `gone=false`. Missing → keep the last `pr`, `gone=true`.
- `m.selected`/`rebuild()`/`nearest()` run unchanged: when the PR leaves, `rebuild()` already moves `m.selected` to the nearest neighbour using the old order (model.go:407-411). `esc` then only closes the view. Result matches D-12 and DETL-03 with no extra selection code.
- `j/k/up/down` scroll the viewport and **must not** change `m.cursor`. `o` must open `detail.pr.URL` (through `safeURL`), never `m.rows[m.cursor]` (that row can be a different PR after a refresh).
- Key precedence in `Update`: help overlay first (existing `m.showHelp` branch, so `?` works in the detail view and returns to it), then prompt, then the detail branch, then list keys. `esc` is already bound to `Clear` in list mode (keys.go:55); the detail branch must run before it. `q`/`ctrl+c` quit in the detail view (D-09).
- Resize: `viewport.SetWidth/SetHeight` then `SetContentLines(detailLines(width))`; `SetContentLines` clamps the offset (viewport.go, `if m.YOffset() > m.maxYOffset() { m.GotoBottom() }`), which keeps scroll valid when content shrinks after a refresh. Call one `syncDetail()` on resize, data change and expand toggle.
- Expand key: `e` (free; existing keys are `j k up down r q ctrl+c ? o 1 2 3 / esc enter`).
- Footer in detail view: reuse `footer()` with a different `ShortHelp` set from the same `keyMap` (`DetailShortHelp`: back, open, refresh, scroll, expand, help, quit) so the help overlay stays the single source (Phase 2 D-17).
- Viewport specifics: `View()` returns `""` when width or height is 0, and pads every line to full width with spaces; goldens run `ansi.Strip` only, so right-trim lines before joining (the current list view output is not trimmed either, see `list_80x24.golden`, so either is consistent; pick one and keep it).

### Anti-Patterns to Avoid
- **Taking the first failing context from a truncated page as "the" failing check:** a re-run on page 2 may have fixed it. Truncated → no name.
- **Counting from `checkRunCountsByState` for `+N`:** over-counts superseded runs (F1).
- **Keying the merge mapping on the string only:** wrong for drafts, merged, closed (F4).
- **Rendering `requestedReviewer` without a nil guard:** `null` is the normal case for teams (F6); an unguarded `.Login` panics in the mapper.
- **Using `rows[cursor]` anywhere in detail code.**
- **Putting contexts/counts in the discovery query:** +1 point per 100 PRs for data only failing PRs need.
- **Calling `fmt.Println`/logging from the new Cmds** (corrupts the alt screen; Phase 2 rule).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cell width / truncation with CJK, emoji, ZWJ | `len`, `utf8.RuneCountInString`, byte slicing | `lipgloss.Width`, `ansi.Truncate`, existing `truncateCell` | Grapheme-correct; goldens already pin CJK and ZWJ cases |
| Scrollable detail body | A second hand-written paging model | `charm.land/bubbles/v2/viewport` driven by `ScrollDown/ScrollUp/SetContentLines` | Clamping on content change is built in |
| Footer/header priority fitting | A new truncation scheme | Existing `fit(segs, sep, width)` and `segment` priorities (view.go:220-257) | Already tested; reuse for the detail header |
| Key hints/help | Hard-coded hint strings | `keyMap` + `help.Model` (`ShortHelp`/`FullHelp`) | One source for footer and `?` overlay |
| Sanitising remote strings | Per-view escaping | `domain.Clean` at the mapper | Check/workflow/context names and team slugs are attacker-influenced |
| Rollup state | Recomputing pass/fail from contexts | The rollup `state` via `ClassifyRollup` | Truncated pages and superseded runs make recomputation wrong (roadmap pitfall) |
| Table borders/padding | ANSI string padding by hand | `lipgloss/table` with `StyleFunc` (existing `renderList`) | Existing pattern; width includes padding in v2 (view.go:379-385) |
| Fetch dedupe / chunking maps | Ad-hoc loops everywhere | One `chunk[T any](s []T, n int) [][]T` helper and `slices.SortStableFunc` | Standard library is enough (Go 1.26) |

**Key insight:** every "obviously simple" step here (name the failing check, count reviewers, say mergeable) has a live-observed trap; the traps are in the data, so the defence is a small set of pure, table-tested domain functions fed by recorded fixtures, not UI logic.

## Common Pitfalls

### Pitfall 1: Stale superseded runs inflate the failing count
**What goes wrong:** `+21` on a PR with one real failure; the detail summary says "21 failing".
**Why:** contexts and per-state counts include every workflow run on the commit (F1).
**How to avoid:** `LatestChecks` before any count or name; counts only as a labelled fallback when paging is capped.
**Warning signs:** a fixture where `totalCount` differs from the de-duplicated length; add the vscode-shaped fixture.

### Pitfall 2: Merged PR shown as "–" / closed PR shown as "blocked"
**Why:** F4. **Avoid:** `DeriveMerge` checks `State` and `IsDraft` first. **Test:** merged → blank, closed+BLOCKED → blank, draft+BLOCKED → `–`.

### Pitfall 3: Team reviewer panics or disappears
**Why:** `requestedReviewer: null` (F6). **Avoid:** wire type `*struct{Typename string; Login string; CombinedSlug string}`; nil → Hidden pending; count it. **Test:** fixture node with null, User, Bot, Team, EnterpriseTeam, Mannequin.

### Pitfall 4: Re-requested reviewer shown as approved
**Why:** F7. **Avoid:** pending wins by lower-cased login. **Test:** same login in both lists.

### Pitfall 5: A COMMENTED review hides an approval (or DISMISSED confuses the list)
**Why:** `latestReviews` replaces APPROVED with COMMENTED (Q1); DISMISSED only exists there (F8). **Avoid:** list rows never read `latestReviews`; detail reads it only to add reviewers who are absent from the other two sources.

### Pitfall 6: Badge from a truncated contexts page
**Why:** roadmap pitfall; `hasNextPage` true when `totalCount` is 131-355 (Q9). **Avoid:** badge stays `ClassifyRollup`; step 2 only supplies the name; `hasNextPage` → `Known=false`.

### Pitfall 7: Detail view opens the wrong PR in the browser or scrolls the list
**Why:** reuse of `rows[cursor]` and the list Up/Down branches (model.go:257-269). **Avoid:** separate detail branch placed before them; test "refresh drops PR, press o" opens the detail PR's URL.

### Pitfall 8: Stale detail response applied to the wrong PR
**Why:** `esc` then `enter` on another PR while the first detail Cmd is in flight. **Avoid:** `detailSeq` + ref check; test with out-of-order `detailMsg`.

### Pitfall 9: Existing fetch tests fail on an unexpected second request
**Why:** `fixtureTransport` fails the test on a request beyond its file list (`unexpected request %d; only %d fixtures`, client_test.go:30-32), and `status.json` contains failing PRs, so every existing `Fetch` test now triggers step 2. **Avoid:** route the fixture transport by operation name in the request body (`query Watch` vs `query FailingChecks` vs `query Detail`) with per-operation file lists; keep existing tests on their fixtures and add a step-2 fixture for them (or an empty-names default).

### Pitfall 10: All existing goldens change
Every list golden gains columns. Regenerate with `go test ./internal/ui -update` after reviewing the diff by eye; `TestTitleTruncatesFirst` asserts the header is exactly `PR,TITLE,AUTHOR,STATE,REVIEW,CI` (view_test.go:~290) and must be rewritten for the new column set and drop order. The plan should schedule golden regeneration as its own reviewed task, not a side effect.

### Pitfall 11: Glyph/width drift between badge and compact mode
**Avoid:** one `Glyph()` per enum, `Badge()` built from it, a test asserting the prefix.

### Pitfall 12: `0x0` ambiguity
Existing convention: width/height 0 means "not sized yet", and the list renders every row uncut (see `list_0x0.golden`). CONTEXT D-14 says "0x0 renders nothing". Pick one in planning (Open Question 4); both must not panic. Note `viewport.View()` already returns `""` at 0x0.

## Code Examples

### Discovery query additions
Existing wire struct (verbatim from client.go:132-138, read this session):
```go
		// Only the opinionated-review connection is requested on purpose: in the
		// non-opinionated one a later COMMENTED review hides an earlier approval
		// (01-SPIKE.md Q1).
		LatestOpinionatedReviews struct {
			Nodes []*struct{ State string }
		}
		StatusCheckRollup *struct{ State string }
```
Change `Nodes` to carry `Author *struct{ Login string }` (the GraphQL text already requests `author { login }`, client.go:79, but the struct drops it), and add:
```graphql
        mergeStateStatus
        mergeable
        reviewRequests(first: 20) { nodes { requestedReviewer { __typename
          ... on User { login } ... on Bot { login } ... on Mannequin { login }
          ... on Team { combinedSlug } ... on EnterpriseTeam { combinedSlug } } } }
```
[VERIFIED: live, first:100, cost 2, no errors, 2026-10-12]. `reviewRequests(first: 20)` is enough in practice (max `totalCount` seen: 13 in 100 home-assistant PRs); request `totalCount` too if a "+N more" guard is wanted.

### Step 2: failing contexts (one chunk)
```graphql
query FailingChecks($ids: [ID!]!) {
  rateLimit { limit cost remaining resetAt }
  nodes(ids: $ids) {
    __typename
    ... on PullRequest {
      id
      statusCheckRollup {
        state
        contexts(first: 100) {
          totalCount
          pageInfo { hasNextPage }
          nodes {
            __typename
            ... on CheckRun { name status conclusion databaseId
              checkSuite { createdAt app { name } workflowRun { workflow { name } } } }
            ... on StatusContext { context state }
          }
        }
      }
    }
  }
}
```
[VERIFIED: live, 30 ids, cost 1, no errors. `app { name }` and `workflowRun { workflow { name } }` were each verified in the larger detail query; this exact trimmed text has not been run as written, so the first fixture-recording task should run it once.] Pass `ids` as a JSON variable (`gh api graphql --input -` with `{"query":…,"variables":{"ids":[…]}}`; `-F ids=…` sends a string and fails with NOT_FOUND, which is a probing trap, not a client issue: go-gh marshals `[]string` correctly).

### Detail query
```graphql
query Detail($id: ID!, $after: String) {
  rateLimit { limit cost remaining resetAt }
  node(id: $id) {
    __typename
    ... on PullRequest {
      id state isDraft reviewDecision mergeStateStatus mergeable
      latestOpinionatedReviews(first: 50) { nodes { state author { __typename login } } }
      latestReviews(first: 50) { nodes { state author { __typename login } } }
      reviewRequests(first: 50) { nodes { requestedReviewer { __typename
        ... on User { login } ... on Bot { login } ... on Mannequin { login }
        ... on Team { combinedSlug } ... on EnterpriseTeam { combinedSlug } } } }
      statusCheckRollup {
        state
        contexts(first: 100, after: $after) {
          totalCount
          checkRunCountsByState { state count }
          statusContextCountsByState { state count }
          pageInfo { hasNextPage endCursor }
          nodes {
            __typename
            ... on CheckRun { name status conclusion databaseId
              checkSuite { createdAt app { name } workflowRun { workflow { name } file { path } } } }
            ... on StatusContext { context state }
          }
        }
      }
    }
  }
}
```
[VERIFIED: live against microsoft/vscode#341132, cost 1, 27 KB, no errors; the version run also selected `detailsUrl`/`targetUrl`, which this phase does not need and should not fetch]. Show `path.Base(file.path)` (for example `pr.yml`) in the WORKFLOW column of a check line, `app.name` when `workflowRun` is null, nothing for a StatusContext.

### Domain skeleton for the failing summary (values from the verified enums above)
```go
func ClassifyCheck(typename, status, conclusion, state string) CheckState {
	if typename == "StatusContext" {
		switch state {
		case "SUCCESS": return CheckPassing
		case "ERROR", "FAILURE": return CheckFailing
		case "PENDING", "EXPECTED": return CheckRunning
		}
		return CheckOther
	}
	if status != "COMPLETED" { return CheckRunning }
	switch conclusion {
	case "SUCCESS": return CheckPassing
	case "FAILURE", "TIMED_OUT", "STARTUP_FAILURE", "ACTION_REQUIRED": return CheckFailing
	case "CANCELLED": return CheckCancelled
	case "SKIPPED", "NEUTRAL", "STALE": return CheckNeutral
	}
	return CheckOther
}
```

### Keys (extends the existing `keyMap`; binding style quoted from keys.go:33-35 and 55)
```go
		Accept: bkey.NewBinding(bkey.WithKeys("enter")),
		Cancel: bkey.NewBinding(bkey.WithKeys("esc")),
		...
		Clear:  bkey.NewBinding(bkey.WithKeys("esc"), bkey.WithHelp("esc", "clear #")),
```
Add `Enter` (`"enter"`, help `enter detail`), `Back` (`"esc"`, help `esc back`), `Expand` (`"e"`, help `e expand`); `ShortHelp` for the list gains `Enter`; add `DetailShortHelp()`.

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| `commits(last:1){…statusCheckRollup}` | `PullRequest.statusCheckRollup` directly | Already used (Phase 1) |
| REST per-PR `/check-runs` | one `nodes(ids:)` GraphQL call per ≤25 failing PRs | cost 1 |
| `mergeStateStatus: DRAFT` (REST `mergeable_state: "draft"`) | GraphQL enum has no DRAFT; use `isDraft` | [VERIFIED: introspection] |

**Deprecated/outdated:** `isRequired` on check contexts (dropped for v1, Q10); `tea.KeyMsg`/`View() string` (v1 APIs, project rule).

## Runtime State Inventory

Not a rename/refactor/migration phase. Omitted.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `... on Team { combinedSlug }` resolves (and renders as `@org/slug`) for a viewer who is a member of the org. Not observable here: the account is in no org and every team request returned `null`. | Findings F6, Pattern 1 | Team names would show as the generic hidden-team form for org members; cosmetic. Add a UAT item. |
| A2 | `UNKNOWN` + `mergeable: CONFLICTING` should render as conflicts. Observed as a state (4 of 79 vscode PRs) but the follow-up value for those PRs was not re-read; for the home-assistant `UNKNOWN`/`UNKNOWN` PRs the follow-up was `DIRTY`/`CONFLICTING`. | Pattern 1, Open Question 5 | Without it those PRs show `–` for one tick; with it the rare case where `mergeable` lags is shown as conflicts early. Needs user acknowledgement because D-07 says UNKNOWN → `–`. |
| A3 | Step-2 chunk size of 25 and the 100-PR-per-refresh cap are sensible. Based on one measurement (30 PRs: 2.9 s, 243 KB). | Pattern 2 | Slow refreshes if many PRs fail at once; tune later. |
| A4 | Same-suite re-run duplicates never occur. Based on 4,676 runs / 150 attempt>1 runs, but none of the sampled attempt>1 runs had a visible earlier failing attempt to prove the old attempt is hidden (only that no duplicates appeared). Dedupe by (workflow, name) is kept anyway, so the rule is safe either way. | F2 | None for correctness; only affects how strongly the fixture claims "re-run" semantics. |
| A5 | `↓ behind` and `⊘ blocked` render in one cell in the user's terminal font. `↓` is East-Asian-ambiguous like the glyphs already shipped (`●`, `○`, `–`, `…`). | Pattern 1 | Misaligned MERGE column on CJK-ambiguous terminals; covered by the deferred ambiguous-width UAT. Fallback `⇣`. |
| A6 | Detail-fetch rate-limit errors can feed the existing pause (`pausedUntil`) without extra rules. | Pattern 3 | A rate-limited detail fetch might not pause refreshes; low risk (the next discovery fetch would hit the same limit and pause). |
| A7 | Compact threshold 40 and column caps (REVIEWERS 22, CI 24) are reasonable starting points. | Pattern 4 | Goldens will show; adjustable. |

## Open Questions

1. **D-02 versus D-03 (counts vs dedupe).**
   - What we know: counts include superseded runs (F1); D-03 says derived counts are de-duplicated.
   - Recommendation: `+N` from de-duplicated contexts; truncated page → no name, no `+N`. Treat as the resolution unless the user objects. Tell the user, since D-02's text said otherwise.

2. **Naming PRs whose contexts are truncated.**
   - What we know: pytorch-scale matrices (131-355 contexts) exceed 100; CONTEXT says unnamed `✗ failing`.
   - Optional improvement: for truncated failing PRs only, page `node(id:){…contexts(first:100, after)}` up to 2 more pages (1 point each) and name from the complete set. Recommendation: not in v1 plans; keep `SummarizeFailing(…, complete bool)` so it can be added.

3. **What happens to TITLE between "TITLE at minimum" and compact mode (about 40-50 columns).**
   - What we know: marker 2 + PR 13 + REVIEW 11 + CI 11 + gaps is about 40 before TITLE's 10-cell minimum, so D-13's order (after MERGE drops) leaves no room for TITLE's minimum around 40-50 columns.
   - Recommendation: allow TITLE to drop entirely after MERGE has dropped and before CI/REVIEW narrow. State the rule in the plan.

4. **0x0 behaviour.** CONTEXT: "renders nothing". Existing convention: 0x0 = unsized, render everything uncut (`list_0x0.golden` is 32 lines). Recommendation: keep the established convention for list and detail (compact mode and column dropping apply only when width > 0); the requirement that matters is no panic. Confirm with the user in plan review.

5. **`UNKNOWN` + `CONFLICTING` → conflicts (A2).** Recommendation: include; it needs the extra scalar `mergeable` (already in the verified query).

6. **D-11 mechanism.** Recommendation as in Pattern 3 (separate sequenced Cmd after each successful refresh while open, plus an immediate one on `enter`). If the user wants the literal "inside the same Cmd", the open-time fetch still needs its own Cmd, so the two-path version is unavoidable.

7. **Hidden-team text.** Recommendation: `…team` (one) / `…2 teams` (several) in the list, and "team (name hidden)" in the detail view.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | build/test | ✓ | go1.27.2 darwin/arm64 (go.mod floor 1.26.0) | — |
| `gh` CLI | recording fixtures / validating queries | ✓ | 2.102.0, logged in as `sbartram` (keyring) | — |
| GitHub GraphQL API | live probing | ✓ | rate limit 5,000/hr, about 4,850 left after research | — |
| Test PRs with failing CI in the user's own account | live UAT | ✗ | the account has 0 open PRs (Phase 1 spike) apart from throwaway `sbartram/prv#1` | Use public repos (cli/cli, microsoft/vscode) as stand-ins via `search` with `repo:` qualifiers; the app itself only lists the viewer's PRs, so UAT uses fixtures or a temporary wider view |

**Missing dependencies with no fallback:** none.
**Note:** `go test ./...` passes on the current branch (verified at session start).

## Testing Notes

(`workflow.nyquist_validation` is `false` in `.planning/config.json`, so no Validation Architecture section; these notes are for the planner.)

- Framework: stdlib `testing` + `charmbracelet/x/exp/golden` (`golden.RequireEqual`, regenerate with `-update`). Quick: `go test ./internal/domain ./internal/github`. Full: `go test ./...`.
- **Domain (table tests, no I/O):** `DeriveMerge` (every enum value x open/draft/merged/closed/unknown-state, plus UNKNOWN+CONFLICTING), `MergeReviewers` (pending-wins, nil reviewer, bot, team, dismissed, commented, ordering), `ClassifyCheck` (every value in the verified tables + unknown), `LatestChecks` (cross-suite duplicates, ties, null workflow → app name, StatusContext), `SummarizeFailing` (failing before cancelled, cancelled-only marker, truncated → unknown, none found → unknown, `More` arithmetic).
- **Fixtures to record:** the vscode-shaped PR (66 raw → 35 after dedupe, FAILURE + stale CANCELLED), a cancelled-only PR, a hidden-team request, a re-request, a DISMISSED review in `latestReviews`, a 100-context page with `hasNextPage: true`, a `nodes` array with a `null` element plus a NOT_FOUND error, a draft PR with `BLOCKED`, a merged PR with `UNKNOWN`. Fixture strings with ANSI/bidi in a check name and a login (extend the existing hostile-PR pattern, `TestMapperCleansRemoteStrings`).
- **github package:** make `fixtureTransport` choose the response file by the operation name found in the request body; assert request count and that step 2 sends only failing, deduped ids chunked at 25 and never more than 100; assert cost summation and that a rate-limited step 2 returns the error.
- **UI:** golden sizes 60x15, 80x24, 120x40, 0x0 for list (all existing states) and for detail (loaded, loading, error-with-stale, gone, expanded, scrolled). Extend `TestViewFitsScreen` (no line wider than the pane, no more lines than the pane) to 60x15 and to detail. Add a property-style test over widths 1..200 and heights 1..50 that both screens never panic and never exceed the pane. Model tests: `enter`/`esc` round trip keeps `selected`; refresh while open reorders rows; refresh removes the PR (gone note, `o` opens the detail PR's URL, `esc` lands on the nearest neighbour); stale `detailMsg` dropped; `j/k` in detail does not move the cursor; `q` quits; `?` returns to the detail view.
- Existing tests that will be rewritten: `TestTitleTruncatesFirst`, all `TestGolden*` goldens, `TestBadgesReadWithoutColour` (extend), `TestHelpGolden` (legend gains the new glyphs).

## Security Domain

`security_enforcement` is on (ASVS level 1, block on high).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no (no new auth; token flow unchanged) | existing `auth.TokenForHost` |
| V3 Session Management | no | — |
| V4 Access Control | no (read-only; no write actions) | — |
| V5 Input Validation | **yes** | `domain.Clean` on every new remote string at the mapper: check names, workflow names, workflow file paths, app names, StatusContext names, reviewer logins, team slugs; GraphQL variables only, never string concatenation; ids come from the API and are passed as `$ids`/`$id` |
| V6 Cryptography | no | — |
| V7 Error handling / logging | **yes** | no token or raw response in errors; new errors built from fixed strings + `domain.Clean(message)` like Phase 1; no `fmt.Println` while the TUI runs |
| V12 Files/resources | no | — |
| V13 API | partial | respect rate limits (cost accounted; chunk and PR caps) |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Terminal escape / OSC injection through a check name or login (attacker controls workflow job names in a PR) | Tampering / Spoofing | `domain.Clean` at the seam (strips ANSI, C0/C1, bidi overrides); go-gh also defangs ESC (Phase 1 finding); hostile fixture test |
| Bidi override or look-alike to fake a "passing" check | Spoofing | `Clean` drops U+202A-202E and U+2066-2069 (clean.go); status words come from enums, not from remote text |
| Huge names / many contexts to blow up rendering | DoS | truncate to cell width; contexts capped at 100 per page and 5 pages; reviewers capped at 20/50 |
| Opening an attacker-controlled URL from the detail view | Tampering | Only the PR URL is opened, via `safeURL`; `detailsUrl`/`targetUrl` are not fetched |
| Rate-limit exhaustion from extra queries | DoS | about 4 points per tick; step-2 chunking and 100-PR cap; Phase 2 backoff unchanged |

## Project Constraints (from CLAUDE.md)

From `/Users/scottb/dev/bartram/prv/.claude/CLAUDE.md` and the user's global `~/.claude/CLAUDE.md`:
- Stack is Go + Bubble Tea v2, Bubbles v2, Lip Gloss v2 (`charm.land/*/v2` imports only); go-gh `api.GraphQLClient.DoWithContext` with raw query strings and typed structs; GraphQL only (no REST); no go-gh `EnableCache`; `LogIgnoreEnv: true`.
- v2 APIs: `View()` returns `tea.View`; keys are `tea.KeyPressMsg`; space is `"space"`; Bubbles v2 uses `SetWidth()/Width()` methods; `lipgloss.Color` is a function; no v1 import paths.
- Use `lipgloss/table` with a `StyleFunc` and an own cursor for the list (not `bubbles/table`); `viewport` for scrolling the detail view.
- No `time.Sleep`/goroutine pollers: ticks are `tea.Tick` commands; no I/O in Update/View; never print to stdout/stderr while the TUI runs; never write the token anywhere.
- Validate any new query shape with `gh api graphql` and read `rateLimit.cost` before recommending it (done; costs recorded in F9).
- Rate limits: auto-refresh must stay well inside GitHub limits.
- Global: prefer the serena plugin for code work (not available to this agent run; plain reads used); Gradle/Maven rule does not apply; feature-branch work for non-trivial changes (current branch `gsd/phase-01-status-at-a-glance` is the GSD phase branch); do not commit/push unless asked by the workflow.
- GSD workflow enforcement: file changes go through GSD commands.

## Sources

### Primary (HIGH confidence)
- Live GitHub GraphQL API via `gh api graphql` (gh 2.102.0), probes run 2026-10-12: introspection of `StatusCheckRollupContext`, `CheckRun`, `StatusContext`, `CheckConclusionState`, `CheckStatusState`, `CheckRunState`, `StatusState`, `MergeStateStatus`, `MergeableState`, `RequestedReviewer`, `StatusCheckRollupContextConnection`, `CheckRunFilter`, `WorkflowRun`, `Team`, `EnterpriseTeam`, `Mannequin`; cost/payload measurements; the dedupe, UNKNOWN, draft/merged, team-null, re-request and DISMISSED probes in cli/cli, microsoft/vscode, home-assistant/core, golang/go, charmbracelet/bubbletea, rust-lang/rust, facebook/react.
- `/Users/scottb/dev/bartram/prv/.planning/phases/01-status-at-a-glance/01-SPIKE.md` (Q1, Q2, Q3, Q6, Q7, Q9, Q10, extra findings 3, 6, 7, 8).
- Repo source read this session: `internal/github/client.go`, `internal/domain/{pr,status,list,clean}.go`, `internal/ui/{model,view,layout,keys,refresh,help,styles}.go`, `internal/ui/view_test.go`, `internal/github/client_test.go`, `cmd/prv/main.go`.
- `charm.land/bubbles/v2@v2.2.1/viewport/{viewport,keymap}.go` in the Go module cache (API: `SetContentLines`, `ScrollDown/Up`, `SetYOffset`, `View` returns `""` for zero size; default `KeyMap` bindings).

### Secondary (MEDIUM confidence)
- Python `unicodedata.east_asian_width` for glyph width classes.

### Tertiary (LOW confidence)
- None used.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH, no new dependencies; versions read from go.mod.
- API shapes and rules (F1-F9): HIGH, each probed live; sample sizes stated.
- Architecture (two-step fetch inside `Fetch`, optional `DetailFetcher`): HIGH, follows existing seams; the D-11 reading is flagged.
- Layout thresholds and caps: MEDIUM, need goldens to settle.
- Team rendering for org members: LOW-MEDIUM (A1).

**Research date:** 2026-10-12
**Valid until:** 2026-11-11 (GraphQL schema is stable; re-check `mergeStateStatus`/`reviewRequests` shapes if GitHub announces changes)

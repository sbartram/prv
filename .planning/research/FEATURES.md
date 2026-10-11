# Feature Research

**Domain:** Read-only GitHub PR monitoring TUI (curated watch list, review decision + CI rollup, auto-refresh)
**Researched:** 2026-10-11
**Confidence:** MEDIUM (tool feature sets confirmed from READMEs/docs of kiroshi, ghzinga, ghui/phui, `gh pr status`, gh-dash landing page; gh-dash column and sidebar details are from memory because its docs pages did not render, so treat those as MEDIUM-LOW)

## Landscape Summary

| Tool | Shape | What it teaches `prv` |
|------|-------|-----------------------|
| `gh pr status` (official CLI) | One-shot, not live. Sections: current branch, created by you, review requested. Per PR: one-line check summary ("3/4 checks failing", "Checks passing") and review state ("Approved", "Review required"). `--json` exposes `reviewDecision`, `statusCheckRollup`. | The baseline: review state and CI summary on one line per PR. `prv` is this, but live and persistent. |
| gh-dash (Go, Bubble Tea) | Full PR/issue manager. Sections defined by GitHub search filters, a preview sidebar, vim keys, YAML config, `refetchIntervalMinutes`, custom keybinding actions, many write actions. | The reference UX. Its breadth (write actions, issues, notifications, diffs) is exactly what `prv` should not copy. |
| kiroshi (Go, Bubble Tea, read-only) | Groups PRs by who must act next. CI states none/pending/success/failure, merge state clear/behind/conflict, unresolved-thread count, detail overlay, `refresh_interval` with an "auto" badge, optional terminal bell, JSON output when not a TTY. | Closest in spirit. Confirms "strictly read-only" is a viable product. Shows that `none` (no checks) must be distinct from passing. |
| ghzinga (Rust, Ratatui) | One PR per tab, checks and status contexts load after core data, 300s default refresh, quiet on successful refresh, persistent tabs across runs, mouse support, `gzg open` to add a PR to a running session. | Confirms progressive loading, quiet refresh, and session persistence as good long-running-TUI UX. |
| ghui/phui (TypeScript) | PR queues, Actions run drill-in (workflow, branch, status, duration, age; jobs and steps; `n`/`p` jump between failures), auto-refresh of active runs only, many write actions. | Best model for the per-check drill-in. Its "refresh only while active" idea maps to cheap polling. |
| prview (Python) | PR review dashboard with at-a-glance pass/fail/running CI status. | Validates the three-state CI badge. |
| GitHub web "Pull requests" page | Tabs: Created, Assigned, Mentioned, Review requests. Open/Closed toggle. | The thing `prv` replaces. Authored + assigned maps to two of its tabs. |

Gap in the field: none of these combine a persistent, hand-curated watch list (paste any PR URL, survives restarts) with authored + assigned auto-inclusion in a read-only monitor. gh-dash sections are search queries, so it cannot pin one arbitrary PR. ghzinga can hold arbitrary PRs but has no authored/assigned dashboard. This combination is `prv`'s niche.

## Feature Landscape

Complexity: LOW (under a day), MEDIUM (1-3 days), HIGH (more than 3 days or notable risk).

### Table Stakes (Users Expect These)

#### Watch list

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Auto-include PRs authored by me | `gh pr status` "Created by you" and every dashboard do this | LOW | Search `author:@me`. Paginate. |
| Auto-include PRs assigned to me | Stated requirement; GitHub's own "Assigned" tab | LOW | Search `assignee:@me`. GitHub search `OR` across qualifiers is not reliable, so run two queries and dedupe by node ID/URL. Prefer this over `involves:@me`, which also returns mentions and commenters. |
| Add PR by URL or `owner/repo#123` | Core differentiator of the product, but also baseline usability once the concept exists | LOW | Parse both forms, plus optional `/files`, `/checks` URL suffixes and trailing fragments. Validate by fetching before persisting. |
| Remove manually added PR | Needed or the list only grows | LOW | Auto-included PRs cannot be removed (they leave when they stop matching). Show a clear message if the user tries. |
| Persist manual PRs across restarts | Stated requirement | LOW | Small file under XDG config/state dir (TOML/JSON). Atomic write. Store `owner/repo#n` only, never fetched data. |
| De-duplicate across sources | A PR both authored and manually added must show once | LOW | Key by node ID or `owner/repo#n`. Track provenance (auto vs manual) so remove is only offered for manual. |

#### Display

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Repo, number, title, author | Minimum to identify a PR | LOW | Truncate title to terminal width. Show repo as `owner/repo` or shortened when unique. |
| State: Open / Merged / Closed | Stated requirement; GitHub exposes `state` plus `merged` | LOW | GraphQL `state` is OPEN/CLOSED/MERGED, so three states come free. |
| Draft indicator | Every PR tool shows it; a draft is never "ready" | LOW | `isDraft`. Draft is not a fourth state; render as a modifier on Open. |
| Review decision badge | `gh pr status` and all dashboards show it | LOW | `reviewDecision` is APPROVED / CHANGES_REQUESTED / REVIEW_REQUIRED or **null** when the repo has no required-review rule. Render null as "no review required" or a neutral dash, not as "Review required". |
| Reviewer names with approve / changes-requested | Stated requirement | MEDIUM | Use `latestReviews` (latest per reviewer) not `reviews`. Also show pending `reviewRequests` as "awaiting". Dismissed and stale reviews need handling. Team reviewers need a label. |
| One CI rollup badge: passing / failing / running | Stated requirement; `gh pr status` shows it | LOW | `statusCheckRollup.state`: SUCCESS, FAILURE, ERROR, PENDING, EXPECTED. Map to three buckets plus "none". |
| Distinct "no checks" state | kiroshi calls this out explicitly: no checks is not passing | LOW | A null rollup must not render green. |
| Color plus glyph (not color alone) | Accessibility, monochrome terminals | LOW | Use symbols such as check, cross, dot, plus color. Offer ASCII fallback (ghzinga has `emoji`/`ascii`). |
| Relative "updated / closed" age | Needed to judge freshness and the closed window | LOW | `updatedAt`, `closedAt`/`mergedAt`. |
| Empty and error states | Long-running TUIs hit these constantly | LOW | "No PRs", "auth failed", "offline", "rate limited until HH:MM". |

#### Detail / drill-in

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Detail view per PR listing every check/job with status | Stated requirement; phui and gh-dash both do it | MEDIUM | `statusCheckRollup.contexts` returns a union of `CheckRun` and `StatusContext` (legacy commit statuses). Both must be rendered. Needs pagination past the first page for big matrix builds. |
| Check name, status, conclusion, duration | Minimum to act on a failure | MEDIUM | Distinguish queued / in_progress / completed with conclusion (success, failure, cancelled, skipped, neutral, timed_out, action_required, stale). Skipped/neutral should not look like failure. |
| Failing checks sorted/grouped first | The reason anyone opens the detail view | LOW | Order: failing, running, queued, passing, skipped. |
| Link to a check's page | Cannot show logs (read-only, huge), so the browser is the escape hatch | LOW | `detailsUrl`/`targetUrl`; open with a key. |
| Reviewer list in detail | List view is space constrained | LOW | Full reviewer breakdown with state per reviewer. |
| Back/escape navigation | Standard TUI behavior | LOW | `esc`/`q` to return. |

#### Filtering

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Filter by Open / Merged / Closed (any combination) | Stated requirement | LOW | Toggle keys `1/2/3` or a single multi-select. Show active filter in the header. Default: Open on, Merged/Closed per config. |
| Recent-window cut-off for Merged/Closed | Stated requirement; avoids history flood | LOW | Apply to auto-included PRs at query time (`closed:>=DATE` / `merged:>=DATE`). Decide whether manual PRs are exempt (recommend: manual PRs always shown until removed). |
| Text filter (`/`) on title/repo/author | Present in kiroshi, phui, gh-dash | LOW | Client-side substring match. Cheap and widely expected once the list exceeds one screen. |

#### Refresh

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Auto-refresh on configurable interval | Stated requirement; core value | MEDIUM | gh-dash `refetchIntervalMinutes`, kiroshi `refresh_interval`, ghzinga 300s. Recommend default 60-120s. Use a Bubble Tea tick that re-arms after the fetch completes (no overlapping fetches). |
| Manual refresh key (`r`) | Every tool has it | LOW | Resets the tick timer. |
| Last-refreshed timestamp and in-flight indicator | User must know the data is not stale | LOW | Header shows "updated 12s ago" and a spinner during fetch. |
| Keep prior data on failed refresh | Network blips are normal for a long-running pane | LOW | Show stale data dimmed with an error banner. Never blank the list. |
| Stay inside rate limits | Stated constraint | MEDIUM | GraphQL budget is 5,000 points/hour. One batched query for N PRs costs about 1 point if connection `first` values are small. Read the `rateLimit` block each response; back off on 403/429 and secondary-limit `Retry-After`. Backoff and a visible "rate limited" state are required, not optional. |
| Preserve selection and scroll across refresh | A list that jumps under the cursor is unusable | MEDIUM | Re-select by PR identity, not row index. Common source of bugs in Bubble Tea list refreshes. |

#### Navigation / actions

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| j/k and arrow navigation, g/G | Universal in this space | LOW | |
| Enter opens detail | Stated requirement | LOW | |
| Open in browser (`o`) | Stated requirement; kiroshi `enter/o` | LOW | Use a cross-platform opener (macOS `open`, Linux `xdg-open`). |
| `?` help overlay | Every tool in the survey has it | LOW | Bubble Tea `bubbles/help` gives most of it. |
| `q` / Ctrl-C quit | | LOW | |
| Handles terminal resize | Used in multiplexer splits (stated intended use) | LOW-MEDIUM | Narrow panes are the primary environment, so the layout must degrade by dropping columns rather than wrapping. |

#### Auth / config

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Token from `gh auth token`, fallback to `GITHUB_TOKEN` | Stated requirement; ghzinga does the same | LOW | Decide precedence explicitly. ghzinga lets `GH_TOKEN`/`GITHUB_TOKEN` override `gh`; the project says `gh` first, env as fallback. Document it. |
| Clear error when no token or token invalid | First-run experience | LOW | Say exactly what to run (`gh auth login`). |
| Config file with refresh interval, closed window, default filters | Stated configurability | LOW | TOML/YAML in XDG config dir. Sensible defaults so zero config works. |
| Read-only token scope guidance | kiroshi recommends fine-grained read-only PAT | LOW | Document. Note fine-grained PATs can silently miss repos not granted, and orgs with SAML need token authorization. Surface a hint when expected PRs are missing. |

### Differentiators (Competitive Advantage)

Not required by the stated requirements; each one is optional and ordered by value for `prv`'s core value ("approved, blocked, or failing at a glance").

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Mixed authored / assigned / manual provenance marker | Shows why a PR is in the list; nobody else has a curated list | LOW | Small glyph per row. Falls out of dedupe logic for free. |
| "Needs attention" sort (failing / changes-requested first, then running, then approved) | Delivers the core value without scanning | LOW | Default sort. Cheap and high value. Alternative sorts: updated, repo. |
| Merge-readiness indicator (`mergeStateStatus`: clean, behind, dirty/conflict, blocked) | "Approved and green but has a conflict" is the classic missed state; `gh pr status -c` and kiroshi expose it | LOW | One extra GraphQL field. Highest-value addition after the stated set. Render as a small badge. |
| Change detection highlight (row flashes/marks when state changed since last refresh) | Replaces notifications for a pane the user glances at; no popups needed | MEDIUM | Diff previous vs new snapshot per PR; mark with "changed" glyph until viewed. Needs previous-snapshot state in the model. |
| Unresolved review thread count | kiroshi shows it; signals "approved but comments open" | LOW | `reviewThreads(first: N){ isResolved }`. Adds query cost; see rate limits. Some orgs restrict it. |
| Failing check names inline in list row (e.g. "failing: lint, e2e") | Avoids the drill-in for the common question | LOW-MEDIUM | Limit to first 1-2 names with `+N`. |
| Check run duration and live elapsed time for running checks | Tells "stuck" from "just started" | LOW | `startedAt`/`completedAt`. |
| Jump to next failing check in detail view | phui's `n`/`p` | LOW | Small but nice for big matrices. |
| Copy PR URL (`y`) | kiroshi, ghzinga | LOW | OSC 52 or clipboard lib. OSC 52 works over SSH/tmux. |
| Adaptive refresh (faster while checks are running, slower when all settled) | Cheaper on rate limit and fresher where it matters; phui refreshes only active runs | MEDIUM | Single global tick with a shorter interval when any watched PR has pending checks. Do not build per-PR timers. |
| Group-by repo or by state with collapsible headers | Large lists become scannable | MEDIUM | Defer unless lists commonly exceed one screen. |
| Rate-limit remaining shown in footer | Builds trust for a long-running session; GraphQL gives `remaining`/`resetAt` | LOW | Free from the response. |
| Opt-in terminal bell on transition to failing/approved | kiroshi `notify` is only a bell, cheap | LOW | Conflicts with the Out of Scope "no notifications" decision only if default-on. Keep off by default or defer. |
| Non-TTY JSON output (`prv --json`) | kiroshi does it; useful for scripts and status bars | MEDIUM | Not requested. Keep out of v1. |
| Add PR from clipboard (`a` then paste) | Pasting a URL is the add flow | LOW | A text input prompt that accepts paste is enough. Bracketed paste must work. |
| Startup argument `prv owner/repo#123` | Add from shell; ghzinga/phui support open-by-ref | LOW | Cheap once the parser exists. |
| Persisted UI state (filters, sort) | ghzinga persists sessions | LOW | Store last filter/sort alongside config. |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Write actions (merge, approve, comment, re-run checks, close) | gh-dash and phui have them | Needs write-scoped token, confirmation UX, error handling; breaks the read-only security story (kiroshi sells "strictly read-only" with a read-only PAT); out of scope in PROJECT.md | `o` opens GitHub. |
| In-TUI diff / file viewer, inline comments | gh-dash, phui, ghzinga have them | Entire second product; large rendering and syntax-highlight work | Open in browser. |
| CI log viewing / streaming | "Why did it fail?" | Logs are large and need a separate REST endpoint (zip archives for Actions); heavy and needs extra scopes | Show failing step name; open check URL in browser. |
| Review-requested-from-me section | In `gh pr status`, gh-dash, GitHub tabs | Explicitly excluded by the user. Different workflow (inbox, not watch) | Manually add a PR by URL when needed. |
| Desktop / push notifications | Natural for a monitor | Needs daemon or OS integration, permissions, dedupe; out of scope (user leaves pane open) | Change-highlight in the row; optional bell later. |
| GitHub notifications inbox | gh-dash and phui have it | Separate API (REST notifications), separate mental model | Out of scope. |
| Issues, discussions, stars, repo browsing | gh-dash, phui broaden into these | Scope creep from "PR watch" | None. |
| Arbitrary search-query sections (gh-dash style) | Powerful, flexible | Turns `prv` into gh-dash; heavy config; the curated list is the point | Authored + assigned + manual only. |
| Full closed/merged history, pagination of old PRs | Completeness | Years of noise; rate cost | Recent window only. |
| GitHub Enterprise / multi-host | Work accounts | Auth host resolution, GHES API version drift; out of scope | Defer to a later milestone. |
| Webhooks / real-time server push | "Instant" updates | Needs a public endpoint or relay; wrong for a local TUI | Polling with adaptive interval. |
| Custom keybinding shell actions, templating | gh-dash power feature | Config surface and security surface for little value in a read-only tool | Fixed, documented keymap. |
| Built-in OAuth / device-flow login | Self-contained | Second auth path to maintain; user already has `gh` | `gh auth token` plus `GITHUB_TOKEN`. |
| Local clone / worktree / checkout integration | gh-dash `c` checkout, phui worktrees | Requires repo-path mapping config | Out of scope. |
| Theme engine, mouse support | Polish | Large surface for a first version | Use the terminal's ANSI palette (inherit colors); revisit later. |
| Per-PR custom refresh timers | Granularity | Complexity and rate cost | One global interval, optionally adaptive. |

## Feature Dependencies

```
Auth (gh token / GITHUB_TOKEN)
    └──required by──> every fetch

PR reference parser (URL | owner/repo#n)
    ├──required by──> Add manual PR
    │                     └──requires──> Persistence file
    │                                        └──enables──> Remove manual PR
    └──enhances──> CLI arg `prv owner/repo#n`

Auto-include query (authored + assigned)
    └──requires──> Dedupe + provenance
                       └──requires──> Unified PR model (state, review, CI)

Unified PR model (batched GraphQL fetch)
    ├──enables──> List view (state, review badge, CI badge)
    │                 ├──enables──> State filter ──requires──> Recent-window rule
    │                 ├──enables──> Text filter
    │                 └──enables──> "Needs attention" sort
    └──enables──> Detail view (per-check list)
                      └──requires──> Pagination of contexts (CheckRun + StatusContext union)

Auto-refresh tick
    ├──requires──> Unified PR model fetch
    ├──requires──> Selection preserved by PR identity
    ├──requires──> Rate-limit awareness + backoff
    └──enhances──> Change-detection highlight (needs previous snapshot)

Adaptive refresh ──enhances──> Auto-refresh tick
Merge-readiness badge ──enhances──> Review badge + CI badge (shares the same query)
Unresolved-thread count ──conflicts (cost)──> tight rate budget with many PRs
Opt-in bell ──conflicts──> "No notifications" scope (only if default-on)
```

### Dependency Notes

- **Everything depends on the batched GraphQL fetch and the unified PR model.** List, detail, filters, refresh, and change detection all read the same snapshot. Build that first.
- **Manual add requires the parser and persistence, but not the auto-include query.** The manual-only path is a fine early vertical slice.
- **State filter requires the recent-window rule** because merged/closed PRs are only in the data set when within the window. Decide the window semantics (applied to auto-included only, or manual too) before building the filter.
- **Selection preservation must exist before auto-refresh ships.** Otherwise the first refresh makes the TUI feel broken.
- **Detail view requires context pagination.** The list rollup is one field; the per-check view needs a second connection, which is also where query cost can grow. Fetch detail lazily for the selected PR (or a bounded first page for all) to protect the rate budget.
- **Change detection needs two snapshots** in memory; it is cheap to add if the model holds the previous snapshot from the start.

## MVP Definition

### Launch With (v1)

All of the stated Active requirements, plus the few low-cost items that prevent known failure modes.

- [ ] Auth via `gh auth token` / `GITHUB_TOKEN`, with a clear failure message
- [ ] Authored + assigned PRs auto-included, deduped, provenance tracked
- [ ] Add by URL / `owner/repo#n`; persist; remove manual PRs
- [ ] List with repo, number, title, author, draft flag, state (Open/Merged/Closed)
- [ ] Review decision badge (null handled) with approving / changes-requesting reviewer names
- [ ] One CI badge: passing / failing / running / none
- [ ] State filter (any combination) and recent-window config for merged/closed
- [ ] Detail view with all checks (CheckRun and StatusContext), failing first, with status and link
- [ ] Auto-refresh on configurable interval, manual `r`, last-updated indicator, keep data on error
- [ ] Rate-limit handling with visible backoff state
- [ ] Selection preserved across refresh
- [ ] Open in browser, help overlay, quit, resize handling
- [ ] Config file with defaults

### Add After Validation (v1.x)

- [ ] Merge-readiness badge (conflict / behind / blocked) — first thing users will ask after "approved and green but won't merge"
- [ ] "Needs attention" default sort — once lists exceed a screen
- [ ] Change-detection highlight — once the user reports missing transitions while looking away
- [ ] Text filter `/` — once lists exceed a screen
- [ ] Failing check names inline in list row
- [ ] Adaptive refresh while checks are running
- [ ] Copy URL (`y`), `prv owner/repo#n` startup arg
- [ ] Rate-limit remaining in footer

### Future Consideration (v2+)

- [ ] Unresolved review-thread count (query cost and org restrictions)
- [ ] Opt-in terminal bell on transitions
- [ ] Group-by repo, collapsible sections
- [ ] `--json` non-TTY output
- [ ] GitHub Enterprise hosts
- [ ] Mouse support, themes

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Batched fetch + unified PR model | HIGH | MEDIUM | P1 |
| Authored + assigned auto-include | HIGH | LOW | P1 |
| Add / remove / persist manual PRs | HIGH | LOW | P1 |
| State badge + filter + recent window | HIGH | LOW | P1 |
| Review decision + reviewers | HIGH | MEDIUM | P1 |
| CI rollup badge (with "none") | HIGH | LOW | P1 |
| Per-check detail view | HIGH | MEDIUM | P1 |
| Auto-refresh + rate-limit backoff | HIGH | MEDIUM | P1 |
| Selection preserved on refresh | HIGH | MEDIUM | P1 |
| Open in browser | HIGH | LOW | P1 |
| Merge-readiness badge | HIGH | LOW | P2 |
| Needs-attention sort | MEDIUM | LOW | P2 |
| Change-detection highlight | MEDIUM | MEDIUM | P2 |
| Text filter | MEDIUM | LOW | P2 |
| Inline failing check names | MEDIUM | LOW | P2 |
| Adaptive refresh | MEDIUM | MEDIUM | P2 |
| Thread counts | LOW | LOW | P3 |
| Bell, JSON output, grouping | LOW | LOW-MEDIUM | P3 |

**Priority key:** P1 = must have for launch; P2 = add when possible; P3 = future.

## Competitor Feature Analysis

| Feature | `gh pr status` | gh-dash | kiroshi | ghzinga | phui | Our Approach |
|---------|---------------|---------|---------|---------|------|--------------|
| Live TUI | No | Yes | Yes | Yes | Yes | Yes |
| Arbitrary pinned PRs | No | No (search sections) | No | Yes (tabs) | Open by ref | Yes, persisted |
| Authored / assigned auto-list | Authored only | Via filters | Yes | No | Yes | Authored + assigned |
| Review decision | Yes | Yes | approval count | Yes | Yes | Decision + named reviewers |
| CI rollup | Text summary | Icon | none/pending/success/failure | Yes | Yes | passing/failing/running/none |
| Per-check drill-in | `gh pr checks` | Sidebar | Detail overlay | Checks view | Jobs and steps | Detail view, failing first |
| Auto-refresh | No | Interval | Optional interval | 300s default | Active runs only | Configurable, adaptive later |
| Merge state | `-c` flag | Yes | clear/behind/conflict | Yes | Yes | v1.x |
| Write actions | No | Yes | No | No (view) | Yes | No |
| Notifications | No | Yes (inbox) | Bell | No | Inbox | None in v1 |

## Sources

- gh-dash README (https://github.com/dlvhdr/gh-dash) — features list; MEDIUM. Docs subpages (columns, sidebar tabs, search) did not load; details on those are unverified.
- gh-dash config via search results (README mirrors): `refetchIntervalMinutes`, `prSections` filters, custom keybindings, pager — MEDIUM.
- kiroshi README (https://github.com/ajardin/kiroshi) — buckets, CI/merge states, refresh, bell, JSON output, read-only stance — HIGH (fetched directly).
- ghzinga 0.2.2 (https://docs.rs/crate/ghzinga/0.2.2) — refresh, checks loading, sessions, quiet refresh — HIGH (fetched directly).
- ghui/phui (https://github.com/phall1/ghui) — Actions drill-in, failure jump keys, active-only refresh — HIGH (fetched directly).
- `gh pr status` manual (https://cli.github.com/manual/gh_pr_status) — sections, check summary, review status, `-c` conflict flag, JSON fields `reviewDecision`/`statusCheckRollup` — HIGH.
- GitHub GraphQL rate limits (https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api) — 5,000 points/hour, point calculation, secondary limits, `rateLimit` fields — HIGH.
- GraphQL schema facts (`reviewDecision` nullable, `latestReviews`, `statusCheckRollup.contexts` as CheckRun | StatusContext union, `mergeStateStatus`, `isDraft`) — from prior knowledge of the GitHub GraphQL schema; MEDIUM, verify against the schema during the data-layer phase.
- GitHub search `OR` behavior across qualifiers — not verified; MEDIUM-LOW. Plan on two queries plus dedupe.

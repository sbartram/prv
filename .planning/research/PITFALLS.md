# Pitfalls Research

**Domain:** Long-running GitHub pull-request monitoring TUI (Go + Bubble Tea, GraphQL, github.com only, `gh` token reuse)
**Researched:** 2026-10-11
**Confidence:** MEDIUM-HIGH (rate-limit numbers, schema enums, search cap, Bubble Tea APIs verified against official docs/schema; a few behaviours are marked "verify with fixture" because GitHub does not document them)

## How to read this file

Confidence tags: **[V]** verified this session against the official GraphQL schema/docs or library docs. **[K]** known GitHub behaviour from community experience, not stated in official docs (verify with a recorded fixture early). **[L]** single-source or inferred; treat as a hypothesis.

Phase names below are *suggested* groupings, since no roadmap exists yet:

- **P1 Foundation:** auth, GitHub client, domain model, status-mapping rules
- **P2 Data layer:** discovery + hydration, watch list persistence, per-item error model
- **P3 TUI shell:** list view, refresh loop, resize, selection
- **P4 Detail view:** per-check/job drill-down, open-in-browser
- **P5 Hardening:** rate-limit budget UI, backoff, soak testing, token edge cases

---

## Critical Pitfalls

### Pitfall 1: Treating a null `reviewDecision` as "no review status"

**What goes wrong:**
`PullRequest.reviewDecision` is nullable and is `null` when the base branch has no branch protection / ruleset that *requires* reviews. [V: schema `reviewDecision: PullRequestReviewDecision`, no `!`; null-when-no-requirement is K]. In that case a PR with two real approvals shows no decision, so `prv` displays a blank or "review required" badge for exactly the repos (personal projects, small teams) where the user has no protection rules. The enum only has three values: `APPROVED`, `CHANGES_REQUESTED`, `REVIEW_REQUIRED` [V]. There is no "none" and no "commented".

**Why it happens:**
The field name sounds like "the overall review state", so devs render it directly. It is actually "the state of the review *requirement*".

**How to avoid:**
Model the review state as a derived value with an explicit fallback chain:
1. If `reviewDecision != null`, use it.
2. Else derive from per-reviewer latest opinionated reviews: any `CHANGES_REQUESTED` -> changes requested; else any `APPROVED` -> approved (label it "approved (no requirement)" or a subtler glyph); else "no reviews".
3. Keep a distinct `NoReviewsRequired`/`None` state; never map null to `REVIEW_REQUIRED`.
Unit-test the derivation with fixtures for: null+approved, null+none, null+changes-requested, `REVIEW_REQUIRED` with an approval from a non-writer.

**Warning signs:**
Own personal repos always show a blank review badge. A PR with visible green "Approved" on github.com shows nothing in `prv`. Tests only contain fixtures where `reviewDecision` is populated.

**Phase to address:** P1 (domain model + mapping), verified in P2 with a real null-decision PR.

---

### Pitfall 2: Using `latestReviews` / `reviews` for the approver list without understanding review states

**What goes wrong:**
The requirement "which reviewers approved or requested changes" is wrong in four ways:
- `reviews(first: N)` is chronological. On a PR with many review rounds, the *latest* reviews fall off the first page. Aggregating from it gives stale approvals. [V: connection semantics]
- `latestReviews` is "latest per user, not also pending" [V]. Its states include `COMMENTED`, `DISMISSED`, `APPROVED`, `CHANGES_REQUESTED` [V enum]. If an approver later leaves a plain comment, `latestReviews` can report that reviewer as `COMMENTED`, hiding their approval. `latestOpinionatedReviews` is the per-user-latest list intended to ignore informational reviews [schema text is terse; "COMMENTED does not override" is K -- verify with a fixture].
- `DISMISSED` reviews are not approvals; counting any non-`CHANGES_REQUESTED` review as an approval inflates the count.
- Approvals from users without write access exist on the PR but do not satisfy "required reviews". `latestOpinionatedReviews(writersOnly: true)` filters to writers [V arg exists].
- Re-requested reviews: after a re-request, the old `APPROVED` is still the latest review, yet the reviewer is again in `reviewRequests`. Showing a green check for them is misleading.
- Community reports `latestReviews` can lag a freshly submitted review by minutes [L: single non-official source]. Do not build "act on a review within seconds" assumptions on it.

**Why it happens:**
The names suggest `latestReviews` = "current verdicts". The `COMMENTED` and `DISMISSED` states are easy to overlook in a switch statement.

**How to avoid:**
- Fetch `latestOpinionatedReviews(first: 20)` for the approver list and `reviewRequests(first: 20)` for pending reviewers. Never aggregate from `reviews(first: N)`.
- Render per reviewer: `APPROVED`, `CHANGES_REQUESTED`, `DISMISSED` (distinct, dimmed), pending-re-request overlay.
- `reviewRequests.requestedReviewer` is a union (`User | Team | Mannequin | Bot`): use inline fragments (`... on User { login }`, `... on Team { name }`) or team requests decode as empty strings.
- `author` / reviewer `Actor` can be a deleted "ghost" user or a `Bot` (e.g. Copilot review); do not dereference without a nil guard.
- Cap and show overflow ("+3 more") rather than paginating reviews in the list query.

**Warning signs:**
Approver names disappear after someone comments. A dismissed review shows green. A Team reviewer renders as an empty name. Approver count differs from github.com's "N approving reviews".

**Phase to address:** P1 (query + mapping), P4 (detail view reviewer list).

---

### Pitfall 3: Collapsing `statusCheckRollup` into a boolean and mishandling CheckRun vs StatusContext

**What goes wrong:**
`statusCheckRollup.contexts` is a union of `CheckRun | StatusContext` [V]. They use *different vocabularies*:
- `CheckRun.status` (`QUEUED, IN_PROGRESS, COMPLETED, WAITING, PENDING, REQUESTED`) is separate from `CheckRun.conclusion` (`SUCCESS, FAILURE, NEUTRAL, SKIPPED, CANCELLED, TIMED_OUT, ACTION_REQUIRED, STALE, STARTUP_FAILURE`), and `conclusion` is `null` until `status == COMPLETED` [V enums, nullable].
- `StatusContext.state` is `EXPECTED, ERROR, FAILURE, PENDING, SUCCESS` [V]. `EXPECTED` means a required status has not reported yet (treat as pending). `ERROR` and `FAILURE` are both "failing".
- The rollup's own `state` is only `EXPECTED | ERROR | FAILURE | PENDING | SUCCESS` [V]: it has no "skipped", "neutral", "queued" or "cancelled". It is GitHub's computed summary: it includes *non-required* checks (a non-required failing check makes the rollup `FAILURE` while the PR is still mergeable) [K], and `NEUTRAL`/`SKIPPED` do not make it fail [K].
- `statusCheckRollup` itself is **nullable**: a PR with no checks at all returns null [V nullable]. Showing "passing" for null is a lie; "no checks" is its own state.
- A QUEUED run is not "running"; WAITING usually means an environment-approval gate that needs a human, which is "blocked", not "running".
- `CANCELLED` is frequently produced by concurrency groups when a newer push supersedes a run. Painting it red creates permanent false alarms on active PRs; it should be a distinct, de-emphasized state unless it is the only/latest outcome.
- Re-runs: after "re-run failed jobs" the new CheckRun supersedes the old, but a naive client that reads from `checkSuite.checkRuns` or merges history can show the old failure beside the new success [K; the rollup normally exposes the latest only: verify by re-running a job against a real PR]. Defensively dedupe by (app, name) keeping latest `startedAt`.
- `contexts(first: 100)` is the maximum page. Matrix-heavy repos exceed 100 contexts, so counting from the first page under-reports failures (a failure on page 2 is invisible and the badge says green). The connection exposes `checkRunCount`, `checkRunCountsByState`, `statusContextCount`, `statusContextCountsByState`, `totalCount` [V].

**Why it happens:**
One "workflow badge" requirement invites `rollup.state == SUCCESS ? green : red`. The semantic gap between "GitHub's rollup" and "what the user cares about (will this merge?)" is invisible until a non-required lint check turns everything red.

**How to avoid:**
- Define one internal `CheckState` enum: `NoChecks, Passing, Failing, Running, Queued, Blocked(WaitingApproval/ActionRequired), Cancelled, Skipped/Neutral`. Write a single total mapping function with exhaustive tests over every enum value in both vocabularies, plus a `default` that maps unknown future values to `Unknown` (never panics, never "passing").
- List view: use the rollup `state` plus `checkRunCountsByState`/`statusContextCountsByState` (cheap, no node paging) to compute the badge. Do **not** page contexts for every PR on every poll.
- Detail view: fetch `contexts` lazily for the *selected* PR only, paginate with `pageInfo` until exhausted (or cap at e.g. 300 with a "+N more" line).
- Decide explicitly whether failing *non-required* checks turn the badge red; the `isRequired(pullRequestId: ...)` field exists on both `CheckRun` and `StatusContext` [V] but adds query weight. Recommended: red only for required failures when cheap to know, otherwise show red with a "non-required" marker in the detail view.
- Precedence for the badge: Failing > Blocked > Running > Queued > Passing; "Cancelled/Skipped/Neutral" never override Passing.

**Warning signs:**
Badge says "passing" on a PR with no CI. A PR flips to green while a job on page 2 failed. Skipped jobs shown as failures. Detail view lists the same job twice after a re-run. A `switch` over `conclusion` without an unknown case.

**Phase to address:** P1 (mapping function + tests), P2 (count-based list query), P4 (lazy detail + pagination + re-run dedupe).

---

### Pitfall 4: Batched GraphQL treated as all-or-nothing (per-item errors discarded or wipe the list)

**What goes wrong:**
Hydrating many watched PRs in one aliased query (`pr0: repository(owner:"o", name:"r"){ pullRequest(number: 1){...} }`) means one inaccessible item yields `data.pr7 == null` plus an entry in top-level `errors[]` whose `path` starts with `"pr7"`; HTTP status is still 200 [K, standard GraphQL partial-success]. Typical failures:
1. The client library returns a Go `error` and the caller discards the whole response, so one SSO-blocked or deleted PR blanks the entire list (or poisons every refresh forever).
2. The error is swallowed and the item silently vanishes, or is auto-removed from the persisted manual watch list (data loss from a transient glitch).
3. All failures are lumped as "error", although they need different user actions:
   - `NOT_FOUND` ("Could not resolve to a Repository/PullRequest...") covers deleted PRs, deleted/renamed/transferred repos, *and* private repos the token cannot see (GitHub deliberately hides existence) [K].
   - `FORBIDDEN` with SAML text ("Resource protected by organization SAML enforcement...", `extensions.saml_failure`) means the token needs SSO authorization for that org [K].
   - `RATE_LIMITED` (can arrive as HTTP 200 + errors in GraphQL) [K].
   - Timeouts / 502 on oversized queries: GitHub aborts expensive GraphQL queries [K].
4. Go typing trap: a non-null struct field decoded from `null` is silently the zero value, so the failed PR becomes a ghost row ("#0", empty title, state "").

**Why it happens:**
Typed GraphQL clients (`shurcooL/graphql`, `githubv4`) and `go-gh` return `error` when `errors[]` is non-empty. Whether partial `data` is still decoded depends on the library [K: shurcooL decodes `data` before returning the error; verify the chosen client with a fixture]. Also, struct-based clients cannot express dynamic aliases, so people fall back to one-query-per-PR (rate-limit and latency problems, see Pitfall 5).

**How to avoid:**
- Use a client that accepts a raw query string and returns raw `data` + `errors` (e.g. go-gh `GraphQLClient.Do`, or a thin `net/http` wrapper), and build alias queries programmatically. Decode each alias into `*PullRequestNode` (pointer: nil = failed).
- Build an explicit per-item result type: `ItemResult{ID, Data *PR, Err *ItemError}` where `ItemError.Kind` is one of `NotFoundOrNoAccess | SSORequired{URL} | RateLimited | Transient | Auth`. Map `errors[].path[0]` back to the alias (keep an alias -> watch-entry table per request).
- On a per-item error keep the *last good* data and mark the row stale/errored ("!" glyph + reason in status bar/detail). Never remove a manual entry automatically; removal is an explicit user action.
- Prefer `nodes(ids: [...])` (max 100 IDs per call [K]) for hydration once a PR's node ID is known: it survives repo rename/transfer, and a deleted PR becomes a `null` element in the returned list (elements are nullable: `[Node]!` [V]). Resolve `owner/repo#N` -> node ID once at add time.
- Chunk batches (25-50 PRs per request) so one slow/expensive query cannot time out everything; on a chunk-level failure, bisect/retry smaller before declaring items broken.
- Fixture tests for: one null alias + NOT_FOUND error; SAML FORBIDDEN; mixed success; whole-request 502; `RATE_LIMITED`.

**Warning signs:**
List goes empty when one SSO org PR is watched. A "#0" or untitled row appears. Manual PRs disappear after a flaky refresh. Error text is a generic "query failed".

**Phase to address:** P2 (core of the data layer; must exist before the UI consumes data), regression-tested in P5.

---

### Pitfall 5: Blowing the rate limit with long-running auto-refresh (and misunderstanding what the limits are)

**What goes wrong:**
A TUI left open for days polls thousands of times. Facts [V unless noted]:
- Primary GraphQL limit: **5,000 points/hour** per user token. Cost = (sum of requests per unique connection, assuming each hits its `first`/`last`) / 100, rounded, minimum 1. Read the actual `cost` and `remaining` from `rateLimit { cost remaining resetAt }`. Docs recommend response headers where possible.
- Secondary limits: max **100 concurrent requests** (shared REST+GraphQL), **2,000 points/min for the GraphQL endpoint** (a non-mutation GraphQL request = 1 point), CPU-time limits, and undocumented abuse heuristics; "limits can change without notice". There is no API to check secondary-limit status. Headers: honor `retry-after`; else if `x-ratelimit-remaining: 0` wait until `x-ratelimit-reset`; else wait >= 1 minute, then exponential backoff. Continuing while limited "can get your integration banned".
- Per-node limit: 500,000 nodes per call; every connection needs `first`/`last` of 1-100.
- **The budget is per token, shared with everything else the user runs**: `gh`, `gh-dash`, editors, scripts, and other `prv` instances (multiplexer panes!). `prv` cannot assume it owns 5,000 points.
- Search is separate: REST search has its own bucket (30 req/min authenticated [V from docs search]); GraphQL `search` instead costs primary points *and* is subject to the secondary limits [K]. Do not assume "search is free because it is a different limit".

Failure modes in practice (steady-state polling of ~20 PRs at 60s is actually cheap, ~100-300 points/hr):
1. One query per PR per tick (N+1): 50 PRs at 30s = 6,000 requests/hr.
2. "Refresh now" key held down or mashed, or a refresh storm at startup (discovery + hydration + detail + retries at once).
3. No in-flight guard: a slow request plus a fixed ticker stacks overlapping refreshes.
4. Retry loops with no backoff on 403/429/5xx, turning a secondary-limit block into a longer one.
5. User sets `interval = 1s` in config.
6. Fetching heavy nested connections (`contexts(first:100)`, `reviews(first:100)`) for every PR every tick; node count multiplies (50 PRs x 100 contexts).
7. Ignoring `rateLimit.resetAt`: after exhaustion every call fails until reset with a cryptic error.

**Why it happens:**
"GraphQL = one cheap request" is assumed without computing cost; rate limits only bite after hours, so they are never hit in dev.

**How to avoid:**
- Budget design: two-tier polling. *Discovery* (search for authored/assigned, Pitfall 6) every 3-5 min; *hydration* (`nodes(ids)` or aliased PR lookups, count-only check data) every 30-60 s. Clamp the configured interval to a floor (suggest >= 15-30 s) and warn instead of obeying.
- Single-flight: at most one refresh in flight; manual refresh while in flight is coalesced; manual refresh also has a minimum spacing (e.g. 2-5 s).
- Serialize requests (concurrency 1-2). Never fan out goroutines per PR.
- Include `rateLimit { cost remaining resetAt }` in every query; keep the latest values in the model; if `remaining` < ~10% of limit, stretch the interval automatically; if 0, pause until `resetAt` and say so.
- Implement the documented backoff (retry-after -> reset -> 60 s -> exponential, bounded attempts); a retry never re-fires immediately.
- Lazy detail data (Pitfall 3). Reduce per-PR nested page sizes to what is displayed (`latestOpinionatedReviews(first: 10)`).
- Status bar shows "API 4,812/5,000, resets 14:32" and "rate limited until 14:32" so the state is observable rather than inferred.
- Consider pausing refresh when the terminal reports no focus is *not* recommended (user watches the pane passively); instead consider slower interval after N minutes of no input (optional, configurable).
- Soak test: run 2+ hours against a mock server that counts points and injects 403/retry-after.

**Warning signs:**
`remaining` trends down faster than expected in logs. 403/429 appear in logs. Two panes running `prv` double the burn. Gaps between refreshes grow without explanation.

**Phase to address:** P1 (client with cost/header capture, backoff), P2 (two-tier polling and batch sizing), P5 (budget UI, soak test).

---

### Pitfall 6: Relying on GitHub search for discovery without handling its caps, lag, and silent omissions

**What goes wrong:**
Watch list = "authored or assigned" needs a query like `is:pr author:@me` and `is:pr assignee:@me`. Issues with search [V unless noted]:
- Hard cap: GraphQL `search` returns **a maximum of 1,000 results** regardless of `issueCount`. A user with hundreds of stale/bot PRs across orgs silently loses the tail; `issueCount` can say 1,400 while only 1,000 are retrievable. Paging past it stops with no error.
- Searches mixing issues and PRs: omit `is:pr` and the `SearchResultItemConnection` returns `Issue` nodes too; with `... on PullRequest {}` fragments the Issue nodes decode as empty objects: ghost rows. Always include `is:pr`; also, `SearchType` offers `ISSUE` and `ISSUE_ADVANCED` [V]; choose one and test quoting/OR syntax with real data.
- Index lag: search is eventually consistent; a PR you opened seconds ago, a just-assigned PR, or a just-merged PR (stale `is:open` membership or stale state) can be absent or wrong for seconds to minutes [K]. A PR present in one poll and absent in the next is common.
- SSO: results from orgs where the token lacks SSO authorization can be silently omitted rather than errored (REST sets an `X-GitHub-SSO` header; GraphQL behaviour is less clear) [L]. The user then sees "no PRs" for their work org and blames the tool.
- `incomplete_results` / timeouts: a search that times out returns partial matches [V for REST docs]; GraphQL surfaces this less clearly.
- `author:@me` / `assignee:@me` resolution is for the token owner; fine for `gh` OAuth tokens. Verify it works in GraphQL `search` strings (otherwise resolve `viewer { login }` first and substitute).
- Closed/merged window: using `closed:>=DATE` / `merged:>=DATE` with local-time dates gives off-by-one windows; use UTC dates and consider timestamps.
- Using search results as *state* (title/state/reviews) is risky because of lag.

**Why it happens:**
Search is the only way to ask "assigned to me" (there is no `viewer.assignedPullRequests`), so it becomes the data source for everything.

**How to avoid:**
- Authored PRs: prefer `viewer.pullRequests(first: 50, states: [...], orderBy: {field: UPDATED_AT, direction: DESC})` which is a direct DB connection with no index lag and no 1,000 cap [K]. Use search only for the *assigned* set.
- Use search/discovery **only to find node IDs**; hydrate authoritative state via `nodes(ids)`/aliased lookups (Pitfall 4). This also decouples the fast poll from the slow, lagging index.
- Add a tombstone/grace rule: a PR that vanishes from discovery is kept for >= 2-3 discovery cycles (or until hydration says it is closed/out of window) before it leaves the list. Newly seen PRs appear immediately.
- Slice by state and window (`is:open`, `is:merged merged:>=`, `is:closed is:unmerged closed:>=`) so each query is well below 1,000; if `issueCount > retrievable`, show a "list truncated" banner.
- Always `is:pr`; always `archived:false` unless archived PRs are wanted; consider excluding bot-assigned noise via config.
- Show a persistent first-run/diagnostic hint: "Not seeing PRs from org X? Authorize your token for SSO: `gh auth refresh -h github.com -s repo,read:org` and approve the org in GitHub settings."
- Manual adds bypass search entirely (resolve directly), so a just-opened PR can always be added by URL.

**Warning signs:**
Rows flicker in/out between polls. A freshly opened PR takes minutes to appear. `issueCount` > number of rows. Work org PRs missing but personal ones present. Empty-object rows.

**Phase to address:** P2 (discovery design) with a dedicated fixture test for each of: Issue-in-results, truncation, vanishing PR.

---

### Pitfall 7: Bubble Tea: blocking work in `Update`/`View`, and unmanaged refresh/tick chains

**What goes wrong:**
- Network or `exec` calls (`gh auth token`, HTTP, browser launch, file writes) directly inside `Update` or `View` freeze rendering and key handling. In Bubble Tea, work belongs in a `tea.Cmd` that returns a `Msg`; the framework runs commands in their own goroutines [V].
- `tea.Tick`/`tea.Every` send a **single** message; looping requires returning another tick from the handler [V, official docs]. Bugs:
  - *Duplicate chains:* re-arming the tick from both the tick handler and the refresh-complete / manual-refresh handler creates two (then three) parallel tick loops; refresh frequency silently doubles, which is a rate-limit bug (Pitfall 5) disguised as a UI bug.
  - *Stacked refreshes:* tick fires every N s regardless of whether the last fetch finished.
  - *Out-of-order responses:* an older slow response arriving after a newer one overwrites fresh data with stale data.
  - *Interval change* via config reload does not cancel the old tick chain.
- Goroutine/resource leaks (the real ones, since an individual `tea.Tick` goroutine ends after its timer): HTTP requests with no `context` that outlive quit; background goroutines using `p.Send` with no stop channel; per-refresh `time.NewTicker` never stopped.
- Using `tea.Every` (aligned to wall clock) for fetch scheduling causes all instances to hit the API on the same second boundary; harmless for one user, but prefer `tea.Tick` after-completion scheduling.
- Mixing a 1 s "relative age" tick with the fetch tick in one handler causes the fetch cadence to equal the display cadence.

**Why it happens:**
Examples in tutorials re-arm the tick unconditionally and do trivial work. Elm-style architecture hides concurrency, so devs forget requests are concurrent.

**How to avoid:**
- Pattern: *schedule next fetch only when the previous fetch result arrives* (`FetchDoneMsg` -> `tea.Tick(interval, ...)` carrying a `generation int`). A tick with a stale generation is ignored. "Refresh now" bumps the generation and fires one fetch (cancelling the old context).
- Model fields: `inFlight bool`, `reqSeq uint64`; responses carry `seq`; drop if `seq < latestApplied`.
- Pass a `context.Context` created in `main`, cancelled on quit (`tea.Quit` path and signal handler), into every Cmd; use `http.NewRequestWithContext` and a per-request timeout (10-20 s).
- Keep a separate, cheap `clockTickMsg` (e.g. every 30 s) only to re-render "updated 3m ago"; it must never trigger I/O.
- Test: unit-test `Update` as a pure function (feed `TickMsg`, `FetchDoneMsg`; assert emitted cmds count). Add `goleak`/`-race` in tests. Count requests in an integration test with a fake server over 10 simulated minutes and assert exactly N.

**Warning signs:**
Request count at the mock server is 2x expected after pressing `r`. UI hiccups during fetch. After quitting, the process hangs for seconds. Data "goes back in time".

**Phase to address:** P3 (refresh loop architecture) -- design it before wiring real fetches; P5 soak test.

---

### Pitfall 8: Selection, scroll, and filter state lost or jumping on each refresh

**What goes wrong:**
- `bubbles/list` `SetItems` replaces all items; the cursor stays at the same *numeric index* [V: `SetItems(i) tea.Cmd`, `Index()` is the filtered index; `GlobalIndex()` is the unfiltered one], so if the order changes, the highlight moves to a different PR mid-keystroke. If the user is about to press "open in browser", they open the wrong PR.
- `SetItems` returns a `tea.Cmd` when a filter is active [V]; dropping that return value breaks filter re-application.
- Recreating the list model every refresh (`list.New(...)`) resets filter text, cursor, page, and scroll.
- Sorting by `updatedAt` or by status makes rows reorder on every poll as CI flips; the list shimmers and the user cannot track a row.
- The list in the detail view: refresh arrives while the detail view is open and the PR disappears (closed out of window) -> nil deref or wrong content.
- The user's own Open/Merged/Closed filter (requirement) interacts with auto-refresh: a newly merged PR vanishes from "Open" under the cursor.

**Why it happens:**
Selection is tracked by index, not identity.

**How to avoid:**
- Track `selectedID` (node ID) in the model. After applying new data: rebuild items, call `SetItems`, then locate `selectedID` in `list.Items()`/`VisibleItems()` and `Select(idx)`; if gone, select the nearest neighbour, not index 0.
- Handle the returned `tea.Cmd` from `SetItems`.
- Use a stable sort: group by state (Open, then Merged, then Closed), then repo, then PR number (or creation time); *never* sort by a field that the poll mutates. If "sort by activity" is offered, make it opt-in and freeze reordering while the user has navigated within the last few seconds.
- Do not rebuild the list model; update items in place. Keep the filter (Open/Merged/Closed set) in the app model and apply it to the item slice you pass in, instead of relying on list fuzzy-filter state.
- The detail view looks up its PR by ID on each render and shows "PR no longer in list" gracefully.
- Visually indicate changes (brief marker for rows whose status changed since last refresh) rather than re-sorting.

**Warning signs:**
Pressing `o` opens a different PR than the highlighted one right after a refresh. Cursor jumps to the top every minute. Filter text clears on refresh.

**Phase to address:** P3 (list view), P4 (detail-view back-navigation).

---

## Moderate Pitfalls

### Pitfall 9: Terminal sizing, narrow panes, and wide characters

**What goes wrong:** The intended use is a *multiplexer pane*, often narrow (60-80 cols) and short. The first `View()` call can occur before any `WindowSizeMsg`, with width/height 0 [K], producing panics (negative `strings.Repeat`), or a one-frame garbage render. Hardcoded column widths wrap and corrupt the layout. Titles containing emoji/CJK/ANSI break width math when using `len()`.
**Prevention:** Treat size 0 as "loading". Handle `WindowSizeMsg` by recomputing every component size (list, viewport, help). Use `lipgloss.Width` / `ansi.Truncate` (not `len`/byte slicing) for truncation. Define a responsive layout: drop columns progressively (reviewers -> author -> repo) as width shrinks; show "terminal too small" under a minimum. Golden-file tests at 60x15, 80x24, 120x40. Re-measure after the user resizes the pane while detail view is open.
**Phase:** P3.

### Pitfall 10: Alt-screen, logging, and terminal restoration

**What goes wrong:** Any `fmt.Println`/`log.Println` to stdout/stderr during the TUI corrupts the display. A panic in a Cmd goroutine can leave the terminal in raw mode / alt screen. v1 vs v2 API differences: in Bubble Tea v2 (`charm.land/bubbletea/v2`) alt-screen is a field on the `tea.View` returned by `View()` (`v.AltScreen = true`), `WithAltScreen` is removed, `View()` returns `tea.View`, and key handling is `tea.KeyPressMsg` [V from UPGRADE_GUIDE_V2]. Bubbles v2 and Lip Gloss v2 must be upgraded together with Bubble Tea v2 [V]. Most blog posts and LLM-generated snippets are v1-style and will not compile or will misbehave against v2. `tea.EnterAltScreen` as a command in `Init` is documented as not to be used in `Init` (v1) [V].
**Prevention:** Decide v1 vs v2 once (STACK.md), pin all three Charm modules to matching majors, and write code from the official upgrade guide, not memory. Log only to a file (`tea.LogToFile` or a `slog` file handler under the XDG state dir) with `--debug` flag; recover from panics in Cmds (wrap Cmd functions in a `safe(cmd)` helper that converts panic to `ErrMsg`). Detect non-TTY stdout and exit with a clear message. Open the browser using `exec.Command(open|xdg-open).Start()` (non-blocking, `Release`) and return a status message; do not pipe output to the terminal.
**Phase:** P1 (stack decision/skeleton), P3.

### Pitfall 11: Token resolution edge cases (`gh auth token` vs `GITHUB_TOKEN`)

**What goes wrong:**
- The requirement is "gh first, `GITHUB_TOKEN` fallback". But `gh` itself prefers `GH_TOKEN`/`GITHUB_TOKEN` env over stored credentials, so `gh auth token` in a shell with a stale or narrow `GITHUB_TOKEN` returns *that* token [K; verify on the installed gh version]. The user's "gh login" is then not what is used, producing confusing 401/SSO errors.
- Scopes: `gh auth login` default scopes include `repo` and `read:org`; a token missing `repo` cannot see private repos' PRs (silent NOT_FOUND). Fine-grained PATs in `GITHUB_TOKEN` are limited to a single resource owner and need explicit read permissions on Pull requests, Checks, Commit statuses, and Metadata; org-owned resources may need org approval [K].
- `gh` OAuth tokens do not expire on a timer [K], but can be revoked, or have SSO authorization revoked, or the user can run `gh auth logout/refresh` in another pane. The TUI keeps the old token in memory and fails forever with 401 "Bad credentials".
- gh not installed/not logged in: `exec` error message is cryptic.
- Non-github.com hosts: `gh auth token` without `--hostname github.com` may return a token for another configured host (GHE) [K]; the requirement is github.com only.
**Prevention:** Prefer go-gh's `auth.TokenForHost("github.com")` (same precedence rules as gh) or call `gh auth token --hostname github.com`. Always report which source was used (`gh`/`env`) in `--version`/debug output (not the token). On 401, re-resolve the token once and retry; if still 401 show an auth-failed screen with the fix ("run `gh auth login`") rather than looping. Probe `viewer { login }` at startup; show the login name and token source in the status bar. Check scopes from the `x-oauth-scopes` response header and warn if `repo` is missing (classic tokens only).
**Phase:** P1 (auth), P5 (401 re-resolution).

### Pitfall 12: Watch-list persistence and URL parsing

**What goes wrong:** Manual PRs are persisted to a local file: partial writes on crash corrupt it; two `prv` instances (two panes) overwrite each other's additions; the file location is the CWD instead of `os.UserConfigDir()`/XDG; the format has no version field so later migration (e.g. adding node ID) breaks old files. URL parsing fails on `/pull/123/files`, `/pull/123/commits`, `#issuecomment-…`, trailing slashes, `www.github.com`, query strings, `http://`, `/issues/123` URLs that redirect to PRs, repo names with dots/underscores, uppercase owners, and `owner/repo#123` with whitespace. Pasting a GitHub Enterprise URL should produce an explicit "github.com only" error. Auto-discovered PRs the user also added manually, then "removed", reappear (or removal silently does nothing).
**Prevention:** Atomic write (temp file in same dir + `os.Rename`), 0600 perms, `version` field, tolerant load (corrupt file -> backup `.bak` and start empty with a message). Re-read-before-write or flock to merge concurrent edits. Parse with `net/url` + a single tested table of accepted/rejected inputs (property: any valid PR URL suffix yields owner/repo/number). Validate by resolving the PR on add (`pullRequest(number)`), store owner/repo/number **and** node ID; show an inline error for NOT_FOUND/SSO. Define semantics: manual entry = pinned; removing an auto-included PR hides it (an "ignored" list) or is disallowed with a message. Dedupe by node ID.
**Phase:** P2.

### Pitfall 13: Draft, mergeable, and merge-queue states mistaken for approval/CI status

**What goes wrong:** Draft PRs are `state == OPEN` with `isDraft` true [V]; showing them identical to ready PRs misleads ("review required" on a draft). `mergeable` returns `UNKNOWN` while GitHub computes it lazily [V enum]; treating UNKNOWN as conflict flashes warnings on every first load. `mergeStateStatus` (`BLOCKED, BEHIND, DIRTY, UNSTABLE, CLEAN, HAS_HOOKS, UNKNOWN`) [V] can disagree with approval+CI (e.g. `BEHIND`, merge queue). A PR in a merge queue (`isInMergeQueue`) [V] has CI running on a different (merge group) ref.
**Prevention:** Show a Draft marker; do not show "approved/blocked" semantics as final for drafts. If merge status is displayed at all, treat `UNKNOWN` as "checking…" and re-poll next cycle. Keep v1 scope tight: approval + CI + state; add `mergeStateStatus` only if a requirement appears.
**Phase:** P1/P3.

### Pitfall 14: Time handling for the closed/merged recency window

**What goes wrong:** The "closed within N days" window computed in local time or with `time.Now()` at query-build time only once (never advanced during a days-long session). Hydration keeps showing PRs that aged out of the window; discovery stops returning them but the tombstone/ID cache keeps them forever.
**Prevention:** Compute the cutoff on every discovery cycle in UTC; filter hydrated results client-side by `closedAt`/`mergedAt` against the same cutoff so aged-out items drop even if present in the ID cache (manual entries are exempt: they stay until removed). Inject a `Clock` interface for tests.
**Phase:** P2.

### Pitfall 15: Query weight and GitHub-side timeouts

**What goes wrong:** One huge aliased query (100 PRs x reviews x contexts x steps) exceeds node limits or times out (502/"Something went wrong"), and then *every* refresh fails identically (no degradation path). [K]
**Prevention:** Chunk (Pitfall 4), keep list-query nested sizes small, lazy detail (Pitfall 3), and implement bisect-on-failure. Log the query size and `cost` at debug level.
**Phase:** P2.

---

## Minor Pitfalls

### Pitfall 16: Color-only status encoding and terminal theme detection

**What goes wrong:** Red/green only signals fail for color-blind users and `NO_COLOR` environments. Background detection (dark/light) can issue terminal queries that misbehave over tmux/ssh [L].
**Prevention:** Pair color with glyphs/text (`✓ ✗ … ○ !`), honor `NO_COLOR`, avoid relying on auto theme detection; provide a config override.
**Phase:** P3.

### Pitfall 17: Relative times and "last updated" lies

**What goes wrong:** Showing data with no freshness indicator: when refresh fails (rate limit, network down, laptop sleep), the screen looks live but is hours old. After laptop sleep, a pending tick fires immediately, plus an overdue discovery, causing a burst.
**Prevention:** Always show "updated Xm ago" and an error/staleness banner; dim rows older than 2 intervals. On resume (large gap between tick timestamps) do one coalesced refresh, not several.
**Phase:** P3/P5.

### Pitfall 18: Link and "open in browser" correctness

**What goes wrong:** Opening `HTMLURL` of the wrong item after refresh (Pitfall 8), using `open` on Linux, or failing silently over SSH/headless.
**Prevention:** Use the node's `url` field (never reconstruct), pick `open`/`xdg-open`/`wslview` by `runtime.GOOS`, show the URL in the status bar when launch fails (so it can be copied).
**Phase:** P4.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| One GraphQL query per PR | Simple code, typed structs | N+1 rate-limit burn, slow refresh, secondary-limit risk | Only for the very first spike; never in released builds |
| Render `reviewDecision` directly | 5 lines of code | Wrong for repos without protection (Pitfall 1) | Never |
| Map rollup `state` to badge only | No union handling | No "no checks", no detail view, non-required failures turn red | MVP list badge only if null/no-check case is handled; detail view needs full mapping |
| Index-based list selection | Default `bubbles/list` behaviour | Wrong PR opened after refresh | Never (cheap to fix at P3) |
| Fixed `tea.Tick` re-armed in two places | Quick manual-refresh key | Duplicate chains, doubled API use | Never |
| Typed `githubv4` structs for everything | Compile-time field safety | Cannot do dynamic aliases or partial data cleanly | OK for single-shot calls (viewer, add-PR validation); not for batch hydration |
| Hard-delete a manual PR on NOT_FOUND | Tidy list | Data loss on transient/SSO errors | Never |
| Store state in the list items only | Less state | Lose last-good data on per-item error | Never |
| Skip fixtures, test against live GitHub | No mock work | Flaky, rate-limit consuming, can't test error cases | Early exploration only |
| Pin nothing (floating Charm versions) | Always latest | v1/v2 mismatches break the build | Never; pin all three Charm modules together |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| GitHub GraphQL | Ignoring HTTP 200 + `errors[]` | Parse `errors[].path/type/extensions` per item; HTTP status alone is not success |
| GitHub GraphQL | Not requesting `rateLimit { cost remaining resetAt }` | Include it in every query; store latest in the model |
| GitHub GraphQL | Unions decoded without fragments (`StatusCheckRollupContext`, `requestedReviewer`, search `SearchResultItem`) | Use `... on CheckRun`, `... on StatusContext`, `... on User/Team`, `... on PullRequest` and a `__typename` check |
| GitHub GraphQL search | Omitting `is:pr`; trusting `issueCount` | Always `is:pr`; compare `issueCount` with retrieved to show truncation |
| GitHub GraphQL | Paginating contexts/reviews for all PRs | Use `*CountsByState` for list; paginate only selected PR |
| `gh` CLI | Shelling out on every request | Resolve once, cache in memory, re-resolve on 401 |
| `gh` CLI | Capturing stderr into logs/errors that might echo env | Capture stdout only; sanitize errors |
| GitHub SSO | Treating FORBIDDEN-SAML as "not found" | Show actionable SSO message with authorization URL when present |
| OS browser | Blocking `cmd.Run()` | `Start()` + `Release()`/goroutine wait |
| Filesystem | Writing watch file in place | Temp + rename; 0600; XDG path |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| N+1 requests per tick | `remaining` drains; 403 secondary limit | Batch via aliases / `nodes(ids)` | ~30+ PRs at <=60 s interval |
| Deep nested connections for every PR | Slow responses, 502 timeouts, cost grows | Counts in list, lazy detail | PRs with >100 checks; batches >50 PRs |
| Overlapping refreshes | Request count > expected, out-of-order UI | Single-flight + sequence IDs | Any time responses exceed the interval (slow network, rate-limit backoff) |
| Multiple `prv`/`gh-dash` instances | Shared 5,000/hr drains early in the hour | Conservative interval floor, adaptive slowdown on low `remaining`, show budget | 2-3 concurrent panes at 30 s |
| Re-render on every message with large item slices | Flicker, CPU | Update only changed rows; avoid re-sorting/re-styling items each frame | Hundreds of items (unlikely, but 1,000-cap search can produce it) |
| Unbounded memory in logs / history | Slow growth over days | Ring buffer for debug events; no per-poll history retention | Multi-day sessions |
| Laptop sleep/wake burst | Immediate multiple fetches | Detect large tick gap, coalesce | Every wake |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Logging HTTP requests/headers (debug transport, `httputil.DumpRequest`) | Token written to log file | Never dump `Authorization`; redact in any debug transport; wrap token in a type whose `String()`/`GoString()`/`MarshalJSON` returns `***` |
| Putting the token into error strings (`fmt.Errorf("auth failed with token %s")`) or panics | Leaks in logs/crash output | Errors reference the *source* ("gh", "env"), never the value |
| Passing token on a command line (`curl -H`, `gh api --header`) | Visible in `ps` | In-process HTTP client only |
| Persisting the token in config/state file | Long-lived credential on disk | Never persist; always resolve at start (gh keyring / env) |
| Recording test cassettes (VCR) with real `Authorization` headers or private PR titles | Secrets/private data committed to git | Scrub headers and private content in fixtures; commit synthetic fixtures; `.gitignore` recordings dir |
| Rendering PR titles/bodies/check names containing terminal escape sequences | Terminal control-sequence injection (PR titles are attacker-controlled in public repos) | Strip/escape control characters and ANSI from all remote strings before rendering (keep `\x1b` out of View output) |
| Using over-scoped token (`GITHUB_TOKEN` with write scopes, classic `repo`) | Larger blast radius if leaked | Document the minimal fine-grained permissions (read-only); v1 is read-only so never request write scopes |
| Opening arbitrary URLs from API data | Launching non-https handlers | Open only `https://github.com/...` URLs validated by host/scheme |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Blank screen on first load | Looks hung | Spinner + "Loading N PRs…" with progressive population |
| Errors replace the whole screen | One bad PR hides all others | Per-row error marker; global errors in status bar |
| No freshness indicator | User trusts stale data | "Updated 2m ago", rate-limit countdown, offline banner |
| Silent omission of SSO org PRs | "My PRs vanished" | Diagnostics line + docs hint (Pitfall 6) |
| Re-sorting and reselecting on refresh | Lost place | Stable order + selection by ID |
| Ambiguous review badge (null, none, required, dismissed) | Wrong merge decisions | Distinct states and glyphs; legend in help |
| Skipped/neutral/cancelled as red | Alert fatigue | Separate low-emphasis state |
| Key conflicts with list filter (`/`) or the multiplexer's prefix | Keys swallowed | Document keys; keep a `?` help; avoid Ctrl combos used by tmux/herdr |
| Filter (Open/Merged/Closed) resets on refresh | Annoyance | Keep filter in app model |
| Closed window setting silently hides PRs | "Where did it go?" | Show "N hidden (closed > 14d)" count |

## "Looks Done But Isn't" Checklist

- [ ] **Review status:** Often missing the null-`reviewDecision` fallback, dismissed/commented states, team reviewers, re-requested reviewers. Verify with fixtures for each, plus one real unprotected repo.
- [ ] **CI badge:** Often missing no-checks, queued vs running, WAITING, CANCELLED, neutral/skipped, >100 contexts, re-run dedupe. Verify with a PR with a 120-job matrix and a re-run.
- [ ] **Per-item errors:** Often missing SSO/NOT_FOUND distinction and last-good retention. Verify by watching a PR in an unauthorized SSO org and a deleted PR; list must still render.
- [ ] **Rate limiting:** Often missing backoff, reset awareness, single-flight. Verify with a mock server that returns 403 + `retry-after`, and by counting requests over a simulated hour.
- [ ] **Search discovery:** Often missing `is:pr`, truncation banner, vanishing-PR grace, UTC windows. Verify with >1,000 matches fixture and an Issue node in results.
- [ ] **Refresh loop:** Often has two tick chains or stale overwrites. Verify via a counting fake client and a test that presses `r` rapidly.
- [ ] **Selection:** Often index-based. Verify by reordering items between refreshes in a test and asserting the selected ID is unchanged.
- [ ] **Resize:** Often panics on 0 size or breaks at 60 cols. Verify golden renders at 0x0, 40x10, 60x15, 200x60.
- [ ] **Terminal restore:** Verify `kill -TERM`, panic in a Cmd, and quit during in-flight request all restore the terminal and exit promptly.
- [ ] **Token handling:** Verify `grep -r` of logs after a full debug session finds no token; 401 mid-session triggers one re-resolve.
- [ ] **Persistence:** Verify kill -9 during save leaves a valid file; two instances don't clobber; URL table tests pass.
- [ ] **Soak:** Run for hours with a short interval against a mock; memory flat, goroutine count flat, request count exact.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Rate-limited (primary) | LOW | Read `resetAt`, pause refresh, show countdown, auto-resume |
| Secondary limit hit | LOW-MEDIUM | Honor `retry-after` (else wait >=60 s), exponential backoff, reduce concurrency/batch size; do not retry immediately |
| Corrupt watch-list file | LOW | Rename to `.bak`, start empty, message user, keep in-memory copy |
| Wrong review/CI mapping shipped | MEDIUM | Mapping lives in one pure function with table tests; add fixture for the regression, patch |
| Duplicate tick chains discovered | LOW | Introduce generation counter; add request-count test |
| Token revoked/SSO deauthorized mid-session | LOW | Re-resolve token once; else show auth screen with fix command; keep showing stale data marked stale |
| Discovery used as state source (lag bugs) | MEDIUM-HIGH | Requires splitting discovery vs hydration; cheaper to design that way from P2 |
| v1/v2 Charm API mismatch late | HIGH | Do the decision in P1; pin versions; follow upgrade guide |
| Tracked selection by index shipped | LOW | Add selectedID tracking; touches list update path only |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| 1 Null `reviewDecision` | P1 mapping, P2 live check | Table test; real unprotected-repo PR shows approvals |
| 2 Reviews state handling | P1, P4 | Fixtures: comment-after-approve, dismissed, team request, re-request |
| 3 Check rollup mapping | P1 (mapper), P2 (counts), P4 (lazy detail) | Exhaustive enum test; 120-job matrix PR; no-checks PR; re-run PR |
| 4 Batched per-item errors | P2 | Mixed success/NOT_FOUND/SAML fixture; list still renders |
| 5 Rate limits | P1 client, P2 polling tiers, P5 budget UI/soak | Counting mock over simulated hours; 403 + retry-after test |
| 6 Search caps/lag/omissions | P2 | >1000 fixture; Issue node; vanishing-PR grace test |
| 7 Update blocking / tick chains | P3 | Fake client call count; race + goleak tests |
| 8 Selection/sort stability | P3, P4 | Reorder test keeps selectedID; manual check of `o` after refresh |
| 9 Resize/narrow panes | P3 | Golden renders at several sizes incl. 0x0 |
| 10 Alt-screen/logging/v1-v2 | P1, P3 | Panic-in-Cmd test; no stdout writes; pinned modules |
| 11 Token edge cases | P1, P5 | 401 re-resolve test; env-vs-gh precedence manual check |
| 12 Watch-list persistence/parsing | P2 | Crash-safety test; URL table test |
| 13 Draft/mergeable semantics | P1/P3 | Draft fixture; UNKNOWN mergeable fixture |
| 14 Time window | P2 | Clock-injected test across midnight/UTC |
| 15 Query weight/timeouts | P2 | Bisect-on-failure test; logged `cost` per query |
| 16-18 Minor UX/links | P3/P4 | Visual review; NO_COLOR run |

## Items needing early live-verification (spike candidates)

These claims are marked [K]/[L] and affect design; resolve with a 1-hour spike against the real API before locking the data layer:
1. `latestOpinionatedReviews` ignores `COMMENTED` (vs `latestReviews` letting a comment override an approval).
2. Rollup `contexts` after "re-run failed jobs": latest-only or both runs?
3. How the chosen GraphQL client surfaces partial `data` with `errors[]` (and the exact `type`/`extensions` for SSO and NOT_FOUND).
4. GraphQL `search` behaviour for SSO-unauthorized orgs (silent omission vs error), and whether `author:@me` works in GraphQL search strings.
5. `gh auth token` precedence when `GITHUB_TOKEN` is set; behaviour with multiple hosts.
6. Whether `nodes(ids:)` max is 100 and how deleted PRs appear.
7. Real `cost` of the chosen list query for ~25 PRs (read from `rateLimit.cost`).

## Sources

- GitHub Docs, "Rate limits and node limits for the GraphQL API" (primary 5,000 pts/hr, cost formula, 500,000 node limit, secondary: 100 concurrent, 2,000 pts/min GraphQL, CPU time) -- https://docs.github.com/en/graphql/overview/rate-limits-and-node-limits-for-the-graphql-api (HIGH)
- GitHub Docs, "Rate limits for the REST API" (secondary-limit backoff rules, `retry-after`, `x-ratelimit-*`, "no way to check secondary limit status", ban warning) -- https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api (HIGH)
- GitHub GraphQL public schema (`schema.docs.graphql` from docs.github.com/public/fpt): `PullRequest.reviewDecision` nullable; `latestReviews`, `latestOpinionatedReviews(writersOnly)`; `statusCheckRollup` nullable; `StatusCheckRollup.state`; `StatusCheckRollupContext = CheckRun | StatusContext`; `CheckStatusState`, `CheckConclusionState`, `StatusState`, `PullRequestReviewState`, `MergeableState`, `MergeStateStatus` enums; `search` "maximum of 1,000 results"; `SearchType` incl. `ISSUE_ADVANCED`; `nodes(ids)` returns `[Node]!`; `isRequired(...)` on checks (HIGH, retrieved 2026-10-11)
- GitHub Docs, REST "Search" (30 req/min authenticated, `incomplete_results` on timeout) -- https://docs.github.com/en/rest/search/search (HIGH for REST; GraphQL search cost accounting is K)
- Bubble Tea docs via Context7 / pkg.go.dev: `Tick`/`Every` send a single message and must be re-armed; `EnterAltScreen` not for `Init`; bubbles `list.SetItems` returns a Cmd when filtering, `Index` vs `GlobalIndex`, `Select` (HIGH)
- Bubble Tea v2 upgrade guide (`UPGRADE_GUIDE_V2.md`): `View() tea.View`, `v.AltScreen`, `KeyPressMsg`, removed `WithAltScreen`; Bubbles v2/Lip Gloss v2 must match (HIGH) -- https://raw.githubusercontent.com/charmbracelet/bubbletea/main/UPGRADE_GUIDE_V2.md
- Community note on `latestReviews` lag behind fresh reviews (single non-official source; LOW) -- https://git.cynarski.dev/Awesome/awesome-copilot/raw/branch/main/skills/copilot-pr-autopilot/references/api-quirks.md
- Behaviours tagged [K] (null `reviewDecision` without required reviews, SAML FORBIDDEN error shape, NOT_FOUND masking private repos, search index lag, GraphQL partial-data semantics, GraphQL query timeouts, `gh` env-token precedence) come from established GitHub/gh community experience and were not confirmable from official docs in this session; see "Items needing early live-verification".

---
*Pitfalls research for: GitHub PR monitoring TUI (Go + Bubble Tea)*
*Researched: 2026-10-11*

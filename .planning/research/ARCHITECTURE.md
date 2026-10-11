# Architecture Patterns

**Domain:** Go + Bubble Tea terminal UI that polls GitHub for a curated set of pull requests
**Project:** prv
**Researched:** 2026-10-11
**Overall confidence:** HIGH (GraphQL shapes verified live against api.github.com with the user's own `gh` token; Bubble Tea v2 API from the official upgrade guide and gh-dash's go.mod; gh-dash internals from its public tree)

---

## Headline Decisions

1. **Use Bubble Tea v2, not v1.** Import `charm.land/bubbletea/v2` (latest v2.1.0), `charm.land/bubbles/v2` (v2.2.1), `charm.land/lipgloss/v2` (v2.0.6). gh-dash itself is on `bubbletea/v2 v2.0.2` + `bubbles/v2 v2.0.0`. Most blog posts and LLM-remembered examples are v1 and will not compile (see "v2 differences" below).
2. **One GraphQL HTTP request per refresh.** A single `ISSUE_ADVANCED` search (with `OR` and parentheses) covers authored + assigned + open + recently-closed. Manual PRs ride in the same request as aliased `repository(...){pullRequest(number:)}` fields. Cost is about 1 to 3 rate-limit points per refresh.
3. **Strict layering with a domain model in the middle.** `internal/github` converts wire JSON into `internal/domain` types. `internal/ui` only ever sees domain types. `internal/store` persists only `domain.PRRef` values, never PR data.
4. **Fetch is a `tea.Cmd` closure over copied inputs, polling is re-armed after each result.** No goroutines mutate the model. No overlapping fetches.

---

## Recommended Architecture

### System Overview

```
                        ┌────────────────────────────────────────┐
  cmd/prv/main.go  ───► │ wiring: config, auth, store, fetcher,  │
  (composition root)    │ tea.NewProgram(ui.New(deps)).Run()     │
                        └───────────────┬────────────────────────┘
                                        │ injects interfaces
┌───────────────────────────────────────▼───────────────────────────────┐
│ internal/ui   (Bubble Tea: Model / Update / View)                     │
│                                                                       │
│  root Model ── owns: snapshot []domain.PR, watch []domain.PRRef,      │
│    │           filter, mode, gen counter, in-flight flag, lastErr     │
│    ├── list view     (cursor, scroll offset, row rendering)           │
│    ├── detail view   (checks + reviewers for selected PR, viewport)   │
│    ├── add prompt    (bubbles textinput, parses URL / owner/repo#N)   │
│    └── statusbar/help/filter bar                                      │
│                                                                       │
│  Cmds: fetchCmd, tickCmd, saveWatchCmd, openBrowserCmd                │
└───────┬──────────────────────┬──────────────────────┬─────────────────┘
        │ Fetcher (interface)  │ WatchStore (iface)   │ uses types
        ▼                      ▼                      ▼
┌──────────────────┐   ┌────────────────────┐   ┌──────────────────────┐
│ internal/github  │   │ internal/store     │   │ internal/domain      │
│  auth.go         │   │  JSON file, atomic │   │  PR, PRRef, State,   │
│  client.go       │   │  write, versioned  │   │  ReviewDecision,     │
│  query.go (build)│   │                    │   │  Review, Check,      │
│  wire.go (JSON)  │   │                    │   │  Rollup, ParseRef()  │
│  map.go (→domain)│   │                    │   │  (no deps on others) │
└────────┬─────────┘   └─────────┬──────────┘   └──────────────────────┘
         │ HTTPS POST             │ read/write
         ▼                        ▼
   api.github.com/graphql   $XDG_CONFIG_HOME/prv/watchlist.json
```

Dependency rule: arrows point down only. `domain` imports nothing from this module. `github` and `store` import `domain`. `ui` imports `domain` and defines the small interfaces it needs (`Fetcher`, `WatchStore`) itself (consumer-side interfaces), so it never imports `github` or `store`. Only `cmd/prv` imports all of them.

### Component Boundaries

| Component | Responsibility | Communicates With | Must NOT |
|-----------|---------------|-------------------|----------|
| `cmd/prv` | Parse flags, load config, resolve token, construct fetcher + store, run program | everything (composition root only) | contain logic |
| `internal/domain` | Pure types and pure functions: `PR`, `PRRef`, `State` derivation, rollup classification, `ParseRef`, sorting/filter predicates | nothing (leaf) | import github/store/ui, know JSON shape |
| `internal/github` | Auth/token resolution, build one GraphQL document, POST it, decode envelope, map wire structs to domain, classify partial errors, report rate-limit info | `domain`; api.github.com | leak wire structs outside the package; know about Bubble Tea |
| `internal/store` | Load/Add/Remove the manual watch list; atomic file writes; schema version | `domain` (PRRef); filesystem | store PR data/state; import ui |
| `internal/config` (small) | Defaults + file/env/flag for refresh interval, closed-window days | none | be required for first run |
| `internal/ui` | All Bubble Tea models, messages, key bindings, styles, layout | `domain`; `Fetcher`/`WatchStore` interfaces | do I/O inside `Update`/`View`; touch wire types |

### Data Flow

```
Startup
  Init() ──► tea.Batch( loadWatchCmd , tea.RequestWindowSize )
  watchLoadedMsg{refs} ──► m.watch = refs ──► fetchCmd(gen=1)

Refresh cycle (steady state)
  fetchCmd(gen, refs, cutoff)            [runs in Bubble Tea's goroutine pool]
     └─ Fetcher.Fetch(ctx, Request{Refs, ClosedSince})
          └─ 1 POST /graphql  → envelope{data, errors}
          └─ map wire → []domain.PR + []RefError + RateInfo
     returns refreshedMsg{gen, Result, err}
  Update(refreshedMsg)
     ├─ gen != m.gen            → drop (stale result)
     ├─ err (total failure)     → keep old snapshot, set m.lastErr, mark stale
     └─ ok                      → merge, re-sort, restore cursor by PR key,
                                  m.inFlight=false, return tickCmd(interval)
  Update(tickMsg{gen})
     └─ if !inFlight && gen == m.gen → m.inFlight=true, fetchCmd(...)

User actions (all mutate model in Update, side-effects returned as Cmds)
  a  → add prompt → ParseRef → fetchCmd for that ref only (validate it exists)
       → on success: m.watch += ref, saveWatchCmd(m.watch copy), then refresh
  d  → m.watch -= ref (only if Sources has Manual), saveWatchCmd, drop/refresh
  r  → m.gen++, fetchCmd (cancels nothing; stale result discarded by gen)
  o  → openBrowserCmd(pr.URL)
  f  → toggle Open/Merged/Closed filter bits (pure view filter, no fetch)
  ⏎/esc → switch list/detail mode (no fetch; detail data is already in snapshot)
```

Key properties:
- **Direction:** GitHub → github (map) → domain → ui. User intent flows ui → store (refs only) and ui → github (request parameters: refs + cutoff).
- **Detail view costs zero extra requests.** The list query already fetches per-check data; the detail view renders from the same snapshot. This is the biggest simplification versus gh-dash, which lazily fetches per-PR detail.
- **Filters are view-level except the closed window.** Open/Merged/Closed toggles filter the in-memory snapshot instantly. The closed-window cutoff is a query parameter (it bounds what is fetched).

---

## Recommended Project Structure

```
prv/
├── cmd/
│   └── prv/
│       └── main.go            # composition root; ~60 lines
├── internal/
│   ├── domain/
│   │   ├── pr.go              # PR, PRRef, State, Source flags, Review, Check, Rollup
│   │   ├── ref.go             # ParseRef("https://github.com/o/r/pull/12" | "o/r#12")
│   │   ├── rollup.go          # classify checks → Passing/Failing/Running/None
│   │   ├── filter.go          # StateSet, Filter(prs, set), Sort(prs)
│   │   └── *_test.go
│   ├── github/
│   │   ├── auth.go            # token: `gh auth token -h github.com`, then GITHUB_TOKEN
│   │   ├── client.go          # Fetcher impl: build → POST → decode → map
│   │   ├── query.go           # builds the GraphQL document + variables
│   │   ├── wire.go            # JSON structs mirroring the response (unexported)
│   │   ├── map.go             # wire → domain (the only place enums are translated)
│   │   ├── errors.go          # partial-error classification (NOT_FOUND, FORBIDDEN, rate limit)
│   │   └── testdata/*.json    # recorded real responses for golden tests
│   ├── store/
│   │   ├── watchlist.go       # Load/Add/Remove, atomic write, schema version
│   │   └── watchlist_test.go
│   ├── config/
│   │   └── config.go          # interval, windowDays, defaults, validation (min interval)
│   └── ui/
│       ├── app.go             # root Model: Init/Update/View, mode switch
│       ├── msgs.go            # all tea.Msg types
│       ├── cmds.go            # fetchCmd, tickCmd, saveWatchCmd, openBrowserCmd
│       ├── keys.go            # key bindings + help text
│       ├── styles.go          # lipgloss styles, badge rendering
│       ├── list.go            # list sub-model (cursor/offset/row render)
│       ├── detail.go          # detail sub-model (viewport)
│       ├── addprompt.go       # textinput sub-model
│       ├── statusbar.go       # last-updated, errors, rate-limit, filter chips
│       └── *_test.go
├── go.mod
└── .planning/
```

### Structure Rationale

- **`domain/` is added to the cmd/internal/github/store/ui layout.** Without it, either `ui` imports `github` types (coupling the view to the API) or `store` and `github` import each other for `PRRef`. A leaf package removes both cycles and makes the rollup/review logic unit-testable with zero mocks.
- **`internal/` for everything** since this is a binary, not a library. Nothing here needs to be importable.
- **Sub-models in `ui/` are plain structs with `update`/`view` methods in the same package**, not separate packages. gh-dash splits into ~25 component packages (`prssection`, `prrow`, `sidebar`, `tabs`, ...) because it has issues, notifications, branches, multiple configurable sections, and a theme system. prv has one list and one detail; one package with several files is the right size. Split a package out only when a component is reused.
- **Wire types are unexported and live next to the query.** The query text and its decoding structs change together; the mapper is the single translation seam.

---

## Elm Architecture Decomposition (Bubble Tea v2)

### v2 differences that will bite (verified from the official upgrade guide)

| v1 (most examples online) | v2 (use this) |
|---------------------------|---------------|
| `github.com/charmbracelet/bubbletea` | `charm.land/bubbletea/v2` (likewise `lipgloss/v2`, `bubbles/v2`) |
| `View() string` | `View() tea.View` via `tea.NewView(s)` |
| `tea.WithAltScreen()` / `tea.EnterAltScreen` | set `v.AltScreen = true` on the returned `tea.View` |
| `case tea.KeyMsg:` + `msg.String()` | `case tea.KeyPressMsg:` (space is now `"space"`, not `" "`) |
| `tea.WindowSize()` cmd | `tea.RequestWindowSize` (a message-returning func); `tea.WithWindowSize(w,h)` for tests |
| `tea.Sequentially` | `tea.Sequence` |

`Init() tea.Cmd` and `Update(tea.Msg) (tea.Model, tea.Cmd)` are unchanged. Confirm exact `tea.Tick`/`tea.Batch` signatures against pkg.go.dev when implementing (the upgrade guide does not cover them; they are believed unchanged).

### Root model

```go
type mode int
const (modeList mode = iota; modeDetail; modeAdd; modeHelp)

type Model struct {
    deps    Deps                 // Fetcher, WatchStore, Opener, Config (interfaces)
    // data
    watch   []domain.PRRef       // manual refs, source of truth in memory
    prs     []domain.PR          // last good snapshot, already sorted
    refErrs map[domain.PRRef]string // per-ref "not found / no access"
    // refresh state
    gen      int                 // bumped on manual refresh / watch change
    inFlight bool
    lastOK   time.Time
    lastErr  error
    rate     domain.RateInfo
    // view state
    mode    mode
    filter  domain.StateSet      // Open|Merged|Closed bitset
    list    listModel
    detail  detailModel
    add     addModel
    width, height int
}
```

- **The snapshot is the single source of truth for rendering.** Views are pure functions of `(snapshot, filter, cursorKey, size)`. Never cache rendered strings across updates except per-row memoization if profiling demands it (it will not at tens of rows).
- **Cursor is identified by PR key (`owner/repo#n`), not by index.** After every refresh or filter change, re-sort and re-locate the cursor by key (fall back to nearest index if the PR vanished). Otherwise rows jump under the user's cursor every poll.
- **Sorting must be deterministic and stable across polls:** e.g. state group (Open, then Merged/Closed by closedAt desc), then repo, then number. Do not sort by `updatedAt` for the primary order or rows will reshuffle while CI runs.

### Messages

```go
type watchLoadedMsg struct{ refs []domain.PRRef; err error }
type tickMsg        struct{ gen int }
type refreshedMsg   struct{ gen int; res github.Result; err error } // use domain.Result, see below
type addResolvedMsg struct{ ref domain.PRRef; pr *domain.PR; err error }
type watchSavedMsg  struct{ err error }
type openedMsg      struct{ err error }
```

Define `Result` (`PRs []domain.PR`, `RefErrors map[domain.PRRef]error`, `Rate domain.RateInfo`) in `domain` so the `ui` package never imports `github`.

### Commands (the only place I/O happens)

```go
func fetchCmd(f Fetcher, gen int, refs []domain.PRRef, since time.Time) tea.Cmd {
    refs = slices.Clone(refs)                   // copy: Cmd runs on another goroutine
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        res, err := f.Fetch(ctx, domain.Request{Refs: refs, ClosedSince: since})
        return refreshedMsg{gen: gen, res: res, err: err}
    }
}

func tickCmd(gen int, d time.Duration) tea.Cmd {
    return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg{gen: gen} })
}
```

Rules:
- A `Cmd` closes over **copies** of what it needs; it never reads or writes the model. Results come back only as messages.
- `Update` never blocks. File writes (`saveWatchCmd`) and the browser launch are Cmds too.
- Return `tea.Batch(...)` when starting several independent Cmds; use `tea.Sequence` only when order matters (e.g. save the watch list, then refresh).

### Polling pattern: re-arm after completion

```
Init/Add/Remove/manual-r:  gen++ ; inFlight=true ; fetchCmd(gen)
refreshedMsg (gen current): inFlight=false ; tickCmd(interval, gen)
tickMsg (gen current):      if !inFlight { inFlight=true ; fetchCmd(gen) }
any msg with stale gen:     ignored
```

- **Re-arm on completion, not on a fixed clock.** A slow or retrying fetch can never overlap the next one. Prefer `tea.Tick` over `tea.Every` (Every aligns to wall-clock boundaries, which only matters for clocks).
- **The `gen` counter** handles manual refresh, watch-list edits and config changes: older in-flight results and older ticks are dropped instead of cancelled. One pending tick chain exists at any time.
- **Errors keep the last good snapshot.** Show "stale since 14:02 – network error" in the status bar and keep ticking (with simple backoff: double the interval up to ~5 min on consecutive failures, reset on success).
- **Rate-limit awareness:** the query selects `rateLimit { cost remaining resetAt }`. If `remaining` falls below a threshold (say 100) or the API returns a rate-limit error, delay the next tick until `resetAt`. Surface "rate limited until HH:MM" in the status bar.
- **Wall-clock labels ("updated 12s ago")**: either render the absolute `lastOK` time (zero extra ticks) or add a separate 1 s tick that only triggers a re-render. Prefer absolute time for v1.

### View decomposition

- `View()` composes: header (title + filter chips) / body (list **or** detail) / footer (status bar + key hints). Set `AltScreen = true` on the `tea.View`.
- **List:** custom rendering with lipgloss, one line per PR: `state-icon  repo#num  title(truncated)  review-badge  CI-badge  reviewers`. Do **not** use `bubbles/list` (its item-delegate/filter/pagination model fights multi-badge rows and a fixed status bar) and `bubbles/table` is a poor fit for per-cell colored badges. Own cursor + scroll offset logic is about 60 lines.
- **Detail:** `bubbles/viewport` (v2) holding: PR header, review decision + per-reviewer list, then one line per check (`workflow / job name`, status glyph, conclusion), grouped failing → running → passing. Scrollable.
- **Layout:** v1 = full-screen swap (Enter opens detail, Esc returns). Side-by-side master/detail at wide terminals is a polish item, not a v1 need; the data flow does not change if added later. Handle `tea.WindowSizeMsg` once in the root and pass sizes down.
- **Truncate by display width** (use lipgloss/`ansi` width helpers, not `len`), since titles contain emoji/CJK.

---

## GitHub GraphQL Query Shape (verified live)

All queries below were executed against api.github.com with a `gh` token; the shapes and behaviors described are observed, not assumed.

### Part 1: Authored + assigned, with closed window, in ONE search

```graphql
query Watch($q: String!) {
  rateLimit { cost remaining resetAt }
  search(query: $q, type: ISSUE_ADVANCED, first: 100) {
    pageInfo { hasNextPage endCursor }
    nodes { __typename ...PR }
  }
  # ...manual aliases appended here (Part 2)
}
```

Variable:

```
is:pr archived:false (author:@me OR assignee:@me) (is:open OR closed:>=2026-09-27) sort:updated-desc
```

Verified facts:
- **`type: ISSUE_ADVANCED` supports `OR` and parentheses.** The same query string with legacy `type: ISSUE` returned **0 results** (silently, no error). Use `ISSUE_ADVANCED`. (The SearchType enum also has `ISSUE_SEMANTIC`/`ISSUE_HYBRID`; irrelevant here.)
- `@me` resolves to the token's user in GraphQL search.
- `closed:>=YYYY-MM-DD` matches **merged and closed** PRs (merged PRs are closed); the user's merged PRs came back under it. Compute the date in the fetcher as `now.AddDate(0,0,-windowDays)` formatted `2006-01-02` (date granularity, so the window is approximate to within a day; filter precisely client-side on `closedAt` if needed).
- Combined `(is:open OR closed:>=DATE)` was accepted. (The test account had no open PRs, so the open branch was verified only in isolation on the legacy type. Re-verify with an account that has open PRs during the fetch-layer phase.)
- `sort:updated-desc` inside the query string orders results; the UI re-sorts anyway.
- **Fallback if `ISSUE_ADVANCED` ever regresses:** four aliased `search(type: ISSUE)` fields in the same document (`author`/`assignee` × `is:open`/`closed:>=DATE`), deduplicated by PR node `id` in the mapper. Same single HTTP request, ~4 points.
- Cost measured: **cost 1 to 2 points** for a 50-PR search with `contexts(first:50)`. The 5,000 pt/hour GraphQL budget allows a 30 s poll (120 req/h) with a vast margin. The 30/min limit applies to the REST search API, not GraphQL search.
- Pagination: `first: 100` is the max page. A personal watch list rarely exceeds 100; implement `hasNextPage` follow-up (cursor) but do not optimize it.

### Shared fragment

```graphql
fragment PR on PullRequest {
  id number title url isDraft state merged closedAt updatedAt
  repository { nameWithOwner }
  author { login }
  reviewDecision
  latestReviews(first: 20) { nodes { state author { login } } }
  reviewRequests(first: 10) {
    nodes { requestedReviewer { __typename ... on User { login } ... on Team { slug } } }
  }
  commits(last: 1) { nodes { commit { oid statusCheckRollup {
    state
    contexts(first: 100) {
      totalCount
      pageInfo { hasNextPage }
      nodes {
        __typename
        ... on CheckRun {
          name status conclusion detailsUrl startedAt completedAt
          checkSuite { workflowRun { workflow { name } } }
        }
        ... on StatusContext { context state targetUrl description }
      }
    }
  } } } }
}
```

Verified facts:
- `commits(last:1){nodes{commit{statusCheckRollup{...}}}}` works and also yields the head `oid` for free (useful for change detection). A shortcut `PullRequest.statusCheckRollup` also exists and returned identical data; either is fine, prefer `commits(last:1)` because it is the established gh CLI shape and gives the oid.
- **`statusCheckRollup` is `null`** when the head commit has no checks (observed on the user's own repos). Map to rollup `None`, not "passing".
- **`reviewDecision` is `null`** when the repo has no required-review rule (observed). Map to `ReviewNone` and show "no review required", not an error.
- `latestReviews` returns the most recent review per reviewer; states: `PENDING, COMMENTED, APPROVED, CHANGES_REQUESTED, DISMISSED`. Show only `APPROVED` and `CHANGES_REQUESTED` as reviewer chips; ignore the rest.
- A real-world PR (cli/cli, 30 contexts) returned `CheckRun` nodes with `checkSuite.workflowRun.workflow.name`. Selecting the workflow name lets the detail view show `Lint / lint` rather than just `lint`. `workflowRun` can be null for third-party apps; handle null.
- **`contexts` truncation:** `totalCount` > nodes returned (or `hasNextPage`) means big matrices were cut. Set `Rollup.Truncated = true` and show "+N more" in the detail view. The badge still comes from GitHub's `statusCheckRollup.state`, which covers all contexts even when the list is truncated. Do not recompute the badge from the truncated list.
- `mergeable` returned `UNKNOWN` for old PRs (computed lazily by GitHub). **Do not use `mergeable` for v1 UI.**

Enum values to translate in `map.go` (from GitHub's schema, standard):
- `CheckStatusState`: QUEUED, IN_PROGRESS, COMPLETED, WAITING, PENDING, REQUESTED
- `CheckConclusionState`: SUCCESS, FAILURE, NEUTRAL, CANCELLED, SKIPPED, TIMED_OUT, ACTION_REQUIRED, STARTUP_FAILURE, STALE
- `StatusState` (both rollup and `StatusContext`): EXPECTED, ERROR, FAILURE, PENDING, SUCCESS
- Introspection is capped at 2 `enumValues` uses per query by the API, so confirm enum lists with `gh api graphql` one at a time if needed.

Badge classification (pure function in `domain/rollup.go`):

| Rollup `state` | Badge |
|----------------|-------|
| `SUCCESS` | Passing |
| `FAILURE`, `ERROR` | Failing |
| `PENDING`, `EXPECTED` | Running |
| `null` (no rollup) | None |

Per-check display: `status != COMPLETED` → running/queued; conclusion `SUCCESS/NEUTRAL/SKIPPED` → ok (skipped dimmed); `FAILURE/TIMED_OUT/CANCELLED/ACTION_REQUIRED/STARTUP_FAILURE` → failed; `StatusContext.state` maps via StatusState. Treat unknown enum values as "unknown" (never panic, never silently pass) so a new GitHub enum value degrades gracefully.

### Part 2: Manual PRs as aliased lookups in the same document

```graphql
query Watch($q: String!, $o0: String!, $n0: String!, $c0: Int!, $o1: String!, ...) {
  rateLimit { cost remaining resetAt }
  search(...) { ... }
  m0: repository(owner: $o0, name: $n0) { pullRequest(number: $c0) { ...PR } }
  m1: repository(owner: $o1, name: $n1) { pullRequest(number: $c1) { ...PR } }
}
```

Verified facts:
- **Aliased lookups return fresh data, bypassing search indexing.** Search results are served from an index that can lag state changes by seconds to minutes (documented GitHub behavior, MEDIUM confidence on exact lag). A freshly added manual PR therefore appears immediately even if it is also in the search set.
- **Missing PR or repo does not fail the request.** Observed response: HTTP 200 with `data.mN = null` (bad repo) or `data.mN.pullRequest = null` (bad number), plus an `errors[]` entry `{"type":"NOT_FOUND","path":["mN", ...]}`. `gh api` exits non-zero for this, so **do not rely on `gh`-style error semantics**: parse the envelope yourself, use `errors[].path[0]` to attribute each error to its alias, and keep the remaining PRs.
- Inaccessible repos (private repo without access, org requiring SAML SSO authorization for the token) surface the same way (`NOT_FOUND`/`FORBIDDEN`). Show these as a placeholder row (`owner/repo#N  unavailable: not found or no access`) that the user can remove; never auto-remove from the persisted list.
- **Use GraphQL variables for owner/name/number, not string interpolation.** Users paste arbitrary text; interpolating into the query text is an injection/parse-error hazard. Also validate refs in `ParseRef` (`[A-Za-z0-9._-]+` for owner/repo, positive int).
- Chunk at 50 aliases per request (very conservative against the 500,000-node limit: 50 PRs × (100 contexts + 20 reviews + 10 requests) ≈ 6.5k nodes). A typical user has <20 manual refs, so this is a single request.
- Alternative for later: `nodes(ids: [...])` fetches up to 100 PRs by node ID in one field and is cheaper, but needs the node ID, which only exists after a first lookup. Not worth the cache-the-ID complexity for v1.

### Transport and auth

- **Use `github.com/cli/go-gh/v2` (latest v2.16.2) for the HTTP client** (`api.NewRESTClient`, which sets headers/user-agent), POST to path `graphql`, and decode into **your own envelope** (`struct{ Data json.RawMessage; Errors []gqlErr }`). Avoid the typed `GraphQLClient.Do` path: it is designed around a single typed response and returns an error for any `errors[]` entry, which makes per-alias partial failures awkward. (Whether it still populates `Data` on error was not verified. Taking raw control sidesteps the question.) gh-dash uses `go-gh/v2` + `cli/shurcooL-graphql`; prv's dynamic aliases fit raw query strings better than shurcooL's struct-tag queries.
- **Token order differs between go-gh and the PROJECT.md requirement.** `auth.TokenForHost("github.com")` prefers env (`GH_TOKEN`, then `GITHUB_TOKEN`), then the gh config file, then the keyring. PROJECT.md says gh login first, `GITHUB_TOKEN` fallback. Either implement the spec exactly (run `gh auth token -h github.com`, fall back to `GITHUB_TOKEN`, pass the result as `ClientOptions.AuthToken`) or consciously adopt go-gh's order (env override is the gh convention). **Recommendation: follow the spec in `auth.go` (small, testable via an injectable exec func) and document it.** Resolve once at startup; on a 401, re-resolve once and retry (the user may have run `gh auth refresh`).
- Fail fast and readably when no token is found ("run `gh auth login` or set GITHUB_TOKEN") before starting the TUI, so the error prints to a normal terminal, not an alt-screen.

---

## Domain Model (decoupled from API types)

```go
package domain

type PRRef struct{ Owner, Repo string; Number int }          // comparable → map key
func ParseRef(s string) (PRRef, error)                        // URL or owner/repo#N
func (r PRRef) String() string                                // "owner/repo#N"
func (r PRRef) URL() string

type State int   // Open, Merged, Closed — derived in map.go: merged→Merged, state=CLOSED→Closed
type ReviewDecision int // None, ReviewRequired, Approved, ChangesRequested
type ReviewState int    // Approved, ChangesRequested  (others dropped in mapping)
type RollupState int    // None, Passing, Failing, Running
type CheckState int     // Queued, Running, Passed, Failed, Skipped, Unknown
type Source uint8       // Authored|Assigned|Manual bit flags

type Review struct{ Reviewer string; State ReviewState }
type Check  struct{ Workflow, Name string; State CheckState; URL string }
type Rollup struct{ State RollupState; Checks []Check; Total int; Truncated bool }

type PR struct {
    Ref        PRRef
    NodeID     string      // dedupe key between search and aliased results
    Title, URL, Author string
    Draft      bool
    State      State
    ClosedAt   time.Time   // zero if open
    UpdatedAt  time.Time
    Decision   ReviewDecision
    Reviews    []Review    // Approved / ChangesRequested only
    PendingReviewers []string
    Rollup     Rollup
    Sources    Source      // why it is on the list
}

type Request struct{ Refs []PRRef; ClosedSince time.Time }
type Result  struct{ PRs []PR; RefErrors map[PRRef]error; Rate RateInfo }
type RateInfo struct{ Remaining int; ResetAt time.Time }
```

Rules:
- **The domain exposes derived, UI-ready semantics** (`State`, `RollupState`, `Decision`), never raw GitHub strings. The mapper is the only code that knows `"CHANGES_REQUESTED"`.
- **`Sources` is a bit set.** The mapper tags search hits `Authored`/`Assigned` (compare `author.login`/assignees to the viewer login, or simply tag as `Auto` if the distinction is not displayed) and aliased hits `Manual`. When the same `NodeID` appears in both, OR the flags. "Remove" is only offered when `Manual` is set; removing a manual ref that is also authored just clears the flag.
- **Manual PRs and the closed window.** PROJECT.md says closed/merged PRs appear only within the window. For *manual* PRs the user explicitly asked to watch, hiding them after the window is surprising. Recommended default: manual PRs are always shown regardless of age (they stay until removed), window applies to auto-discovered PRs only. **This needs a product decision in requirements** (flagged in Open Questions).
- Domain functions are pure and exhaustively table-tested: `ParseRef`, `ClassifyRollup`, `Sort`, `Filter`.

---

## Local Persistence of the Watch List

- **What is stored:** only refs plus metadata. Nothing derived from GitHub (state, title, checks) is ever persisted; the next refresh is authoritative. This avoids stale-cache bugs and schema churn.
- **Format:** JSON, versioned.

```json
{ "version": 1, "prs": [ { "owner": "cli", "repo": "cli", "number": 14634, "addedAt": "2026-10-11T12:00:00Z" } ] }
```

- **Location:** `$XDG_CONFIG_HOME/prv/watchlist.json`, defaulting to `~/.config/prv/` on macOS and Linux. Note `os.UserConfigDir()` returns `~/Library/Application Support` on macOS and ignores `XDG_CONFIG_HOME`; most terminal-tool users expect `~/.config`. Pick one convention in `config` and apply it to both the watch list and any config file. (It is user-authored intent, so config dir rather than state/cache dir.)
- **Atomic writes:** write to `os.CreateTemp` in the same directory, `fsync`, `os.Rename` over the target. A crash or a second `prv` instance can never leave a truncated file. `MkdirAll(0o755)` for the dir, `0o600` for the file.
- **Tolerant load:** missing file → empty list; unparseable file → surface a clear error and **do not overwrite it** (rename to `.bak` or refuse to start) so a hand-edit typo cannot wipe the list. Dedupe on load.
- **Interface (declared in `ui`, implemented in `store`):**

```go
type WatchStore interface {
    Load() ([]domain.PRRef, error)
    Save([]domain.PRRef) error     // whole-list replace; the list is tiny
}
```

  Whole-list `Save` is simpler and safer than Add/Remove with read-modify-write; the model owns the in-memory list and writes a copy via `saveWatchCmd`.
- **Concurrent instances:** last write wins. Acceptable for a personal tool; do not build locking.

---

## Patterns to Follow

### Pattern 1: Consumer-side interfaces + fakes
`ui` declares `Fetcher` and `WatchStore`; tests inject fakes returning canned `domain.Result`. `Update` can be tested as a pure function: feed messages, assert model state and returned Cmd types. No terminal, no network.

### Pattern 2: Generation counter for async results
Every request-producing action carries `gen`; `Update` drops messages whose `gen` is not current. Cheaper and less error-prone than context cancellation plumbing through Bubble Tea.

### Pattern 3: Wire → domain mapping as a single seam, golden-tested
Record real GraphQL responses (including null rollup, null reviewDecision, NOT_FOUND aliases, truncated contexts, StatusContext-only commits) into `internal/github/testdata/*.json`. Table tests assert the mapped `domain.PR`. This is where regressions from GitHub schema drift show up, in CI, not on the user's screen.

### Pattern 4: Last-good-snapshot rendering
A refresh error never blanks the screen. The status bar carries the error and a staleness indicator; the table keeps the previous data.

### Pattern 5: Record-and-replay dev loop
A hidden `prv --dump` (or `--once --json`) flag prints the mapped result without starting the TUI. It validates the fetch layer against live GitHub before any UI exists and generates the golden fixtures.

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: API types in the view
**What:** `ui` rendering `githubv4.PullRequest` or wire structs directly.
**Why bad:** every GitHub schema or query tweak ripples through view code; null-handling is scattered; tests need JSON.
**Instead:** `domain.PR` only; `map.go` is the only translator.

### Anti-Pattern 2: Blocking or mutating in the wrong place
**What:** network/disk calls in `Update`/`View`; goroutines or Cmds that write to model fields.
**Why bad:** freezes the UI or races with rendering.
**Instead:** Cmds capture copies and return messages.

### Anti-Pattern 3: N+1 requests
**What:** one search for the list, then one request per PR for checks/reviews (what you would do with the REST API, and what gh-dash does for details).
**Why bad:** burns rate limit and makes refresh slow. A 20-PR list becomes 40+ calls per tick.
**Instead:** one GraphQL document with everything; detail view reads from the snapshot.

### Anti-Pattern 4: Fixed-clock ticks that stack
**What:** `tea.Every` or a tick that fires a fetch regardless of whether the previous one finished.
**Why bad:** overlapping fetches, out-of-order results overwriting newer data.
**Instead:** re-arm after completion + `gen`.

### Anti-Pattern 5: Index-based cursor
**What:** cursor = row number.
**Why bad:** the selected PR changes whenever the list re-sorts or an item appears/disappears during a poll.
**Instead:** track the selected `PRRef`.

### Anti-Pattern 6: Persisting PR data or auto-pruning the watch list
**What:** caching titles/state in the watch file; removing refs that returned NOT_FOUND.
**Why bad:** stale display; a transient 404/SSO hiccup would permanently delete the user's entries.
**Instead:** persist refs only; show an error row; let the user remove.

### Anti-Pattern 7: Recomputing the CI badge from a truncated check list
**What:** deriving pass/fail from `contexts(first:N)` nodes.
**Why bad:** wrong when `totalCount > N`.
**Instead:** badge from `statusCheckRollup.state`; per-check list is for display only.

### Anti-Pattern 8: Using v1 Bubble Tea idioms or `bubbles/list` for the main view
See the v2 table and the View decomposition section.

---

## Scalability Considerations

| Concern | Typical (10 to 30 PRs) | Heavy (100 PRs) | Beyond (hundreds) |
|---------|-----------------------|-----------------|-------------------|
| Requests per refresh | 1 | 1 to 2 (search pagination) | chunked aliases, 4+ |
| Rate-limit points/hour at 30 s poll | ~120 to 240 of 5,000 | ~360 to 720 | raise interval; consider `nodes(ids:)` |
| Render cost | negligible | negligible | add row windowing (already scroll-offset based) |
| Payload size | ~50 KB | ~300 KB | trim `contexts` to `first: 50`, drop `startedAt` etc. |
| Poll interval floor | enforce ≥ 15 s in config validation | same | same |

The design does not need a cache layer, database, or worker pool at any realistic size.

---

## Suggested Build Order (dependencies)

```
domain ──► store ──┐
   │               ├──► ui (fake data) ──► ui (live fetch+poll) ──► detail ──► add/remove/filter ──► polish
   └──► github ────┘
         (auth → query → wire → map → --dump)
```

| Order | Component | Depends on | Exit criterion |
|-------|-----------|-----------|----------------|
| 1 | `domain` types, `ParseRef`, rollup classify, sort/filter | none | table tests pass; URL and `owner/repo#N` parse |
| 2 | `store` (watch list) + `config` | domain | round-trip, atomic write, corrupt-file safety tested |
| 3 | `github`: auth, query builder, wire, map, `--dump` | domain | `prv --dump` prints the user's real PRs; golden fixtures recorded; null/NOT_FOUND/truncation cases covered |
| 4 | `ui` skeleton: root model, list view, styles, key map, alt screen, resize — driven by **fake** `Fetcher` | domain | list renders badges for fixture data; cursor stable across re-sort |
| 5 | Live wiring: `fetchCmd`, `tickCmd`, gen counter, status bar, error/stale handling, rate-limit backoff | 3, 4 | auto-refresh visible; failure keeps last data; no overlapping fetches |
| 6 | Detail view | 5 | per-check list with workflow names, reviewers, truncation note |
| 7 | Watch management: add prompt (validate by fetching), remove, persist; state filter chips; closed-window honored | 2, 5 | add by URL persists across restart; remove works; filter toggles instantly |
| 8 | Polish: open-in-browser, help overlay, narrow-terminal behavior, manual refresh, `--version` | any | all PROJECT.md "Active" requirements checked |

Roadmap implications:
- Steps 1 to 3 are UI-free and fully testable; they retire the main technical risk (query shape, auth, partial errors) before any Bubble Tea code. They can be one phase.
- Step 4 can proceed in parallel with step 3 if a fake `Fetcher` returns fixtures; the interface boundary is what makes that possible.
- Steps 5 to 7 are the phases where user-visible value lands; the "Core Value" (at-a-glance approved/blocked/failing) is delivered at the end of step 5.

---

## Open Questions / Flags for Requirements and Phase Research

1. **Manual PRs vs closed window** (see Domain Model): always show manual PRs, or apply the window to them too? Recommendation: always show.
2. **Token precedence**: spec (gh first) vs go-gh default (env first). Recommendation: follow spec explicitly.
3. **Opening the browser**: use `cli/browser` or exec `open`/`xdg-open`. Not researched (LOW); trivial to decide at implementation.
4. **`ISSUE_ADVANCED` stability**: verified working today, but its open-branch (`is:open`) combined with `OR` was not exercised with live open PRs. Re-verify in step 3 with real open PRs; the 4-alias fallback is specified above.
5. **Test harness for the TUI**: `teatest` v2 compatibility not verified. Default plan is to test `Update` directly and golden-test `View()` output; evaluate `teatest` only if needed.
6. **Detail layout**: full-screen swap (v1) vs side pane; purely a presentation decision, no architectural impact.
7. **Typed GraphQL client vs raw envelope**: whether `go-gh`'s `GraphQLClient.Do` still decodes `Data` on partial errors was not verified; the raw-envelope approach recommended here does not depend on it.

---

## Sources

- Live GraphQL verification against `api.github.com/graphql` with the local `gh` token (2026-10-11): `ISSUE_ADVANCED` OR/parenthesis search, legacy `ISSUE` returning 0 for the same query, `closed:>=` matching merged PRs, aliased `repository.pullRequest` lookups with NOT_FOUND partial errors, `statusCheckRollup`/`contexts` shapes on cli/cli, null `reviewDecision`/rollup, rateLimit costs, `PullRequest.statusCheckRollup` shortcut (HIGH)
- Bubble Tea v2 upgrade guide: https://raw.githubusercontent.com/charmbracelet/bubbletea/main/UPGRADE_GUIDE_V2.md (HIGH)
- Go module proxy latest versions: `charm.land/bubbletea/v2` v2.1.0, `charm.land/bubbles/v2` v2.2.1, `charm.land/lipgloss/v2` v2.0.6, `github.com/cli/go-gh/v2` v2.16.2 (HIGH)
- gh-dash repository tree and `go.mod` (via `gh api repos/dlvhdr/gh-dash`): `internal/{config,data,git,tui/{components,context,constants,keys,common}}`, deps `bubbletea/v2 v2.0.2`, `bubbles/v2 v2.0.0`, `go-gh/v2 v2.13.0`, `cli/shurcooL-graphql` (HIGH for structure; internals of individual components not read)
- go-gh docs via Context7 (`/cli/go-gh`): `TokenForHost` precedence (GH_TOKEN, GITHUB_TOKEN, config file, keyring), `ClientOptions`, `DefaultGraphQLClient` (MEDIUM-HIGH)
- Bubble Tea docs via Context7 (`/websites/pkg_go_dev_github_com_charmbracelet_bubbletea`): returned v1-era examples; used only to confirm `Init`/`Update` shape (note: v1 content, superseded by the upgrade guide)

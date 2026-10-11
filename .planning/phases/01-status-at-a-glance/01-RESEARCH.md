# Phase 1: Status at a Glance - Research

**Researched:** 2026-10-11
**Domain:** Go + Bubble Tea v2 terminal UI over the GitHub GraphQL API (auth, one discovery query, review/CI derivation, list rendering)
**Confidence:** HIGH (query shape, review derivation, token order, rollup behaviour and v2 APIs were executed live or read from source this session; gaps are listed in the Assumptions Log)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Row look & badges**
- **D-01:** Each badge is a glyph plus a short word, with ANSI colour added on top. Examples: `✓ approved`, `✗ changes`, `– none`, `✓ passing`, `✗ failing`, `● running`. A badge must still read correctly when colour is stripped.
- **D-02:** Columns are identity first: `PR` (owner/repo#n), `TITLE`, `AUTHOR`, `STATE`, `REVIEW`, `CI`. TITLE is the flexible column that gets truncated first.
- **D-03:** AUTHOR shows `me` when the author is the viewer, and the login otherwise. The query must therefore get `viewer { login }`. It is one cheap field, so add it to the discovery request.
- **D-04:** A draft PR has its whole row dimmed and carries a `draft` tag, so it reads without colour too. Its STATE stays the true state (open).
- **D-05:** The palette uses the 16 ANSI colours only (terminal-theme red/green/yellow, plus faint/dim). No hex colours and no background detection. It must read on light and dark themes and inside multiplexers.

**Ordering (interim, until the Phase 2 attention sort)**
- **D-06:** Sort by state group (Open, then Merged, then Closed), then `updatedAt` newest first. The final tiebreak is owner/repo#n, so the order is deterministic. Phase 2 may reuse this as the secondary key of the attention sort.
- **D-07:** Merged and closed rows are dimmed like drafts. There is no divider and there are no section headers.

**Launch & failure states**
- **D-08:** The TUI enters the alt screen at once and shows a spinner with "Fetching your PRs…" during the first fetch. `q` and `ctrl+c` work while the fetch runs.
- **D-09:** When the initial fetch fails (network, 401, 5xx, GraphQL error), the TUI shows an error panel with an actionable hint (for example, 401 → "run `gh auth login`"). `r` retries and `q` quits. **Scope note:** this pulls a minimal `r` into Phase 1, only for the error screen. Phase 2 (RFSH-02) generalises it into refresh-anytime and single-flight.
- **D-10:** With zero PRs, show a friendly empty state that names the account: "No open PRs authored by or assigned to @login, and none closed in the last 14 days." This makes a wrong-account token obvious.
- **D-11:** With no token (AUTH-02), print to stderr before the TUI starts: "prv needs a GitHub token. Run `gh auth login`, or set GH_TOKEN / GITHUB_TOKEN." Then exit with code 1.

**Authored vs assigned**
- **D-12:** There is no provenance column. The AUTHOR column implies it: `me` means authored, and another login means assigned. A PR that is both shows `me`.
- **D-13:** A draft PR shows its true REVIEW and CI values. The dimmed row plus the `draft` tag is the only treatment.
- **D-14:** The CI badge follows `statusCheckRollup.state` as GitHub reports it (FAILURE/ERROR → failing, PENDING/EXPECTED → running, SUCCESS → passing). A null rollup → `no checks`. CI must match github.com (success criterion 4), so there is no client-side override of the precedence.

**Carried forward (already decided upstream; do not re-litigate)**
- Token resolution: go-gh `auth.TokenForHost("github.com")`, which checks GH_TOKEN, then GITHUB_TOKEN, then the gh login. The token is never logged, shown or persisted (`LogIgnoreEnv: true`).
- Discovery: a single `ISSUE_ADVANCED` search `is:pr (author:@me OR assignee:@me) (is:open OR closed:>=DATE)` with a fixed 14-day window, deduplicated by node ID. Fallback is separate queries or `viewer.pullRequests`, if spike Q4/Q8 shows missing PRs.
- Review: use `reviewDecision` when it is set. Otherwise derive from `latestOpinionatedReviews`. A null decision never maps to "review required". Spike Q1 checks the comment-after-approve case.
- One total check-state mapping covers CheckRun and StatusContext, with an unknown case. CANCELLED, NEUTRAL and SKIPPED are never red.
- Strip control characters and ANSI from remote strings. No I/O in Update/View. Read the raw GraphQL envelope (partial data plus `errors[]`).

### Claude's Discretion
- The exact glyph set and wording for each badge value, within D-01 and D-05
- Spinner style, header/status-line content and the cursor/selection highlight style
- Column widths and truncation rules at a normal width (full narrow-pane work is Phase 3, NAV-05)
- The `prv --version` output format (ldflags `main.version` with a `debug.ReadBuildInfo()` fallback)
- How partial results (data plus `errors[]`) appear in Phase 1, if the spike shows they happen during discovery

### Deferred Ideas (OUT OF SCOPE)
None. The discussion stayed within phase scope. (A minimal `r` retry on the error screen was pulled forward deliberately; see D-09.)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| AUTH-01 | Same token `gh` would use: `GH_TOKEN`, `GITHUB_TOKEN`, then `gh` login | go-gh `TokenForHost` order read from source and confirmed by hash comparison (Spike Q5). Pass the result as `ClientOptions.AuthToken`. |
| AUTH-02 | No token: exit with a clear login message | `TokenForHost` returns `"", "default"`; main prints D-11 text to stderr and exits 1 before `tea.NewProgram`. A present-but-invalid token is only detectable at fetch time (401), handled by the D-09 error panel. |
| AUTH-03 | Token never logged, shown or written | `LogIgnoreEnv: true` verified silent under `GH_DEBUG=api` (0 bytes on stderr). Errors name the source (`env`/`gh`), never the value. See Security Domain. |
| CONF-02 | `go install .../cmd/prv@latest` works; `prv --version` prints version | Module path and `cmd/prv` layout; `go.mod` floor must be set to `go 1.26.0` by hand (`go mod init` writes `go 1.27.2` here). `--version` via `debug.ReadBuildInfo()` verified on this toolchain. |
| WATCH-01 | Authored PRs appear automatically | Single `ISSUE_ADVANCED` search with `author:@me`; verified: 15 results on the user's account. |
| WATCH-02 | Assigned PRs appear automatically | `assignee:@me` in the same search; union verified on a stand-in (268 = 265 + 144 - 141). The user's own account has 0 assigned PRs, so this is stand-in evidence only. |
| WATCH-03 | A PR both authored and assigned appears once | Server-side union already de-duplicates (100 unique ids in 100 nodes); still de-duplicate by node `id` in the mapper (cheap, and required when pagination or a fallback query is used). |
| LIST-01 | Row shows repo, number, title, author, draft marker | Fields `repository.nameWithOwner number title author{login} isDraft` + `viewer.login` for `me` (D-03). |
| LIST-02 | Row shows Open / Merged / Closed | `state` enum is `OPEN`, `CLOSED`, `MERGED` (introspected). |
| LIST-03 | Review status incl. no false "review required" | `DeriveReview` below. Evidence: 49 of ~350 sampled PRs had null `reviewDecision` with real approvals; derivation never contradicted a non-null decision. |
| LIST-05 | One CI badge; no checks is never "passing" | `ClassifyRollup` below; null rollup is `no checks` (all 15 of the user's PRs have a null rollup). |
| NAV-01 | j/k and arrows | `msg.String()` returns `"j"`, `"k"`, `"up"`, `"down"` for v2 `KeyPressMsg` (verified by running it). |
| NAV-04 | Quit with `q` | `q` and `ctrl+c` (`"ctrl+c"` in v2) handled in `Update` in every state (D-08). |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

From `/Users/scottb/dev/bartram/prv/.claude/CLAUDE.md` (project) and `~/.claude/CLAUDE.md` (global). Treat as locked.

**Stack (project):**
- Use `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`. Never the v1 paths (`github.com/charmbracelet/bubbletea|lipgloss|bubbles`).
- v2 API: `View()` returns `tea.View` (`tea.NewView(s)`, `v.AltScreen = true`); no `tea.WithAltScreen()`; keys are `tea.KeyPressMsg` and space is `"space"`; Bubbles v2 uses `SetWidth()/Width()` methods; `lipgloss.Color()` is a function.
- GitHub: go-gh `api.GraphQLClient.DoWithContext` with hand-written query strings and typed `json`-tagged structs; `api.ClientOptions{Host: "github.com", Timeout: 20s, LogIgnoreEnv: true}`; no `EnableCache`.
- Auth: `auth.TokenForHost("github.com")` once at startup; token passed explicitly via `ClientOptions.AuthToken`. No token: clear "run `gh auth login` or set GITHUB_TOKEN" message, exit non-zero before the TUI starts. Never persist the token.
- List rendering: `lipgloss/table` with an own cursor index (not `bubbles/table`, not `bubbles/list`). Use `viewport` only if it helps scrolling.
- CLI: stdlib `flag` (no Cobra). Logging: `log/slog` to a file only if `--debug` exists; never `fmt.Println`/`log.Println` to stdout/stderr while the TUI runs.
- Tests: stdlib `testing`, pure `Update`/`View` tests, golden files via `charmbracelet/x/exp/golden`; teatest/v2 only for a few end-to-end flows.
- Go floor `go 1.26`; single static binary (`CGO_ENABLED=0`); module path `github.com/sbartram/prv`; layout `cmd/prv`.
- github.com only; auto-refresh stays inside rate limits (Phase 2).
- GSD workflow: edits go through a GSD command (this is `/gsd-plan-phase` then `/gsd-execute-phase`).

**Global (user):**
- Prefer the serena plugin (or an LSP) over grep for code navigation.
- Non-trivial work on a feature branch or worktree by default; merges to main use `--no-ff`; push only when the user asks. Subagents stay on the feature branch (no detached HEAD).
- Handoff skill for pause/resume.
- Minimal code; no speculative abstractions; surgical changes.

## Summary

Phase 1 is a thin vertical slice: resolve a token, run one GraphQL search, map it to a small domain model, and render a navigable list in Bubble Tea v2. The live-verification spike (full record in `01-SPIKE.md`) confirmed every default the earlier research had left open. A single `ISSUE_ADVANCED` search with `(author:@me OR assignee:@me) (is:open OR closed:>=DATE)` returns the exact union, already de-duplicated, at cost 1, including the open branch; the legacy `ISSUE` type silently returns 0 for the same OR query. `latestOpinionatedReviews` correctly ignores a later COMMENTED review, and, critically, `reviewDecision` is null while approvals exist on repos without a required-review rule (49 of about 350 sampled PRs), so the fallback derivation is mandatory and matches GitHub's own decision in every non-null case sampled.

Two spike results change how the carried-forward rules should be read. First, GitHub's rollup state treats `CANCELLED` check runs as `FAILURE` and lets `FAILURE` win over in-progress runs, so under D-14 (follow GitHub, no override) a cancelled-only PR shows "failing"; the "CANCELLED is never red" rule belongs to per-check display in Phase 3. Second, go-gh's `DoWithContext` populates the typed response before returning `*api.GraphQLError`, so partial data is usable without bypassing the client (this supersedes the architecture note that advised avoiding the typed path). The user's own account has zero open and zero recently closed PRs, so a live run shows the empty state (D-10); human verification of rows needs a wider window or fixtures.

**Primary recommendation:** Build the tracer first (go.mod floor 1.26.0, `cmd/prv`, `--version`, `TokenForHost`, the proven query from `01-SPIKE.md` through `DoWithContext`), keep `DeriveReview`, `ClassifyRollup`, sanitising and sorting as pure functions in `internal/domain` with fixture tests, and render with an own-cursor list whose column widths are computed explicitly (the default `lipgloss/table` auto-sizing does not truncate TITLE first).

## Architectural Responsibility Map

prv is a single local binary, so "tiers" are its internal layers plus the remote API.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Token resolution, no-token exit, `--version` | Process bootstrap (`cmd/prv`) | `internal/github` (auth helper) | Must run before the TUI starts so errors print to a normal terminal (D-11). |
| Discovery query, pagination, wire decoding | GitHub client (`internal/github`) | — | Owns query text, unexported wire structs and the single wire-to-domain mapper. |
| Review derivation, CI classification, sort, 14-day filter, string cleaning | Domain (`internal/domain`) | — | Pure, I/O-free functions; the highest-risk logic, so fixture-testable without Bubble Tea or the network. |
| Loading / error / empty / list state machine, cursor, key handling | TUI model (`internal/ui`) | — | Pure `Update`; all I/O is a `tea.Cmd` that returns a message. |
| Rendering rows, badges, colours, truncation | TUI view (`internal/ui`) | — | Pure `View`; ANSI-16 styles; widths via display-width helpers. |
| Remote API / rate limit / auth validity | GitHub API (external) | — | Authoritative for state, reviews and rollup; prv shows what it reports. |
| Persistence | None in Phase 1 | — | Watch list store and config are Phase 4 and Phase 2. Do not create them now. |

## Standard Stack

### Core (Phase 1 only)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/bubbletea/v2` | v2.1.0 (2026-10-08, `go 1.26.0`) | TUI runtime | Locked by project stack. [VERIFIED: proxy.golang.org via `go list -m -json`] |
| `charm.land/bubbles/v2` | v2.2.1 (2026-08-24) | `spinner` (loading), `key` (keymap) | Locked. Only `spinner` and `key` are needed. [VERIFIED: proxy.golang.org] |
| `charm.land/lipgloss/v2` | v2.0.6 (2026-08-11) | Styles (ANSI-16), `lipgloss.Width`, optional `lipgloss/table` | Locked. [VERIFIED: proxy.golang.org] |
| `github.com/cli/go-gh/v2` | v2.16.2 (2026-10-09) | `api.GraphQLClient`, `auth.TokenForHost` | Locked. Source read this session. [VERIFIED: proxy.golang.org; go-gh source `pkg/auth/auth.go`, `pkg/api/graphql_client.go`] |
| `github.com/charmbracelet/x/ansi` | selected by `go mod tidy` (v0.11.9 in the spike) | `ansi.Strip`, `ansi.Truncate`, `ansi.StringWidth` | Already a transitive dependency of lipgloss; import it directly for sanitising and width-safe truncation. [VERIFIED: ran in spike] |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/charmbracelet/x/exp/golden` | v0.1.0 (2026-09-24) | `golden.RequireEqual(t, out)` with `-update` flag | Golden tests of `View()` for loading, error, empty and list states. [VERIFIED: source read, `golden.go` lines 16-27] |
| stdlib `flag`, `log/slog`, `net/http/httptest`, `encoding/json`, `runtime/debug` | go 1.26 | `--version`, fixtures, build info | Per project stack. |

Not needed in Phase 1 (do not add): `BurntSushi/toml` (CONF-01 is Phase 2), `go-gh/pkg/browser` (NAV-02 is Phase 2), `teatest/v2` (optional spike only), `testify`, Cobra, Viper.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `lipgloss/table` | Hand-rendered rows (`lipgloss.NewStyle().Width(n)` per cell + `JoinHorizontal`) | The table's auto-sizing shrinks all columns proportionally (spike: AUTHOR header became `AUT…` while TITLE still had room). Either use the table with explicit per-column `Width` in `StyleFunc`, or render rows by hand. Both are deterministic; pick one and golden-test it. |

**Installation:**
```bash
go mod init github.com/sbartram/prv
go mod edit -go=1.26.0            # go mod init writes "go 1.27.2" on this machine; that would break go install for Go 1.26 users
go get charm.land/bubbletea/v2@v2.1.0 charm.land/bubbles/v2@v2.2.1 charm.land/lipgloss/v2@v2.0.6 \
       github.com/cli/go-gh/v2@v2.16.2 github.com/charmbracelet/x/exp/golden@v0.1.0
go mod tidy
```

**Version verification:** all six versions above were confirmed against the module proxy on 2026-10-11 (`go list -m -json <mod>@latest`), and their origin repos resolved to `github.com/charmbracelet/*` and `github.com/cli/go-gh`. [VERIFIED: proxy.golang.org]

## Package Legitimacy Audit

> The GSD legitimacy seam (`gsd-tools query package-legitimacy check`) accepts only `npm|pypi|crates`; it rejects `--ecosystem go` ("Usage: ... --ecosystem <npm|pypi|crates>"). The check could not be run for Go modules. As a substitute, each module was resolved on proxy.golang.org and its `Origin.URL` read. All modules come from the user's locked stack in `.claude/CLAUDE.md`, not from web search.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| charm.land/bubbletea/v2 | Go proxy | v2 line since 2026 (v2.1.0 on 2026-10-08) | n/a (proxy does not publish) | github.com/charmbracelet/bubbletea | seam n/a | Approved (locked stack, origin verified) |
| charm.land/bubbles/v2 | Go proxy | v2.2.1 on 2026-08-24 | n/a | github.com/charmbracelet/bubbles | seam n/a | Approved |
| charm.land/lipgloss/v2 | Go proxy | v2.0.6 on 2026-08-11 | n/a | github.com/charmbracelet/lipgloss | seam n/a | Approved |
| github.com/cli/go-gh/v2 | Go proxy | v2.16.2 on 2026-10-09 | n/a | github.com/cli/go-gh | seam n/a | Approved |
| github.com/charmbracelet/x/exp/golden | Go proxy | v0.1.0 on 2026-09-24 | n/a | github.com/charmbracelet/x | seam n/a | Approved (test-only) |
| github.com/charmbracelet/x/ansi | Go proxy | transitive of lipgloss | n/a | github.com/charmbracelet/x | seam n/a | Approved |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none. The seam could not rate Go modules; no checkpoint is required because every module is in the user's locked stack and its origin repo is the official one.

## Spike Results That Drive Planning

Full evidence is in `01-SPIKE.md`. The decisions, in one place:

| Item | Result | Planning consequence |
|------|--------|----------------------|
| C2 discovery | Single `ISSUE_ADVANCED` search confirmed (union exact, de-duplicated, cost 1, open branch works on a stand-in). Legacy `ISSUE` returns 0 silently. | Hard-code `type: ISSUE_ADVANCED`. Add a test asserting the query string contains it. No fallback queries needed in Phase 1. |
| C4 review fallback | `latestOpinionatedReviews` keeps APPROVED after a later COMMENTED; `latestReviews` does not. `reviewDecision` is null with approvals in 49 of ~350 sampled PRs. | Use `DeriveReview` below. Never request `latestReviews` for status. |
| C1 token order | go-gh: `GH_TOKEN`, `GITHUB_TOKEN`, `hosts.yml oauth_token`, then `gh auth token --secure-storage --hostname github.com` (ignores env). | Use `TokenForHost` unchanged. 401 hint must mention env tokens (a stale `GH_TOKEN` beats a valid `gh` login). |
| Partial data | `DoWithContext` fills the struct then returns `*api.GraphQLError`. | Use the typed client; on `GraphQLError` keep the decoded data and surface a warning. |
| Rollup precedence | `CANCELLED` and `FAILURE` produce rollup `FAILURE`, even beside in-progress runs; `PENDING` only when running with no failure. | D-14 holds as written. Document that cancelled-only shows "failing". |
| Cost | Lean list query for 25 PRs: cost 1, 5.7 KB. Eager contexts: cost 1, 165 KB. | Phase 1 uses the lean query (rollup `state` only). Cost is not a constraint for Phase 3's C3 choice. |
| `isRequired` | Needs a PR number or id; fails inside a search list. | Do not use. |
| Window | `closed:>=DATE` is date-granular. | Query margin plus exact client-side filter (see Pattern 3). |

**Not verified (carry as risks, see Assumptions Log):** SAML/SSO omission and FORBIDDEN shape (no SAML org on the account), `assignee:@me` and `is:open` on the user's own account (none exist), re-run de-duplication.

## Architecture Patterns

### System Architecture Diagram

```
 prv [--version]
   │
   ▼
 cmd/prv/main.go ── flag parse ──► --version? ──► print version, exit 0
   │
   ├─ auth.TokenForHost("github.com") ── token == "" ──► stderr: D-11 message, exit 1
   │          (token held in memory only, passed as ClientOptions.AuthToken)
   │
   ├─ github.NewClient(token)  ──────────────────────────────┐ implements ui.Fetcher
   │                                                          │
   └─ tea.NewProgram(ui.New(fetcher), tea.WithContext(ctx)).Run()
          │
          ▼
   ┌─────────────── ui.Model (pure Update/View) ────────────────┐
   │ state: loading ──fetchedMsg(ok)──► list | empty             │
   │          │  ▲                                               │
   │          │  └──── r (only in error) ◄── error ◄─fetchedMsg(err)
   │          └─ q / ctrl+c in every state ──► tea.Quit          │
   │  Init(): Batch(spinner.Tick, fetchCmd)                      │
   └───────────────────────────┬─────────────────────────────────┘
                               │ fetchCmd (tea.Cmd, runs off the Update loop)
                               ▼
        github.Client.Fetch(ctx)
          │  build q: is:pr archived:false (author:@me OR assignee:@me)
          │           (is:open OR closed:>=<now-15d UTC>) sort:updated-desc
          │  loop pages (≤5): DoWithContext(query, {q, after}, &wire)
          │      *api.GraphQLError ──► keep wire data + warnings
          │      *api.HTTPError / net error ──► typed error with Hint()
          ▼
        wire structs ──map──► domain.PR{Ref, Title, Author, IsDraft, State, Review, CI, UpdatedAt, ClosedAt}
          │   skip nil nodes / non-PullRequest; dedupe by node id;
          │   Clean() every remote string; exact 14-day filter on closedAt
          ▼
        domain.Snapshot{Viewer, PRs (sorted by D-06), Truncated, Warnings}
```

### Recommended Project Structure

```
cmd/prv/
  main.go                 # flag, token, wiring, exit codes; var version = "dev"
internal/domain/
  pr.go                   # PR, Ref, State, Review, CI types; Snapshot
  review.go               # DeriveReview
  ci.go                   # ClassifyRollup
  sort.go                 # Sort (D-06) + InWindow (14-day filter)
  clean.go                # Clean (remote-string sanitiser)
  *_test.go               # table tests, no I/O
internal/github/
  client.go               # NewClient(token), Fetch(ctx) (Snapshot, error); paging loop
  query.go                # const discoveryQuery; BuildSearch(now) string
  wire.go                 # unexported wire structs
  map.go                  # wire -> domain (the single translation seam)
  errors.go               # typed errors + Hint() (auth, network, rate/SSO, server, graphql)
  testdata/*.json         # synthetic response fixtures
  *_test.go               # httptest-backed or fake-doer tests
internal/ui/
  model.go                # Model, Init, Update, key handling, states
  view.go                 # View: loading, error, empty, list; header/footer
  styles.go               # ANSI-16 styles, badge rendering
  columns.go              # explicit width allocation
  *_test.go + testdata/*.golden
```

`internal/ui` defines its own `Fetcher` interface and never imports `internal/github`. Do not create `store`, `config` or `browser` packages in this phase.

### Pattern 1: Review derivation (pure, total)

**What:** `reviewDecision` wins when non-null; otherwise derive from `latestOpinionatedReviews`; null never means "review required".
**Evidence:** In ~350 sampled PRs the derivation matched every non-null APPROVED (125) and CHANGES_REQUESTED (51) decision; 78 REVIEW_REQUIRED PRs had no opinionated reviews; 49 null-decision PRs had approvals; 47 null-decision PRs had none.
```go
// Source: derived from live spike data (01-SPIKE.md); compiled and run against the live API this session.
func DeriveReview(decision string, states []string) Review {
	switch decision { // "" when reviewDecision is null
	case "APPROVED":          return ReviewApproved
	case "CHANGES_REQUESTED": return ReviewChangesRequested
	case "REVIEW_REQUIRED":   return ReviewRequired
	}
	var approved bool
	for _, s := range states { // latestOpinionatedReviews[].state
		switch s {
		case "CHANGES_REQUESTED": return ReviewChangesRequested // wins over approvals
		case "APPROVED":          approved = true
		}
	}
	if approved { return ReviewApproved }
	return ReviewNone // null decision, no opinion: NEVER ReviewRequired
}
```
Unknown `decision` strings fall through to derivation (safe). Fixtures needed (each is a separate test case): null+approved, null+none, null+changes-requested (synthetic: not observed live), REVIEW_REQUIRED, APPROVED, CHANGES_REQUESTED, approved-then-commented (opinionated=APPROVED), two approvals, CHANGES_REQUESTED+APPROVED mixed (decision=CHANGES_REQUESTED; observed live 3 times), REVIEW_REQUIRED on a CLOSED PR (observed live).

### Pattern 2: CI classification (D-14, total with unknown case)

```go
func ClassifyRollup(rollup *string) CI { // statusCheckRollup.state, nil when no checks
	if rollup == nil { return CINone } // null rollup is "no checks", never "passing"
	switch *rollup {
	case "SUCCESS":           return CIPassing
	case "FAILURE", "ERROR":  return CIFailing
	case "PENDING", "EXPECTED": return CIRunning
	}
	return CIUnknown // a new enum value must not read as passing
}
```
`StatusState` enum introspected live: `EXPECTED, ERROR, FAILURE, PENDING, SUCCESS`. Rollup precedence is GitHub's: observed `CANCELLED=1 IN_PROGRESS=181` gives `FAILURE`. Do not recompute from contexts.

### Pattern 3: Discovery query construction and the 14-day window

- Query text is the proven string in `01-SPIKE.md` ("The proven Phase 1 query"). Variables: `q` (and `after` from page 2). Omit the `after` key on the first page.
- Search string: `is:pr archived:false (author:@me OR assignee:@me) (is:open OR closed:>=DATE) sort:updated-desc`.
- `DATE` is date-granular. Compute `DATE = now.UTC().AddDate(0, 0, -15).Format("2006-01-02")` (one extra day of margin), then filter client-side: keep OPEN PRs; keep MERGED/CLOSED only when `closedAt >= now.UTC().Add(-14*24*time.Hour)`. Inject a `now func() time.Time` so tests are deterministic. This makes the "last 14 days" criterion exact whatever time zone the date qualifier uses [ASSUMED A2 on the zone].
- Pagination: `first: 100`, loop while `pageInfo.hasNextPage` up to 5 pages (cost 1 each); if still more, set `Snapshot.Truncated` and show it in the footer. Verified live: cursor pagination returns a second page of 100.
- De-duplicate by node `id`. Skip `nil` nodes and any node whose `__typename != "PullRequest"`.

### Pattern 4: Partial results and error classification (Claude's discretion, recommended)

`DoWithContext` returns `*api.GraphQLError` after filling the struct. Recommended policy:
- `errors.As(err, &*api.GraphQLError)` and `search.nodes` decoded: show the list, set `Snapshot.Warnings`, show a footer note ("some data could not be loaded"). Nodes can be `null` when a nested error bubbles up (seen live with `isRequired`), so always nil-check.
- `*api.GraphQLError` with no search data: error panel.
- `*api.HTTPError` 401: hint "GitHub rejected the token. Run `gh auth login`. If GH_TOKEN or GITHUB_TOKEN is set, it overrides the gh login: fix or unset it." (verified: a bad token yields `HTTPError` status 401 "Bad credentials" with no fallback to the gh login.)
- 403 / 429: hint about rate limit or SSO; 5xx: "GitHub is having problems, press r to retry"; network/timeout (`net.Error`, `context.DeadlineExceeded`): "check your connection".
- Never include the token or a request dump in any error text.

### Pattern 5: Bubble Tea v2 model (states, quit, cancel)

```go
// Source: compiled against charm.land/bubbletea/v2 v2.1.0 and bubbles/v2 v2.2.1 in the spike.
type fetchedMsg struct{ snap domain.Snapshot; err error }

func (m Model) Init() tea.Cmd { return tea.Batch(m.spin.Tick, m.fetchCmd()) }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c": return m, tea.Quit // every state, incl. while loading (D-08)
		case "r": if m.state == stateError && !m.loading { /* set loading, return fetchCmd + spin.Tick */ }
		case "j", "down": /* move cursor, clamp */
		case "k", "up":   /* move cursor, clamp */
		}
	case tea.WindowSizeMsg: m.w, m.h = msg.Width, msg.Height // width/height 0 means "not sized yet"
	case spinner.TickMsg:   var c tea.Cmd; m.spin, c = m.spin.Update(msg); return m, c
	case fetchedMsg:        /* loading=false; state = error | empty | list */
	}
	return m, nil
}

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true // alt screen is declared here in v2 (no WithAltScreen option)
	return v
}
```
- Key strings verified by running `KeyPressMsg.String()`: `"up" "down" "j" "k" "q" "ctrl+c" "space" "home" "end" "g" "G" "pgdown"`.
- `fetchCmd` closes over a `context.Context` created in `main` (`ctx, cancel := context.WithCancel(...)`; `tea.WithContext(ctx)`; `defer cancel()`), so quit cancels an in-flight request. The `loading` flag makes `r` a no-op while a fetch runs; a generation counter is Phase 2 work (RFSH-02).
- Spinner: `spinner.New(spinner.WithSpinner(spinner.Dot))`; its `Tick` is a `tea.Cmd`-compatible method value. `spinner.Model` ignores ticks from other spinners (ID check), so one spinner is safe.
- Panics: `Program.Run` catches them and returns an error wrapping `ErrProgramKilled`/`ErrProgramPanic` (tea.go lines ~1056). Still wrap the fetch `tea.Cmd` in a `recover` that returns a `fetchedMsg{err}` so a panic shows the error panel instead of a dead terminal.
- After `Run()` returns an error, print it to stderr (the alt screen has been left) and exit 1. A user quit exits 0.

### Pattern 6: Rendering rows (D-01 to D-07)

- Suggested badge text (all within D-01/D-05; widths from `lipgloss.Width`): REVIEW `✓ approved` (10), `✗ changes` (9), `○ required` (10), `– none` (6); CI `✓ passing` (9), `✗ failing` (9), `● running` (9), `– none` or `no checks` (6/9), `? unknown`. STATE `open`, `merged`, `closed`.
- Colour: green for approved/passing, red for changes/failing, yellow for running/required, faint for `none`, merged and closed (D-07) and drafts (D-04). Use `lipgloss.Green`, `lipgloss.Red`, `lipgloss.Yellow` (ANSI 4-bit constants, verified in `color.go`) and `Faint(true)`. No hex, no `LightDark`.
- Draft: prefix the TITLE cell with `[draft] ` (the prefix survives truncation because it is at the front) and dim the whole row.
- AUTHOR: `me` when `strings.EqualFold(author, viewer)` else the login; nil author renders `ghost` (the schema allows null authors; not observed live, see A4). Bot authors arrive as `dependabot` (no `[bot]`).
- Selection: reverse video or a `>` marker in a fixed first column; selection must not rely on colour alone.
- Widths (the table pitfall): the default `lipgloss/table` auto-size shrank AUTHOR before TITLE in the spike, so D-02 (TITLE truncates first) will not happen by itself. Compute widths yourself: PR, AUTHOR, STATE, REVIEW, CI get their content max (capped), TITLE gets the remainder with a floor (for example 10), then pass a `Width(n)` per column in `StyleFunc` (in lipgloss v2 `Width` includes padding, so add 1 for the column gap) or hand-render with `ansi.Truncate(s, n, "…")`. Never use `len()`; titles contain emoji and CJK (`lipgloss.Width("你好")` is 4).
- Scrolling: slice the visible rows yourself from `cursor` and the body height, and render only that slice. Avoid the table's own `Height`/`YOffset`: it appends an overflow row and shifts `StyleFunc` row indexes. Treat `m.w == 0 || m.h == 0` as "not sized yet" and render nothing special (no negative-width panics).
- Header and footer are discretionary. Suggested: header `prv · @login · 12 PRs`; footer `j/k move · q quit` plus a truncation/partial-data note when set.
- Sorting (D-06): `stateRank` OPEN=0, MERGED=1, CLOSED=2, then `UpdatedAt` descending, then `repo` ascending, then `number` ascending (compare number numerically, not as text).

### Pattern 7: Remote-string cleaning

Apply `Clean()` once, in the mapper, to title, author login, repo name and any error message from the API:
```go
func Clean(s string) string {
	s = ansi.Strip(s)
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t': return ' '
		case unicode.IsControl(r), r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069: return -1 // C0/C1 + bidi overrides
		}
		return r
	}, s)
}
```
Compiled and run in the spike (`"a\x1b[31mred\x1b[0m\u202eb\tc"` becomes `"aredb c"`). The bidi-range choice is a design recommendation (A3).

### Pattern 8: Version output

```go
var version = "dev" // overridable: -ldflags "-X main.version=1.2.3"

func buildVersion() string {
	if version != "dev" { return version }
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" { return bi.Main.Version }
	return version
}
```
Verified on this toolchain (Go 1.27.2): a local `go build` in a git repo stamps `v0.0.0-<date>-<sha>` untagged, `v0.1.0` (plus `+dirty` if the tree is dirty) when tagged; `-ldflags -X` wins. Handle `--version` before auth so it works with no token and no TTY. Output format is discretionary, for example `prv v0.1.0`.

### Testing approach (no Validation Architecture section: `workflow.nyquist_validation` is `false`)

- `internal/domain`: table tests for `DeriveReview`, `ClassifyRollup`, `Sort`, `InWindow`, `Clean`. No I/O.
- `internal/github`: synthetic JSON fixtures decoded through the real wire structs and mapper. Do not commit real PR data or recorded `Authorization` headers; build fixtures from the shapes in `01-SPIKE.md`. Cases: null rollup, null `reviewDecision` with approval, approve-then-comment, REVIEW_REQUIRED on closed, draft, bot author, nil author, nil node, non-PR node, duplicate id across pages, GraphQL error with partial data, HTTP 401 body, merged and closed inside and outside the window.
- `internal/ui`: drive `Update` with messages and assert on the model; `golden.RequireEqual` on `View().Content` at fixed sizes (80x24, 120x40; plus 0x0 not-sized) for loading, error, empty and list states. Strip ANSI in some goldens or set a fixed colour profile so goldens are stable.
- One manual UAT (end of phase): run on the user's account (expect the empty state), then with a wide window to see real rows (see Open Question 1), and confirm the terminal is restored after `q`.

### Anti-Patterns to Avoid

- **Mapping null `reviewDecision` to review required**, or reading `latestReviews` for status (spike Q1).
- **Recomputing the CI badge from contexts** or from per-check conclusions; use the rollup `state`.
- **`type: ISSUE`** for the OR search: silently 0 results.
- **Using `GH_TOKEN`-less assumptions in the 401 hint:** a stale env token is a real cause.
- **`fmt.Println` or `log` output to stdout/stderr while the TUI runs**; set `LogIgnoreEnv: true` (otherwise `GH_DEBUG` paints over the screen).
- **I/O in `Update`/`View`**, `time.Sleep` pollers, v1 import paths, `tea.KeyMsg`, `case " "`.
- **Creating store/config/browser code now.**
- **`go mod init` without `go mod edit -go=1.26.0`.**

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Token resolution and `gh` keyring lookup | `exec gh auth token` + env code | `auth.TokenForHost("github.com")` | Already ordered `GH_TOKEN`, `GITHUB_TOKEN`, hosts.yml, `gh`; uses `safeexec`; handles `GH_PATH`. |
| HTTP client, headers, timeouts, 401/GraphQL error types | Custom `net/http` GraphQL POST | `api.NewGraphQLClient` + `DoWithContext` | Returns `*api.HTTPError` and `*api.GraphQLError`, fills partial data. |
| ANSI stripping and width-safe truncation | Regex strip, `len()`, byte slicing | `ansi.Strip`, `ansi.Truncate`, `lipgloss.Width` / `ansi.StringWidth` | Emoji, CJK and combining characters break naive width math. |
| Spinner animation | A `time.Ticker` goroutine | `bubbles/v2/spinner` | Tick messages route through the Elm loop; ID-checked. |
| Golden snapshot plumbing | Own file compare and update flag | `golden.RequireEqual` | Handles the `-update` flag and `testdata` layout. |
| Union and de-duplication of authored/assigned | Two searches merged client-side | One `ISSUE_ADVANCED` search | Verified exact union (268 = 265 + 144 - 141). Keep the id-set de-dupe anyway. |
| Check precedence (FAILURE vs PENDING vs CANCELLED) | Re-derive from contexts | GitHub's `statusCheckRollup.state` | GitHub's own result is what "matches github.com" means; also correct when contexts are truncated. |

**Key insight:** every status the user sees is a mapping from GitHub's own enums. Keep that mapping in two small total functions with unknown cases, and let GitHub's computed fields (`reviewDecision`, rollup `state`) win wherever they exist.

## Common Pitfalls

### Pitfall 1: `reviewDecision` null is not "no reviews"
**What goes wrong:** An approved PR in an unprotected repo shows "none" (or worse, "required"). **Why:** GitHub returns null decision with approvals present (49 of ~350 sampled PRs). **How to avoid:** `DeriveReview` fallback. **Warning signs:** a fixture with null decision and an APPROVED review rendering anything but approved.

### Pitfall 2: `latestReviews` hides approvals behind comments
**What goes wrong:** An approve-then-comment reviewer reads as COMMENTED. **How to avoid:** request `latestOpinionatedReviews` only. Verified on 5 of 5 sampled PRs.

### Pitfall 3: Null rollup shown as passing, cancelled shown as passing
**What goes wrong:** Unprotected repos have `statusCheckRollup: null` (all 15 of the user's PRs). **How to avoid:** nil check first. A cancelled-only PR is `FAILURE` per GitHub; show it (D-14).

### Pitfall 4: Stale env token masks the gh login
**What goes wrong:** `GITHUB_TOKEN` from an old shell or CI wins over a valid `gh` login; the user sees 401 and "run `gh auth login`" does nothing. **How to avoid:** 401 hint names the env vars (Pattern 4). Never fall back silently to another source (it would hide which identity is shown); the header shows `@login` so a wrong account is visible (D-10).

### Pitfall 5: `go mod init` writes `go 1.27.2`
**What goes wrong:** `go install ...@latest` fails for anyone on Go 1.26. **How to avoid:** `go mod edit -go=1.26.0` right after init and check `go.mod` after every `go get`/`tidy` (Bubble Tea v2.1.0 itself needs 1.26.0, so 1.26.0 is the floor).

### Pitfall 6: Empty live result on the user's account
**What goes wrong:** The user's account has no open, assigned or last-14-day PRs, so a live run shows the empty state and rows cannot be eyeballed. **How to avoid:** fixture/golden tests for rows; a wider-window manual check (Open Question 1).

### Pitfall 7: The `ISSUE_ADVANCED` `closed:` date and time zones
**What goes wrong:** A PR closed on the boundary day appears or vanishes. **How to avoid:** one-day query margin plus exact client-side `closedAt` filter with an injected clock.

### Pitfall 8: Nulls inside `nodes[]`
**What goes wrong:** When a nested field errors, GraphQL nulls the parent list element (seen live: `contexts.nodes` elements became null). A nil deref panics the mapper. **How to avoid:** use pointer element types in wire structs and skip nil nodes; test with a fixture containing `null` in `search.nodes` and in `latestOpinionatedReviews.nodes`.

### Pitfall 9: `GH_DEBUG` and logging corrupt the screen
**How to avoid:** `LogIgnoreEnv: true` (verified: 0 stderr bytes with `GH_DEBUG=api`). Optional `--debug` slog-to-file is Phase 2 territory; not required now.

### Pitfall 10: Table auto-sizing and wide glyphs
**How to avoid:** explicit column widths (Pattern 6). `● ○ ✓ ✗ –` measure width 1 in `lipgloss`/`ansi`; some terminals in CJK locales draw ambiguous-width glyphs as 2 (A7).

### Pitfall 11: Terminal not restored
**How to avoid:** let `tea.Program` own the terminal; handle `q`/`ctrl+c` as key presses (raw mode delivers ctrl+c as `KeyPressMsg` in v2; an `InterruptMsg` also exists); never `os.Exit` inside `Update`; wrap the fetch Cmd in `recover`.

### Pitfall 12: Token leakage via errors, panics or fixtures
**How to avoid:** errors name the source only; `Authorization` never appears in fixtures; a test asserts a fake token string never appears in any error text or rendered view.

## Code Examples

### Fetch through go-gh with partial-data tolerance (compiled and run live)
```go
// Source: go-gh v2.16.2 pkg/api/graphql_client.go lines 68-110; behaviour verified in the spike.
c, err := api.NewGraphQLClient(api.ClientOptions{
	Host: "github.com", AuthToken: token, Timeout: 20 * time.Second, LogIgnoreEnv: true,
})
var out wireResp
err = c.DoWithContext(ctx, discoveryQuery, vars, &out) // vars: map[string]any{"q": q} (+ "after")
var ge *api.GraphQLError
var he *api.HTTPError
switch {
case errors.As(err, &ge): // out is already populated; ge.Errors[i].{Type,Path,Message}
case errors.As(err, &he): // he.StatusCode, he.Message ("Bad credentials" for 401)
case err != nil:          // network, timeout, ctx cancelled
}
```

### Wire struct (pointers for everything nullable)
```go
type wireResp struct {
	Viewer struct{ Login string }
	RateLimit struct{ Cost, Remaining int; ResetAt string } // parse now, display in Phase 2
	Search struct {
		IssueCount int
		PageInfo   struct{ HasNextPage bool; EndCursor string }
		Nodes []*struct {
			Typename string `json:"__typename"`
			ID, Title, URL, State string
			Number int
			IsDraft bool
			ClosedAt, MergedAt, UpdatedAt string
			Repository struct{ NameWithOwner string }
			Author *struct{ Login string }
			ReviewDecision *string
			LatestOpinionatedReviews struct{ Nodes []*struct{ State string } }
			StatusCheckRollup *struct{ State string }
		}
	}
}
```

### Token resolution in main (no token value ever printed)
```go
token, _ := auth.TokenForHost("github.com") // second return is the source: "GH_TOKEN", "GITHUB_TOKEN", "oauth_token", "gh", or "default"
if token == "" {
	fmt.Fprintln(os.Stderr, "prv needs a GitHub token. Run `gh auth login`, or set GH_TOKEN / GITHUB_TOKEN.")
	os.Exit(1)
}
```

### Golden test
```go
// Source: github.com/charmbracelet/x/exp/golden golden.go (RequireEqual, -update flag)
func TestViewList(t *testing.T) {
	m := newTestModel(fixtureSnapshot(), 120, 40)
	golden.RequireEqual(t, []byte(ansi.Strip(m.View().Content))) // or keep colours with a fixed profile
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `tea.KeyMsg`, `View() string`, `tea.WithAltScreen()` | `tea.KeyPressMsg`, `View() tea.View` with `v.AltScreen = true` | Bubble Tea v2 | Every snippet from before 2026 will not compile. |
| `github.com/charmbracelet/*` imports | `charm.land/*/v2` | v2 | Mixing gives two runtimes. |
| `search(type: ISSUE)` with OR | `type: ISSUE_ADVANCED` | GitHub advanced issue search | Legacy returns 0 for OR queries, silently (verified). |
| `brews` in GoReleaser | `homebrew_casks` | GoReleaser v2.10+ | Release is a later phase. |

**Deprecated/outdated:**
- The architecture note "avoid typed `GraphQLClient.Do` because it hides partial data": superseded; `DoWithContext` populates the struct before returning `*api.GraphQLError` (verified in source and by running it).
- The `commits(last:1){...statusCheckRollup}` shape is unnecessary: the `PullRequest.statusCheckRollup` shortcut matched it in 50 of 50 PRs. Use the shortcut (head `oid` is not needed in Phase 1).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Search may silently omit PRs from SSO-unauthorized orgs, and the error shape is FORBIDDEN-like. Not tested: the account has no SAML organisations. | Spike Results | A user in a SAML org sees PRs missing with no message. Mitigation is a future "PRs may be hidden by SSO" hint; ask the user whether they have SSO orgs. |
| A2 | `closed:>=DATE` is evaluated as a UTC date. | Pattern 3 | Off-by-one-day inclusion; neutralised by the one-day margin and exact client filter. |
| A3 | Stripping controls plus U+202A-202E and U+2066-2069 is a sufficient sanitiser for terminal safety. | Pattern 7 | Some exotic format characters could still reorder text; low impact. ZWJ is left alone so emoji sequences survive. |
| A4 | `PullRequest.author` can be null (deleted account) and should render `ghost`. Not observed live (search for `author:ghost` returned 0). | Pattern 6 | A nil deref if unhandled; pointer type plus test fixture covers it either way. |
| A5 | Showing GitHub's `REVIEW_REQUIRED` on CLOSED/MERGED PRs is acceptable. | Open Question 2 | A closed PR reading "required" may look odd; dimming (D-07) softens it. |
| A6 | Re-run behaviour: whether a failed-then-passed re-run leaves both runs in contexts. | Spike (Q2) | Irrelevant for the Phase 1 badge (rollup state); matters for Phase 3 detail. |
| A7 | `●`/`○` (East Asian ambiguous width) render as 1 column in the user's terminal. | Pitfall 10 | Misaligned columns in CJK-locale terminals. |
| A8 | `go install github.com/sbartram/prv/cmd/prv@latest` yields a pseudo-version or the latest tag in `--version`. Verified only by analogy (local `go build`), not through the proxy; the repo's `main` on origin has no Go code yet. | Phase Requirements (CONF-02) | `--version` could print `dev`; fallback is still valid output. Needs push (and ideally a `v0.1.0` tag) to verify. |
| A9 | Bubble Tea's `Run` restores the terminal on panic and `ctrl+c` as designed. Source read, not exercised in a real terminal. | Pattern 5 | Manual UAT covers it. |

## Open Questions

1. **How will the human verify rows on an account with no recent PRs?**
   - What we know: `sbartram` has 0 open, 0 assigned and 0 PRs closed in the last 14 days, so the real run shows the D-10 empty state. With `closed:>=2026-01-01` the same query returns 6 real merged PRs (cost 1).
   - What's unclear: whether to add a dev-only override (for example a hidden `--closed-days N` flag, or an unexported option set only in tests).
   - Recommendation: add a minimal hidden `--closed-days` flag defaulting to 14 (it is plumbed through `Fetch` options anyway for the injected clock and Phase 2 CONF-01), and have the end-of-phase UAT run `prv --closed-days 300`. The planner should confirm with the user; if declined, UAT of rows is fixture/golden only.

2. **REVIEW for merged/closed PRs.**
   - What we know: GitHub returns `REVIEW_REQUIRED` on some CLOSED PRs (two of the user's).
   - Recommendation: show GitHub's value unchanged (consistent with D-13 and "matches github.com"); the row is dimmed. Revisit in Phase 2 if it reads as noise.

3. **Verifying `go install` (CONF-02).**
   - What we know: origin `https://github.com/sbartram/prv` exists, is public, and its `main` (aa11d7e) is behind local `main`; STATE.md says push only when asked.
   - Recommendation: the plan includes a final, user-gated step (push, optionally tag `v0.1.0`, then `go install github.com/sbartram/prv/cmd/prv@latest` in a temp `GOBIN`). Mark it `checkpoint:human-verify`.

4. **Exit code when the user quits from the error panel.** Recommended 0 (user-initiated quit); a failed `Run()` exits 1.

5. **Show `rateLimit` now?** Recommended: request and decode it (one line) so Phase 2 does not change the query, but do not display it in Phase 1.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | build, tests | ✓ | go1.27.2 darwin/arm64 (`go.mod` floor must still be 1.26.0) | — |
| `gh` CLI | token via keyring, live checks | ✓ | 2.102.0, logged in as `sbartram` (keyring), scopes `admin:public_key, gist, read:org, repo` | `GH_TOKEN` / `GITHUB_TOKEN` |
| `GH_TOKEN` / `GITHUB_TOKEN` in this shell | precedence testing | ✗ (neither set) | — | set a throwaway value in a test only |
| Network to api.github.com | live fetch, spike | ✓ | HTTP 200, 5,000 pt/h budget, ~4,890 remaining after the spike | fixtures |
| git + `origin` remote | `go install` verification | ✓ | origin `https://github.com/sbartram/prv` (public); remote `main` behind local | — |
| golangci-lint, goreleaser | later phases | ✗ | — | not needed in Phase 1 |
| Module proxy (proxy.golang.org) | `go get`, `go install` | ✓ | all six modules resolved | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none that block this phase. golangci-lint is optional now.

## Security Domain

`security_enforcement` is enabled (absent means enabled; config has `security_enforcement: true`, ASVS level 1, block on high).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | yes (token acquisition only; no login flow) | go-gh `TokenForHost`; token in memory only, passed via `ClientOptions.AuthToken`; nothing persisted |
| V3 Session Management | no | No sessions |
| V4 Access Control | no | Read-only, the user's own token and GitHub's access control |
| V5 Input Validation | yes | All remote strings (title, logins, repo, API messages) cleaned by `Clean()` before rendering; GraphQL variables used for the search string, never concatenated into the query document |
| V6 Cryptography | no | TLS handled by `net/http` via go-gh; nothing hand-rolled |
| V7 Error Handling and Logging | yes | Errors name the token source, never the value; `LogIgnoreEnv: true`; no request dumps; no stdout/stderr writes while the TUI runs |
| V8 Data Protection | yes | Token never written to disk, config, fixtures, golden files or logs |
| V14 Configuration | yes | Document least-privilege tokens (classic `repo` + `read:org` from `gh`; fine-grained read-only Pull requests, Checks, Commit statuses, Metadata); v1 never requests write scopes |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Terminal escape / bidi injection via attacker-controlled PR titles (public repos) | Tampering / Spoofing | `Clean()` in the mapper; assert no `\x1b` or bidi override in `View().Content` in a test |
| Token leaked via `GH_DEBUG` HTTP logging | Information disclosure | `LogIgnoreEnv: true` (verified silent); never set `Log` |
| Token in error strings, panics or crash output | Information disclosure | Error types carry source and status, never the value; test that a fake token never appears |
| Stale env token overriding the intended identity | Spoofing (wrong account) | Header shows `@login`; D-10 empty state names the account; 401 hint names env vars |
| `gh` binary resolution (PATH hijack) | Elevation | go-gh uses `safeexec.LookPath` and honours `GH_PATH`; do not shell out yourself |
| Token or private PR data committed in fixtures | Information disclosure | Synthetic fixtures only; no recorded headers |

## Sources

### Primary (HIGH confidence)
- Live `gh api graphql` spike on 2026-10-11 (queries and results recorded in `/Users/scottb/dev/bartram/prv/.planning/phases/01-status-at-a-glance/01-SPIKE.md`)
- go-gh v2.16.2 source read: `pkg/auth/auth.go` (token order, lines 36-109), `pkg/api/graphql_client.go` (`DoWithContext`, lines 68-110), `pkg/api/errors.go`, `pkg/api/client_options.go`
- Bubble Tea v2.1.0 source: `tea.go` (`Model`, `View`, `NewView`, `Run` panic handling), `key.go`, `commands.go`
- Bubbles v2.2.1 `spinner/spinner.go`, `key/key.go`; Lip Gloss v2.0.6 `color.go`, `table/table.go`, `size.go`; `x/exp/golden` v0.1.0 `golden.go`
- proxy.golang.org module metadata via `go list -m -json` (versions, publish times, origin repos)
- Throwaway Go programs compiled and run this session (partial-error handling, `LogIgnoreEnv`, key strings, table sizing, version stamping, width measurement, derivation functions against live data)
- `/Users/scottb/dev/bartram/prv/.claude/CLAUDE.md`, `.planning/PROJECT.md`, `REQUIREMENTS.md`, `ROADMAP.md`, `STATE.md`, `01-CONTEXT.md`, `research/SUMMARY.md`, `ARCHITECTURE.md`, `PITFALLS.md`

### Secondary (MEDIUM confidence)
- None used.

### Tertiary (LOW confidence)
- None. Items that could not be verified are in the Assumptions Log.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH, versions and origins verified on the proxy; APIs compiled against the real modules.
- Architecture: HIGH, query and error behaviour executed live; layering follows the project's stated architecture.
- Pitfalls: HIGH for data-layer items (measured); MEDIUM for terminal-restoration and glyph-width items (A7, A9).
- Not covered by evidence: SAML/SSO behaviour, user-account assignee/open branches (A1, stand-in only).

**Research date:** 2026-10-11
**Valid until:** 2026-11-10 (stable stack; re-run the spike query if GitHub changes advanced search or `reviewDecision` semantics)

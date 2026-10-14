# Phase 4: Curated Watch List - Research

**Researched:** 2026-10-12
**Domain:** Go + Bubble Tea v2 TUI; GitHub GraphQL aliased lookups; a small local JSON store
**Confidence:** HIGH for the lookup query, error shapes, store design and code integration points. MEDIUM for the SAML/SSO error shape (not observable here; see A1).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Add flow (WATCH-04)
- **D-01:** `a` (list view only) opens a one-line prompt at the bottom, the same place and feel as the `/` number prompt (Phase 2 D-11). Example: `add PR: █  (URL or owner/repo#n · enter add · esc cancel)`. It takes free text and bracketed paste (`tea.PasteMsg`). The digits-only `/` prompt does not; extend it or use `bubbles/textinput`, as the planner chooses.
- **D-02:** On `enter`, prv validates the PR by **fetching it before saving** (roadmap pitfall). While it checks, the prompt stays open and shows `checking owner/repo#n ⠋`. **On error, the typed text stays** and the reason shows inline in the prompt line (e.g. `✗ not found: owner/repo#4`), so a typo can be fixed. `esc` cancels at any time; a check result that arrives after esc is dropped. **On success, the prompt closes, the entry is saved, the PR appears with full status, and the cursor jumps to the new row** (selected `Ref`). Nothing is saved on any error (malformed, not found, no access, SSO, network).
- **D-03:** **A duplicate is not an error.** If the PR is already listed (auto-found, manual, or both), prv adds no row and saves nothing. The prompt closes, the cursor jumps to the existing row, and the footer briefly shows `already watching owner/repo#n`. When the 1/2/3 or `/` filters hide that row, the message adds `· hidden by filters`. **Pasting an auto-found PR does NOT pin it as manual.** Duplicates are matched by normalised `Ref`: owner and repo are case-insensitive, and the number is an integer. Once the PR is fetched, also check its node ID, because a renamed repo can give a different ref for the same PR.
- **D-04:** **The parser is lenient for github.com PR forms only.** It accepts `https://github.com/o/r/pull/12` with an optional trailing `/files`, `/checks`, `/commits`, or `/commits/<sha>`, plus `#fragment` and `?query`. The `https://` scheme and `www.` may be missing, and surrounding whitespace is trimmed. It also accepts `o/r#12`. It rejects other hosts (GHE: `only github.com PRs are supported`), issue URLs (`that is an issue, not a PR`), and anything else (`not a PR reference`), each with its specific reason. One ref per paste. Use one anchored regexp per form, as CLAUDE.md says. go-gh `repository.Parse` is for OWNER/REPO only.

#### Manual marker & removal (WATCH-06)
- **D-05:** A manually added row has a **marker glyph before the ref in the PR column** (e.g. `+ cli/cli#123`). The marker also stays in compact mode (Phase 3 D-14): `+ api#42 ✗⊘✗`. There is no new column. The `?` help legend explains it. The exact glyph is Claude's discretion (`+` was the example). It must not clash with existing glyphs and must read without colour.
- **D-06:** `d` (list view only) asks in the footer: `remove owner/repo#n from watch list? y/n`. Only `y` confirms; any other key cancels. There is no undo.
- **D-07:** A PR that is **both manual and auto-found** shows the marker (it is in the saved list). `d` removes only the saved entry, and the row stays because it is still auto-found. The footer says `unpinned — still listed (authored/assigned)`. After that the closed window applies to it again.
- **D-08:** `d` on an auto-found-only row removes nothing and shows a short footer note, e.g. `owner/repo#n is authored/assigned — only + rows can be removed`. `a` and `d` do not work in the detail view, which stays read-only.

#### Unavailable rows (LIST-09)
- **D-09:** An unavailable row shows the marker and ref, then `⚠ unavailable: <reason>` across the TITLE..CI columns, in the **warning (yellow) colour, not faint**: faint already means draft/merged/closed (Phase 2 D-07). All other cells are blank. Compact mode shows `+ api#42 ⚠`. It must read without colour (glyph plus word).
- **D-10:** prv tells three reasons apart:
  - NOT_FOUND → `not found (deleted, renamed, or no access)`. GitHub returns NOT_FOUND for private repos the token can't see, so "deleted" and "no access" can't be told apart.
  - SAML/SSO FORBIDDEN → `SSO: authorize your token for <org>`, with a hint (`gh auth refresh`, or the org's SSO page). The spike never saw this shape (no SAML orgs on the account), so the research must find the real `errors[]` type/message and the tests must use a constructed fixture.
  - Anything else → `could not load (<short error>)`.
  The add prompt (D-02) uses the same reason wording.
- **D-11:** Unavailable rows sort in a **new top bucket (0) above failing CI**, because a pinned PR is broken and the user must decide (fix SSO or remove it). Inside the bucket, rows sort by ref. `o` opens the github.com URL built from the ref, which helps with SSO. `enter` opens a detail view that shows only the ref and the full reason, with no checks or reviewers fetched. `d` works on them as on any manual row.
- **D-12:** **The cause decides what a failure shows:**
  - The whole request fails (network, 5xx, 401, rate limit) → the existing Phase 2 `✗ refresh failed` header marker (Phase 2 D-07/D-08). Every row keeps its last data, and no row turns unavailable.
  - A per-PR NOT_FOUND or SSO error → the row turns unavailable **at once**, because it is a real answer from GitHub.
  - A per-PR "other" error → keep the PR's last data until it fails on **2 refreshes in a row**. After that it turns unavailable. On the first launch there is no last data, so it is unavailable at once.
  - The saved entry is **never deleted automatically** (roadmap). An unavailable PR that comes back shows normally on the next good refresh.

#### Manual PRs in the list (WATCH-07)
- **D-13:** Manual PRs are fetched and listed whatever their closed date (WATCH-07). The **1/2/3 state toggles and the `/` number filter apply to every row**, manual or auto. The window exemption only decides which PRs are fetched and kept. The state toggles never hide unavailable rows, because their state is unknown. The `/` number filter matches them by their PR number.
- **D-14:** Manual rows use the same attention buckets as auto rows (Phase 2 D-01), plus bucket 0 for unavailable rows (D-11). Merged and closed manual PRs go in the merged and closed buckets.
- **D-15:** The header count stays `N PRs`, with no split between manual and auto. When any rows are unavailable, add `· N unavailable` in the warning colour. This segment is an integrity note, so it survives narrowing before the key hints do (Phase 2 WR-03 rule). The empty state keeps the Phase 1 D-10 wording and adds `· press a to add any PR`. The footer hint list gains `a add`, plus `d remove` where it fits.
- **D-16:** The saved list stays in sync across prv instances (two panes) and hand edits. **Each add and remove reloads `watchlist.json`, applies that one change, and writes atomically** (temp file in the same dir, then `os.Rename`), so two instances never overwrite each other. **Each refresh tick also re-reads the file**, which is cheap and local, so another pane's adds and removes appear within one interval. Every file read and write runs in a `tea.Cmd` (no I/O in Update/View).

#### Carried forward (do not re-litigate)
- Store (roadmap pitfalls): the file holds PR refs only, as versioned JSON at `$XDG_DATA_HOME/prv/watchlist.json` (fallback `~/.local/share/prv/`). The write is atomic. The load is tolerant, and a corrupt file is backed up rather than overwritten. Manual entries are never removed automatically.
- Batched lookups: aliased `repository(owner:,name:){pullRequest(number:)}` lookups built **with GraphQL variables, never string interpolation**, in chunks of 25-50 aliases. Read the envelope yourself and map `errors[].path[0]` to its alias. go-gh fills partial data and then returns `*api.GraphQLError` (spike Q3).
- Dedupe by node ID across auto and manual (WATCH-03). Clean remote strings at the mapper seam (`domain.Clean`). Single-flight refresh, generation counter, backoff order, panic-wrapped Cmds (Phase 2). Every query includes `rateLimit`.
- `domain.Ref` is the cursor's source of truth, and `rebuild()` builds the rows (Phase 2 D-03). ANSI-16 palette, and every badge is glyph plus word (Phase 1 D-01/D-05). Display-width helpers only (Phase 3).
- Keys are single-key and come from one keymap that also feeds the footer and the `?` help (Phase 2 D-17). `q` means quit everywhere (Phase 3 D-09).

### Claude's Discretion
- The exact marker glyph (D-05) and the unavailable glyph (D-09), and how both appear in the help legend.
- Whether the add prompt extends the existing hand-rolled prompt or uses `bubbles/textinput`.
- The query shape for manual PRs: a separate aliased query after discovery, or aliases merged into the discovery request. Chunk size between 25 and 50. Whether an add-validation fetch reuses the same lookup path.
- How the "2 refreshes in a row" counter (D-12) is stored: per Ref, in the model or the domain.
- The JSON schema details (version field, entry fields such as `added_at`), the corrupt-file backup naming, and how a load warning appears (a one-time TUI warning like the config warnings, Phase 2 D-15).
- How long the transient footer messages (D-03, D-07, D-08) stay up. Reuse the existing `setFlash` timing.
- The exact wording of the reason strings and hints, within D-04 and D-10.

### Deferred Ideas (OUT OF SCOPE)
- Pasting several refs at once (space- or newline-separated) — considered and not chosen. It could be a later enhancement.
- Pinning an auto-found PR as manual when it is pasted, so it outlives the closed window — considered and not chosen (D-03).
- Undo for removal — not chosen. The y/n confirmation is used instead.

Out of scope (CONTEXT domain): write actions, GHE hosts, saved searches, multi-ref paste, any new list columns.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| WATCH-04 | Add a PR by pasting a github.com PR URL or `owner/repo#123`; invalid or inaccessible PRs show an error and are not saved | `domain.ParseRef` (one anchored regexp per form), `Client.Lookup` (variables-only aliased query, live-verified), add-prompt state machine with sequence-number drop, `PasteMsg` sanitising, validate-before-save order |
| WATCH-05 | Manually added PRs are saved locally and reappear after restart | New `internal/watchlist` store: versioned JSON, atomic temp+rename, tolerant load with hash-named backup; `Client.Fetch` reads it on every refresh |
| WATCH-06 | User can remove a manually added PR | `d` + y/n confirm, `Store.Remove` (reload-apply-write), `PR.Manual`/`PR.Auto`/`PR.Watched` fields so removal targets the saved key and the "unpin" case keeps the row |
| WATCH-07 | Manually added PRs stay listed after they close (closed window does not apply) | `InWindow` returns true for `Manual`; manual refs not found in discovery are fetched by lookup, which is not window-filtered |
| LIST-09 | A PR that can't be fetched shows as an "unavailable" row with a reason; other PRs still display | Per-alias error attribution (`errors[].path[0]`), 3 reason kinds, bucket 0, `domain.Reconcile` for the 2-strike rule, line-replacement rendering, reason-only detail |
</phase_requirements>

## Summary

Phase 4 extends four existing seams rather than adding new architecture. (1) `internal/github` gains one more query (`Lookup`, aliased `repository(owner:$oN,name:$nN){pullRequest(number:$pN)}`, chunked at 25) that `Client.Fetch` runs after discovery for every saved ref the discovery did not already return. (2) `internal/domain` gains the pure parts: `ParseRef`, `Ref.Key`, the unavailable value type, `Combine`/`MarkWatched`, `Reconcile` (the 2-strike rule), bucket 0, and `Manual` in `InWindow`. (3) A new `internal/watchlist` package owns the JSON file (load, add, remove, atomic write, corrupt backup). (4) `internal/ui` gains an add prompt, a remove confirmation, unavailable-row rendering and a reason-only detail. `ui.Fetcher` and its many test fakes stay unchanged: the `Client` is handed a small `WatchSource` interface at construction and reads the file inside `Fetch`, which already runs in a `tea.Cmd`.

I ran the aliased lookup against the live API. Variables work for owner, name and number. Owner and name are case-insensitive and the response returns the canonical `nameWithOwner`. A 50-alias query with the full list field set reported `rateLimit.cost = 1`. A missing repo, a missing PR number, a private repo the token cannot see, and out-of-range numbers (0, -1, 2147483648, 99999999999) all come back as HTTP 200 with a per-alias `NOT_FOUND` at `path` `["rN"]` or `["rN","pullRequest"]`, so a bad ref cannot poison its chunk. go-gh decoded the envelope into `map[string]json.RawMessage` with `Type` and `Path` populated, and `errors.As(err, &*api.GraphQLError)` succeeded. The one thing that could not be observed is the SAML FORBIDDEN shape (this account is in no SAML orgs). The best public evidence is that GitHub GraphQL FORBIDDEN errors carry `extensions.saml_failure` (a boolean) and the message "Resource protected by organization SAML enforcement". Detect SSO on `type == FORBIDDEN` together with either signal, and test it with a constructed fixture (A1).

Two design risks need a decision before coding, and both have a recommended answer below. First, a stale in-flight refresh can overwrite a just-added row; the fix is an epoch counter that discards (and re-fetches) any result started before the last local mutation. Second, the lipgloss table cannot draw one cell across TITLE..CI; the existing code already splits the table output into lines, so the unavailable row should be a hand-built line that replaces its table line.

**Primary recommendation:** Add `domain` pure functions first (parser, unavailable type, combine/reconcile, bucket 0), then the `watchlist` store, then `github.Client.Lookup` wired into `Fetch`, then the ui prompt/rows/detail, keeping `ui.Fetcher` unchanged and giving the `Client` a `WatchSource`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Parse a pasted reference | `internal/domain` (pure) | `ui` (calls it, shows the reason) | No I/O; needs a table test of accepted and rejected inputs |
| Persist saved refs (load/add/remove, atomic write, backup) | `internal/watchlist` (file I/O) | `cmd/prv` (wires it), `ui` via a consumer-side interface | File I/O must not live in `domain` or run in Update/View |
| Fetch saved PRs, map per-alias errors | `internal/github` | `domain` (types) | Only `github` knows the wire format and `errors[]` shapes |
| Dedupe auto vs manual, window exemption, bucket 0, 2-strike rule | `internal/domain` (pure) | `github` (calls Combine), `ui` (calls Reconcile) | Pure functions are testable without mocks |
| Add prompt, remove confirm, cursor jump, flashes | `internal/ui` | — | Bubble Tea state; all I/O in Cmds |
| Unavailable row and marker rendering | `internal/ui` (view.go) | `domain` (reason text) | Display width and styling are ui concerns |
| Wiring (store path, client options) | `cmd/prv` | — | Composition root only |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/cli/go-gh/v2` `api.GraphQLClient` | v2.16.2 (already in go.mod) | Aliased lookup query | Already the project's client; `DoWithContext` fills `data` then returns `*api.GraphQLError` [VERIFIED: live probe this session] |
| Go stdlib `regexp`, `strconv`, `strings` | go 1.26 floor, toolchain go1.27.2 | Reference parser | CLAUDE.md: "one anchored regexp" per form |
| Go stdlib `encoding/json`, `os`, `path/filepath`, `crypto/sha256` | stdlib | Store, atomic write, backup name | No new dependency |
| `charm.land/bubbletea/v2` | v2.1.0 (in go.mod) | `tea.PasteMsg` (`Content string`) | [VERIFIED: bubbletea@v2.1.0/paste.go: `type PasteMsg struct { Content string }`] |
| `charm.land/bubbles/v2/spinner` | v2.2.1 (in go.mod) | "checking" spinner | Duplicate tick chains are rejected by tag [VERIFIED: bubbles@v2.2.1/spinner/spinner.go:136-153] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/charmbracelet/x/exp/golden` | v0.1.0 (in go.mod) | Golden files for the new states | Prompt (checking/error), confirm, unavailable rows, marker, compact, empty state |
| stdlib `net/http/httptest` / the existing `fixtureTransport` | — | Lookup fixtures | Extend `fixtureTransport` with a `Lookup` operation |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled add prompt buffer (recommended) | `bubbles/textinput` v2 | textinput handles `PasteMsg`, a sanitiser and cursor movement, but it adds a blinking virtual cursor (Cmd-driven) that makes goldens time-dependent, and the existing `/` prompt is a hand-rolled buffer with a static `█`. Free text with end-only editing is enough for a paste-and-fix flow. [VERIFIED: textinput.go has `PasteMsg` case at line 654 and `virtualCursor` blink logic] |
| Separate `Lookup` query after discovery (recommended) | Merge aliases into the discovery request | Discovery is paginated by cursor (up to 5 pages); merging would repeat the aliases per page or couple lookup to page 1. A separate query also lets the add prompt reuse it. |
| Read the file inside `Client.Fetch` via `WatchSource` (recommended) | Change `ui.Fetcher` to `Fetch(ctx, refs)` | The signature change touches every `fakeFetcher` and test; the read still runs inside a Cmd either way. |
| Accept the load-modify-write window (recommended) | `flock` / lockfile | `syscall.Flock` does not exist on Windows (build tags needed) and a lockfile can go stale. Add/remove happen at human speed; the window is milliseconds. |

**Installation:** none. No new module is needed (`go.mod` already has everything). `[VERIFIED: /Users/scottb/dev/bartram/prv/go.mod]`

**Version verification:** versions above are the ones already pinned in go.mod (read this session) and match the CLAUDE.md stack table; no new registry lookups were required.

## Package Legitimacy Audit

No external packages are added by this phase (stdlib plus modules already in `go.mod`). The legitimacy gate does not apply.

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
 user keys / paste                                  watchlist.json
        |                                                  ^  |
        v                                                  |  | (read each Fetch,
 +-------------+  a/enter   +-------------------+  Add/Remove|  |  inside the Cmd)
 | ui.Model    |----------->| add flow (Cmds)   |-----------+  |
 | Update/View |            | parse -> dup check|              v
 | no I/O      |            | -> Lookup -> Save |        +--------------+
 +------+------+            +---------+---------+        | watchlist    |
        ^  | tick / r                 | Lookup(1 ref)    | Store (Load/ |
        |  v                          v                  | Add/Remove)  |
        | fetch Cmd            +--------------+           +------+-------+
        |  (single flight)     | github.Client|<-- Refs() -------+ WatchSource
        |                      |  Fetch:      |
        |  fetchedMsg{epoch}   |  1 discovery |--> GraphQL search (pages <=5)
        +----------------------+  2 MarkWatched|
           Snapshot            |  3 Lookup rest|--> GraphQL "Lookup" (chunks of 25)
           (Auto|Manual|       |  4 Combine+   |      per-alias: PR | NOT_FOUND |
            Unavail rows)      |    InWindow   |      FORBIDDEN(SAML) | other
                               |  5 fetchFailing|--> GraphQL FailingChecks (existing)
                               |  6 Sort       |
                               +--------------+
 ui on fetchedMsg: Reconcile(prev, next, strikes) -> rebuild() -> rows
```

### Recommended Project Structure
```
internal/domain/
├── watch.go          # NEW: Ref.Key, ParseRef + RefError, Unavail type, Combine, MarkWatched, Reconcile
├── pr.go             # EDIT: PR gains Auto, Manual, Watched, Unavail
└── list.go           # EDIT: InWindow (Manual), Bucket (0), Sort (bucket 0 by ref), Visible (unavailable never hidden by 1/2/3)
internal/watchlist/
├── store.go          # NEW: Path(), Store, Load, Add, Remove, atomic write, backup
└── store_test.go     # t.TempDir() + XDG_DATA_HOME
internal/github/
├── lookup.go         # NEW: lookupQuery builder, Client.Lookup, alias attribution, reason mapping
├── client.go         # EDIT: Options.Watch, Fetch integration, split mapNode out of mapResponse
└── testdata/lookup_*.json  # NEW fixtures (see Test approach)
internal/ui/
├── add.go            # NEW: add-prompt state, ParseRef -> dup check -> lookup Cmd -> save Cmd
├── keys.go           # EDIT: Add (a), Remove (d); addKeys for the prompt
├── model.go          # EDIT: modes (adding / confirming), epoch, strikes, snapshot apply helper
├── view.go           # EDIT: bottom(), header segment, rows (marker, unavailable), compact, empty text
├── detail.go         # EDIT: reason-only detail for unavailable rows
└── help.go           # EDIT: legend lines for the marker and unavailable rows
cmd/prv/main.go       # EDIT: build the store, pass to github.Options.Watch and ui.Options.Store
```

### Pattern 1: Aliased lookup with variables only
**What:** Build the document from fixed text plus the integer index. Owner, name and number travel only as `$oN`, `$nN`, `$pN`. [VERIFIED: live `gh api graphql` run this session, cost 1 for 25 and for 50 aliases]
**When to use:** every refresh (saved refs the discovery did not return) and the add-validation (one ref).
**Example:**
```go
// Source: live-verified shape; built the same way as checks.go builds its $ids request.
const lookupFields = `id number title url isDraft state createdAt closedAt mergedAt updatedAt
  repository { nameWithOwner } author { login } reviewDecision mergeStateStatus mergeable
  latestOpinionatedReviews(first: 20) { nodes { state author { login } } }
  reviewRequests(first: 20) { nodes { requestedReviewer { __typename
    ... on User { login } ... on Bot { login } ... on Mannequin { login }
    ... on Team { combinedSlug } ... on EnterpriseTeam { combinedSlug } } } }
  statusCheckRollup { state }`   // same selection as discoveryQuery's PullRequest fragment

func buildLookup(n int) string {
	var decl, body strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 { decl.WriteString(", ") }
		fmt.Fprintf(&decl, "$o%d: String!, $n%d: String!, $p%d: Int!", i, i, i)
		fmt.Fprintf(&body, "  r%d: repository(owner: $o%d, name: $n%d) { pullRequest(number: $p%d) { %s } }\n",
			i, i, i, i, lookupFields)
	}
	return "query Lookup(" + decl.String() + ") {\n  rateLimit { limit cost remaining resetAt }\n" + body.String() + "}"
}
// vars: {"o0": "acme", "n0": "api", "p0": 42, ...}   // never inside the document
```
The operation name `Lookup` matters: the test `fixtureTransport.opName` routes by the word after `query ` and cuts at `(`, space or `{` [VERIFIED: internal/github/client_test.go:25-38].

### Pattern 2: Read the envelope yourself, attribute by `path[0]`
**What:** Decode into `map[string]json.RawMessage`; `rateLimit` and `rN` are separate keys; a failed alias is the JSON `null`. [VERIFIED: probe test this session printed `r1 => null` and `type="NOT_FOUND" path=[]interface {}{"r1"}`]
```go
var resp map[string]json.RawMessage
err := c.gql.DoWithContext(ctx, query, vars, &resp)
var ge *api.GraphQLError
perAlias := map[string][]api.GraphQLErrorItem{}
if err != nil {
	err = classify(err, c.capture.latest())          // rate limit / cancel / HTTP => whole request
	if !errors.As(err, &ge) || len(resp) == 0 || rateLimited(err) { return nil, err }
	for _, it := range ge.Errors {
		if len(it.Path) > 0 { if a, ok := it.Path[0].(string); ok { perAlias[a] = append(perAlias[a], it) } }
	}
}
```
Rules: an alias is unavailable only when `r.pullRequest` is `null` (or the alias itself is `null`). An error deeper than `pullRequest` (for example `["r3","pullRequest","statusCheckRollup"]`) on a non-null PR is a snapshot warning, not an unavailable row. An error with no matching alias path, and a null alias with no error, map to the "other" kind. `classify` returns the existing `*Error` with `Kind == KindGraphQL` for a GraphQL error, and the rate-limit decision (headers first, then `RATE_LIMITED` type) is already inside it [VERIFIED: internal/github/errors.go:142-160, 171-181].

### Pattern 3: Per-alias reason kinds
Map the errors of a null alias to the three D-10 kinds, in this priority:
1. any item with `Type == "FORBIDDEN"` and (`Extensions["saml_failure"] == true` or message contains "SAML", case-insensitive) → `UnavailSSO`. The org shown is the owner part of the saved ref (validated by the parser regexp), not parsed out of the message.
2. any item with `Type == "NOT_FOUND"` → `UnavailNotFound`.
3. everything else, including a non-SAML `FORBIDDEN` → `UnavailOther` with a short fixed reason built from `domain.Clean(item.Message)` cut to about 40 cells. Never put the token or headers in it.

Use a closed enum, not strings, so the UI cannot get a typo (`UnavailNone` is the zero value so existing PR literals stay "available").

### Pattern 4: Domain additions (pure)
```go
// internal/domain/watch.go (shape; names are recommendations)
type UnavailKind int
const (
	UnavailNone UnavailKind = iota // zero value: the PR loaded
	UnavailNotFound
	UnavailSSO
	UnavailOther      // hard: shown as a row
	UnavailTransient  // soft: Client emits it for "other"; Reconcile decides (D-12)
)
type Unavail struct{ Kind UnavailKind; Reason string }

// PR gains: Auto bool (discovery found it), Manual bool (it is in the saved list),
// Watched Ref (the saved key, set when Manual; differs from Ref after a repo rename), Unavail Unavail.
func (r Ref) Key() Ref // Repo lowercased: the dedupe and removal key
func ParseRef(s string) (Ref, error) // *RefError{Reason string} with the three D-04 reasons
func MarkWatched(auto []PR, saved []Ref) (rest []Ref) // sets Manual/Watched on matching auto PRs (Key match); returns saved refs no auto PR covers
func Combine(auto, looked []PR) []PR // dedupe by node ID (a renamed repo): keep one row, Manual+Watched true
func Reconcile(prev, next []PR, strikes map[Ref]int) ([]PR, map[Ref]int) // D-12 two-strike rule
```
`Reconcile` for each `next` row with `UnavailTransient`: if `prev` has the same `Watched.Key()` row that is available and `strikes[key]+1 < 2`, substitute the prev copy and bump strikes; otherwise convert to hard `UnavailOther`. Any row that loaded resets its strike. A whole-request error never reaches `Reconcile` (the model keeps `m.snap`), so strikes only count per-PR answers. Return a new map (the `Model` is a value type; do not mutate a shared map in `Update`).

`Bucket`: add `if pr.Unavail.Kind != UnavailNone { return 0 }` as the first check. `Sort` already orders equal buckets by CreatedAt (zero for unavailable rows), then repo, then number, which gives "by ref" for bucket 0; compare the repo with `strings.ToLower` for bucket 0 so `Acme/x` and `acme/y` do not split. `Visible`: add an explicit `pr.Unavail` check before the state test (do not rely on `State` being `StateUnknown`; the zero `State` is `StateOpen`) [VERIFIED: internal/domain/pr.go "The zero value is StateOpen" comment, and list.go:49-50 `case StateUnknown: return 9`]. `InWindow`: `if pr.Manual { return true }` first.

### Pattern 5: Fetch integration order (inside `Client.Fetch`)
1. Discovery pages (unchanged). Keep the window filter **after** step 2.
2. `saved, warn := c.watch.Refs()`; `rest := domain.MarkWatched(snap.PRs, saved)` and mark all discovery PRs `Auto = true`. A saved PR that discovery already returned costs no lookup call.
3. `looked := c.lookupRefs(ctx, rest)` in chunks of 25. A request-level error (network, 5xx, 401, rate limit, cancel) returns `domain.Snapshot{}, err` so the model keeps all last data (D-12 row 1).
4. `snap.PRs = domain.Combine(snap.PRs, looked)`, then `slices.DeleteFunc(... !domain.InWindow ...)` (Manual rows pass).
5. `fetchFailing` over the combined list (manual failing PRs get their check name for free), then `domain.Sort`.
6. Add the lookup `cost` to `snap.RateLimit.Cost`, and update `Limit/Remaining/ResetAt` from the last response, exactly as the existing code does for discovery pages [VERIFIED: client.go:266-269].

### Pattern 6: UI add flow (all I/O in Cmds, sequence-stamped)
```
a            -> m.adding = true, buf = ""            (list or empty screen only; not in detail, loading or error)
typing/paste -> sanitise, append (cap ~256 runes)
enter        -> ParseRef            -> error?  show "✗ <reason>" inline, keep text
             -> dup check vs m.snap.PRs by Ref.Key  -> found? close, select, flash "already watching …"
             -> addSeq++, m.checking = ref, Cmd: Lookup([ref]) -> addCheckedMsg{seq, pr, err}
addCheckedMsg-> drop if seq != m.addSeq or prompt closed (esc bumps addSeq)
             -> err or pr.Unavail != None -> inline "✗ <reason>", keep text, save nothing
             -> second dup check by pr.ID vs m.snap.PRs (renamed repo)
             -> Cmd: store.Add(pr.Ref) -> addSavedMsg{pr, added bool, err}
addSavedMsg  -> err: inline "✗ could not save: …"; ok: close prompt, insert pr (Manual, Watched, Auto from snap),
                Sort, select pr.Ref, rebuild(), mutEpoch = fetchEpoch
```
The add-validation uses the same `Client.Lookup`, so the reason wording is shared with unavailable rows (D-10 last line). During a rate-limit pause (`m.now().Before(m.pausedUntil)`), answer inline with `✗ rate limited — resumes HH:MM` and send nothing (Phase 2 D-08).

`Store.Add` returns `(added bool, err)`. `added == false` means another pane saved it first: treat it as the D-03 duplicate message rather than an error.

### Pattern 7: The stale-fetch race and the epoch rule
**What goes wrong:** an add (or remove) finishes while a refresh is in flight. That refresh read the file before the save, so when it lands it overwrites the model and the new row vanishes (or the removed row returns) until the next tick. Bumping `gen` does not help: `fetchedMsg` with another `gen` is dropped but `inFlight` stays `true`, which would wedge refreshing [VERIFIED: internal/ui/model.go:307-314 drops on `msg.gen != m.gen` without clearing `inFlight`].
**Recommended:** add `epoch int` to `fetchedMsg` (zero value keeps every existing test valid) and `fetchEpoch`, `mutEpoch` to the model. `beginFetch` does `m.fetchEpoch++` and captures it; add/remove success sets `m.mutEpoch = m.fetchEpoch`. In the `fetchedMsg` handler: `if msg.epoch < m.mutEpoch { m.inFlight = false; return m, m.beginFetch() }` (discard, start a fresh fetch, which re-reads the file). `Init()` uses epoch 0. This costs one extra request only in the rare overlap.

### Pattern 8: Row rendering
- **PR column text:** `+ ` prefix for `Manual` rows. Reserve the 2 cells only when at least one visible row is manual, otherwise every existing golden stays byte-identical and narrow panes lose nothing. When reserving, pad non-manual rows with 2 spaces so refs align. Compute `natural[colPR]` from the final cell text (it already takes `lipgloss.Width` of each cell [VERIFIED: view.go:391-393]).
- **Unavailable row:** the lipgloss table pads or wraps each cell to its own column, so it cannot show one string across TITLE..CI. The code already does `tableLines := strings.Split(t.String(), "\n")` and `m.stack(tableLines[0], tableLines[1:])` [VERIFIED: view.go:485-486], so rows are exactly one line each. Leave every cell except the PR cell blank in the table (and skip the row when computing natural widths so a 50-cell reason does not widen TITLE), then replace that table line with a hand-built line: marker, padded PR cell, gap, then `warnStyle.Render("⚠ unavailable: "+reason)` truncated with `ansi.Truncate` to the width of the shown TITLE..CI span. Apply `selectedStyle` per segment on the cursor row, the same way `compactRow` does. A short spike (one test) should confirm line index = `row - m.offset + 1` before the golden work.
- **Compact:** `compactRow` prefixes `+ ` for manual rows (inside the ref budget) and, for unavailable rows, replaces the three glyphs with one `⚠` in `warnStyle` [VERIFIED: view.go:549-579 builds marker + ref + three glyphs]. The "no room for a ref" branch must still draw the marker.
- **Not faint:** `dimmed(pr)` only checks `IsDraft` and `State` merged/closed [VERIFIED: styles.go dimmed]; unavailable rows keep `State == StateOpen`-zero or any value, so make `dimmed` return false for them explicitly.
- **Header:** add a segment `warnStyle.Render(strconv.Itoa(n)+" unavailable")` with prio 6 or higher so it outlives `updated` (prio 5) [VERIFIED: header segment prios in view.go:281-286, 297, 317, 322]. The count `N PRs` keeps counting unavailable rows.
- **Empty text:** append ` · press a to add any PR` to the Phase 1 D-10 line in `renderEmpty`. Do not change the truncated/warnings variant's lead sentence.

### Pattern 9: Modes in Update
`Update` checks `showHelp`, then `prompting`, then `detail.open`. Add `adding` and `confirming` right after `prompting` with the same shape (`promptKey`-style handler; `ctrl+c` still quits). `bottom()` returns the number prompt today when `m.prompting`; extend it to return the add prompt or the confirm line. In stateEmpty the footer path goes through `bottom()` too [VERIFIED: view.go:85-92, 120-130], so `a` can be allowed for `stateList` and `stateEmpty`. Extract the "set snapshot, set state list/empty, rebuild" block of the `fetchedMsg` handler into a helper so the add success path can reuse it (an add from the empty screen must switch to `stateList`).

### Pattern 10: Store
```go
// internal/watchlist/store.go (shape)
type file struct {
	Version int     `json:"version"` // 1
	PRs     []entry `json:"prs"`
}
type entry struct {
	Repo    string    `json:"repo"`    // owner/repo as the user typed it, validated
	Number  int       `json:"number"`
	AddedAt time.Time `json:"added_at"`
}
```
- `Path()` mirrors `config.Path`: `$XDG_DATA_HOME` when set and absolute, else `~/.local/share`, then `prv/watchlist.json` [VERIFIED: internal/config/config.go:53-62 is the pattern]. Main exposes it as a replaceable variable, as it does for `configPath`.
- Read with `io.LimitReader` at 1 MiB, as `config.Load` does [VERIFIED: config.go:130-137].
- Re-validate every entry through `domain.ParseRef`-grade rules on load (a hand-edited file is untrusted: the owner/repo end up in GraphQL variables, URLs and the `SSO: … <org>` text). Skip bad entries and report one warning.
- Atomic write: `os.MkdirAll(dir, 0o700)`, `os.CreateTemp(dir, ".watchlist-*.tmp")` (mode 0600), write, `Sync`, `Close`, `os.Rename`; remove the temp file on any error path.
- Corrupt or unknown-version file: copy (not rename) to `watchlist.json.corrupt-<first 8 hex of sha256>` if that name does not exist, return the empty list plus a warning. The hash name makes the 60-second re-read idempotent, so a still-corrupt file does not create a backup per tick. The read path never rewrites the file; only `Add`/`Remove` write, and they back up first.
- `Add` and `Remove` each do Load, apply one change on the in-memory list (dedupe by `Ref.Key`), write. They match by key, so `Remove` of a renamed repo's `Watched` ref works.
- On a mid-session read failure, `Client` keeps the last good refs it saw and adds a snapshot warning; it never treats "unreadable" as "empty".

### Anti-Patterns to Avoid
- **String-interpolating owner/repo/number into the query.** Only the integer alias index goes into the document. Add a test that no owner, repo or number string occurs in the request `query` text.
- **Treating any GraphQL error as "whole refresh failed" or as "PR unavailable".** Attribute by `path[0]`; a nested error on a loaded PR is a warning.
- **Deleting a saved entry because a lookup failed.** The store only changes in `Add`/`Remove` called from user keys.
- **Writing the file from `Update`/`View`.** Use Cmds (D-16).
- **Using `State` to mean "unavailable".** Use the explicit `Unavail` field.
- **Letting `o` build a URL from unvalidated text.** Build `https://github.com/<repo>/pull/<n>` only from a `Ref` that passed `ParseRef`/`Load` validation, then still pass it through `safeURL` [VERIFIED: ui/open.go:24-30 allow-list].
- **Storing node IDs or PR data in the file.** CONTEXT locks "PR refs only". (The older project research suggested storing a node ID; CONTEXT supersedes it.)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| GraphQL transport, auth, error decoding | A raw `net/http` client | Existing `c.gql.DoWithContext` + `classify` | Already handles HTTP errors, rate limits, partial data [VERIFIED: client.go, errors.go] |
| Atomic file replace | Write in place | `os.CreateTemp` in the same dir + `os.Rename` | Rename in one filesystem is atomic; in-place writes corrupt on crash |
| Terminal-safe strings | Ad hoc stripping | `domain.Clean` at the mapper seam, plus display-width helpers (`lipgloss.Width`, `ansi.Truncate`, `trimLeft`) | Phase 1/3 rules; CJK and emoji widths |
| Spinner | A second ticker | The model's existing `m.spin` | Extra tick chains are rejected by tag [VERIFIED: spinner.go:136-153]; extend the `spinner.TickMsg` guard (`m.inFlight \|\| m.state == stateLoading`) with the checking state |
| Transient footer messages | New timers | `setFlash(text, flashFor)` | Already redraws on expiry [VERIFIED: refresh.go:88-94] |
| Browser open | A new opener | `m.openURL` → `safeURL` → `openCmd` | Already the gate for `o` [VERIFIED: model.go:349-360] |
| XDG path resolution | `os.UserConfigDir` / a library | A copy of `config.Path` for `XDG_DATA_HOME` | `UserConfigDir` returns `~/Library/Application Support` on macOS; CLAUDE.md fixes `~/.config` and `~/.local/share` |

**Key insight:** every hard part (partial GraphQL data, rate limit classification, cursor-by-ref, flash timing, golden harness) already exists. The risk in this phase is wiring order and the two races (stale fetch, lost update), not missing libraries.

## Common Pitfalls

### Pitfall 1: The stale in-flight refresh erases a new add (or resurrects a remove)
**What goes wrong:** see Pattern 7. **How to avoid:** epoch rule. **Warning signs:** a test that starts a fetch, completes an add, then delivers the old `fetchedMsg` and expects the new row to survive (or a fresh fetch to be started).

### Pitfall 2: Cursor and selection use the canonical ref, removal uses the saved ref
**What goes wrong:** a repo renamed after the entry was saved: the saved key is `old/name#7`, the API returns `new/name#7`. Removing by the displayed ref finds nothing; deduping by ref shows two rows.
**How to avoid:** `PR.Ref` is the canonical API ref (cursor source of truth); `PR.Watched` is the key in the file; `d` calls `Remove(pr.Watched)`; `Combine` dedupes by node ID. For unavailable rows `Ref == Watched`.
**Note:** I could not exercise a real rename here. That `repository(owner:, name:)` follows GitHub's rename redirect is an assumption (A3); the code must not depend on it (an unresolved old name simply shows as not found).

### Pitfall 3: Whole chunk poisoned by one bad ref
**Verified not to happen** for out-of-range numbers: 0, -1, 2147483648 and 99999999999 each gave a per-alias `NOT_FOUND`, HTTP 200. The parser should still cap the number (`1..2147483647`, digits only) so a future GitHub change cannot fail a whole chunk, and so the number is a valid GraphQL `Int`.

### Pitfall 4: A saved ref in different letter case duplicates an auto row
`ACME/API#1` saved, discovery returns `acme/api#1`. `Ref` equality is case-sensitive. Match with `Ref.Key()` (lowercase repo) before the network step, mark the auto row `Manual`, and skip its lookup. Also dedupe by `ID` after the lookup (rename case).

### Pitfall 5: Paste content is hostile or multi-line
`tea.PasteMsg.Content` may hold escape sequences, `\r\n`, tabs, or a very long string. Run it through `domain.Clean`, drop newlines (a multi-line paste is "one ref per paste": reject with a clear reason, or keep only if it is a single line after trimming), and cap the buffer. Never draw raw paste text. The existing `/` prompt avoids this by accepting digits only [VERIFIED: model.go:196-204].

### Pitfall 6: The long buffer pushes the cursor and the status off screen
At 60 columns the prompt line `add PR: <url>█ (URL or owner/repo#n · enter add · esc cancel)` is wider than the pane. Build it as prefix + buffer + cursor + state suffix; drop the hint suffix first; if the buffer alone is too wide, show its tail with `ansi.TruncateLeft` so the cursor end stays visible. The `checking …` and `✗ reason` suffix outranks the hint. `bottom()` already ends with `m.cut(line)`.

### Pitfall 7: Hidden-by-filter outcomes
An added PR that the active filters hide (for example a merged PR with `2` off) is saved and not visible. The cursor cannot jump to it. Flash `added owner/repo#n · hidden by filters` in that case, mirroring D-03. Same for the duplicate case when `m.rows` does not contain the ref.

### Pitfall 8: Footer hint overflow
`ShortHelp` is already about 85 cells at the last width; the footer drops whole hints from the end when they do not fit [VERIFIED: view.go:188-197], so `? help` and `q quit` go first at 80 columns. Put `Add` before `Filter` and show `Remove` only when the selected row is manual (build the hint list in the model, not in `keyMap.ShortHelp`). Every list golden's footer changes: regenerate with `go test ./internal/ui -update` and review each diff.

### Pitfall 9: Detail view for an unavailable row
`openDetail` stores the row, then `startDetailFetch` calls `fetchDetail` with `pr.ID`, which is empty for unavailable rows [VERIFIED: detail.go:45-62]. Guard: no fetch when `pr.Unavail.Kind != UnavailNone`; `detailLines` returns the ref, a `⚠ unavailable` line, the full reason and the hint. `refreshDetail` copies the fresh row on every refresh, so a PR that comes back starts fetching again on its own.

### Pitfall 10: Concurrent requests share `captureTransport.latest()`
The add lookup can overlap a refresh (and the detail fetch already can). `classify` reads "the headers of the latest response", so a rate-limit decision for one request could read the other's headers. This risk already exists for detail; keep it, but do not add a third source of overlap beyond the add lookup.

### Pitfall 11: Confirmation prompt vs the quit key
D-06 says "any other key cancels"; Phase 3 D-09 says `q` quits everywhere. These collide. Recommended: in the confirm line, `y` removes, `ctrl+c` still quits, every other key including `q` cancels (the number prompt already ignores `q`). See Open Question 2.

### Pitfall 12: Corrupt-file churn and data loss
Renaming a corrupt file away every 60 s, or rewriting it from a failed parse, loses the user's data. Use the hash-named copy, never rewrite from the read path, and keep the last good in-memory refs.

## Code Examples

### The accepted and rejected forms (table test seed)
```go
// Source: D-04 + live-verified GitHub limits; owner/repo character classes per GitHub naming.
var (
	owner   = `[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?`  // <= 39 chars, no leading/trailing hyphen
	repo    = `[A-Za-z0-9._-]{1,100}`
	prURL   = regexp.MustCompile(`^(?:https?://)?(?:www\.)?github\.com/(` + owner + `)/(` + repo + `)/pull/([0-9]{1,10})` +
		`(?:/(?:files|checks|commits(?:/[0-9a-fA-F]{4,40})?))?/?(?:[?#].*)?$`)
	shortRe = regexp.MustCompile(`^(` + owner + `)/(` + repo + `)#([0-9]{1,10})$`)
	issueRe = regexp.MustCompile(`^(?:https?://)?(?:www\.)?github\.com/` + owner + `/` + repo + `/issues/[0-9]+(?:[/?#].*)?$`)
	hostRe  = regexp.MustCompile(`^(?:[a-z][a-z0-9+.-]*://)?([^/\s]+)/[^/\s]+/[^/\s]+/pull/[0-9]+`) // non-github host => GHE reason
)
// Order: TrimSpace -> reject inner whitespace/control -> issueRe -> prURL -> shortRe -> hostRe (host != github.com) -> "not a PR reference".
// Then strconv.Atoi, require 1 <= n <= 2147483647. Repo "." and ".." are rejected.
```
Table to cover: `https://github.com/o/r/pull/12`, `/files`, `/checks`, `/commits`, `/commits/abc1234`, trailing slash, `#issuecomment-1`, `?diff=split`, no scheme, `www.`, uppercase owner, repo with dots and underscores, `o/r#12`, `  o/r#12\n`; rejected: `https://ghe.corp.com/o/r/pull/1` (GHE), `github.com/o/r/issues/3` (issue), `o/r#0`, `o/r#99999999999`, `o/r#-1`, `o/r`, `#12`, two refs in one string, empty.

### Fixture for a constructed SAML error (tests only; shape is an assumption, A1)
```json
{"data":{"rateLimit":{"limit":5000,"cost":1,"remaining":4990,"resetAt":"2026-10-13T03:12:59Z"},
         "r0":null,
         "r1":{"pullRequest":{"id":"PR_x","number":2,"state":"OPEN", "...":"..."}}},
 "errors":[{"type":"FORBIDDEN","path":["r0"],"extensions":{"saml_failure":true},
            "message":"Resource protected by organization SAML enforcement. You must grant your Personal Access token access to this organization."}]}
```
Also add variants: error at `["r0","pullRequest"]`, `saml_failure` true with a different message, message-only SAML (no extension), and a non-SAML FORBIDDEN (`"saml_failure":false`, "Resource not accessible by personal access token") which must map to the "other" kind.

### Extending the test transport
`fixtureTransport.RoundTrip` fatals on an unknown operation (`default: f.t.Fatalf("unexpected operation %q", op)`) [VERIFIED: client_test.go:94-95]. Add a `case "Lookup"` that serves ordered fixtures and records bodies, like `Detail`, so tests can assert the variables and the no-interpolation rule.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Per-PR REST calls for a watch list | One aliased GraphQL request, 25-50 PRs, cost 1 | n/a (project decision) | One point per 25 saved PRs per refresh; 100 saved PRs cost about 4 points a minute at the 60 s default |
| `bubbles/textinput` with blink for every prompt | Static-cursor hand-rolled prompt where text is short-lived | project convention (Phase 2 D-11) | Deterministic goldens |

**Deprecated/outdated:** the older project research (`PITFALLS.md`, `ARCHITECTURE.md`) suggests storing a node ID with each entry and placing the file in the config dir. CONTEXT supersedes both: refs only, `$XDG_DATA_HOME`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | A SAML-blocked repo returns HTTP 200 with a per-alias `FORBIDDEN` error that carries `extensions.saml_failure: true` and/or a message containing "SAML"; the exact text "Resource protected by organization SAML enforcement. You must grant your … token access to this organization." is the message. Evidence: `saml_failure` exists on FORBIDDEN errors [CITED: dev.to/callmekatootie/debug-resource-not-accessible-by-integration-error-when-working-with-githubs-graphql-endpoint-5bim shows `"extensions":{"saml_failure":false}` on a FORBIDDEN]; the message text is [ASSUMED] from vendor support pages that quote it for REST. | Pattern 3, fixtures | SSO rows would show as "could not load (…)" instead of the SSO hint; no data loss. Mitigation: two independent signals plus tests; reconfirm on a real SAML org (UAT). |
| A2 | The SSO hint wording ("re-run `gh auth login` and grant the org, or open `https://github.com/orgs/<org>/sso`") fixes a gh OAuth token. GitHub docs say an OAuth/GitHub App needs an active SSO session via `https://github.com/orgs/ORGANIZATION-NAME/sso` before authorising [CITED: docs.github.com about-authentication-with-saml-single-sign-on]; that `gh auth refresh` alone grants org access is [ASSUMED]. | Pattern 3 hint | A hint that does not fix it; wording only. Prefer naming the SSO page first. |
| A3 | `repository(owner:, name:)` follows GitHub's repository rename redirect and returns the new `nameWithOwner`. | Pitfall 2 | A renamed repo shows "not found" until the user re-adds it; the code does not depend on it. |
| A4 | Tests/lookup cost stays 1 for 25 aliases of PRs with full review and request lists. Verified only with mostly-missing PRs (11 of 25 and 19 of 50 resolved) on `cli/cli`; a fully populated 25-alias chunk may cost more but is bounded by the 500,000-node limit and 2,000 points/min. | Standard Stack | Slightly higher API use; no correctness impact. Always read `rateLimit.cost`. |
| A5 | The `⚠` (U+26A0) glyph renders one cell wide in the user's terminal. `lipgloss.Width("⚠")` is 1 here, but `"⚠️"` is 2 and some terminals render emoji-capable glyphs wide. | Pattern 8 | Misaligned tail of an unavailable row only. Add it to the UAT checks (Phase 1 already deferred an ambiguous-width glyph check). |
| A6 | The leading `+` marker reads clearly next to `+N` suffixes in the CI and REVIEWERS cells. | Pattern 8 | Cosmetic. `+` is ASCII (width 1 everywhere); the alternative `◆` also measured width 1. |

## Open Questions

1. **Issue-URL wording and GitHub's redirect**
   - What we know: D-04 says issue URLs get "that is an issue, not a PR". GitHub web redirects `/issues/N` to `/pull/N` when N is a PR (project PITFALLS.md notes it).
   - What's unclear: a user pasting `…/issues/12` where 12 is a PR gets told it is an issue.
   - Recommendation: keep the rejection (locked) but word it "that is an issue link; paste the …/pull/N link". Low impact.
2. **`q` inside the remove confirmation**
   - D-06 "any other key cancels" vs Phase 3 D-09 "q means quit everywhere".
   - Recommendation: `y` removes, `ctrl+c` quits, all else (including `q`) cancels, matching the prompt modes. Confirm with the user at plan check.
3. **Non-SAML `FORBIDDEN` per D-12**
   - A token without access (fine-grained PAT scope) returns FORBIDDEN, a real answer, but D-10/D-12 only name NOT_FOUND and SSO as immediate. Recommendation: follow the decisions literally (other kind, 2-strike), since its short reason still shows after 2 refreshes.
4. **Header segment priority for `N unavailable`**
   - CONTEXT says it survives narrowing before key hints; header priorities 1-7 are taken. Recommendation: prio 6 (ties with the rate warning, which drops leftmost-first). The planner may choose 8.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | build and tests | ✓ | go1.27.2 darwin/arm64 (go.mod floor 1.26.0) | — |
| `gh` CLI, authenticated | live query checks, UAT | ✓ | 2.102.0, logged in as `sbartram` | — |
| Network to api.github.com | live verification, UAT | ✓ | — | fixtures for all automated tests |
| A SAML-enforced org | verifying the real SSO error | ✗ | — | constructed fixture (A1); UAT item deferred |
| golangci-lint | lint | ✗ (not on PATH) | — | `go vet ./...` (clean) |

`go test ./...` passes on the current branch (all five packages ok). **Missing with no fallback:** none. **Missing with fallback:** SAML org (fixture), golangci-lint (go vet).

## Security Domain

`security_enforcement` is enabled in `.planning/config.json` (ASVS level 1).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no (token handling unchanged) | Keep the AUTH-03 rule: no new code reads or stores the token; `captureTransport` keeps only its three rate-limit headers, do not add `X-GitHub-SSO` (it carries a one-time authorisation URL) |
| V3 Session Management | no | — |
| V4 Access Control | no | Read-only; no write actions |
| V5 Input Validation | yes | Anchored regexps + length and range caps in `domain.ParseRef`; owner, repo and number only ever travel as GraphQL variables; re-validate entries read from the file |
| V6 Cryptography | no | `sha256` only to name a backup; not a security use |
| V7 Error handling/logging | yes | Reasons built from fixed strings and `domain.Clean`ed API messages; no token, header or request dump in any message |
| V12 Files | yes | Fixed path under XDG; 0700 dir, 0600 file; 1 MiB read cap; atomic temp+rename; never follow a path from the file content |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| GraphQL injection through owner/repo/number | Tampering | Variables only; parser charset; test that the request text holds no user data |
| Terminal escape injection via paste, via PR titles, via API error messages | Tampering / Spoofing | `domain.Clean` on paste and on every remote string at the seam; display-width helpers |
| Hand-edited or malicious `watchlist.json` (huge, odd refs) | Tampering / DoS | Size cap, entry validation, skip-and-warn, cap on entries looked up per refresh (for example 500) |
| Opening an attacker-chosen URL with `o` | Spoofing | URL built only from a validated `Ref`, then `safeURL` host allow-list |
| Lost update between two panes | Tampering | Reload-apply-write per change; residual millisecond window accepted (documented) |
| Corrupt file silently destroyed | Repudiation / data loss | Hash-named backup copy; read path never rewrites |
| Hostile `Retry-After` / reset values | DoS | Already clamped in Phase 2/3; the lookup reuses `classify` |

## Test approach (no Validation Architecture section: `workflow.nyquist_validation` is `false`)

- **domain:** table test for `ParseRef`; `Reconcile` strike sequences (first launch, one transient, two transients, recovery, hard kinds); `Combine`/`MarkWatched` (case, rename-by-ID); `InWindow`, `Bucket` (0), `Sort` bucket 0 order, `Visible` with unavailable rows and number prefix.
- **watchlist:** `t.TempDir()` + `t.Setenv("XDG_DATA_HOME", …)`; missing file; round trip; corrupt JSON (backup exists, second load creates no second backup); unknown version; bad entries skipped; add duplicate by case; remove missing; two stores on one path interleaving add/remove; file mode 0600; no temp files left after a forced error.
- **github:** fixtures `lookup_ok.json`, `lookup_notfound.json` (live shape, copied from the probe above), `lookup_saml*.json` (constructed), `lookup_other.json`, `lookup_partial_nested.json` (error below `pullRequest`), `lookup_null_no_error.json`; chunking at 25 (26 refs → 2 requests); request-level failure returns an error and no snapshot; rate limit pauses; no-interpolation assertion; `Fetch` dedupe of an auto PR that is also saved (no lookup for it) and of a renamed repo (same ID).
- **ui:** pure `Update` tests with the existing `runCmd`/`key`/`newModel` helpers [VERIFIED: model_test.go:36-86]; add flow including late result after esc, error keeps text, duplicate flash, hidden-by-filters flash, epoch discard; remove flow (y, other key, auto-only row, manual+auto unpin); `PasteMsg` sanitising. Goldens at 0x0, 60x15, 80x24, 120x40 for: `add-checking`, `add-error`, `confirm`, `unavailable` (list), `marker` (list with manual rows), `empty` (new hint), plus compact (36x12) and the unavailable detail. Regenerate existing list goldens for the footer change.
- **cmd:** `main_test.go` replaceable `watchlistPath` variable like `configPath`.

## Sources

### Primary (HIGH confidence)
- Live GitHub GraphQL runs with `gh api graphql` (2026-10-12/13): variables in aliases, case-insensitivity, canonical `nameWithOwner`, per-alias NOT_FOUND shapes, out-of-range numbers, cost 1 at 25 and 50 aliases, private/invisible repo = NOT_FOUND.
- Go probe test against the real API (deleted after use): `DoWithContext` into `map[string]json.RawMessage`, `*api.GraphQLError` with `Type` and `Path`.
- Repository files read this session: `internal/github/client.go`, `errors.go`, `checks.go`, `ratelimit.go`, `client_test.go`; `internal/ui/model.go`, `refresh.go`, `view.go`, `keys.go`, `detail.go`, `help.go`, `layout.go`, `cells.go`, `styles.go`, `open.go`; `internal/domain/pr.go`, `list.go`, `clean.go`, `status.go`; `internal/config/config.go`; `cmd/prv/main.go`; `go.mod`; `.planning/phases/01-status-at-a-glance/01-SPIKE.md` (Q3, Q6, observed shapes).
- Module cache source: go-gh v2.16.2 `pkg/api/graphql_client.go`, `errors.go`; bubbletea v2.1.0 `paste.go`; bubbles v2.2.1 `spinner/spinner.go`, `textinput/textinput.go`.
- https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api (cost formula, 500,000-node limit, secondary limits).

### Secondary (MEDIUM confidence)
- https://dev.to/callmekatootie/debug-resource-not-accessible-by-integration-error-when-working-with-githubs-graphql-endpoint-5bim (FORBIDDEN error shape with `extensions.saml_failure`).
- https://docs.github.com/en/enterprise-cloud@latest/authentication/authenticating-with-saml-single-sign-on/about-authentication-with-saml-single-sign-on (OAuth apps need an SSO session at `https://github.com/orgs/ORGANIZATION-NAME/sso`).

### Tertiary (LOW confidence)
- Vendor support pages (Heroku, HashiCorp, Code Climate, Zilla) quoting "Resource protected by organization SAML enforcement" for REST; used only for the message text (A1).

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH. No new dependencies; everything is already pinned and used.
- Architecture: HIGH. Built on code read this session; the two risky spots (stale fetch, spanning row) have concrete recommendations. The spanning-row line replacement should be proven by a quick spike test first.
- Lookup query and per-alias errors: HIGH for NOT_FOUND and request-level cases (live); MEDIUM for SAML (A1).
- Pitfalls: HIGH.

**Research date:** 2026-10-12
**Valid until:** 2026-11-12 (stable; the GitHub GraphQL surface used here is long-lived; recheck only if go-gh or Bubble Tea are bumped)

# Stack Research

**Domain:** Go + Bubble Tea terminal UI that monitors GitHub pull requests (state, review decision, CI rollup), github.com only
**Researched:** 2026-10-11
**Overall confidence:** HIGH (versions verified live against proxy.golang.org, GitHub releases, upstream source, and a real GraphQL call on 2026-10-11; one MEDIUM area is teatest v2, see Testing)

## TL;DR (prescriptive)

- **Bubble Tea v2** (`charm.land/bubbletea/v2` v2.1.0) with **Bubbles v2** (`charm.land/bubbles/v2` v2.2.1) and **Lip Gloss v2** (`charm.land/lipgloss/v2` v2.0.6). Do not start a greenfield app on v1.
- **GitHub client: `github.com/cli/go-gh/v2` v2.16.2, `api.GraphQLClient.DoWithContext` with hand-written query strings** and typed response structs. It resolves auth the same way `gh` does and needs no extra HTTP/OAuth glue.
- **Auth: `auth.TokenForHost("github.com")`** (go-gh). It already shells out to `gh auth token` for keyring-stored tokens and reads env tokens. No custom `exec` code is needed.
- **Browser: `github.com/cli/go-gh/v2/pkg/browser`** (already in the module graph; honours `GH_BROWSER` / `BROWSER`).
- **Config: TOML via `github.com/BurntSushi/toml` v1.6.0**, at `$XDG_CONFIG_HOME/prv/config.toml` (fallback `~/.config/prv/`, on macOS too). **Watch list: JSON at `$XDG_DATA_HOME/prv/watchlist.json`** (fallback `~/.local/share/prv/`), written atomically.
- **Tests: stdlib `testing` + pure `Update`/`View` tests + golden files (`charmbracelet/x/exp/golden`); `teatest/v2` only for a few end-to-end flows.**
- **Release: GoReleaser v2.18.3, `version: 2` config, `CGO_ENABLED=0`, `homebrew_casks` (not the deprecated `brews`).**

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended | Confidence |
|------------|---------|---------|-----------------|------------|
| Go | 1.26+ (`go 1.26` in go.mod; dev toolchain here is 1.27.2) | Language | Bubble Tea v2.1.0 declares `go 1.26.0`; Bubbles/Lip Gloss need 1.25+. The go.mod floor must be 1.26. | HIGH |
| `charm.land/bubbletea/v2` | v2.1.0 (2026-10-08) | TUI runtime (Elm architecture) | v2 is stable and actively patched (v2.0.7 to v2.0.10 to v2.1.0 over Jun-Oct 2026). Charm's own apps and `bubbles` v2 target it. v1 (`github.com/charmbracelet/bubbletea` v1.3.10, last release 2025-09-17) is in maintenance and Bubbles v1.0.0 is its last companion. Starting on v1 means a forced migration later. | HIGH |
| `charm.land/lipgloss/v2` | v2.0.6 (2026-08-11) | Styling, layout, `lipgloss/table` for rendering | Required companion; Bubbles v2 depends on it. Do styling here, not with raw ANSI. | HIGH |
| `charm.land/bubbles/v2` | v2.2.1 (2026-08-24) | Prebuilt components | Use `key` + `help` (keymap and footer), `textinput` (add-PR prompt), `spinner` (refresh indicator), `viewport` (scrollable detail view). | HIGH |
| `github.com/cli/go-gh/v2` | v2.16.2 (2026-10-09) | GitHub GraphQL client, auth, browser opener | Maintained by the `gh` team; its `api` package gives one authenticated `http.Client`/GraphQL client with gh's headers, `GH_HOST`/config handling, optional response cache, and typed `*api.GraphQLError`/`*api.HTTPError`. Auth and browser come from the same module, so you add one dependency for three jobs. | HIGH |
| `github.com/BurntSushi/toml` | v1.6.0 (2025-12-18) | Config parsing | Small, no transitive deps, `MetaData.Undecoded()` lets you warn on unknown/typo'd keys. Fits a flat config (interval, window days, default filters). | HIGH |

### Important v2 API facts (these change how you write every model)

Verified against `UPGRADE_GUIDE_V2.md` in the v2.1.0 tag.

- `View()` returns `tea.View`, not `string`: `tea.NewView(s)`, then set `v.AltScreen = true`, `v.MouseMode`, `v.WindowTitle` on it. **There is no `tea.WithAltScreen()` option any more**; alt screen is declared in `View()`.
- Key input is `tea.KeyPressMsg` (not `tea.KeyMsg`, which is now an interface covering press+release). Space is `"space"`, not `" "`. Use `msg.String()` matching (`"ctrl+c"`, `"enter"`, `"o"`) or `key.Matches` from `bubbles/key`.
- Paste is `tea.PasteMsg` (matters for the paste-a-PR-URL flow). Terminal bracketed paste is on by default; handle `PasteMsg` in the `textinput`.
- Bubbles v2 components replaced exported `Width`/`Height` fields with `SetWidth()/Width()` methods. Do not copy v1 blog snippets.
- Lip Gloss v2: `lipgloss.Color("...")` is a function returning `color.Color`; `AdaptiveColor` moved to `lipgloss/v2/compat`. Prefer `lipgloss.LightDark(hasDark)` fed from Bubble Tea's background-color message, or just pick a palette that reads on both backgrounds (ANSI 0-15 colors) for a tool that lives in a multiplexer pane.
- Lip Gloss v2 is pure: it no longer touches stdout. Bubble Tea handles colour downsampling; do not call `fmt.Println` styled strings outside the program (use `lipgloss.Println` for any pre/post-TUI output).

### GitHub API access

| Choice | Detail |
|--------|--------|
| Transport | GraphQL v4 only. REST is not needed for v1 scope. |
| Client | `api.NewGraphQLClient(api.ClientOptions{Host: "github.com", Timeout: 20*time.Second, LogIgnoreEnv: true})` then `client.DoWithContext(ctx, query, vars, &resp)` |
| Query style | Raw query string constants + typed `json`-tagged response structs, not struct-reflected queries |
| Cost control | One batched query per refresh, `statusCheckRollup { state }` only in the list query, per-check detail fetched on demand for the selected PR |

**Why go-gh `api` over githubv4 (shurcooL):** `shurcooL/githubv4` (pseudo-version `v0.0.0-20260209...`, no tagged releases) needs you to build an `oauth2` transport and resolve the token yourself, then express a deep, aliased, inline-fragment query as a reflected Go struct. go-gh gives auth + host + headers + error types for free and `DoWithContext` accepts a literal query, which is clearer for `... on CheckRun` / `... on StatusContext` unions and for aliased batches (`pr0: repository(...) { pullRequest(...) }`). go-gh's own `Query()` is the same shurcooL engine (`cli/shurcooL-graphql` v0.0.4) if you ever want struct-style queries.

**Why not REST `google/go-github` (v90.0.0):** a PR's review decision, per-reviewer latest state, and the check rollup need several REST calls per PR (pulls, reviews, check-runs, statuses) and REST has no `reviewDecision`. GraphQL returns all of it in one request. With N watched PRs REST is O(N x 4) calls per refresh against a 5,000 req/hr budget; GraphQL is O(1).

**Verified GraphQL facts (2026-10-11, live call as the authed user):**
- `PullRequest` exposes `state`, `isDraft`, `closedAt`, `mergedAt`, `reviewDecision`, `latestOpinionatedReviews`, `latestReviews`, `reviewRequests`, `mergeable`, `mergeStateStatus`, **and `statusCheckRollup` directly** (no need for `commits(last:1){...}`).
- `search(query: "is:pr author:@me", type: ISSUE, first: N)` works; `@me` resolves to the token's user. `assignee:@me` works too. Authored+assigned means two searches, deduplicated client-side by PR node ID/URL (OR across different qualifiers is not reliable in search syntax).
- A 50-PR page with `latestOpinionatedReviews(first:10)` and `statusCheckRollup.contexts(first:50)` reported **`rateLimit.cost = 1`**. Always request `rateLimit { cost remaining resetAt }` and surface `remaining` in a debug/status line.
- Documented limits: 5,000 points/hr per user (PAT / gh OAuth token); secondary limits of 2,000 points/min to GraphQL and 100 concurrent requests. A 30-60 s refresh of a one-or-two-query batch is far inside both.
- GraphQL POSTs cannot use ETag/304; savings come from query batching and slimming, not conditional requests. Do not enable go-gh's `EnableCache` for polling (it would serve stale data for its TTL, default 24 h).

### Authentication

| Choice | Detail |
|--------|--------|
| Function | `auth.TokenForHost("github.com")` from `github.com/cli/go-gh/v2/pkg/auth`, called once at startup, token passed explicitly via `ClientOptions.AuthToken` |
| What it does (source-verified, v2.16.2) | 1) `GH_TOKEN`, 2) `GITHUB_TOKEN`, 3) `oauth_token` in `~/.config/gh/hosts.yml`, 4) shells out to `gh auth token --secure-storage --hostname github.com` if `gh` is on PATH (or `GH_PATH`) |
| No token | Returns `"", "default"`; show a clear "run `gh auth login` or set GITHUB_TOKEN" screen and exit non-zero before starting the TUI |

**Why not hand-rolled `exec.Command("gh","auth","token")`:** it duplicates what go-gh already does, and misses the `hosts.yml` plaintext path and `GH_PATH`. You would still need the env fallback. Fewer code paths = fewer bugs.

**DECISION FLAG for the roadmap:** PROJECT.md says "reuse gh login, fall back to `GITHUB_TOKEN`". go-gh (and `gh` itself) do the reverse: **env tokens win over the gh login.** That is the established convention and what the user's other gh-based tools do, so recommend adopting it and updating the wording. If the user truly wants gh-first, implement it by calling `auth.TokenForHost` only after trying `exec gh auth token` yourself; this is about 10 lines but diverges from `gh` behaviour (a stale `GITHUB_TOKEN` exported in a shell will be ignored by prv but honoured by gh).

**Token refresh:** resolve once, but on HTTP 401 re-run `TokenForHost` and rebuild the client once before surfacing an error. The TUI is long-running and `gh auth refresh` can rotate the token mid-session.

### Opening URLs

| Choice | Detail |
|--------|--------|
| Library | `github.com/cli/go-gh/v2/pkg/browser`: `browser.New("", io.Discard, io.Discard).Browse(url)` |
| Why | Already a dependency; wraps `cli/browser` v1.3.0; honours `GH_BROWSER`, gh config `browser`, and `BROWSER`, so it opens the same browser as `gh pr view --web`. |
| Gotcha | Pass `io.Discard` (or a log file) for stdout/stderr. If a custom launcher writes to the real terminal while Bubble Tea owns the screen, it corrupts the display. Run it inside a `tea.Cmd`, not in `Update`. |

Not recommended: `pkg/browser` (last commit 2024-01, unmaintained), hand-rolled `open`/`xdg-open` switch (reimplements the above).

### Config and persistence

| Item | Choice | Location |
|------|--------|----------|
| User settings (hand-edited) | TOML, `BurntSushi/toml` v1.6.0 | `$XDG_CONFIG_HOME/prv/config.toml`, default `~/.config/prv/config.toml` |
| Watch list (app-written) | JSON, stdlib `encoding/json`, atomic write (temp file in same dir + `os.Rename`) | `$XDG_DATA_HOME/prv/watchlist.json`, default `~/.local/share/prv/watchlist.json` |
| Debug log (opt-in `--debug`) | stdlib `log/slog` to a file | `$XDG_STATE_HOME/prv/prv.log`, default `~/.local/state/prv/prv.log` |

**XDG on macOS:** the user is on macOS. `os.UserConfigDir()` and the `adrg/xdg` library (v0.5.3) default to `~/Library/Application Support` there. `gh`, `gh-dash`, `lazygit`-style tools use `~/.config`, so users expect dotfile-style paths. Write a ~10-line helper: use `$XDG_*_HOME` if set, else `~/.config`, `~/.local/share`, `~/.local/state`, on every OS. This avoids a dependency and the surprise. (If Windows ever becomes a target, switch to `adrg/xdg`.)

**Split config from data:** config is human-edited and may live in dotfiles; the watch list is machine-written state. Mixing them means the app rewrites a file the user hand-edits, destroying comments. Keep the watch list a versioned JSON object (`{"version":1,"prs":[{"owner":"o","repo":"r","number":123}]}`) so the format can change.

**Config keys likely needed (defaults in code, file optional):** `refresh_interval` (default 60s, floor 15s), `closed_window_days` (14), default state filters. Use `toml.Decode` into a struct pre-filled with defaults; validate and clamp the interval.

**Not recommended:** Viper (large dependency tree, global state, overkill for ~5 keys), YAML (gh-dash uses it, but it adds nothing here and TOML is less error-prone for hand edits), storing state in the config file, SQLite/bbolt (a watch list is tens of entries).

### CLI parsing

stdlib `flag` is enough (`--debug`, `--config`, `--version`). Cobra v1.10.2 is justified only if subcommands appear (e.g. `prv add <url>`); defer.

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `charm.land/lipgloss/v2/table` | (part of lipgloss v2.0.6) | Render the PR list as a table with a `StyleFunc` | Use for the list instead of `bubbles/table`: Bubbles' table has no per-cell style hook, so coloured status badges inside cells fight the selected-row style. Keep your own cursor index and highlight inside `StyleFunc`. |
| `charm.land/bubbles/v2/viewport` | (bubbles v2.2.1) | Scrolling in list (if >screen) and detail view | Window the rows yourself, or render the list into a viewport. |
| `github.com/charmbracelet/x/exp/golden` | v0.1.0 (tagged 2026-09-24) | Golden-file assertions for `View()` output | Snapshot tests of list, detail, empty, and error states at fixed sizes. |
| `github.com/charmbracelet/x/exp/teatest/v2` | pseudo-version `v2.0.0-20261011004359-2acd7e345453` (untagged) | Drive a real `tea.Program` in tests | 2-4 end-to-end tests (startup, refresh tick, add PR, quit). See Testing. |
| `github.com/stretchr/testify` | v1.12.1 | Assertions | Optional; stdlib `testing` + `cmp.Diff` is also fine. |
| stdlib `log/slog`, `net/http/httptest`, `regexp`, `time` | n/a | Logging, fake API, PR URL parsing, ticks | Parse `https://github.com/o/r/pull/123` and `o/r#123` with one anchored regexp; go-gh's `repository.Parse` is for `OWNER/REPO` only. |

### Development Tools

| Tool | Version | Purpose | Notes |
|------|---------|---------|-------|
| GoReleaser | v2.18.3 (2026-10-09) | Cross-compile, archive, checksum, GitHub Release, Homebrew cask | `.goreleaser.yaml` must start with `version: 2`. |
| golangci-lint | v2.14.0 | Lint | v2 config format (`version: "2"`). |
| `goreleaser/goreleaser-action` + `actions/setup-go` | latest majors | CI release on tag | Pin by major tag; use `go-version-file: go.mod`. |
| `gh api graphql` | gh 2.102.0 locally | Prototype/inspect queries | Use `gh api graphql -f query=...` to validate a query and read `rateLimit.cost` before coding it. |

## Testing (Bubble Tea v2)

1. **Unit-test the model, not the terminal.** `Update(tea.KeyPressMsg{Code: 'o', Text: "o"})` returns `(model, cmd)`; assert on the model and on the cmd's message by executing the cmd. Because `View()` returns `tea.View`, assert on `m.View().Content` (strip ANSI first, or set a `lipgloss` colour profile of `ascii` in tests).
2. **Inject the GitHub client behind a small interface** (`FetchPRs(ctx) ([]PR, error)`, `FetchChecks(ctx, ref) ([]Check, error)`). The model only sees interface values, so tests use an in-memory fake. This is the main reason refreshes are testable without a network.
3. **Test the GraphQL layer separately** with go-gh's `ClientOptions{Host:"github.com", AuthToken:"x", Transport: fakeRoundTripper}`. Setting all three skips config/token resolution entirely (verified in `optionsNeedResolution`), so tests never touch the real `gh` login. Feed recorded JSON fixtures (captured with `gh api graphql`) including the awkward cases: no checks (`statusCheckRollup: null`), `StatusContext` entries, `reviewDecision: null`, merged PR, deleted-fork PR.
4. **Golden-file `View()`** at fixed `tea.WindowSizeMsg{Width: 100, Height: 30}` and one narrow size (60x20) with `x/exp/golden`.
5. **teatest v2** (`teatest.NewTestModel`, `WaitFor`, `FinalModel`) for a handful of flows. **MEDIUM confidence:** the v2 package has no tagged release (pseudo-version only) and its `go.mod` still requires `bubbletea v2.0.0-rc.1`; Go's MVS will lift that to v2.1.0 because your go.mod requires it, and the source builds against the current API, but pin the exact pseudo-version and run one spike test in the first phase to confirm. The v1 teatest (`github.com/charmbracelet/x/exp/teatest`) targets Bubble Tea v1 and will not compile against `charm.land/bubbletea/v2`. If the spike fails, fall back to steps 1-4, which cover nearly everything.
6. Use `tea.WithoutSignals()` and fixed sizes in tests; avoid real timers (inject the tick interval so a test can use 10 ms).

## Build and release

```yaml
# .goreleaser.yaml (sketch)
version: 2
project_name: prv
builds:
  - main: ./cmd/prv
    env: [CGO_ENABLED=0]
    goos: [darwin, linux]       # add windows only if you will test it
    goarch: [amd64, arm64]
    ldflags: ["-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}"]
archives:
  - formats: [tar.gz]
checksum:
  name_template: checksums.txt
homebrew_casks:                 # NOT `brews`: deprecated since v2.10 (soft) / v2.16
  - repository: { owner: <you>, name: homebrew-tap }
```

- `CGO_ENABLED=0` works: no dependency in this stack needs cgo. That delivers the "single static binary" constraint.
- GoReleaser deprecation (verified at goreleaser.com/deprecations): `brews` is soft-deprecated since v2.10 and deprecated since v2.16; use `homebrew_casks`. Casks are macOS-focused; if Linux brew users matter, ship tar.gz plus `go install` and skip Linux brew. Homebrew publishing needs a separate tap repo and a PAT; treat as optional for a personal tool.
- `go install github.com/<you>/prv/cmd/prv@latest` works for free as long as the module path and `cmd/prv` layout are right. Version injection falls back to `debug.ReadBuildInfo()` for that path.
- Run `goreleaser release --snapshot --clean` in CI on PRs to catch config drift; tag-triggered job does the real release.
- Start with release as a late phase; it does not constrain earlier design beyond the `cmd/prv` layout and `main.version` variables.

## Installation

```bash
go mod init github.com/<you>/prv        # go.mod: go 1.26

# Core TUI (all three from charm.land, v2 line)
go get charm.land/bubbletea/v2@v2.1.0
go get charm.land/bubbles/v2@v2.2.1
go get charm.land/lipgloss/v2@v2.0.6

# GitHub client, auth, browser
go get github.com/cli/go-gh/v2@v2.16.2

# Config
go get github.com/BurntSushi/toml@v1.6.0

# Test-only
go get github.com/charmbracelet/x/exp/golden@v0.1.0
go get github.com/charmbracelet/x/exp/teatest/v2@v2.0.0-20261011004359-2acd7e345453   # spike first
go get github.com/stretchr/testify@v1.12.1                                             # optional

# Tools (not in go.mod)
brew install goreleaser golangci-lint
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Bubble Tea v2 | Bubble Tea v1.3.10 | Only if a required third-party Bubble is v1-only and cannot be vendored. Not the case here (no third-party bubbles needed). |
| go-gh `api.GraphQLClient` | `shurcooL/githubv4` + `oauth2` | If you want compile-time-typed queries via struct reflection and don't mind owning token resolution. Reasonable, but more glue and no tagged releases. |
| go-gh `api.GraphQLClient` | `Khan/genqlient` v0.8.1 (codegen from schema) | If the query set grows large and you want generated types. Overkill for 2-3 queries and needs the GitHub schema file checked in. |
| go-gh `api.GraphQLClient` | `google/go-github` v90.0.0 (REST) | Only for features GraphQL lacks. Not needed for v1; also costs N x several calls per refresh. |
| `auth.TokenForHost` | `exec gh auth token` + `os.Getenv` | If you must have gh-first precedence (see decision flag). |
| go-gh `pkg/browser` | `cli/browser`, `pkg/browser`, hand-rolled | `cli/browser` directly if you drop go-gh; never `pkg/browser` (stale). |
| TOML (`BurntSushi/toml`) | `pelletier/go-toml/v2` v2.4.3 | Equivalent; pick it if you want marshal-with-comments or stricter decoding. Either is fine. |
| TOML | YAML (gh-dash style) | If you want config format parity with gh-dash. Not a technical reason. |
| Hand-rolled XDG helper | `adrg/xdg` v0.5.3 | If Windows/macOS-native paths (`~/Library/Application Support`) are wanted, or Windows becomes a target. |
| `lipgloss/table` + own cursor | `bubbles/table` v2 | Plain monochrome rows with no coloured cells. |
| `lipgloss/table` + own cursor | `bubbles/list` | Single-column item lists with built-in fuzzy filter. Its pagination/filter model fights a multi-column dashboard with a state-filter toggle. |
| stdlib `flag` | Cobra v1.10.2 | If subcommands (`prv add`, `prv remove`) are introduced. |
| stdlib `slog` | `charmbracelet/log` v1.0.0 | Pretty colourised logs; unneeded since logs go to a file. |
| `homebrew_casks` | `brews` | Never for new work (deprecated). |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `github.com/charmbracelet/bubbletea` (v1 path), `github.com/charmbracelet/lipgloss`, `github.com/charmbracelet/bubbles` | v1 import paths. Mixing v1 and v2 in one binary compiles but gives two runtimes and type mismatches; most search results and LLM-remembered snippets still use these paths. | `charm.land/*/v2` imports everywhere. |
| `tea.KeyMsg` struct patterns, `tea.WithAltScreen()`, `tea.EnterAltScreen`, `View() string`, `case " ":` | v1 API, removed or changed in v2 | `tea.KeyPressMsg`, `View().AltScreen = true`, `tea.View`, `"space"` |
| `github.com/charmbracelet/x/exp/teatest` (non-/v2) | Built against Bubble Tea v1 | `.../teatest/v2` |
| `go-gh` `api.ClientOptions{EnableCache: true}` for polling | Caches responses for TTL (default 24 h); you would display stale CI status | Leave caching off |
| go-gh with `GH_DEBUG` set and default `Log` options | go-gh logs HTTP traffic to stderr when `GH_DEBUG` is set, which paints over the TUI | `LogIgnoreEnv: true`, or set `Log` to your log file |
| Per-PR REST calls (`/pulls`, `/reviews`, `/check-runs`) | O(N x 4) requests per refresh, no `reviewDecision` | One batched GraphQL query |
| Fetching all check contexts for every PR in the list query | Needless payload; list only needs the rollup `state` | Fetch contexts for the selected PR on demand |
| `time.Sleep`/goroutine pollers sending on channels | Bypasses the Elm loop, hard to test and cancel | `tea.Tick` / `tea.Every` commands that return a message; cancel via `context` stored on the model |
| Viper, SQLite/bbolt, keyring libs (`zalando/go-keyring`) | Overkill; and token storage is `gh`'s job | TOML + JSON files; `go-gh/auth` |
| Writing the token to prv's config | Security and redundancy | Never persist the token |
| `fmt.Println`/`log.Println` to stdout/stderr while the TUI runs | Corrupts the screen | `slog` to file; show errors in a status line |

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| bubbletea v2.1.0 | Go >= 1.26 | `go.mod` says `go 1.26.0`. |
| bubbles v2.2.1 | bubbletea >= 2.0.8, lipgloss >= 2.0.5 | MVS lifts to your pinned newer versions; upgrade the three together (`go get` all three). |
| go-gh v2.16.2 | Go >= 1.25 | Pulls lipgloss **v1** and termenv transitively for its `term`/`tableprinter` packages. Those packages are not imported by `api`/`auth`/`browser`, so they are not linked into the binary, but `go.sum` will contain them. Not a runtime conflict (different module path from `charm.land/lipgloss/v2`). |
| teatest/v2 (pseudo) | bubbletea v2.x | Untagged; pin the pseudo-version. |

## Confidence summary

| Recommendation | Confidence | Basis |
|----------------|------------|-------|
| Bubble Tea v2.1.0 + Bubbles v2.2.1 + Lip Gloss v2.0.6 | HIGH | proxy.golang.org, GitHub release notes, upgrade guides at the tags |
| go-gh `api` GraphQL client, raw queries | HIGH | Source read at v2.16.2; live query run |
| `auth.TokenForHost` behaviour and order | HIGH | Source read at v2.16.2 |
| GraphQL field availability and cost = 1 | HIGH | Live introspection and call on 2026-10-11 with gh 2.102.0 |
| Two-search (authored, assigned) + dedupe | MEDIUM | Each search verified; OR-combining qualifiers not tested, so two aliased searches in one request is the safe plan |
| go-gh `pkg/browser` | HIGH | Source read |
| TOML + XDG layout, hand-rolled helper | MEDIUM | Convention-based judgement; no technical blocker either way |
| Test strategy; teatest v2 | MEDIUM | teatest/v2 is untagged; compatibility to be proven by a spike |
| GoReleaser v2.18.3, `homebrew_casks` | HIGH (version, deprecation) / MEDIUM (cask suitability for a Linux audience) | goreleaser.com deprecations page, GitHub releases API |

## Sources

- proxy.golang.org `@latest` and `.mod` files for bubbletea/v2 (v2.1.0), bubbles/v2 (v2.2.1), lipgloss/v2 (v2.0.6), go-gh/v2 (v2.16.2), BurntSushi/toml (v1.6.0), x/exp/golden (v0.1.0), x/exp/teatest/v2 (pseudo), checked 2026-10-11 (HIGH)
- https://github.com/charmbracelet/bubbletea/releases (v2.1.0 published 2026-10-08) and `UPGRADE_GUIDE_V2.md` at tag v2.1.0 (HIGH)
- https://github.com/charmbracelet/bubbles `UPGRADE_GUIDE_V2.md` at v2.2.1; https://github.com/charmbracelet/lipgloss `UPGRADE_GUIDE_V2.md` at v2.0.6 (HIGH)
- https://github.com/cli/go-gh at v2.16.2: `pkg/auth/auth.go`, `pkg/api/graphql_client.go`, `pkg/api/client_options.go`, `pkg/browser/browser.go` (HIGH, primary source)
- https://github.com/charmbracelet/x/tree/main/exp/teatest/v2 `teatest.go` (MEDIUM, untagged)
- https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api (HIGH)
- Live `gh api graphql` introspection and test query, 2026-10-11 (HIGH)
- https://goreleaser.com/deprecations/ and https://goreleaser.com/customization/builds/go/ ; GitHub releases API for goreleaser v2.18.3 and golangci-lint v2.14.0 (HIGH)
- Context7 library index lists `/charmbracelet/bubbles` (v2.0.0 tag) and the bubbletea-app-template; upstream upgrade guides were used instead as they track the newer tags (HIGH)

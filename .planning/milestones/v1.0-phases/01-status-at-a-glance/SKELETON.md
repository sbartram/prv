# Walking Skeleton — prv

**Phase:** 1
**Generated:** 2026-10-11
**Built by:** `01-01-PLAN.md` Task 1 (the tracer). Later plans in Phase 1 and Phases 2-4 add slices on top of it and do not change the decisions below.

## Capability Proven End-to-End

Running `prv` with the user's existing `gh` login fetches their authored and assigned PRs live from the GitHub GraphQL API and shows them in a Bubble Tea v2 list that moves with j/k and the arrow keys and closes with `q`.

## Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Language / build | Go, `go.mod` floor `go 1.26.0` (no `toolchain` line), module `github.com/sbartram/prv`, binary at `cmd/prv` | Single static binary (`CGO_ENABLED=0`). Bubble Tea v2.1.0 needs Go 1.26. `go install github.com/sbartram/prv/cmd/prv@latest` is the install path (CONF-02). |
| TUI framework | `charm.land/bubbletea/v2` v2.1.0, `charm.land/bubbles/v2` v2.2.1 (spinner), `charm.land/lipgloss/v2` v2.0.6 (styles, `lipgloss/table`) | Locked project stack (`.claude/CLAUDE.md`). v2 API only: `tea.KeyPressMsg`, `View() tea.View` with `AltScreen = true`. |
| Data layer | No local store in Phase 1. The "data layer" is one live read: a single `search(type: ISSUE_ADVANCED)` GraphQL query (`01-SPIKE.md`, "The proven Phase 1 query"), mapped once into `domain.Snapshot` in memory | prv is read-only (watch-only is a project decision). Phase 4 adds the watch-list file; Phase 2 adds the config file. |
| GitHub client | `github.com/cli/go-gh/v2` v2.16.2 `api.NewGraphQLClient` + `DoWithContext`, hand-written query constant, unexported typed wire structs, one wire-to-domain mapper | Same headers and error types as `gh`; partial data is decoded before `*api.GraphQLError` is returned. `LogIgnoreEnv: true` so `GH_DEBUG` never paints the screen or logs the token. |
| Auth | `auth.TokenForHost("github.com")` once in `cmd/prv` before the TUI starts; token passed only as `api.ClientOptions.AuthToken`; never logged, shown or written | Same precedence as `gh`: `GH_TOKEN`, `GITHUB_TOKEN`, hosts.yml, then `gh auth token` (AUTH-01). No token: D-11 message on stderr, exit 1 (AUTH-02). |
| Deployment target | Local `go build ./cmd/prv` / `go install ./cmd/prv`; after the user pushes, `go install github.com/sbartram/prv/cmd/prv@latest` | No server. GoReleaser and Homebrew are deferred (v2 requirement UTIL-04). |
| Directory layout | `cmd/prv` (flags, token, wiring, exit codes) · `internal/domain` (pure types and derivations: `DeriveReview`, `ClassifyRollup`, `InWindow`, `Sort`, `Clean`) · `internal/github` (query, wire structs, mapper, typed errors) · `internal/ui` (Bubble Tea model, view, styles, layout) | `internal/ui` never imports `internal/github`; it depends on its own `ui.Fetcher` interface and on `internal/domain`. All I/O runs in a `tea.Cmd`; `Update` and `View` are pure. |
| Tests | stdlib `testing`; table tests for domain; a fake `http.RoundTripper` serving synthetic GraphQL JSON for the client; golden files (`github.com/charmbracelet/x/exp/golden`) for `View()`; one live end-to-end test gated by `PRV_LIVE=1` | Deterministic CI-safe tests plus one real read against api.github.com. No real PR data or `Authorization` headers in fixtures. |

## Stack Touched in Phase 1

- [ ] Project scaffold: `go.mod`/`go.sum`, `go build`, `go vet`, `go test`, `gofmt`
- [ ] Routing: the CLI entry point `cmd/prv` with `--version` and the hidden `--closed-days N` flag
- [ ] Database: one real read (the live GraphQL search). There is no write: prv is watch-only by project decision, and the only persistent store (the watch list) arrives in Phase 4
- [ ] UI: an interactive Bubble Tea v2 list (j/k/arrows move the selection, `q`/`ctrl+c` quit, `r` retries from the error panel)
- [ ] Deployment: `go install ./cmd/prv` locally (automated), and `go install github.com/sbartram/prv/cmd/prv@latest` after the user pushes (user-gated checkpoint in `01-04-PLAN.md`)

## Out of Scope (Deferred to Later Slices)

- Auto-refresh, `r` refresh-anytime with single-flight, rate-limit display and backoff, attention sort, Open/Merged/Closed filters, number filter, `config.toml`, open-in-browser, help screen (Phase 2)
- Reviewer names, merge-readiness, failing check names, per-PR detail view, narrow-pane and resize layout (Phase 3)
- Add by URL or `owner/repo#n`, remove, watch-list persistence, per-PR "unavailable" rows (Phase 4)
- GoReleaser, Homebrew cask, `--debug` log file, `--json` output (later / v2)

## Subsequent Slice Plan

Each later phase adds one vertical slice on top of this skeleton without changing its architectural decisions:

- Phase 2: Live Monitoring. The same `ui.Fetcher` and `domain.Snapshot` are re-fetched on a tick; `domain.Snapshot.RateLimit` (decoded since Phase 1) drives the status bar and backoff; `--closed-days` becomes the config-file closed window (CONF-01, FILT-02).
- Phase 3: Blocker Detail. The discovery query gains reviewer and merge-state fields; a detail view fetches check contexts.
- Phase 4: Curated Watch List. Aliased PR lookups on the same go-gh client; a versioned JSON store at `$XDG_DATA_HOME/prv/watchlist.json`.

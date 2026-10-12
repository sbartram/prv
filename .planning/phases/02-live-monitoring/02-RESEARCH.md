# Phase 2: Live Monitoring - Research

**Researched:** 2026-10-11
**Domain:** Long-running Bubble Tea v2 TUI. Single-flight polling, rate-limit backoff, stable sort and selection, TOML config, filters, help overlay, open-in-browser.
**Confidence:** HIGH (code read this session, library source read at the pinned tags, two spikes run, live GraphQL checked). MEDIUM on the numeric backoff thresholds, which are tunables and not facts.

## Summary

Phase 2 adds almost no new libraries. The one new dependency is `github.com/BurntSushi/toml` v1.6.0. Everything else is already in `go.mod`: Bubble Tea v2.1.0, Bubbles v2.2.1 (`key`, `help`, `spinner`), Lip Gloss v2.0.6, go-gh v2.16.2. The work is mostly a state machine in `internal/ui`, three small pure additions in `internal/domain`, and extra data coming out of `internal/github`. The hard parts are correctness under time, not API surface: one tick chain that cannot double, a generation counter, a cursor that follows a ref, and a backoff that reads the right header.

Three findings change how the existing code must be used. (1) `Model.Init()` has a value receiver, so it cannot set `inFlight` or `gen`. The first fetch must be marked in `New()`. (2) `tea.Tick` starts its timer when the Cmd is *created*, not when it runs, and it blocks when run. The existing test helper `runCmd` runs every Cmd that `Init` returns, so `Init` must never return a tick; the tick is armed only from `Update` after a result. (3) When GitHub's *primary* GraphQL limit is exhausted the HTTP status is still 200 and the failure is in the body, so go-gh returns a `*api.GraphQLError` that carries **no headers**. The reset time is only reachable through a response-header capture in the `http.RoundTripper` (the `Options.Transport` seam already exists). A `*api.HTTPError` (403/429, secondary limit) does carry `Headers`.

The config spike showed two traps in BurntSushi/toml. A TOML integer for a native `time.Duration` field (`refresh_interval = 60`) decodes silently to 60 **nanoseconds**, and a wrong-type error (`closed_days = "x"`) is a plain `errors.errorString`, not a `toml.ParseError`, so its line number must be parsed from the message. Both have a verified fix below.

**Primary recommendation:** Build the refresh loop as a pure, clock-injected state machine (`gen`, `inFlight`, `failures`, `pausedUntil`, a `nextDelay` pure function) inside the existing `Model`, with the `tick` constructor and `now` injectable, so the "simulated hour with `-race`" test is a loop over a virtual clock and not a real wait. Put sort buckets, the filter predicate and `nextDelay` in `internal/domain` as pure functions.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Attention sort (LIST-08, RFSH-06)**
- **D-01:** Buckets, top to bottom. The first rule that matches wins:
  1. failing CI (open, non-draft)
  2. changes requested
  3. ready to merge: approved and CI passing
  4. CI running
  5. everything else that is open and not a draft (review required / no review status)
  6. drafts
  7. merged
  8. closed (then unknown state)
- **D-02:** Within a bucket the order is `createdAt` newest first, then ref (owner/repo, then number as an integer). Neither key changes when polling. `updatedAt` is **not** a sort key any more: it replaces D-06's secondary key from Phase 1, because comments and pushes would make rows inside a bucket swap places.
- **D-03:** When a refresh moves a PR to a different bucket, the PR moves at once. The cursor follows the selected PR by ref. When the selected PR leaves the list, the cursor goes to its nearest neighbour (roadmap pitfall).
- **D-04:** A draft always sorts into the draft bucket (6), even with failing CI. It stays dimmed, and its badges still show the true values (Phase 1 D-13).

**Status bar & refresh (RFSH-01..05)**
- **D-05:** The header shows `updated HH:MM:SS`, which is a clock time with no redraw every second. A spinner shows next to it while a refresh is running. Example: `prv · @sbartram · 12 PRs · updated 14:32:05 ⠋`.
- **D-06:** The rate limit always shows as a faint `api 4812` at the right end of the footer. This meets success criterion 2 and keeps the limit visually quiet. When `remaining` falls below 20% of `limit`, it also shows in the header in yellow with the reset time.
- **D-07:** When a refresh fails and data is on screen, the header shows `✗ refresh failed HH:MM (<short reason>) · data from HH:MM:SS` in an ANSI warning colour. The rows are **not** dimmed, because faint already means draft/merged/closed. The next successful refresh clears the marker.
- **D-08:** During a backoff (limit low or exhausted, or retry-after active), the header shows `paused until HH:MM (rate limit)`. Pressing `r` shows a short message (`rate limited — resumes HH:MM`) and sends **no** request.
- **D-09:** When the first fetch fails and there is no data yet, the Phase 1 error panel and hint stay (Phase 1 D-09), and the auto-refresh tick keeps retrying with backoff. The panel shows `retrying at HH:MM`. `r` still retries at once, unless a rate-limit backoff blocks it (D-08).

**Filters (FILT-01..03)**
- **D-10:** `1` / `2` / `3` toggle Open / Merged / Closed visibility, in any combination. The header shows which states are active (for example `[open merged ·]`). Letter keys are not used, because `o` is open-in-browser.
- **D-11:** `/` opens a one-line prompt at the bottom that accepts digits only. The list narrows live as the user types, by **prefix** match on the PR number (`12` matches #12, #120, #1234). Enter keeps the filter and returns to list navigation. Esc clears it. The prompt shows a count, such as `(2 of 14 · enter keep · esc clear)`.
- **D-12:** At launch all three states are visible, unless the config key `show` sets a different default. Toggles last only for the session; prv saves no state to disk in Phase 2.
- **D-13:** When the filters hide every PR, prv shows a filtered-empty message that is different from the D-10 empty state of Phase 1: `No PRs match: [open · ·] #99 — 3 hidden by filters · esc clear · 1/2/3 toggle`. A filter must never look as if the list was lost.

**Config & help (CONF-01, FILT-02, NAV-03)**
- **D-14:** The config file is flat TOML at `$XDG_CONFIG_HOME/prv/config.toml`, with fallback `~/.config/prv/config.toml` (macOS too). It has three keys:
  ```toml
  refresh_interval = "60s"                        # Go duration; < 15s is raised to 15s
  closed_days      = 14                           # clamped to 1..365
  show             = ["open", "merged", "closed"] # default visibility
  ```
  A missing file means all defaults.
- **D-15:** prv exits before the TUI starts on a TOML syntax error, a wrong type or a value that cannot be parsed (for example `refresh_interval = "soon"`). It prints `prv: <path> line N: <problem>` to stderr and exits with code 2. Unknown or misspelt keys (`MetaData.Undecoded()`) do not stop prv: they show as a one-time warning in the TUI. Out-of-range numbers are clamped (D-14), not rejected.
- **D-16:** The hidden `--closed-days` flag stays as a UAT override. The flag beats the config, and the config beats the default. Both share the 1..365 bound, which fixes WR-01. No other CLI flags are added.
- **D-17:** `?` opens a full-screen help overlay. It shows the key table, a legend of badge glyphs, the effective settings (interval, closed window, config path) and the API remaining/limit with the reset time. Any key returns to the list. Refreshes continue behind the overlay. The footer hints and the help screen come from one keymap (`bubbles/key` + `help`).

**Carried forward (do not re-litigate)**
- Single-flight refresh with one tick chain that re-arms after each result. A generation counter drops stale ticks and stale results. A manual `r` merges with a refresh that is already running (roadmap pitfalls).
- Backoff order: retry-after, then `resetAt`, then a bounded exponential. Every query includes `rateLimit { cost remaining resetAt }` (add `limit` if needed for D-06).
- Cancel the context on quit. Wrap Cmds so that a panic is caught (existing `fetchCmd` pattern). Logs go to a file only.
- `o` opens only `https://github.com` URLs, through go-gh `pkg/browser` with `io.Discard`, inside a `tea.Cmd`.
- The palette is ANSI-16 only, and every badge and marker reads correctly without colour (Phase 1 D-01, D-05).
- Verify the refresh with a counting fake client over a simulated hour, with `-race`.

### Claude's Discretion
- Whether "approved + no checks" counts as bucket 3 (ready) or bucket 5. Choose and document it, keeping in mind that "no checks" is never shown as passing.
- The exact wording and glyphs of the header and footer messages, within D-05..D-09.
- How the footer is ordered and cut on narrow panes. Integrity notes (truncated, partial data, config warnings) must survive before the key hints do (WR-03). Show partial-fetch warnings on the empty and filtered-empty screens too (WR-02).
- The feedback when `o` fails (no browser, non-github URL): a short message in the footer is enough.
- How long transient messages (D-08 refusal, `o` failure) stay before they clear.
- Debug log file path and the `--debug` gating (CLAUDE.md suggests `$XDG_STATE_HOME/prv/prv.log`), if logging is added in this phase.

### Deferred Ideas (OUT OF SCOPE)
None. The discussion stayed within phase scope. (Not in this phase per the domain boundary: reviewer names, merge-readiness, failing check names, the detail view and narrow-pane layout (Phase 3); add/remove by URL and persistence (Phase 4).)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| RFSH-01 | Auto-refresh on a configurable interval (default 60s, min 15s) | Tick chain re-armed from `Update` after each result; floor enforced in `internal/config`, not in the model (so tests can use ms intervals). Patterns 1, 2. |
| RFSH-02 | `r` refreshes immediately | `r` bumps `gen` when idle, coalesces when `inFlight`, refused while a rate-limit pause is active. Pattern 1. |
| RFSH-03 | Time of last successful update shown | `lastOK` set only on success; header `updated HH:MM:SS` (absolute time, no per-second redraw). Pattern 5. |
| RFSH-04 | Failed refresh keeps last data and shows an error indicator | Failure branch never touches `snap`; `Short()` reason via a local interface. Patterns 1, 5. |
| RFSH-05 | Remaining rate limit shown; automatic backoff when low or hit | Add `limit` to the query; `nextDelay` pure function; header capture in the Transport for the HTTP-200 primary-limit case. Patterns 2, 3. |
| RFSH-06 | Selected PR stays selected across refreshes | Track `selected domain.Ref`; `reselect` with nearest-neighbour fallback; remove the `m.cursor, m.offset = 0, 0` reset. Pattern 4. |
| CONF-01 | Settings read from `~/.config/prv/config.toml`, defaults if missing | New `internal/config` package, BurntSushi/toml v1.6.0. Pattern 6. |
| LIST-08 | Sorted by need for attention, stable | `domain.Bucket` + `domain.Sort` rewrite; add `createdAt` to query and `PR.CreatedAt`. Pattern 7. |
| FILT-01 | Toggle Open/Merged/Closed in any combination | Visibility bitmask in the model; a derived `rows` slice. Pattern 4. |
| FILT-02 | Auto-found merged/closed only within the configured window | Already implemented by `BuildSearch` + `InWindow`; Phase 2 only wires `closed_days` through (clamped 1..365). Pattern 6. |
| FILT-03 | Type a PR number to narrow the list | Hand-rolled digit buffer (prefix match), prompt replaces the footer line. Pattern 8. |
| NAV-02 | `o` opens the selected PR in the browser | URL allow-list + injected `Opener` + `tea.Cmd`. Pattern 9. |
| NAV-03 | `?` shows key bindings | One `keyMap` feeding footer (`ShortHelp`) and overlay (`FullHelp`); override help's hex colours. Pattern 10. |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

Read from `/Users/scottb/dev/bartram/prv/.claude/CLAUDE.md` and the user's global `~/.claude/CLAUDE.md`. Treated with the authority of locked decisions.

- **Bubble Tea v2 import paths only** (`charm.land/*/v2`). Never the v1 `github.com/charmbracelet/{bubbletea,lipgloss,bubbles}` paths. `View()` returns `tea.View`; keys are `tea.KeyPressMsg`; space is `"space"`; paste is `tea.PasteMsg`.
- **No `time.Sleep` / goroutine pollers.** Use `tea.Tick` (this research confirms `Tick` over `Every`, see Pitfall 3). Cancel via the context stored on the model.
- **Do not enable go-gh `EnableCache`.** Keep `LogIgnoreEnv: true` (already in `NewClient`).
- **No stdout/stderr writes while the TUI runs.** Errors go to a status line. Any log goes to a file only.
- **Token is never persisted or logged.** The Transport wrapper in Pattern 3 must record only three header values, never the `Authorization` header.
- **Browser:** `github.com/cli/go-gh/v2/pkg/browser`, `io.Discard` for stdout/stderr, inside a `tea.Cmd`.
- **Config:** TOML via BurntSushi/toml v1.6.0 at `$XDG_CONFIG_HOME/prv/config.toml`, fallback `~/.config/prv/` on macOS too (hand-rolled XDG helper, not `os.UserConfigDir`; see Pitfall 8).
- **Palette ANSI-16 only; status never colour-alone** (existing `TestStylesUseAnsi16` enforces no 24-bit or 256-colour sequences).
- **Tests:** stdlib `testing`, pure `Update`/`View` tests, golden files via `charmbracelet/x/exp/golden`. `teatest/v2` is untagged; this phase does not need it (Pattern 11).
- **User global rules:** Gradle-over-Maven is irrelevant here. Work happens on the feature branch `gsd/phase-01-status-at-a-glance` today (git status); subagents must stay on the branch and never use detached HEAD. Surgical changes; no speculative features. Prefer the serena plugin for code navigation.
- **GSD workflow:** file changes go through a GSD command (`/gsd-execute-phase`).
- Project skills: none found (`.claude/skills`, `.agents/skills` absent).

## Architectural Responsibility Map

prv is a single-process TUI, so "tiers" here are the repo's packages, which is how the planner will assign tasks.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Config load, path resolution, clamping, exit 2 | `internal/config` (new) + `cmd/prv` | — | Pure parse + one file read at startup, before `tea.NewProgram`. `ui` never reads files. |
| Attention buckets, sort, state/number filter predicate | `internal/domain` | — | Pure functions, table-testable, no I/O. |
| Backoff arithmetic (`nextDelay`) | `internal/domain` | `internal/ui` (applies it) | Pure; the model supplies `now`, interval, failures, rate limit. |
| Fetch, `limit`/`createdAt` in query, error classification, header capture | `internal/github` | — | Only package that knows the wire format and `*api.HTTPError` / `*api.GraphQLError`. |
| Single-flight, gen counter, tick chain, selection, filters state | `internal/ui` `Model.Update` | — | Elm loop owns all mutable state; no goroutine pollers. |
| Header/footer composition, help overlay, prompt | `internal/ui` `View` | — | Pure render from model state; Update/View do no I/O. |
| Open in browser | `internal/ui` (`tea.Cmd`) | `cmd/prv` (constructs the go-gh `Opener`) | `ui` depends on a local `Opener` interface; it never imports go-gh `browser` or `internal/github`. |
| Cancel on quit | `cmd/prv` | `internal/ui` | `context.WithCancel` + `defer cancel()` already in `run`; `tea.WithContext(ctx)` already set. |

## Standard Stack

### Core (already in go.mod, versions as pinned)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/bubbletea/v2` | v2.1.0 | `tea.Tick`, `tea.Batch`, `tea.KeyPressMsg`, `tea.PasteMsg` | Locked stack. [VERIFIED: /Users/scottb/dev/bartram/prv/go.mod:7] `charm.land/bubbletea/v2 v2.1.0` |
| `charm.land/bubbles/v2` | v2.2.1 | `key`, `help`, `spinner` | Locked stack. [VERIFIED: go.mod:6] `charm.land/bubbles/v2 v2.2.1` |
| `charm.land/lipgloss/v2` | v2.0.6 | Table, styles, `lipgloss.Width` | [VERIFIED: go.mod:8] `charm.land/lipgloss/v2 v2.0.6` |
| `github.com/cli/go-gh/v2` | v2.16.2 | GraphQL client, `pkg/browser` | [VERIFIED: go.mod:11] `github.com/cli/go-gh/v2 v2.16.2` |
| `github.com/charmbracelet/x/ansi` | v0.11.9 | `ansi.Truncate` for line cutting | Already used by `cut`/`truncateCell`. [VERIFIED: go.mod:9] |

### New
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/BurntSushi/toml` | v1.6.0 | Parse `config.toml`, `MetaData.Undecoded()` | Named in the project stack. `@latest` on proxy.golang.org is v1.6.0, published 2025-12-18. [VERIFIED: proxy.golang.org/github.com/!burnt!sushi/toml/@latest] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled digit prompt | `bubbles/textinput` | `textinput.Validate` only sets `Err`; it does **not** block the insert (`setValueInternal(runes, err)` always stores the runes). A digits-only prompt would need a filter around it anyway. A 20-line buffer is simpler. [VERIFIED: charm.land/bubbles/v2@v2.2.1/textinput/textinput.go:218-233] |
| `tea.Tick` | `tea.Every` | `Every` truncates to a wall-clock boundary (`n.Truncate(duration).Add(duration).Sub(n)`), which is wrong for a re-armed, variable delay. [VERIFIED: charm.land/bubbletea/v2@v2.1.0/commands.go:102-114] |
| `os.UserConfigDir` | hand-rolled XDG | `os.UserConfigDir()` returned `/Users/scottb/Library/Application Support` on this macOS. D-14 requires `~/.config`. [VERIFIED: spike run this session] |

**Installation:**
```bash
go get github.com/BurntSushi/toml@v1.6.0
```

**Version verification:** `curl https://proxy.golang.org/github.com/!burnt!sushi/toml/@latest` returned `{"Version":"v1.6.0","Time":"2025-12-18T12:15:22Z","Origin":{"VCS":"git","URL":"https://github.com/BurntSushi/toml",...}}`.

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| github.com/BurntSushi/toml | Go module proxy | long-established; v1.6.0 published 2025-12-18 | not available from the proxy | github.com/BurntSushi/toml (proxy `Origin.URL`) | not checkable: `gsd-tools query package-legitimacy check` accepts only `npm\|pypi\|crates` | Approved: it is a locked choice in `.claude/CLAUDE.md`; no checkpoint added |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none. The seam does not support the Go ecosystem, so the verdict is `[ASSUMED]` from the project's own stack decision plus the proxy `Origin` match. The planner may add a `go mod verify` / `go.sum` review step instead of a human checkpoint.

No other external package is added. Test-only code uses stdlib only.

## Architecture Patterns

### System Architecture Diagram

```
                      cmd/prv run()
   flags ──► config.Load(path) ──exit 2 on error──► stderr "prv: <path> line N: ..."
      │            │ Config{Interval,ClosedDays,Show,Warnings}
      │            ▼
      └─► merge --closed-days (flag > config > default, 1..365)
                   │
                   ▼
   token ─► github.NewClient{Token,ClosedDays,Transport=headerCapture(base)}
                   │
                   ▼  ui.New(ctx, client, Options{Interval,ClosedDays,Show,ConfigPath,Warnings,Opener})
 ┌────────────────────────────── ui.Model (Elm loop, one goroutine) ─────────────────────────────┐
 │                                                                                               │
 │  New(): gen=1, inFlight=true            Init(): Batch(spin.Tick, fetchCmd(gen))   (NO tick)   │
 │                                                                                               │
 │  fetchedMsg{gen,snap|err} ──► gen != m.gen? drop                                              │
 │        │ ok: snap=…, lastOK, failures=0, rl=snap.RateLimit ─┐                                 │
 │        │ err: keep snap, failures++, lastErr  ──────────────┤                                 │
 │        ▼                                                    ▼                                 │
 │   rebuild(): filter(show,number) → rows → reselect(selected ref, nearest neighbour)           │
 │        └──► nextDelay(now,interval,failures,rl,errInfo) ──► tea.Tick(delay) → tickMsg{gen}   │
 │                                                                                               │
 │  tickMsg{gen} ──► gen != m.gen or inFlight? drop : inFlight=true, Batch(spin.Tick,fetchCmd)  │
 │  key r        ──► inFlight: coalesce │ paused: flash │ else: gen++, inFlight=true, fetchCmd   │
 │  key o        ──► safeURL(selected.URL) → openCmd → openedMsg{err}                            │
 │  keys 1/2/3,/,esc,?,j,k ──► pure state edits                                                  │
 └───────────────┬───────────────────────────────────────────────────────────────┬───────────────┘
                 │ fetchCmd (goroutine, panic-safe)                              │ View()
                 ▼                                                               ▼
        github.Client.Fetch(ctx) ── POST /graphql (1..5 pages) ──► api.github.com    header │ rows │ footer
        headerCapture RoundTripper records Retry-After, X-RateLimit-Remaining/Reset  (composed with priority drop)
```

### Recommended Project Structure
```
cmd/prv/main.go              # + load config, flag precedence, build Opener, header-capture transport
internal/config/config.go    # NEW: Path(), Load(), Config, clamping, error formatting  (+ config_test.go)
internal/domain/pr.go        # + PR.CreatedAt, RateLimit.Limit
internal/domain/list.go      # Bucket(), Sort() rewritten, Visible() filter predicate
internal/domain/backoff.go   # NEW: NextDelay() pure
internal/github/client.go    # query: + createdAt, + limit; map them
internal/github/errors.go    # + Short(), Backoff(); detect rate limiting; KindRateLimited appended
internal/github/ratelimit.go # NEW: headerCapture RoundTripper
internal/ui/model.go         # state machine, messages, Options
internal/ui/keys.go          # NEW: keyMap (key.Binding), ShortHelp/FullHelp
internal/ui/refresh.go       # NEW: tick/fetch Cmds, safeCmd wrapper
internal/ui/view.go          # header/footer composer, filtered-empty, WR-02
internal/ui/help.go          # NEW: overlay render
internal/ui/open.go          # NEW: safeURL, openCmd, Opener
```

### Pattern 1: Single-flight refresh with one tick chain and a generation counter
**What:** State fields `gen int`, `inFlight bool`, `failures int`. Exactly one tick is pending at any time because a tick is created only (a) from a `fetchedMsg` with the current `gen`, and (b) never while one is pending. A manual `r` (when idle) increments `gen`, which orphans the pending tick: when it fires, `tickMsg.gen != m.gen` and it is dropped.
**When to use:** all refresh triggers.
**Rules (prescriptive):**
- `New()` sets `gen: 1, inFlight: true` (`Init` has a value receiver and cannot persist state). [VERIFIED: internal/ui/model.go:87-89] `func (m Model) Init() tea.Cmd {` / `return tea.Batch(m.spin.Tick, m.fetchCmd())`.
- `Init()` returns `Batch(spin.Tick, fetchCmd(gen))` and **no tick**. The existing `runCmd` helper in `model_test.go` and `main_test.go` executes every Cmd `Init` returns; a `tea.Tick` would block it for the interval.
- `fetchedMsg` gains `gen int`. Drop it if `msg.gen != m.gen`. A fetch result is never "stale by `r`", because `r` is refused while `inFlight`; the check still protects against future callers (add/remove in Phase 4 will bump `gen`).
- `r`: `inFlight` -> no-op (coalesce, optionally flash nothing); `time.Now() < pausedUntil` -> flash `rate limited — resumes HH:MM`, no request (D-08); otherwise `gen++`, `inFlight=true`, return `Batch(spin.Tick, fetchCmd(gen))`.
- Optional 2 s minimum gap between manual refreshes (PITFALLS.md Pitfall 5 suggests it). Single-flight already bounds `r` mashing to roughly one request per request latency (about 1.7 s measured in the spike), so this is cheap insurance, not a requirement.
- The next tick is built with `tea.Tick(delay, func(time.Time) tea.Msg { return tickMsg{gen: gen} })`. Go through an injectable `m.tick` field so tests never run a real timer.
- Cmd closures capture only the values they need (`fetcher`, `ctx`, `gen`), **not `m`**. The existing `fetchCmd` is a method on `Model` value `m`, so its closure copies the whole struct including slices; once `rows`/maps live in the model, a cmd goroutine reading them races with `Update`. [VERIFIED: internal/ui/model.go:74-84] `func (m Model) fetchCmd() tea.Cmd {` ... `snap, err := m.fetcher.Fetch(m.ctx)`.

**Example:**
```go
// Source: derived from .planning/research/ARCHITECTURE.md "Pattern 2" and the code read this session
case tickMsg:
    if msg.gen != m.gen || m.inFlight { return m, nil }
    m.inFlight = true
    return m, tea.Batch(m.spin.Tick, fetch(m.ctx, m.fetcher, m.gen))

case fetchedMsg:
    if errors.Is(msg.err, context.Canceled) { return m, nil } // quitting
    if msg.gen != m.gen { return m, nil }
    m.inFlight = false
    m.applyResult(msg)                           // sets snap/lastOK/err/failures/rl, calls rebuild()
    d := domain.NextDelay(m.now(), m.interval, m.failures, m.rl, m.failInfo)
    m.pausedUntil = d.PausedUntil                // zero when not a rate-limit pause
    m.nextAt = m.now().Add(d.Delay)
    return m, m.tick(d.Delay, m.gen)
```

### Pattern 2: `NextDelay`, a pure backoff function (order: retry-after, then resetAt, then bounded exponential)
**What:** One pure function in `internal/domain/backoff.go`, table-tested. All server-supplied durations are clamped (see Pitfall 5).
**Recommended rules** (thresholds are tunables, [ASSUMED], see Assumptions Log):
1. Last fetch failed and carried `RetryAfter > 0`: `delay = clamp(RetryAfter, 1s, 65m)`; `PausedUntil = now+delay`.
2. Else last fetch failed rate-limited (header `x-ratelimit-remaining: 0`, or a 403/429) and a reset time is known (error's `ResetAt`, else the last known `rl.ResetAt` still in the future): `delay = clamp(until(reset)+1s, 1s, 65m)`; `PausedUntil` set.
3. Else any failure: `delay = min(max(interval, 30s) << (failures-1), 5m)`. If the failure was rate limited with no reset info, floor at 60 s (the documented "wait at least one minute").
4. Else success: `delay = interval`. Then **pace**: if `rl.Limit > 0` and `rl.Remaining <= reserve` (reserve = `10 * max(rl.Cost,1)`) -> pause until `rl.ResetAt + 1s`; else if `rl.Remaining/max(rl.Cost,1)` refreshes cannot cover `until(ResetAt)` at `interval` spacing -> `delay = until(ResetAt) / refreshesLeft` (never below `interval`).
5. The failure counter resets to 0 on success.
**Why pacing and not a flat percentage:** at 20% remaining (1000 points) and cost 1-5 per refresh, the interval is almost always still affordable; pacing only stretches when another process (`gh`, other panes) is actually draining the shared token. The yellow header (D-06) stays a pure display threshold: `Remaining*5 < Limit`.
**Documented basis:** [CITED: docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api] primary limit: 403/429 with `x-ratelimit-remaining: 0`, wait until `x-ratelimit-reset`. Secondary: honour `retry-after`; else if remaining is 0 wait for reset; else "wait for at least one minute"; on repeat "an exponentially increasing amount of time"; "Continuing to make requests while you are rate limited may result in the banning of your integration."

### Pattern 3: Getting the retry/reset time out of go-gh
**Facts (go-gh v2.16.2 source, read this session):**
- A non-2xx response becomes `*api.HTTPError` with `Headers http.Header` populated. [VERIFIED: pkg/api/errors.go:13-19] `type HTTPError struct { Errors []HTTPErrorItem; Headers http.Header; Message string; RequestURL *url.URL; StatusCode int }`. Read `Retry-After`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` from it.
- A 2xx response whose body has `errors` becomes `*api.GraphQLError` with **no headers**. [VERIFIED: pkg/api/graphql_client.go] `if len(gr.Errors) > 0 { return &GraphQLError{Errors: gr.Errors} }`.
- GitHub says the primary GraphQL limit returns HTTP 200 with the error in the body and `x-ratelimit-remaining: 0`. [CITED: docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api] "the response status will still be 200, but you will receive an error message, and the value of the x-ratelimit-remaining header will be 0". Secondary: status 200 or 403, `retry-after` if present.
- Therefore the exhausted-primary-limit case is **only** visible through headers if a transport records them. `github.Options.Transport` already exists and is the base transport under go-gh's sanitizer, cache, log and header round-trippers. [VERIFIED: go-gh pkg/api/http_client.go: `if opts.Transport != nil { transport = opts.Transport }` then wrapped by `newSanitizerRoundTripper` etc.]
- The GraphQL error `type` for rate limiting is **not** documented ("The page does not mention a RATE_LIMITED error type" per the docs fetch). `GraphQLErrorItem.Type` exists. [VERIFIED: pkg/api/errors.go:47-56] Detect rate limiting with the header first, `Type == "RATE_LIMITED"` as a second signal [ASSUMED], never by message text alone.

**Recommended:** `internal/github/ratelimit.go` — a small `http.RoundTripper` wrapper that, after `base.RoundTrip`, stores `Retry-After`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` under a mutex. It must read **only those three headers**, never `Authorization` (AUTH-03). `Client.Fetch` consults it inside `classify` to build the error's `RetryAfter`/`ResetAt`. `x-ratelimit-reset` is UTC epoch seconds. Parse `Retry-After` as integer seconds; ignore anything else.

**Error surface for `ui` without importing `internal/github`:** add methods on `*github.Error` and declare local interfaces in `ui` (the existing `hinter` pattern, view.go:50 `type hinter interface{ Hint() string }`):
```go
// internal/github/errors.go (new methods)
func (e *Error) Short() string                      // "network" | "auth" | "rate limit" | "server" | "error"
func (e *Error) Backoff() (retryAfter time.Duration, resetAt time.Time, rateLimited bool)
// internal/ui (local interfaces)
type shorter  interface{ Short() string }
type backoffer interface{ Backoff() (time.Duration, time.Time, bool) }
```
Do **not** reuse `Hint()` for the D-07 short reason: those strings are 50-170 characters (errors.go:43-57), far longer than the `(network)` in the D-07 example. Append `KindRateLimited` at the end of the enum (do not renumber; `KindAuth` is the zero value, IN-04).

Also add `limit` to the query: `rateLimit { limit cost remaining resetAt }`. [VERIFIED: live `gh api graphql` this session returned `{"rateLimit":{"limit":5000,"cost":1,"remaining":5000,"resetAt":"2026-10-12T04:19:36Z","used":0,"nodeCount":0}}`] The current query is `rateLimit { cost remaining resetAt }` (client.go:62) and `domain.RateLimit` has `Cost, Remaining, ResetAt` only (pr.go:101-105). Update `search_page1.json`, `search_page2.json`, `status.json` fixtures and `TestRateLimitDecoded`.

### Pattern 4: Selection by ref, visibility filters, nearest-neighbour
**What:** Keep the sorted full list in `m.snap.PRs`. Derive `m.rows []domain.PR` (the visible rows) in one `rebuild()` that runs after every snapshot, every toggle and every number-filter edit. Track `m.selected domain.Ref`; `m.cursor` is derived from it.
**Algorithm** (deterministic, testable):
```go
func (m *Model) rebuild() {
    oldRows, oldCur := m.rows, m.cursor
    m.rows = domain.Visible(m.snap.PRs, m.show, m.numFilter)
    switch {
    case len(m.rows) == 0:
        m.cursor, m.offset = 0, 0                        // keep m.selected: it may come back
    case idx(m.rows, m.selected) >= 0:                    // selected PR still visible: follow it
        m.cursor = idx(m.rows, m.selected)
    default:                                              // it left the list: nearest neighbour
        m.cursor = nearest(oldRows, oldCur, m.rows)       // scan old order outward, prefer below then above
        m.selected = m.rows[m.cursor].Ref
    }
    m.clampOffset()
}
```
`nearest`: walk `d = 1, 2, 3…`, check `oldRows[oldCur+d]` then `oldRows[oldCur-d]`; return the index in the new rows of the first ref that still exists; fall back to `min(oldCur, len(rows)-1)`. On first load (no `selected`), select row 0.
`j`/`k` set `m.selected = m.rows[m.cursor].Ref`.
**Filter semantics to document:** `StateUnknown` PRs are always visible regardless of toggles (never hide a PR prv cannot classify; matches `InWindow`, which always keeps unknown state, list.go:13-22). The number filter is a prefix match on `strconv.Itoa(pr.Ref.Number)`.
**Header counts:** `12 PRs` should be the visible count. The filtered-empty message needs `hidden = len(m.snap.PRs) - len(m.rows)`.
**The Phase 1 reset to remove:** `m.cursor, m.offset = 0, 0` on every fetch (model.go:129). [VERIFIED: internal/ui/model.go:129] `m.cursor, m.offset = 0, 0`.
**State:** `stateEmpty` stays "the snapshot has zero PRs". "Filters hid everything" is a render branch inside the list state, not a new view state.

### Pattern 5: Header and footer composed by priority (fixes WR-03, WR-02)
**What:** Replace the string-append footer and right-edge `cut` with a composer that drops *lowest-priority* segments until the line fits, then pads, rather than chopping the tail. [VERIFIED: internal/ui/view.go:141-147] `footer := "j/k move · q quit"` then `footer += " · list truncated"` and `footer += " · some data could not be loaded"`, and `view.go:152-154` `lines[i] = m.cut(l)` truncates from the right — the cause of WR-03.
**Header segments (high to low priority):** problem marker (`✗ refresh failed 14:33 (network) · data from 14:32:05` or `paused until 14:50 (rate limit)`) > yellow rate warning (when `Remaining*5 < Limit`) > `updated 14:32:05 ⠋` > `[open merged ·]` (+ `#12`) > `12 PRs` > `@login` > `prv`.
**Footer:** left = integrity notes first (`list truncated`, `some data could not be loaded`, config warning), then key hints from `help.ShortHelpView`; right = faint `api 4812`. Priority: notes > `api N` > hints. `help.Model.ShortHelpView` already drops trailing items to fit `SetWidth(w)` and adds an ellipsis. [VERIFIED: charm.land/bubbles/v2@v2.2.1/help/help.go:128 "If the line is longer than the maximum width it will be gracefully truncated"] Give it the width left after notes and `api N`.
**Prompt (D-11):** the digit prompt **replaces the footer line**. Visible rows stay `h - chromeLines`, `chromeLines = 3`. [VERIFIED: internal/ui/layout.go:25-27] `// chromeLines are the lines around the rows: the "prv · @login" line, the` ... `chromeLines = 3`. Adding a line would shift every golden file and the `visibleRows` maths.
**WR-02:** the empty-state and filtered-empty screens must append the truncated/warnings notice (view.go:28-31 currently prints the fixed "No open PRs … none closed in the last %d days." and drops `Warnings`/`Truncated`). Re-word to "No PRs could be listed" when `Warnings` or `Truncated` is set.
**Golden impact:** the header and footer change, so every `list_*` and `empty_*` golden regenerates (`go test ./internal/ui -update`, golden package convention). Review the diffs by eye at 80x24 and 120x40.
**Time format:** `t.Local().Format("15:04:05")` for `updated`, `"15:04"` for resume/failed times. Local time is what the user reads.
**Spinner:** show it next to `updated` only while `inFlight`. In `spinner.Update` a `TickMsg` is rejected when `msg.tag != m.tag`, and each accepted tick increments `m.tag`; a second `m.spin.Tick` started while a chain is alive therefore self-heals (the duplicate is rejected). [VERIFIED: charm.land/bubbles/v2@v2.2.1/spinner/spinner.go:131-155] Stop the chain by not returning the next tick when `!m.inFlight`; start it by adding `m.spin.Tick` to the batch that starts a fetch. The Phase 1 handler only advances the spinner `if m.state == stateLoading` (model.go:118-123); change to `m.inFlight || m.state == stateLoading`.

### Pattern 6: Config package
**Spike results (BurntSushi/toml v1.6.0, run this session):**
| Input | Result |
|-------|--------|
| `refresh_interval = "60s"` into a **native `time.Duration`** | decodes to 60s |
| `refresh_interval = 60` into a native `time.Duration` | **no error**, becomes 60 ns (raised silently to 15 s) |
| `refresh_interval = "soon"` into a native `time.Duration` | `toml.ParseError`, line 2: `invalid duration: "soon"` |
| `refresh_interval = 60` into a type with `UnmarshalText` that calls `time.ParseDuration` | error with line 3: `missing unit in duration "60"` |
| `closed_days = "fourteen"` | `*errors.errorString`: `toml: line 1 (last key "closed_days"): incompatible types: TOML value has type string; destination has type integer` — **not** a `toml.ParseError` |
| `closed_days = 14.5` | same `incompatible types` class of error, line 2 |
| syntax error `closed_days = = 14` | `toml.ParseError`, `Position.Line` 2, `Message` `expected value but found '=' instead` |
| `show = ["open", "opne"]` with an `UnmarshalText` element type | `toml.ParseError`, line 2, `unknown state "opne"` |
| `refresh_intervall = "60s"` (typo) | no error; `md.Undecoded()` returns `[refresh_intervall]` |
| empty file | no error, all zero values |

**Therefore:**
- Decode `refresh_interval` with a small `type duration time.Duration` that implements `encoding.TextUnmarshaler` via `time.ParseDuration`. Do **not** use a native `time.Duration` field (the integer trap). The TextUnmarshaler error carries a line number for free.
- Decode `show` elements with a `type showState string` `UnmarshalText` that accepts only `open|merged|closed`.
- Format errors for D-15 with: `errors.As(err, &toml.ParseError)` -> `Position.Line` + `Message`; otherwise match `^toml: line (\d+) \(last key "[^"]*"\): (.*)$` on `err.Error()` and rebuild `prv: <path> line N: <msg>`; if neither matches, print `prv: <path>: <err>`. Exit code 2.
- Missing file: `errors.Is(err, fs.ErrNotExist)` -> defaults, no warning. Any other read error (permission, is-a-directory) -> exit 2 with the path.
- Clamp after decode: interval `< 15s` (including zero/negative) -> 15 s; `closed_days` to 1..365; empty `show` array -> defaults? An explicit `show = []` means "show nothing" which is a trap; treat empty as all three and add a warning. [ASSUMED]
- Path helper: `$XDG_CONFIG_HOME/prv/config.toml` if that variable is set **and absolute** (XDG says relative values are invalid), else `$HOME/.config/prv/config.toml`. Do not use `os.UserConfigDir` (macOS gives `~/Library/Application Support`). Make the path function a package var in `cmd/prv` so tests do not read the developer's real config. **Every existing `run()` test must set it** (see Pitfall 9).
- `Undecoded()` warnings are shown once. Pass key names through `domain.Clean` before display: a quoted TOML key can contain control characters and would otherwise reach the terminal.
- `--version` must be handled before config load so a broken config never blocks `prv --version` (current order in `run`: parse, `*showVersion`, then closed-days check, then token, main.go:57-73).
- Flag precedence: the flag `--closed-days` defaults to `domain.DefaultClosedDays`, so "was it set?" needs `fs.Visit` (visits only flags that were set). Out-of-range flag value: keep the existing behaviour (exit 2, `TestHiddenClosedDaysFlag` expects `--closed-days 0` to give 2) and add the upper bound 365 (fixes WR-01). Config values are clamped, flag values are rejected; this is an interpretation of D-14/D-16, see Open Questions.

### Pattern 7: Attention buckets in `domain`
**Rules** (the first match wins; state is checked **before** draft so that a closed or merged draft is not shown as a live draft):
```go
// Source: D-01/D-02/D-04 + the existing enums read from internal/domain/pr.go
func Bucket(pr PR) int {
    switch pr.State {
    case StateMerged:  return 7
    case StateClosed:  return 8
    case StateUnknown: return 9              // "closed (then unknown state)": same slot, after closed
    }
    if pr.IsDraft                              { return 6 }          // D-04: even with failing CI
    switch {
    case pr.CI == CIFailing:                   return 1
    case pr.Review == ReviewChangesRequested:  return 2
    case pr.Review == ReviewApproved && pr.CI == CIPassing: return 3
    case pr.CI == CIRunning:                   return 4
    }
    return 5                                   // incl. approved + no checks / unknown CI
}
```
**Verbatim enums** [VERIFIED: internal/domain/pr.go:16-21, 54-59, 65-71]: `StateOpen State = iota`, `StateMerged`, `StateClosed`, `StateUnknown`; `ReviewNone Review = iota`, `ReviewApproved`, `ReviewChangesRequested`, `ReviewRequired`; `CINone CI = iota`, `CIPassing`, `CIFailing`, `CIRunning`, `CIUnknown`.
**Recommended discretion calls:** approved + `CINone` or `CIUnknown` is bucket **5**, not 3, because "ready to merge" must not rest on "no checks", which is never shown as passing (REQUIREMENTS LIST-05). Approved + running is bucket 4. Changes-requested + running is bucket 2.
**Comparator:** bucket ascending, then `b.CreatedAt.Compare(a.CreatedAt)`, then `cmp.Compare(a.Ref.Repo, b.Ref.Repo)`, then `cmp.Compare(a.Ref.Number, b.Ref.Number)` (integer, not string). `Ref` is unique after dedupe by node ID, so the order is total and independent of input order. `slices.SortStableFunc` can stay.
**Data changes:** `PR.CreatedAt time.Time`; add `createdAt` to the query's field list (client.go:69 currently `id number title url isDraft state closedAt mergedAt updatedAt`); map it in `mapResponse`. [VERIFIED: live query this session: `{"number":1,"createdAt":"2026-10-12T01:36:08Z","updatedAt":"2026-10-12T01:36:08Z","state":"OPEN","isDraft":false}`]. `UpdatedAt` stays decoded but is no longer a sort key; the existing `TestSort` cases keyed on `UpdatedAt` must be rewritten (domain_test.go:178-227).
**Where sort runs:** keep `domain.Sort(snap.PRs)` in `Client.Fetch` (client.go:200) as today. Sorting is cheap and idempotent; the UI tests that hand-build snapshots should build them in sorted order or call `domain.Sort` in their helper.

### Pattern 8: Digit-only prompt (hand-rolled)
A `numFilter string` and `prompting bool`. While `prompting`: digit keys (`msg.String()` is `"0"`..`"9"`) append (cap at 7 digits); `backspace` trims; `tea.PasteMsg` appends only its digit runes; `enter` -> `prompting=false`, keep filter; `esc` -> clear and `prompting=false`; `ctrl+c` still quits; everything else is ignored (so `q` does not quit while typing). In list mode with a kept filter, `esc` clears it. **Digits `1/2/3` are toggles in list mode and typed digits in prompt mode**; the mode switch is the only disambiguation. Each change calls `rebuild()`, so the list narrows live. `key.Matches` and `KeyPressMsg.String()` return the text for printable keys, so `"?"` and `"1"` match as written. [VERIFIED: ultraviolet key.go:385-396] `// For example, you'll always get "?" and instead of "shift+/"`.

### Pattern 9: Open in browser
- `ui` defines `type Opener interface{ Open(url string) error }` and `Options.Opener`; `cmd/prv` supplies `browser.New("", io.Discard, io.Discard).Browse`. [VERIFIED: go-gh pkg/browser/browser.go] `func New(launcher string, stdout, stderr io.Writer) *Browser` and `func (b *Browser) Browse(url string) error`. `New` reads `GH_BROWSER`, gh config and `BROWSER` once at construction; build it once in `run`.
- **Validate with `net/url`, not `strings.HasPrefix`.** Accept only `Scheme == "https"`, `Hostname()` equal (case-insensitively) to `github.com`, `Port() == ""`, `User == nil`. Then pass `u.String()`. [VERIFIED: spike run this session] results: `https://github.com.evil.com/x` -> host `github.com.evil.com`; `https://github.com@evil.com/x` -> host `evil.com`, user `github.com`; `http://github.com/x` -> scheme `http`; `https://GitHub.com/x` -> host `GitHub.com` (mixed case); `https://github.com:444/x` -> port `444`; `" https://github.com/x"` -> parse error; `javascript:alert(1)` -> scheme `javascript`. A `strings.HasPrefix(u, "https://github.com")` check would wrongly accept the first two.
- go-gh `Browse` itself allows `http`, `https`, `vscode`, `vscode-insiders`, so the allow-list must be ours. [VERIFIED: pkg/browser/browser.go `isSupportedScheme`]
- Run inside a `tea.Cmd` wrapped by the shared `safeCmd` (recovers panics, returns `openedMsg{err}`). A failure becomes a transient footer message (`could not open browser`), without echoing the launcher's output.
- Known gap, document it, do not solve it: if `BROWSER`/`GH_BROWSER` names a terminal browser (w3m, lynx), `cmd.Run()` blocks and that program writes to the real terminal while Bubble Tea owns it (the launcher's stdout/stderr are `io.Discard`, but it can still open `/dev/tty`). `gh` has the same limitation.

### Pattern 10: One keymap, footer and overlay
`internal/ui/keys.go`: `type keyMap struct{ Up, Down, Refresh, Open, Filter, Open1/Merged/Closed toggles, Help, Quit key.Binding }` with `ShortHelp() []key.Binding` and `FullHelp() [][]key.Binding`. `key.Matches(msg, binding)` replaces the `msg.String()` switch. Keep `j/k/down/up/q/ctrl+c` working exactly as today (existing tests send `"q"`, `"ctrl+c"`, `"j"`).
**Override the help styles.** `help.DefaultStyles(isDark)` uses hex colours (`#909090`, `#626262`, …), which breaks the ANSI-16-only rule and `TestStylesUseAnsi16`. [VERIFIED: charm.land/bubbles/v2@v2.2.1/help/help.go:48-65] `keyStyle := lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("#909090"), lipgloss.Color("#626262")))`. Set `m.help.Styles` to plain/faint styles built from the existing `faintStyle`.
**Overlay:** full screen, any key (except `ctrl+c`, which still quits) returns to the list; refreshes keep running behind it (the tick/fetch handlers do not look at the overlay flag). It must fit 80x24: key table, badge legend, one settings line, one `api remaining / limit, resets HH:MM` line, `press any key to return`. Use `m.wrap` for the settings lines and a width/height fit test like `TestViewFitsScreen`.

### Pattern 11: Verification harness for "a simulated hour with `-race`"
**Design:** the model has two injectable fields: `now func() time.Time` and `tick func(d time.Duration, gen int) tea.Cmd`. The test installs a fake `tick` that records `(due, gen)` in a queue and returns a non-nil placeholder Cmd, and a `now` backed by a virtual clock. The driver loop pops the earliest due tick, advances the clock to it, calls `Update(tickMsg{gen})`, runs the returned fetch Cmd **synchronously**, and feeds the resulting `fetchedMsg` back. No `time.Sleep`, so an hour runs in milliseconds.
**Assertions over the virtual hour** (default 60 s): fetch calls within `[59, 61]`; calls never overlap (fake records max concurrency via `atomic`); a scripted external consumer that drains the fake bucket makes prv pause and **zero** calls land while `now < pausedUntil`; calls resume at `resetAt`; with a 5xx burst the gaps grow `30s, 60s, 120s, … ≤ 5m` and reset after a success; `gen` mismatch ticks never fetch.
**Overlap test (real concurrency, `-race`):** run a real `tea.Program` (`tea.WithInput(nil)`, `tea.WithOutput(io.Discard)`), a fake fetcher that blocks on a channel, a 5 ms interval, and `p.Send` of `r` keypresses from several goroutines; assert exactly one in-flight call at a time and clean shutdown through `p.Quit()` / context cancel. This is where the "closure captures `m`" data race from Pattern 1 would show.
**Existing helpers:** `runCmd` (model_test.go:40-56) runs Cmds recursively and never runs commands that `Update` returns. [VERIFIED: internal/ui/model_test.go:38-39] `// runCmd runs cmd, recurses into tea.BatchMsg, and passes every non-nil` / `// message to Update once. It never runs commands that Update returns.` That property is why `Init` must not return a tick.
**Test migration cost:** `fetchedMsg{snap: …}` / `fetchedMsg{err: …}` literals appear throughout `model_test.go` and `view_test.go` and carry no `gen`. Add a helper that stamps the model's current `gen`, and update them in one sweep.

### Anti-Patterns to Avoid
- **A tick returned from `Init()`**: blocks `runCmd` and double-arms (Pattern 1).
- **Setting state in `Init()`** (value receiver): lost. Initialise in `New()`.
- **Re-arming the tick from `tickMsg`**: it stacks when a fetch outlives the interval. Re-arm only from `fetchedMsg`.
- **`tea.Every`** for a variable delay (wall-clock aligned).
- **Cmd closures over the whole `Model`** once the model holds slices or maps.
- **Sorting by `updatedAt`, or by anything polling changes**: rows swap places on comments and pushes.
- **Cursor by index**: it stays on index 3 while the PR moves.
- **A per-second redraw tick** for "updated 12s ago": use the absolute clock time (D-05).
- **Trusting a server delay unclamped** (Pitfall 5).
- **`fmt.Println` / `log` to stdout/stderr while the TUI runs**.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TOML parsing, unknown-key detection | a line-based parser | `BurntSushi/toml` `Decode` + `MetaData.Undecoded()` | Syntax errors carry a line number; typos surface for free |
| Duration parsing | regex on `"60s"` | `time.ParseDuration` inside an `UnmarshalText` type | Handles `1m30s`, rejects `60` (see Pattern 6 trap) |
| Timers | `time.Sleep` loops, goroutine pollers | `tea.Tick` re-armed from `Update` | Testable, cancellable, no overlap |
| Key bindings and the help screen | a hand-written key table | `bubbles/key` + `bubbles/help` (restyled to ANSI-16) | One source for footer and overlay |
| Spinner | frame counter + timer | `bubbles/spinner` | Tag logic already de-duplicates tick chains |
| Browser launching | `exec.Command("open"/"xdg-open")` | go-gh `pkg/browser` | Honours `GH_BROWSER`, gh config, `BROWSER` |
| URL validation | `strings.HasPrefix(u, "https://github.com")` | `net/url` parse + exact host/scheme/port/userinfo checks | Prefix check accepts `github.com.evil.com` and `github.com@evil.com` |
| Width-safe line cutting | `s[:w]` | `ansi.Truncate` / `lipgloss.Width` | Already used; CJK and emoji widths |
| Sorting | hand-written sort | `slices.SortStableFunc` + `cmp.Compare` | Existing pattern in `domain.Sort` |

**Hand-rolling is correct for exactly two things:** the digits-only prompt buffer (Pattern 8) and the `NextDelay` arithmetic (Pattern 2), because no library fits either and both are small, pure and table-testable.

**Key insight:** every "time" problem in this phase becomes easy once time is an input (`now`, `tick`). Do not call `time.Now()` or `tea.Tick` directly from `Update`, `View` or `NextDelay`.

## Common Pitfalls

### Pitfall 1: `Init` cannot carry state and must not return a timer
**What goes wrong:** `inFlight` stays false after `Init`, so the first `r` starts a second concurrent fetch; or `Init` returns a tick and the test helper `runCmd` hangs for the interval.
**Why it happens:** `Init()` has a value receiver and returns only a Cmd; `runCmd` executes all Cmds it is given.
**How to avoid:** set `gen: 1, inFlight: true` in `New()`; `Init` returns spinner tick plus the first fetch only.
**Warning signs:** a `main_test` end-to-end test that takes 60 s; two fetches on startup.

### Pitfall 2: Duplicate tick chains
**What goes wrong:** every manual refresh or error adds another chain, so refreshes accelerate over a day.
**Why it happens:** re-arming from `tickMsg` or from both `tickMsg` and `fetchedMsg`.
**How to avoid:** arm only from `fetchedMsg` with `msg.gen == m.gen`; bump `gen` on manual refresh so the old tick dies on arrival. Test: after N manual `r`, the virtual-hour fetch count stays within bounds.

### Pitfall 3: `tea.Tick` timer starts at creation
**What goes wrong:** a Cmd built in `Update` but executed later still fires at the creation-time deadline; building the Cmd early (for example in `Init`) shortens the first interval.
**Why it happens:** `Tick` calls `time.NewTimer(d)` before returning the Cmd. [VERIFIED: charm.land/bubbletea/v2@v2.1.0/commands.go:154-163]
**How to avoid:** build the tick in the same `Update` call that processes the result.

### Pitfall 4: The primary rate limit arrives as HTTP 200 with no headers in the error
**What goes wrong:** `classify` sees `*api.GraphQLError`, returns `KindGraphQL` with the hint "Press r to retry", and prv hammers a limit that is already exhausted.
**Why it happens:** go-gh discards the response when it unmarshals a 2xx body that has `errors` (Pattern 3).
**How to avoid:** header-capturing Transport; `rateLimited = (remaining header == "0") || Type == "RATE_LIMITED" [ASSUMED]`; model falls back to the last known `rl.ResetAt` if no header was captured.
**Warning signs:** the error panel says "GitHub returned an error for the PR search" while `api` shows 0.

### Pitfall 5: A server-supplied delay is unbounded input
**What goes wrong:** a bad `Retry-After` / `x-ratelimit-reset` (or a clock skew) yields a pause of days, and because D-08 refuses `r` during a pause, the user is locked out.
**How to avoid:** clamp every server delay to `[1s, 65m]` (the primary window is one hour). Parse failures mean "header absent".

### Pitfall 6: `time.Duration` in TOML silently accepts integers
See Pattern 6. `refresh_interval = 60` becomes 60 ns. Use the `UnmarshalText` type.

### Pitfall 7: Selection and scroll offset drift
**What goes wrong:** after a toggle or refresh the cursor points past the end, or the viewport shows blank rows.
**How to avoid:** every mutation path calls `rebuild()`, which ends with `clampOffset()`. Note `clampOffset` computes `max(0, min(m.offset, len(m.snap.PRs)-v))` (layout.go:85) and `visibleRows` uses `len(m.snap.PRs)` (layout.go:71): both must switch to `len(m.rows)`.

### Pitfall 8: `os.UserConfigDir` is not `~/.config` on macOS
Hand-roll the XDG helper (Pattern 6). Verified on this machine.

### Pitfall 9: Tests that read the developer's real config
`run()` will load `~/.config/prv/config.toml`. Without isolation `TestNoToken`, `TestEndToEndFixture` and friends depend on whoever runs them. Inject the path function and set it in every `run` test; add a test that a broken config exits 2 before the token lookup.

### Pitfall 10: A header that does not fit
At 80 columns `prv · @sbartram · 12 PRs · ✗ refresh failed 14:33 (network) · data from 14:32:05` is already about 80 cells, before `[open merged ·]`. Without the priority composer the failure marker is the first thing cut off the right edge. Golden-test the header at widths 40, 60, 80, 120 with a failure, a pause and a rate warning active.

### Pitfall 11: Closed window slides but the sort key does not
`BuildSearch(now, closedDays)` runs with a fresh `c.now()` on every fetch, so a PR that ages out of the window disappears from the list on a later refresh. That is correct (FILT-02) and is handled by `reselect`'s nearest-neighbour fallback. No special code; add a test.

### Pitfall 12: Laptop sleep
Go timers on macOS use the monotonic clock, which pauses during sleep, so after wake the next refresh can come up to one interval late [ASSUMED]. The `updated HH:MM:SS` label stays honest, which is the mitigation. Do not add a wall-clock watchdog in this phase.

## Code Examples

### Config load (sketch)
```go
// Source: spike run this session against BurntSushi/toml v1.6.0
type duration time.Duration
func (d *duration) UnmarshalText(b []byte) error {
    v, err := time.ParseDuration(string(b))
    if err != nil { return fmt.Errorf("refresh_interval: %w", err) }
    *d = duration(v); return nil
}

var lineRE = regexp.MustCompile(`^toml: line (\d+) \(last key "[^"]*"\): (.*)$`)

func describe(path string, err error) string {
    var pe toml.ParseError
    if errors.As(err, &pe) { return fmt.Sprintf("prv: %s line %d: %s", path, pe.Position.Line, pe.Message) }
    if m := lineRE.FindStringSubmatch(err.Error()); m != nil { return fmt.Sprintf("prv: %s line %s: %s", path, m[1], m[2]) }
    return fmt.Sprintf("prv: %s: %v", path, err)
}
```

### Header-capturing transport (sketch)
```go
// Records only three rate-limit headers; never Authorization.
type captureTransport struct {
    base http.RoundTripper
    mu   sync.Mutex
    last rateHeaders
}
func (t *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
    resp, err := t.base.RoundTrip(r)
    if resp != nil {
        t.mu.Lock()
        t.last = rateHeaders{
            RetryAfter: resp.Header.Get("Retry-After"),
            Remaining:  resp.Header.Get("X-RateLimit-Remaining"),
            Reset:      resp.Header.Get("X-RateLimit-Reset"),
        }
        t.mu.Unlock()
    }
    return resp, err
}
```
`NewClient` must default the base to `http.DefaultTransport` when `opts.Transport == nil`.

### Panic-safe Cmd wrapper (generalises `fetchCmd`)
```go
func safeCmd(f func() tea.Msg, onPanic tea.Msg) tea.Cmd {
    return func() (msg tea.Msg) {
        defer func() { if recover() != nil { msg = onPanic } }()
        return f()
    }
}
```
The panic value is never displayed (existing test asserts `secret-panic-value` is not on screen).

### Safe URL (sketch)
```go
func safeURL(raw string) (string, bool) {
    u, err := url.Parse(raw)
    if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" ||
        !strings.EqualFold(u.Hostname(), "github.com") { return "", false }
    return u.String(), true
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Phase 1 sort: state rank, then `updatedAt` desc | D-01/D-02 buckets, `createdAt` desc | this phase | Rows no longer swap when someone comments |
| `msg.String()` switch in `Update` | `key.Matches` over one `keyMap` | this phase | Footer and overlay stay in sync (D-17) |
| Cursor reset to 0 on each fetch | cursor follows ref | this phase | RFSH-06 |
| Footer string append, right-edge `cut` | priority composer | this phase | WR-03 fix |

**Deprecated/outdated:** `tea.WithAltScreen()` and `View() string` (v1) — already avoided. `help.DefaultStyles` hex colours — override.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | GitHub's GraphQL primary-limit error carries `errors[].type == "RATE_LIMITED"` | Pattern 3, Pitfall 4 | Low: detection also uses the `x-ratelimit-remaining: 0` header, which is documented. Check `errors[].type` on a real exhausted response if ever observed. |
| A2 | Thresholds: reserve = `10 * max(cost,1)`; base for failures `max(interval,30s)`; failure cap 5 min; server-delay clamp `[1s,65m]`; no-info rate-limit floor 60 s | Pattern 2 | Low: tunables. Wrong values cost latency, not correctness. Planner may confirm with the user. |
| A3 | Approved + no checks / unknown CI is bucket 5, not 3 | Pattern 7 | Low: a sort-order preference; CONTEXT leaves it to discretion |
| A4 | `StateUnknown` PRs are always visible regardless of 1/2/3 toggles | Pattern 4 | Low: a PR in an unrecognised state could otherwise vanish silently |
| A5 | A closed or merged *draft* sorts by state (bucket 7/8), not in the draft bucket 6 | Pattern 7 | Low: D-04 says "a draft always sorts into the draft bucket"; the intent is open drafts with failing CI. User may want the literal reading. |
| A6 | `--closed-days` out of range is rejected with exit 2 (existing behaviour), the config key is clamped | Pattern 6 | Low: D-14 says clamp, D-16 says shared bound; both readings satisfy WR-01 |
| A7 | `Retry-After` from GitHub is integer seconds | Pattern 3 | Low: an HTTP-date value would be treated as absent and the reset header or exponential rule applies |
| A8 | Empty `show = []` is treated as "all three" with a warning | Pattern 6 | Low |
| A9 | Go monotonic timers pause during macOS sleep, delaying the next refresh | Pitfall 12 | Low: only affects freshness after wake; the timestamp stays honest |
| A10 | Transient messages clear after about 4 s (config warning about 8 s) | Pattern 5 | Low: discretion item |
| A11 | `github.com/BurntSushi/toml` is legitimate; the seam cannot verify Go modules | Package Legitimacy Audit | Low: locked in CLAUDE.md, proxy `Origin` matches |
| A12 | No logging in this phase (D-16 forbids new CLI flags, so `--debug` cannot be added) | Open Questions | Low |

## Open Questions

1. **Logging in Phase 2?**
   - What we know: CONTEXT allows logging "if added", suggests `$XDG_STATE_HOME/prv/prv.log` gated by `--debug`; D-16 says "No other CLI flags are added".
   - What's unclear: whether a `--debug` flag (or an env var) is wanted.
   - Recommendation: add no logging in Phase 2. The status line already shows `api N`, last-ok, and the failure reason. If wanted, gate on `PRV_LOG=<path>` (an env var is not a CLI flag) and write with `slog` to that file only.

2. **Flag out of range: reject or clamp?** (A6)
   - Recommendation: keep exit 2 for the flag (the existing test pins `--closed-days 0` to 2); clamp only config values.

3. **Literal D-04 for closed/merged drafts?** (A5)
   - Recommendation: state first, then draft. Confirm during planning if the user cares.

4. **Config warnings display.** "One-time warning in the TUI": recommended as an 8 s transient message plus a permanent line in the help overlay's settings block.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | build/test | yes | go1.27.2 darwin/arm64 (`go.mod` says `go 1.26.0`) | — |
| `gh` CLI | live checks, `auth.TokenForHost` | yes | 2.102.0 | — |
| Module proxy / network | `go get BurntSushi/toml` | yes (spike fetched v1.6.0) | — | vendor offline copy |
| GitHub API (live) | optional live UAT | yes (`gh api graphql` ran) | — | fixture tests |
| A desktop browser | manual check of `o` | not verifiable headless | — | unit-test `Opener` with a fake; manual UAT for the real launch |
| `go test -race` | simulated-hour test | yes: baseline `go build ./... && go vet ./... && go test -race ./...` passed all four packages | — | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** real-browser launch is covered by a fake `Opener` plus one manual UAT step.

## Security Domain

`security_enforcement` is enabled in `.planning/config.json` (ASVS level 1). `workflow.nyquist_validation` is `false`, so no Validation Architecture section is included; test strategy is under Pattern 11.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no change | Token handling unchanged (go-gh `TokenForHost`, passed via `Options.Token`); the new transport wrapper must not read `Authorization` |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | yes | Config file (TOML decode + clamp + `UnmarshalText` validators), digit-only prompt, `net/url` allow-list for `o`, bounded server-supplied delays |
| V6 Cryptography | no | None added |
| V7 Error handling / logging | yes | Panic values never shown; no token in errors; no logging in this phase |
| V12 Files/resources | yes | Config path from `XDG_CONFIG_HOME` only if absolute; read-only access; config never written |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Terminal escape injection via strings from the config (unknown key names, path) or API (titles, messages) | Tampering | `domain.Clean` at the seam for remote strings (exists); also clean config-derived strings before display |
| Opening an attacker-chosen URL (`o`) from a manipulated API `url` field | Spoofing / Elevation | Exact scheme+host+port+userinfo check with `net/url`; open only the rebuilt `u.String()` |
| Header leakage: logging or storing request headers | Information disclosure | Capture exactly `Retry-After`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` |
| Denial of service on self through a hostile or buggy `Retry-After` / reset | Denial of service | Clamp to `[1s, 65m]`; `r` refusal only while a bounded pause is active |
| Request storm (held `r`, 15 s floor) burning the shared token | Denial of service (self, others) | Single-flight, optional 2 s manual gap, 15 s config floor, pacing in `NextDelay`; worst case is 5 pages x 240 refreshes/hour = 1200 points, inside 5,000 |
| Malicious or oversized config file | Denial of service | Read with a size cap (for example 1 MiB) before decoding; refuse larger |
| Browser launcher hijack via `BROWSER` env | Elevation | Out of scope: it is the user's own environment, same as `gh`; documented |
| Panic in a Cmd leaving the terminal in raw mode | Denial of service | Shared `safeCmd` for fetch and open |

## Sources

### Primary (HIGH confidence)
- Repo code read this session: `internal/ui/{model,view,layout,styles}.go`, `internal/domain/{pr,list,status}.go`, `internal/github/{client,errors}.go`, `cmd/prv/main.go`, `cmd/prv/main_test.go`, `internal/ui/{model,view}_test.go`, `internal/github/client_test.go`, `go.mod`.
- go-gh v2.16.2 source in the module cache: `pkg/api/{errors,graphql_client,http_client}.go`, `pkg/browser/browser.go`.
- Bubbles v2.2.1 source: `spinner/spinner.go`, `help/help.go`, `key/key.go`, `textinput/textinput.go`. Bubble Tea v2.1.0 `commands.go`. Ultraviolet `key.go`.
- Spikes run this session: BurntSushi/toml v1.6.0 decode behaviour (scratchpad), `os.UserConfigDir` on macOS, `net/url` parsing of hostile URLs.
- Live `gh api graphql` (gh 2.102.0): `rateLimit.limit`, `createdAt`.
- proxy.golang.org `github.com/!burnt!sushi/toml/@latest`.
- https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api and https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api (rate-limit handling, 5,000 points/hour, 2,000 points/minute, 100 concurrent, HTTP 200 for primary GraphQL errors).
- `.planning/research/{PITFALLS,ARCHITECTURE}.md`, `.planning/phases/01-status-at-a-glance/{01-REVIEW,01-REVIEW-DISPOSITION,01-SPIKE}.md`.

### Secondary (MEDIUM confidence)
- WebSearch on the GraphQL rate-limit response shape (confirms the docs; did not confirm the `RATE_LIMITED` type string).

### Tertiary (LOW confidence)
- Backoff numeric thresholds and the macOS-sleep timer behaviour (Assumptions A2, A9).

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — one new dependency, version confirmed on the proxy, spike run.
- Architecture: HIGH — state machine derived from code read this session and the library source at the pinned tags.
- Pitfalls: HIGH for the code-level ones (Init receiver, Tick creation time, toml integer trap, headers missing on HTTP 200); MEDIUM for rate-limit numbers.

**Research date:** 2026-10-11
**Valid until:** 2026-11-10 (stable; re-check Bubble Tea / Bubbles patch releases before executing)

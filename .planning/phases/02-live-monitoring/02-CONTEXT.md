# Phase 2: Live Monitoring - Context

**Gathered:** 2026-10-11
**Status:** Ready for planning

<domain>
## Phase Boundary

prv becomes a pane the user can leave open all day. It refreshes on its own at a configurable interval, and `r` refreshes at once. Refreshes are single-flight. It shows when the data was last updated and how much API rate limit remains, and it backs off by itself when the limit is low. When a refresh fails, the last data stays on screen. The list is sorted by need for attention in an order that does not shuffle, and the selection follows the PR across refreshes. The user can toggle Open/Merged/Closed, narrow the list by PR number, open the selected PR in the browser with `o`, and see the key bindings with `?`. Settings come from `config.toml`, and prv uses defaults when the file is missing.

Requirements: RFSH-01, RFSH-02, RFSH-03, RFSH-04, RFSH-05, RFSH-06, CONF-01, LIST-08, FILT-01, FILT-02, FILT-03, NAV-02, NAV-03.

Not in this phase: reviewer names, merge-readiness, failing check names, the detail view and narrow-pane layout (Phase 3); add/remove by URL and persistence (Phase 4).

</domain>

<decisions>
## Implementation Decisions

### Attention sort (LIST-08, RFSH-06)
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

### Status bar & refresh (RFSH-01..05)
- **D-05:** The header shows `updated HH:MM:SS`, which is a clock time with no redraw every second. A spinner shows next to it while a refresh is running. Example: `prv · @sbartram · 12 PRs · updated 14:32:05 ⠋`.
- **D-06:** The rate limit always shows as a faint `api 4812` at the right end of the footer. This meets success criterion 2 and keeps the limit visually quiet. When `remaining` falls below 20% of `limit`, it also shows in the header in yellow with the reset time.
- **D-07:** When a refresh fails and data is on screen, the header shows `✗ refresh failed HH:MM (<short reason>) · data from HH:MM:SS` in an ANSI warning colour. The rows are **not** dimmed, because faint already means draft/merged/closed. The next successful refresh clears the marker.
- **D-08:** During a backoff (limit low or exhausted, or retry-after active), the header shows `paused until HH:MM (rate limit)`. Pressing `r` shows a short message (`rate limited — resumes HH:MM`) and sends **no** request.
- **D-09:** When the first fetch fails and there is no data yet, the Phase 1 error panel and hint stay (Phase 1 D-09), and the auto-refresh tick keeps retrying with backoff. The panel shows `retrying at HH:MM`. `r` still retries at once, unless a rate-limit backoff blocks it (D-08).

### Filters (FILT-01..03)
- **D-10:** `1` / `2` / `3` toggle Open / Merged / Closed visibility, in any combination. The header shows which states are active (for example `[open merged ·]`). Letter keys are not used, because `o` is open-in-browser.
- **D-11:** `/` opens a one-line prompt at the bottom that accepts digits only. The list narrows live as the user types, by **prefix** match on the PR number (`12` matches #12, #120, #1234). Enter keeps the filter and returns to list navigation. Esc clears it. The prompt shows a count, such as `(2 of 14 · enter keep · esc clear)`.
- **D-12:** At launch all three states are visible, unless the config key `show` sets a different default. Toggles last only for the session; prv saves no state to disk in Phase 2.
- **D-13:** When the filters hide every PR, prv shows a filtered-empty message that is different from the D-10 empty state of Phase 1: `No PRs match: [open · ·] #99 — 3 hidden by filters · esc clear · 1/2/3 toggle`. A filter must never look as if the list was lost.

### Config & help (CONF-01, FILT-02, NAV-03)
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

### Carried forward (do not re-litigate)
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

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Scope and requirements
- `.planning/ROADMAP.md` § Phase 2: goal, success criteria and the pitfalls list (single-flight, generation counter, backoff order, selection by ref, simulated-hour test)
- `.planning/REQUIREMENTS.md`: RFSH-01..06, CONF-01, LIST-08, FILT-01..03, NAV-02, NAV-03
- `.planning/PROJECT.md`: core value, constraints (rate limits for a long-running session), Key Decisions

### Prior phase
- `.planning/phases/01-status-at-a-glance/01-CONTEXT.md`: D-01/D-05 badge and palette rules, D-04/D-07 dimming, D-06 interim sort (now superseded by D-02 here), D-09 error panel, D-10 empty state
- `.planning/phases/01-status-at-a-glance/01-REVIEW.md` and `01-REVIEW-DISPOSITION.md`: WR-01 (fixed by D-16), WR-02 and WR-03 (handled under Claude's Discretion)
- `.planning/phases/01-status-at-a-glance/01-SPIKE.md`: live GraphQL findings, including the rateLimit shape

### Research and stack
- `.claude/CLAUDE.md`: prescriptive stack. Use `tea.Tick`/`tea.Every`, not goroutine pollers. Do not use go-gh `EnableCache`. `LogIgnoreEnv: true`. Use `pkg/browser` with `io.Discard`. Use BurntSushi/toml `MetaData.Undecoded()`. XDG paths.
- `.planning/research/PITFALLS.md`: refresh, rate-limit and selection pitfalls
- `.planning/research/ARCHITECTURE.md`: model structure and refresh message flow
- `.planning/research/STACK.md`: library versions and Bubble Tea v2 API facts

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/ui/model.go` `fetchCmd`: a panic-safe Cmd wrapper. Extend it with a generation number for single-flight.
- `internal/ui/model.go` `Fetcher` interface: the seam for the counting fake client in the simulated-hour test.
- `internal/domain/list.go` `Sort` and `InWindow`: replace the `Sort` keys with buckets + createdAt + ref (D-01/D-02). `InWindow` already applies the closed window. `domain.PR` may need a `CreatedAt` field and the query a `createdAt`.
- `internal/domain/pr.go` `RateLimit`, `Snapshot`: rateLimit is decoded already but not shown (Phase 1 note). Check whether `limit` is decoded.
- `internal/ui/view.go` `renderError`, `cut`, `wrap`, and the footer build in `renderList`: the header and footer change here. Fix the WR-03 ordering where the footer is built.
- `internal/github/errors.go` `Error.Hint()`: reuse it for the short reason in D-07.

### Established Patterns
- Update/View do no I/O, and all I/O runs in a `tea.Cmd`. `ui` never imports `internal/github` (local interfaces such as `hinter`).
- Pure Update/View tests plus golden files at fixed sizes (`view_test.go`).
- The list view has no blank lines, so visible rows = height − 3. A prompt line (D-11) changes that count.
- `fetchedMsg` with `context.Canceled` is ignored on quit. The cursor is reset to 0 on every fetch today, and D-03 changes that.

### Integration Points
- `cmd/prv/main.go` `run`: load the config before `tea.NewProgram` (exit 2 on a bad config, D-15), merge `--closed-days` (D-16), and pass the settings into `ui.Options` and `github.Options`.
- `internal/github/client.go` `BuildSearch(now, closedDays)`: the closed window already comes in as a parameter.

</code_context>

<specifics>
## Specific Ideas

- Header in normal operation: `prv · @sbartram · 12 PRs · updated 14:32:05 ⠋`
- Header after a failed refresh: `prv · @sbartram · 12 PRs · ✗ refresh failed 14:33 (network) · data from 14:32:05`
- PR# prompt: `/ 12█   (2 of 14 · enter keep · esc clear)`
- Help overlay layout as previewed: key table, badge legend, `config ~/.config/prv/config.toml · every 60s · 14 days`, `api 4812 / 5000, resets 15:02`, `press any key to return`.

</specifics>

<deferred>
## Deferred Ideas

None. The discussion stayed within phase scope.

</deferred>

---

*Phase: 02-live-monitoring*
*Context gathered: 2026-10-11*

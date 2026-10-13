# Phase 4: Curated Watch List - Context

**Gathered:** 2026-10-12
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase widens the list from "my PRs" to "any github.com PR I care about". The user adds a PR by pasting a github.com PR URL or `owner/repo#n` (WATCH-04). Added PRs persist across restarts in `$XDG_DATA_HOME/prv/watchlist.json` (WATCH-05). Only manually added PRs can be removed (WATCH-06). Manual PRs ignore the closed window (WATCH-07). A saved PR that can't be fetched shows as an "unavailable" row with a reason, and the rest of the list still displays (LIST-09).

Out of scope: write actions, GHE hosts, saved searches, multi-ref paste, any new list columns.

</domain>

<decisions>
## Implementation Decisions

### Add flow (WATCH-04)
- **D-01:** `a` (list view only) opens a one-line prompt at the bottom, the same place and feel as the `/` number prompt (Phase 2 D-11). Example: `add PR: █  (URL or owner/repo#n · enter add · esc cancel)`. It takes free text and bracketed paste (`tea.PasteMsg`). The digits-only `/` prompt does not; extend it or use `bubbles/textinput`, as the planner chooses.
- **D-02:** On `enter`, prv validates the PR by **fetching it before saving** (roadmap pitfall). While it checks, the prompt stays open and shows `checking owner/repo#n ⠋`. **On error, the typed text stays** and the reason shows inline in the prompt line (e.g. `✗ not found: owner/repo#4`), so a typo can be fixed. `esc` cancels at any time; a check result that arrives after esc is dropped. **On success, the prompt closes, the entry is saved, the PR appears with full status, and the cursor jumps to the new row** (selected `Ref`). Nothing is saved on any error (malformed, not found, no access, SSO, network).
- **D-03:** **A duplicate is not an error.** If the PR is already listed (auto-found, manual, or both), prv adds no row and saves nothing. The prompt closes, the cursor jumps to the existing row, and the footer briefly shows `already watching owner/repo#n`. When the 1/2/3 or `/` filters hide that row, the message adds `· hidden by filters`. **Pasting an auto-found PR does NOT pin it as manual.** Duplicates are matched by normalised `Ref`: owner and repo are case-insensitive, and the number is an integer. Once the PR is fetched, also check its node ID, because a renamed repo can give a different ref for the same PR.
- **D-04:** **The parser is lenient for github.com PR forms only.** It accepts `https://github.com/o/r/pull/12` with an optional trailing `/files`, `/checks`, `/commits`, or `/commits/<sha>`, plus `#fragment` and `?query`. The `https://` scheme and `www.` may be missing, and surrounding whitespace is trimmed. It also accepts `o/r#12`. It rejects other hosts (GHE: `only github.com PRs are supported`), issue URLs (`that is an issue, not a PR`), and anything else (`not a PR reference`), each with its specific reason. One ref per paste. Use one anchored regexp per form, as CLAUDE.md says. go-gh `repository.Parse` is for OWNER/REPO only.

### Manual marker & removal (WATCH-06)
- **D-05:** A manually added row has a **marker glyph before the ref in the PR column** (e.g. `+ cli/cli#123`). The marker also stays in compact mode (Phase 3 D-14): `+ api#42 ✗⊘✗`. There is no new column. The `?` help legend explains it. The exact glyph is Claude's discretion (`+` was the example). It must not clash with existing glyphs and must read without colour.
- **D-06:** `d` (list view only) asks in the footer: `remove owner/repo#n from watch list? y/n`. Only `y` confirms; any other key cancels. There is no undo.
- **D-07:** A PR that is **both manual and auto-found** shows the marker (it is in the saved list). `d` removes only the saved entry, and the row stays because it is still auto-found. The footer says `unpinned — still listed (authored/assigned)`. After that the closed window applies to it again.
- **D-08:** `d` on an auto-found-only row removes nothing and shows a short footer note, e.g. `owner/repo#n is authored/assigned — only + rows can be removed`. `a` and `d` do not work in the detail view, which stays read-only.

### Unavailable rows (LIST-09)
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

### Manual PRs in the list (WATCH-07)
- **D-13:** Manual PRs are fetched and listed whatever their closed date (WATCH-07). The **1/2/3 state toggles and the `/` number filter apply to every row**, manual or auto. The window exemption only decides which PRs are fetched and kept. The state toggles never hide unavailable rows, because their state is unknown. The `/` number filter matches them by their PR number.
- **D-14:** Manual rows use the same attention buckets as auto rows (Phase 2 D-01), plus bucket 0 for unavailable rows (D-11). Merged and closed manual PRs go in the merged and closed buckets.
- **D-15:** The header count stays `N PRs`, with no split between manual and auto. When any rows are unavailable, add `· N unavailable` in the warning colour. This segment is an integrity note, so it survives narrowing before the key hints do (Phase 2 WR-03 rule). The empty state keeps the Phase 1 D-10 wording and adds `· press a to add any PR`. The footer hint list gains `a add`, plus `d remove` where it fits.
- **D-16:** The saved list stays in sync across prv instances (two panes) and hand edits. **Each add and remove reloads `watchlist.json`, applies that one change, and writes atomically** (temp file in the same dir, then `os.Rename`), so two instances never overwrite each other. **Each refresh tick also re-reads the file**, which is cheap and local, so another pane's adds and removes appear within one interval. Every file read and write runs in a `tea.Cmd` (no I/O in Update/View).

### Carried forward (do not re-litigate)
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

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Live API evidence
- `.planning/phases/01-status-at-a-glance/01-SPIKE.md` — Q3 (partial data plus `*api.GraphQLError`, NOT_FOUND shape with `path` `[alias]` / `[alias pullRequest]`, "Observed shapes" section), Q6 (`nodes(ids:)` max 100, a bad id gives null plus NOT_FOUND at `["nodes", i]`). The SAML FORBIDDEN shape was **not observed**.

### Scope and requirements
- `.planning/ROADMAP.md` §Phase 4 — goal, success criteria 1-4, pitfalls list (variables not interpolation, alias chunking, validate before persist, bracketed paste, versioned JSON store, atomic write, tolerant load, corrupt backup, SSO hint).
- `.planning/REQUIREMENTS.md` — WATCH-03..07, LIST-09.
- `.planning/PROJECT.md` — core value, Key Decisions ("Manually added PRs exempt from closed window").
- `.planning/research/PITFALLS.md`, `.planning/research/ARCHITECTURE.md` — project-level research on the watch-list store and the batched lookups.

### Prior phase decisions
- `.planning/phases/01-status-at-a-glance/01-CONTEXT.md` — badge style, columns, D-10 empty state, D-12 no provenance column.
- `.planning/phases/02-live-monitoring/02-CONTEXT.md` — attention buckets D-01, cursor-by-ref D-03, refresh-failed marker D-07, backoff D-08, `/` prompt D-11, config warnings D-15, keymap and help D-17.
- `.planning/phases/03-blocker-detail/03-CONTEXT.md` — detail view D-08/D-09, compact mode D-14, column drop order D-13.
- `.planning/phases/03-blocker-detail/03-UAT.md` — Deferred Follow-Ups: large-CI-matrix detail, team review requests, MERGE vs github.com. Re-test them with PRs added through this phase.
- `CLAUDE.md` (project) — XDG data path, atomic write, PR URL regexp guidance, no go-gh cache, Bubble Tea v2 `tea.PasteMsg`.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/ui/keys.go` `keyMap` / `promptKeys`: add `a` (Add) and `d` (Remove) bindings. ShortHelp, the detail help and the `?` overlay already render from this keymap. `promptKeys` is the digits-only `/` prompt, the model for the add prompt's place and behaviour.
- `internal/ui/model.go` `promptKey`, `rebuild()`, `nearest()`, `indexOf()`: cursor-by-ref and the move to the nearest neighbour after a row is removed.
- `internal/ui/refresh.go` `fetch` / `fetchDetail` / `beginFetch` / `arm` / `setFlash`: the single-flight and sequence-number pattern for the add-validation fetch and the per-tick watchlist re-read. `setFlash` covers the transient footer messages.
- `internal/github/errors.go` `classify` / `Error.Hint` / `Error.Short`: the error classification to extend with per-alias NOT_FOUND / FORBIDDEN / other reasons.
- `internal/github/checks.go` `fetchFailing`: the existing chunked batched-query pattern (`nodes(ids:)`, chunks of up to 100, partial-error handling) that the aliased manual lookup can follow.
- `internal/config/config.go` `Path()`: the XDG path resolution pattern (absolute `$XDG_*` or `~/.config` fallback) to mirror for `$XDG_DATA_HOME/prv/watchlist.json`.
- `internal/domain/list.go` `InWindow` / `Bucket` / `Sort` / `Visible`: the window exemption (WATCH-07), bucket 0 (D-11) and the filter rules (D-13) go here as pure functions.

### Established Patterns
- `internal/domain` is pure and does no I/O. Mapping and sanitising happen at the github seam. The ui model does no I/O in Update/View; file I/O goes in Cmds.
- `domain.Snapshot` is the unit a refresh hands to the UI. It will need a manual and unavailable notion: a `Manual` flag on `PR` and an unavailable reason, or a separate slice.
- Golden tests in `internal/ui/testdata/TestGolden/` at 0x0, 60x15, 80x24 and 120x40. Add the add prompt (checking / error), the remove confirmation, the unavailable rows, the marker in compact mode, and the empty state with the `a` hint.
- `internal/github/testdata/*.json` fixtures. Add aliased-lookup fixtures for NOT_FOUND, a constructed SAML FORBIDDEN, other errors and partial data.

### Integration Points
- `ui.Fetcher` / `github.Client.Fetch`: refresh must fetch the auto discovery and the saved refs, merge them, and dedupe by node ID. `main.go` must open the store and pass it, or its path, into `ui.Options`.
- A new store package (e.g. `internal/watchlist`) for load, save and backup, with tests that use `t.TempDir()` and `XDG_DATA_HOME`.
- `internal/ui/view.go` header and footer segments: `· N unavailable` and the new hints, following the existing `fit` priorities.
- `internal/ui/detail.go`: a reason-only detail for unavailable rows (D-11).

</code_context>

<specifics>
## Specific Ideas

- Add prompt states:
  ```
  add PR: github.com/acme/api/pull/42█   (URL or owner/repo#n · enter add · esc cancel)
  add PR: acme/api#42   checking ⠋
  add PR: acme/api#4    ✗ not found (deleted, renamed, or no access) · esc cancel
  ```
- Rows with the marker and an unavailable row:
  ```
    PR            TITLE ...
  > + acme/api#42 ⚠ unavailable: SSO: authorize your token for acme
    web#7         Bump deps ...  ✗ lint +2
    + cli/cli#9   Fix flag parsing ...  ✓ passing
  ```
- Remove confirmation in the footer: `remove cli/cli#9 from watch list? y/n`

</specifics>

<deferred>
## Deferred Ideas

- Pasting several refs at once (space- or newline-separated) — considered and not chosen. It could be a later enhancement.
- Pinning an auto-found PR as manual when it is pasted, so it outlives the closed window — considered and not chosen (D-03).
- Undo for removal — not chosen. The y/n confirmation is used instead.

</deferred>

---

*Phase: 04-curated-watch-list*
*Context gathered: 2026-10-12*

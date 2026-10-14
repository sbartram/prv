# prv

## What This Is

`prv` is a terminal UI for monitoring a chosen group of GitHub pull requests. It starts with every PR the user authored or is assigned to, and lets them add any other PR by pasting its URL. It shows each PR's state, approval status, and workflow status, and refreshes on its own. It replaces a pile of GitHub browser tabs that the user refreshes by hand.

## Current State

**Shipped:** v1.0 MVP on 2026-10-14 (5 phases, 35 plans; tag v1.0). See `.planning/MILESTONES.md` and `.planning/milestones/v1.0-*`.

prv is usable day to day: authored + assigned PRs plus a curated watch list, state/review/CI/merge badges, a detail view, auto-refresh within rate limits, and safe concurrent use from several panes. v1.0 closed as an override: phases 1–4 verification read stale at close and the milestone audit predates Phase 5.

## Next Milestone Goals

Not yet defined — start with `/gsd-new-milestone`. Candidates already on record:
- Live UAT still deferred: ROADMAP backlog 999.1 (api footer legibility) and 999.2 (live config.toml edits, never run live), plus the D-17 set (SSO, live rate-limit pause, assignee:@me, CJK glyphs, team reviews, large CI matrix, bad-token 401, theme/narrow pane)
- Remaining audit tech debt: captureTransport shared header slot (P3 WR-03), detail/Lookup rate cost not counted (G2), unknown JSON keys dropped in watchlist.json (P4 WR-04), open Info findings
- Release packaging (GoReleaser, Homebrew cask) was planned in research but not built in v1.0

## Core Value

At a glance, without opening a browser, the user can see which of their watched PRs are approved, blocked, or failing CI.

## Requirements

### Validated

- ✓ Watch list automatically includes all PRs the user authored or is assigned to — Phase 1 (live assignee check deferred in UAT: no assigned PRs on the account)
- ✓ List shows each PR's state: Open / Merged / Closed — Phase 1
- ✓ List shows one rollup workflow badge per PR (passing / failing / running) — Phase 1
- ✓ Authenticates with the same token `gh` would use: `GH_TOKEN`, then `GITHUB_TOKEN`, then the `gh` CLI login — Phase 1
- ✓ User can filter the list by any combination of Open / Merged / Closed — Phase 2 (real-terminal UAT passed in Phase 5)
- ✓ Auto-found merged/closed PRs only appear if closed within a configurable recent window (default 14 days) — Phase 2 (manual-PR exemption lands with WATCH-07 in a later phase)
- ✓ Data refreshes automatically on a configurable interval, with failure marker and rate-limit backoff — Phase 2 (failure marker UAT passed in Phase 5; live rate-limit pause still deferred, D-17)
- ✓ User can open the selected PR in the browser with a key — Phase 2 (real-browser UAT passed in Phase 5)
- ✓ List shows the overall review decision plus which reviewers approved, requested changes or are pending, merge-readiness, and the failing check name — Phase 3 (badges and MERGE vs github.com passed in Phase 5 UAT; team-reviewer UAT still deferred, D-17)
- ✓ Selecting a PR shows a detail view with each check/job and its status, failing first, and every reviewer — Phase 3 (large-CI-matrix UAT deferred until add-by-URL)

- ✓ User can add any PR by pasting a github.com PR URL or `owner/repo#123` — Phase 4 (real-SSO-org UAT deferred)
- ✓ Manually added PRs persist across restarts in a local file until the user removes them, and stay listed after they close — Phase 4
- ✓ User can remove a manually added PR from the watch list — Phase 4

### Active

(none — all v1 requirements validated)

### Out of Scope

- Write actions (merge, approve, comment, re-run workflows): v1 is watch-only plus "open in browser"; acting happens on github.com
- PRs where the user's review is requested (when not author or assignee): the user chose authored + assigned only
- Full closed/merged history: noise; only a recent window is shown
- GitHub Enterprise hosts: github.com only for v1
- Desktop/push notifications: the user leaves the TUI open in a pane and looks at it

## Context

- Today the user keeps GitHub PR pages open in browser tabs and refreshes them by hand
- Intended use: a long-running TUI in a terminal pane (e.g. a multiplexer split)
- Prior art: `gh-dash` (Go + Bubble Tea) is the closest existing tool; `prv` is narrower, with a curated watch list and a focus on approvals and CI
- GitHub GraphQL API can return review decision, reviews, and status check rollup in one query, which keeps polling cheap against rate limits
- After v1.0: ~21.2k lines of Go (~14.7k of them tests, incl. golden files and an opt-in live smoke test); stack as in Constraints, no cgo

## Constraints

- **Tech stack**: Go + Bubble Tea — single static binary; the established choice for GitHub TUIs
- **Auth**: Same token resolution as `gh` (env vars, then `gh` login); no separate login flow — the user already uses `gh`
- **Repository / module path**: https://github.com/sbartram/prv — Go module `github.com/sbartram/prv`
- **Host**: github.com only
- **Rate limits**: Auto-refresh must stay well inside GitHub API rate limits for a long-running session

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go + Bubble Tea | Single binary; gh-dash shows it fits this domain | ✓ Good (Phase 1) |
| "Owned" = authored + assigned | User's definition; review-requested excluded | ✓ Good (v1.0) |
| Three PR states: Open / Merged / Closed | User wants merged and closed-unmerged kept apart | ✓ Good (Phase 1) |
| Closed PRs limited to a recent window | Avoids years of history flooding the list | ✓ Good (v1.0) |
| Watch-only + open in browser | Keeps v1 small; actions stay on github.com | ✓ Good (v1.0) |
| Auth matches gh: GH_TOKEN → GITHUB_TOKEN → gh login | Same token gh uses; go-gh TokenForHost unchanged | ✓ Good (Phase 1) |
| Manually added PRs exempt from closed window | User added them on purpose; they stay until removed | ✓ Good (Phase 4) |
| Clean remote strings once at the mapper seam | One audit point; go-gh already defangs C0 controls, Clean adds bidi/other controls | ✓ Good (Phase 1) |
| ANSI-16 colours + glyph/word badges, reverse-video selection | Readable on light and dark themes without background detection; not colour-only | ✓ Good (Phase 1) |
| Bubble Tea v2 (charm.land/*/v2) + go-gh v2 | v2 is stable; go-gh gives GraphQL, auth, browser | ✓ Good (Phase 1) |
| Failing check names in a second batched query; full check list fetched lazily for the open detail only (C3) | Keeps the list query at rollup cost; one Detail query per refresh only while the view is open | ✓ Good (Phase 3) |
| Partial data is never shown as complete: Truncated (runs not read) and Partial (GraphQL error kept) are separate, each with its own notice | The detail screen is where the user decides why a PR is blocked | ✓ Good (Phase 3) |
| Watch list store keeps no state: every call re-reads watchlist.json, writes via temp file + rename, and never overwrites a file it cannot parse (backs it up once instead) | Two prv panes share one file; a hand-edit mistake must never lose saved PRs | ✓ Good (Phase 4) |
| Saved PRs GitHub cannot return stay on the list as unavailable rows with a reason, never silently dropped | A saved PR must disappear only when the user removes it | ✓ Good (Phase 4) |
| Epoch rule: a refresh begun before a local add/remove is discarded and re-run | The screen never shows a list older than the user's own change | ✓ Good (Phase 4) |
| Watch-list writers take an exclusive flock on watchlist.json.lock (3 s wait, then "busy"); reads stay lock-free; a file of another version is refused, never overwritten | Two prv panes writing at once must not lose a change; a newer prv's file must survive an older prv | ✓ Good (Phase 5) |
| Opt-in live smoke (PRV_LIVE=1) against pinned public PRs; default `go test ./...` stays offline | Checks the GitHub mapping against reality without making CI depend on the network | ✓ Good (Phase 5) |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-10-14 after v1.0 milestone*

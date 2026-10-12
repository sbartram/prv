# prv

## What This Is

`prv` is a terminal UI for monitoring a chosen group of GitHub pull requests. It starts with every PR the user authored or is assigned to, and lets them add any other PR by pasting its URL. It shows each PR's state, approval status, and workflow status, and refreshes on its own. It replaces a pile of GitHub browser tabs that the user refreshes by hand.

## Core Value

At a glance, without opening a browser, the user can see which of their watched PRs are approved, blocked, or failing CI.

## Requirements

### Validated

- ✓ Watch list automatically includes all PRs the user authored or is assigned to — Phase 1 (live assignee check deferred in UAT: no assigned PRs on the account)
- ✓ List shows each PR's state: Open / Merged / Closed — Phase 1
- ✓ List shows one rollup workflow badge per PR (passing / failing / running) — Phase 1
- ✓ Authenticates with the same token `gh` would use: `GH_TOKEN`, then `GITHUB_TOKEN`, then the `gh` CLI login — Phase 1

### Active

- [ ] User can add any PR by pasting a github.com PR URL or `owner/repo#123`
- [ ] Manually added PRs persist across restarts in a local file until the user removes them
- [ ] User can remove a manually added PR from the watch list
- [ ] User can filter the list by any combination of Open / Merged / Closed
- [ ] Auto-found merged/closed PRs only appear if closed within a configurable recent window (default 14 days); manually added PRs are exempt
- [ ] List shows the overall review decision (approved / changes requested / review required) plus which reviewers approved or requested changes
- [ ] Selecting a PR shows a detail view with each check/job and its status
- [ ] Data refreshes automatically on a configurable interval
- [ ] User can open the selected PR in the browser with a key

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
| "Owned" = authored + assigned | User's definition; review-requested excluded | — Pending |
| Three PR states: Open / Merged / Closed | User wants merged and closed-unmerged kept apart | ✓ Good (Phase 1) |
| Closed PRs limited to a recent window | Avoids years of history flooding the list | — Pending |
| Watch-only + open in browser | Keeps v1 small; actions stay on github.com | — Pending |
| Auth matches gh: GH_TOKEN → GITHUB_TOKEN → gh login | Same token gh uses; go-gh TokenForHost unchanged | ✓ Good (Phase 1) |
| Manually added PRs exempt from closed window | User added them on purpose; they stay until removed | — Pending |
| Clean remote strings once at the mapper seam | One audit point; go-gh already defangs C0 controls, Clean adds bidi/other controls | ✓ Good (Phase 1) |
| ANSI-16 colours + glyph/word badges, reverse-video selection | Readable on light and dark themes without background detection; not colour-only | ✓ Good (Phase 1) |
| Bubble Tea v2 (charm.land/*/v2) + go-gh v2 | v2 is stable; go-gh gives GraphQL, auth, browser | ✓ Good (Phase 1) |

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
*Last updated: 2026-10-12 after Phase 1*

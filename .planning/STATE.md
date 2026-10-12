---
gsd_state_version: "1.0"
current_phase: 3
current_phase_name: Blocker Detail
status: planning
stopped_at: Phase 3 context gathered
last_updated: "2026-10-12T19:01:02.395Z"
last_activity: 2026-10-12
last_activity_desc: Phase 02 complete, transitioned to Phase 3
state_head: a40034928aef69b4e1829ada3e0e0dea0615d1d6
progress:
  total_phases: 4
  completed_phases: 2
  total_plans: 14
  completed_plans: 14
  percent: 50
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-10-12)

**Core value:** At a glance, without opening a browser, the user can see which of their watched PRs are approved, blocked, or failing CI.
**Current focus:** Phase 3 — Blocker Detail

## Current Position

Phase: 3 — Blocker Detail
Plan: Not started
Status: Ready to plan
Last activity: 2026-10-12 — Phase 02 complete, transitioned to Phase 3

Progress: [█████░░░░░] 50%

## Performance Metrics

**Velocity:**
- Total plans completed: 14
- Average duration: -
- Total execution time: 0.0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 4 | - | - |
| 02 | 10 | - | - |

**Recent Trend:**
- Last 5 plans: -
- Trend: -

*Updated after each plan completion*
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 01 P01 | 6 min | 2 tasks | 13 files |
| Phase 01 P02 | 4 min | 2 tasks | 11 files |
| Phase 01 P03 | 12min | 2 tasks | 17 files |
| Phase 01 P04 | 40min | 2 tasks | 2 files |
| Phase 02 P01 | 15min | 2 tasks | 9 files |
| Phase 02 P02 | 12min | 3 tasks | 12 files |
| Phase 02 P03 | 6 min | 2 tasks | 7 files |
| Phase 02 P04 | 3 min | 2 tasks | 18 files |
| Phase 02 P05 | 4 min | 2 tasks | 10 files |
| Phase 02 P06 | 6 min | 2 tasks | 7 files |
| Phase 02 P07 | 8 min | 2 tasks | 5 files |
| Phase 02 P08 | 14min | 2 tasks | 20 files |
| Phase 02 P09 | 18 min | 2 tasks | 7 files |
| Phase 02 P10 | 6 min | 2 tasks | 23 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: Vertical MVP in 4 phases. Phase 1 is a thin live slice (auth + GraphQL fetch + list with state/review/CI badges).
- [Roadmap]: The live-verification spike moves from research's "Phase 2" to the start of Phase 1, as a gate before the query is frozen.
- [Research C1]: Resolved. Token order is GH_TOKEN → GITHUB_TOKEN → gh login (go-gh TokenForHost), per PROJECT.md. Spike Q5 confirms it on the installed gh.
- [Research C5]: Resolved. Manually added PRs are exempt from the closed window (WATCH-07).
- [Research C2/C4]: Defaults recorded in REQUIREMENTS.md (single ISSUE_ADVANCED search; latestOpinionatedReviews fallback). The Phase 1 spike confirms or overturns them.
- [Phase 01]: go-gh defangs C0 control characters in API responses (ESC becomes visible ^[), so domain.Clean at the mapper seam is a second layer, tested through the client and directly
- [Phase 01]: State zero value is StateOpen per the plan interface order; unknown remote states map to StateUnknown via ParseState and are never shown as open
- [Phase 01]: 01-02: a GraphQL error on page 2 or later discards that page; earlier pages stay and the list is marked truncated
- [Phase 01]: 01-02: API error messages are not token-scrubbed; built from status, fixed strings and domain.Clean(API message)
- [Phase 01]: 01-03: TITLE width is max(10, min(widest title, space left)); narrow panes cut at the right edge until NAV-05
- [Phase 01]: 01-03: list view has no blank lines (header, column row, rows, footer) so visible rows = height - 3
- [Phase 01]: 01-04: GSD runtime files .planning/state.json and .planning/milestone.lock are gitignored; .planning stays tracked
- [Phase 02]: 02-01: refresh interval floor (15 s) lives in config.Load, not the model; generation numbering starts at 0 in New()
- [Phase 02]: 02-01: unresolved config path keeps defaults; a broken config.toml exits 2 before the token lookup, and --version ignores it
- [Phase 02]: 02-02: bubbles/key is imported as bkey in internal/ui because the test helper key() owns that package-level name
- [Phase 02]: 02-02: selected domain.Ref is the source of truth; rebuild() derives rows and cursor; nearest neighbour looks below first, then above
- [Phase 02]: A merged or closed draft PR sorts as merged (7) or closed (8); only an open draft is bucket 6 (D-04 as clarified 2026-10-12)
- [Phase 02]: Approved PR with no checks or unknown CI sorts in bucket 5, never ready to merge; updatedAt is not a sort key (D-02)
- [Phase 02]: 02-04: header segments carry priorities (prv 1, @login 2, N PRs 4, updated 5, failure marker 7); prio 3 and 6 are reserved for the filter segment (02-08) and rate warning (02-05)
- [Phase 02]: 02-04: footer shows integrity notes before key hints; notes are cut with an ellipsis, never dropped; hints need at least 8 cells and come from keyMap.ShortHelp()
- [Phase 02]: D-06 'always shows' read as 'from the first successful response on': api segment absent while Limit is 0, last value kept through failed refreshes
- [Phase 02]: Rate warning boundary is Remaining*5 < Limit (999/5000 warns, 1000/5000 does not); header prio 6, footer drops hints before notes or api
- [Phase 02]: 02-06: a current-generation tick that fires before pausedUntil re-arms instead of fetching; r during a pause keeps the generation and only shows the 4 s flash
- [Phase 02]: 02-06: NextDelay is pure in domain (clock passed in); ui reads Backoff() through its own backoffer interface, so 02-07 adds the method to *github.Error without ui importing it
- [Phase 02]: 02-07: a rate limit on any discovery page returns an empty snapshot and the error (previous data stays, model pauses); other later-page errors still truncate
- [Phase 02]: 02-07: rate limit decided from captured headers first, RATE_LIMITED error type second, never message text; Retry-After saturates at 2^31 s
- [Phase 02]: 02-08: unknown-state PRs are never hidden by 1/2/3; the number prompt opens only on the list screen and its keys are handled before the list keys
- [Phase 02]: 02-08: rows = domain.Visible(snap.PRs, show, numFilter) in rebuild(); bottom() is the prompt while open, else the footer, so the prompt costs no list row
- [Phase 02]: 02-09: config values are clamped to 1..365 while the --closed-days flag is rejected outside the same range; flag beats config beats default 14
- [Phase 02]: 02-09: config warnings start their 8 s clock at the first fetch result and sit in their own footer segment, below the flash and notes and above key hints
- [Phase 02]: 02-10: help overlay keeps 'press any key to return' as the last line when cut to pane height; safeURL refuses whitespace/control runes before parsing

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 3]: C3 is open: check contexts eager vs lazy. Decide before planning Phase 3. LIST-07 (failing check name in rows) affects the choice.
- [Phase 1]: Open code-review warnings (01-REVIEW-DISPOSITION.md): WR-01 unbounded hidden --closed-days, WR-02 empty state hides partial-fetch warnings, WR-03 footer integrity notes truncated first. Triage WR-02/WR-03 with Phase 2 footer/refresh work.
- [Phase 1]: 4 UAT checks deferred (01-UAT.md Deferred Follow-Ups): live assignee:@me, badge fidelity vs github.com, real 401 flow, ambiguous-width glyphs.
- [Phase 2]: 7 UAT checks deferred (02-UAT.md Deferred Follow-Ups): failure marker at 80/40 cols, api footer/rate warning legibility, live rate-limit pause, filter keys and paste, `o` in a real browser, `?` overlay, live config edits. VERIFICATION.md was set to passed on the deferral, not on observed behaviour.
- [Phase 1]: Throwaway test PR sbartram/prv#1 (branch uat/test-pr-scratch) is open; close with `gh pr close 1 --delete-branch`.

## Deferred Items

Items acknowledged and deferred at milestone close, most recent first:

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| *(none)* | | | | |

## Session Continuity

Last session: 2026-10-12T19:01:02.355Z
Stopped at: Phase 3 context gathered
Resume file: .planning/phases/03-blocker-detail/03-CONTEXT.md

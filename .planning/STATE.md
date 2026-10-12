---
gsd_state_version: "1.0"
current_phase: 03
current_phase_name: Blocker Detail
status: verifying
stopped_at: Completed 03-08-PLAN.md
last_updated: "2026-10-12T23:53:10.907Z"
last_activity: 2026-10-12
last_activity_desc: Phase 03 execution started
state_head: a253086de92db9e1571661887e5e41b491940c4d
progress:
  total_phases: 4
  completed_phases: 2
  total_plans: 22
  completed_plans: 22
  percent: 50
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-10-12)

**Core value:** At a glance, without opening a browser, the user can see which of their watched PRs are approved, blocked, or failing CI.
**Current focus:** Phase 03 — Blocker Detail

## Current Position

Phase: 03 (Blocker Detail) — EXECUTING
Plan: 8 of 8
Status: Phase complete — ready for verification
Last activity: 2026-10-12 — Phase 03 execution started

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
| Phase 03 P01 | 4 min | 2 tasks | 11 files |
| Phase 03 P02 | 3 min | 2 tasks | 7 files |
| Phase 03 P03 | 5 min | 2 tasks | 13 files |
| Phase 03 P04 | 20 min | 2 tasks | 10 files |
| Phase 03 P05 | 8 min | 2 tasks | 20 files |
| Phase 03 P06 | 9 min | 2 tasks | 8 files |
| Phase 03 P07 | 30 min | 2 tasks | 35 files |
| Phase 03 P08 | 6 min | 2 tasks | 22 files |

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
- [Phase 03]: 03-01: '+N' counts only other failing checks when a failing check is named, and other cancelled checks when the named check is cancelled — Row agrees with the detail summary, which shows failing and cancelled separately (D-10)
- [Phase 03]: 03-01: truncated contexts page, missing node, null rollup or failed query keeps the plain failing badge; a check name is never guessed — LIST-07 v1 limit accepted in D-02
- [Phase 03]: 03-02: Step-2 failing-name behavior already held; tests pinned and proven by mutation, no production change — 03-01 implementation covered re-runs, truncation, partial errors, chunking, rate limit and cancel
- [Phase 03]: 03-02: ciCell cuts only the check name with ansi.Truncate and keeps (cancelled) and +N; under 2 name cells it shows the cut plain badge — Display-width safe for CJK and ZWJ emoji; suffix carries cancelled/more information
- [Phase 03]: 03-03: compact 'widen for columns' hint sits on the column-header line, not the header line — Keeps the header segments asserted by filter_test.go at 36 cells unchanged
- [Phase 03]: 03-03: a compact row with no room for a ref draws only the marker and the REVIEW and CI glyphs — Approved/blocked/failing state stays visible down to 4 cells
- [Phase 03]: 03-04: review logins lose [bot] like request names, so a re-requested bot shows once as pending
- [Phase 03]: 03-04: PR holds a Reviewers slice, so PR values are no longer comparable with ==; tests compare with reflect.DeepEqual
- [Phase 03]: 03-05: reviewers legend in the ? overlay is two lines, because the one-line text is 82 cells and wrapped mid-token at 80
- [Phase 03]: 03-05: renderList cell and header tables are [colCount]string arrays indexed by column constant, so inserting a column cannot shift another column's data
- [Phase 03]: Detail counts are the de-duplicated counts at every size (D-03 reading of D-08); paging to 5 pages keeps them exact to 500 runs, past that Read/Total disclose the partial read
- [Phase 03]: A later Detail page that fails (not rate limit or cancel) keeps the earlier pages and sets Truncated; a rate limit or cancel on any page is returned
- [Phase 03]: 03-07: openDetail takes the PR from Update so detail.go never reads the list row at the cursor
- [Phase 03]: 03-07: r in the detail also refetches the detail when it failed with no data; footer drops whole hints from the end when help would overflow
- [Phase 03]: D-12 'the list' is the rows on screen after the filters, so the no-longer-in-list note and the esc landing spot agree
- [Phase 03]: A rate-limited detail error extends pausedUntil directly and arms no tick; the pending tick re-arms to the end of the pause
- [Phase 03]: schedule() keeps a still-running pause, so a list result cannot end a pause a detail error set
- [Phase 03]: A gone PR is not refetched; o still opens the detail PR's own URL

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

Last session: 2026-10-12T23:53:10.864Z
Stopped at: Completed 03-08-PLAN.md
Resume file: None

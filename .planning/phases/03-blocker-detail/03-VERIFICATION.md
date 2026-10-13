---
phase: 03-blocker-detail
verified: 2026-10-13T02:00:00Z
status: human_needed
score: 4/4 roadmap truths verified; 3/3 gap-closure truths (WR-01, WR-02, WR-04) verified
covered_files:
  - ".planning/phases/03-blocker-detail/03-01-PLAN.md"
  - ".planning/phases/03-blocker-detail/03-01-SUMMARY.md"
  - ".planning/phases/03-blocker-detail/03-02-PLAN.md"
  - ".planning/phases/03-blocker-detail/03-02-SUMMARY.md"
  - ".planning/phases/03-blocker-detail/03-03-PLAN.md"
  - ".planning/phases/03-blocker-detail/03-03-SUMMARY.md"
  - ".planning/phases/03-blocker-detail/03-04-PLAN.md"
  - ".planning/phases/03-blocker-detail/03-04-SUMMARY.md"
  - ".planning/phases/03-blocker-detail/03-05-PLAN.md"
  - ".planning/phases/03-blocker-detail/03-05-SUMMARY.md"
  - ".planning/phases/03-blocker-detail/03-06-PLAN.md"
  - ".planning/phases/03-blocker-detail/03-06-SUMMARY.md"
  - ".planning/phases/03-blocker-detail/03-07-PLAN.md"
  - ".planning/phases/03-blocker-detail/03-07-SUMMARY.md"
  - ".planning/phases/03-blocker-detail/03-08-PLAN.md"
  - ".planning/phases/03-blocker-detail/03-08-SUMMARY.md"
  - ".planning/phases/03-blocker-detail/03-09-PLAN.md"
  - ".planning/phases/03-blocker-detail/03-09-SUMMARY.md"
  - ".planning/phases/03-blocker-detail/03-10-PLAN.md"
  - ".planning/phases/03-blocker-detail/03-10-SUMMARY.md"
  - "cmd/prv/main.go"
  - "cmd/prv/main_test.go"
  - "internal/domain/backoff.go"
  - "internal/domain/checks.go"
  - "internal/domain/merge.go"
  - "internal/domain/pr.go"
  - "internal/domain/reviewers.go"
  - "internal/domain/status.go"
  - "internal/github/checks.go"
  - "internal/github/client.go"
  - "internal/ui/cells.go"
  - "internal/ui/detail.go"
  - "internal/ui/help.go"
  - "internal/ui/keys.go"
  - "internal/ui/layout.go"
  - "internal/ui/model.go"
  - "internal/ui/refresh.go"
  - "internal/ui/styles.go"
  - "internal/ui/view.go"
covered_digest: "v3:sha256:5fcc408a813ed98bf7444b9f6c7538f3af04e2265ccb8a47a5f469f7021bc68c"
behavior_unverified: 0
overrides_applied: 0
gaps: []
advisory_warnings:
  - id: WR-05
    file: "internal/ui/detail.go:308-310 (cause: internal/github/checks.go:331-335)"
    summary: "Truncated conflates 'fewer runs read than exist' with 'a GraphQL error was kept'. A reviewer-only partial error prints 'read 3 of 3 runs'; a no-checks PR with a kept error prints 'CHECKS  read 0 of 0 runs' instead of 'no checks'. Confirmed by reading both files."
    decision: "Not goal-blocking; owner decides fix now or defer."
  - id: WR-03
    summary: "Shared latest-headers slot in captureTransport; detail cost not in the budget. Deferred by the user in UAT."
  - id: IN-06
    summary: "ServerWait has no fallback to the snapshot's rl.ResetAt and its doc comment overstates parity with NextDelay."
  - id: IN-07
    summary: "Help legend does not name the new 'unknown' check class."
human_verification:
  - test: "Run `prv` in a real terminal, open the detail (enter) of a PR with a large CI matrix. Check failing checks sit on top, unknown checks stay listed, passing and skipped ones fold into '... +N more passing or skipped', e expands and collapses, j/k scroll, esc returns to the same PR."
    expected: "Layout is readable; the selected PR is unchanged after esc."
    why_human: "Layout and readability in a real terminal cannot be asserted by golden tests. Deferred in UAT test 1."
  - test: "Leave the detail open across a refresh (press r), then drag the pane narrow (about 30 columns) and wide again."
    expected: "The 'updated' time and the checks refresh; the body re-lays out; the footer stays on the last line; esc returns to the same PR. The list drops columns in the D-13 order and compact rows appear below 40 columns."
    why_human: "Resize and refresh feel in a real terminal. Deferred in UAT test 2."
  - test: "Run against a PR that has a visible team review request (an org account)."
    expected: "The row shows '...@org/slug' and the detail shows '@org/slug  pending'; a hidden team shows as '...team'."
    why_human: "The `... on Team { combinedSlug }` branch has not been observed live. Deferred in UAT test 3."
  - test: "Compare the MERGE column with the github.com merge box for several real PRs (mergeable, blocked, behind, conflicts, draft)."
    expected: "Each row agrees with GitHub's own merge state."
    why_human: "Agreement with the live merge box needs a person and real PRs. Deferred in UAT test 4."
  - test: "Check the glyph widths of the check, merge, reviewer and compact-row glyphs (including the new unknown-check glyph) in the user's font."
    expected: "Every glyph is one cell wide; no column shifts."
    why_human: "Font-dependent width. Deferred in UAT test 5."
  - test: "Decide the disposition of WR-05 in 03-REVIEW-DISPOSITION.md (open): fix before ship, or defer."
    expected: "WR-05 has a recorded disposition."
    why_human: "It is a display-honesty edge case on partial GraphQL data, an owner call (see Gaps Summary)."
---

# Phase 3: Blocker Detail Verification Report

**Phase Goal:** User can see, without leaving the terminal, who and what is holding each PR up: which reviewers, which failing check, and whether it can merge
**Verified:** 2026-10-13
**Status:** human_needed
**Re-verification:** Yes. The earlier report was stale (covered source changed). This run re-checked every roadmap truth and the gap-closure plans 03-09 and 03-10. The prior report recorded `gaps: []`, so every truth got a fresh check, not only the gap items.

## Goal Achievement

The goal is achieved in the code. The row shows reviewers, merge-readiness and the name of a failing check. The detail view lists every check (failing first) and every reviewer. The three UAT gaps the user marked "fix before ship" (WR-01, WR-02, WR-04) are closed in code, tests and goldens. WR-05, raised by the re-review, is a real but narrow defect in how a partial read is worded. It does not make any must-have false, so I did not record it as a gap. Status is `human_needed` because five real-terminal checks were deferred by the user in UAT, and WR-05 needs an owner decision.

Evidence I ran myself, not taken from SUMMARY.md:

- `go build ./... && go vet ./...` clean. `go test ./... -count=1 -race`: all 5 packages pass (one full run).
- Named tests, run singly with `-v`: `TestFetchDetailPartial` (3 subtests), `TestDetailPartialEndToEnd`, `TestDetailEndToEnd`, `TestFailingNameEndToEnd`, `TestServerWait`, `TestDetailRateLimitExtendsPause`, `TestUnknownAlwaysVisible`, `TestDetailTruncatedSummary`: all PASS.
- No `TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER` in non-test, non-golden source changed since `034f67d`.
- Code read: `ServerWait` (`backoff.go:93-101`), `pauseForRateLimit` (`detail.go:146-163`), `checkLines` and `checksSummary` (`detail.go:290-345`), `FetchDetail` and `detailPage` (`github/checks.go:285-350`).
- Live query check from the earlier run (the exact `detailQuery` / `failingQuery` accepted by github.com, cost 1) still applies: neither query text changed in 03-09 or 03-10 (those plans touched `FetchDetail`/`detailPage` handling, not the query constants).

### Observable Truths (roadmap contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Each row shows approved / changes-requested / pending reviewers plus merge-readiness (mergeable, blocked, behind, conflicts) | VERIFIED | `DeriveMerge`, `MergeReviewers` filled in `client.go` for each PR; `list_120x40.golden` shows approved, reviewer names, `conflicts`, `behind`. Compact rows keep review and merge glyphs. |
| 2 | The row of a PR with failing CI names one of its failing checks (v1 limit: >100 contexts gives the plain badge) | VERIFIED | `fetchFailing` feeds `SummarizeFailing`; no name from a truncated page; `TestFailingNameEndToEnd` passes through the real client. |
| 3 | Selecting a PR opens a detail of every check (failing first) and every reviewer; returning keeps the same PR selected, also after a refresh while open | VERIFIED | `openDetail`/`closeDetail`/`refreshDetail`; `TestDetailEndToEnd`, `TestDetailOpenClose`, `TestDetailRefreshes`, `TestDetailEscAfterReorder` pass. Unknown checks are now always listed (`TestUnknownAlwaysVisible`). Passing checks fold by design (D-10). |
| 4 | In a narrow pane and after a resize, list and detail drop or truncate columns and stay readable | VERIFIED | `layoutColumns` D-13 drop order; compact rows below 40 cells; width/height sweep tests; goldens at 60x15, 80x24, 120x40, 0x0 for list and detail. Real-drag feel is a human item. |

**Score:** 4/4 roadmap truths verified; 0 present-behavior-unverified.

### Gap-closure truths (03-09, 03-10)

| Truth | Status | Evidence |
|-------|--------|----------|
| WR-01: detail rate-limit pause uses the list path's clamp | VERIFIED | `pauseForRateLimit` sets `pausedUntil` from `now.Add(domain.ServerWait(now, after, reset))` and never moves earlier. `ServerWait` clamps Retry-After to [1 s, 65 min], cuts a reset span before adding 1 s (no wrap), and returns `RateLimitFloor` for a past or missing reset. `TestServerWait` and `TestDetailRateLimitExtendsPause` pass, including hostile inputs. |
| WR-02: partial detail reads are marked Truncated | VERIFIED in the data layer, display imperfect (WR-05) | `FetchDetail` sets `Truncated` when `partial` (kept GraphQL error), when a later page fails, and when `Read < Total` (null node skipped without `Read++`). `TestFetchDetailPartial` (null node, page-1 error, page-2 error) and `TestDetailPartialEndToEnd` pass. Complete reads still have `Truncated == false`. Rate limit and cancel on any page, and non-GraphQL failure on page 1, still return errors. |
| WR-04: unknown checks never shown as passing or counted as skipped | VERIFIED | `CheckCounts.Other` exists; `checksSummary` adds `c.Other` as "unknown" after "skipped"; `checkLines` lists `CheckOther` in the collapsed view; the fold line reads "more passing or skipped". `TestUnknownAlwaysVisible` and the regenerated goldens pass. |

### Plan-level prohibitions (spot-checked)

| Item | Status | Evidence |
|------|--------|----------|
| No inflated counts; early paging disclosed | VERIFIED, one wording edge | Paging cap, failed later page, null nodes and kept errors all set `Truncated`. Edge: the wording is wrong in two cases (WR-05). |
| Unknown check never shown as passing | VERIFIED | Closed by 03-10 (WR-04). |
| Detail request only while open, one in flight, none during a pause | VERIFIED | Unchanged from the earlier run; `TestDetailNoFetchWhenClosed`, `TestDetailNoOverlap`, `TestDetailPausedNoRequest` pass in the full run. |
| Out-of-scope guards: 03-09 and 03-10 did not touch `captureTransport` or WR-03 areas | VERIFIED | `internal/github/ratelimit.go` unchanged by those plans; WR-03 stays deferred. |

### Required Artifacts

| Artifact | Status | Details |
|----------|--------|---------|
| `internal/domain/backoff.go` (`ServerWait`) | VERIFIED | Exported, substantive, used by `detail.go`. |
| `internal/domain/checks.go` (`CheckCounts.Other`) | VERIFIED | Used by `checksSummary`. |
| `internal/github/checks.go` (`FetchDetail`, `detailPage`) | VERIFIED | `partial` flag returned and consumed. |
| `internal/ui/detail.go` | VERIFIED | Clamped pause, unknown listing, reworded fold. |
| Tests and goldens (`detail_test.go`, `backoff_test.go`, `detail_test.go` in github, `main_test.go`) | VERIFIED | All named tests pass; 12 detail goldens regenerated and passing. |

### Key Link Verification

| From | To | Status | Details |
|------|----|--------|---------|
| `pauseForRateLimit` | `domain.ServerWait` | WIRED | `detail.go` calls it directly. |
| `checksSummary` | `CheckCounts.Other` | WIRED | `add(c.Other, "unknown", domain.CheckOther)`. |
| `detailPage` partial flag | `FetchDetail` `Truncated` | WIRED | `if partial { d.Truncated = true }`. |
| `Truncated` | "read N of M runs" text | WIRED, imprecise | One boolean drives one sentence for two causes (WR-05). |
| Earlier wiring (Fetch to fetchFailing, mapResponse to Merge/Reviewers, enter to openDetail, fetchedMsg to refreshDetail, detailMsg to applyDetail) | | WIRED | Unchanged by 03-09/03-10; end-to-end tests pass. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `ciCell` | `pr.Failing` | FailingChecks GraphQL query | Yes | FLOWING |
| `reviewersCell` | `pr.Reviewers` | discovery query | Yes | FLOWING |
| detail checks / reviewers / counts | `detail.data` | `node(id:)` Detail query, de-duplicated | Yes | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Partial detail end to end through the real client | `go test ./cmd/prv -run TestDetailPartialEndToEnd -v` | PASS | PASS |
| Detail open/refresh/esc end to end | `go test ./cmd/prv -run TestDetailEndToEnd -v` | PASS | PASS |
| Failing name end to end | `go test ./cmd/prv -run TestFailingNameEndToEnd -v` | PASS | PASS |
| Clamp on hostile Retry-After / reset | `go test ./internal/domain -run TestServerWait -v` | PASS | PASS |
| Unknown checks always listed | `go test ./internal/ui -run TestUnknownAlwaysVisible -v` | PASS | PASS |

### Probe Execution

Step 7c: SKIPPED. The phase declares no probe scripts.

### Requirements Coverage

| Requirement | Source plans | Description | Status | Evidence |
|-------------|--------------|-------------|--------|----------|
| LIST-04 | 03-04, 03-05 | Reviewers approved / changes requested / pending in the row | SATISFIED | Truth 1; real-team case is a human item |
| LIST-06 | 03-04, 03-05 | Merge-readiness in the row | SATISFIED | Truth 1; comparison with the live merge box is a human item |
| LIST-07 | 03-01, 03-02 | Failing check name in the row | SATISFIED | Truth 2 (accepted >100-context limit) |
| DETL-01 | 03-06, 03-07, 03-09, 03-10 | Detail lists every check, failing first | SATISFIED | Truth 3; unknown now listed |
| DETL-02 | 03-06, 03-07, 03-09 | Detail lists each reviewer and state | SATISFIED | Truth 3 |
| DETL-03 | 03-07, 03-08 | Return keeps the same PR selected | SATISFIED | Truth 3 |
| NAV-05 | 03-03, 03-05, 03-08 | Layout adapts to resizes and narrow panes | SATISFIED | Truth 4; real drag is a human item |

All 7 IDs appear in at least one PLAN `requirements:` field and in REQUIREMENTS.md (checked, marked Complete). Plan 03-10 also lists RFSH-05, which belongs to an earlier phase and is not a Phase 3 requirement. No orphaned Phase 3 requirements.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/ui/detail.go` | 308-310 | One `Truncated` flag drives "read N of M runs" for two causes (WR-05) | WARNING | A reviewer-only kept error shows "read 3 of 3 runs" (looks complete); a no-checks PR with a kept error shows "read 0 of 0 runs" instead of "no checks". The user is not told that reviewers may be incomplete. Not goal-blocking. |
| `internal/domain/backoff.go` | 93-101 | `ServerWait` has no `rl.ResetAt` fallback; doc overstates parity (IN-06) | INFO | A rate-limited detail with no timing pauses 60 s even when the list snapshot knows a later reset. |
| `internal/ui/help.go` | 58-60 | Legend omits "unknown" (IN-07) | INFO | The `?` help does not explain the new word. |
| `internal/github/ratelimit.go`, `checks.go` | | Shared latest-headers slot; detail cost not tracked (WR-03) | WARNING (deferred by the user) | Possible header misclassification when Detail and Fetch overlap; footer "api N" undercounts by at most 5 points per refresh while open. |
| IN-01..IN-05 | | Stale cached summary, dropped rate limit after esc, "loading…" forever, duplicate reviewer query per page, line-scroll only | INFO | Open, not goal-blocking. |

No debt markers and no stubs in changed non-test files.

### Human Verification Required

1. Real-terminal detail view on a large CI matrix (layout, fold, expand, scroll, esc, unknown checks).
2. Real-terminal refresh with the detail open, plus resize from wide to about 30 columns and back.
3. A visible team review request (never seen live).
4. MERGE column against the github.com merge box on real PRs.
5. Glyph widths in the user's font, including the unknown-check glyph.
6. Disposition of WR-05 (open).

Items 1 to 5 were deferred by the user in UAT ("defer UI tests until we have multiple PRs in multiple states to test with"). They are carried forward, not new.

### Gaps Summary

No must-have is FAILED, so `gaps:` is empty.

**WR-05 judgment.** I weighed whether WR-05 blocks the goal and decided it does not, for three reasons. (1) The roadmap truths concern reviewers, the failing check name, merge-readiness, the detail list and layout. WR-05 changes none of these; it changes one faint summary phrase in a failure mode (GraphQL returns data next to an error). (2) The 03-09 truths, as worded, hold: the null-node case shows "read 1 of 2 runs" end to end, and a kept error sets `Truncated`. WR-05 shows the new flag with the wrong sentence in two sub-cases. (3) Before 03-09 these cases showed no signal at all; now they show a signal with imprecise text, so the behavior is no worse than before. It is still a real defect against the honesty principle (D-08), because "read 3 of 3 runs" reads as complete when reviewers may be missing. The owner should decide: either fix it with a separate `Partial` field and its own sentence (a small change to `domain.Detail`, `FetchDetail` and `checksSummary`, with goldens unaffected unless a golden uses `Truncated`), or defer. If the owner makes it gating, change the status to `gaps_found` and run `/gsd-plan-phase --gaps`.

---

_Verified: 2026-10-13_
_Verifier: Claude (gsd-verifier)_

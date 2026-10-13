---
phase: 03-blocker-detail
verified: 2026-10-13T03:00:00Z
status: human_needed
score: 4/4 roadmap truths verified; 3/3 earlier gap-closure truths (WR-01, WR-02, WR-04) verified; 5/5 plan 03-11 (G-03-6 / WR-05) truths verified
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
  - ".planning/phases/03-blocker-detail/03-11-PLAN.md"
  - ".planning/phases/03-blocker-detail/03-11-SUMMARY.md"
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
  - "internal/github/detail_test.go"
  - "internal/ui/cells.go"
  - "internal/ui/detail.go"
  - "internal/ui/detail_test.go"
  - "internal/ui/help.go"
  - "internal/ui/keys.go"
  - "internal/ui/layout.go"
  - "internal/ui/model.go"
  - "internal/ui/refresh.go"
  - "internal/ui/styles.go"
  - "internal/ui/view.go"
  - "internal/ui/view_test.go"
covered_digest: "v3:sha256:11112186bd7327d9983bcd9b513220c6763e460e9a35cb54f02a4d0194d426ed"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: "4/4 roadmap truths; 3/3 gap-closure truths"
  gaps_closed:
    - "G-03-6 / WR-05: a partial detail read is no longer worded as a run count, and 'no checks' is kept"
  gaps_remaining: []
  regressions: []
gaps: []
advisory_warnings:
  - id: WR-03
    summary: "Shared latest-headers slot in captureTransport; detail cost not in the budget. Deferred by the user in UAT (disposition: deferred)."
  - id: IN-01..IN-09
    summary: "Nine info findings remain open in 03-REVIEW-DISPOSITION.md. None is goal-blocking. IN-08 (a kept error on a later page sets Partial although that page's reviewer data is unused) and IN-09 (the notice is untested below its own length; its refresh behaviour is untested) are the two that touch the new 03-11 code."
human_verification:
  - test: "Run `prv` in a real terminal, open the detail (enter) of a PR with a large CI matrix. Check failing checks sit on top, unknown checks stay listed, passing and skipped ones fold into '... +N more passing or skipped', e expands and collapses, j/k scroll, esc returns to the same PR."
    expected: "Layout is readable; the selected PR is unchanged after esc."
    why_human: "Layout and readability in a real terminal cannot be asserted by golden tests. UAT test 1 was skipped by the user (no reachable PR with a large CI matrix; revisit after Phase 4 add-by-URL)."
  - test: "Run against a PR that has a visible team review request (an org account)."
    expected: "The row shows '...@org/slug' and the detail shows '@org/slug  pending'; a hidden team shows as '...team'."
    why_human: "The `... on Team { combinedSlug }` branch has not been observed live. UAT test 3 was skipped (no org PR available)."
  - test: "Compare the MERGE column with the github.com merge box for several real PRs (mergeable, blocked, behind, conflicts, draft)."
    expected: "Each row agrees with GitHub's own merge state."
    why_human: "Agreement with the live merge box needs a person and real PRs of varied state. UAT test 4 was skipped (only one PR, no merge variety)."
---

# Phase 3: Blocker Detail Verification Report

**Phase Goal:** User can see, without leaving the terminal, who and what is holding each PR up: which reviewers, which failing check, and whether it can merge
**Verified:** 2026-10-13
**Status:** human_needed
**Re-verification:** Yes, after gap-closure plan 03-11 (UAT gap G-03-6 / review finding WR-05). The previous report is replaced.

## Goal Achievement

The goal is achieved in the code. G-03-6 is closed in code, tests and goldens. No must-have is FAILED, so `gaps:` is empty. Status is `human_needed` only because three real-world UAT checks (1, 3, 4) were skipped by the user for lack of suitable PRs. They are carried forward, not new, and none of them is a code defect. UAT tests 2 and 5 passed in the user's terminal.

Evidence I ran myself on HEAD (`debd54f`), not taken from SUMMARY.md:

- `go build ./... && go vet ./...` clean. `go test ./... -count=1 -race`: all 5 packages pass in one full run.
- Named tests, run singly with `-v`: `TestDetailKeptErrorEndToEnd` (every_run_read, no_checks), `TestDetailPartialEndToEnd`, `TestDetailEndToEnd`, `TestFailingNameEndToEnd`, `TestFetchDetailPartial` (4 subtests), `TestDetailPartialNotice` (6 subtests), `TestDetailTruncatedSummary`, `TestDetailEmpty`, `TestUnknownAlwaysVisible`, `TestDetailRefreshes`, `TestDetailEscAfterReorder`, `TestServerWait`: all PASS.
- `git diff a2f111f..HEAD` on `internal/ui/testdata`: only 4 files added (`detail-partial_*.golden`), none modified. Every pre-existing golden is unchanged, as plan 03-11 required.
- No `TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER` in `internal/domain/checks.go`, `internal/github/checks.go`, `internal/ui/detail.go`.
- Working tree is clean apart from `.planning/config.json` (pre-existing, not part of this phase).

### G-03-6 / WR-05 closure (code read)

| Item | Where | Finding |
|------|-------|---------|
| New field | `internal/domain/checks.go:285-293` | `Detail.Partial bool` with a doc comment. `Truncated` is documented as "check runs were not read". |
| Producer | `internal/github/checks.go:333-337` | `if partial { d.Partial = true }`. `Truncated` is set only at the three run-not-read sites: a later page failed, page cap or empty cursor, and `Read < Total` after the loop. |
| Renderer, notice | `internal/ui/detail.go:274-276` | `detailNotes` adds faint "some details could not be loaded" under the title when `data.Partial`. |
| Renderer, counts | `internal/ui/detail.go:314-318` | `checksSummary` appends "read N of M runs" only on `Truncated`. With no segments it still returns "no checks". |

### Observable Truths (roadmap contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Each row shows approved / changes-requested / pending reviewers plus merge-readiness | VERIFIED | Unchanged in 03-11 (no edits to `client.go`, `cells.go`, `merge.go`, `reviewers.go`). Full-suite list tests and `list_*` goldens pass. |
| 2 | The row of a PR with failing CI names one of its failing checks (v1 limit: >100 contexts gives the plain badge) | VERIFIED | `TestFailingNameEndToEnd` passes through the real client. |
| 3 | Selecting a PR opens a detail of every check (failing first) and every reviewer; returning keeps the same PR selected, also after a refresh while open | VERIFIED | `TestDetailEndToEnd`, `TestDetailRefreshes`, `TestDetailEscAfterReorder` pass. Detail data is rebuilt fresh by `FetchDetail` on each refresh, so a stale `Partial` cannot persist. |
| 4 | In a narrow pane and after a resize, list and detail drop or truncate columns and stay readable | VERIFIED | Golden suite at 60x15, 80x24, 120x40, 0x0 passes, now including `detail-partial`. The 60x15 golden shows the notice on its own line and the CHECKS line cut at the pane edge, as for every other state. Real-drag feel was confirmed by the user in UAT test 2 (pass). |

**Score:** 4/4 roadmap truths verified; 0 present-behavior-unverified.

### Plan 03-11 truths (G-03-6)

| Truth | Status | Evidence |
|-------|--------|----------|
| Kept GraphQL error next to complete check data shows the notice and never "read N of N runs" (WR-05 case 1) | VERIFIED | `TestDetailKeptErrorEndToEnd/every_run_read` through the real `github.Client` and `ui.Model`: asserts "✗ lint" and the notice, and asserts the view contains no " runs". `TestDetailPartialNotice/every_run_read` and `TestFetchDetailPartial/page_1_kept_a_GraphQL_error` pass. |
| No statusCheckRollup plus a kept error on page 1 reads "CHECKS  no checks" plus the notice, never "read 0 of 0 runs" (WR-05 case 2) | VERIFIED | `TestDetailKeptErrorEndToEnd/no_checks` asserts "CHECKS  no checks" and the notice. In code: a nil rollup leaves `Total == Read == 0`, so `Truncated` stays false and `checksSummary` falls through to "no checks". |
| "read N of M runs" still shows when runs were not read (page cap, failed later page, null nodes) | VERIFIED | `TestDetailPartialEndToEnd` (null node, "read 1 of 2 runs"), `TestFetchDetailPartial/a_null_context_node`, `TestDetailTruncatedSummary`, and `TestDetailPartialNotice/Truncated_only` and `/Truncated_and_Partial`. |
| FetchDetail sets `Partial` (not `Truncated`) on a kept error; `Truncated` only when runs were not read | VERIFIED | Code read above; `TestFetchDetailPartial` asserts both flags per subtest. |
| A complete read with no kept error shows neither text, and existing goldens are unchanged | VERIFIED | `TestDetailPartialNotice/neither` passes; `git diff` shows no modified golden. |

### Earlier gap-closure truths (regression check)

| Truth | Status | Evidence |
|-------|--------|----------|
| WR-01: detail rate-limit pause uses the clamp | VERIFIED | `TestServerWait` passes. `backoff.go` and the pause path are untouched by 03-11. |
| WR-02: partial reads are marked in the data layer | VERIFIED | `TestFetchDetailPartial` passes. The display defect that WR-02 left behind (WR-05) is now fixed. |
| WR-04: unknown checks listed and counted as unknown | VERIFIED | `TestUnknownAlwaysVisible` passes; `add(c.Other, "unknown", ...)` and the `CheckOther` case in `checkLines` are present. |

### Required Artifacts

| Artifact | Status | Details |
|----------|--------|---------|
| `internal/domain/checks.go` (`Detail.Partial`) | VERIFIED | Exists, documented, set by `FetchDetail`, read by `detailNotes`. |
| `internal/github/checks.go` (`d.Partial = true`) | VERIFIED | Present at line 337. |
| `internal/ui/detail.go` (`detailNotes` notice) | VERIFIED | Present at line 275; wired through `detailLines`. |
| `cmd/prv/main_test.go` (`TestDetailKeptErrorEndToEnd`) | VERIFIED | Substantive: real client, routed transport, real model. |
| `internal/ui/detail_test.go` (`TestDetailPartialNotice`) | VERIFIED | Covers all Truncated/Partial combinations plus a compact pane. |
| `detail-partial_{0x0,60x15,80x24,120x40}.golden` | VERIFIED | All four exist and are consumed by `TestGolden`. |

### Key Link Verification

| From | To | Status | Details |
|------|----|--------|---------|
| `FetchDetail` | `detailNotes` | WIRED | `Detail.Partial` is set, then rendered; proven end to end. |
| `FetchDetail` | `checksSummary` | WIRED | `Detail.Truncated` set only for runs not read. |
| Earlier links (Fetch to fetchFailing, enter to openDetail, fetchedMsg to refreshDetail, detailMsg to applyDetail, pause to ServerWait) | | WIRED | Unchanged; end-to-end tests pass. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `detailNotes` notice | `detail.data.Partial` | `detailPage` partial flag from a real kept `api.GraphQLError` | Yes | FLOWING |
| `checksSummary` | `Truncated`, `Read`, `Total` | `FetchDetail` page loop | Yes | FLOWING |
| `ciCell`, `reviewersCell`, detail checks and reviewers | `pr.Failing`, `pr.Reviewers`, `detail.data` | GraphQL queries | Yes | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Kept error, both WR-05 cases, end to end | `go test ./cmd/prv -run TestDetailKeptErrorEndToEnd -v` | PASS (2 subtests) | PASS |
| Data layer sets Partial vs Truncated | `go test ./internal/github -run TestFetchDetailPartial -v` | PASS (4 subtests) | PASS |
| Every flag combination rendered | `go test ./internal/ui -run TestDetailPartialNotice -v` | PASS (6 subtests) | PASS |
| Whole workspace | `go test ./... -count=1 -race` | all 5 packages ok | PASS |

### Probe Execution

Step 7c: SKIPPED. The phase declares no probe scripts.

### Requirements Coverage

| Requirement | Source plans | Description | Status | Evidence |
|-------------|--------------|-------------|--------|----------|
| LIST-04 | 03-04, 03-05 | Reviewers approved / changes requested / pending in the row | SATISFIED | Truth 1; real team request is a human item (UAT 3 skipped) |
| LIST-06 | 03-04, 03-05 | Merge-readiness in the row | SATISFIED | Truth 1; comparison with the live merge box is a human item (UAT 4 skipped) |
| LIST-07 | 03-01, 03-02 | Failing check name in the row | SATISFIED | Truth 2 (accepted >100-context limit) |
| DETL-01 | 03-06, 03-07, 03-09, 03-10, 03-11 | Detail lists every check, failing first | SATISFIED | Truth 3; unknown checks listed; partial reads now honestly worded |
| DETL-02 | 03-06, 03-07, 03-09, 03-11 | Detail lists each reviewer and state | SATISFIED | Truth 3; a possibly incomplete reviewer list now carries the notice |
| DETL-03 | 03-07, 03-08 | Return keeps the same PR selected | SATISFIED | Truth 3 |
| NAV-05 | 03-03, 03-05, 03-08 | Layout adapts to resizes and narrow panes | SATISFIED | Truth 4; UAT 2 passed |

All 7 IDs appear in at least one PLAN `requirements:` field and in REQUIREMENTS.md (checked, marked Complete). No orphaned Phase 3 requirements.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/github/checks.go` | 333-337 | A kept error on a later page sets `Partial` although that page's reviewer data is never used (IN-08) | INFO | May show the notice when reviewers are in fact complete. Errs toward saying less than certain; not misleading. |
| `internal/ui/detail.go` | 274-276 | Notice not tested below its own length; refresh behaviour untested (IN-09) | INFO | The 60x15 golden and the `compact_pane` subtest cover narrow panes; sub-30-column widths are not pinned. |
| `internal/github/ratelimit.go` and others | | Shared latest-headers slot; detail cost not tracked (WR-03) | WARNING (deferred by the user) | Unchanged and out of scope for this phase. |
| IN-01..IN-07 | | Open info findings | INFO | Not goal-blocking. |

No debt markers and no stubs in the changed non-test files.

### Human Verification Required

1. Real-terminal detail view on a large CI matrix (UAT 1, skipped; revisit after Phase 4 add-by-URL).
2. A visible team review request (UAT 3, skipped).
3. MERGE column against the github.com merge box on real PRs (UAT 4, skipped).

UAT 2 (refresh and resize) and UAT 5 (glyph widths, including the unknown-check glyph) passed in the previous round and are not re-listed. The UAT item for the WR-05 disposition (UAT 6) is resolved: the user chose "fix before ship", 03-11 fixed it, and `03-REVIEW-DISPOSITION.md` records WR-05 as `fixed`.

### Gaps Summary

No gaps. G-03-6 is closed: the two wrong outputs the user flagged ("read 3 of 3 runs" and "read 0 of 0 runs") cannot occur, because a kept GraphQL error now sets a separate flag with its own faint sentence, and "read N of M runs" is tied to runs that were not read. This is proven through the real client and model, at the unit level for every flag combination, and in four new goldens, with no change to any existing golden. The remaining status is `human_needed` only because the user chose to defer three live-data checks until suitable PRs exist.

---

_Verified: 2026-10-13_
_Verifier: Claude (gsd-verifier)_

---
phase: 03-blocker-detail
verified: 2026-10-12T00:00:00Z
status: human_needed
score: 4/4 roadmap success criteria verified (code and tests); 4 review warnings and 6 human items outstanding
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
  - "cmd/prv/main_test.go"
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
  - "internal/ui/view.go"
covered_digest: "v3:sha256:bea384c8e903feebd5c7c3a01b77e2a1ebdc106a592e0f6d97c18b265d8614a3"
behavior_unverified: 0
overrides_applied: 0
gaps: []
review_warnings:
  - id: WR-01
    file: "internal/ui/detail.go:148-164"
    summary: "pauseForRateLimit does not clamp the server wait (domain.MaxServerWait = 65 min) that every list pause uses; Retry-After 2147483647 parks all refreshes for years. Confirmed in code."
  - id: WR-02
    file: "internal/github/checks.go (FetchDetail, fetchFailing)"
    summary: "A null context/review node from partial GraphQL data is skipped without Read++ or Truncated=true, so the detail summary shows exact-looking counts with no 'read N of M runs'. fetchFailing can name a check from a page that has a hole. Confirmed in code."
  - id: WR-03
    file: "internal/github/ratelimit.go, internal/github/checks.go:284"
    summary: "captureTransport keeps one 'latest headers' slot; FetchDetail now overlaps Fetch, so classify() can read the other request's headers. Mutex-protected (no data race), but wrong-headers misclassification is possible. Detail requests are not added to snap.RateLimit (footer 'api N' undercounts). Confirmed in code."
  - id: WR-04
    file: "internal/ui/detail.go:308,329-336; internal/domain/checks.go CountChecks"
    summary: "CheckOther (unrecognised status) is folded into '... +N more passing' and counted as 'skipped' in the default detail view. Confirmed in code and in detail_80x24.golden ('+12 more passing' = 10 passing + 2 skipped)."
human_verification:
  - test: "Run `prv` in a real terminal, open the detail (enter) of a PR with a large CI matrix. Check failing checks sit on top, passing ones fold into '... +N more passing', e expands and collapses, j/k scroll, esc returns to the same PR."
    expected: "Layout is readable; the selected PR is unchanged after esc."
    why_human: "Layout and readability in a real terminal cannot be asserted by golden tests (03-07 D7)."
  - test: "Leave the detail open across a refresh (press r), then drag the pane narrow (about 30 columns) and wide again."
    expected: "The 'updated' time and the checks refresh; the body re-lays out; the footer stays on the last line; esc returns to the same PR. The list drops columns in the D-13 order and compact rows appear below 40 columns."
    why_human: "Resize and refresh feel in a real terminal (03-03 D-human, 03-08 D8)."
  - test: "Run against a PR that has a visible team review request (an org account)."
    expected: "The row shows '...@org/slug' and the detail shows '@org/slug  pending'; a hidden team shows as '...team'."
    why_human: "The `... on Team { combinedSlug }` branch could not be observed live: the viewer is in no org (03-04, RESEARCH A1)."
  - test: "Compare the MERGE column with the github.com merge box for several real PRs (mergeable, blocked, behind, conflicts, draft)."
    expected: "Each row agrees with GitHub's own merge state."
    why_human: "Agreement with the live merge box needs a person and real PRs (03-05 D1)."
  - test: "Check the glyph widths of the check, merge, reviewer and compact-row glyphs in the user's font (the check, cancelled, behind, blocked and dismissed glyphs)."
    expected: "Every glyph is one cell wide; no column shifts."
    why_human: "Font-dependent width, RESEARCH A5 (03-05 D1)."
  - test: "Decide the disposition of WR-01..WR-04 in 03-REVIEW-DISPOSITION.md (all still `open`): fix before ship, defer, or skip."
    expected: "Each warning has a recorded disposition."
    why_human: "These are accepted-risk decisions on rate-limit robustness and on whether an unknown check may read as 'passing'; they are the owner's call."
---

# Phase 3: Blocker Detail Verification Report

**Phase Goal:** User can see, without leaving the terminal, who and what is holding each PR up: which reviewers, which failing check, and whether it can merge
**Verified:** 2026-10-12
**Status:** human_needed
**Re-verification:** No, initial verification

## Goal Achievement

The goal is achieved in the code. A user sees reviewers, merge-readiness and the failing check name in the list, and a detail view of every check and reviewer. Every roadmap success criterion has code, wiring and test evidence. Status is `human_needed`, not `passed`, for two reasons: the plans deferred real-terminal checks to this step, and four open review warnings touch rate-limit and "unknown is never passing" behaviour. None of the warnings makes a must-have fail as the plan worded it, so I did not mark a gap. If the owner wants WR-01, WR-02 and WR-04 gating, route them to `/gsd-plan-phase --gaps`.

Evidence I ran myself (not taken from SUMMARY.md):

- `go build ./... && go vet ./...` clean. `go test ./... -count=1 -race` passes in all 5 packages (the full suite ran once).
- `TestFailingNameEndToEnd` and `TestDetailEndToEnd` re-run singly with `-v`: PASS. These go through the real `github.Client` against a fake HTTP server.
- No `TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER` in non-test files changed since `034f67d`. No GraphQL `mutation` in `internal/github`.
- **Live check:** I ran the exact `detailQuery` and `failingQuery` text from `internal/github/checks.go` against github.com with `gh api graphql`. Both are accepted with no errors. On `cli/cli` PR_kwDODKw3uc8AAAABIEByQw the Detail query returned 18 contexts with `checkSuite.workflowRun.file.path`, `app.name`, `databaseId` and a rate-limit `cost` of 1, matching the wire structs. This settles the 03-06 D5 "this exact query text has not been run" item.

### Observable Truths (roadmap contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Each row shows approved / changes-requested / pending reviewers plus merge-readiness (mergeable, blocked, behind, conflicts) | VERIFIED | `list_120x40.golden`: `✓ approved  ✗erin …frank  ✗ conflicts`, `↓ behind`, `–` for draft, blank for merged. `DeriveMerge` (state, then draft, then status, then mergeable) and `MergeReviewers` (pending wins, hidden teams kept, bot suffix stripped) read in full. Both are filled in `client.go:342-345` for every PR. At 60 cols REVIEWERS drops by design (D-13); below 40 cells the compact row keeps the review and merge glyphs. |
| 2 | The row of a PR with failing CI names one of its failing checks (v1 limit: >100 contexts gives the plain badge) | VERIFIED | `fetchFailing` (chunks of 25, only for `CIFailing` PRs) feeds `SummarizeFailing(LatestChecks(...), complete)`. `complete` is false on `hasNextPage`, so no name comes from a truncated page. `ciCell` gives `✗ NAME (cancelled) +N`, capped at 24 cells. `TestFailingRerunFixture`, `TestFailingTruncatedUnnamed`, `TestFailingChunks`, `TestFailingNameEndToEnd` pass. Edge noted under WR-02. |
| 3 | Selecting a PR opens a detail of every check (failing first) and every reviewer; returning keeps the same PR selected, also after a refresh while open | VERIFIED | `detail.go` + `model.go:257` (enter) / `closeDetail` (selection is never touched). `refreshDetail` re-finds the PR by ref and re-selects it; `gone` keeps last data. Tests: `TestDetailOpenClose`, `TestDetailRefreshes`, `TestDetailEscAfterReorder`, `TestDetailGone`, `TestDetailStaleDropped`, `TestDetailStaleAfterSwitch`, `TestDetailEndToEnd`. `detail_80x24.golden` shows failing, cancelled, running, then the fold, then reviewers in changes-requested, pending, approved, commented, dismissed order. Passing checks are listed only after `e` (D-10, by design). Caveat: WR-04. |
| 4 | In a narrow pane and after a resize, list and detail drop or truncate columns and stay readable | VERIFIED | `layoutColumns` with the D-13 drop order; compact rows below 40 cells (`TestGoldenCompact.golden`); `TestNeverOverflows` and `TestNeverOverflowsDetail` sweep widths 1..200 and heights 1..50; `TestResizeKeepsSelection`, `TestDetailResize`, `TestDetailTinyHeights`. Goldens exist for list, empty, error, loading, filtered, prompt and all detail states at 60x15, 80x24, 120x40 and 0x0. How it feels on a real drag is a human item. |

**Score:** 4/4 roadmap truths verified; 0 present-behavior-unverified.

### Plan-level truths and prohibitions (spot-checked against code)

| Item | Status | Evidence |
|------|--------|----------|
| D-03 re-runs not double counted; counts never from GitHub per-state counts | VERIFIED | `LatestChecks` keys on (context, workflow, name), newest suite then ID; `CountChecks` runs on the de-duplicated list; the queries request no count fields. |
| D-01 budget: no FailingChecks request when nothing fails; ≤25 ids per request | VERIFIED | `fetchFailing` loops over failing PRs only, chunk 25. `TestFailingChunks`. |
| 03-08 prohibition: detail request only while open, one in flight, none during a pause or after `gone` | VERIFIED | `startDetailFetch` guards `detailInFlight`, `gone`, `pausedUntil`; `closeDetail` bumps the seq. `TestDetailNoFetchWhenClosed`, `TestDetailNoOverlap`, `TestDetailPausedNoRequest`. |
| 03-08 prohibition: stale/other-ref result never drawn; `o` opens the detail PR | VERIFIED | `applyDetail` checks `seq` and `ref`; `detailKey` opens `m.detail.pr.URL`. |
| 03-04 prohibition: never look more mergeable than the data | VERIFIED | `DeriveMerge`: non-open gives none, draft gives dash, unknown/empty gives dash. |
| 03-01/02 prohibition: no name from a truncated page, failed query or guess | VERIFIED, one edge | Truncated and failed paths give the plain badge. Edge: a null context node inside a complete page is dropped silently (WR-02). |
| 03-06 prohibition: no inflated counts; early paging disclosed | VERIFIED, one edge | Paging cap and a failed later page set `Truncated` and Read/Total. Edge: nulled nodes are not disclosed (WR-02). |
| Rate-limit pause in detail extends to reset / Retry-After | VERIFIED as worded; defect WR-01 | The pause is applied and respected by tick, `r` and detail fetch. It is unclamped. |
| Unknown check never shown as passing | UNCERTAIN (WARNING, WR-04) | `ClassifyCheck` is total and returns `CheckOther`. The detail fold and summary then label it "more passing" / "skipped". The plan's D-10 text itself folds "passing, skipped and unknown". The ClassifyCheck contract and the display disagree. |

### Required Artifacts

| Artifact | Status | Details |
|----------|--------|---------|
| `internal/domain/checks.go` | VERIFIED | Substantive and wired: classify, de-dup, summarize, sort, count, `Detail`. |
| `internal/domain/merge.go`, `reviewers.go`, `status.go` | VERIFIED | Wired through `client.go` and `cells.go`. |
| `internal/github/checks.go` | VERIFIED | `fetchFailing` called from `Fetch` (`client.go:284`); `FetchDetail` reached through `DetailFetcher`. |
| `internal/ui/cells.go`, `layout.go`, `detail.go`, `keys.go`, `help.go` | VERIFIED | Used by `view.go` / `model.go`. |
| `cmd/prv/main_test.go` | VERIFIED | End-to-end tests through the real client pass. |
| Roadmap goldens (60x15, 80x24, 120x40, 0x0) | VERIFIED | All present under `internal/ui/testdata/TestGolden/`. |

### Key Link Verification

| From | To | Status | Details |
|------|----|--------|---------|
| `Client.Fetch` | `fetchFailing` | WIRED | `client.go:284` |
| `fetchFailing` | `PR.Failing` | WIRED | `SummarizeFailing(LatestChecks(...))` |
| `PR.Failing` | `ciCell` | WIRED | `cells.go:27-46` |
| `mapResponse` | `PR.Merge`, `PR.Reviewers` | WIRED | `client.go:342,345` |
| enter | `openDetail` → `FetchDetail` | WIRED | `model.go:257`; `fetchDetail` Cmd |
| `fetchedMsg` success | `refreshDetail` | WIRED | `model.go:340-342` |
| `detailMsg` | `applyDetail` | WIRED | `model.go:301-302` |

### Data-Flow Trace (Level 4)

| Artifact | Variable | Source | Real data | Status |
|----------|----------|--------|-----------|--------|
| `ciCell` | `pr.Failing` | `nodes(ids:)` FailingChecks query, live-validated | Yes | FLOWING |
| `reviewersCell` | `pr.Reviewers` | discovery `latestOpinionatedReviews` / `reviewRequests` | Yes | FLOWING |
| detail checks / reviewers | `detail.data` | `node(id:)` Detail query, live-validated | Yes | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Failing name through the real client | `go test ./cmd/prv -run TestFailingNameEndToEnd -v` | PASS | PASS |
| Detail open / refresh / esc through the real client | `go test ./cmd/prv -run TestDetailEndToEnd -v` | PASS | PASS |
| Detail query accepted by GitHub | `gh api graphql` with the exact `detailQuery` text on a real PR | no `errors`, cost 1, shapes match | PASS |
| FailingChecks query accepted by GitHub | `gh api graphql` with the exact `failingQuery` text | no `errors`, cost 1 | PASS (the one PR tried had a null rollup, so the contexts branch of this query was not seen live; the Detail query has the same contexts selection and returned data) |

### Probe Execution

Step 7c: SKIPPED. The phase declares no probe scripts.

### Requirements Coverage

| Requirement | Source plans | Description | Status | Evidence |
|-------------|--------------|-------------|--------|----------|
| LIST-04 | 03-04, 03-05 | Reviewers who approved / requested changes / pending | SATISFIED | Truth 1; real-team case is a human item |
| LIST-06 | 03-04, 03-05 | Merge-readiness in the row | SATISFIED | Truth 1 |
| LIST-07 | 03-01, 03-02 | Failing check name in the row | SATISFIED | Truth 2 (accepted >100-context limit) |
| DETL-01 | 03-06, 03-07 | Detail lists every check, failing first | SATISFIED | Truth 3; WR-04 caveat |
| DETL-02 | 03-06, 03-07 | Detail lists each reviewer and state | SATISFIED | Truth 3 |
| DETL-03 | 03-07, 03-08 | Return keeps the same PR selected | SATISFIED | Truth 3 |
| NAV-05 | 03-03, 03-05, 03-08 | Layout adapts to resize / narrow panes | SATISFIED | Truth 4 |

All 7 IDs from ROADMAP.md appear in at least one PLAN `requirements:` field and in REQUIREMENTS.md (marked Complete). No orphaned Phase 3 requirements.

### Anti-Patterns Found

No debt markers and no stubs in the changed non-test files. The code-review defects are listed below.

| File | Pattern | Severity | Impact |
|------|---------|----------|--------|
| `internal/ui/detail.go:148-164` | Unclamped server wait (WR-01) | WARNING | A huge `Retry-After` or far-future reset on a rate-limited detail response pauses all refreshes until restart. Fails safe on API budget, fails unsafe on availability. |
| `internal/github/checks.go` `FetchDetail` / `fetchFailing` | Null nodes dropped silently (WR-02) | WARNING | Undercount with no "read N of M" note on partial GraphQL data; a failing-check name can come from a page with a hole. |
| `internal/github/ratelimit.go`, `checks.go:284` | Shared latest-headers slot; detail cost not tracked (WR-03) | WARNING | Possible misclassification when Detail and Fetch overlap; footer "api N" and the reserve check undercount detail requests (at most 5 points per refresh while open). |
| `internal/ui/detail.go:308,329-336` | Unknown folded as "passing", counted "skipped" (WR-04) | WARNING | Breaks the "unknown is never passing" principle in the default detail view. |
| IN-01..IN-05 | Stale cached summary, dropped rate-limit after esc, "loading…" forever, duplicate reviewer query per page, line-scroll only | INFO | Not goal-blocking. |

### Human Verification Required

See the `human_verification` list in the frontmatter. In short:

1. Real-terminal run of the detail view on a large CI matrix (layout, fold, expand, scroll, esc).
2. Real-terminal refresh while the detail is open, plus resize drag from wide to about 30 columns and back.
3. A visible team review request (`... on Team { combinedSlug }` never seen live).
4. MERGE column against the github.com merge box on real PRs.
5. Glyph widths in the user's font.
6. Disposition of WR-01..WR-04 (all `open`).

### Gaps Summary

No must-have is FAILED, so there is no `gaps:` entry. The warnings are real, confirmed defects in robustness, not misses of the stated truths. My recommendation, as a judgment call for the owner: fix WR-01 (reuse the domain clamp), WR-02 (mark `Truncated` when `Read < Total`) and WR-04 (list or relabel `CheckOther`) before shipping, since each is small and each touches a stated principle (bounded pauses, honest counts, unknown is not passing). WR-03 is larger and can be deferred, given that the mutex prevents a data race and detail cost is at most 5 points per refresh. If any of the first three is made gating, change the status to `gaps_found` and run `/gsd-plan-phase --gaps`.

---

_Verified: 2026-10-12_
_Verifier: Claude (gsd-verifier)_

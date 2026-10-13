---
status: complete
phase: 03-blocker-detail
source: [03-VERIFICATION.md]
started: 2026-10-13T01:48:49Z
updated: 2026-10-13T02:35:15.561Z
---

## Current Test

[testing complete]

## Tests

### 1. Run `prv` in a real terminal, open the detail (enter) of a PR with a large CI matrix. Check failing checks sit on top, unknown checks stay listed, passing and skipped ones fold into '... +N more passing or skipped', e expands and collapses, j/k scroll, esc returns to the same PR.
expected: Layout is readable; the selected PR is unchanged after esc.
result: skipped
reason: "Deferred follow-up: No PR with a large CI matrix is reachable yet (user's only open PR has 0 checks; add-by-URL is Phase 4). Revisit after Phase 4 with e.g. prometheus/prometheus#19994."

### 2. Leave the detail open across a refresh (press r), then drag the pane narrow (about 30 columns) and wide again.
expected: The 'updated' time and the checks refresh; the body re-lays out; the footer stays on the last line; esc returns to the same PR. The list drops columns in the D-13 order and compact rows appear below 40 columns.
result: pass

### 3. Run against a PR that has a visible team review request (an org account).
expected: The row shows '...@org/slug' and the detail shows '@org/slug  pending'; a hidden team shows as '...team'.
result: skipped
reason: "I don't have an org PR with team reviewers"

### 4. Compare the MERGE column with the github.com merge box for several real PRs (mergeable, blocked, behind, conflicts, draft).
expected: Each row agrees with GitHub's own merge state.
result: skipped
reason: "only one PR and it has no merge variety"

### 5. Check the glyph widths of the check, merge, reviewer and compact-row glyphs (including the new unknown-check glyph) in the user's font.
expected: Every glyph is one cell wide; no column shifts.
result: pass

### 6. Decide the disposition of WR-05 in 03-REVIEW-DISPOSITION.md (open): fix before ship, or defer.
expected: WR-05 has a recorded disposition.
result: issue
reported: "fix before ship"
severity: major

## Summary

total: 6
passed: 2
issues: 1
pending: 0
skipped: 3
blocked: 0

## Gaps

- gap_id: G-03-6
  truth: "The detail view reports a partial read accurately: 'read N of M runs' only when runs are missing, a distinct notice when a GraphQL error was kept next to the data, and 'no checks' when the PR has no checks (WR-05)"
  status: resolved
  resolved_by: 03-11-PLAN.md
  resolved_at: 2026-10-12
  reason: "User reported: fix before ship"
  severity: major
  test: 6
  root_cause: "domain.Detail.Truncated carries two meanings: FetchDetail sets it both when fewer check runs were read than exist and when GitHub kept a GraphQL error next to the data (partial). The only renderer, checksSummary, turns any Truncated into 'read Read of Total runs', so a partial reviewer error shows 'read 3 of 3 runs' (looks complete), and a PR with no statusCheckRollup plus a partial error shows 'read 0 of 0 runs' instead of 'no checks'."
  artifacts:
    - path: "internal/github/checks.go"
      issue: "FetchDetail (lines ~331-335) sets d.Truncated = true on a partial page instead of a separate flag"
    - path: "internal/domain/checks.go"
      issue: "Detail (line ~285) has no field for 'an error was kept next to the data'"
    - path: "internal/ui/detail.go"
      issue: "checksSummary (lines 308-313) renders 'read N of M runs' for every Truncated and skips the 'no checks' branch"
  missing:
    - "Add Detail.Partial bool; set it (not Truncated) when detailPage reports partial"
    - "Show 'read N of M runs' only when runs were actually not read (Read < Total or paging cut)"
    - "Show a distinct faint 'some details could not be loaded' notice when Partial is set, not worded as a run count"
    - "Keep 'no checks' when there are no checks and nothing was cut, even if Partial is set"
    - "Rendered-output tests (not just struct fields) for both WR-05 cases: partial reviewer error with Read == Total, and nil rollup + partial on page 1"
  debug_session: ".planning/phases/03-blocker-detail/03-REVIEW.md#wr-05"

## Deferred Follow-Ups

- test: 1
  idea: "Verify the large-CI-matrix detail view once Phase 4 add-by-URL exists (or with a fixture PR): failing on top, unknown listed, passing/skipped folded, e expands, j/k scroll, esc keeps selection."
  deferred_at: 2026-10-12

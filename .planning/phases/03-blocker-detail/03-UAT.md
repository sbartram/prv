---
status: testing
phase: 03-blocker-detail
source: [03-VERIFICATION.md]
started: 2026-10-13T00:03:11Z
updated: 2026-10-13T00:03:11Z
---

## Current Test

number: 1
name: Run `prv` in a real terminal, open the detail (enter) of a PR with a large CI matrix. Check failing checks sit on top, passing ones fold into '... +N more passing', e expands and collapses, j/k scroll, esc returns to the same PR.
expected: |
  Layout is readable; the selected PR is unchanged after esc.
awaiting: user response

## Tests

### 1. Run `prv` in a real terminal, open the detail (enter) of a PR with a large CI matrix. Check failing checks sit on top, passing ones fold into '... +N more passing', e expands and collapses, j/k scroll, esc returns to the same PR.
expected: Layout is readable; the selected PR is unchanged after esc.
result: [pending]

### 2. Leave the detail open across a refresh (press r), then drag the pane narrow (about 30 columns) and wide again.
expected: The 'updated' time and the checks refresh; the body re-lays out; the footer stays on the last line; esc returns to the same PR. The list drops columns in the D-13 order and compact rows appear below 40 columns.
result: [pending]

### 3. Run against a PR that has a visible team review request (an org account).
expected: The row shows '...@org/slug' and the detail shows '@org/slug  pending'; a hidden team shows as '...team'.
result: [pending]

### 4. Compare the MERGE column with the github.com merge box for several real PRs (mergeable, blocked, behind, conflicts, draft).
expected: Each row agrees with GitHub's own merge state.
result: [pending]

### 5. Check the glyph widths of the check, merge, reviewer and compact-row glyphs in the user's font (the check, cancelled, behind, blocked and dismissed glyphs).
expected: Every glyph is one cell wide; no column shifts.
result: [pending]

### 6. Decide the disposition of WR-01..WR-04 in 03-REVIEW-DISPOSITION.md (all still `open`): fix before ship, defer, or skip.
expected: Each warning has a recorded disposition.
result: [pending]

## Summary

total: 6
passed: 0
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps

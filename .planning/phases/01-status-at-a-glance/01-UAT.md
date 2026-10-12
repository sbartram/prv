---
status: testing
phase: 01-status-at-a-glance
source: [01-VERIFICATION.md]
started: 2026-10-12T01:30:00Z
updated: 2026-10-12T01:30:00Z
---

## Current Test

number: 1
name: Assigned PRs appear (live assignee:@me)
expected: |
  A PR assigned to the viewer but authored by someone else appears once, with the author's login in AUTHOR
awaiting: user response

## Tests

### 1. Assigned PRs appear (live assignee:@me)
Run prv on an account that has PRs assigned to it, or add an assigned PR to the test account, with a wide window (for example a hidden --closed-days 365 run).
expected: A PR assigned to the viewer but authored by someone else appears once, with the author's login in AUTHOR
result: [pending]

### 2. Badge fidelity against github.com
Compare several rows with github.com: one with approvals but no required-review rule, one with no checks, one with failing and running checks.
expected: REVIEW and CI badges match github.com; a PR without checks never shows passing; an approved PR on a repo without required reviews shows approved, never required
result: [pending]

### 3. Themes and narrow pane
Open prv in a light-theme and a dark-theme terminal, and in a narrow multiplexer pane (about 40 columns).
expected: ANSI-16 badge colours stay readable on both themes; the selected row is visible; the footer and list stay usable when narrow; terminal restored after q and ctrl+c
result: [pending]

### 4. Real 401 flow
Start prv with an invalid GH_TOKEN (for example GH_TOKEN=bad prv).
expected: The error panel shows the 401 hint (run `gh auth login`, fix or unset GH_TOKEN/GITHUB_TOKEN); r retries; q exits 0; the token string never appears
result: [pending]

### 5. Ambiguous-width glyph alignment (RESEARCH A7)
View the badge glyphs (✓ ✗ ○ ● – ?) in a terminal with a CJK locale or ambiguous-width setting.
expected: Columns stay aligned
result: [pending]

## Summary

total: 5
passed: 0
issues: 0
pending: 5
skipped: 0
blocked: 0

## Gaps

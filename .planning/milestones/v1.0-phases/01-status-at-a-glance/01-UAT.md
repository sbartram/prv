---
status: complete
phase: 01-status-at-a-glance
source: [01-VERIFICATION.md]
started: 2026-10-12T01:30:00Z
updated: 2026-10-12T01:44:33.883Z
---

## Current Test

[testing complete]

## Tests

### 1. Assigned PRs appear (live assignee:@me)
Run prv on an account that has PRs assigned to it, or add an assigned PR to the test account, with a wide window (for example a hidden --closed-days 365 run).
expected: A PR assigned to the viewer but authored by someone else appears once, with the author's login in AUTHOR
result: skipped
reason: "Deferred follow-up: no PRs assigned to this account; re-check live assignee:@me once a PR by another author is assigned"

### 2. Badge fidelity against github.com
Compare several rows with github.com: one with approvals but no required-review rule, one with no checks, one with failing and running checks.
expected: REVIEW and CI badges match github.com; a PR without checks never shows passing; an approved PR on a repo without required reviews shows approved, never required
result: skipped
reason: "Deferred follow-up: no open PRs to compare; re-check badges vs github.com once there are PRs with approvals and CI checks"

### 3. Themes and narrow pane
Open prv in a light-theme and a dark-theme terminal, and in a narrow multiplexer pane (about 40 columns).
expected: ANSI-16 badge colours stay readable on both themes; the selected row is visible; the footer and list stay usable when narrow; terminal restored after q and ctrl+c
result: pass

### 4. Real 401 flow
Start prv with an invalid GH_TOKEN (for example GH_TOKEN=bad prv).
expected: The error panel shows the 401 hint (run `gh auth login`, fix or unset GH_TOKEN/GITHUB_TOKEN); r retries; q exits 0; the token string never appears
result: skipped
reason: "Deferred follow-up: real 401 flow not exercised yet; re-check with GH_TOKEN=bad prv"

### 5. Ambiguous-width glyph alignment (RESEARCH A7)
View the badge glyphs (✓ ✗ ○ ● – ?) in a terminal with a CJK locale or ambiguous-width setting.
expected: Columns stay aligned
result: skipped
reason: "Deferred follow-up: no CJK/ambiguous-width terminal available; re-check glyph alignment (RESEARCH A7)"

## Summary

total: 5
passed: 1
issues: 0
pending: 0
skipped: 4
blocked: 0

## Gaps

## Deferred Follow-Ups

- test: 1
  idea: "no PRs assigned to this account; re-check live assignee:@me once a PR by another author is assigned"
  deferred_at: 2026-10-12
- test: 2
  idea: "no open PRs to compare; re-check badges vs github.com once there are PRs with approvals and CI checks"
  deferred_at: 2026-10-12
- test: 4
  idea: "real 401 flow not exercised yet; re-check with GH_TOKEN=bad prv"
  deferred_at: 2026-10-12
- test: 5
  idea: "no CJK/ambiguous-width terminal available; re-check glyph alignment (RESEARCH A7)"
  deferred_at: 2026-10-12

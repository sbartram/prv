---
status: testing
phase: 04-curated-watch-list
source: [04-VERIFICATION.md]
started: 2026-10-14T00:10:30Z
updated: 2026-10-14T00:10:30Z
---

## Current Test

number: 1
name: Glyph width and + marker in a real terminal and multiplexer pane (~80 and ~40 columns)
expected: |
  The warning glyph occupies one cell, columns stay aligned, '+' sits before the ref, '+ repo#n ⚠' fits below 40 columns, and the header 'N unavailable' stays readable. No row wraps or shifts.
awaiting: user response

## Tests

### 1. Glyph width and + marker in a real terminal and multiplexer pane (~80 and ~40 columns)
Run prv with at least one saved PR and one unavailable saved PR (e.g. a typo'd number added by editing watchlist.json).
expected: The warning glyph occupies one cell, columns stay aligned, '+' sits before the ref, '+ repo#n ⚠' fits below 40 columns, and the header 'N unavailable' stays readable. No row wraps or shifts.
result: [pending]

### 2. SAML SSO reason on a real SSO-enforcing org
With a token not yet authorized for the org, add a PR from it by URL; also save it, restart, and press enter on the unavailable row.
expected: The prompt line and the row both say 'SSO: authorize your token for ORG', with the https://github.com/orgs/ORG/sso / gh auth refresh hint. (If GitHub's real error shape differs, it reads 'could not load (forbidden)' — record that as an issue.)
result: [pending]

### 3. Live smoke on github.com: add, restart, close, remove, second pane
Press a, paste a PR URL from a repo you neither authored nor are assigned to, quit, restart, check it stays after close/merge, then d then y. Keep a second prv pane open.
expected: Row appears with full status and '+', survives restart, stays after close, is removed by d,y; the second pane picks up the add/remove on its next refresh.
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps

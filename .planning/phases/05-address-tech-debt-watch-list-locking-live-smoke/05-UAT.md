---
status: testing
phase: 05-address-tech-debt-watch-list-locking-live-smoke
source: [05-CONTEXT.md D-16, 01-UAT.md, 02-UAT.md, 03-UAT.md]
started: 2026-10-14T03:40:00Z
updated: 2026-10-14T03:40:00Z
---

## Current Test

number: 1
name: Failure marker and recovery at 80 and 40 columns
expected: |
  The header shows the amber 'refresh failed HH:MM (network) · data from HH:MM:SS' marker and the rows stay. After recovery the header shows 'updated HH:MM:SS' again. At 40 columns the footer keeps its notes and the api segment readable. (02-UAT #1)
awaiting: user response

## Tests

### 1. Failure marker and recovery at 80 and 40 columns
steps: Run prv in a real terminal at 80 columns, then at 40. Turn Wi-Fi off, wait for one refresh, then turn it on again.
expected: The header shows the amber 'refresh failed HH:MM (network) · data from HH:MM:SS' marker and the rows stay. After recovery the header shows 'updated HH:MM:SS' again. At 40 columns the footer keeps its notes and the api segment readable. (02-UAT #1)
result: [pending]

### 2. api footer and low-budget warning are legible
steps: From the prv source directory, run `PRV_SHOW=1 go test ./internal/ui -run '^TestShowRateWarning$' -count=1 -v` in your usual terminal, then again inside a tmux or herdr pane. It prints prv's real list screen with a low API budget (999 of 5000, below 20% of the hourly limit) at 80 and then 40 columns. Look at the yellow 'api 999/5000 · resets 15:02' warning in the header and the faint 'api 999' at the right end of the last line.
expected: The api number is legible but quiet; the warning reads as a warning in your usual terminal and inside the tmux/herdr pane. (02-UAT #2; the other theme comparison stays deferred per D-17)
result: [pending]

### 3. Filter keys, / prompt and paste
steps: Press 1, 2 and 3 in turn. Type / and then digits. Paste digits, then text with letters, into the prompt. Press esc, then try enter.
expected: The '[open merged closed]' indicator toggles. The prompt accepts only digits, also from a paste. The list narrows, the hidden count is right, and esc clears the number. (02-UAT #4)
result: [pending]

### 4. o opens the PR in a real browser
steps: Select a PR and press o. Quit, start prv again with BROWSER set to a command that does not exist (for example `BROWSER=nope prv`), and press o.
expected: The PR opens on github.com in your browser. With the bad launcher the footer flashes 'could not open the browser' and the screen is not corrupted. (02-UAT #5)
result: [pending]

### 5. ? help overlay
steps: Press ? in a real terminal, once with a config warning (see test 6) and once without, and once in a short pane.
expected: The overlay shows the keys, the badge legend, the settings in effect, the api budget and the warnings. Any key returns to the list; q does not quit. (02-UAT #6)
result: [pending]

### 6. config.toml edits reach prv
steps: Edit ~/.config/prv/config.toml and restart prv after each edit: add a misspelt key (refresh_intervall = 30); then set refresh_interval = "banana"; then set show = ["open"] and closed_days = 7.
expected: Misspelt key: prv starts, the footer shows 'config: unknown key ...' for about 8 s, and ? lists it. Bad value: prv exits with status 2 and prints 'PATH line N: ...' before the TUI starts. show and closed_days: the header indicator and the closed window follow the file. (02-UAT #7)
result: [pending]

### 7. REVIEW and CI badges match github.com
steps: Press a and add public PRs by URL: an approved PR on a repo with no required-review rule (for example komoot/photon#254), a PR with no checks, an open PR with a failing check and an open PR whose checks are still running (busy repos such as cli/cli have both). Press o on each and compare.
expected: Each REVIEW and CI badge matches github.com. A PR with no checks never shows passing. An approved PR on a repo without required reviews shows approved, never required. (01-UAT #2)
result: [pending]

### 8. MERGE column matches the github.com merge box
steps: Add open PRs in different merge states (mergeable, blocked, behind, conflicts and a draft; for example from cli/cli or golang/go — golang/go#27813 had conflicts on 2026-10-13). Compare each row's MERGE cell with the merge box at the bottom of the PR page. Remove the added PRs afterwards with d then y.
expected: Each MERGE cell agrees with GitHub's merge box. Merged and closed PRs show a blank MERGE cell. (03-UAT #4)
result: [pending]

## Summary

total: 8
passed: 0
issues: 0
pending: 8
skipped: 0
blocked: 0

## Gaps

[none yet]

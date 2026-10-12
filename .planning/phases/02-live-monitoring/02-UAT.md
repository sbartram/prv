---
status: testing
phase: 02-live-monitoring
source: [02-VERIFICATION.md]
started: 2026-10-12T17:11:17Z
updated: 2026-10-12T17:11:17Z
---

## Current Test

number: 1
name: Failure marker and recovery at 80 and 40 columns
expected: |
  Header shows the amber 'refresh failed HH:MM (network) · data from HH:MM:SS' marker with the rows kept, then returns to 'updated HH:MM:SS' after recovery. The 40-column footer keeps notes and the api segment readable. (02-04 D6)
awaiting: user response

## Tests

### 1. Failure marker and recovery at 80 and 40 columns
steps: Run prv in a real terminal at 80 and 40 columns. Toggle Wi-Fi off, wait for one refresh, then on again.
expected: Header shows the amber 'refresh failed HH:MM (network) · data from HH:MM:SS' marker with the rows kept, then returns to 'updated HH:MM:SS' after recovery. The 40-column footer keeps notes and the api segment readable. (02-04 D6)
result: [pending]

### 2. Faint api footer and yellow low-budget warning
steps: View the faint 'api N' footer segment and the yellow header warning (below 20% of the limit) on a real theme and inside a tmux/herdr pane.
expected: The api number is legible but quiet; the warning reads as a warning on light and dark backgrounds. (02-05 D5)
result: [pending]

### 3. Live rate-limit pause and refused r
steps: Against a real or near-exhausted token, watch the header and press r during a pause.
expected: Header shows 'paused until HH:MM (rate limit)'; r flashes 'rate limited - resumes HH:MM' and sends no request; prv resumes by itself. (02-06 D5, 02-07)
result: [pending]

### 4. Filter keys, / prompt and paste in a real terminal
steps: In a real terminal, press 1/2/3, type '/' then digits, paste digits and non-digits into the prompt, press esc and enter. (02-08 D-judgment)
expected: Indicator '[open merged closed]' toggles. Only digits are accepted, including from a paste. The list narrows, the hidden count is right, and esc clears.
result: [pending]

### 5. o opens the PR in a real browser
steps: Press 'o' on a selected PR with the default browser, then with BROWSER set to a bad command. (02-10)
expected: The PR opens in the browser on github.com. A bad launcher gives a footer flash 'could not open the browser', and the screen is not corrupted.
result: [pending]

### 6. ? help overlay on a real terminal
steps: Press '?' on a real terminal, with and without config warnings and at a short pane height. (02-10)
expected: Overlay shows keys, badge legend, settings in effect, api budget and warnings. Any key returns, q does not quit.
result: [pending]

### 7. Live config edits reach prv
steps: Edit ~/.config/prv/config.toml: add a misspelt key (refresh_intervall), then a bad value (refresh_interval = \"banana\"), then show = [\"open\"] and closed_days = 7. (02-09)
expected: Misspelt key: prv starts and the footer shows 'config: unknown key ...' for about 8 s, and '?' lists it. Bad value: exit 2 with 'path line N: ...' before the TUI. show/closed_days: the header indicator and the closed window follow the file.
result: [pending]

## Summary

total: 7
passed: 0
issues: 0
pending: 7
skipped: 0
blocked: 0

## Gaps

---
phase: 02-live-monitoring
plan: 05
subsystem: ui
tags: [go, rate-limit, graphql, footer, header, bubbletea-v2]

requires:
  - phase: 02-live-monitoring
    provides: header segment composer fit() with free prio 6, footer() with notes first (plan 02-04); Cost/Remaining/ResetAt decode (plan 02-03)
provides:
  - "domain.RateLimit.Limit, decoded from the discovery query's rateLimit { limit cost remaining resetAt }"
  - "Faint 'api N' footer segment at the right edge, shown from the first successful fetch on and kept through failed refreshes"
  - "Yellow header segment 'api R/L · resets HH:MM' at prio 6 when Remaining*5 < Limit"
  - "Footer priority on narrow panes: notes, then the api segment, then key hints (dropped first)"
affects: [02-06 backoff uses the same RateLimit data, 02-08 filter segment takes header prio 3, 02-10 footer bindings]

actuals:
  tokens: 4500
  tasks: 2
  commits: 4
plan_head_before: ea9dd3168e0ac1f56bcc20d536c2a291959f6fa6
plan_head_after: 86544eee6db347f7c27d655d738d00ec4359ffdf

tech-stack:
  added: []
  patterns:
    - "Footer is left (notes, hints) plus a right-aligned api segment; the left part gets the width minus the api segment and one space"
    - "Display thresholds use integer maths (Remaining*5 < Limit), no float rounding at the boundary"

key-files:
  created: []
  modified:
    - internal/domain/pr.go
    - internal/github/client.go
    - internal/github/client_test.go
    - internal/github/testdata/search_page1.json
    - internal/github/testdata/search_page2.json
    - internal/ui/view.go
    - internal/ui/view_test.go
    - internal/ui/testdata/TestGolden/list_0x0.golden
    - internal/ui/testdata/TestGolden/list_80x24.golden
    - internal/ui/testdata/TestGolden/list_120x40.golden

key-decisions:
  - "D-06 'always shows' is read as 'from the first successful response on': with Limit 0 the api segment is absent, never a made-up value; a failed refresh keeps the last known value because the snapshot is kept"
  - "The yellow warning boundary is Remaining*5 < Limit: 999/5000 warns, 1000/5000 does not"
  - "Without a known width the api segment follows the hints after ' · '; with a width it is padded to end at the right edge"

patterns-established:
  - "status.json stays without a limit field on purpose, so the Limit 0 path is tested by a real fixture"

requirements-completed: [RFSH-05]

coverage:
  - id: D1
    description: "Every fetch requests rateLimit { limit cost remaining resetAt }; Limit, Remaining and ResetAt come from the last page and Cost is summed; a response without limit decodes Limit 0"
    requirement: RFSH-05
    verification:
      - kind: unit
        ref: "internal/github/client_test.go#TestRateLimitDecoded"
        status: pass
    human_judgment: false
  - id: D2
    description: "The footer ends with a faint 'api N' at the right edge at widths 80 and 120, is absent while Limit is 0, and keeps the last value after a failed refresh"
    requirement: RFSH-05
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestApiFooter"
        status: pass
      - kind: golden
        ref: "internal/ui/view_test.go#TestGolden"
        status: pass
    human_judgment: false
  - id: D3
    description: "Below 20% the header shows 'api R/L · resets HH:MM' in ANSI yellow; 999/5000 warns and 1000/5000 does not"
    requirement: RFSH-05
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestRateWarning"
        status: pass
    human_judgment: false
  - id: D4
    description: "On a narrow pane the footer keeps the notes and the api segment and drops the key hints first; no line is wider than the pane"
    requirement: RFSH-05
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestFooterPriorityNarrow"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestViewFitsScreen"
        status: pass
    human_judgment: false
  - id: D5
    description: "How the faint api segment and the yellow warning look in a real terminal and in a multiplexer pane"
    requirement: RFSH-05
    verification: []
    human_judgment: true
    rationale: "Unit tests cover the text, the SGR codes and the widths, not the look on a real theme"

duration: 4min
completed: 2026-10-12
status: complete
---

# Phase 2 Plan 05: API budget in the footer, yellow warning below 20% Summary

**Every fetch now reports the GraphQL rate limit, the footer shows a faint right-aligned "api N" from the first successful response on, and the header warns in ANSI yellow with the reset time when less than 20% is left.**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-10-12T15:51:15Z
- **Completed:** 2026-10-12T15:53:00Z (code); SUMMARY and state follow
- **Tasks:** 2 (both TDD: RED test commit, then GREEN feature commit)
- **Files modified:** 10 (3 of them golden snapshots)

## Accomplishments
- Discovery query selects `rateLimit { limit cost remaining resetAt }`; `domain.RateLimit.Limit` is filled from the last page (Cost stays summed over pages).
- `footer()` builds a left part (notes, then hints) and a faint `api N` segment padded to the right edge. Hints need at least 8 cells, notes are cut with an ellipsis before the api segment is ever dropped. At width 0 the segment follows the hints after " · ".
- `header()` adds a prio 6 segment in `warnStyle` when `Limit > 0 && Remaining*5 < Limit`. The time uses the model clock, so tests are zone stable.
- Only `list_0x0`, `list_80x24` and `list_120x40` goldens changed, only on the footer line. The colour, empty, error and loading goldens are unchanged.

## Task Commits

1. **Task 1: rate limit in every fetch**
   - RED `7338025` (test): build failed on missing `RateLimit.Limit`
   - GREEN `a797a79` (feat)
2. **Task 2: footer segment and header warning**
   - RED `46f8e39` (test): TestApiFooter, TestRateWarning and TestFooterPriorityNarrow failed
   - GREEN `86544ee` (feat), with the three list goldens

**Plan metadata:** committed separately (docs: complete plan)

## Deviations from Plan

None - plan executed exactly as written.

Notes (not deviations): the first draft of `TestFooterPriorityNarrow` required the text "list truncated" as a prefix at every width. That was a test mistake, because a cut note ends in an ellipsis ("list…"). The assertion now checks the prefix "list". No code change.

## Issues Encountered
None. Every commit used an explicit pathspec; the staged `.serena` files and the modified `.planning/config.json` stayed out of all commits.

## Known Stubs
None.

## Threat Flags
None beyond the plan's threat model. T-02-29 is mitigated: the segment shows only server-reported values, is absent before the first response (`TestApiFooter/no rate data yet`), and survives a failed refresh (`TestApiFooter/failed refresh keeps the last value`). T-02-SC: no new modules (`go.mod` and `go.sum` unchanged).

## Next Phase Readiness
Ready for 02-06 (automatic backoff can read `Snapshot.RateLimit.Limit`) and 02-08 (filter segment at header prio 3). `go build ./...`, `go vet ./...` and `go test ./... -count=1 -race` pass.

## Self-Check: PASSED

Commits `7338025`, `a797a79`, `46f8e39`, `86544ee` are ancestors of HEAD. Acceptance greps pass: `rateLimit { limit cost remaining resetAt }` in client.go, `Limit     int` in pr.go, `"limit": 5000` in both search fixtures, `api 4812` in list_80x24.golden. Golden hygiene command prints nothing. The 8 named UI tests and the 4 named github tests report PASS. Golden tests pass without `-update`.

---
*Phase: 02-live-monitoring*
*Completed: 2026-10-12*

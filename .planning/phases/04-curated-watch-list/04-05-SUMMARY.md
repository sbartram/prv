---
phase: 04-curated-watch-list
plan: 05
subsystem: ui-list
tags: [go, bubbletea-v2, lipgloss-table, attention-sort, two-strike, tdd]
status: complete

requires:
  - phase: 04-curated-watch-list
    provides: "04-01 Manual/Watched/Unavail on PR, MarkWatched/Combine; 04-02 Lookup with NOT_FOUND/SSO/other reasons; 04-03 add prompt and epoch rule; 04-04 saved list that survives anything"
provides:
  - "Unavailable saved PRs as rows: marker and ref, then yellow '⚠ unavailable: REASON' across TITLE..CI, never faint, never widening a column"
  - "Bucket 0 above failing CI, ordered by ref (repo without letter case, then number); Visible never hides an unavailable row; the / number filter still matches it"
  - "Header segment 'N unavailable' (priority 6) right after the PR count"
  - "Compact rows: '+ repo#n' for saved PRs, '+ repo#n ⚠' for unavailable ones"
  - "InWindow keeps every saved PR whatever its close date (WATCH-07)"
  - "domain.Reconcile and Model.strikes: the D-12 two-strike rule for 'other' errors"
affects: [04-06, 04-07]

plan_head_before: c994155e42b58d916e06335a5d784576e180dfab
plan_head_after: 0b6a693a6698d0fa762f938a5a76a7c708498d97

actuals:
  tokens: 10800
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Unavailable row line is replaced by hand after the lipgloss table renders; table line 1 + i is row m.offset + i"
    - "Each segment of a hand-built line is styled on its own, text cut with ansi.Truncate in display cells"
    - "Pure Reconcile(prev, next, strikes) returns a new slice and a new map; the counter lives in the Model, keyed by the saved ref's Key"

key-files:
  created:
    - internal/ui/unavail_test.go
  modified:
    - internal/domain/list.go
    - internal/domain/domain_test.go
    - internal/domain/watch.go
    - internal/domain/watch_test.go
    - internal/ui/view.go
    - internal/ui/styles.go
    - internal/ui/model.go
    - cmd/prv/main_test.go

key-decisions:
  - "Reconcile sorts its result only when it replaced a row with that row's last data, not always: an unconditional Sort reordered the unsorted fixtures of existing view tests; Fetch already sorts, and only a replaced row can change bucket"
  - "Compact rows add '+ ' only to saved rows and do not pad other rows to align (the plan states the compact form per row type)"
  - "The strike counter is a map[domain.Ref]int in the ui Model keyed by Watched.Key(); a refresh that errors, is discarded by the epoch rule, or is never applied does not reach Reconcile"
  - "Unavailable rows are drawn by unavailLine after the table renders; their table cells are blank, so the reason never widens a column and does not raise the CI or REVIEWERS minimum"

requirements-completed: [LIST-09, WATCH-07]

coverage:
  - id: D1
    description: "A saved PR GitHub could not return is a row at the top with its reason in yellow, the header counts it, and every other PR keeps all its columns"
    requirement: LIST-09
    verification:
      - kind: unit
        ref: "internal/ui/unavail_test.go#TestUnavailableRow"
        status: pass
      - kind: unit
        ref: "internal/ui/unavail_test.go#TestUnavailableNoWiden"
        status: pass
      - kind: unit
        ref: "internal/ui/unavail_test.go#TestUnavailableStyles"
        status: pass
      - kind: unit
        ref: "internal/ui/unavail_test.go#TestUnavailableHeaderCount"
        status: pass
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestUnavailableEndToEnd"
        status: pass
    human_judgment: true
    rationale: "Screen-reader reading of the row and terminals that draw the warning glyph two cells wide are not asserted by any test (RESEARCH A5; human check in plan 04-07)"
  - id: D2
    description: "Unavailable rows sort in bucket 0 by ref and no state toggle hides them; the / filter matches them by number; compact rows keep the + and show one warning glyph"
    requirement: LIST-09
    verification:
      - kind: unit
        ref: "internal/domain/domain_test.go#TestBucketUnavailable"
        status: pass
      - kind: unit
        ref: "internal/domain/domain_test.go#TestSortUnavailableByRef"
        status: pass
      - kind: unit
        ref: "internal/domain/domain_test.go#TestVisibleUnavailable"
        status: pass
      - kind: unit
        ref: "internal/ui/unavail_test.go#TestUnavailableFilters"
        status: pass
      - kind: unit
        ref: "internal/ui/unavail_test.go#TestCompactWatchRows"
        status: pass
    human_judgment: false
  - id: D3
    description: "A saved PR stays listed after it closes, whatever the close date, as one row however often the list refreshes; the 1/2/3 toggles still hide it like any closed PR; the saved entry is never removed"
    requirement: WATCH-07
    verification:
      - kind: unit
        ref: "internal/domain/domain_test.go#TestInWindowManual"
        status: pass
      - kind: unit
        ref: "internal/ui/unavail_test.go#TestClosedSavedFilters"
        status: pass
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestClosedSavedPREndToEnd"
        status: pass
    human_judgment: false
  - id: D4
    description: "A not-found or SSO answer makes a row unavailable at once; another per-PR error keeps the last data once and shows unavailable on the second failed refresh in a row; a refresh that fails as a whole changes neither rows nor counts"
    requirement: LIST-09
    verification:
      - kind: unit
        ref: "internal/domain/watch_test.go#TestReconcile"
        status: pass
      - kind: unit
        ref: "internal/ui/unavail_test.go#TestTwoStrikeRule"
        status: pass
    human_judgment: false

duration: 35 min
completed: 2026-10-13
---

# Phase 4 Plan 05: Saved PRs never vanish — a closed one stays, a broken one says why Summary

**A saved PR that GitHub cannot load now shows as a yellow "⚠ unavailable: REASON" row at the top of the list, a saved PR that closed long ago stays listed, and a one-off GitHub error keeps the last data until it repeats.**

## Performance

- **Duration:** about 35 min
- **Tasks:** 2 (both TDD: RED commit, then GREEN commit)
- **Files:** 9 changed (1 new)

## Accomplishments

- **Unavailable rows (LIST-09, D-09).** `unavailLine` draws the row by hand after the table renders: the cursor marker and the `+ ref` cell as in every other row, then `⚠ unavailable: REASON` over the space of the TITLE..CI columns, cut with `…` to fit. The reason is yellow (`warnStyle`), never faint (`dimmed` is false for these rows), and the selected row carries reverse video. The table cells of these rows are blank, so a long reason never widens a column; the column-header line is identical for a 1-character and an 80-character reason.
- **Order and filters (D-11, D-13, D-14).** `Bucket` returns 0 for any row with an `Unavail` kind, above failing CI. Bucket 0 sorts by ref (repo compared without letter case, then number, then repo as a last tie-break). `Visible` skips the state test for these rows, so 1/2/3 never hide them; the `/` number prefix still matches them. Loaded saved PRs use the ordinary buckets, so a merged or closed saved PR sorts in bucket 7 or 8.
- **Header and compact form (D-15, D-05).** The header shows `N unavailable` in yellow right after the PR count with priority 6, so on a 30-column pane it outlives the count and the update time. Compact rows draw `+ ` for saved PRs (kept even when the ref is dropped for lack of room) and one `⚠` instead of the three glyphs for unavailable rows.
- **Closed saved PRs (WATCH-07).** `InWindow` returns true for every `Manual` PR. The end-to-end test saves a PR closed 2026-08-01 with a 14-day window: it is listed with `closed`, stays one row over repeated refreshes, and survives a refresh whose Lookup answers HTTP 502 (the header shows `refresh failed`, the row and the saved entry stay).
- **Two-strike rule (D-12).** `domain.Reconcile(prev, next, strikes)` is pure. A row that is unavailable with an "other" error is a strike for its saved ref: the first strike shows the previous row's last data, the second in a row (or no last data at launch) shows it unavailable. NOT_FOUND and SSO show at once; a PR that loads clears its count. `Model.strikes` holds the counts; the fetch error branch and the epoch discard never reach Reconcile, so they never count. Nothing in this code touches `watchlist.json`.

## TDD Gate Compliance

| Task | RED commit | GREEN commit |
| ---- | ---------- | ------------ |
| 1 (tracer) | fcb4b10 `test(04-05): add failing tests for unavailable saved PR rows` | dd27bbb `feat(04-05): show saved PRs GitHub cannot load as warning rows at the top` |
| 2 | 68bc84b `test(04-05): add failing tests for closed saved PRs and the two-strike rule` | 0b6a693 `feat(04-05): keep saved closed PRs listed and apply the two-strike rule` |

- **RED evidence, task 1.** `go test` of the ten named tests failed on the planned assertions: `Bucket = 5, want 0`, `order = [acme/x#1 Acme/api#10 acme/api#2 acme/web#9]`, `rows = ""` for the merged-only filter, row 3 not `acme/gone#5`, no yellow or reverse SGR on the unavailable row, header lacking `2 unavailable`, compact row `> api#1 ✗ ✗` instead of `> + gone#5 ⚠`, and the end-to-end first row not the unavailable one. `TestUnavailableNoWiden` passed at RED: it is a guard that the reason never widens a column, which holds before any reason is drawn.
- **RED evidence, task 2.** `InWindow = false, want true` (3 cases), `TestReconcile` strike counts and kept row wrong against a stub that returned its input, `TestTwoStrikeRule` showing `unavailable` after one failed refresh, and `TestClosedSavedPREndToEnd` with no `octo/old#7` line. `TestClosedSavedFilters` passed at RED: the state toggles already worked, and the test pins that they still do once saved PRs are exempt from the window.
- **Semantic assessment.** Each failure above names the planned assertion for the intended reason; none is a compile, import, fixture or zero-test failure. To keep the RED commits compiling, the task 2 RED commit holds two stubs: `Reconcile` returning its input, and the unused `Model.strikes` field.
- `gsd_run check tdd-red-evidence` was not run: it reads TAP, JUnit, swift-testing and unittest reports, and `go test` plain output is none of these. The plan is `type: execute`, not `type: tdd`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Reconcile sorts only when it replaced a row**
- **Found during:** Task 2, first full `go test ./...` after the GREEN code
- **Issue:** The plan says the result of `Reconcile` is sorted with `domain.Sort`. Sorting every refresh in the model reordered the unsorted fixtures that existing view tests feed straight into `fetchedMsg` (for example `TestScrollFollowsCursor` failed), which would also break the goldens the plan requires to stay byte-identical.
- **Fix:** `Reconcile` calls `Sort` only when it swapped an unavailable row for its last data, which is the one case where a row's bucket changes. Otherwise it keeps the order of `next`; `Fetch` already sorts in production. `TestReconcile` pins both cases ("a row replaced by its last data is sorted into place", "rows that were not replaced keep their order").
- **Files modified:** internal/domain/watch.go, internal/domain/watch_test.go
- **Verification:** `go test ./... -count=1 -race` passes; `git diff --exit-code internal/ui/testdata` exits 0
- **Commit:** 0b6a693

**Total deviations:** 1 auto-fixed (1 blocking). **Impact:** none on behavior in production; the doc comment of `Reconcile` states the rule.

## Issues Encountered

None.

## Known Stubs

None. The two RED-commit stubs (`Reconcile` returning its input, the unused `strikes` field) are replaced in 0b6a693.

## Threat Flags

None. The only new text on screen is `Unavail.Reason`, built by domain constructors (04-02) and drawn through `ansi.Truncate` in its own styled segment (T-04-21). No new module, endpoint or file access (`git diff --exit-code go.mod go.sum` exits 0).

## Verification

- `go vet ./... && go test ./... -count=1 -race`: all six packages pass.
- The ten Task 1 tests and the five Task 2 tests all report `--- PASS`.
- `git diff --exit-code go.mod go.sum internal/ui/testdata` exits 0.

## Next Phase Readiness

Ready for 04-06 (remove with confirmation) and 04-07 (detail view for unavailable rows, `o` open, help legend, goldens). Both still need the `?` legend entries and the detail view for unavailable rows (D-11); this plan did not touch `detail.go` or `help.go`. WATCH-07 is also declared by 04-07, so its checkbox in REQUIREMENTS.md stays pending until that plan finishes.

## Self-Check: PASSED

- Created file `internal/ui/unavail_test.go`: found.
- Commits fcb4b10, dd27bbb, 68bc84b and 0b6a693 are ancestors of HEAD.

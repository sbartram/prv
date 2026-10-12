---
phase: 03-blocker-detail
plan: 06
subsystem: github-detail-data
tags: [go, github-graphql, detail-view, checks, reviewers, paging, tdd]

requires:
  - phase: 03-blocker-detail
    provides: "plan 03-02 ClassifyCheck, LatestChecks and mapContext; plan 03-04 MergeReviewers, reviewInputs and requestInputs"
provides:
  - "FetchDetail(ctx, id): one PR's de-duplicated, sorted checks, every reviewer (including commented and dismissed) and de-duplicated counts"
  - "detailQuery (operation Detail): reviewers plus contexts(first: 100, after: $after), with no per-state count and no per-check link fields"
  - "domain.Detail, CheckCounts, CountChecks, SortChecks and CheckState.Glyph"
  - "Check.Source: the workflow file base name, else the app name, empty for a status context"
  - "Paging up to maxPages (5) pages; Truncated, Read and Total disclose a partial read"
affects: [03-07, 03-08]

plan_head_before: c8fbc9ac69c6663a6661b88b6545d698840b8296
plan_head_after: d4c9bd61164e018e44cd437cffec8f3c3f28252e

actuals:
  tokens: 34400
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Detail counts are always CountChecks of the de-duplicated checks, at every size; GitHub's per-state counts are never requested"
    - "All pages' contexts are de-duplicated together (LatestChecks), so a run superseded on another page collapses"
    - "fixtureTransport routes the Detail operation to its own ordered file list and fails the test on a request past the list"
    - "Hostile-string mapper tests decode raw JSON straight into the wire structs, bypassing go-gh's own sanitiser"

key-files:
  created:
    - internal/github/detail_test.go
    - internal/github/testdata/detail_basic.json
    - internal/github/testdata/detail_page1.json
    - internal/github/testdata/detail_page2.json
  modified:
    - internal/domain/checks.go
    - internal/domain/checks_test.go
    - internal/github/checks.go
    - internal/github/client_test.go

key-decisions:
  - "D-08's 'summary built from the counts' is read as the de-duplicated counts at every size (D-03); paging to 5 pages keeps them exact up to 500 check runs, past that Read and Total let plan 03-07 say 'read N of M'"
  - "A page that fails after page 1 (other than a rate limit or a cancelled context) keeps the earlier pages and sets Truncated; a rate limit or cancel on any page is returned, as in Fetch"
  - "A hasNextPage with an empty endCursor stops paging as Truncated, so a malformed response cannot resend the first page"

patterns-established:
  - "TDD stubs: the RED commit carries compiling stubs (Glyph, CountChecks, SortChecks, FetchDetail, detailReviewers) so the tests fail on assertions"

requirements-completed: [DETL-01, DETL-02]

coverage:
  - id: D1
    description: "FetchDetail returns the PR's checks de-duplicated by (workflow, name), newest run kept, sorted failing, cancelled, running, passing, skipped or neutral, unknown, with Source set from the workflow file or app"
    requirement: DETL-01
    verification:
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetail"
        status: pass
      - kind: unit
        ref: "internal/domain/checks_test.go#TestSortChecks"
        status: pass
      - kind: unit
        ref: "internal/domain/checks_test.go#TestCheckGlyph"
        status: pass
    human_judgment: false
  - id: D2
    description: "FetchDetail returns every reviewer with their state: changes requested, pending (including hidden team requests), approved, commented, dismissed; a re-requested reviewer is listed once as pending"
    requirement: DETL-02
    verification:
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetail"
        status: pass
    human_judgment: false
  - id: D3
    description: "Counts come from the de-duplicated checks at every size; a PR with more than 100 check runs is paged up to 5 pages, and a partial read reports Read of Total"
    requirement: DETL-01
    verification:
      - kind: unit
        ref: "internal/domain/checks_test.go#TestCountChecks"
        status: pass
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetailPaging"
        status: pass
    human_judgment: false
  - id: D4
    description: "A missing or non-PullRequest node is a KindGraphQL error 'this pull request could not be loaded'; a rate limit is KindRateLimited; hostile names, workflow file paths and logins are cleaned at the seam"
    requirement: DETL-01
    verification:
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetailNotFound"
        status: pass
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetailRateLimited"
        status: pass
      - kind: unit
        ref: "internal/github/detail_test.go#TestFetchDetailCleans"
        status: pass
    human_judgment: false
  - id: D5
    description: "The Detail query as written is accepted by GitHub and returns the shapes the fixtures model (file { path } on the workflow run was verified live in RESEARCH; this exact trimmed text has not been run as written)"
    requirement: DETL-01
    verification: []
    human_judgment: true
    rationale: "Fixtures are recorded by hand from the RESEARCH-verified shapes, not from a live call of this exact query text; plan 03-07 or a live run of the app should confirm it once against a real PR"

duration: 9min
completed: 2026-10-12
status: complete
---

# Phase 3 Plan 06: One query brings a PR's every check and every reviewer, counted without re-runs Summary

**`FetchDetail` returns a PR's de-duplicated, failing-first checks, every reviewer with their state (commented and dismissed included) and counts that never include superseded re-runs, paging up to 500 check runs and reporting how many were read of how many exist**

## Performance

- **Duration:** about 9 min
- **Started:** 2026-10-12T23:22Z (approximate; the start time was not captured)
- **Completed:** 2026-10-12T23:32Z
- **Tasks:** 2 (each as a RED and a GREEN commit)
- **Files modified:** 8 (4 Go files, 1 new Go test file, 3 JSON fixtures)

## Accomplishments

- `domain.Detail`, `CheckCounts`, `CountChecks`, `SortChecks` and `CheckState.Glyph`: the sort is the D-10 order (failing, cancelled, running, passing, skipped or neutral, unknown), then lower-cased name, then workflow, then name, so any input order gives the same output.
- `detailQuery` and `FetchDetail`: one `node(id:)` query with `latestOpinionatedReviews`, `latestReviews`, `reviewRequests` and `contexts(first: 100, after: $after)`. The per-state count selections and per-check link fields are left out, and the acceptance greps for `ByState`, `detailsUrl` and `targetUrl` in `internal/github/checks.go` print nothing.
- `Check.Source` is filled by `mapContext`: `path.Base` of the cleaned workflow file path, else the cleaned app name, empty for a status context.
- Paging: up to `maxPages` pages; all pages' contexts are de-duplicated together; `Counts` is always `CountChecks(Checks)`. Past the cap, or after a failed later page, `Truncated` is set and `Read` and `Total` say how many runs were read of how many exist.
- Error policy follows `Fetch`: a null or non-PullRequest node is a `KindGraphQL` error "this pull request could not be loaded"; a rate limit or a cancelled context on any page is returned; partial GraphQL data on page 1 is kept.

## Task Commits

Each task was committed atomically as a RED and a GREEN commit:

1. **Task 1: One Detail query returns a PR's de-duplicated, sorted checks and every reviewer with their state**
   - RED: `06e2588` (test)
   - GREEN: `51a4aa7` (feat)
2. **Task 2: PRs with more than 100 check runs are paged; past the cap the counts stay de-duplicated and the read cap is reported**
   - RED: `c4c1782` (test)
   - GREEN: `d4c9bd6` (feat)

**Plan metadata:** the commits that follow this file (docs: complete plan, then state and roadmap)

## TDD Gate Compliance

Both tasks have a `test(03-06)` commit before the `feat(03-06)` commit.

- **RED, Task 1:** `TestCheckGlyph`, `TestSortChecks` and `TestCountChecks` failed on their assertions against stubs (empty glyph, no-op sort, zero counts). `TestFetchDetail` failed with "Detail requests = 0, want 1", `TestFetchDetailNotFound` and `TestFetchDetailRateLimited` with "err = nil", `TestFetchDetailCleans` with "checks/reviewers = 3/0, want 3/3". The stubs make the tests compile, so no failure was a load or compile error. The semantic assessment: every failing test named the planned behaviour it checks.
- **RED, Task 2:** `TestFetchDetailPaging` failed in all four subtests against the Task 1 single-page `FetchDetail` ("Detail requests = 1, want 2", "Read 100, want 500", "err = nil, want KindRateLimited"), with no stub needed.
- **GREEN:** the full suite (`go vet ./...`, `go test ./... -count=1 -race`) passes after each GREEN commit. No REFACTOR commit was needed.
- The `gsd_run check tdd-red-evidence` classifier was not run: `workflow.tdd_mode` is not enabled in this project and Go's `go test` output is not one of its supported report formats.

## Files Created/Modified

- `internal/domain/checks.go` - `Glyph`, `rank`, `CheckCounts`, `CountChecks`, `SortChecks`, `Detail`; `Check.Source` comment updated
- `internal/domain/checks_test.go` - `TestCheckGlyph`, `TestSortChecks`, `TestCountChecks`
- `internal/github/checks.go` - `detailQuery`, wire types, `detailReviewers`, `detailPage`, `FetchDetail` with paging; `wireSuite` gains `File`; `mapContext` fills `Source`
- `internal/github/client_test.go` - `fixtureTransport` routes the Detail operation (`detail`, `detailBodies`); one struct literal gains the `File` field
- `internal/github/detail_test.go` - mapping, not-found, rate-limit, cleaning and paging tests
- `internal/github/testdata/detail_basic.json`, `detail_page1.json`, `detail_page2.json` - Detail response fixtures

## Decisions Made

- D-08's "built from the counts" is read as the de-duplicated counts at every size, as the plan flagged. Nothing the user decided changes.
- A later-page failure other than a rate limit or a cancelled context keeps the earlier pages and sets `Truncated`.
- Paging stops as `Truncated` when GitHub reports another page but no end cursor, so a malformed response cannot resend page 1.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Existing test literal did not compile after `wireSuite` gained `File`**
- **Found during:** Task 1 (GREEN)
- **Issue:** `TestFailingCleansNames` builds the anonymous workflow-run struct by literal; adding `File *struct{ Path string }` made the type differ.
- **Fix:** added the `File` field to that literal. `client_test.go` is in the plan's `files_modified`.
- **Files modified:** internal/github/client_test.go
- **Verification:** `go vet ./...` and the full suite pass.
- **Committed in:** 51a4aa7 (Task 1 GREEN)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** None on scope. One small test edit, within a file the plan names.

## Issues Encountered

- `TestSortChecks` first expected `Build/z` before `build/a`; the plan's sort key (lower-cased name, then workflow, then name) puts `build/a`, `build/b`, `Build/z`. The test was fixed before the RED commit.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. The RED-commit stubs were replaced in the GREEN commits.

## Next Phase Readiness

- Plan 03-07 can draw `domain.Detail` (checks with `Source`, reviewers, `Counts`, and `Truncated` with `Read` and `Total` for a "read N of M" note). Plan 03-08 can call `FetchDetail` on each refresh tick.
- `FetchDetail` does not return a rate-limit snapshot; if 03-08 needs the query cost, `Detail` or the return values will need a field for it.
- The exact `detailQuery` text has not been run against GitHub as written (see coverage D5).

## Self-Check: PASSED

- Files: `internal/github/detail_test.go`, `internal/github/testdata/detail_basic.json`, `detail_page1.json`, `detail_page2.json`, `internal/domain/checks.go`, `internal/github/checks.go` all exist.
- Commits `06e2588`, `51a4aa7`, `c4c1782` and `d4c9bd6` are ancestors of HEAD.
- `go vet ./...` and `go test ./... -count=1 -race` pass; `git diff --exit-code go.mod go.sum` exits 0.

---
*Phase: 03-blocker-detail*
*Completed: 2026-10-12*

---
phase: 03-blocker-detail
plan: 01
subsystem: github-checks
tags: [go, github-graphql, checks, tracer, bubbletea-v2]

requires:
  - phase: 01-status-at-a-glance
    provides: discovery query, rollup CI badge, domain.Clean seam, fixture transports
  - phase: 02-live-refresh
    provides: single-flight Client.Fetch, rate-limit error policy, backoff
provides:
  - "FailingChecks nodes(ids:) query, chunked 25 per request, inside Client.Fetch (D-01)"
  - "domain.ClassifyCheck / LatestChecks / SummarizeFailing and PR.Failing"
  - "ui.ciCell: '✗ lint +2' and '✗ deploy (cancelled)' in the CI cell, capped at 24 cells"
  - "Operation-routed fixture transports (opName, failingNoneBody, answerFailingNone)"
affects: [03-02, 03-03, 03-06, 03-07]

actuals:
  tokens: 10400
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Two-step fetch: lean discovery, then a read-only nodes(ids:) query only for failing PRs"
    - "Pure check classification and re-run de-duplication in internal/domain, fed by the github mapper"
    - "Fixture transport routed by GraphQL operation name"

key-files:
  created:
    - internal/domain/checks.go
    - internal/domain/checks_test.go
    - internal/github/checks.go
    - internal/ui/cells.go
    - internal/github/testdata/failing_contexts.json
  modified:
    - internal/domain/pr.go
    - internal/github/client.go
    - internal/github/client_test.go
    - internal/github/errors_test.go
    - internal/ui/view.go
    - cmd/prv/main_test.go

key-decisions:
  - "'+N' counts only other failing checks when a failing check is named, and other cancelled checks when the named check is cancelled; cancelled checks are never added to a failing count"
  - "The named check is the failing one with the smallest lower-cased name, so the row text is stable between refreshes"
  - "A truncated contexts page (hasNextPage), a missing node, a null rollup or a failed query keeps the plain '✗ failing' badge: a name is never guessed"
  - "ciCell falls back to the plain badge cut to the cell when the text is wider than 24 cells; plan 03-02 replaces that with a name-only cut"

patterns-established:
  - "ClassifyCheck is total: unknown status, conclusion or state is (CheckOther, 'unknown'), never failing or passing"
  - "Remote check, workflow, app and context names pass through domain.Clean in mapContext"
  - "Ids travel only as the $ids variable; tests assert the query text holds no id"

requirements-completed: [LIST-07]

coverage:
  - id: D1
    description: "A failing PR's CI cell names a failing check plus how many more fail ('✗ lint +2'), with a superseded run neither named nor counted"
    requirement: LIST-07
    verification:
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestFailingNameEndToEnd"
        status: pass
      - kind: unit
        ref: "internal/github/client_test.go#TestFailingNames"
        status: pass
    human_judgment: false
  - id: D2
    description: "One FailingChecks query per 25 failing PRs inside the existing Fetch, none when nothing fails, ids only as a variable"
    requirement: LIST-07
    verification:
      - kind: unit
        ref: "internal/github/client_test.go#TestNoFailingNoRequest"
        status: pass
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestFailingNameEndToEnd"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every check kind is classified and named right: status contexts, timed-out, startup failure, cancelled-only, skipped, unknown values; hostile names are cleaned"
    requirement: LIST-07
    verification:
      - kind: unit
        ref: "internal/domain/checks_test.go#TestClassifyCheck"
        status: pass
      - kind: unit
        ref: "internal/domain/checks_test.go#TestSummarizeFailing"
        status: pass
      - kind: unit
        ref: "internal/github/client_test.go#TestFailingCleansNames"
        status: pass
    human_judgment: false

duration: 4min
completed: 2026-10-12
plan_head_before: 034f67d5d38c5742f974b0515b7662ddf2d53449
plan_head_after: c22cb91ec17db70928f2c0f8f32cf2b650151129
commits: 4
status: complete
---

# Phase 3 Plan 1: Tracer, a failing PR's row names the check that is failing Summary

**Two-step GitHub fetch (lean discovery, then one read-only `FailingChecks nodes(ids:)` query per 25 failing PRs) with pure re-run de-duplication, so a failing row reads `✗ lint +2` or `✗ deploy (cancelled)`**

## Performance

- **Duration:** 4 min
- **Started:** 2026-10-12T22:54:51Z
- **Completed:** 2026-10-12T22:58:16Z
- **Tasks:** 2 (tracer plus one auto task, each RED then GREEN)
- **Files modified:** 11 (5 created, 6 modified)

## Accomplishments

- The D-01 two-step fetch works end to end through the real `github.Client` and `ui.Model`: `TestFailingNameEndToEnd` shows `✗ lint +2` on the failing row, `✓ passing` on the other, and exactly one `FailingChecks` request with `ids == ["PR_7"]`.
- D-03 re-run collapse: a job that failed in an older check suite and passed in a newer one is neither named nor counted (keyed by context kind, workflow, name; newest suite then `databaseId` wins).
- Total check classification (D-04 and the enum table): FAILURE, TIMED_OUT, STARTUP_FAILURE, ACTION_REQUIRED and status-context ERROR/FAILURE are failing; CANCELLED is separate; SKIPPED/NEUTRAL/STALE neutral; unknown values are `CheckOther`.
- A cancelled-only PR reads `✗ deploy (cancelled)`; the badge colour and attention bucket still come from the rollup state only.
- "Never guess a name": truncated pages, missing nodes, null rollups and failed queries keep the plain `✗ failing` badge (warning "Check names could not be loaded" on a failed query).

## Task Commits

TDD, RED then GREEN per task:

1. **Task 1 (tracer): a failing PR's row names its failing check**
   - RED: `7c42586` (test)
   - GREEN: `c4d2bfc` (feat)
2. **Task 2: every kind of failing check is named right**
   - RED: `bd22f36` (test)
   - GREEN: `c22cb91` (feat)

**Plan metadata:** the commit that adds this SUMMARY and the STATE/ROADMAP updates follows (docs).

## TDD evidence

- **Task 1 RED:** `go test ./cmd/prv -run TestFailingNameEndToEnd` failed on the planned assertions: the row read `✗ failing` instead of `✗ lint +2`, and `FailingChecks requests = 0, want exactly 1`. The test compiled and ran; no syntax, import or fixture fault.
- **Task 2 RED:** 12 `TestClassifyCheck` rows, 2 `TestSummarizeFailing` rows, `TestFailingNames` (PR 11 `More:1` not 2, PR 12 and 16 unnamed) and `TestFailingCleansNames` failed on assertions; `TestLatestChecks` already passed because Task 1 implemented that function.
- **Classifier note:** `gsd_run check tdd-red-evidence` supports TAP, JUnit, swift-testing and unittest reports only; Go's `go test` output has no supported reporter here (no gotestsum or go-junit-report installed), so the semantic assessment above was done by hand on the real `go test` output. The plan is `type: execute`, so the plan-level gate does not apply.
- **Gate commits present:** `test(03-01)` precedes `feat(03-01)` in both tasks; no REFACTOR commit was needed.

## Files Created/Modified

- `internal/domain/checks.go` - `CheckState`, `Check`, `ClassifyCheck`, `LatestChecks`, `FailingCheck`, `SummarizeFailing`
- `internal/domain/checks_test.go` - table tests for all three
- `internal/domain/pr.go` - `PR.Failing`
- `internal/github/checks.go` - `failingQuery`, `failingChunk = 25`, wire structs, `mapContext`, `fetchFailing` with the step-2 error policy
- `internal/github/client.go` - `Fetch` calls `fetchFailing` after the window filter and before `Sort`
- `internal/github/client_test.go` - operation-routed `fixtureTransport`, `opName`, `failingNoneBody`, `TestNoFailingNoRequest`, `TestFailingNames`, `TestFailingCleansNames`
- `internal/github/errors_test.go` - `answerFailingNone`, wraps `TestLaterPageFailure`
- `internal/github/testdata/failing_contexts.json` - PR_11 (lint, test, vet timed out, build failed then passed), PR_12 (Jenkins ERROR), PR_16 (cancelled deploy)
- `internal/ui/cells.go` - `maxCIColW = 24`, `ciCell`
- `internal/ui/view.go` - the CI cell is built with `ciCell(pr, maxCIColW)`
- `cmd/prv/main_test.go` - `routedTransport`, `TestFailingNameEndToEnd`

## Decisions Made

- `+N` semantics and name choice as in `key-decisions` above (follows the plan's flagged assumptions).
- Cost accounting: `snap.RateLimit.Cost` sums discovery and FailingChecks costs; `Limit`, `Remaining` and `ResetAt` come from the step-2 response only when it carries a `limit` (the fixture test pins this).
- `fetchFailing` adds the warning before deciding whether a partial GraphQL response is usable, so both error paths share one `addWarning` call.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test expectation for summed cost**
- **Found during:** Task 2 (RED, writing `TestFailingNames`)
- **Issue:** I first wrote the cost assertion as 1; `status.json` carries `cost: 1`, so the correct total over both queries is 2.
- **Fix:** Corrected the expectation and comment before the RED commit.
- **Files modified:** `internal/github/client_test.go`
- **Committed in:** `bd22f36`

**2. [Process] Stray comment fix**
- **Found during:** Task 1 (after RED commit)
- **Issue:** A `sed -i` for a doc comment ("run builds" to "ctxRun builds") failed on BSD sed and the RED commit kept the old wording.
- **Fix:** Corrected the comment in the GREEN commit.
- **Committed in:** `c4d2bfc`

---

**Total deviations:** 1 auto-fixed (1 bug in my own test), 1 process note
**Impact on plan:** None on scope or behaviour.

Not done as written: the plan said to wrap "any other errors_test.go test whose discovery body holds a FAILURE or ERROR rollup" with `answerFailingNone`. Only `TestLaterPageFailure` needed it. The `TestTokenNeverAppears` "later page" scenario also reaches step 2 (it receives a 500 and a "Check names could not be loaded" warning) but asserts only that the token never appears, so it passes unchanged and was left alone.

## Issues Encountered

None.

## Live validation (Task 1 step 10)

Ran `failingQuery` once, read-only, through `gh api graphql --input` with `ids: [one PullRequest id]` found by `repo:cli/cli is:pr is:open status:failure`. Result: no errors, `rateLimit.cost` 1, node typename `PullRequest`, 25 contexts. The exact query text from RESEARCH is therefore validated live. No probe output was committed.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. The `ciCell` over-width fallback (plain badge cut to the cell) is intentional and is replaced by a name-only cut in plan 03-02.

## Threat Flags

None. The new network surface (the `FailingChecks` read-only query) and the new remote strings (check, workflow, app, context names) are the ones in the plan's threat model (T-03-01, T-03-05, T-03-SC). `git diff --exit-code go.mod go.sum` exits 0.

## Next Phase Readiness

- Plan 03-02 can build on `fetchFailing`, `SummarizeFailing(…, complete)` and the routed fixture transports: re-run fixture, truncated pages, partial errors, chunking over 25, rate limits and the long-name cut in `ciCell`.
- LIST-07 is also claimed by plan 03-02; it stays pending in REQUIREMENTS.md until the last declaring plan finishes.
- Carried risks: a refresh with hundreds of failing PRs sends up to 20 sequential `FailingChecks` requests (1 point each); chunk size 25 rests on one measurement (RESEARCH A3).

## Self-Check: PASSED

- Files: `internal/domain/checks.go`, `internal/domain/checks_test.go`, `internal/github/checks.go`, `internal/ui/cells.go`, `internal/github/testdata/failing_contexts.json` all FOUND.
- Commits: `7c42586`, `c4d2bfc`, `bd22f36`, `c22cb91` all FOUND on the branch.
- `go vet ./... && go test ./... -count=1 -race` passes; all acceptance-criteria greps print a line; `git diff --exit-code go.mod go.sum` exits 0.

---
*Phase: 03-blocker-detail*
*Completed: 2026-10-12*

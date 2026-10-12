---
phase: 03-blocker-detail
plan: 04
subsystem: domain-github
tags: [go, github-graphql, reviewers, merge-state, tdd]

requires:
  - phase: 03-blocker-detail
    provides: "plan 03-01 PR.Failing and the FailingChecks step; fixtureTransport routing by operation name"
provides:
  - "domain.Merge (MergeNone, MergeUnknown, MergeMergeable, MergeBlocked, MergeBehind, MergeConflicts), DeriveMerge, Badge and Glyph"
  - "domain.MergeReviewers with Reviewer, ReviewerState (Glyph, Word), ReviewerKind, ReviewInput, RequestInput"
  - "PR.Merge and PR.Reviewers filled on every refresh by the discovery query (mergeStateStatus, mergeable, reviewRequests)"
  - "wireReview and wireRequest wire types, reviewInputs and requestInputs helpers that plan 03-06 reuses"
  - "blockers.json discovery fixture and hostile reviewer logins in status.json PR 18"
affects: [03-05, 03-06, 03-07]

plan_head_before: 13cab8b4d4a1428e6087542fa29da5b1e6edfa90
plan_head_after: 682ff2b7b0bb6e9825518fada0ee2285177a5a1e

actuals:
  tokens: 8600
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "State and draft are decided before any GitHub status string, so a stale value can never make a PR look ready"
    - "Union fragments decode into one tagged struct behind a nil-guarded pointer; a null element becomes a counted hidden entry, never a dropped one"
    - "Pure merge function over cleaned inputs: the github seam cleans, domain.MergeReviewers only orders and dedupes"

key-files:
  created:
    - internal/domain/merge.go
    - internal/domain/merge_test.go
    - internal/domain/reviewers.go
    - internal/domain/reviewers_test.go
    - internal/github/testdata/blockers.json
  modified:
    - internal/domain/pr.go
    - internal/domain/domain_test.go
    - internal/github/client.go
    - internal/github/client_test.go
    - internal/github/testdata/status.json

key-decisions:
  - "A review login loses its [bot] suffix the same way a request name does, so a bot that reviewed and was re-requested compares equal and shows once, as pending"
  - "Named requests are also de-duplicated by lower-cased name, so a repeated request cannot show a reviewer twice; hidden requests are never de-duplicated"
  - "A request that has a type but an empty name (a User or Team whose login or slug is empty after cleaning) is hidden pending, not a blank person"
  - "go-gh shows an ESC in a login as a visible '^[' before the mapper runs, so TestCleanAtSeam pins that exact output and TestMapperCleansRemoteStrings pins the cleaned form"

patterns-established:
  - "TDD stubs: the RED commit carries compiling stubs (zero values) so tests fail on their assertions, not on the build"

requirements-completed: [LIST-04, LIST-06]

coverage:
  - id: D1
    description: "Every refresh gives each PR a merge-readiness value: mergeable for CLEAN, HAS_HOOKS and UNSTABLE; blocked; behind; conflicts for DIRTY and for UNKNOWN with CONFLICTING; a dash for UNKNOWN, empty and unrecognised values and open drafts; blank for merged, closed and unknown-state PRs"
    requirement: LIST-06
    verification:
      - kind: unit
        ref: "internal/domain/merge_test.go#TestDeriveMerge"
        status: pass
      - kind: unit
        ref: "internal/domain/merge_test.go#TestMergeGlyphPrefix"
        status: pass
      - kind: integration
        ref: "internal/github/client_test.go#TestMergeMapping"
        status: pass
    human_judgment: false
  - id: D2
    description: "Merge and Reviewers never reorder the list: Bucket and Sort read neither field"
    requirement: LIST-06
    verification:
      - kind: unit
        ref: "internal/domain/merge_test.go#TestMergeDoesNotSort"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every refresh gives each PR its reviewers in changes requested, pending, approved order; a re-requested reviewer is only pending (case-insensitive); a null requested reviewer is a hidden pending reviewer; bots lose [bot]; a null review author is skipped"
    requirement: LIST-04
    verification:
      - kind: unit
        ref: "internal/domain/reviewers_test.go#TestMergeReviewers"
        status: pass
      - kind: unit
        ref: "internal/domain/reviewers_test.go#TestReviewerGlyphs"
        status: pass
      - kind: integration
        ref: "internal/github/client_test.go#TestReviewerMapping"
        status: pass
    human_judgment: false
  - id: D4
    description: "A hostile reviewer login or team slug never reaches the screen strings: domain.Clean runs at the seam on every login and slug"
    requirement: LIST-04
    verification:
      - kind: integration
        ref: "internal/github/client_test.go#TestCleanAtSeam"
        status: pass
      - kind: unit
        ref: "internal/github/client_test.go#TestMapperCleansRemoteStrings"
        status: pass
    human_judgment: false
  - id: D5
    description: "The extended discovery query is accepted by GitHub and costs 2 points per page; a team reviewer for an org member (Team branch) was not observable"
    requirement: LIST-04
    verification:
      - kind: other
        ref: "gh api graphql with the full discoveryQuery text, is:pr author:@me is:open: no errors, rateLimit.cost 2"
        status: pass
    human_judgment: true
    rationale: "The viewer is in no org, so the '... on Team { combinedSlug }' branch (RESEARCH A1) could not be observed live; it stays on the UAT list"

duration: 20min
completed: 2026-10-12
status: complete
---

# Phase 3 Plan 04: Merge readiness and reviewers reach every PR from GitHub Summary

**DeriveMerge (state and draft before status, UNKNOWN+CONFLICTING is conflicts) and MergeReviewers (pending wins, hidden teams, bots, detail-only states) fill PR.Merge and PR.Reviewers from an extended 2-point discovery query, proven through the real mapper**

## Performance

- **Duration:** about 20 min
- **Started:** 2026-10-12 (about 22:55Z)
- **Completed:** 2026-10-12T23:16Z
- **Tasks:** 2 (TDD, 2 commits each)
- **Files modified:** 10

## Accomplishments

- Merge readiness: a merged PR reporting UNKNOWN is blank, a closed PR with a stale BLOCKED or DIRTY is blank, and a draft is a dash, because state and draft are read before `mergeStateStatus`. An empty or unrecognised status is a dash, never mergeable.
- Reviewers: `MergeReviewers` orders changes requested, pending, approved, then the detail-only commented and dismissed; within a group by kind (person, team, hidden) and lower-cased name, so refreshes never reshuffle rows. The same inputs reversed give the same output (checked in every table case).
- The 12-PR `blockers.json` fixture covers every merge state; its reviewer data covers re-requests, null and team reviewers, bots, mannequins, enterprise teams, a null review author and the viewer. PR 18 of `status.json` carries a hostile reviewer login and request.
- Live check (read-only, once): the full new `discoveryQuery` text through `gh api graphql` with `q = is:pr author:@me is:open` returned no errors and `rateLimit.cost` 2. Probe output was not committed.

## TDD Gate Compliance

| Task | RED | GREEN |
|------|-----|-------|
| 1 merge readiness | `5b38df8` test: TestDeriveMerge (14 subtests), TestMergeBadgeAndGlyph, TestMergeGlyphPrefix and TestMergeMapping failed on their assertions | `79be1c7` feat |
| 2 reviewers | `10707b1` test: TestMergeReviewers (14 subtests), TestReviewerGlyphs, TestReviewerMapping, TestCleanAtSeam, TestMapperCleansRemoteStrings failed on their assertions | `682ff2b` feat |

RED evidence: Go's `go test` output is not one of the formats `gsd_run check tdd-red-evidence` reads, so the classifier was not run. Semantic assessment instead: each RED commit carried compiling stubs (zero values), so the target tests built, ran and failed on the planned assertions (wrong Merge or Reviewers value, empty badge), not on a build, import or fixture fault. `TestMergeDoesNotSort` passed in RED by design: it asserts that the stub's zero values change nothing and still guards the real values after GREEN. No REFACTOR commit was needed.

## Task Commits

1. **Task 1: Every refresh knows whether each PR can merge**
   - RED `5b38df8` (test)
   - GREEN `79be1c7` (feat)
2. **Task 2: Every refresh knows who approved, who requested changes and who is pending**
   - RED `10707b1` (test)
   - GREEN `682ff2b` (feat)

**Plan metadata:** the commit that adds this file (docs: add summary), then a state and roadmap commit.

## Files Created/Modified

- `internal/domain/merge.go` - Merge enum, DeriveMerge, Badge, Glyph
- `internal/domain/merge_test.go` - DeriveMerge table, badge and glyph pins, glyph-prefix and no-sort tests
- `internal/domain/reviewers.go` - Reviewer types, ReviewerState Glyph and Word, MergeReviewers
- `internal/domain/reviewers_test.go` - MergeReviewers table (with a reversed-input check) and glyph test
- `internal/domain/pr.go` - PR gains Merge and Reviewers
- `internal/domain/domain_test.go` - the Visible input check compares PRs with reflect.DeepEqual (see Deviations)
- `internal/github/client.go` - discovery query fields, wireReview and wireRequest, reviewInputs and requestInputs, mapper wiring
- `internal/github/client_test.go` - TestMergeMapping, TestReviewerMapping, extended clean-at-seam tests
- `internal/github/testdata/blockers.json` - 12-PR discovery fixture
- `internal/github/testdata/status.json` - PR 18 hostile reviewer login and request

## Decisions Made

See `key-decisions` above. The plan was followed; the four decisions fill gaps it left open (bot suffix on reviews, de-duplicating named requests, a typed request with an empty name, and the exact go-gh output for an ESC).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] PR is no longer comparable once it holds a slice**
- **Found during:** Task 2 (adding `PR.Reviewers []Reviewer`)
- **Issue:** `go vet` rejected `slices.Equal(in, states)` in `domain_test.go` (TestVisible, "input is not modified"): `PR does not satisfy comparable`.
- **Fix:** compare with `slices.EqualFunc` and `reflect.DeepEqual`. The test still proves `Visible` leaves its input unchanged. No other package compares PR values.
- **Files modified:** `internal/domain/domain_test.go` (not in the plan's file list)
- **Verification:** `go vet ./...` and `go test ./... -race` pass
- **Committed in:** `10707b1` (Task 2 RED commit)

**2. [Rule 2 - Missing critical] Bot suffix removed from review logins as well as request names**
- **Found during:** Task 2 (MergeReviewers design)
- **Issue:** the plan strips `[bot]` from requests only. A bot that reviewed (login `renovate[bot]`) and was re-requested (stripped to `renovate`) would not match, so the old approval would show beside the pending entry, which the "pending wins" rule forbids.
- **Fix:** strip the suffix on review logins too; test case "bot review and bot request are one pending reviewer".
- **Files modified:** `internal/domain/reviewers.go`, `internal/domain/reviewers_test.go`
- **Committed in:** `10707b1` (test), `682ff2b` (implementation)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical)
**Impact on plan:** both necessary for compile and for the pending-wins prohibition; no scope creep.

## Issues Encountered

None.

## Known Stubs

None. The RED-phase stubs in `merge.go` and `reviewers.go` were replaced in the GREEN commits.

## Threat Flags

None. No new endpoint, auth path or file access; the new query fields are read-only and covered by T-03-08 to T-03-10.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 03-05 can draw the MERGE and REVIEWERS columns from `PR.Merge` (`Badge`, `Glyph`) and `PR.Reviewers` (`State.Glyph`). A `ReviewerHidden` entry has an empty `Name`; the UI renders hidden ones together (RESEARCH: "…team" or "…N teams") and a team as "…@org/slug".
- Plan 03-06 can reuse `wireReview`, `wireRequest`, `reviewInputs`, `requestInputs` and pass `latestReviews` as the third argument of `MergeReviewers`.
- Open for UAT: the `Team` branch (RESEARCH A1) and the one-cell width of `↓` and `⊘` (A5; fallback `⇣`).
- A steady tick now costs about 2 points for discovery plus the failing-check and detail queries (RESEARCH F9).

## Self-Check: PASSED

- Created files exist: `internal/domain/merge.go`, `merge_test.go`, `reviewers.go`, `reviewers_test.go`, `internal/github/testdata/blockers.json`.
- Commits `5b38df8`, `79be1c7`, `10707b1`, `682ff2b` are ancestors of HEAD.
- `go vet ./...` and `go test ./... -count=1 -race` pass; `git diff --exit-code go.mod go.sum` exits 0.
- All acceptance greps print a line (DeriveMerge, `mergeStateStatus mergeable`, `domain.DeriveMerge(`, MergeReviewers, `reviewRequests(first: 20)`, `combinedSlug`, `domain.MergeReviewers(`).

---
*Phase: 03-blocker-detail*
*Completed: 2026-10-12*

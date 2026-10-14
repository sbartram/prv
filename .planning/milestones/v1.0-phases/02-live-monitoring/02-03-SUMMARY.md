---
phase: 02-live-monitoring
plan: 03
subsystem: domain
tags: [go, sort, domain, graphql, attention-buckets]

requires:
  - phase: 02-live-monitoring
    provides: selection tracked by domain.Ref (plan 02-02), so reordered rows keep the cursor on the same PR
provides:
  - "domain.Bucket(pr) 1..9: the D-01 attention bucket of a PR, lifecycle state checked before the draft flag (D-04 as clarified)"
  - "domain.Sort orders by bucket, createdAt newest first, repo, number as an integer; updatedAt is not a key"
  - "domain.PR.CreatedAt, filled from the createdAt field of the discovery query"
affects: [02-04 footer, 02-08 filtered rows (inherit attention order), 02-10 help overlay]

actuals:
  tokens: 7000
  tasks: 2
  commits: 4
plan_head_before: 009edbadea05d650221c08a7025b2dccbc1dc4ed
plan_head_after: 054c62c811a8dd34da4a8d825e0d6431827ef523

tech-stack:
  added: []
  patterns:
    - "Sort keys are server values that polling does not change (bucket inputs, createdAt, repo, number)"
    - "Bucket switches on lifecycle state first, then the draft flag, then CI and review"

key-files:
  created: []
  modified:
    - internal/domain/list.go
    - internal/domain/pr.go
    - internal/domain/domain_test.go
    - internal/github/client.go
    - internal/github/client_test.go
    - internal/github/testdata/search_page1.json
    - internal/github/testdata/search_page2.json

key-decisions:
  - "A merged or closed draft sorts as merged (7) or closed (8); only an open draft is bucket 6 (user clarification of D-04, 2026-10-12)"
  - "Approved with no checks or unknown CI is bucket 5: 'no checks' never counts as ready to merge (RESEARCH A3)"
  - "updatedAt is not a sort key, so comments and pushes cannot swap rows between refreshes (D-02)"

patterns-established:
  - "Fixtures carry createdAt on every PullRequest node; PR_A is failing CI (bucket 1) and PR_BOT is approved + passing (bucket 3)"

requirements-completed: [LIST-08]

coverage:
  - id: D1
    description: "Every PR lands in its D-01 bucket; open drafts in 6, merged and closed drafts with their lifecycle state, approved without checks in 5, unknown state last"
    requirement: LIST-08
    verification:
      - kind: unit
        ref: "internal/domain/domain_test.go#TestBucket"
        status: pass
    human_judgment: false
  - id: D2
    description: "Sort is total and permutation independent: bucket, createdAt newest first at full precision, repo, number as an integer; updatedAt never moves a row; nil, empty and single slices are safe"
    requirement: LIST-08
    verification:
      - kind: unit
        ref: "internal/domain/domain_test.go#TestSort"
        status: pass
    human_judgment: false
  - id: D3
    description: "Fetch returns the 11 fixture PRs in attention order with createdAt decoded, the status fixture never decreases in bucket, and the query requests createdAt"
    requirement: LIST-08
    verification:
      - kind: unit
        ref: "internal/github/client_test.go#TestMembership"
        status: pass
      - kind: unit
        ref: "internal/github/client_test.go#TestFetchAttentionOrder"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-10-12
status: complete
---

# Phase 2 Plan 03: Attention order Summary

**The list is sorted by need for attention (failing CI, changes requested, ready to merge, CI running, other open, open drafts, merged, closed, unknown) with createdAt, repo and integer number as tie-breaks, so the order stays put between refreshes.**

## Performance

- **Duration:** ~6 min
- **Completed:** 2026-10-12T15:40Z
- **Tasks:** 2 (both TDD: RED test commit, then GREEN feature commit)
- **Files modified:** 7 (0 created, 7 modified)

## Accomplishments
- `domain.Bucket` implements D-01 with the 2026-10-12 clarification of D-04: it switches on `pr.State` first (merged 7, closed 8, unknown 9), then `IsDraft` (6), then failing CI (1), changes requested (2), approved and passing (3), CI running (4), else 5.
- `domain.Sort` replaces the Phase 1 state rank: bucket, `CreatedAt` newest first, repo, number as an integer. `stateRank` is gone and `list.go` no longer mentions the last-update field.
- `PR.CreatedAt` is decoded from `createdAt`, now in the discovery query (`state createdAt closedAt`).
- `TestMembership` pins the 11-ref order from the fixtures; `TestFetchAttentionOrder` checks that buckets never decrease for the status fixture and that the request contains `createdAt`.

## Task Commits

1. **Task 1: Bucket**
   - RED `dc8329e` (test): TestBucket failed to build (`undefined: Bucket`)
   - GREEN `93b2b95` (feat)
2. **Task 2: Sort and createdAt**
   - RED `aba369d` (test): TestSort, TestMembership, TestFetchAttentionOrder and the fixtures; failed to build (`unknown field CreatedAt`)
   - GREEN `054c62c` (feat)

**Plan metadata:** committed separately (docs: complete plan)

## Deviations from Plan

None - plan executed exactly as written.

Notes (not deviations): the fixtures were edited with a small script (`json.dumps(indent=1)`), which kept the existing layout; the diff shows only the added `createdAt` lines plus the PR_A and PR_BOT rollup and review changes. The ui goldens did not change (`git diff --name-only -- internal/ui` is empty).

## Issues Encountered
None. Every commit used an explicit pathspec; the pre-staged `.serena` files and the modified `.planning/config.json` stayed out of all commits.

## Known Stubs
None.

## Threat Flags
None beyond the plan's threat model. T-02-13 is mitigated: `TestBucket` pins approved + no checks and approved + unknown CI to bucket 5, and the `TestSort` permutation and updatedAt subtests pin the order keys. T-02-SC: no new modules (go.mod and go.sum unchanged).

## Next Phase Readiness
Ready for 02-04 and the remaining wave. `go build ./...`, `go vet ./...` and `go test ./... -count=1 -race` pass.

## Self-Check: PASSED

Commits `dc8329e`, `93b2b95`, `aba369d`, `054c62c` are ancestors of HEAD. Task acceptance greps pass: `stateRank` 0, `UpdatedAt` in list.go 0, `CreatedAt` in pr.go, `state createdAt closedAt` in client.go, `"createdAt"` in both fixtures, lifecycle switch before the draft check in Bucket, 10 top-level PASS lines in the task 2 verify run, ui untouched.

---
*Phase: 02-live-monitoring*
*Completed: 2026-10-12*

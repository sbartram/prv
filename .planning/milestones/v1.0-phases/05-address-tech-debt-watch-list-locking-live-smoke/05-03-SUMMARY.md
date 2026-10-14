---
phase: 05-address-tech-debt-watch-list-locking-live-smoke
plan: 03
subsystem: testing
tags: [go, live-test, github-graphql, opt-in, uat, docs]

requires:
  - phase: 01-status-at-a-glance
    provides: "github.Client Fetch and the domain mapping (State, Review, CI, Merge)"
  - phase: 04-curated-watch-list
    provides: "WatchSource and the saved-ref lookup path"
provides:
  - "TestLive: opt-in (PRV_LIVE=1) real-Fetch check of 11 pinned public PRs, with drift rules"
  - "TestReviewVerdict: offline proof of the drift and bug outcomes"
  - "TestShowRateWarning: opt-in (PRV_SHOW=1) print of the real low-budget list screen at 80 and 40 columns"
  - "README Development and Platforms sections"
  - "05-UAT.md: 8 pending manual checks for /gsd-verify-work 5"
affects: [verify-work 5, milestone audit live verification debt]

plan_head_before: 40cf48dcd6d93b8332d4216525d2494e0314b485
plan_head_after: 2c3bf5f3f25a7f890a0da94be922a22a8cbce3bc

actuals:
  tokens: 4800
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Env-gated live test: the skip is the first statement, before any token lookup"
    - "Drift versus bug: on a mismatch, read one raw field to tell a GitHub-side change (skip, re-pin) from a prv mapping change (fail)"
    - "Test-only render of a state that cannot be produced live (low API budget)"

key-files:
  created:
    - internal/github/live_test.go
    - internal/ui/show_test.go
    - .planning/phases/05-address-tech-debt-watch-list-locking-live-smoke/05-UAT.md
  modified:
    - README.md

key-decisions:
  - "Test file is package github_test, so the live test uses only the exported API like a real caller"
  - "Every Review assertion is rule-sensitive; none is dropped. A raw reviewDecision query (one point) runs only on a mismatch"
  - "No product code, go.mod or go.sum change"

requirements-completed: [LIST-02, LIST-03, LIST-05, LIST-06, LIST-09, RFSH-04, RFSH-05, FILT-01, FILT-03, NAV-02, NAV-03, CONF-01]

coverage:
  - id: D1
    description: "Plain go test ./... stays offline: TestLive skips before any token lookup, TestShowRateWarning skips and prints nothing"
    verification:
      - kind: unit
        ref: "env -u PRV_LIVE -u PRV_SHOW go test ./... -count=1 -race"
        status: pass
    human_judgment: false
  - id: D2
    description: "State, Review, CI and MERGE of 10 pinned merged or closed public PRs and the not-found row match prv's domain mapping on real github.com responses"
    requirement: LIST-02
    verification:
      - kind: integration
        ref: "PRV_LIVE=1 go test ./internal/github -run '^TestLive$' -count=1 -v (11 of 11 PASS on 2026-10-14)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Review drift is told apart from a prv mapping bug (11 offline cases)"
    verification:
      - kind: unit
        ref: "internal/github/live_test.go#TestReviewVerdict"
        status: pass
    human_judgment: false
  - id: D4
    description: "The real low-budget warning prints at 80 and 40 columns with the yellow header and the faint api footer"
    verification:
      - kind: unit
        ref: "PRV_SHOW=1 go test ./internal/ui -run '^TestShowRateWarning$' -count=1 -v"
        status: pass
    human_judgment: true
    rationale: "Whether the warning is legible on the user's theme and in a tmux or herdr pane is a visual judgement (05-UAT test 2)"
  - id: D5
    description: "Eight manual UAT checks (failure marker, rate warning legibility, filter keys, o in a browser, ? overlay, config edits, badges, MERGE column) are listed for /gsd-verify-work 5"
    verification: []
    human_judgment: true
    rationale: "Real terminal, real browser, real network loss and visual comparison with github.com cannot be automated"

duration: 2min
completed: 2026-10-14
status: complete
---

# Phase 05 Plan 03: Live smoke test and manual UAT checklist Summary

**An opt-in PRV_LIVE=1 test runs prv's real Fetch on 11 pinned public PRs and checks state, review, CI and MERGE (GitHub-side drift becomes a re-pin skip, a prv review-mapping bug still fails), plus a PRV_SHOW=1 render of the real low-budget warning and an 8-check 05-UAT.md.**

## Performance

- **Duration:** about 2 min of wall clock (2026-10-14T03:36Z to 03:41Z)
- **Tasks:** 3
- **Files modified:** 4 (3 created, README.md changed; 0 product files)

## Accomplishments

- `TestLive` (package `github_test`) skips first unless `PRV_LIVE=1`, then resolves the token like prv, builds a real `github.Client` with a `pinSource` Watch source and asserts, per pin, `State`, `Review`, `CI` and `Merge == MergeNone` of the saved row. Pin 11 (`cli/cli#2147483647`) asserts `UnavailNotFound`.
- Drift rules: expected-closed pin now open or merged, a Review mismatch whose raw `reviewDecision` differs from the recorded value (any direction), and expected CI now "no checks" are `t.Skipf` re-pin messages. An unchanged raw value is a failure that says prv's mapping changed. `TestReviewVerdict` proves all three outcomes offline with 11 cases.
- `TestShowRateWarning` (`PRV_SHOW=1`) prints the real list screen (fixtureSix, budget 999 of 5000, ANSI kept) at 80 and 40 columns, after asserting the yellow header warning and the faint footer `api 999`. No `view.go` change.
- README gets `## Development` (the live command, token, cost, re-pin note) and `### Platforms` (macOS and Linux; Windows unsupported because of `flock`). No CI workflow was added.
- `05-UAT.md` lists exactly 8 pending checks (D-16), none of the D-17 deferred ones, mirrored as human-checks in the plan.

## Live run (2026-10-14)

`PRV_LIVE=1 go test ./internal/github -count=1 -v -run '^TestLive$'`: `--- PASS: TestLive (2.97s)` with 11 of 11 subtests passing and no skips, so all pins still match their 2026-10-13 values and no raw reviewDecision query was sent. The reported rate cost is not printed by the test; the plan estimates about 4 points.

## Task Commits

1. **Task 1 (tracer): PRV_LIVE smoke test for one pinned PR** - `332c082` (test)
2. **Task 2: 11 pins, drift rules, TestReviewVerdict** - `05a0e2b` (test)
3. **Task 3: README, TestShowRateWarning, 05-UAT.md** - `2c3bf5f` (docs)

**Plan metadata:** the docs commit that holds this file.

## Files Created/Modified

- `internal/github/live_test.go` - `pinSource`, `livePin`, `livePins` (11), `reviewVerdict`, `rawReviewQuery`, `rawReviewDecision`, `TestLive`, `TestReviewVerdict`
- `internal/ui/show_test.go` - `TestShowRateWarning`
- `README.md` - Development and Platforms sections
- `.planning/phases/05-address-tech-debt-watch-list-locking-live-smoke/05-UAT.md` - 8 pending checks

## Decisions Made

- Followed the plan as written. The tracer gate was satisfied by re-running the tracer verify (offline skip, live pass, full suite) before expanding to the full pin table.
- The task commit for Task 3 uses type `docs` because it holds a README section and a UAT file along with a test file; the plan did not fix a type.

## Deviations from Plan

None - plan executed exactly as written. Two notes:

- The TDD RED/GREEN split does not apply: these tasks add test code only and no product behavior, so each task is one commit. `TestReviewVerdict` was written together with `reviewVerdict` (both in the same test file) and passed on the first run.
- I changed one comment after the first Task 2 run so the file holds no `prometheus/prometheus` text (acceptance grep requires 0). It was in the Task 2 commit.

## Issues Encountered

None.

## Verification

- `env -u PRV_LIVE go test ./internal/github -count=1 -v -run '^(TestLive|TestReviewVerdict)$'`: `--- SKIP: TestLive`, `--- PASS: TestReviewVerdict` with 11 subtests.
- `env -u PRV_SHOW go test ./internal/ui -run '^TestShowRateWarning$' -v`: skip, no frame printed. With `PRV_SHOW=1`: two frames, each with the yellow header warning and the faint `api 999` footer.
- `go vet ./... && go test ./... -count=1 -race` (PRV_LIVE and PRV_SHOW unset): all 6 packages pass.
- `git diff --exit-code go.mod go.sum` exits 0; no `.github/workflows`.
- All acceptance greps for Tasks 1-3 pass (live_test.go pin count 11, no `prometheus/prometheus`, `decision:` 10, `LogIgnoreEnv: true`, no token in any message; 05-UAT.md 8 tests and 8 pending, no D-17 words; show_test.go has `PRV_SHOW` as the first statement and 5 imports).

## Known Stubs

None.

## Threat Flags

None. The live test sends GraphQL queries only (Fetch, and one `reviewDecision` field on a mismatch) and never prints the token. `TestShowRateWarning` reads no token, file or network.

## User Setup Required

None - no external service configuration required. The live test needs a token from `GH_TOKEN`, `GITHUB_TOKEN` or `gh auth login` only when `PRV_LIVE=1` is set.

## Next Phase Readiness

`/gsd-verify-work 5` can walk `05-UAT.md` (8 pending checks). Plan 05-02 is independent and does not touch the helpers `show_test.go` uses (`newTestModel`, `withRate`, `fixtureSix`).

## Self-Check: PASSED

- FOUND: internal/github/live_test.go, internal/ui/show_test.go, README.md, .planning/phases/05-address-tech-debt-watch-list-locking-live-smoke/05-UAT.md
- FOUND commits: 332c082, 05a0e2b, 2c3bf5f

---
*Phase: 05-address-tech-debt-watch-list-locking-live-smoke*
*Completed: 2026-10-14*

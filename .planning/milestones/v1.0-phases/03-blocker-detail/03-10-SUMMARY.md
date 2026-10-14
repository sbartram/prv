---
phase: 03-blocker-detail
plan: 10
subsystem: detail-view
tags: [go, bubbletea-v2, detail-view, rate-limit, backoff, gap-closure, tdd]

requires:
  - phase: 03-blocker-detail
    provides: "plan 03-08 detail pause (pauseForRateLimit), collapsed check list with fold line, CountChecks"
provides:
  - "domain.ServerWait: the pause after a rate-limited response, clamped to [MinServerWait, MaxServerWait], 60 s floor without timing"
  - "pauseForRateLimit uses ServerWait: the detail path and the list path share one clamp"
  - "CheckCounts.Other: unknown (CheckOther) checks are counted apart from skipped"
  - "Collapsed detail view always lists unknown checks (wide and compact); fold line reads '+N more passing or skipped'; summary shows 'N unknown'"
affects: [phase-03-verification]

plan_head_before: 7c5898f2787efbcd45b600945df81a85bc68e187
plan_head_after: c1012211a1614fb1f4dc9a0b1a4d2243169c5a91

actuals:
  tokens: 5400
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "One server-wait policy: the detail path calls domain.ServerWait, which reuses NextDelay's private clamp and resetWait, and TestServerWait cross-checks it against NextDelay"
    - "A state prv cannot classify is listed and counted as its own class, never folded into passing or skipped"

key-files:
  created: []
  modified:
    - internal/domain/backoff.go
    - internal/domain/backoff_test.go
    - internal/domain/checks.go
    - internal/domain/checks_test.go
    - internal/ui/detail.go
    - internal/ui/detail_test.go
    - internal/ui/testdata/TestGolden/detail_0x0.golden
    - internal/ui/testdata/TestGolden/detail_60x15.golden
    - internal/ui/testdata/TestGolden/detail_80x24.golden
    - internal/ui/testdata/TestGolden/detail_120x40.golden
    - internal/ui/testdata/TestGolden/detail-error_0x0.golden
    - internal/ui/testdata/TestGolden/detail-error_60x15.golden
    - internal/ui/testdata/TestGolden/detail-error_80x24.golden
    - internal/ui/testdata/TestGolden/detail-error_120x40.golden
    - internal/ui/testdata/TestGolden/detail-gone_0x0.golden
    - internal/ui/testdata/TestGolden/detail-gone_60x15.golden
    - internal/ui/testdata/TestGolden/detail-gone_80x24.golden
    - internal/ui/testdata/TestGolden/detail-gone_120x40.golden

key-decisions:
  - "ServerWait goes straight to RateLimitFloor (60 s) when there is no Retry-After and no reset after now; it does not fall back to the snapshot's reset time, so it stays a pure function of the response"
  - "Retry-After now wins over a reset time on the detail path, as in NextDelay"
  - "The detail pause ends at reset + 1 s (the list path's margin), a deliberate change to two existing expectations"
  - "Fold line wording follows the user's UAT decision ('+N more passing or skipped'), superseding the D-10 text"

requirements-completed: [DETL-01, RFSH-05]

coverage:
  - id: D1
    description: "A rate-limited detail error pauses refreshes only within the list path's clamp (1 s to 65 min); a hostile Retry-After or far-future reset cannot freeze refreshes for years, and a past or equal reset pauses for 60 s, never zero"
    requirement: RFSH-05
    verification:
      - kind: unit
        ref: "internal/domain/backoff_test.go#TestServerWait"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailRateLimitExtendsPause"
        status: pass
    human_judgment: false
  - id: D2
    description: "Unknown (CheckOther) checks are listed in the collapsed detail at every width, counted as 'N unknown', and the fold line says 'more passing or skipped'"
    requirement: DETL-01
    verification:
      - kind: unit
        ref: "internal/domain/checks_test.go#TestCountChecks"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailFold"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestGolden"
        status: pass
    human_judgment: false

duration: 4min
completed: 2026-10-13
status: complete
---

# Phase 3 Plan 10: The detail pause stays bounded, and unknown checks are never shown as passing Summary

**`domain.ServerWait` gives the detail view's rate-limit pause the list path's 1 s to 65 min clamp, and unknown checks are now listed, counted as "unknown", and kept out of the "more passing or skipped" fold line.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-10-13T01:38:50Z
- **Completed:** 2026-10-13T01:42:00Z
- **Tasks:** 2 (both TDD: RED then GREEN)
- **Files modified:** 18 (6 Go files, 12 goldens)

## Accomplishments

- G-03-6-WR01: `pauseForRateLimit` pauses for `now + domain.ServerWait(now, after, reset)`. Retry-After `2147483647 s` or a reset ten years ahead pauses for exactly 65 minutes; a reset in the past or equal to now pauses for 60 s; a 1 ms Retry-After pauses for 1 s. The pause still never moves earlier, and a non-rate-limit error pauses nothing.
- `TestServerWait` proves for ten input pairs that `ServerWait` stays in `[MinServerWait, MaxServerWait]` and equals `NextDelay(...).Wait` for the same rate-limited outcome, so the two paths cannot drift apart.
- G-03-6-WR04: `CountChecks` counts `CheckOther` in the new `CheckCounts.Other`; `Neutral` counts only `CheckNeutral`. The CHECKS summary adds "N unknown" after "skipped".
- The collapsed view lists unknown checks with their "?" glyph after the running checks (wide and compact branch); the fold line reads "… +N more passing or skipped".
- Exactly 12 detail goldens changed, one line each (the fold line); no golden file was added.

## Task Commits

1. **Task 1: clamped detail rate-limit pause**
   - RED: `5e0be4a` (test) - `TestServerWait` (compile error: `ServerWait` undefined) and 7 failing `TestDetailRateLimitExtendsPause` subtests (2 updated, 6 new, assertion failures on `pausedUntil`)
   - GREEN: `c867fcb` (fix) - `ServerWait` and the new `pauseForRateLimit`
2. **Task 2: list unknown checks, honest fold label**
   - RED: `d415d1a` (test) - `CheckCounts.Other` undefined (domain compile error); `TestDetailFold` "an unknown check is listed, never folded" and the 12-golden fold label failing in ui
   - GREEN: `c101221` (fix) - `CheckCounts.Other`, `checkLines`, `appendFold`, `checksSummary`, 12 regenerated goldens

**Plan metadata:** committed in the docs(03-10) commit that follows this summary.

## Files Created/Modified

- `internal/domain/backoff.go` - exported `ServerWait`
- `internal/domain/backoff_test.go` - `TestServerWait`
- `internal/domain/checks.go` - `CheckCounts.Other`; `CountChecks` counts `CheckNeutral` and `CheckOther` apart
- `internal/domain/checks_test.go` - `TestCountChecks` expects `Neutral: 1, Other: 1`
- `internal/ui/detail.go` - `pauseForRateLimit` via `ServerWait`; `checkLines` lists `CheckOther`; `appendFold` label; `checksSummary` "unknown" segment
- `internal/ui/detail_test.go` - six new and two updated `TestDetailRateLimitExtendsPause` subtests; `TestDetailFold` unknown-check subtest and new fold label
- `internal/ui/testdata/TestGolden/{detail,detail-error,detail-gone}_{0x0,60x15,80x24,120x40}.golden` - fold line only

## TDD Gate Compliance

- RED (`test(03-10)`) precedes GREEN (`fix(03-10)`) for both tasks.
- RED evidence: in both tasks the domain package fails to build (`ServerWait` undefined; `CheckCounts.Other` unknown field), which the plan names as RED. The ui subtests were also seen failing on the planned assertions before the fix (Task 1: 8 `TestDetailRateLimitExtendsPause` subtests on `pausedUntil`; Task 2: `TestDetailFold` and `TestGolden`).
- Semantic assessment: the failures are on the planned behaviors (clamp values, listing of the unknown check, fold label), not on setup or syntax faults. Go's `testing` output has no format-based adapter in the RED classifier, so `gsd_run check tdd-red-evidence` was not run; the assessment above is by inspection of the real failure output.
- No REFACTOR commit was needed.

## Decisions Made

See `key-decisions` above. The plan's flagged decisions were followed as written.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Known Stubs

None.

## Threat Flags

None. T-03-26, T-03-27 and T-03-28 are mitigated as planned; no new network, auth or schema surface; `go.mod` and `go.sum` are unchanged (T-03-SC).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Gaps G-03-6-WR01 and G-03-6-WR04 are closed. WR-03 stays deferred and IN-01..IN-05 stay open, as the plan states.
- Ready for phase verification.

## Self-Check: PASSED

- Files exist: `internal/domain/backoff.go`, `internal/domain/checks.go`, `internal/ui/detail.go` and the test files (found).
- Commits `5e0be4a`, `c867fcb`, `d415d1a`, `c101221` are ancestors of HEAD.
- `go vet ./...` and `go test ./... -count=1 -race` pass; `git diff --exit-code go.mod go.sum` is clean; `git show --numstat --format= c101221 -- internal/ui/testdata` lists the 12 files at `1 1`; no untracked file under `internal/ui/testdata`.

---
*Phase: 03-blocker-detail*
*Completed: 2026-10-13*

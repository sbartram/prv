---
phase: 02-live-monitoring
plan: 06
subsystem: ui
tags: [go, rate-limit, backoff, simulated-clock, bubbletea-v2]

requires:
  - phase: 02-live-monitoring
    provides: refresh chain with one tick per current-generation result (02-01), header fit() and footer composer (02-04), RateLimit with Limit/Cost/Remaining/ResetAt (02-05)
provides:
  - "domain.NextDelay(now, interval, failures, rl, last) Delay: one pure policy for every refresh delay (retry-after, then reset time, then bounded exponential; reserve pause and pacing on success)"
  - "Model pause state: pausedUntil gates r and the tick; failures counts consecutive failures; a flash message with its own expiry"
  - "Header segment 'paused until HH:MM (rate limit)' (prio 7), error panel line in place of 'retrying at', footer flash 'rate limited — resumes HH:MM' for 4 s"
  - "backoffer interface in ui (Backoff() (time.Duration, time.Time, bool)) that plan 02-07 gives *github.Error"
  - "Virtual-clock harness (sim, scriptFetcher) and TestSimulatedHour with six scenarios"
affects: [02-07 real GitHub rate-limit errors feed Backoff(), 02-08 filter segment takes header prio 3, 02-10 footer]

actuals:
  tokens: 14000
  tasks: 2
  commits: 4
plan_head_before: eeffd14a9e96de2e909412a71195ef65862effce
plan_head_after: 19eaa892a608fb0b9a4eb92eb3177a56c0ef4dd5

tech-stack:
  added: []
  patterns:
    - "Pure policy function with the clock as a parameter (like InWindow); the model calls it once per current-generation result"
    - "Exponential backoff as a loop that stops at its cap, never a shift; durations are only divided, never multiplied, in pacing"
    - "Simulated hour: model.after is replaced by a time-ordered queue, a scripted counting fetcher records virtual call times, no real waiting"

key-files:
  created:
    - internal/domain/backoff.go
    - internal/domain/backoff_test.go
    - internal/ui/ratelimit_test.go
  modified:
    - internal/ui/model.go
    - internal/ui/refresh.go
    - internal/ui/view.go
    - internal/ui/view_test.go

key-decisions:
  - "A tick of the current generation that fires before pausedUntil re-arms for the remaining time instead of fetching, so the no-request-during-a-pause rule does not depend on timer precision"
  - "r during a pause does not bump the generation: the pending tick stays current and the only command is the 4 s redraw timer"
  - "schedule() lives in model.go beside the fetchedMsg handler; setFlash and redrawMsg live in refresh.go"
  - "In the error panel (no footer) the refusal is shown by the panel's own 'paused until HH:MM (rate limit)' line, so a refused r still says when refreshing resumes"

patterns-established:
  - "Pause is decided only from the data of the last result (m.snap.RateLimit stays through failures), so a failure with no timing information still uses the last known reset time"

requirements-completed: [RFSH-05, RFSH-02]

coverage:
  - id: D1
    description: "NextDelay picks retry-after, then reset time, then a bounded exponential; reserve pause at Remaining <= 10*max(Cost,1); pacing never below the interval; every server delay clamped to [1 s, 65 min]; 1000 and 1<<30 failures give 5 min without overflow"
    requirement: RFSH-05
    verification:
      - kind: unit
        ref: "internal/domain/backoff_test.go#TestNextDelay"
        status: pass
      - kind: unit
        ref: "internal/domain/backoff_test.go#TestNextDelayNoOverflow"
        status: pass
    human_judgment: false
  - id: D2
    description: "Over a simulated hour: 59-61 calls at 60 s; failure gaps 60/60/120/240/300/60 s; no call before reset+1 s and the next call exactly then; retry-after 90 s and a hostile 100 h (65 min) honoured; with r every 7 s exactly one current-generation tick is pending after every step and no stale tick starts a fetch"
    requirement: RFSH-05
    verification:
      - kind: unit
        ref: "internal/ui/ratelimit_test.go#TestSimulatedHour"
        status: pass
    human_judgment: false
  - id: D3
    description: "During a pause r sends no request and shows 'rate limited — resumes HH:MM' in the footer for about 4 s; the header shows 'paused until HH:MM (rate limit)'; the flash leaves after expiry"
    requirement: RFSH-05
    verification:
      - kind: unit
        ref: "internal/ui/ratelimit_test.go#TestPauseRefusesRefresh"
        status: pass
      - kind: unit
        ref: "internal/ui/ratelimit_test.go#TestPausedHeader"
        status: pass
      - kind: unit
        ref: "internal/ui/ratelimit_test.go#TestFlashExpires"
        status: pass
    human_judgment: false
  - id: D4
    description: "A failed first fetch retries on the backoff schedule; the error panel says 'retrying at HH:MM' for a plain failure and 'paused until HH:MM (rate limit)' during a rate-limit pause; r retries at once unless a pause blocks it"
    requirement: RFSH-02
    verification:
      - kind: unit
        ref: "internal/ui/ratelimit_test.go#TestErrorPanelPaused"
        status: pass
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestFirstFetchFailureRetries"
        status: pass
    human_judgment: false
  - id: D5
    description: "Real GitHub rate-limit responses pause prv end to end"
    requirement: RFSH-05
    verification: []
    human_judgment: true
    rationale: "Until plan 02-07 gives *github.Error a Backoff() method, only test doubles carry retry-after and reset times; the reserve pause and pacing already work on real rateLimit data. TestRateLimitedEndToEnd in 02-07 covers the rest."

duration: 6min
completed: 2026-10-12
status: complete
---

# Phase 2 Plan 06: Automatic backoff, visible pause and refused r during a pause Summary

**One pure `domain.NextDelay` decides every refresh delay (Retry-After, then reset time, then a bounded exponential, plus a reserve pause and pacing), and the model uses it after every result, shows "paused until HH:MM (rate limit)", and refuses `r` with a "rate limited — resumes HH:MM" flash while a pause is active.**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-10-12T15:53:40Z (approx.)
- **Completed:** 2026-10-12T15:59:13Z (code); SUMMARY and state follow
- **Tasks:** 2 (both TDD: RED test commit, then GREEN feature commit)
- **Files modified:** 7 (3 new)

## Accomplishments
- `domain/backoff.go`: `NextDelay` with exported tunables (`MinServerWait`, `MaxServerWait`, `MinFailureBase`, `MaxFailureWait`, `RateLimitFloor`, `ReservePerCost`). Pure, clock passed in, no `time.Now`. The exponential is a loop that stops at its cap; pacing only divides.
- `ui/model.go`: `schedule(err)` runs after every current-generation result, fills a `domain.Outcome` from the optional `Backoff()` method, sets `pausedUntil` and arms the one tick with `Wait`. Both fixed-interval arms are gone. The Refresh key goes inFlight, then pause refusal, then the old path. A tick that fires inside a pause re-arms.
- `ui/view.go`: prio 7 header segment, footer flash first among the notes (cut last), error panel pause line.
- `ui/ratelimit_test.go`: virtual clock queue plus scripted counting fetcher. `TestSimulatedHour` covers steady (61 calls), server-errors (gaps 60, 60, 120, 240, 300, 60 s), exhausted (second call exactly at reset+1 s, r at 5 min makes no call), retry-after (90 s), hostile-retry-after (65 min) and presses (r every 7 s, one current tick after every step).

## Task Commits

1. **Task 1: NextDelay policy**
   - RED `72c5439` (test): build failed, `Outcome` and `NextDelay` undefined
   - GREEN `1751826` (feat)
2. **Task 2: pause, refusal, header, simulated hour**
   - RED `49e1308` (test): build failed on `redrawMsg`, `pausedUntil`
   - GREEN `19eaa89` (feat)

**Plan metadata:** committed separately (docs: complete plan)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Test conflict] Narrow-pane rate warning test used a fixture that now pauses**
- **Found during:** Task 2
- **Issue:** `TestRateWarning/narrow pane keeps the warning over the count` (02-05) used `withRate(5000, 10)`. Ten left is at the reserve (`Remaining <= 10*Cost`), so the new pause segment (prio 7) outranks the warning (prio 6) at 45 cells. That is the intended behaviour of this plan, not a bug in it.
- **Fix:** The test now uses `withRate(5000, 100)` (under 20%, above the reserve), which keeps its intent: the warning wins over the PR count.
- **Files modified:** internal/ui/view_test.go
- **Commit:** 19eaa89

**2. [Rule 3 - Tooling] `interface{ Backoff() ... }` acceptance grep cannot match gofmt output**
- **Found during:** Task 2
- **Issue:** The acceptance grep expects `interface{ Backoff() (time.Duration, time.Time, bool) }` on one line. gofmt always expands an interface with a method to three lines.
- **Fix:** Kept the gofmt form. The same contract is present (`type backoffer interface { Backoff() (time.Duration, time.Time, bool) }` over three lines).
- **Files modified:** internal/ui/model.go
- **Commit:** 19eaa89

**3. [Rule 2 - Safety] Tick that fires inside a pause**
- **Found during:** Task 2
- **Issue:** The prohibition says no request while paused. A tick of the current generation could in principle arrive before `pausedUntil` (timer skew), and the plan only gated `r`.
- **Fix:** The `tickMsg` handler re-arms for `pausedUntil - now` instead of fetching. At equality the tick fetches, so calls resume exactly at the pause end.
- **Files modified:** internal/ui/model.go
- **Commit:** 19eaa89

**Total deviations:** 3 auto-fixed (1 test fixture, 1 tooling, 1 safety). **Impact:** none on scope; no new files beyond the plan's list.

Notes (not deviations): the plan's NextDelay bullet "Remaining 11 → Wait 60 s" is only true when pacing does not apply. With the same ResetAt T+20m that bullet uses for Remaining 10, rule (d) stretches the delay to 20 min / 11 (about 109 s) with no pause. The table tests both: Remaining 11 with ResetAt T+10m gives 60 s, and Remaining 11 with ResetAt T+20m gives 20 min / 11 and no pause. The rules in the Action text were followed. The plan's behaviour bullet says r returns "only a redraw timer"; in the error panel there is no footer, so the panel line carries the resume time (see key-decisions).

## Issues Encountered
None. Every commit used an explicit pathspec; the staged `.serena` files and the modified `.planning/config.json` stayed out of all commits.

## Known Stubs
None.

## Threat Flags
None beyond the plan's threat model. T-02-08 mitigated: server delays clamped to [1 s, 65 min] (`TestNextDelay` rows for 1 ms, 100 h, a reset a year away), exponential stops at its cap (`TestNextDelayNoOverflow`, hostile-retry-after simulation). T-02-09 mitigated: pause until retry-after or reset, `r` refused during the pause, `TestSimulatedHour` fails on any call started before `pausedUntil`. T-02-SC: no new modules (`go.mod` and `go.sum` unchanged).

## Next Phase Readiness
Ready for 02-07 (give `*github.Error` the `Backoff()` method and the header-capturing transport) and 02-08. `go build ./...`, `go vet ./...` and `go test ./... -count=1 -race` pass.

## Self-Check: PASSED

Commits `72c5439`, `1751826`, `49e1308`, `19eaa89` are ancestors of HEAD. `backoff.go`, `backoff_test.go` and `ratelimit_test.go` exist. Acceptance checks: `func NextDelay(now time.Time, interval time.Duration, failures int, rl RateLimit, last Outcome) Delay` is present; `time.Now` count in backoff.go is 0; `domain.NextDelay(` is in model.go; `m.arm(m.interval)` count in model.go is 0; `time.Sleep` count in ratelimit_test.go is 0; the Backoff contract is present (multi-line, see deviation 2); no change under `internal/ui/testdata`, `go.mod` or `go.sum`. All six `TestSimulatedHour` subtests report PASS under `-race`; plan-level `go vet ./... && go test ./... -count=1 -race` passes.

---
*Phase: 02-live-monitoring*
*Completed: 2026-10-12*

---
phase: 02-live-monitoring
plan: 01
subsystem: ui
tags: [go, bubbletea-v2, refresh, single-flight, toml, config, tracer]

requires:
  - phase: 01-status-at-a-glance
    provides: ui.Model with Fetcher seam, github.Client, error panel, test helpers
provides:
  - internal/config: refresh_interval from config.toml (Go duration, 15 s floor, 60 s default, 1 MiB cap)
  - Generation-stamped single-flight tick chain in ui (tickMsg, fetchedMsg{gen}, fetch, beginFetch, arm)
  - Failed refresh keeps rows on screen; failed first fetch keeps the panel and retries on the tick
affects: [02-02 r from every screen and updated time, 02-04 failure marker and retrying-at line, 02-09 config error text]

actuals:
  tokens: 7350
  tasks: 2
  commits: 3
plan_head_before: 368338571944d5e0399aaf425ea8f84a6c0edbf0
plan_head_after: 75d5b2d5ab7bc72829173cb40433fdf0504be09d

tech-stack:
  added: [github.com/BurntSushi/toml v1.6.0]
  patterns:
    - "One tick armed only from a current-generation fetch result; Init arms nothing"
    - "fetch Cmd closure captures ctx, fetcher and generation only, never the Model"
    - "Model.after and Options.Now are test seams; tests record armed timers instead of waiting on them"
    - "Config decoded through a private TextUnmarshaler so an integer is never read as nanoseconds"

key-files:
  created:
    - internal/config/config.go
    - internal/config/config_test.go
    - internal/ui/refresh.go
    - internal/ui/refresh_test.go
  modified:
    - go.mod
    - go.sum
    - cmd/prv/main.go
    - cmd/prv/main_test.go
    - internal/ui/model.go

key-decisions:
  - "Generation numbering starts at 0 in New(); the first fetch carries generation 0, so Phase 1 test literals fetchedMsg{snap: ...} stay valid"
  - "The 15 s floor lives in config.Load, not in the model, so model tests can use millisecond intervals"
  - "A config path that cannot be resolved (no home directory) keeps defaults instead of failing"

patterns-established:
  - "Config errors are 'prv: <path>: <reason>' on stderr with exit 2, raised before the token lookup; --version never reads a broken config"
  - "Tests that call run() must call useConfig(t, \"\") so they never read the developer's real config"

requirements-completed: [RFSH-01, RFSH-04, CONF-01]

coverage:
  - id: D1
    description: "refresh_interval from config.toml reaches the model (default 60 s, floor 15 s, Go-duration strings only); missing or empty file gives defaults"
    requirement: RFSH-01
    verification:
      - kind: unit
        ref: "internal/config/config_test.go#TestLoadInterval"
        status: pass
      - kind: unit
        ref: "internal/config/config_test.go#TestPath"
        status: pass
    human_judgment: false
  - id: D2
    description: "Single-flight tick chain: one tick per current-generation result, ticks during a fetch and stale ticks dropped, Init arms none"
    requirement: RFSH-01
    verification:
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestAutoRefreshTickChain"
        status: pass
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestTickWhileInFlightDropped"
        status: pass
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestStaleTickDropped"
        status: pass
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestInitArmsNoTick"
        status: pass
    human_judgment: false
  - id: D3
    description: "Real Bubble Tea runtime over the real github client keeps re-fetching and quits cleanly"
    requirement: RFSH-01
    verification:
      - kind: e2e
        ref: "cmd/prv/main_test.go#TestRefreshEndToEnd"
        status: pass
    human_judgment: false
  - id: D4
    description: "A broken refresh_interval exits 2 with a 'prv: <path>' line before the token lookup; --version ignores it"
    requirement: CONF-01
    verification:
      - kind: unit
        ref: "cmd/prv/main_test.go#TestBrokenConfigExitsBeforeToken"
        status: pass
      - kind: unit
        ref: "cmd/prv/main_test.go#TestVersionIgnoresBrokenConfig"
        status: pass
    human_judgment: false
  - id: D5
    description: "A failed refresh with data on screen keeps the rows undimmed and shows no error panel; a failed first fetch keeps the panel and retries on the tick; stale-generation results are dropped"
    requirement: RFSH-04
    verification:
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestFailedRefreshKeepsData"
        status: pass
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestFirstFetchFailureRetries"
        status: pass
      - kind: unit
        ref: "internal/ui/refresh_test.go#TestStaleResultDropped"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-10-12
status: complete
---

# Phase 2 Plan 01: Refresh tracer Summary

**prv re-fetches itself on a `refresh_interval` from config.toml (60 s default, 15 s floor) through one generation-stamped tick chain, and a failed refresh never hides the rows on screen.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-10-12T15:30Z
- **Tasks:** 2
- **Files modified:** 9 (4 created, 5 modified)

## Accomplishments
- `internal/config` reads `refresh_interval` as a Go duration string; the integer `60` and `"soon"` are errors, values under 15 s are raised to 15 s, a missing or empty file gives 60 s. Path is `$XDG_CONFIG_HOME/prv/config.toml` (absolute only) else `~/.config/prv/config.toml`, 1 MiB cap.
- `run()` loads the config after `--version` and the `--closed-days` check but before the token lookup; a broken file prints `prv: <path>: ...` and exits 2.
- The model arms exactly one tick, only from a current-generation fetch result (success or failure); a tick that arrives mid-fetch or from another generation does nothing; Init arms no timer; `r` on the error screen bumps the generation.
- A failed refresh with data shown keeps rows and state (no dimming, no panel) and records `failedAt`; a failed first fetch keeps the Phase 1 panel while the tick retries on its own.
- `TestRefreshEndToEnd` runs a real `tea.Program` over the real github client and a counting transport at a 20 ms interval, then quits cleanly under `-race`.

## Task Commits

1. **Task 1 (tracer): config to cmd to ui to github client refresh path** - `ce62a25` (feat)
2. **Task 2 (TDD): failed refresh keeps rows, first-fetch retry, stale result dropped**
   - RED `0d6e8f7` (test): `TestFailedRefreshKeepsData` failed as expected
   - GREEN `75d5b2d` (feat)

**Plan metadata:** committed separately (docs: complete plan)

## Files Created/Modified
- `internal/config/config.go` - path resolution, TOML load, interval floor, size cap
- `internal/config/config_test.go` - `TestPath`, `TestLoadInterval`
- `internal/ui/refresh.go` - `tickMsg`, `fetchedMsg{gen}`, `fetch`, `afterTick`, `beginFetch`, `arm`
- `internal/ui/refresh_test.go` - tick-chain and failed-refresh tests, `afterRec`, `result`
- `internal/ui/model.go` - refresh state machine; `loading` field and `fetchCmd` removed
- `cmd/prv/main.go` - `configPath` seam, config load before token, `Interval` passed to the UI
- `cmd/prv/main_test.go` - `useConfig`, `countingTransport`, three new tests
- `go.mod`, `go.sum` - `github.com/BurntSushi/toml v1.6.0`

## Decisions Made
- Generation starts at 0 in `New()` (plan decision) so Phase 1 test literals still work.
- The interval floor is in `config.Load`, not the model.
- If the config path cannot be resolved (no home directory), prv keeps the defaults.
- Package legitimacy: the exact module path and version prescribed in `.claude/CLAUDE.md` was installed with the checksum database on; `go mod verify` passes. No human checkpoint was raised, as the plan decided.

## Deviations from Plan

### Auto-fixed Issues

**1. [Process] Task 1 commit first swept in pre-staged `.serena` files**
- **Found during:** Task 1 commit
- **Issue:** `git commit` committed the whole index, including the user's pre-staged `.serena/.gitignore` and `.serena/project.yml`.
- **Fix:** `git reset --soft HEAD~1` (no working-tree change), then recommitted with explicit paths (`git commit -- <paths>`). The `.serena` files are still staged and uncommitted, as before. All later commits used explicit paths.
- **Committed in:** `ce62a25`

**2. [Rule 1 - Acceptance] Doc comment contained the string `UserConfigDir`**
- **Found during:** Task 1 acceptance check (`grep -c UserConfigDir` must print 0)
- **Fix:** reworded the comment in `internal/config/config.go`.

---

**Total deviations:** 2 (1 process, 1 acceptance wording). **Impact on plan:** none on behaviour.

## Issues Encountered
None. The `r` key, spinner and view code were left as in Phase 1, as the plan says; the spinner tick returned by `beginFetch` during an auto-refresh in the list state is delivered once and then ignored (spinner only runs in the loading state).

## Known Stubs
None. `failedAt` and `nextAt` are set but not yet rendered; plans 02-02 and 02-04 consume them (stated in the plan).

## Threat Flags
None beyond the plan's threat model (T-02-01, T-02-02, T-02-03, T-02-06, T-02-SC all mitigated as listed).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
Ready for 02-02 (keymap, `r` from every screen, updated time, selection by ref) and 02-04 (failure marker, retrying-at line). `go build ./...`, `go vet ./...`, `go mod verify` and `go test ./... -race` pass; no golden files changed.

## Self-Check: PASSED

All four created files exist; commits `ce62a25`, `0d6e8f7`, `75d5b2d` are ancestors of HEAD.

---
*Phase: 02-live-monitoring*
*Completed: 2026-10-12*

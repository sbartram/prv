---
phase: 02-live-monitoring
plan: 09
subsystem: config
tags: [go, toml, config, flags, bubbletea-v2]

requires:
  - phase: 02-live-monitoring
    provides: "internal/config with refresh_interval (02-01); ui.Options.Show, domain.StateSet and domain.Visible (02-08); flash and redrawMsg footer machinery (02-04)"
provides:
  - "config.toml keys closed_days (clamped 1..365) and show (open/merged/closed), with defaults 14 and all three states"
  - "D-15 errors 'prv: <path> line N: <problem>' for syntax errors, wrong types, an integer refresh_interval and an unknown show value"
  - "Unknown-key and empty-show warnings, cleaned with domain.Clean"
  - "effectiveClosedDays: --closed-days flag beats config beats default 14; flag range 1..365 (WR-01)"
  - "Config warnings shown for 8 s from the first fetch result in their own footer segment and on the first-fetch error panel"
affects: [02-10]

actuals:
  tokens: 6500
  tasks: 2
  commits: 4
plan_head_before: fc00bae2b546118c24be4e54ee7b716df6b392b8
plan_head_after: 6d4e77e53d5d8ce4670ab4b11748fbbe5cb062ed

tech-stack:
  added: []
  patterns:
    - "TOML decode into a pointer-field shape so an absent key keeps its default; TextUnmarshaler element types give line numbers for free"
    - "toml wrong-type errors are plain errors: the line number is recovered by regexp on err.Error()"
    - "Warnings have their own Model fields and footer segment; the 8 s clock starts at the first fetch result and a redrawMsg is armed through the injectable after()"

key-files:
  created: []
  modified:
    - internal/config/config.go
    - internal/config/config_test.go
    - cmd/prv/main.go
    - cmd/prv/main_test.go
    - internal/ui/model.go
    - internal/ui/model_test.go
    - internal/ui/view.go

key-decisions:
  - "Config values are clamped, flag values are rejected, both inside 1..365 (D-14, D-16, RESEARCH A6)"
  - "show = [] means all three states plus a warning (RESEARCH A8)"
  - "Footer priority: flash and integrity notes first, then the config warning (cut with an ellipsis, left out under 12 cells), then key hints; the api segment stays at the right edge"
  - "Config.Warnings reach the model through ui.Options.Warnings; the warning clock starts on the first current-generation fetch result, success or failure"

patterns-established:
  - "config.Load is called from cmd/prv only; the config is read once before the TUI starts"

requirements-completed: [CONF-01, FILT-02]

coverage:
  - id: D1
    description: "closed_days and show load with defaults and clamping; show order and duplicates do not matter; empty show warns and shows all states"
    requirement: CONF-01
    verification:
      - kind: unit
        ref: "internal/config/config_test.go#TestLoadAllKeys"
        status: pass
      - kind: unit
        ref: "internal/config/config_test.go#TestClampClosedDays"
        status: pass
      - kind: unit
        ref: "internal/config/config_test.go#TestShowValues"
        status: pass
    human_judgment: false
  - id: D2
    description: "A bad config stops prv with 'prv: <path> line N: <problem>' before the token lookup; unknown keys warn and have terminal escapes removed"
    requirement: CONF-01
    verification:
      - kind: unit
        ref: "internal/config/config_test.go#TestLoadErrors"
        status: pass
      - kind: unit
        ref: "internal/config/config_test.go#TestUnknownKeys"
        status: pass
      - kind: unit
        ref: "internal/config/config_test.go#TestUnknownKeyCleaned"
        status: pass
      - kind: unit
        ref: "cmd/prv/main_test.go#TestConfigErrorFormat"
        status: pass
      - kind: unit
        ref: "cmd/prv/main_test.go#TestBrokenConfigExitsBeforeToken"
        status: pass
    human_judgment: false
  - id: D3
    description: "--closed-days beats closed_days beats 14; the flag accepts 1..365 only; the configured window reaches the search date and the exact filter"
    requirement: FILT-02
    verification:
      - kind: unit
        ref: "cmd/prv/main_test.go#TestHiddenClosedDaysFlag"
        status: pass
      - kind: unit
        ref: "cmd/prv/main_test.go#TestEffectiveClosedDays"
        status: pass
      - kind: unit
        ref: "cmd/prv/main_test.go#TestClosedWindowFromConfig"
        status: pass
      - kind: unit
        ref: "cmd/prv/main_test.go#TestDisplayPath"
        status: pass
    human_judgment: false
  - id: D4
    description: "Config warnings show for exactly 8 s from the first fetch result, never hide a flash or an integrity note, and also show on the first-fetch error panel"
    requirement: CONF-01
    verification:
      - kind: unit
        ref: "internal/ui/model_test.go#TestConfigWarningLifetime"
        status: pass
      - kind: unit
        ref: "internal/ui/model_test.go#TestConfigWarningKeepsFlash"
        status: pass
      - kind: unit
        ref: "internal/ui/model_test.go#TestConfigWarningOnErrorPanel"
        status: pass
    human_judgment: false
  - id: D5
    description: "The show key sets the states visible at launch"
    requirement: CONF-01
    verification:
      - kind: unit
        ref: "internal/config/config_test.go#TestLoadAllKeys"
        status: pass
    human_judgment: true
    rationale: "The wiring cfg.Show to ui.Options.Show is plain assignment in run(); no test drives run() through to the TUI header. Plan 02-08 tests the model side (TestDefaultShowFromOptions); a live check of show in a real config file belongs in end-of-phase UAT"

duration: 18min
completed: 2026-10-12
status: complete
---

# Phase 02 Plan 09: Settings come from config.toml Summary

**config.toml now sets refresh_interval, closed_days (clamped 1..365) and show; a bad file stops prv with "prv: <path> line N: <problem>", the --closed-days flag beats the file inside the same bound, and unknown keys show as a bounded 8 s footer warning that never hides a flash.**

## Performance

- **Duration:** about 18 min
- **Tasks:** 2 (both TDD: RED then GREEN)
- **Files modified:** 7 (no new files, no golden changes, go.mod and go.sum unchanged)

## Accomplishments

- `internal/config` decodes into pointer fields, so a missing key keeps its default. `closed_days` is clamped (0 gives 1, 366 gives 365). `show` takes `open`, `merged`, `closed` in any order with duplicates allowed; an empty array gives all three states and the warning `config: show is empty; showing all states`.
- Errors carry the file and line: `toml.ParseError` values use `Position.Line`; the plain wrong-type errors (`closed_days = "fourteen"`, `14.5`) are matched by regexp; anything else falls back to `<path>: <problem>`. Messages pass through `domain.Clean`.
- `effectiveClosedDays` uses `fs.Visit`, so `--closed-days 14` still beats a config of 20. The flag range is 1..365 with the message `prv: --closed-days must be between 1 and 365` (fixes WR-01). The result goes to both `github.Options.ClosedDays` and `ui.Options.ClosedDays`. `TestClosedWindowFromConfig` confirms `closed_days = 3` on 2026-10-11 puts `closed:>=2026-10-07` in the search and a PR closed five days ago out of the list.
- The UI starts the warning clock on the first current-generation fetch result (success or failure) and arms a `redrawMsg` for 8 s through `m.after`. Init still arms nothing. The footer puts the flash and notes first, then the warning (cut with an ellipsis, left out under 12 cells), then key hints, with `api N` at the right edge. The error panel gets one warning line above `r retry · q quit`.

## Task Commits

1. **Task 1 (TDD):** RED `750f31d` (test) then GREEN `87b2991` (feat): config keys, clamping, show, warnings, line-numbered errors
2. **Task 2 (TDD):** RED `6776412` (test) then GREEN `6d4e77e` (feat): flag/config precedence, settings and warnings wired to github and ui

## Deviations from Plan

None - plan executed exactly as written. (The RED commits are compile failures, because the tests reference fields and functions that do not exist yet; this counts as RED for a typed language.)

## Verification

- `go vet ./...`, `go build ./...` and `go test ./... -count=1 -race`: pass.
- Task 2 verify command: 10 top-level PASS lines. Hygiene grep (terminal output or file writes in non-test `internal/` code): no matches.
- Acceptance greps: `effectiveClosedDays(` and `between 1 and 365` in cmd/prv/main.go; no `config.Load(` under `internal/`; `warnUntil` in model.go and view.go; Init arms 0 timers; no changes under `internal/ui/testdata`.

## Known Stubs

None. `Options.ConfigPath` and `Model.configPath` are stored but not yet rendered; plan 02-10 shows them in the help overlay (stated in the plan).

## Threat Flags

None. T-02-17 (key names cleaned), T-02-18 (flag bound, config clamp), T-02-22 (exit 2 with path and line, size cap), T-02-28 (separate warning fields, bounded lifetime) are mitigated as listed; T-02-SC: no new modules.

## Issues Encountered

None.

## User Setup Required

None.

## Next Phase Readiness

Ready for 02-10, which lists the warnings and the config path in the help overlay. Unrelated pre-existing staged `.serena` files and the modified `.planning/config.json` were left out of every commit.

## Self-Check: PASSED

- Modified files exist: internal/config/config.go, cmd/prv/main.go, internal/ui/model.go, internal/ui/view.go and their tests.
- Commits `750f31d`, `87b2991`, `6776412`, `6d4e77e` are on the branch.

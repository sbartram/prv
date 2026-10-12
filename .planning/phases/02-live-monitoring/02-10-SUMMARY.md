---
phase: 02-live-monitoring
plan: 10
subsystem: ui
tags: [go, bubbletea-v2, help-overlay, browser, go-gh, url-allow-list]

requires:
  - phase: 02-live-monitoring
    provides: "02-09 Options.ConfigPath/Warnings, Model.configPath/warnings; 02-02 keyMap; 02-08 number prompt; 02-01 setFlash and the fetch recover pattern"
provides:
  - "'?' full-screen help overlay built from the same keyMap as the footer hints (key table, badge legend, settings, API line, config warnings)"
  - "'o' opens the selected PR through go-gh pkg/browser, gated by an exact https://github.com allow-list (safeURL)"
  - "ui.Opener interface and cmd/prv browserOpener (browser.New with io.Discard)"
affects: [phase-02-verification, phase-03-add-by-url]

actuals:
  tokens: 7800
  tasks: 2
  commits: 4
plan_head_before: 95dbb5f4892273833d0c45dfa8ce7f0a456649e9
plan_head_after: f8647b1fb00a0a4981b0f06db39b136d77e1bfb2

tech-stack:
  added: ["github.com/cli/browser v1.3.0 (indirect, via go-gh pkg/browser)", "github.com/google/shlex (indirect, via go-gh pkg/browser)"]
  patterns:
    - "Help overlay is a render() short-circuit on Model.showHelp; the key handler consumes any key before anything else"
    - "Opener injected through ui.Options; ui never imports go-gh's browser package"
    - "URL allow-list by parsed components (scheme, user, port, EqualFold hostname), never a string prefix"

key-files:
  created:
    - internal/ui/help.go
    - internal/ui/help_test.go
    - internal/ui/open.go
    - internal/ui/open_test.go
    - internal/ui/testdata/TestHelpGolden/80x24.golden
    - internal/ui/testdata/TestHelpGolden/120x40.golden
  modified:
    - internal/ui/keys.go
    - internal/ui/model.go
    - internal/ui/view.go
    - internal/ui/filter_test.go
    - cmd/prv/main.go
    - go.mod
    - go.sum
    - internal/ui/testdata/TestGolden/*.golden
    - internal/ui/testdata/TestGoldenListColor.golden

key-decisions:
  - "Help overlay cuts to the pane height by dropping the middle lines and keeping 'press any key to return' as the last line, so a long warning list can never hide the way out"
  - "safeURL also refuses any space or control rune up front; url.Parse alone accepts a space inside a path"
  - "Open flash lasts flashFor (4 s), the existing constant, matching the plan's 4 s"

patterns-established:
  - "Overlay state lives in the model and is consumed ahead of the prompt handler; refresh messages ignore it"

requirements-completed: [NAV-02, NAV-03]

coverage:
  - id: D1
    description: "'?' opens a full-screen help overlay with the keymap table, badge legend, settings, API line and config warnings; any key closes it, ctrl+c quits, refreshes run behind it; fits 80x24"
    requirement: NAV-03
    verification:
      - kind: unit
        ref: "internal/ui/help_test.go#TestHelpOverlay,TestHelpContent,TestHelpFromKeymap,TestRefreshBehindHelp,TestHelpFits,TestHelpWarningsAndCleanPath"
        status: pass
      - kind: unit
        ref: "internal/ui/help_test.go#TestHelpGolden"
        status: pass
    human_judgment: false
  - id: D2
    description: "'o' opens the selected PR in the browser only for exact https://github.com URLs; refused links, launcher errors and panics become short footer messages"
    requirement: NAV-02
    verification:
      - kind: unit
        ref: "internal/ui/open_test.go#TestSafeURL,TestOpenKey,TestOpenRefusesForeignURL,TestOpenFailureFlash,TestOpenPanicRecovered,TestOpenNilOpener,TestOpenIgnoredOutsideList,TestOpenCanceledIgnored,TestOpenHint"
        status: pass
    human_judgment: false
  - id: D3
    description: "In a real terminal, 'o' opens the same browser as `gh pr view --web` without disturbing the screen, and the overlay reads well"
    verification: []
    human_judgment: true
    rationale: "The plan's <human-check>: real browser launch, terminal rendering and config edits cannot be asserted by tests; recorded for end-of-phase UAT"

duration: 6min
completed: 2026-10-12
status: complete
---

# Phase 02 Plan 10: "?" explains every key and badge, and "o" opens the selected PR on github.com Summary

**A keymap-driven help overlay (`?`) and a safe `o` that hands only exact https://github.com links to go-gh's browser from inside a tea.Cmd.**

## Performance

- **Duration:** about 6 min
- **Started:** 2026-10-12T16:13:50Z (approximate; start time was not recorded)
- **Completed:** 2026-10-12T16:19:00Z
- **Tasks:** 2
- **Files modified:** 23 (14 golden files, go.mod, go.sum, 7 Go files plus 4 new Go files)

## Accomplishments

- `?` shows one screen with the key table from `keyMap.FullHelp()`, a glyph legend for all four review and five CI badges, `config <path> · every 60s · 14 days`, `api 4812 / 5000, resets 15:02` (or "api limit not known yet"), and every config warning for the whole session. It fits 80x24 and 120x40, uses only ANSI-16 styles, and passes the path and warnings through `domain.Clean`.
- Any key closes the overlay and is consumed (`q` closes, never quits); `ctrl+c` still quits; ticks and fetch results keep flowing behind it, and the list shows the new data on return.
- `o` validates `PR.URL` with `safeURL` (https, `strings.EqualFold` host `github.com`, no port, no user info, no space or control rune) and returns `openCmd`; the launch happens only in the Cmd. Refused links show "not a github.com link — not opened", a nil opener "no browser available", a launcher error or recovered panic "could not open the browser"; `context.Canceled` is ignored.
- `cmd/prv` builds `browser.New("", io.Discard, io.Discard)` once and passes it as `ui.Options.Opener`; `internal/` never imports the browser package.

## Task Commits

1. **Task 1: help overlay** - `f612e00` (test, RED, build failed) then `0e9a707` (feat, GREEN, with goldens)
2. **Task 2: open in browser** - `a481f67` (test, RED, build failed) then `f8647b1` (feat, GREEN, with goldens, main.go, go.mod/go.sum)

**Plan metadata:** committed separately (docs: complete plan).

## Files Created/Modified

- `internal/ui/help.go` - `renderHelp` and `fmtInterval`
- `internal/ui/open.go` - `Opener`, `safeURL`, `openedMsg`, panic-safe `openCmd`
- `internal/ui/keys.go` - `Help` (`?`) and `Open` (`o`) bindings; ShortHelp and FullHelp updated (three columns)
- `internal/ui/model.go` - `showHelp`, `opener`, `Options.Opener`, key handling order, `openedMsg` handling
- `internal/ui/view.go` - `render()` returns the overlay first when `showHelp`
- `cmd/prv/main.go` - `browserOpener` and its construction
- `internal/ui/help_test.go`, `internal/ui/open_test.go` - the plan's named tests
- `internal/ui/testdata/**` - regenerated list/empty/filtered/colour footers; new help goldens

## Decisions Made

- When the overlay is taller than the pane, lines from the middle are dropped and "press any key to return" stays last, so many warnings cannot trap the user.
- `safeURL` rejects whitespace and control runes before parsing, because `url.Parse` accepts a space in a path.
- Open and help messages reuse `flashFor` (4 s) and `setFlash`; the config warning fields are untouched.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] go.sum lacked entries for go-gh's browser dependencies**
- **Found during:** Task 2 (`go vet` on cmd/prv)
- **Issue:** The plan states go-gh `pkg/browser` is "already in the module graph; go.mod and go.sum unchanged". `go.sum` had no hashes for `github.com/cli/browser` and `github.com/google/shlex`, which that package imports, so the build failed.
- **Fix:** Resolved them with the Go toolchain (`GOFLAGS=-mod=mod go build`), which adds only the two versions that go-gh v2.16.2's own go.mod pins (cli/browser v1.3.0 is the module the project stack doc names). No new direct dependency, no version chosen by me, hashes checked against the Go checksum database.
- **Files modified:** go.mod, go.sum
- **Verification:** `go vet ./...` and `go test ./... -race` pass
- **Committed in:** f8647b1

**2. [Rule 1 - Bug] Stale assertion that "o" is an unbound key**
- **Found during:** Task 2 (`TestStateToggles` failed)
- **Issue:** `filter_test.go` checked that the letters m, o, c do nothing; `o` is now a bound key.
- **Fix:** Replaced "o" with "x" in that list.
- **Files modified:** internal/ui/filter_test.go
- **Committed in:** f8647b1

**3. [Rule 1 - Bug] Own test waited on a real 4 s timer**
- **Found during:** Task 2 (suite took 16 s)
- **Issue:** `TestOpenRefusesForeignURL` ran the flash timer Cmd.
- **Fix:** Removed the call; the test only checks the Cmd exists.
- **Files modified:** internal/ui/open_test.go
- **Committed in:** f8647b1

---

**Total deviations:** 3 auto-fixed (2 bug, 1 blocking)
**Impact on plan:** None on behavior. The go.sum change contradicts the plan's threat-register line T-02-SC ("go.mod and go.sum unchanged"); the two added modules are go-gh's own pinned dependencies. Worth a glance at the go.sum diff in review.

## Issues Encountered

None beyond the deviations above. Known gap carried from the plan: a terminal browser named in BROWSER/GH_BROWSER (w3m, lynx) can take over the terminal; `gh` has the same limit.

## Known Stubs

None.

## Threat Flags

None. The new surface (URL open, overlay rendering of config text) is the plan's T-02-16, T-02-26, T-02-19 and T-02-20, each mitigated and tested.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The plan's `<human-check>` (real browser launch, overlay on a real terminal, config edits with `closed_days = 3` and a bad key) is left for end-of-phase UAT.
- Last plan of phase 02: ready for phase verification.

---
*Phase: 02-live-monitoring*
*Completed: 2026-10-12*

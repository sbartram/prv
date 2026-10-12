---
phase: 02-live-monitoring
verified: 2026-10-12T18:00:00Z
status: human_needed
score: 5/5 must-haves verified
covered_files:
  - .planning/phases/02-live-monitoring/02-01-PLAN.md
  - .planning/phases/02-live-monitoring/02-01-SUMMARY.md
  - .planning/phases/02-live-monitoring/02-02-PLAN.md
  - .planning/phases/02-live-monitoring/02-02-SUMMARY.md
  - .planning/phases/02-live-monitoring/02-03-PLAN.md
  - .planning/phases/02-live-monitoring/02-03-SUMMARY.md
  - .planning/phases/02-live-monitoring/02-04-PLAN.md
  - .planning/phases/02-live-monitoring/02-04-SUMMARY.md
  - .planning/phases/02-live-monitoring/02-05-PLAN.md
  - .planning/phases/02-live-monitoring/02-05-SUMMARY.md
  - .planning/phases/02-live-monitoring/02-06-PLAN.md
  - .planning/phases/02-live-monitoring/02-06-SUMMARY.md
  - .planning/phases/02-live-monitoring/02-07-PLAN.md
  - .planning/phases/02-live-monitoring/02-07-SUMMARY.md
  - .planning/phases/02-live-monitoring/02-08-PLAN.md
  - .planning/phases/02-live-monitoring/02-08-SUMMARY.md
  - .planning/phases/02-live-monitoring/02-09-PLAN.md
  - .planning/phases/02-live-monitoring/02-09-SUMMARY.md
  - .planning/phases/02-live-monitoring/02-10-PLAN.md
  - .planning/phases/02-live-monitoring/02-10-SUMMARY.md
  - cmd/prv/main.go
  - go.mod
  - go.sum
  - internal/config/config.go
  - internal/domain/backoff.go
  - internal/domain/clean.go
  - internal/domain/list.go
  - internal/domain/pr.go
  - internal/domain/status.go
  - internal/github/client.go
  - internal/github/errors.go
  - internal/github/ratelimit.go
  - internal/ui/help.go
  - internal/ui/keys.go
  - internal/ui/layout.go
  - internal/ui/model.go
  - internal/ui/open.go
  - internal/ui/refresh.go
  - internal/ui/styles.go
  - internal/ui/view.go
covered_digest: "v3:sha256:b7a9fc7444b87435a9a5625151ab9da947f17716f8980410d1864ac20737946e"
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "Run prv in a real terminal at 80 and 40 columns. Toggle Wi-Fi off, wait for one refresh, then on again."
    expected: "Header shows the amber 'refresh failed HH:MM (network) · data from HH:MM:SS' marker with the rows kept, then returns to 'updated HH:MM:SS' after recovery. The 40-column footer keeps notes and the api segment readable. (02-04 D6)"
    why_human: "Unit tests cover the strings, SGR codes and widths, not the real-terminal look"
  - test: "View the faint 'api N' footer segment and the yellow header warning (below 20% of the limit) on a real theme and inside a tmux/herdr pane."
    expected: "The api number is legible but quiet; the warning reads as a warning on light and dark backgrounds. (02-05 D5)"
    why_human: "Colour legibility on real themes and in multiplexers cannot be asserted in code"
  - test: "Against a real or near-exhausted token, watch the header and press r during a pause."
    expected: "Header shows 'paused until HH:MM (rate limit)'; r flashes 'rate limited - resumes HH:MM' and sends no request; prv resumes by itself. (02-06 D5, 02-07)"
    why_human: "The simulated hour and the fake-transport tests prove the logic. Only a live GitHub 200/403/429 rate-limit response proves the real header and body shapes."
  - test: "In a real terminal, press 1/2/3, type '/' then digits, paste digits and non-digits into the prompt, press esc and enter. (02-08 D-judgment)"
    expected: "Indicator '[open merged closed]' toggles. Only digits are accepted, including from a paste. The list narrows, the hidden count is right, and esc clears."
    why_human: "Bracketed paste and key delivery depend on the real terminal"
  - test: "Press 'o' on a selected PR with the default browser, then with BROWSER set to a bad command. (02-10)"
    expected: "The PR opens in the browser on github.com. A bad launcher gives a footer flash 'could not open the browser', and the screen is not corrupted."
    why_human: "A real browser launch is outside what the fake Opener tests exercise"
  - test: "Press '?' on a real terminal, with and without config warnings and at a short pane height. (02-10)"
    expected: "Overlay shows keys, badge legend, settings in effect, api budget and warnings. Any key returns, q does not quit."
    why_human: "Visual layout on a real terminal"
  - test: "Edit ~/.config/prv/config.toml: add a misspelt key (refresh_intervall), then a bad value (refresh_interval = \"banana\"), then show = [\"open\"] and closed_days = 7. (02-09)"
    expected: "Misspelt key: prv starts and the footer shows 'config: unknown key ...' for about 8 s, and '?' lists it. Bad value: exit 2 with 'path line N: ...' before the TUI. show/closed_days: the header indicator and the closed window follow the file."
    why_human: "End-to-end config-to-header wiring with a live token. The bad-value exit and the token-missing exit were run in this verification (see Spot-Checks); the live TUI parts were not."
---

# Phase 2: Live Monitoring Verification Report

**Phase Goal:** User can leave prv open in a terminal pane all day and trust that it is current, stays inside rate limits, and lets them narrow the list and jump to a PR on github.com
**Verified:** 2026-10-12
**Status:** human_needed
**Re-verification:** No, initial verification

## Goal Achievement

### Observable Truths

| # | Truth (ROADMAP success criterion) | Status | Evidence |
|---|-----------------------------------|--------|----------|
| 1 | Left open, the list refreshes on the config.toml interval (default 60s, floor 15s). `r` refreshes at once. Header shows the last successful update time. Missing config means defaults. | VERIFIED | `config.Load` returns `Default()` on `fs.ErrNotExist` and applies `max(interval, MinInterval)` (config.go:104-141). `main.go:run` loads config once and passes `cfg.Interval` to `ui.New`. Update arms one gen-stamped tick from each current-generation result (model.go:299-331, `schedule`/`arm`). `r` handled at model.go:226-240. Header `updated HH:MM:SS` from `lastOK`, set only on success (model.go:324, view.go:274-277). `TestAutoRefreshTickChain`, `TestUpdatedOnlyOnSuccess`, `TestSimulatedHour/steady` (59-61 calls per hour) pass under `-race`. |
| 2 | A failed refresh keeps the last data with an error indicator. The status bar shows remaining API rate limit. A low or used-up limit makes prv back off and show when it resumes. Repeated `r` never overlaps refreshes. | VERIFIED | Failed fetch with data keeps `snap`/rows and sets `failedAt`; header shows the "refresh failed ... data from" marker at prio 7 (model.go:307-314, view.go:277-288). Footer `api N` from `snap.RateLimit` when `Limit>0`; header warning below 20% (view.go:140, 299-302). Query asks for `rateLimit { limit cost remaining resetAt }` (client.go:68). `domain.NextDelay` is pure: Retry-After, then reset, then bounded exponential, plus reserve pause and pacing, all server delays clamped to [1 s, 65 min] (backoff.go). Header shows "paused until HH:MM (rate limit)" (view.go:294-296). `r` is refused while in flight and during a pause (model.go:227-234). Real responses are recognised through `captureTransport` (HTTP 200 with errors, 403/429) (ratelimit.go, errors.go:108-181). `TestSimulatedHour` (steady, server-errors, exhausted, retry-after, hostile-retry-after, presses), `TestNoOverlappingFetches` and `TestRateLimitDetection` pass under `-race`. |
| 3 | The list is sorted by need for attention (failing CI and changes requested first). Order does not shuffle between refreshes. The selected PR stays selected after a refresh, even when it moves. | VERIFIED | `domain.Bucket` puts open failing CI at 1 and changes requested at 2. `domain.Sort` is stable on bucket, createdAt desc, repo, number. `updatedAt` is deliberately not a key (list.go:26-85). `Fetch` calls `domain.Sort` (client.go:218). Selection is held as `Ref` and `rebuild()` re-derives the cursor with a nearest-neighbour fallback (model.go:395-432). `TestSelectionFollowsRef`, `TestBucket`/`TestSort` pass. |
| 4 | User can toggle Open/Merged/Closed in any combination and type a PR number to narrow the list. Auto-found merged/closed PRs only appear within the configured window (default 14 days). | VERIFIED | Keys 1/2/3 toggle `m.show` (model.go:248-256), `domain.Visible` filters by state set and number prefix (list.go:107-118). `/` opens a digits-only prompt, max 7 digits, with a paste handler (model.go:181-189, 348-370). Header shows `[open merged closed]` indicator and "N of M PRs"; a filtered-empty screen names the hidden count (view.go:44-77). Window: `BuildSearch` uses `closed:>=` date plus `InWindow` exact filter (client.go:91-94, 215-217). `closed_days` flows config, then `effectiveClosedDays` (flag beats config, 1..365), then `github.NewClient` (main.go:70-100, config.go:128-130). `TestHiddenCount`, `TestNumberPrompt`, `TestClampClosedDays` pass. |
| 5 | User can press `o` to open the selected PR in the browser and `?` to see the key bindings screen. | VERIFIED | `o`: `safeURL` allow-list (https, host exactly github.com, no port/userinfo/whitespace/control), then `openCmd` in a `tea.Cmd` with panic recovery. `main.go` supplies `browser.New("", io.Discard, io.Discard)` (open.go, model.go:212-225, main.go:107-111). `?`: `renderHelp` draws from the same keymap, with badge legend, settings, api budget and warnings (help.go). `TestSafeURL`, `TestOpenKey`, `TestOpenPanicRecovered`, `TestHelpOverlay`, `TestHelpContent`, `TestHelpFits` pass. Real browser launch and real-terminal look are human items. |

**Score:** 5/5 truths verified (0 present, behavior-unverified). The behavior-dependent truths (single-flight, generation-dropped stale ticks and results, pause refusal, selection follow) each have a passing named test: `TestNoOverlappingFetches`, `TestStaleTickDropped`, `TestStaleResultDropped`, `TestSimulatedHour`, `TestSelectionFollowsRef`.

### Deferred Items

None. Remaining roadmap items (reviewer names, merge-readiness, detail view, add-by-URL, NAV-05) are Phase 3 and 4 scope, not gaps in this phase.

### Advisory (New Scope, Unevidenced)

Re-verification did not run, so none. Info-level observations:

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | README.md does not document `config.toml` (`refresh_interval` as a duration string such as `"60s"`, `closed_days`, `show`). A bare TOML integer is rejected with a line-numbered error. | other | Not in the roadmap criteria. Discoverability only. |
| 2 | Code review items IN-01 to IN-04 are still `open` in 02-REVIEW-DISPOSITION.md: dead `MergedAt` payload, dead `Config.Path`, a 32-bit build break on `1 << 31`, silently ignored positional arguments. | other | Info severity. No effect on the goal. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/config/config.go` | TOML load, floor, clamp, warnings, 1 MiB cap | VERIFIED | Substantive, imported by `main.go`. Feeds Interval/ClosedDays/Show/Warnings into `ui.Options`. |
| `internal/ui/refresh.go`, `model.go` | Single-flight tick chain, `r`, backoff, filters | VERIFIED | Wired through `Update`. Data flows from `fetchedMsg`. |
| `internal/domain/backoff.go` | Pure `NextDelay` | VERIFIED | Called from `Model.schedule`. |
| `internal/domain/list.go` | Bucket, Sort, Visible, InWindow | VERIFIED | `Sort` and `InWindow` used by `Fetch`; `Visible` by `rebuild`. |
| `internal/github/ratelimit.go`, `errors.go`, `client.go` | Header capture, classification, rateLimit in query | VERIFIED | `captureTransport` installed as the go-gh transport (client.go:43-50). `classify` is applied in `Fetch`. |
| `internal/ui/view.go`, `help.go`, `keys.go`, `open.go` | Header/footer, help overlay, keymap, safe opener | VERIFIED | Wired. All other `verify.artifacts` runs (10 PLAN files) report `all_passed: true`. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| main.go | config.Load | configPath() before token and TUI | WIRED | A load error exits 2 (run in this verification) |
| config.Interval/Show/Warnings | ui.Options | `ui.New(...)` | WIRED | main.go:100-109 |
| days (flag or config) | github.NewClient | `ClosedDays: days` | WIRED | main.go:93 |
| Fetch | domain.Sort / InWindow | direct calls | WIRED | client.go:215-218 |
| *github.Error | ui (Backoff/Short/Hint) | errors.As into ui-local interfaces | WIRED | No ui to github import |
| model `o` | browser | safeURL, then openCmd, then browserOpener | WIRED | |

`verify.key-links` for all 10 PLAN files: `all_verified: true`.

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| header "updated" | `lastOK` | `m.now()` on a successful `fetchedMsg` | Yes | FLOWING |
| footer/header api | `snap.RateLimit` | `rateLimit {...}` in the GraphQL response, `mapResponse` | Yes | FLOWING |
| rows | `domain.Visible(snap.PRs, show, numFilter)` | Search query, mapped, windowed, sorted | Yes | FLOWING |
| pause | `NextDelay(..., snap.RateLimit, outcome)` | Error `Backoff()` from captured headers | Yes | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Whole suite | `go build ./... && go vet ./... && go test ./... -race -count=1` | `ok` for cmd/prv, config, domain, github, ui | PASS |
| Bad config value stops prv before the TUI | build, `refresh_interval = "banana"`, run | `prv: .../config.toml line 1: refresh_interval: time: invalid duration "banana"`, exit 2 | PASS |
| Misspelt key does not block startup | `refresh_intervall = "5s"` plus valid keys, no token | Passes config, reaches the token check (exit 1, login message) | PASS |
| `--closed-days 0` rejected | `prv --closed-days 0` | `--closed-days must be between 1 and 365`, exit 2 | PASS |
| `--version` | `prv --version` | `prv v0.0.0-...+dirty` | PASS |
| Named behavioral tests | `-run TestSimulatedHour\|TestNoOverlappingFetches\|TestSelectionFollowsRef\|TestOpenKey\|TestSafeURL\|TestHelp...` with `-race -v` | All listed tests PASS | PASS |

### Probe Execution

SKIPPED: no `scripts/*/tests/probe-*.sh` and no probes declared in the PLANs.

### Requirements Coverage

All 13 phase IDs are claimed by at least one PLAN `requirements:` field and are marked Complete in REQUIREMENTS.md. None is orphaned.

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| RFSH-01 | 02-01 | Auto refresh, 60s default, 15s min | SATISFIED | config.go, tick chain, `TestAutoRefreshTickChain`, `TestSimulatedHour/steady` |
| RFSH-02 | 02-02, 02-06 | `r` refreshes at once | SATISFIED | model.go:226-240, `TestRefreshKey`, `TestNoOverlappingFetches` |
| RFSH-03 | 02-02, 02-04 | Time of last successful update shown | SATISFIED | view.go:274-277, `TestUpdatedOnlyOnSuccess` |
| RFSH-04 | 02-01, 02-04, 02-07 | Failed refresh keeps data with an error indicator | SATISFIED | model.go:307-314, view.go:277-288, `TestFailureMarker`, `TestFirstFetchFailureRetries` |
| RFSH-05 | 02-05, 02-06, 02-07 | Rate limit shown, backoff when low or hit | SATISFIED | view.go, backoff.go, ratelimit.go, errors.go, `TestSimulatedHour`, `TestRateLimitDetection` |
| RFSH-06 | 02-02 | Selection survives refresh | SATISFIED | model.go:395-432, `TestSelectionFollowsRef` |
| CONF-01 | 02-01, 02-09 | Settings from config.toml, defaults if missing | SATISFIED | config.go, main.go, `TestLoad*`, CLI spot-checks |
| LIST-08 | 02-03 | Stable attention sort | SATISFIED | list.go, client.go:218, `TestBucket`, `TestSort` |
| FILT-01 | 02-08 | Toggle Open/Merged/Closed | SATISFIED | model.go:248-256, `TestHiddenCount` |
| FILT-02 | 02-09 | Closed window configurable (14 d default) | SATISFIED | client.go:91-94, 215-217, main.go, `TestClampClosedDays` |
| FILT-03 | 02-08 | Type a PR number to narrow | SATISFIED | model.go:348-370, list.go:107-118, `TestNumberPrompt` |
| NAV-02 | 02-10 | `o` opens in browser | SATISFIED | open.go, `TestOpenKey`, `TestSafeURL` (real launch is a human item) |
| NAV-03 | 02-10 | `?` shows key bindings | SATISFIED | help.go, `TestHelpOverlay`, `TestHelpContent` (real-terminal look is a human item) |

### Post-Execution Fixes Checked

- **T-02-08** (unbounded pacing wait): `min(untilReset/left, MaxServerWait)` is present at backoff.go:83. The `TestNextDelay` pacing rows pass.
- **Review WR-01** (`Time.Sub` saturation wrapping to a 1 s pause): `resetWait` cuts the span before adding the second (backoff.go:99-101), used on both the rate-limited and reserve branches. `TestNextDelayNoOverflow` passes.
- **Review WR-02** (return-order dependence): the `cmd := m.x(); return m, cmd` form is present at every mutating site in `Update` (model.go:219, 232, 239, 285, 288, 292, 312, 318, 331).
- **02-SECURITY.md**: `threats_open: 0`. The mitigations spot-read in code (T-02-16 `safeURL`, T-02-07 header-only capture, T-02-01 1 MiB cap, T-02-06 absolute `XDG_CONFIG_HOME` only) match the cited lines.

### Anti-Patterns Found

None. A scan for TBD/FIXME/XXX/TODO/HACK across all Go files returns nothing. No stub returns reach rendering, and no Update/View I/O was found (fetch and open run inside `tea.Cmd`).

### Human Verification Required

Seven items are listed in the frontmatter `human_verification`. All were recorded by the SUMMARYs as `human_judgment: true`. Automated evidence for the logic behind each is complete. What remains is real-terminal look and feel, a real browser launch, a live GitHub rate-limit response, and live config-to-header wiring.

### Gaps Summary

No gaps. All five roadmap success criteria and all 13 requirement IDs are backed by code that exists, is wired and has flowing data. The behavior-dependent invariants (single-flight, stale-drop, pause refusal, selection follow) are exercised by passing named tests under `-race`. The status is `human_needed` only because of the real-terminal, real-browser and live-API checks above.

---

_Verified: 2026-10-12_
_Verifier: Claude (gsd-verifier)_

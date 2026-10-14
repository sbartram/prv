---
phase: 01-status-at-a-glance
verified: 2026-10-12T02:00:00Z
status: passed
score: 5/6 must-haves verified
covered_files:
  - ".gitignore"
  - ".planning/phases/01-status-at-a-glance/01-01-PLAN.md"
  - ".planning/phases/01-status-at-a-glance/01-01-SUMMARY.md"
  - ".planning/phases/01-status-at-a-glance/01-02-PLAN.md"
  - ".planning/phases/01-status-at-a-glance/01-02-SUMMARY.md"
  - ".planning/phases/01-status-at-a-glance/01-03-PLAN.md"
  - ".planning/phases/01-status-at-a-glance/01-03-SUMMARY.md"
  - ".planning/phases/01-status-at-a-glance/01-04-PLAN.md"
  - ".planning/phases/01-status-at-a-glance/01-04-SUMMARY.md"
  - "README.md"
  - "cmd/prv/main.go"
  - "cmd/prv/main_test.go"
  - "go.mod"
  - "go.sum"
  - "internal/domain/clean.go"
  - "internal/domain/domain_test.go"
  - "internal/domain/list.go"
  - "internal/domain/pr.go"
  - "internal/domain/status.go"
  - "internal/github/client.go"
  - "internal/github/client_test.go"
  - "internal/github/errors.go"
  - "internal/github/errors_test.go"
  - "internal/ui/layout.go"
  - "internal/ui/model.go"
  - "internal/ui/model_test.go"
  - "internal/ui/styles.go"
  - "internal/ui/view.go"
  - "internal/ui/view_test.go"
covered_digest: "v3:sha256:6ae9a26f8ff371d5ed612de8c6f899da80b8eeb3dbad4be36dfd6e351b6b07da"
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "Run prv on an account that has PRs assigned to it (assignee:@me), or add an assigned PR to the test account, with a wide window (for example a hidden --closed-days 365 run)"
    expected: "A PR assigned to the viewer but authored by someone else appears once, with the author's login in AUTHOR"
    why_human: "Plan 01-02 declares this truth as 'backstop' (verification: backstop). The test account has 0 assigned PRs, so live evidence is a cli/cli stand-in only. TestMembership proves the client mapping on a fixture, not the live assignee:@me behaviour. Presence plus wiring never qualifies as evidence for a backstop truth."
  - test: "Compare several rows with github.com: one with approvals but no required-review rule, one with no checks, one with failing and running checks"
    expected: "REVIEW and CI badges match github.com; a PR without checks never shows passing; an approved PR on a repo without required reviews shows approved, never required"
    why_human: "The test account has no open PRs, so live row evidence comes from fixtures and the spike's ~350-PR sample on stand-in repos, not from the user's own data. The user's closing terminal run confirmed the list rendered, but did not record a row-by-row comparison."
  - test: "Open prv in a light-theme and a dark-theme terminal, and in a narrow multiplexer pane (about 40 columns)"
    expected: "ANSI-16 badge colours stay readable on both themes; the selected row is visible; the footer and list stay usable when narrow; terminal restored after q and ctrl+c"
    why_human: "Colour rendering and real terminal behaviour cannot be proven by golden files. Deferred by the 01-03 and 01-04 SUMMARYs. Terminal restore after q was confirmed by the user at the 01-04 checkpoint."
  - test: "Start prv with an invalid GH_TOKEN (for example GH_TOKEN=bad prv)"
    expected: "The error panel shows the 401 hint (run `gh auth login`, fix or unset GH_TOKEN/GITHUB_TOKEN); r retries; q exits 0; the token string never appears"
    why_human: "The 401 flow is covered by unit tests with a fake transport only; it has not been exercised against real GitHub in a real terminal."
  - test: "View the badge glyphs (✓ ✗ ○ ● – ?) in a terminal with a CJK locale or ambiguous-width setting (RESEARCH A7)"
    expected: "Columns stay aligned"
    why_human: "Ambiguous-width glyph handling depends on the terminal and locale; cannot be checked programmatically."
---

# Phase 1: Status at a Glance Verification Report

**Phase Goal:** User runs `prv` and, from live GitHub data, sees every PR they authored or are assigned to with its state, review status, and CI badge, without opening a browser
**Verified:** 2026-10-12
**Status:** human_needed
**Re-verification:** No, initial verification

## Goal Achievement

The code delivers the goal end to end. `main` resolves the token, builds a go-gh GraphQL client, and hands it to the Bubble Tea model through the `ui.Fetcher` interface. The model runs the fetch in a `tea.Cmd` and renders the result as a table with state, review and CI columns. I read all production files, ran build, vet, gofmt and `-race` tests, ran the live end-to-end test against github.com, and installed the pushed module from GitHub. No must-have is FAILED. The remaining items are human-only checks, plus one plan-declared backstop truth with no direct live evidence.

### Observable Truths (ROADMAP success criteria, plus plan backstop truth)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Token from `GH_TOKEN`, `GITHUB_TOKEN`, then gh login shows live PRs; no token gives a clear exit; token never on screen, in logs, or in a file | VERIFIED | `cmd/prv/main.go` uses `auth.TokenForHost("github.com")` (go-gh order confirmed in 01-SPIKE Q5), passes the token only to `github.Options.Token`. Ran the built binary with an empty environment: stderr is exactly "prv needs a GitHub token. Run `gh auth login`, or set GH_TOKEN / GITHUB_TOKEN.", stdout empty, exit 1. `PRV_LIVE=1 go test ./cmd/prv -run TestLiveEndToEnd` PASSED against live GitHub. `LogIgnoreEnv: true` in `internal/github/client.go`. `TestTokenNeverAppears` passes. Grep of the repo for token patterns finds only grep commands inside PLAN files. Grep of non-test internal code for stdout/stderr/log/file writes finds nothing. |
| 2 | List has every open authored or assigned PR plus merged/closed in the last 14 days; deduped once; row shows repo, number, title, author, draft marker, state | VERIFIED | `BuildSearch` produces `is:pr (author:@me OR assignee:@me) (is:open OR closed:>=DATE) sort:updated-desc`, sent as variable `$q` with `ISSUE_ADVANCED`. `Fetch` pages up to 5x100, dedupes on node id, applies `domain.InWindow` (exact, inclusive) and `domain.Sort`. `renderList` shows PR, TITLE with `[draft] ` prefix, AUTHOR (`me` for the viewer), STATE. Tests: `TestPagination`, `TestMembership`, `TestInWindow`, `TestSort`, `TestTruncatedAfterFivePages`, `TestStatusMapping` all pass. The 500-result cap is a declared, accepted limit and the footer says "list truncated". |
| 3 | Review status matches github.com; null `reviewDecision` derived from opinionated reviews; never "review required" by mistake | VERIFIED | `domain.DeriveReview` uses `reviewDecision` when set; otherwise any CHANGES_REQUESTED wins, then APPROVED, else none; it never returns required on its own. Only `latestOpinionatedReviews` is requested (01-SPIKE Q1: the non-opinionated list lets a later comment hide an approval). `TestDeriveReview`, `TestDeriveReviewInvariant` (order independence), `TestCommentAfterApprove`, `TestStatusMapping` pass. Fidelity to github.com rests on fixtures and the spike sample; see human item 2. |
| 4 | One CI badge matches github.com; no checks never shows passing | VERIFIED | `domain.ClassifyRollup`: nil rollup gives CINone ("– no checks"); SUCCESS passing; FAILURE/ERROR failing; PENDING/EXPECTED running; other gives unknown. Only `statusCheckRollup { state }` is requested. `TestClassifyRollup`, `TestBadge`, `TestStatusMapping` pass. |
| 5 | j/k and arrows move; q quits and the terminal is restored; `go install ...@latest` installs a working binary; `prv --version` prints a version | VERIFIED | `Update` handles j/down/k/up with end clamps and q/ctrl+c to `tea.Quit` in every state (`TestQuitInEveryState`, `TestEndToEndFixture`, `TestScrollFollowsCursor`). `View` sets `AltScreen = true`, so Bubble Tea restores the screen. I ran `GOPROXY=direct go install github.com/sbartram/prv/cmd/prv@latest` into an empty GOBIN: it installed `v0.0.0-20261012010335-d2057f9d080b` and `prv --version` printed `prv v0.0.0-20261012010335-d2057f9d080b`. A local `CGO_ENABLED=0` build records `CGO_ENABLED=0`. `go.mod` is `go 1.26.0` with no toolchain line. The user ran the binary in a real terminal and approved at the 01-04 checkpoint. |
| 6 | (Plan 01-02, backstop) `assignee:@me` returns the PRs assigned to the user on the user's own account | UNCERTAIN (insufficient_spec) | Plan marks this `verification: backstop`. The account has 0 assigned PRs, so live evidence is a stand-in (`cli/cli`, union count 268 = 265 + 144 - 141 in 01-SPIKE Q8). `TestMembership` proves the mapping on a fixture only. Routed to human verification. |

**Score:** 5/6 truths verified (0 present, behavior-unverified; 1 insufficient_spec routed to human)

Plan-level truths I spot-checked in code and tests, all consistent: draft keeps state `open`; unknown state maps to `unknown`; author null shows `ghost`; zero-row j/k does not panic; TITLE is the only truncated column and is cell-width safe; fetch panic shows the error panel without the panic value; r retries only from the error panel and not while loading; partial data keeps rows and adds a footer note; later-page failure keeps earlier pages and marks the list truncated; all remote strings pass through `domain.Clean`.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `go.mod` | module path, go 1.26.0 | VERIFIED | `module github.com/sbartram/prv`, `go 1.26.0`, no toolchain line, only `charm.land/*/v2` Charm imports |
| `cmd/prv/main.go` | flags, token, no-token exit, wiring | VERIFIED | 87 lines, substantive, `auth.TokenForHost`, `github.NewClient`, `ui.New`, `tea.NewProgram` |
| `internal/domain/{pr,status,clean,list}.go` | types, derivations, sanitiser, window and sort | VERIFIED | substantive, used by `internal/github` and `internal/ui` |
| `internal/github/client.go`, `errors.go` | query, paging, mapper, error classification | VERIFIED | `ISSUE_ADVANCED`, `maxPages = 5`, `classify(` on every request failure |
| `internal/ui/{model,view,layout,styles}.go` | states, table, widths, ANSI-16 styles | VERIFIED | `AltScreen = true`, `recover()`, `ansi.Truncate`, `lipgloss.Green` |
| `internal/ui/testdata/*` golden files | locked screens | VERIFIED | `TestGolden`, `TestGoldenListColor` pass |
| `README.md`, `.gitignore` | install, auth, scopes, keys; ignore binary | VERIFIED | install command, token order, read-only scopes present |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `cmd/prv/main.go` | `internal/github` and `internal/ui` | `github.NewClient` passed to `ui.New` as `Fetcher` | WIRED | `*github.Client` implements `Fetch(ctx)`; build passes |
| `github.Fetch` | `domain.InWindow` / `domain.Sort` | called before returning the snapshot | WIRED | `client.go` lines in `Fetch` |
| `github.Fetch` | `classify` | every `DoWithContext` failure | WIRED | `client.go`, `errors.go` |
| `github.Error` | error panel | `Hint()` via the `hinter` interface in `ui` (ui never imports github) | WIRED | `TestErrorPanelShowsHint` |
| `view.go` | `layout.go`, `styles.go` | `columnWidths(`, `StyleFunc(`, `offset` | WIRED | `renderList` |
| `README.md` | `cmd/prv` | `go install github.com/sbartram/prv/cmd/prv@latest` | WIRED | remote install ran successfully |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `ui/view.go` list | `m.snap.PRs` | `Fetcher.Fetch` -> go-gh GraphQL `search(type: ISSUE_ADVANCED)` | Yes: `TestLiveEndToEnd` passes against github.com; fixture tests cover row content | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Whole module builds | `go build ./...` | ok | PASS |
| Format and vet | `gofmt -l .`, `go vet ./...` | empty, ok | PASS |
| Tests with race detector | `go test ./... -count=1 -race` | 4 packages ok; 44 PASS, 1 SKIP (live test, needs `PRV_LIVE=1`) | PASS |
| Live end to end | `PRV_LIVE=1 go test ./cmd/prv -run TestLiveEndToEnd -count=1` | PASS | PASS |
| Version | built binary `--version` | `prv v0.0.0-...+dirty`, exit 0 | PASS |
| No token | `env -i ... prv` | exact D-11 line on stderr, stdout empty, exit 1 | PASS |
| Remote install | `GOPROXY=direct go install github.com/sbartram/prv/cmd/prv@latest` | installed, `--version` printed | PASS |

### Probe Execution

SKIPPED: the phase declares no probe scripts, and `scripts/*/tests/probe-*.sh` does not exist.

### Requirements Coverage

The union of `requirements` across the four PLAN files is all 13 phase IDs. REQUIREMENTS.md maps exactly these 13 IDs to Phase 1, so there are no orphaned requirements.

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| AUTH-01 | 01-01 | Same token as gh, env then gh login | SATISFIED | `auth.TokenForHost`; spike Q5 |
| AUTH-02 | 01-01 | No token gives a clear exit message | SATISFIED | binary run, exit 1, `TestNoToken` |
| AUTH-03 | 01-01, 01-02, 01-04 | Token never logged, shown or written | SATISFIED | `LogIgnoreEnv`, no I/O in internal, `TestTokenNeverAppears`, token-pattern grep |
| CONF-02 | 01-01, 01-04 | `go install ...@latest` works; `--version` prints | SATISFIED | remote install run by me |
| WATCH-01 | 01-01, 01-02 | Authored PRs listed | SATISFIED | `author:@me` in search; live test |
| WATCH-02 | 01-01, 01-02 | Assigned PRs listed | SATISFIED (live evidence is a stand-in; see truth 6) | `assignee:@me` in search; `TestMembership` |
| WATCH-03 | 01-02 | Authored and assigned PR appears once | SATISFIED | single search plus node-id dedupe; `TestPagination` |
| LIST-01 | 01-01..03 | Repo, number, title, author, draft marker | SATISFIED | `renderList`; tests |
| LIST-02 | 01-01..03 | State open/merged/closed | SATISFIED | `ParseState`; `TestStatusMapping` |
| LIST-03 | 01-01, 01-03 | Review status, never wrongly "required" | SATISFIED | `DeriveReview`; tests |
| LIST-05 | 01-01, 01-03 | One CI badge; no checks never passing | SATISFIED | `ClassifyRollup`; tests |
| NAV-01 | 01-01, 01-03 | j/k and arrows | SATISFIED | `Update`; `TestEndToEndFixture`, `TestScrollFollowsCursor` |
| NAV-04 | 01-01, 01-02 | q quits | SATISFIED | `TestQuitInEveryState`; user-confirmed terminal restore |

### Anti-Patterns Found

None blocking. A grep for TBD, FIXME, XXX, TODO and HACK in `cmd/`, `internal/` and `README.md` found nothing. The code review (`01-REVIEW.md`, 0 critical, 3 warning, 4 info, all dispositioned `open`) is advisory and none of the findings violates a must-have. Two warnings are worth noting because they touch the "never look complete when partial" intent:

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/ui/view.go` | render empty state | WR-02: "No open PRs..." shown even when the fetch was partial or truncated | Warning | An empty result from a degraded fetch reads as complete |
| `internal/ui/view.go` | footer, `cut` | WR-03: "list truncated" and "some data could not be loaded" come last in the footer, so narrow panes cut them first | Warning | Degradation notice can be hidden below about 65 columns |
| `cmd/prv/main.go` | 57 | WR-01: hidden `--closed-days` has no upper bound; overflow silently drops closed PRs | Warning | Hidden flag only |

### Human Verification Required

See the `human_verification` frontmatter list. Five items: live assignee evidence (backstop truth), badge fidelity against github.com on real rows, light/dark theme and narrow-pane rendering, the real 401 flow, and ambiguous-width glyph alignment.

### Gaps Summary

No gaps. Every ROADMAP success criterion is backed by code I read and tests I ran. Status is `human_needed` because Step 8 produced items: the one plan-declared backstop truth (live `assignee:@me`) and the human-only visual and terminal checks that the SUMMARYs deferred. The three open code review warnings are advisory; consider triaging WR-02 and WR-03 in Phase 2, when the footer and refresh state get reworked.

---

_Verified: 2026-10-12_
_Verifier: Claude (gsd-verifier)_

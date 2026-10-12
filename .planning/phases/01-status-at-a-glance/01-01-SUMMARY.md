---
phase: 01-status-at-a-glance
plan: 01
subsystem: ui
tags: [go, bubbletea-v2, go-gh, graphql, walking-skeleton, tracer, lipgloss-table]

requires: []
provides:
  - "Go module github.com/sbartram/prv (go 1.26.0, no toolchain line) with the cmd/prv layout"
  - "prv binary: token via go-gh TokenForHost, D-11 no-token exit 1, --version, hidden --closed-days"
  - "internal/domain: PR, Ref, State, Review, CI, Snapshot, RateLimit, DeriveReview, ClassifyRollup, Clean"
  - "internal/github: NewClient, Client.Fetch, BuildSearch, ISSUE_ADVANCED discovery query, wire-to-domain mapper"
  - "internal/ui: Fetcher, Options, Model (loading/list/error), list view with REVIEW and CI badge columns"
affects: [01-02, 01-03, 01-04, phase-02, phase-03, phase-04]

actuals:
  tokens: 12276
  tasks: 2
  commits: 3

plan_head_before: 16945038c739fa30993cd6074f2f6b38e1f2f060
plan_head_after: f37e3969ea04e0f77eb7189861c68acb1332d2a8
commits: 3

tech-stack:
  added:
    - "charm.land/bubbletea/v2 v2.1.0"
    - "charm.land/bubbles/v2 v2.2.1 (spinner)"
    - "charm.land/lipgloss/v2 v2.0.6 (table)"
    - "github.com/cli/go-gh/v2 v2.16.2"
    - "github.com/charmbracelet/x/ansi v0.11.9"
  patterns:
    - "Pure Update/View; all I/O inside a tea.Cmd (fetchCmd)"
    - "ui depends on its own Fetcher interface and never imports internal/github"
    - "Zero values of Review and CI are the non-favourable ones (none, no checks); enum mappings are total with an unknown case"
    - "Remote strings cleaned once at the mapper seam with domain.Clean"
    - "Search string travels only as the GraphQL variable q"

key-files:
  created:
    - go.mod
    - go.sum
    - cmd/prv/main.go
    - cmd/prv/main_test.go
    - internal/domain/pr.go
    - internal/domain/status.go
    - internal/domain/clean.go
    - internal/domain/domain_test.go
    - internal/github/client.go
    - internal/github/client_test.go
    - internal/github/testdata/status.json
    - internal/ui/model.go
    - internal/ui/view.go
  modified: []

key-decisions:
  - "go-gh already defangs C0 control characters in API responses (asciisanitizer turns ESC into the visible text ^[), so domain.Clean at the mapper seam is a second layer; the seam is tested both through the client and directly"
  - "State's zero value is StateOpen (order given by the plan's interface block); unknown remote values map to StateUnknown through ParseState, never to open"
  - "Tracer verified end-to-end (fixture e2e, built-binary checks, live test) before the expansion task started"

patterns-established:
  - "Fixture e2e test: fake http.RoundTripper through github.Options.Transport, ui.New, runCmd helper that recurses into tea.BatchMsg and never runs commands returned by Update"
  - "Synthetic GraphQL fixtures generated from the spike shapes; no real repository data, no tokens, no Authorization text"
  - "Pathspec commits (git commit -- <paths>) so pre-staged unrelated files are never swept into a task commit"

requirements-completed: [AUTH-01, AUTH-02, AUTH-03, CONF-02, WATCH-01, WATCH-02, LIST-01, LIST-02, LIST-03, LIST-05, NAV-01, NAV-04]

coverage:
  - id: D1
    description: "prv resolves the token like gh, fetches the viewer's PRs live with the proven ISSUE_ADVANCED query and lists them in the alt screen"
    requirement: "AUTH-01"
    verification:
      - kind: e2e
        ref: "PRV_LIVE=1 go test ./cmd/prv -run TestLiveEndToEnd"
        status: pass
      - kind: integration
        ref: "cmd/prv/main_test.go#TestEndToEndFixture"
        status: pass
    human_judgment: false
  - id: D2
    description: "No token: exactly the D-11 line on stderr, nothing on stdout, exit 1 before the TUI; prv --version works without a token"
    requirement: "AUTH-02"
    verification:
      - kind: unit
        ref: "cmd/prv/main_test.go#TestNoToken"
        status: pass
      - kind: unit
        ref: "cmd/prv/main_test.go#TestVersionFlag"
        status: pass
      - kind: other
        ref: "built binary run with env -i (exit 1, D-11 line on stderr)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Token is never printed, logged or persisted (LogIgnoreEnv true, no cache, token only in Options.Token, fixtures carry a fake token)"
    requirement: "AUTH-03"
    verification:
      - kind: other
        ref: "grep -Eq 'LogIgnoreEnv: *true' internal/github/client.go; grep for token patterns in internal/github/testdata"
        status: pass
    human_judgment: true
    rationale: "Absence of leakage under every code path and under GH_DEBUG is not asserted by an automated test in this plan; TestTokenNeverAppears arrives in 01-02."
  - id: D4
    description: "Each row shows owner/repo#n, title with [draft] prefix, AUTHOR as me/login, and STATE (open, merged, closed, unknown)"
    requirement: "LIST-01"
    verification:
      - kind: integration
        ref: "cmd/prv/main_test.go#TestEndToEndFixture"
        status: pass
    human_judgment: false
  - id: D5
    description: "Review badge derived from reviewDecision with opinionated-review fallback; null never reads required; comment-after-approve stays approved"
    requirement: "LIST-03"
    verification:
      - kind: unit
        ref: "internal/domain/domain_test.go#TestDeriveReviewInvariant"
        status: pass
      - kind: unit
        ref: "internal/github/client_test.go#TestCommentAfterApprove"
        status: pass
      - kind: unit
        ref: "internal/github/client_test.go#TestStatusMapping"
        status: pass
    human_judgment: false
  - id: D6
    description: "CI badge follows GitHub's rollup state only; null is no checks, unknown values are not passing"
    requirement: "LIST-05"
    verification:
      - kind: unit
        ref: "internal/domain/domain_test.go#TestClassifyRollup"
        status: pass
      - kind: unit
        ref: "internal/github/client_test.go#TestStatusMapping"
        status: pass
    human_judgment: false
  - id: D7
    description: "j/k and arrows move the marker with clamping and without panic on zero rows; q and ctrl+c quit"
    requirement: "NAV-01"
    verification:
      - kind: integration
        ref: "cmd/prv/main_test.go#TestEndToEndFixture"
        status: pass
    human_judgment: false
  - id: D8
    description: "Terminal is restored after q or ctrl+c in a real terminal"
    requirement: "NAV-04"
    verification: []
    human_judgment: true
    rationale: "Relies on Bubble Tea's Program.Run (RESEARCH A9); the fixture test only asserts that q yields tea.QuitMsg. Covered by the end-of-phase human check in 01-03."
  - id: D9
    description: "go install path: module github.com/sbartram/prv with go 1.26.0 floor and cmd/prv layout"
    requirement: "CONF-02"
    verification:
      - kind: other
        ref: "grep -qx 'go 1.26.0' go.mod; go build ./cmd/prv"
        status: pass
    human_judgment: true
    rationale: "go install github.com/sbartram/prv/cmd/prv@latest needs a push and is the user-gated checkpoint in 01-04."
  - id: D10
    description: "Remote strings (title, author, repo, viewer) carry no ESC, control or bidi characters after the mapper"
    verification:
      - kind: unit
        ref: "internal/github/client_test.go#TestMapperCleansRemoteStrings"
        status: pass
      - kind: unit
        ref: "internal/github/client_test.go#TestCleanAtSeam"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-10-12
status: complete
---

# Phase 1 Plan 1: Walking Skeleton Summary

**Go + Bubble Tea v2 `prv` that resolves a gh-style token, runs one live `ISSUE_ADVANCED` GraphQL search through go-gh, and lists each PR with state, a review badge derived from `reviewDecision` plus opinionated reviews, and a CI badge taken from GitHub's rollup state**

## Performance

- **Duration:** about 6 min of execution after reading the plan context (start time was not recorded before the read phase)
- **Started:** 2026-10-12T00:25Z (approximate)
- **Completed:** 2026-10-12T00:32Z
- **Tasks:** 2
- **Files modified:** 13 (all created)

## Accomplishments

- Tracer (Task 1): one real path through every layer. `prv` fetches the viewer's authored and assigned PRs live, shows them in the alt screen with PR, TITLE (draft prefix), AUTHOR (`me` or login) and STATE, moves with j/k/arrows (clamped, no panic on zero rows) and quits with `q`/`ctrl+c`. `prv --version` works without a token. With no token it prints exactly the D-11 line to stderr and exits 1 before the TUI.
- Tracer gate: the fixture end-to-end test, the built-binary checks (version and empty-environment exit 1) and the live `TestLiveEndToEnd` all re-ran green before Task 2 started (`Tracer verified end-to-end — expanding`).
- Review status (Task 2): `DeriveReview` uses GitHub's `reviewDecision` when set and falls back to `latestOpinionatedReviews`, so a null decision with an approval shows `✓ approved`, a null decision with nothing shows `– none`, and only `REVIEW_REQUIRED` shows `○ required`. CHANGES_REQUESTED wins over approvals in any order. An invariant test covers every decision against every opinionated-state list of length 0-2 in all orders.
- CI status (Task 2): `ClassifyRollup` follows `statusCheckRollup.state` only (SUCCESS passing, FAILURE/ERROR failing, PENDING/EXPECTED running, null `– no checks`, anything else `? unknown`).
- Sanitising (Task 2): `domain.Clean` is applied once at the mapper seam to title, author login, repository name and viewer login.
- Live proof: the exact search string `is:pr (author:@me OR assignee:@me) (is:open OR closed:>=DATE) sort:updated-desc` (no archive filter) returned a list with `PRV_LIVE=1`.

## Task Commits

1. **Task 1 (tracer): live PR list through every layer** - `8619142` (feat)
2. **Task 2 (TDD): review and CI badges, sanitised strings**
   - RED: `c6981b2` (test) - failing tests, 19-PR `status.json` fixture, zero-value stubs
   - GREEN: `f37e396` (feat) - implementation, mapper wiring, REVIEW and CI columns

**Plan metadata:** committed separately as `docs(01-01): complete walking skeleton plan`.

## TDD Gate Compliance

- **RED:** `c6981b2`. Target tests run against zero-value stubs (`DeriveReview` returned `ReviewNone`, `ClassifyRollup` returned `CINone`, `Badge()` returned `""`, `Clean` returned its input). `go test ./internal/... ./cmd/prv` failed on assertions in `TestDeriveReview` (10 cases), `TestDeriveReviewInvariant`, `TestClassifyRollup` (8 cases), `TestBadge`, `TestClean` (4 cases), `TestStatusMapping`, `TestCommentAfterApprove`, `TestCleanAtSeam`, and `TestEndToEndFixture` (missing `– none`, `– no checks`, `✓ approved`, `✓ passing`). Everything compiled; no setup, import or fixture fault.
- **Semantic assessment:** each target test executed and failed on the planned assertion for the intended reason (behaviour absent), not on a build or fixture error. `gsd_run check tdd-red-evidence` supports TAP, JUnit, swift-testing and unittest report formats, not Go `go test` output, so no machine-classifier record was produced for RED. This is plan `type: execute` with one `tdd="true"` task, so the plan-level gate does not apply.
- **GREEN:** `f37e396`. All listed tests pass.
- **REFACTOR:** none needed.

## Files Created/Modified

- `go.mod`, `go.sum` - module `github.com/sbartram/prv`, `go 1.26.0`, no toolchain line, locked charm.land v2 and go-gh versions
- `cmd/prv/main.go` - flags (`--version`, hidden `--closed-days`), `tokenForHost`, no-token exit, program wiring with `tea.WithContext(ctx)`, exit codes
- `cmd/prv/main_test.go` - `TestVersionFlag`, `TestNoToken`, `TestHiddenClosedDaysFlag`, `TestEndToEndFixture`, `TestLiveEndToEnd` (gated by `PRV_LIVE=1`), `runCmd` helper
- `internal/domain/pr.go` - `PR`, `Ref`, `State`, `Review`, `CI`, `RateLimit`, `Snapshot`, `ParseState`, `DefaultClosedDays`
- `internal/domain/status.go` - `DeriveReview`, `ClassifyRollup`, `Review.Badge`, `CI.Badge`
- `internal/domain/clean.go` - `Clean`
- `internal/domain/domain_test.go` - review, invariant, rollup, badge and clean tests
- `internal/github/client.go` - `Options`, `NewClient`, `Fetch`, `BuildSearch`, `discoveryQuery`, wire structs, mapper
- `internal/github/client_test.go` - fixture transport, `TestStatusMapping`, `TestCommentAfterApprove`, `TestCleanAtSeam`, `TestMapperCleansRemoteStrings`
- `internal/github/testdata/status.json` - synthetic 19-PR response page
- `internal/ui/model.go` - `Fetcher`, `Options`, `Model`, `New`, `Init`, `Update`
- `internal/ui/view.go` - `View` with `AltScreen = true`; loading, error and list rendering with the `lipgloss/table` list

## Decisions Made

- Accepted the plan's interface order for `State`, so the zero value of `State` is `StateOpen`. Unknown remote values still map to `StateUnknown` through `ParseState` and are listed, never shown as open.
- Kept `domain.Clean` at the mapper seam even though go-gh already neutralises control characters (see deviation 1). Bidi overrides and tabs are not handled by go-gh, and the seam must not depend on a transport detail.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TestCleanAtSeam expectations assumed raw control characters reach the mapper**
- **Found during:** Task 2 (GREEN)
- **Issue:** The plan expects PR 18 to map to Title `eviltitle x`, Author `badguy`, Repo `acme/repo` when served through the client. go-gh v2.16.2 wraps the transport in an `asciisanitizer` that turns ESC and BEL in JSON responses into visible caret text (`^[`, `^G`) before the mapper runs, so the mapped strings were `evil^[]0;pwned^Gtitle x`, `bad^[[31mguy` and `acme/^[[2Jrepo`. They are harmless to the terminal but do not equal the planned values.
- **Fix:** `TestCleanAtSeam` now asserts the outcome through the real client (no control or bidi characters in any mapped string, tab turned into a space). A new `TestMapperCleansRemoteStrings` decodes the same fixture straight into the wire structs, bypassing go-gh, and asserts the exact planned values `eviltitle x`, `badguy`, `acme/repo`. The truth "no ESC, C0/C1 control or bidi override character survives the mapper" holds and is tested both ways.
- **Files modified:** `internal/github/client_test.go`
- **Verification:** `go test ./internal/github -run 'TestCleanAtSeam|TestMapperCleansRemoteStrings'` passes; the `-run 'TestStatusMapping|TestCommentAfterApprove|TestCleanAtSeam'` acceptance command reports three PASS lines.
- **Committed in:** `f37e396`

**2. [Rule 1 - Bug] Task 1 commit swept in two pre-staged files**
- **Found during:** Task 1 commit
- **Issue:** `.serena/.gitignore` and `.serena/project.yml` were already staged before the plan began; a plain `git commit` included them. The orchestrator said to leave them alone.
- **Fix:** Soft-reset my own just-made commit, unstaged those two files, recommitted only plan files, then re-staged the `.serena` files so the index matches the starting state. Later commits used `git commit -- <paths>`. The abandoned commit was never pushed or referenced.
- **Files modified:** none (history of the feature branch only)
- **Verification:** `git show --stat HEAD` lists only plan files; `git status --short` shows `.serena/*` still staged.
- **Committed in:** `8619142`

---

**Total deviations:** 2 auto-fixed (2 Rule 1)
**Impact on plan:** The first is a test-expectation correction with equal or stronger coverage; the second restores the instructed repo state. No scope creep.

## Issues Encountered

- `go mod tidy` ran `go mod edit -go=1.26.0` afterwards as planned; `go.mod` ends with `go 1.26.0` and no toolchain line.
- The hidden `--closed-days` flag is defined with an empty usage string and the custom `Usage` prints only `usage: prv [--version]`.

## Authentication Gates

None. The live test used the existing `gh` login (keyring) via `auth.TokenForHost`.

## Known Stubs

None. No placeholder text or empty data flowing to the UI. The empty-list message (D-10), error hints, sorting, paging and the colour layer are scheduled for 01-02 and 01-03 and are not stubs in this plan.

## Threat Flags

None. All new surface (HTTPS to api.github.com through go-gh, the token entering memory, rendered remote strings) is covered by the plan's threat register (T-01-01 to T-01-07, T-01-SC).

## User Setup Required

None - no external service configuration required. `TestLiveEndToEnd` needs the existing `gh auth login` and runs only with `PRV_LIVE=1`.

## Next Phase Readiness

- 01-02 can build sorting, window filter, paging and error hints on `domain.Snapshot`, `Client.Fetch` and `ui.Fetcher` as defined here.
- 01-03 adds colour, explicit column widths and scrolling; the list currently relies on `lipgloss/table` natural widths and will wrap on narrow terminals.
- Open flagged items carried forward: NAV-04 terminal restoration (human check in 01-03), AUTH-01 precedence relies on go-gh (no test sets all three sources), CONF-02 `go install ...@latest` is user-gated in 01-04.

## Self-Check: PASSED

- All 13 created files exist on disk.
- Commits `8619142`, `c6981b2`, `f37e396` are ancestors of HEAD; RED (`test(01-01)`) precedes GREEN (`feat(01-01)`) for Task 2.
- `go vet ./...` and `go test ./... -count=1` pass; `PRV_LIVE=1 go test ./cmd/prv -run TestLiveEndToEnd` passes; built-binary version and empty-environment checks pass.
- Task acceptance criteria re-run: go.mod floor and module line, `ISSUE_ADVANCED` present, `isRequired` and the non-opinionated review field absent, exact search-string assertion, `LogIgnoreEnv: true`, ui does not import github, no v1 charm import paths, `tea.WithContext(ctx)`, `AltScreen = true`, four top-level PASS lines for the Task 1 tests, three PASS lines for the status tests, `Badge()` on two lines of `view.go`, no token patterns or `Authorization` in testdata.

---
*Phase: 01-status-at-a-glance*
*Completed: 2026-10-12*

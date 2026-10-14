---
phase: 01-status-at-a-glance
plan: 02
subsystem: ui
tags: [go, bubbletea-v2, go-gh, graphql, pagination, error-handling]

requires:
  - phase: 01-status-at-a-glance
    provides: "01-01 walking skeleton: domain types, Client.Fetch, ui Fetcher and loading/list/error model"
provides:
  - "domain.InWindow (exact, boundary-inclusive closed window) and domain.Sort (Open, Merged, Closed, unknown; updatedAt desc, repo asc, number asc)"
  - "Fetch pages up to 5x100 results, de-duplicates by node id, sets Truncated at the cap, sums rate-limit cost"
  - "github.Error (Kind, Status, Message, Hint) and classify for 401, 403/429, 5xx, network, GraphQL and unknown failures"
  - "Partial data kept with a warning; later-page failure keeps earlier pages and marks the list truncated"
  - "ui states: spinner, error panel with hint and r retry, empty state naming the account, footer notes for truncated or partial lists, panic-safe fetchCmd"
affects: [01-03, 01-04, phase-02]

actuals:
  tokens: 12088
  tasks: 2
  commits: 4

plan_head_before: 76b7cfa9b8842f915f23352bbccafad2c6ac76c4
plan_head_after: f72873ea1fe34dcb8887f8cdbafbdcd79abfbf3a
commits: 4

tech-stack:
  added: []
  patterns:
    - "ui finds Hint() through errors.As into its own unexported hinter interface; it never imports internal/github"
    - "Error text is built only from HTTP status, fixed strings and domain.Clean(API message); the underlying error is kept only for Unwrap"
    - "A canceled context passes through classify unchanged and is ignored by the model, so quitting never shows an error panel"
    - "fetchCmd recovers a panic into a fixed generic error; the panic value is never rendered"
    - "Fetch computes now once so every page and the window filter use the same instant"

key-files:
  created:
    - internal/domain/list.go
    - internal/github/errors.go
    - internal/github/errors_test.go
    - internal/github/testdata/search_page1.json
    - internal/github/testdata/search_page2.json
    - internal/ui/model_test.go
  modified:
    - internal/domain/domain_test.go
    - internal/github/client.go
    - internal/github/client_test.go
    - internal/ui/model.go
    - internal/ui/view.go

key-decisions:
  - "A GraphQL error on page 2 or later discards that page even if it carried data: earlier pages stay, Truncated is set and the warning says only part of the list loaded (plan rule: later-page errors of any kind)"
  - "A canceled context on a later page is returned as-is instead of being turned into a truncation warning, so a quit never produces a half-list message"
  - "No token scrubbing of API messages: messages are built from status plus cleaned API text per the plan; an API that echoed the token in its message is not covered"

patterns-established:
  - "RED commits compile against stubs and fail on assertions; the ui RED run aborted on the unrecovered panic test, which is itself a valid failure"
  - "Pathspec commits (git commit -- <paths>) keep pre-staged .serena files out of every commit"

requirements-completed: [WATCH-01, WATCH-02, WATCH-03, LIST-01, LIST-02, AUTH-03, NAV-04]

coverage:
  - id: D1
    description: "Authored and assigned PRs both listed; an assigned PR shows its author's login; same number in two repositories gives two rows"
    requirement: "WATCH-02"
    verification:
      - kind: unit
        ref: "internal/github/client_test.go#TestMembership"
        status: pass
    human_judgment: false
  - id: D2
    description: "A PR returned twice (same node id, on one page or across pages) appears exactly once"
    requirement: "WATCH-03"
    verification:
      - kind: unit
        ref: "internal/github/client_test.go#TestMembership"
        status: pass
    human_judgment: false
  - id: D3
    description: "Closed window is exact and inclusive: a PR closed exactly 14 days ago is kept, one second earlier is dropped; open and unknown-state PRs are always kept"
    requirement: "LIST-02"
    verification:
      - kind: unit
        ref: "internal/domain/domain_test.go#TestInWindow"
        status: pass
      - kind: unit
        ref: "internal/github/client_test.go#TestMembership"
        status: pass
    human_judgment: false
  - id: D4
    description: "Rows ordered Open, Merged, Closed, unknown; updatedAt newest first, then repo ascending, then number ascending as an integer; stable for equal keys"
    requirement: "LIST-01"
    verification:
      - kind: unit
        ref: "internal/domain/domain_test.go#TestSort"
        status: pass
      - kind: unit
        ref: "internal/github/client_test.go#TestMembership"
        status: pass
    human_judgment: false
  - id: D5
    description: "Paging reads at most 5 pages of 100, follows endCursor, flags Truncated at the cap, and the footer says 'list truncated'; rate-limit cost is summed and the last page's remaining and reset are kept"
    requirement: "WATCH-01"
    verification:
      - kind: unit
        ref: "internal/github/client_test.go#TestPagination"
        status: pass
      - kind: unit
        ref: "internal/github/client_test.go#TestTruncatedAfterFivePages"
        status: pass
      - kind: unit
        ref: "internal/github/client_test.go#TestRateLimitDecoded"
        status: pass
      - kind: unit
        ref: "internal/ui/model_test.go#TestFooterNotes"
        status: pass
    human_judgment: false
  - id: D6
    description: "Failures classified with an actionable hint (401 names gh auth login, GH_TOKEN and GITHUB_TOKEN; 403/429 name rate limit and SSO; 5xx; network); no fallback to another token source"
    requirement: "AUTH-03"
    verification:
      - kind: unit
        ref: "internal/github/errors_test.go#TestClassify"
        status: pass
      - kind: other
        ref: "temporary live test with an invalid token returned 'HTTP 401: Bad credentials' with the 401 hint (file removed after the run)"
        status: pass
    human_judgment: false
  - id: D7
    description: "Partial data (errors together with data) shows the loaded rows and a warning; a later-page failure keeps earlier pages and marks the list truncated; data null with errors is a KindGraphQL error"
    requirement: "WATCH-01"
    verification:
      - kind: unit
        ref: "internal/github/errors_test.go#TestPartialData"
        status: pass
      - kind: unit
        ref: "internal/github/errors_test.go#TestLaterPageFailure"
        status: pass
      - kind: unit
        ref: "internal/github/errors_test.go#TestGraphQLErrorNoData"
        status: pass
    human_judgment: false
  - id: D8
    description: "The token appears in no error message, hint or warning, including a transport error that carries the Authorization header"
    requirement: "AUTH-03"
    verification:
      - kind: unit
        ref: "internal/github/errors_test.go#TestTokenNeverAppears"
        status: pass
    human_judgment: false
  - id: D9
    description: "Loading spinner, error panel, r retry once, empty state naming @login with the closed window, panic recovery with a generic message, q and ctrl+c quit in every state"
    requirement: "NAV-04"
    verification:
      - kind: unit
        ref: "internal/ui/model_test.go#TestLoadingView"
        status: pass
      - kind: unit
        ref: "internal/ui/model_test.go#TestQuitInEveryState"
        status: pass
      - kind: unit
        ref: "internal/ui/model_test.go#TestRetryOnlyInError"
        status: pass
      - kind: unit
        ref: "internal/ui/model_test.go#TestEmptyStateNamesAccount"
        status: pass
      - kind: unit
        ref: "internal/ui/model_test.go#TestErrorPanelShowsHint"
        status: pass
      - kind: unit
        ref: "internal/ui/model_test.go#TestFetchPanicShowsErrorPanel"
        status: pass
      - kind: unit
        ref: "internal/ui/model_test.go#TestCanceledFetchIgnored"
        status: pass
    human_judgment: true
    rationale: "Pure Update/View tests cover every state, but terminal restoration and exit status 0 after q in a real terminal, and the look of the panel with a bad GH_TOKEN, need the end-of-phase human check (plan verify block, steps 1 to 4)."
  - id: D10
    description: "Live assigned PRs on the user's own account"
    requirement: "WATCH-02"
    verification: []
    human_judgment: true
    rationale: "Backstop truth in the plan: the account has zero assigned PRs, so live evidence is a stand-in and TestMembership proves the client side."

duration: 4min
completed: 2026-10-12
status: complete
---

# Phase 1 Plan 2: Trustworthy List and Honest States Summary

**The PR list now holds every authored and assigned PR in the exact closed window once each, from up to five pages in a stable Open/Merged/Closed order, and launch, failure, empty and partial states tell the user what is happening and how to fix it without ever showing the token**

## Performance

- **Duration:** 4 min (2026-10-12T00:33:43Z to 00:37:40Z, after reading plan and context)
- **Tasks:** 2 (both TDD, RED then GREEN)
- **Files:** 6 created, 5 modified

## Accomplishments

- List correctness (Task 1): `domain.InWindow` keeps open and unknown-state PRs and keeps merged or closed PRs only at or after `now - closedDays`, boundary inclusive (one second earlier is dropped, a missing close time is dropped). `domain.Sort` orders Open, Merged, Closed, unknown, then newest update, then repo, then number as an integer (`acme/a#9` before `acme/a#10`). `Fetch` pages with `maxPages = 5`, passes the previous `endCursor` as `after`, de-duplicates by node id with first occurrence winning, flags `Truncated` when page 5 still has a next page, sums cost across pages and keeps the last page's remaining and reset.
- Failures (Task 2): `github.Error` carries a kind, status and cleaned message, and a per-kind `Hint()`. 401 says run `gh auth login` and fix or unset `GH_TOKEN` or `GITHUB_TOKEN`; 403 and 429 name the rate limit and SSO; 5xx says GitHub is having problems; network failures say check the connection. There is no retry with another token source.
- Honest partial results: errors together with data keep the rows and add "Some data could not be loaded: ..."; a failed later page keeps earlier pages, sets `Truncated` and adds "Only part of the list could be loaded". The list footer renders both as " · list truncated" and " · some data could not be loaded".
- UI states: empty state with the exact sentence naming `@login` and the closed window; error panel with hint and `r retry · q quit`; `r` acts only in the error state when no retry is running; `fetchCmd` recovers a panic into "unexpected internal error while loading PRs"; a canceled fetch is ignored.
- Live check: a throwaway test against api.github.com with an invalid token produced `GitHub request failed: HTTP 401: Bad credentials` and the 401 hint (the file was deleted, not committed). `PRV_LIVE=1 go test ./cmd/prv -run TestLiveEndToEnd` still passes after paging and filtering.

## Task Commits

1. **Task 1 (TDD): the right PRs, once each, in a stable order**
   - RED: `8ab0ae8` (test) - `InWindow`/`Sort` stubs, page fixtures, `TestInWindow`, `TestSort`, `TestMembership`, `TestPagination`, `TestTruncatedAfterFivePages`, `TestRateLimitDecoded`
   - GREEN: `b3f2228` (feat) - `list.go` and paged `Fetch`
2. **Task 2 (TDD): loading, failure and empty states**
   - RED: `6de477d` (test) - `Error` type with a stub `classify`, github error tests, ui state tests
   - GREEN: `f72873e` (feat) - `classify`, partial-data handling, ui states and footer notes

**Plan metadata:** committed separately as `docs(01-02): complete trustworthy list and honest states plan`.

## TDD Gate Compliance

- **RED (Task 1):** `8ab0ae8`. Everything compiled; `TestInWindow` (4 subtests), `TestSort` (2 subtests), `TestMembership`, `TestPagination`, `TestTruncatedAfterFivePages` and `TestRateLimitDecoded` failed on assertions against the stubs and the single-page `Fetch`.
- **RED (Task 2):** `6de477d`. `TestClassify` (all subtests but the cancel case), `TestPartialData`, `TestGraphQLErrorNoData`, `TestLaterPageFailure`, `TestTokenNeverAppears/leaky`, `TestEmptyStateNamesAccount`, `TestErrorPanelShowsHint` and two `TestRetryOnlyInError` subtests failed on assertions. `TestFetchPanicShowsErrorPanel` aborted the ui test binary with the unrecovered panic, which stopped `TestFooterNotes` and `TestCanceledFetchIgnored` from reporting in that run; they were run on their own and failed on assertions.
- **Semantic assessment:** each target test failed for the intended reason (behaviour absent), not on a build or fixture error. `gsd_run check tdd-red-evidence` does not parse Go test output, so no machine-classifier record exists. The plan is `type: execute` with two `tdd="true"` tasks, so the plan-level gate does not apply.
- **GREEN:** `b3f2228` and `f72873e`. All listed tests pass, `go vet ./...` is clean, and `go test ./... -race -count=1` passes.
- **REFACTOR:** none needed.

## Files Created/Modified

- `internal/domain/list.go` - `InWindow`, `Sort`
- `internal/domain/domain_test.go` - `TestInWindow`, `TestSort`
- `internal/github/client.go` - paged `Fetch`, `maxPages`, window filter, sort, partial-data and later-page handling, `addWarning`
- `internal/github/client_test.go` - `TestMembership`, `TestPagination`, `TestTruncatedAfterFivePages`, `TestRateLimitDecoded`
- `internal/github/errors.go` - `ErrorKind`, `Error`, `Hint`, `classify`
- `internal/github/errors_test.go` - `TestClassify`, `TestPartialData`, `TestGraphQLErrorNoData`, `TestLaterPageFailure`, `TestTokenNeverAppears`
- `internal/github/testdata/search_page1.json`, `search_page2.json` - synthetic pages (no real data, no tokens)
- `internal/ui/model.go` - `stateEmpty`, `r` retry, panic recovery, canceled-fetch handling, j/k only in the list state
- `internal/ui/view.go` - error panel with `hinter`, empty state, footer notes
- `internal/ui/model_test.go` - fake Fetcher, `runCmd`, tests for every state

## Decisions Made

- A GraphQL error on page 2 or later discards that page even when it carried data. The plan says an error on a later page of any kind keeps earlier pages and marks the list truncated; this is the simplest honest reading.
- No token scrubbing is added to API messages. Messages are built from the status, fixed strings and `domain.Clean(API message)` as planned. `TestTokenNeverAppears` covers a transport error that carries the Authorization header, but an API that echoed the token in its own message text would not be redacted.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] Canceled context on a later page**
- **Found during:** Task 2 (GREEN, Fetch error handling)
- **Issue:** The plan turns any later-page error into a truncation warning. On quit, the context is canceled, and that would be reported as "Only part of the list could be loaded" even though the program is exiting.
- **Fix:** A later-page error that wraps `context.Canceled` is returned unchanged, so the model ignores it like a page-1 cancel.
- **Files modified:** `internal/github/client.go`
- **Verification:** `TestClassify/context_canceled_is_passed_through_for_quit` and `TestCanceledFetchIgnored` pass.
- **Committed in:** `f72873e`

---

**Total deviations:** 1 auto-fixed (1 Rule 2)
**Impact on plan:** None on scope; the change keeps quit clean.

## Issues Encountered

- The first version of the cancel test used a transport that ignored the context, so no cancel error arose. The test transport now returns `req.Context().Err()`, as a real transport does.
- Fix attempts per task stayed at zero beyond that test correction.

## Authentication Gates

None. The live 401 check used a deliberately invalid token; the live end-to-end test used the existing `gh` login.

## Known Stubs

None. The `InWindow` and `Sort` stubs and the stub `classify` existed only in the RED commits and are replaced in the GREEN commits.

## Threat Flags

None. All new surface is covered by the plan's threat register: T-01-08 (messages and `TestTokenNeverAppears`), T-01-09 (panic recovery), T-01-10 (no token fallback, 401 hint), T-01-11 (page cap), T-01-12 (footer notes), T-01-13 (`domain.Clean` on API messages), T-01-SC (`go.mod` and `go.sum` unchanged).

## User Setup Required

None. The end-of-phase human check from the plan's verify block is recorded for UAT: run a built binary with `GH_TOKEN=invalid-token` to see the spinner and then the 401 panel, press `r` to see the spinner again, press `q` and check that the prompt returns and `echo $?` prints 0, then run with the normal login and check the empty-state sentence if the account has no PRs in the window.

## Next Phase Readiness

- 01-03 can add colour, explicit column widths, scrolling and narrow-width wrapping. The tests that assert full sentences size the model at 120x40 for that reason.
- Open items carried forward: NAV-04 terminal restoration and exit status 0 need the human check above; live assigned PRs are a stand-in only (the account has none); a PR returned under two different node ids is not handled; silent omission of SSO-protected organizations cannot be detected.
- Accepted Phase 1 limit: discovery reads at most 500 search results, so past that the list shows the 500 most recently updated and the footer says "list truncated".

## Self-Check: PASSED

- All six created files exist on disk.
- Commits `8ab0ae8`, `b3f2228`, `6de477d`, `f72873e` are ancestors of HEAD; RED precedes GREEN in both tasks.
- `go vet ./...` and `go test ./... -count=1` pass, as does `go test ./... -race -count=1`; `PRV_LIVE=1 go test ./cmd/prv -run TestLiveEndToEnd` passes.
- Task acceptance criteria re-run: `maxPages = 5` present, two `domain.InWindow`/`domain.Sort` call sites in `client.go`, `TestMembership`, `TestTruncatedAfterFivePages`, `TestTokenNeverAppears`, `TestQuitInEveryState`, `TestRetryOnlyInError`, `TestEmptyStateNamesAccount` and `TestFooterNotes` pass, `Hint() string` present in `errors.go` and in `internal/ui/view.go`, no import of `internal/github` in `internal/ui`, `recover()` in `model.go`, no token patterns or `Authorization` in testdata, `go.mod` and `go.sum` unchanged.

---
*Phase: 01-status-at-a-glance*
*Completed: 2026-10-12*

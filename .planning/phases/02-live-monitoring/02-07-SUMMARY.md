---
phase: 02-live-monitoring
plan: 07
subsystem: github
tags: [go, rate-limit, graphql, go-gh, http-headers]

requires:
  - phase: 02-live-monitoring
    provides: "backoffer interface, NextDelay policy, pause, refused r and paused header in the ui (02-06); footer budget segment (02-05); Short() reasons for the failure marker (02-04)"
provides:
  - "captureTransport: records exactly Retry-After, X-RateLimit-Remaining and X-RateLimit-Reset of the latest response (3-field rateHeaders)"
  - "parseRetryAfter (non-negative whole seconds, saturating) and parseReset (positive epoch) - anything else is absent"
  - "KindRateLimited with Error.RetryAfter, Error.ResetAt and (*Error).Backoff() (time.Duration, time.Time, bool); Short 'rate limit'; fixed hint"
  - "classify(err, rateHeaders): HTTP 200 primary limit (remaining 0, Retry-After, or error type RATE_LIMITED), 403 with Retry-After or remaining 0, any 429; a bare 403 stays KindForbidden"
  - "Fetch returns the rate-limit error and an empty snapshot on any page, never a truncated list"
  - "TestRateLimitedEndToEnd through the real client and ui"
affects: [02-08, 02-10]

actuals:
  tokens: 11000
  tasks: 2
  commits: 4
plan_head_before: 015854c7ce3637027db9fe42e28aad451b3c4657
plan_head_after: 8aa9f371ce56049dc58187992c6ff32ea748e38a

tech-stack:
  added: []
  patterns:
    - "RoundTripper wrapper below go-gh to read headers that go-gh drops from *GraphQLError"
    - "Rate-limit decision from headers first, error type second, never from message text"

key-files:
  created:
    - internal/github/ratelimit.go
  modified:
    - internal/github/errors.go
    - internal/github/errors_test.go
    - internal/github/client.go
    - cmd/prv/main_test.go

key-decisions:
  - "A rate limit on any discovery page returns an empty snapshot and the error, so the ui keeps the previous data and pauses; other later-page errors keep the truncated-list behaviour"
  - "parseRetryAfter saturates at 2^31 seconds so a hostile header cannot overflow time.Duration; plan 02-06 clamps the accepted wait"
  - "On page 1 a rate-limited result is returned at once even with partial data; the partial-data path applies only to KindGraphQL"

patterns-established:
  - "Only three named response headers are ever kept; no request header is read (AUTH-03)"

requirements-completed: [RFSH-05, RFSH-04]

coverage:
  - id: D1
    description: "HTTP 200 primary limit (header or RATE_LIMITED type) and 403/429 secondary limit are KindRateLimited with RetryAfter/ResetAt from the headers; a 403 without rate headers is KindForbidden with no Backoff"
    requirement: RFSH-05
    verification:
      - kind: unit
        ref: "internal/github/errors_test.go#TestRateLimitDetection"
        status: pass
      - kind: unit
        ref: "internal/github/errors_test.go#TestClassify"
        status: pass
    human_judgment: false
  - id: D2
    description: "Malformed Retry-After (date, negative) and X-RateLimit-Reset (non-number, zero) are treated as absent"
    requirement: RFSH-05
    verification:
      - kind: unit
        ref: "internal/github/errors_test.go#TestParseRetryAfter"
        status: pass
      - kind: unit
        ref: "internal/github/errors_test.go#TestParseReset"
        status: pass
    human_judgment: false
  - id: D3
    description: "The capture record has exactly 3 fields and never holds the Authorization token; error texts never hold the token"
    requirement: RFSH-05
    verification:
      - kind: unit
        ref: "internal/github/errors_test.go#TestCaptureHeadersOnly"
        status: pass
      - kind: unit
        ref: "internal/github/errors_test.go#TestTokenNeverAppears"
        status: pass
    human_judgment: false
  - id: D4
    description: "A rate limit on page 2 is returned as an error with an empty snapshot; another page-2 failure still truncates"
    requirement: RFSH-05
    verification:
      - kind: unit
        ref: "internal/github/errors_test.go#TestLaterPageRateLimit"
        status: pass
      - kind: unit
        ref: "internal/github/errors_test.go#TestLaterPageFailure"
        status: pass
    human_judgment: false
  - id: D5
    description: "End to end through the real client: 'api 4999' in the footer; a rate-limited refresh shows '✗ refresh failed 12:00 (rate limit)' with 'paused until 12:30 (rate limit)' and the rows; a second r sends no request and the footer shows 'rate limited — resumes 12:30'"
    requirement: RFSH-04
    verification:
      - kind: unit
        ref: "cmd/prv/main_test.go#TestRateLimitedEndToEnd"
        status: pass
      - kind: unit
        ref: "cmd/prv/main_test.go#TestEndToEndFixture"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-10-12
status: complete
---

# Phase 2 Plan 07: Real GitHub rate-limit responses pause prv end to end Summary

**A header-capturing transport under go-gh and `KindRateLimited` with `Backoff()` recognise GitHub's HTTP-200 primary limit and its 403/429 secondary limit with their retry-after and reset times, so the 02-06 policy pauses prv on real responses.**

## Performance

- **Duration:** ~8 min
- **Tasks:** 2 (both TDD)
- **Files modified:** 5 (1 new)

## Accomplishments
- `ratelimit.go`: `captureTransport` wraps the base transport (default `http.DefaultTransport`) and keeps three response headers in a 3-field struct; it reads no request header.
- `errors.go`: `classify(err, rateHeaders)` decides a rate limit from headers first and the `RATE_LIMITED` error type second, never from message text. `*Error.Backoff()` feeds the ui `backoffer` interface through `errors.As`, with no import between the packages.
- `client.go`: `Fetch` classifies every error with the latest captured headers; a rate-limited result is returned at once on any page with an empty snapshot.
- `TestRateLimitedEndToEnd` shows the failure marker, the paused marker, the rows kept, the refused `r` (request count stays 2) and the footer text; `TestEndToEndFixture` now shows `api 4999` (fixture has `"limit":5000`).

## Task Commits

1. **Task 1: rate-limit recognition**
   - RED `1cb59c2` (test): build failed on `KindRateLimited`, `rateHeaders`, `captureTransport`
   - GREEN `497e9f8` (feat)
2. **Task 2: later pages and end to end**
   - RED `48e725d` (test): `TestLaterPageRateLimit` failed (`err = <nil>`); the end-to-end test and `api 4999` already passed because the page-1 path was complete in Task 1
   - GREEN `8aa9f37` (feat)

## Deviations from Plan

**1. [Rule 3 - Tooling] Acceptance awk for const order cannot pass**
- **Found during:** Task 1
- **Issue:** `awk '/KindUnknown/{u=NR} ...'` keeps the last `KindUnknown` match in the file, and `kind := KindUnknown` appears after `KindRateLimited` in the const block.
- **Fix:** Checked the order by hand: `KindUnknown` is line 24 and `KindRateLimited` line 25, inside the const block, after it. No code change.

**2. [Rule 3 - Tooling] `gofmt` form of the Backoff assertion**
- **Issue:** Same as 02-06: gofmt writes the compile-time interface assertion over three lines. The `func (e *Error) Backoff() (time.Duration, time.Time, bool)` grep matches.

**3. Existing TestClassify row changed (not a bug)**
- The 429 row expected `KindForbidden` with an "SSO" hint. The plan makes every 429 `KindRateLimited`, so the row now expects `KindRateLimited` and the rate-limit hint.

**Total deviations:** 3 (2 tooling, 1 expected test update). **Impact:** none on scope.

Note: Task 1's text keeps later pages on the Phase 1 behaviour, and the code did so (`page == 1` gate) until the Task 2 GREEN commit removed the gate.

## Issues Encountered
None. Every commit used an explicit pathspec; the staged `.serena` files and the modified `.planning/config.json` stayed out of all commits.

## Known Stubs
None.

## Threat Flags
None beyond the plan's threat model. T-02-07 mitigated (3-field record, no request header read, `TestCaptureHeadersOnly`). T-02-25 mitigated (strict parsers; Retry-After saturates at 2^31 s so it cannot overflow). T-02-10 mitigated (a 403 is limited only with Retry-After or remaining 0). T-02-11 mitigated (fixed message and hint, `TestTokenNeverAppears` extended with rate-limited cases). T-02-SC: `go.mod` and `go.sum` unchanged.

## Next Phase Readiness
Ready for 02-08. `go build ./...`, `go vet ./...` and `go test ./... -count=1 -race` pass.

## Self-Check: PASSED

Commits `1cb59c2`, `497e9f8`, `48e725d`, `8aa9f37` are ancestors of HEAD. `internal/github/ratelimit.go` exists. `grep -c Authorization internal/github/ratelimit.go` is 0; `grep -c EnableCache internal/github/client.go` is 0; `LogIgnoreEnv: true` is present; `captureTransport{` is in client.go. The six-test selection of Task 2 and the eight-test selection of Task 1 report PASS under `-race`.

---
*Phase: 02-live-monitoring*
*Completed: 2026-10-12*

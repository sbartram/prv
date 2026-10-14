---
phase: 04-curated-watch-list
plan: 02
subsystem: watch-list
tags: [go, github-graphql, parser, error-attribution, sso, tdd]

requires:
  - phase: 04-curated-watch-list
    provides: ParseRef, Unavail, aliased lookupRefs, Lookup, WatchSource (plan 04-01)
provides:
  - "domain.SSORequired(org): reason 'SSO: authorize your token for ORG', help naming the org SSO page and gh auth refresh"
  - "github.unavailFor: SSO, not found, or 'could not load (SHORT)' per alias, from that alias's own errors"
  - "lookupRefs attributes errors[].path[0] to its alias; a nested error keeps the PR loaded and adds a snapshot warning"
  - "lookup_mixed.json fixture and a Lookup route in the test transport"
affects: [04-03, 04-04, 04-05, 04-06, 04-07]

plan_head_before: 07c42f31ad680121bd2a58bc413e69ca06fbd782
plan_head_after: 231a4e4217b6d4d2e7e71fdafc4f24418d0d67dc

actuals:
  tokens: 7600
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Reason text is built only from fixed strings, the validated ref owner and domain.Clean'd GitHub text cut to 40 and 200 runes"
    - "Per-alias error grouping by path[0]; errors with no alias are the fallback for a null alias that has none of its own"

key-files:
  created:
    - internal/github/lookup_test.go
    - internal/github/testdata/lookup_mixed.json
  modified:
    - internal/domain/pr.go
    - internal/domain/ref_test.go
    - internal/github/lookup.go
    - internal/github/client_test.go

key-decisions:
  - "The error type is cleaned and cut to 40 runes before it becomes the short reason, so a hostile type cannot carry a terminal escape (T-04-07)"
  - "An SSO failure wins over NOT_FOUND when one alias carries both, then NOT_FOUND, then the first error's type"
  - "An error whose path[0] is not an alias of the chunk is ignored; only errors with no path at all act as the fallback"

patterns-established:
  - "unavailFor is the single place that maps GitHub errors to Unavail, so the add prompt and an unavailable row show the same text"

requirements-completed: [WATCH-04, LIST-09]

coverage:
  - id: D1
    description: "Every accepted and rejected paste form gets its D-04 result: one Ref of owner, repo and number, or exactly one of three reasons"
    requirement: WATCH-04
    verification:
      - kind: unit
        ref: "internal/domain/ref_test.go#TestParseRef"
        status: pass
      - kind: unit
        ref: "internal/domain/ref_test.go#TestValidRef"
        status: pass
    human_judgment: false
  - id: D2
    description: "A saved or pasted PR that GitHub does not return says why: SSO with the fix, not found, or GitHub's own error, per alias"
    requirement: LIST-09
    verification:
      - kind: unit
        ref: "internal/github/lookup_test.go#TestLookupReasons"
        status: pass
      - kind: unit
        ref: "internal/domain/ref_test.go#TestUnavailWording"
        status: pass
      - kind: unit
        ref: "internal/github/lookup_test.go#TestLookupCleans"
        status: pass
    human_judgment: false
  - id: D3
    description: "A request that fails as a whole fails the refresh and leaves no unavailable rows"
    requirement: LIST-09
    verification:
      - kind: unit
        ref: "internal/github/lookup_test.go#TestLookupRequestFailure"
        status: pass
      - kind: unit
        ref: "internal/github/lookup_test.go#TestFetchWatchedRequestFailure"
        status: pass
    human_judgment: false
  - id: D4
    description: "Owner, repo and number travel only as variables; 26 refs are 2 requests in ref order"
    requirement: WATCH-04
    verification:
      - kind: unit
        ref: "internal/github/lookup_test.go#TestBuildLookupVariablesOnly"
        status: pass
      - kind: unit
        ref: "internal/github/lookup_test.go#TestLookupChunks"
        status: pass
    human_judgment: false
  - id: D5
    description: "A real SAML-enforced org answers with the error shape the fixture assumes (RESEARCH A1)"
    verification: []
    human_judgment: true
    rationale: "The account is in no SAML org; the SAML entries in lookup_mixed.json are constructed. The plan's human-check stands for UAT."

duration: 25min
completed: 2026-10-13
status: complete
---

# Phase 04 Plan 02: Every paste and every saved PR that GitHub refuses is explained in its own words Summary

**`lookupRefs` now groups GraphQL errors by alias (`errors[].path[0]`) and `unavailFor` turns each alias's own errors into one of three D-10 reasons: SSO (with the org SSO page and `gh auth refresh`), not found, or `could not load (SHORT)`. The add prompt and an unavailable row show the same text because both go through this path.**

## Performance

- **Duration:** about 25 min
- **Completed:** 2026-10-13
- **Tasks:** 2
- **Files:** 2 created, 4 modified

## Accomplishments

- **Task 1:** `domain.SSORequired(org)` gives Reason `SSO: authorize your token for ORG` and Help `open https://github.com/orgs/ORG/sso to authorize this token, or run gh auth refresh`. `TestParseRef` and `TestValidRef` now hold every D-04 row (all accepted suffixes, hosts, letter cases, the 39 and 40 character owner limits, the `?token=...#frag` privacy row, the lookalike host `github.com.evil.com`, the inner newline). `TestUnavailWording` pins the three reasons.
- **Task 2:** `unavailFor(items, ref)` picks, in order, SSO (a `FORBIDDEN` with `extensions.saml_failure` true or "SAML" in the message), `NOT_FOUND`, then `could not load (type)` with GitHub's cleaned message (200 runes) as Help; `no data` when the alias has no error. `lookupRefs` keeps the request-level rules (cancel, rate limit, non-GraphQL error, error with no data return the error and no rows) and adds snapshot warning `Some data could not be loaded: MESSAGE` for an error under a PR that loaded.
- `lookup_mixed.json` covers a loaded PR, both live NOT_FOUND shapes (`["r1"]`, `["r2","pullRequest"]`), two constructed SAML shapes, a non-SAML FORBIDDEN, another error type, a nested error and a null alias with no error. `fixtureTransport` gained a `Lookup` route (`lookup`, `lookupBodies`).
- `client.go` was not touched (it belongs to plan 04-04 in this wave); `go.mod` and `go.sum` are unchanged.

## TDD Gate Compliance

Both tasks ran RED then GREEN. A Go test that does not compile is not a valid RED, so task 1's RED commit carries a compile stub (`SSORequired` returning the zero value); task 2 needed no stub. `workflow.tdd_mode` is off, so the `tdd-red-evidence` classifier (no Go report adapter) was not applicable. Semantic assessment: each target test failed on the planned assertion.

| Gate | Commit | Failing assertion |
| ---- | ------ | ----------------- |
| RED (task 1) | ad75328 | `TestUnavailWording`: `SSORequired(acme) = {Kind:0 Reason: Help:}, want kind SSO ...` |
| GREEN (task 1) | da02329 | |
| RED (task 2) | b009e4d | `TestLookupReasons`: `PR 3 ... unavailable = {... could not load (GitHub error)}, want {... SSO: authorize your token for samlcorp ...}`, also the missing nested warning; `TestLookupCleans`; `TestFetchUnavailableRows` |
| GREEN (task 2) | 231a4e4 | |

No REFACTOR commit was needed. Note on task 1: plan 04-01 had already built `ParseRef` and `ValidRef` to the D-04 table, so every added parser row passed at once; only the SSO wording was a true RED. The parser rows are regression cover for the plan's must-haves, not new behavior.

## Task Commits

1. Task 1 RED: `ad75328`
2. Task 1 GREEN: `da02329`
3. Task 2 RED: `b009e4d`
4. Task 2 GREEN: `231a4e4`

## Deviations from Plan

**1. [Rule 2 - Security] The error type is cleaned and cut before it becomes the short reason**
- **Found during:** Task 2 GREEN
- **Issue:** The plan lowercases the first item's Type for the short reason but names no cleaning for it. A hostile Type would reach the row text, against T-04-07.
- **Fix:** `unavailFor` runs `domain.Clean` on the type and cuts it to 40 runes. `TestLookupCleans` uses a type with ESC and U+202E.
- **Files modified:** internal/github/lookup.go, internal/github/lookup_test.go
- **Commit:** 231a4e4

No other deviations. No auth gates.

## Known Stubs

None.

## Threat Flags

None beyond the plan's threat model. T-04-06 (variables only: `TestBuildLookupVariablesOnly`), T-04-07 (`TestLookupCleans`), T-04-08 (no header is read beyond the three rate-limit headers; the SSO page URL is built from the validated owner), T-04-09 (`?token=abc#frag` row), T-04-10 (`TestLookupReasons`: one bad ref never changes another alias) and T-04-SC (`git diff --exit-code go.mod go.sum` is clean) are mitigated as planned.

## Open item for UAT

RESEARCH A1 stands: the SAML shapes are constructed. When an account in a SAML-enforced org is available, save or paste a PR from that org before authorizing the token, confirm the row reads `SSO: authorize your token for ORG`, and record the real `errors[]` entry. If the real shape differs, the row reads `could not load (forbidden)` and nothing is lost.

## Verification

- `go vet ./... && go test ./... -count=1 -race`: all packages pass.
- All four domain tests and all eight github tests report `--- PASS`.
- `! grep -rnE 'mutation *[A-Z{(]' internal/github --include='*.go'`, `git diff --exit-code go.mod go.sum` and `gofmt -l .` are clean.
- Acceptance greps print their lines; none of this plan's commits lists `internal/github/client.go`.

## Self-Check: PASSED

- Files: internal/github/lookup_test.go, internal/github/testdata/lookup_mixed.json, internal/domain/ref_test.go found.
- Commits ad75328, da02329, b009e4d, 231a4e4 found on the branch.

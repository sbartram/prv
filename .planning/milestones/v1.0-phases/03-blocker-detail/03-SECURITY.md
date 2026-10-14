---
phase: "03"
slug: "blocker-detail"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-10-12"
---

# Phase 03 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| api.github.com → prv | Check-run, status-context, workflow and app names (chosen by workflow authors, including fork PR authors), reviewer logins, team slugs and merge-state strings | Untrusted display text, enum values |
| api.github.com → prv | Detail responses can be partial (null nodes plus `errors[]` on HTTP 200), truncated, stale on arrival, or rate limited; Retry-After / X-RateLimit-Reset can be hostile | PR detail data, rate-limit timing |
| prv → api.github.com | FailingChecks and Detail queries carry node ids and page cursors and spend the shared token budget | Authenticated read-only GraphQL requests |
| terminal → prv | Any size at any time, including 1x1 and very wide panes | Window size |
| prv → terminal | Remote strings drawn in the CI, REVIEWERS and MERGE cells, compact rows and the full-screen detail view | Cleaned, width-cut display text |
| prv → browser | `o` in the detail view opens a URL after the list may have changed | PR URL |
| prv → user | The detail screen is where the user decides why a PR is blocked; incomplete data must not read as complete | Integrity of displayed status |
| Go module proxy → build | Third-party code enters the binary | Source code (no new modules this phase) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-03-01 | Tampering / Spoofing | mapContext (check, context, workflow, app names) | high | mitigate | `domain.Clean` at the github seam; CI cell capped at `maxCIColW = 24` (cells.go:14); TestFailingCleansNames | closed |
| T-03-05 | Tampering (query injection) | FailingChecks ids | low | mitigate | Ids sent only as the `$ids` variable; TestFailingNames, TestFailingNameEndToEnd | closed |
| T-03-02 | Denial of service | fetchFailing, ciCell | medium | mitigate | 25-id chunks, 20-request cap, `contexts(first: 100)` (checks.go:26); non-rate-limit failure stops step 2; TestFailingChunks, TestFailingStepServerError, TestCICell | closed |
| T-03-03 | Information disclosure | step-2 failure warning | medium | mitigate | Fixed `failingWarning` text (checks.go:202); TestFailingStepServerError, TestTokenNeverAppears | closed |
| T-03-04 | Tampering (integrity) | SummarizeFailing on superseded / truncated data | medium | mitigate | LatestChecks first; complete=false on hasNextPage; TestFailingRerunFixture, TestFailingTruncatedUnnamed | closed |
| T-03-06 | Denial of service | layoutColumns, renderList, renderCompact at extreme sizes | medium | mitigate | Width floors, final cut per line; TestNeverOverflows (1..200 x 1..50) | closed |
| T-03-07 | Tampering (integrity) | column dropping | medium | mitigate | D-13 drop order keeps PR, REVIEW, CI; TestDropOrder, TestCompactRows | closed |
| T-03-08 | Tampering / Spoofing | reviewInputs, requestInputs (logins, slugs) | high | mitigate | `domain.Clean` on every login and slug; TestMapperCleansRemoteStrings, TestCleanAtSeam | closed |
| T-03-09 | Tampering (integrity) | DeriveMerge, MergeReviewers | medium | mitigate | State/draft before mergeStateStatus; unknown never mergeable; TestDeriveMerge, TestMergeReviewers, TestReviewerMapping | closed |
| T-03-10 | Denial of service | union decoding, reviewer lists | medium | mitigate | Nil guards on reviewers/authors; 20-per-PR caps in list query; TestReviewerMapping | closed |
| T-03-19 | Denial of service | reviewersCell, layoutColumns | medium | mitigate | 22-cell cap with "+N"; TestReviewersCell, TestNeverOverflows | closed |
| T-03-20 | Spoofing (integrity) | MERGE and REVIEWERS cells | low | mitigate | Glyph plus word, no nested colour; TestBadgesReadWithoutColour, TestMergeColours | closed |
| T-03-11 | Tampering / Spoofing | FetchDetail mapping | high | mitigate | `domain.Clean` at the seam, `path.Base` after cleaning; TestFetchDetailCleans | closed |
| T-03-12 | Denial of service | Detail paging | medium | mitigate | `contexts(first: 100, after: $after)` (checks.go:222), at most 5 pages, caps of 50; TestFetchDetailPaging | closed |
| T-03-21 | Tampering | per-check links | medium | mitigate | No per-check URL field requested (grep of internal/github/checks.go finds none) | closed |
| T-03-22 | Tampering (integrity) | Detail.Counts | medium | mitigate | Counts from de-duplicated checks; no per-state count fields; TestCountChecks, TestFetchDetailPaging | closed |
| T-03-13 | Tampering | `o` in the detail view | medium | mitigate | Only `detail.pr.URL` through `safeURL` (open.go:24); TestDetailKeys | closed |
| T-03-14 | Information disclosure | detail fetch panic / error text | medium | mitigate | Fixed recover text (refresh.go:62); TestDetailPanicRecovered | closed |
| T-03-23 | Denial of service | detail rendering | low | mitigate | Viewport renders visible lines only; lines cut by cells; TestViewFitsScreen | closed |
| T-03-15 | Spoofing (wrong PR shown) | applyDetail | medium | mitigate | `detailSeq` + ref check drops stale results; TestDetailStaleAfterSwitch | closed |
| T-03-16 | Tampering | `o` after the PR left or moved | medium | mitigate | Opens `detail.pr.URL` via `safeURL`; TestDetailGoneOpensDetailURL | closed |
| T-03-17 | Denial of service (shared budget) | refreshDetail, startDetailFetch | medium | mitigate | One Detail query per successful refresh while open only; TestDetailRefreshes, TestDetailNoOverlap, TestDetailPausedNoRequest, TestDetailRateLimitExtendsPause, TestDetailEndToEnd | closed |
| T-03-18 | Denial of service | renderDetail at extreme sizes | low | mitigate | Viewport sized first; heights 1 and 2 handled; TestNeverOverflows | closed |
| T-03-24 | Tampering (integrity) | FetchDetail with nulled nodes or a kept GraphQL error | medium | mitigate | `Truncated` when Read < Total; kept-error case moved to `Partial` by 03-11 (see T-03-29); TestFetchDetailPartial, TestDetailPartialEndToEnd | closed |
| T-03-25 | Information disclosure | kept GraphQL error message | low | mitigate | Error reduced to a bool; never shown or logged | closed |
| T-03-26 | Denial of service | pauseForRateLimit with hostile delays | medium | mitigate | `domain.ServerWait` clamps to [1 s, 65 min] (backoff.go:11-12,93); TestServerWait, TestDetailRateLimitExtendsPause | closed |
| T-03-27 | Denial of service (shared budget) | reset at or before now | low | mitigate | Pauses for RateLimitFloor (60 s), never zero; TestDetailRateLimitExtendsPause | closed |
| T-03-28 | Spoofing (unknown shown as passing) | checkLines fold, CountChecks | medium | mitigate | CheckOther listed and counted as "N unknown"; fold reads "more passing or skipped"; TestDetailFold, TestCountChecks | closed |
| T-03-29 | Tampering (integrity) | detail view on a kept GraphQL error (WR-05 / G-03-6) | medium | mitigate | Separate `Detail.Partial` flag; fixed notice "some details could not be loaded" (detail.go:275); "no checks" kept; TestFetchDetailPartial, TestDetailKeptErrorEndToEnd, TestDetailPartialEndToEnd, TestDetailPartialNotice, detail-partial goldens | closed |
| T-03-30 | Information disclosure | kept GraphQL error message | low | mitigate | Fixed notice string; error stays a bool | closed |
| T-03-SC | Tampering | Go module installs | high | mitigate | No new modules in any phase-3 plan: `git diff 034f67d HEAD -- go.mod go.sum` is empty | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

No accepted risks.

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-12 | 31 | 31 | 0 | orchestrator — ASVS L1 grep-depth (register authored at plan time; all 39 named tests present; `go test ./...` passes; go.mod/go.sum unchanged since 034f67d) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-12

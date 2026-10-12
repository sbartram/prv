# API Coverage — GitHub GraphQL API (via go-gh v2) for Phase 3

> Full coverage by default. Opt-outs are explicit, reasoned decisions.
> Scope: the GitHub surface prv could use to explain why a watched PR is blocked. Phases 1 and 2 integrated discovery, status, polling and rate limits (see `../01-status-at-a-glance/COVERAGE.md` and `../02-live-monitoring/COVERAGE.md`). Phase 3 re-decides every capability against its own needs: reviewer names, merge readiness, failing check names and the per-PR detail view.

| capability | decision | reason |
|---|---|---|
| auth-token-resolution | INTEGRATE | |
| search-pull-requests | INTEGRATE | |
| search-pagination | INTEGRATE | |
| viewer-login | INTEGRATE | |
| rate-limit-read | INTEGRATE | |
| rate-limit-limit-field | INTEGRATE | |
| rate-limit-response-headers | INTEGRATE | |
| rate-limited-error-classification | INTEGRATE | |
| graphql-partial-errors | INTEGRATE | |
| http-error-classification | INTEGRATE | |
| polling-refresh | INTEGRATE | |
| browser-open | INTEGRATE | |
| pull-request-core-fields | INTEGRATE | |
| pull-request-created-at | INTEGRATE | |
| review-decision | INTEGRATE | |
| latest-opinionated-reviews | INTEGRATE | |
| latest-opinionated-review-authors | INTEGRATE | |
| review-requests | INTEGRATE | |
| requested-reviewer-union | INTEGRATE | |
| latest-reviews | INTEGRATE | |
| merge-state-status | INTEGRATE | |
| mergeable | INTEGRATE | |
| status-check-rollup-state | INTEGRATE | |
| status-check-rollup-contexts | INTEGRATE | |
| check-run-fields | INTEGRATE | |
| check-suite-ordering | INTEGRATE | |
| workflow-run-name-and-file | INTEGRATE | |
| status-context-fields | INTEGRATE | |
| check-run-counts-by-state | OPT-OUT | not needed: the counts include superseded re-runs (RESEARCH F1) and D-03 requires de-duplicated counts at every size; the detail summary counts the de-duplicated contexts it read and discloses a partial read (03-06, 03-07) |
| status-context-counts-by-state | OPT-OUT | not needed: same reason as check-run-counts-by-state |
| contexts-pagination | INTEGRATE | |
| node-batch-lookup | INTEGRATE | |
| node-lookup | INTEGRATE | |
| check-run-is-required | OPT-OUT | explicitly out of scope: dropped for v1 (Phase 1 spike Q10; CONTEXT carried-forward rule) |
| check-run-details-url | OPT-OUT | explicitly out of scope: o opens only the PR URL on https://github.com; fetching per-check links would add an attacker-chosen URL to the screen (03-06 threat T-03-21) |
| status-context-target-url | OPT-OUT | explicitly out of scope: same reason as check-run-details-url |
| check-run-annotations-and-logs | OPT-OUT | explicitly out of scope: CI log viewing is out of scope (REQUIREMENTS.md Out of Scope: open in browser instead) |
| check-suite-latest-filter | OPT-OUT | not needed: CheckRunFilter checkType LATEST de-duplicates inside one suite only, and duplicates occur across suites (RESEARCH F2); prv de-duplicates client-side |
| review-requests-total-count | OPT-OUT | not needed: at most 13 requests were seen per PR (RESEARCH); 20 in the list and 50 in the detail cover it |
| team-members | OPT-OUT | not needed: a team request is shown by its slug or as a hidden team; members are not reviewers until they review |
| review-threads | OPT-OUT | not needed yet: unresolved thread counts are DISP-02 (v2) |
| repository-pull-request-lookup | OPT-OUT | not needed yet: Phase 4 (WATCH-04, LIST-09) |
| graphql-mutations | OPT-OUT | explicitly out of scope: prv is watch-only (REQUIREMENTS.md Out of Scope); a grep gate in 03-02 keeps write documents out of internal/github |
| rest-api | OPT-OUT | explicitly out of scope: GraphQL covers the need in one request per chunk; REST costs several calls per PR |
| rest-rate-limit-endpoint | OPT-OUT | not needed: every query already returns rateLimit { limit cost remaining resetAt } |
| conditional-requests-etag | OPT-OUT | explicitly out of scope: GraphQL POSTs cannot use ETag/304 |
| notifications-inbox | OPT-OUT | explicitly out of scope: REQUIREMENTS.md Out of Scope |
| issues | OPT-OUT | explicitly out of scope: PRs only |
| review-requested-search | OPT-OUT | explicitly out of scope: "owned" means authored + assigned only |
| enterprise-hosts | OPT-OUT | explicitly out of scope: github.com only |
| go-gh-response-cache | OPT-OUT | explicitly out of scope: would show stale check and review status during polling |
| go-gh-http-logging | OPT-OUT | explicitly out of scope: terminal logging corrupts the TUI and can expose request headers (LogIgnoreEnv true) |

## Capability notes

- latest-opinionated-review-authors, review-requests, requested-reviewer-union: the discovery query gains `author { login }` decoding and `reviewRequests(first: 20)` with User, Bot, Mannequin, Team and EnterpriseTeam fragments; a null `requestedReviewer` is a hidden team and counts as pending (03-04 Task 2; rendered by 03-05 Task 2; RESEARCH F6, F7).
- latest-reviews: read only by the Detail query, only to add COMMENTED and DISMISSED reviewers not otherwise listed; list rows never read it (03-06 Task 1; RESEARCH F8; Phase 1 spike Q1).
- merge-state-status, mergeable: decided after the PR state and isDraft; UNKNOWN with CONFLICTING is conflicts (03-04 Task 1; rendered by 03-05 Task 1; D-07 revised; RESEARCH F4, F5).
- node-batch-lookup: `nodes(ids:)` FailingChecks query, at most 25 ids per request, every failing PR in the list (at most 500, the discovery cap, so at most 20 requests), only for PRs whose rollup is FAILURE or ERROR (03-01, 03-02; D-01; spike Q6).
- status-check-rollup-contexts, check-run-fields, check-suite-ordering, status-context-fields: contexts(first: 100) with name, status, conclusion, databaseId, checkSuite createdAt and app name, and StatusContext context and state; re-runs collapse by (kind, workflow, name), newest suite wins (03-01, 03-02; D-03; RESEARCH F1-F3).
- node-lookup, workflow-run-name-and-file, contexts-pagination: the Detail `node(id:)` query, paged up to 5 x 100 contexts; the summary always counts the de-duplicated contexts, and when paging stops early it says how many runs were read of how many exist (03-06, 03-07; D-03, D-08, D-10; refreshed while open by 03-08, D-11).

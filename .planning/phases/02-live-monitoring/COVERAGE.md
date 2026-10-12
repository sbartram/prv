# API Coverage — GitHub GraphQL API (via go-gh v2) for Phase 2

> Full coverage by default. Opt-outs are explicit, reasoned decisions.
> Scope: the GitHub surface prv could use to watch pull requests. Phase 1 integrated discovery and status (see `../01-status-at-a-glance/COVERAGE.md`). Phase 2 re-decides every capability against its own needs: repeated polling, rate-limit awareness, the attention sort and open-in-browser.

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
| pull-request-core-fields | INTEGRATE | |
| pull-request-created-at | INTEGRATE | |
| review-decision | INTEGRATE | |
| latest-opinionated-reviews | INTEGRATE | |
| status-check-rollup-state | INTEGRATE | |
| graphql-partial-errors | INTEGRATE | |
| http-error-classification | INTEGRATE | |
| polling-refresh | INTEGRATE | |
| browser-open | INTEGRATE | |
| latest-reviews | OPT-OUT | explicitly out of scope: hides an approval behind a later comment (Phase 1 spike Q1); latest-opinionated-reviews is used instead |
| status-check-rollup-contexts | OPT-OUT | not needed yet: Phase 3 (LIST-07, DETL-01, decision C3); the list badge uses the rollup state only |
| check-run-is-required | OPT-OUT | explicitly out of scope: needs a PR id or number and nulls the contexts element inside a search (Phase 1 spike Q10) |
| review-requests | OPT-OUT | not needed yet: Phase 3 (LIST-04, DETL-02) |
| merge-state-status | OPT-OUT | not needed yet: Phase 3 (LIST-06) |
| repository-pull-request-lookup | OPT-OUT | not needed yet: Phase 4 (WATCH-04, LIST-09) |
| node-batch-lookup | OPT-OUT | not needed yet: Phase 4 refresh of saved PRs (Phase 1 spike Q6) |
| rest-rate-limit-endpoint | OPT-OUT | not needed: every GraphQL query already returns rateLimit { limit cost remaining resetAt }, so a separate GET /rate_limit call would only spend a request |
| conditional-requests-etag | OPT-OUT | explicitly out of scope: GraphQL POSTs cannot use ETag/304; savings come from one batched query per refresh |
| graphql-mutations | OPT-OUT | explicitly out of scope: prv is watch-only (REQUIREMENTS.md Out of Scope) |
| rest-api | OPT-OUT | explicitly out of scope: GraphQL covers the need in one request per page; REST costs several calls per PR |
| notifications-inbox | OPT-OUT | explicitly out of scope: REQUIREMENTS.md Out of Scope |
| issues | OPT-OUT | explicitly out of scope: PRs only (is:pr stays in the search) |
| review-requested-search | OPT-OUT | explicitly out of scope: "owned" means authored + assigned only |
| enterprise-hosts | OPT-OUT | explicitly out of scope: github.com only; `o` opens only https://github.com URLs |
| go-gh-response-cache | OPT-OUT | explicitly out of scope: would show stale CI and review status for the cache TTL during polling |
| go-gh-http-logging | OPT-OUT | explicitly out of scope: terminal logging corrupts the TUI and can expose request headers (LogIgnoreEnv true) |

## Capability notes

- rate-limit-limit-field: `rateLimit { limit cost remaining resetAt }`; `limit` feeds the faint `api N` footer, the yellow <20% header warning (D-06) and the reserve/pacing rules of `domain.NextDelay` (02-02 Task 1 and 2).
- rate-limit-response-headers: a `captureTransport` under go-gh records exactly `Retry-After`, `X-RateLimit-Remaining` and `X-RateLimit-Reset`, because the exhausted primary limit arrives as HTTP 200 with a `*api.GraphQLError` that carries no headers (02-02 Task 3).
- rate-limited-error-classification: `KindRateLimited` with `RetryAfter`/`ResetAt` and `Backoff()`; 403 without rate headers stays `KindForbidden` (SSO) (02-02 Task 3).
- pull-request-created-at: `createdAt` is the within-bucket sort key that polling does not change (D-02; 02-03 Task 1).
- polling-refresh: the same discovery query re-run on one generation-stamped tick chain, single-flight, interval from config.toml with a 15 s floor (02-01).
- browser-open: go-gh `pkg/browser` built once with `io.Discard`, called from a `tea.Cmd`, gated by an exact https://github.com allow-list (02-05 Task 3).

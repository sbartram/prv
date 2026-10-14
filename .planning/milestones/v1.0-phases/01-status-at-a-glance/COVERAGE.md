# API Coverage — GitHub GraphQL API (via go-gh v2) for Phase 1

> Full coverage by default. Opt-outs are explicit, reasoned decisions.
> Scope: the GitHub surface prv could use to watch pull requests. Phase 1 integrates the read path for discovery and status. Capabilities marked `not needed yet` are owned by the named later phase.

| capability | decision | reason |
|---|---|---|
| auth-token-resolution | INTEGRATE | |
| search-pull-requests | INTEGRATE | |
| search-pagination | INTEGRATE | |
| viewer-login | INTEGRATE | |
| rate-limit-read | INTEGRATE | |
| pull-request-core-fields | INTEGRATE | |
| review-decision | INTEGRATE | |
| latest-opinionated-reviews | INTEGRATE | |
| status-check-rollup-state | INTEGRATE | |
| graphql-partial-errors | INTEGRATE | |
| http-error-classification | INTEGRATE | |
| latest-reviews | OPT-OUT | explicitly out of scope: hides an approval behind a later comment (spike Q1); latest-opinionated-reviews is used instead |
| status-check-rollup-contexts | OPT-OUT | not needed yet: Phase 3 (LIST-07, DETL-01, decision C3); the Phase 1 badge uses the rollup state only |
| check-run-is-required | OPT-OUT | explicitly out of scope: needs a PR id or number and nulls the contexts element inside a search (spike Q10) |
| review-requests | OPT-OUT | not needed yet: Phase 3 (LIST-04, DETL-02) |
| merge-state-status | OPT-OUT | not needed yet: Phase 3 (LIST-06) |
| repository-pull-request-lookup | OPT-OUT | not needed yet: Phase 4 (WATCH-04, LIST-09) |
| node-batch-lookup | OPT-OUT | not needed yet: Phase 4 refresh of saved PRs (spike Q6) |
| browser-open | OPT-OUT | not needed yet: Phase 2 (NAV-02) |
| graphql-mutations | OPT-OUT | explicitly out of scope: prv is watch-only (REQUIREMENTS.md Out of Scope) |
| rest-api | OPT-OUT | explicitly out of scope: GraphQL covers the need in one request; REST costs several calls per PR |
| notifications-inbox | OPT-OUT | explicitly out of scope: REQUIREMENTS.md Out of Scope |
| issues | OPT-OUT | explicitly out of scope: PRs only (is:pr stays in the search) |
| review-requested-search | OPT-OUT | explicitly out of scope: "owned" means authored + assigned only |
| enterprise-hosts | OPT-OUT | explicitly out of scope: github.com only |
| go-gh-response-cache | OPT-OUT | explicitly out of scope: would show stale CI and review status for the cache TTL |
| go-gh-http-logging | OPT-OUT | explicitly out of scope: terminal logging corrupts the TUI and can expose request headers (LogIgnoreEnv true) |

## Capability notes

- auth-token-resolution: go-gh `auth.TokenForHost("github.com")`; GH_TOKEN, GITHUB_TOKEN, hosts.yml, then `gh auth token` (01-01).
- search-pull-requests: `search(type: ISSUE_ADVANCED)` with `is:pr (author:@me OR assignee:@me) (is:open OR closed:>=DATE) sort:updated-desc` (01-01). No archive filter: open PRs in archived repositories stay listed.
- search-pagination: `pageInfo { hasNextPage endCursor }` and `after`, at most 5 pages (500 results, an accepted Phase 1 limit disclosed by the "list truncated" footer) (01-02).
- rate-limit-read: `rateLimit { cost remaining resetAt }`, decoded into `Snapshot.RateLimit`, displayed in Phase 2 (01-02).
- pull-request-core-fields: `id number title url isDraft state closedAt mergedAt updatedAt repository { nameWithOwner } author { login }` (01-01).
- review-decision and latest-opinionated-reviews: inputs to `domain.DeriveReview` (01-01).
- status-check-rollup-state: `statusCheckRollup { state }`, input to `domain.ClassifyRollup` (01-01).
- graphql-partial-errors and http-error-classification: `*api.GraphQLError` with decoded data becomes warnings; `*api.HTTPError` 401/403/429/5xx and network errors become `github.Error` with a hint (01-02).

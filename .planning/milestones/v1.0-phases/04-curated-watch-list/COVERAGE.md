# API Coverage — GitHub GraphQL API (via go-gh v2) for Phase 4

> Full coverage by default. Opt-outs are explicit, reasoned decisions.
> Scope: the GitHub surface prv could use to watch any github.com PR the user names, keep it across restarts, and explain why a saved PR cannot be loaded. Phases 1-3 integrated discovery, status, polling, rate limits, failing check names and the detail view (see `../01-status-at-a-glance/COVERAGE.md`, `../02-live-monitoring/COVERAGE.md`, `../03-blocker-detail/COVERAGE.md`). Phase 4 re-decides every capability against its own needs.

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
| graphql-error-path-attribution | INTEGRATE | |
| graphql-error-type | INTEGRATE | |
| graphql-error-extensions-saml | INTEGRATE | |
| http-error-classification | INTEGRATE | |
| polling-refresh | INTEGRATE | |
| browser-open | INTEGRATE | |
| repository-lookup | INTEGRATE | |
| repository-pull-request-lookup | INTEGRATE | |
| aliased-batch-query | INTEGRATE | |
| graphql-variables | INTEGRATE | |
| pull-request-core-fields | INTEGRATE | |
| pull-request-created-at | INTEGRATE | |
| repository-name-with-owner | INTEGRATE | |
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
| contexts-pagination | INTEGRATE | |
| node-batch-lookup | INTEGRATE | |
| node-lookup | INTEGRATE | |
| resource-by-url | OPT-OUT | not needed: the parser yields owner, repo and number locally; resource(url:) would send the user's pasted text to GitHub and would accept non-PR URLs (D-04) |
| repository-rename-redirect | OPT-OUT | not needed: prv does not depend on GitHub following a rename; a renamed repo that resolves is folded into its row by node ID, one that does not shows as not found (RESEARCH A3) |
| issue-to-pr-redirect | OPT-OUT | explicitly out of scope: D-04 rejects issue links with "that is an issue, not a PR" |
| organization-sso-status-query | OPT-OUT | not needed: SSO is detected from the per-alias FORBIDDEN error (saml_failure or "SAML" in the message); no extra request |
| x-github-sso-header | OPT-OUT | explicitly out of scope: the header carries a one-time authorization URL; captureTransport keeps only Retry-After and two X-RateLimit headers (AUTH-03) |
| check-run-counts-by-state | OPT-OUT | not needed: counts include superseded re-runs (Phase 3 RESEARCH F1); the detail counts de-duplicated contexts |
| status-context-counts-by-state | OPT-OUT | not needed: same reason as check-run-counts-by-state |
| check-run-is-required | OPT-OUT | explicitly out of scope: dropped for v1 (Phase 1 spike Q10) |
| check-run-details-url | OPT-OUT | explicitly out of scope: o opens only the PR URL on https://github.com |
| status-context-target-url | OPT-OUT | explicitly out of scope: same reason as check-run-details-url |
| check-run-annotations-and-logs | OPT-OUT | explicitly out of scope: CI log viewing is out of scope (REQUIREMENTS.md) |
| check-suite-latest-filter | OPT-OUT | not needed: duplicates occur across suites; prv de-duplicates client-side (Phase 3) |
| review-requests-total-count | OPT-OUT | not needed: 20 requests in the list and 50 in the detail cover what was seen (Phase 3) |
| team-members | OPT-OUT | not needed: a team request is shown by its slug or as a hidden team |
| review-threads | OPT-OUT | not needed yet: unresolved thread counts are DISP-02 (v2) |
| graphql-mutations | OPT-OUT | explicitly out of scope: prv is watch-only; the read-only grep gate in internal/github stays (04-02) |
| rest-api | OPT-OUT | explicitly out of scope: one aliased GraphQL request covers 25 saved PRs; REST costs several calls per PR |
| rest-rate-limit-endpoint | OPT-OUT | not needed: every query, Lookup included, returns rateLimit { limit cost remaining resetAt } |
| conditional-requests-etag | OPT-OUT | explicitly out of scope: GraphQL POSTs cannot use ETag/304 |
| notifications-inbox | OPT-OUT | explicitly out of scope: REQUIREMENTS.md Out of Scope |
| issues | OPT-OUT | explicitly out of scope: PRs only |
| review-requested-search | OPT-OUT | explicitly out of scope: "owned" means authored + assigned; other PRs are added by hand (this phase) |
| saved-search-sections | OPT-OUT | explicitly out of scope: the curated watch list is the model, not saved searches (REQUIREMENTS.md, CONTEXT) |
| enterprise-hosts | OPT-OUT | explicitly out of scope: github.com only; other hosts are refused by the parser (D-04) |
| go-gh-response-cache | OPT-OUT | explicitly out of scope: would show stale status during polling |
| go-gh-http-logging | OPT-OUT | explicitly out of scope: terminal logging corrupts the TUI and can expose request headers (LogIgnoreEnv true) |

## Capability notes

- repository-lookup, repository-pull-request-lookup, aliased-batch-query, graphql-variables: one `query Lookup` per chunk of at most 25 saved refs, aliases `rN: repository(owner: $oN, name: $nN) { pullRequest(number: $pN) { same fields as discovery } }`; only the alias index is in the document, owner, repo and number travel as variables (04-01 tracer, 04-02 tests; RESEARCH Pattern 1, live-verified cost 1 at 25 and 50 aliases). Saved PRs that discovery already returned are matched by key and cost no lookup (04-01, 04-04). At most 500 saved entries, so at most 20 Lookup requests per refresh (04-04).
- graphql-error-path-attribution, graphql-error-type, graphql-error-extensions-saml: errors are grouped by `errors[].path[0]`; NOT_FOUND gives "not found", FORBIDDEN with `extensions.saml_failure` or "SAML" in the message gives the SSO reason and hint, anything else gives "could not load (…)"; an error under a loaded PR is a snapshot warning (04-02; spike Q3 shapes; SAML shape constructed, RESEARCH A1).
- repository-name-with-owner: the canonical `nameWithOwner` from the Lookup is the ref saved in watchlist.json and the ref the cursor follows; the saved entry's own ref is kept in `PR.Watched` for removal (04-01, 04-06).
- rate-limit-read, rate-limited-error-classification: every Lookup asks for `rateLimit` and its cost is added to the snapshot; a rate limit or any request-level failure fails the whole refresh (D-12 row 1) and the add prompt honours the running pause (04-01, 04-02, 04-03).
- status-check-rollup-contexts, node-batch-lookup: a saved PR whose rollup fails gets its failing check name through the existing FailingChecks step, both in the refresh and in the add-validation lookup (04-01).
- browser-open: `o` on an unavailable row opens `https://github.com/OWNER/REPO/pull/N` built from the validated saved ref through the existing safeURL gate (04-07).

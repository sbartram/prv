# Phase 1 Live-Verification Spike: Findings

**Run:** 2026-10-11, `gh` 2.102.0, account `sbartram` (token never printed or recorded; hashes only).
**Method:** read-only `gh api graphql` queries plus one throwaway Go program (go-gh v2.16.2) in the session scratchpad. No GitHub data was modified. Total cost about 45 of 5,000 points.
**Source:** `.planning/research/SUMMARY.md` "Live-Verification Spike" (10 questions).

## Account facts that limit the evidence

- `sbartram` has **15 authored PRs, 0 open, 0 assigned, 0 closed in the last 14 days** (6 closed/merged since 2026-01-01). A live `prv` run on this account with the 14-day window shows the **empty state** (D-10). Any human check of list rows needs a wider window or a fixture.
- The account belongs to **no organizations** (`viewer.organizations` is empty), so SAML/SSO behaviour could not be tested (Q4 SSO half, Q3 SAML shape).
- Where the account lacked data, public PRs in `cli/cli` and `pytorch/pytorch` were used as read-only stand-ins. Search qualifiers `author:williammartin` replaced `author:@me` for the open-branch test.

## Results

| # | Question | Answer | Evidence (summary) | Decides |
|---|----------|--------|--------------------|---------|
| Q1 | Does `latestOpinionatedReviews` ignore a later COMMENTED review? | **Yes.** It keeps the earlier APPROVED. `latestReviews` does not (shows COMMENTED). | 5 of 5 cli/cli PRs with approve-then-comment: `latestReviews` = COMMENTED, `latestOpinionatedReviews` = APPROVED, `reviewDecision` = APPROVED. In ~350 sampled PRs, derived-from-opinionated never contradicted a non-null `reviewDecision`. | C4 closed: use `latestOpinionatedReviews`. |
| Q2 | After a re-run, does rollup contexts show only the latest run or both? | **Inconclusive for failed-then-passed re-runs** (no sample found). Contexts are **not** de-duplicated across check suites: one PR listed the same job 4 times (4 suites, all SKIPPED). | 50 cli/cli PRs scanned; no job with both FAILURE and SUCCESS. | Phase 3 (not needed for the Phase 1 badge, which uses the rollup `state`). |
| Q3 | How does the client surface partial data with `errors[]`? | go-gh `DoWithContext` **fills the response struct, then returns `*api.GraphQLError`.** Partial data is usable. NOT_FOUND shape recorded below. SAML FORBIDDEN shape **not observed**. | Go spike: with 1 good alias and 2 bad, `errors.As(err, &*api.GraphQLError)` succeeded, `Errors[i].Type` = `NOT_FOUND`, `Path` = `[c]` and `[b pullRequest]`, and the good alias and `viewer.login` were populated. A bad token returned `*api.HTTPError` status 401 "Bad credentials". | Phase 4 per-alias handling; Phase 1 partial-result policy. |
| Q4 | Does `author:@me` work in GraphQL search? Does search omit SSO PRs? | `author:@me` works (15 results, equals `author:sbartram`). SSO omission **not testable** (no SAML orgs). | See Q8. | Discovery shape (C2). |
| Q5 | What does `gh auth token` return when `GITHUB_TOKEN` is set? | `gh auth token` returns the **env token** when `GH_TOKEN`/`GITHUB_TOKEN` is set. `gh auth token --secure-storage --hostname github.com` (what go-gh runs) **ignores env** and returns the keyring token. go-gh checks `GH_TOKEN`, then `GITHUB_TOKEN`, then `hosts.yml` `oauth_token`, then the `gh` subprocess. | Hash comparison: `GITHUB_TOKEN=fake gh auth token` hash equals the fake token's hash; `--secure-storage` hash equals the real token's hash in both cases. go-gh source read at `pkg/auth/auth.go` lines 36-109. Single host (`github.com`) in `hosts.yml`; token is in the keyring (`oauth_token` count = 0). | Token precedence: go-gh order matches AUTH-01. C1 closed. |
| Q6 | Is the `nodes(ids:)` maximum 100? How do deleted PRs appear? | **Max 100** (101 ids returns `ARGUMENT_LIMIT`: "You may not provide more than 100 node ids; you provided 101."). A bad id gives `null` at that index plus `NOT_FOUND` with `path` `["nodes", <index>]`; other ids still resolve. | Live calls. | Phase 4. |
| Q7 | Real `rateLimit.cost` for a ~25-PR list query? | **Cost 1** for 25 PRs both lean (`nodeCount` 275) and eager (`contexts(first:50)`, `nodeCount` 1275). 100 pytorch contexts x 15 PRs (`nodeCount` 1515) also cost 1. Payload: lean 5.7 KB, eager 165 KB for 25 PRs. Latency 1.7 s vs 1.9 s. | Live calls. | C3 (Phase 3): cost is not a deciding factor; payload and complexity are. |
| Q8 | Does `ISSUE_ADVANCED` return open PRs with `(is:open OR closed:>=DATE)`? | **Yes.** Union semantics verified and server-side de-duplicated. Legacy `ISSUE` returns **0 silently** for any OR query. | cli/cli stand-in: `author:X` = 265, `assignee:X` = 144, both = 141, union query = 268 = 265 + 144 - 141. `(author OR assignee) (is:open OR closed:>=2026-10-01)` = 14 = 9 open + 5 closed; with the legacy type it returned 0. The user's own account with `closed:>=2026-01-01` returned 6 on `ISSUE_ADVANCED` and 0 on `ISSUE`. 100 unique ids in 100 nodes; cursor pagination works (`hasNextPage` true, second page 100 nodes). | C2 closed: single `ISSUE_ADVANCED` search confirmed. |
| Q9 | Does a large matrix truncate contexts? Do counts match? | **Yes.** `contexts(first:100)` returns 100 with `hasNextPage` true when `totalCount` is 131 to 355. `totalCount` = `checkRunCount` + `statusContextCount` in all 15 samples. `checkRunCountsByState` / `statusContextCountsByState` give per-state counts for every state. The rollup `state` stays correct on truncated PRs. | pytorch/pytorch open PRs. | C3 (Phase 3). Badge from `state`. |
| Q10 | Is `isRequired` worth its weight in v1? | **No.** `CheckRun.isRequired` needs `pullRequestId` or `pullRequestNumber`. Used inside a `search` list without an argument it returns `UNPROCESSABLE` ("A pull request ID or pull request number is required.") and **nulls the whole `contexts.nodes` element**. It cannot reference the parent PR's number from inside a search. | Live call. | Drop for v1. |

### Extra findings that change Phase 1 design

1. **`reviewDecision` is null while approvals exist** (49 of ~350 sampled PRs: null decision with APPROVED reviews, e.g. repos with no required-review rule). So the fallback is required, not optional; this is exactly success criterion 3.
2. **`REVIEW_REQUIRED` can appear on CLOSED PRs** (two of the user's own closed PRs). It never appears together with approvals in the samples.
3. **`CANCELLED` check runs make the rollup `FAILURE`.** Examples: `CANCELLED=1 IN_PROGRESS=181` gave rollup `FAILURE`; `CANCELLED=11 IN_PROGRESS=44` gave `FAILURE`. `FAILURE` also wins over in-progress runs. The rollup is `PENDING` only when runs are in progress and nothing has failed or been cancelled. So per D-14 a PR whose only bad check is CANCELLED shows **failing**, because that is what GitHub reports. (The carried-forward rule "CANCELLED is never red" applies to per-check display in Phase 3, not to the list badge.)
4. `statusCheckRollup` on `PullRequest` equals `commits(last:1){nodes{commit{statusCheckRollup}}}` in 50 of 50 cli/cli PRs (0 mismatches); the user's PRs have `null` in both.
5. `statusCheckRollup` is `null` for all 15 of the user's PRs; `reviewDecision` is `null` for 13 of them and `REVIEW_REQUIRED` for 2 closed ones.
6. `latestOpinionatedReviews` only ever contained `APPROVED` or `CHANGES_REQUESTED` (191 and 53 entries). The schema enum for review state is `PENDING, COMMENTED, APPROVED, CHANGES_REQUESTED, DISMISSED`.
7. Bot reviewers can disagree between `latestReviews` and `latestOpinionatedReviews` (a `CHANGES_REQUESTED` bot review appears in `latestReviews` but not in `latestOpinionatedReviews`, and `reviewDecision` followed `latestOpinionatedReviews`).
8. Bot PR authors have `author.__typename` = `Bot` and a login without `[bot]` (for example `dependabot`). `author` is nullable in the schema (deleted users); no live example was found.
9. A search without `is:pr` returns both `Issue` and `PullRequest` nodes; keep `is:pr` and filter with `... on PullRequest`.
10. Very broad global searches (`is:merged review:approved stars:0`) fail with "Resource limits for this query exceeded." The scoped discovery query never did.
11. Invalid token via env: HTTP 401 `Bad credentials` with no fallback to the `gh` login. A stale `GH_TOKEN`/`GITHUB_TOKEN` therefore beats a valid `gh` login.

### Observed shapes

NOT_FOUND on an aliased lookup (HTTP 200):

```json
{"data":{"a":{"pullRequest":{"number":1,"state":"MERGED"}},"b":{"pullRequest":null},"c":null},
 "errors":[
  {"type":"NOT_FOUND","path":["c"],"message":"Could not resolve to a Repository with the name 'cli-no-such-org-xyz/nope'."},
  {"type":"NOT_FOUND","path":["b","pullRequest"],"message":"Could not resolve to a PullRequest with the number of 99999999."}]}
```

Enums read by introspection (`__type`):

- `StatusState`: `EXPECTED`, `ERROR`, `FAILURE`, `PENDING`, `SUCCESS`
- `PullRequestReviewDecision`: `CHANGES_REQUESTED`, `APPROVED`, `REVIEW_REQUIRED`
- `PullRequestState`: `OPEN`, `CLOSED`, `MERGED`
- `PullRequestReviewState`: `PENDING`, `COMMENTED`, `APPROVED`, `CHANGES_REQUESTED`, `DISMISSED`

### The proven Phase 1 query

Executed against the live API with `$q` = `is:pr archived:false (author:@me OR assignee:@me) (is:open OR closed:>=2026-01-01) sort:updated-desc`: returned 6 PRs, cost 1. Executed against the cli/cli stand-in: 13 PRs (9 OPEN, 4 MERGED), cost 1, no errors. Second page by cursor worked.

```graphql
query Watch($q: String!, $after: String) {
  viewer { login }
  rateLimit { cost remaining resetAt }
  search(query: $q, type: ISSUE_ADVANCED, first: 100, after: $after) {
    issueCount
    pageInfo { hasNextPage endCursor }
    nodes {
      __typename
      ... on PullRequest {
        id number title url isDraft state closedAt mergedAt updatedAt
        repository { nameWithOwner }
        author { login }
        reviewDecision
        latestOpinionatedReviews(first: 20) { nodes { state author { login } } }
        statusCheckRollup { state }
      }
    }
  }
}
```

## What was not verified (carry as open risks)

- SAML/SSO: whether search omits PRs from SSO-unauthorized orgs, and the exact `FORBIDDEN` error shape. No SAML org on the account. A user with such an org may see PRs silently missing.
- `assignee:@me` on the user's own account (zero assigned PRs). Verified with a stand-in login only.
- The `is:open` branch against the user's own open PRs (none exist). Verified with a stand-in.
- Re-run de-duplication (Q2).
- Whether GitHub's `closed:>=DATE` is evaluated in UTC (date granularity only; handled by a client-side exact filter plus a one-day query margin).

## Reproduce

```bash
gh api graphql -f q='is:pr archived:false (author:@me OR assignee:@me) (is:open OR closed:>=2026-01-01) sort:updated-desc' \
  -f query='<the query above>'
GITHUB_TOKEN=x gh auth token -h github.com | shasum    # prints the env value
gh auth token --secure-storage --hostname github.com   # ignores env (do not print; compare hashes)
```

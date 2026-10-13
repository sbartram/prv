package github

import (
	"context"
	"errors"
	"path"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/sbartram/prv/internal/domain"
)

// failingQuery is the second step of a refresh (D-01): the check contexts of
// the PRs whose rollup is FAILURE or ERROR. The ids come from the discovery
// response and travel only as the $ids variable, never inside this document.
// The query is read-only.
const failingQuery = `query FailingChecks($ids: [ID!]!) {
  rateLimit { limit cost remaining resetAt }
  nodes(ids: $ids) {
    __typename
    ... on PullRequest {
      id
      statusCheckRollup {
        state
        contexts(first: 100) {
          totalCount
          pageInfo { hasNextPage }
          nodes {
            __typename
            ... on CheckRun { name status conclusion databaseId
              checkSuite { createdAt app { name } workflowRun { workflow { name } } } }
            ... on StatusContext { context state }
          }
        }
      }
    }
  }
}`

// failingChunk is how many failing PRs one FailingChecks request names. Every
// failing PR is looked up, in chunks (D-01): discovery lists at most
// maxPages x 100 PRs, so a refresh sends at most 20 of these requests.
const failingChunk = 25

// Wire structs for the FailingChecks response. Everything nullable is a
// pointer; GraphQL nulls a list element when a nested field errors.

type wireFailResp struct {
	RateLimit struct {
		Limit     int
		Cost      int
		Remaining int
		ResetAt   time.Time
	}
	Nodes []*wireFailNode
}

type wireFailNode struct {
	Typename          string `json:"__typename"`
	ID                string
	StatusCheckRollup *struct {
		State    string
		Contexts wireContexts
	}
}

type wireContexts struct {
	TotalCount int
	PageInfo   struct {
		HasNextPage bool
		EndCursor   string
	}
	Nodes []*wireContext
}

type wireContext struct {
	Typename   string `json:"__typename"`
	Name       string
	Status     string
	Conclusion string
	DatabaseID int64 `json:"databaseId"`
	CheckSuite *wireSuite
	Context    string
	State      string
}

type wireSuite struct {
	CreatedAt   time.Time
	App         *struct{ Name string }
	WorkflowRun *struct {
		Workflow *struct{ Name string }
		// File is requested by the Detail query only; FailingChecks leaves it empty.
		File *struct{ Path string }
	}
}

// mapContext maps one check context. Every remote string is cleaned here, at
// the seam.
func mapContext(c *wireContext) domain.Check {
	ch := domain.Check{ID: c.DatabaseID}
	ch.State, ch.Word = domain.ClassifyCheck(c.Typename, c.Status, c.Conclusion, c.State)
	if c.Typename == "StatusContext" {
		ch.Context = true
		ch.Name = domain.Clean(c.Context)
		return ch
	}
	ch.Name = domain.Clean(c.Name)
	if s := c.CheckSuite; s != nil {
		ch.SuiteCreatedAt = s.CreatedAt
		switch {
		case s.WorkflowRun != nil && s.WorkflowRun.Workflow != nil:
			ch.Workflow = domain.Clean(s.WorkflowRun.Workflow.Name)
		case s.App != nil:
			ch.Workflow = domain.Clean(s.App.Name)
		}
		// Source is for the detail view: the workflow file's base name, else the
		// app name. The path is cleaned before path.Base takes its last element.
		switch {
		case s.WorkflowRun != nil && s.WorkflowRun.File != nil && s.WorkflowRun.File.Path != "":
			ch.Source = path.Base(domain.Clean(s.WorkflowRun.File.Path))
		case s.App != nil:
			ch.Source = domain.Clean(s.App.Name)
		}
	}
	return ch
}

// fetchFailing names the failing check of every PR in snap whose rollup is
// failing, with one FailingChecks request per failingChunk PRs. It sends
// nothing when no PR fails. Errors follow the discovery-page policy: a rate
// limit or a cancelled context fails the whole fetch; any other error keeps
// the discovery data, leaves the rows unnamed and adds one warning.
func (c *Client) fetchFailing(ctx context.Context, snap *domain.Snapshot) error {
	var failing []int
	for i, pr := range snap.PRs {
		if pr.CI == domain.CIFailing {
			failing = append(failing, i)
		}
	}
	for start := 0; start < len(failing); start += failingChunk {
		chunk := failing[start:min(start+failingChunk, len(failing))]
		ids := make([]string, len(chunk))
		byID := make(map[string]int, len(chunk))
		for j, i := range chunk {
			ids[j] = snap.PRs[i].ID
			byID[ids[j]] = i
		}

		var resp wireFailResp
		err := c.gql.DoWithContext(ctx, failingQuery, map[string]any{"ids": ids}, &resp)
		if err != nil {
			err = classify(err, c.capture.latest())
			if errors.Is(err, context.Canceled) {
				return err
			}
			var ce *Error
			isErr := errors.As(err, &ce)
			if isErr && ce.Kind == KindRateLimited {
				return err
			}
			snap.Warnings = addWarning(snap.Warnings, failingWarning)
			// GitHub reports errors together with data: keep the nodes that loaded.
			var ge *api.GraphQLError
			if !isErr || ce.Kind != KindGraphQL || !errors.As(err, &ge) || len(resp.Nodes) == 0 {
				return nil
			}
		}

		snap.RateLimit.Cost += resp.RateLimit.Cost
		if resp.RateLimit.Limit > 0 {
			snap.RateLimit.Limit = resp.RateLimit.Limit
			snap.RateLimit.Remaining = resp.RateLimit.Remaining
			snap.RateLimit.ResetAt = resp.RateLimit.ResetAt
		}

		for _, n := range resp.Nodes {
			if n == nil || n.Typename != "PullRequest" {
				continue
			}
			i, ok := byID[n.ID]
			if !ok {
				continue
			}
			var checks []domain.Check
			complete := false
			if r := n.StatusCheckRollup; r != nil {
				complete = !r.Contexts.PageInfo.HasNextPage
				for _, ctxNode := range r.Contexts.Nodes {
					if ctxNode != nil {
						checks = append(checks, mapContext(ctxNode))
					}
				}
			}
			snap.PRs[i].Failing = domain.SummarizeFailing(domain.LatestChecks(checks), complete)
		}
	}
	return nil
}

// failingWarning is the snapshot warning when check names could not be loaded.
const failingWarning = "Check names could not be loaded"

// detailQuery is the on-demand query for one pull request (D-01): its reviewers
// and every check context, 100 at a time. The id and the page cursor travel only
// as variables. It requests no per-state count selection (D-03: GitHub's numbers
// include superseded runs, RESEARCH F1) and no per-check link field (T-03-21).
// The query is read-only.
const detailQuery = `query Detail($id: ID!, $after: String) {
  rateLimit { limit cost remaining resetAt }
  node(id: $id) {
    __typename
    ... on PullRequest {
      id state isDraft reviewDecision mergeStateStatus mergeable
      latestOpinionatedReviews(first: 50) { nodes { state author { __typename login } } }
      latestReviews(first: 50) { nodes { state author { __typename login } } }
      reviewRequests(first: 50) { nodes { requestedReviewer { __typename
        ... on User { login } ... on Bot { login } ... on Mannequin { login }
        ... on Team { combinedSlug } ... on EnterpriseTeam { combinedSlug } } } }
      statusCheckRollup {
        state
        contexts(first: 100, after: $after) {
          totalCount
          pageInfo { hasNextPage endCursor }
          nodes {
            __typename
            ... on CheckRun { name status conclusion databaseId
              checkSuite { createdAt app { name } workflowRun { workflow { name } file { path } } } }
            ... on StatusContext { context state }
          }
        }
      }
    }
  }
}`

// Wire structs for the Detail response.

type wireDetailResp struct {
	RateLimit struct {
		Limit     int
		Cost      int
		Remaining int
		ResetAt   time.Time
	}
	Node *wireDetailNode
}

type wireDetailNode struct {
	Typename                 string `json:"__typename"`
	ID                       string
	LatestOpinionatedReviews struct{ Nodes []*wireReview }
	LatestReviews            struct{ Nodes []*wireReview }
	ReviewRequests           struct{ Nodes []*wireRequest }
	StatusCheckRollup        *struct {
		State    string
		Contexts wireContexts
	}
}

// detailReviewers maps the reviewers of a Detail node under the list rules,
// plus the commented and dismissed ones from latestReviews.
func detailReviewers(n *wireDetailNode) []domain.Reviewer {
	return domain.MergeReviewers(
		reviewInputs(n.LatestOpinionatedReviews.Nodes),
		requestInputs(n.ReviewRequests.Nodes),
		reviewInputs(n.LatestReviews.Nodes),
	)
}

// errDetailGone is returned when the node is missing or is not a pull request.
func errDetailGone() error {
	return &Error{Kind: KindGraphQL, Message: "this pull request could not be loaded"}
}

// detailPage sends one Detail request and returns its pull request node. Partial
// GraphQL data is kept when the node is a pull request; a rate limit, a
// cancelled context and any other error are returned. A missing node, or one
// that is not a pull request, is errDetailGone.
func (c *Client) detailPage(ctx context.Context, vars map[string]any) (*wireDetailNode, error) {
	var resp wireDetailResp
	err := c.gql.DoWithContext(ctx, detailQuery, vars, &resp)
	if err != nil {
		err = classify(err, c.capture.latest())
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		var ce *Error
		var ge *api.GraphQLError
		keep := errors.As(err, &ce) && ce.Kind == KindGraphQL && errors.As(err, &ge) &&
			resp.Node != nil && resp.Node.Typename == "PullRequest"
		if !keep {
			return nil, err
		}
	}
	if resp.Node == nil || resp.Node.Typename != "PullRequest" {
		return nil, errDetailGone()
	}
	return resp.Node, nil
}

// FetchDetail returns the full detail of one pull request: its de-duplicated,
// sorted checks, every reviewer with their state, and the counts.
//
// Contexts are read 100 at a time for at most maxPages pages. Counts always
// come from the de-duplicated checks of the pages read (D-03), so a run
// superseded on another page is not counted. When the pull request has more
// check runs than were read, a later page fails, or fewer runs were read than
// GitHub reports (for example nodes that GraphQL returned as null), Truncated
// is set and Read and Total say how many runs were read of how many exist. A
// failure on page 1
// is returned; so are a rate limit and a cancelled context on any page.
func (c *Client) FetchDetail(ctx context.Context, id string) (domain.Detail, error) {
	vars := map[string]any{"id": id}
	var d domain.Detail
	var checks []domain.Check
	for page := 1; page <= maxPages; page++ {
		n, err := c.detailPage(ctx, vars)
		if err != nil {
			var ce *Error
			if page == 1 || errors.Is(err, context.Canceled) || (errors.As(err, &ce) && ce.Kind == KindRateLimited) {
				return domain.Detail{}, err
			}
			d.Truncated = true // a later page failed: keep the pages read so far
			break
		}
		if page == 1 {
			d.Reviewers = detailReviewers(n)
		}
		r := n.StatusCheckRollup
		if r == nil {
			break
		}
		if page == 1 {
			d.Total = r.Contexts.TotalCount
		}
		for _, ctxNode := range r.Contexts.Nodes {
			if ctxNode != nil {
				checks = append(checks, mapContext(ctxNode))
				d.Read++
			}
		}
		if !r.Contexts.PageInfo.HasNextPage {
			break
		}
		if page == maxPages || r.Contexts.PageInfo.EndCursor == "" {
			d.Truncated = true
			break
		}
		vars["after"] = r.Contexts.PageInfo.EndCursor
	}
	// Fewer runs read than GitHub reports: some nodes came back null (GraphQL
	// nulls a node it could not resolve) or were never read (G-03-6-WR02).
	if d.Read < d.Total {
		d.Truncated = true
	}
	d.Checks = domain.LatestChecks(checks)
	domain.SortChecks(d.Checks)
	d.Counts = domain.CountChecks(d.Checks)
	return d, nil
}

// Package github fetches the viewer's pull requests from the GitHub GraphQL
// API and maps them into domain types. It is the only package that knows the
// wire format.
package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/sbartram/prv/internal/domain"
)

// Options configures a Client.
type Options struct {
	Token      string
	ClosedDays int
	Now        func() time.Time
	Transport  http.RoundTripper // nil means the default transport; tests inject a fake
}

// Client fetches pull requests. It implements ui.Fetcher.
type Client struct {
	gql        *api.GraphQLClient
	closedDays int
	now        func() time.Time
}

// NewClient builds a Client. The token is passed explicitly and never logged:
// LogIgnoreEnv stops GH_DEBUG from writing HTTP traffic over the screen, and
// the response cache stays off so polling never shows stale data.
func NewClient(opts Options) (*Client, error) {
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:         "github.com",
		AuthToken:    opts.Token,
		Timeout:      20 * time.Second,
		LogIgnoreEnv: true,
		Transport:    opts.Transport,
	})
	if err != nil {
		return nil, fmt.Errorf("creating GitHub client: %w", err)
	}
	c := &Client{gql: gql, closedDays: opts.ClosedDays, now: opts.Now}
	if c.closedDays <= 0 {
		c.closedDays = domain.DefaultClosedDays
	}
	if c.now == nil {
		c.now = time.Now
	}
	return c, nil
}

// discoveryQuery is the query proven live in 01-SPIKE.md. The search string is
// always passed as the variable $q, never concatenated into this document.
const discoveryQuery = `query Watch($q: String!, $after: String) {
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
}`

// BuildSearch returns the search string: every PR the viewer authored or is
// assigned to, open or closed inside the window. DATE is date-granular, so it
// carries one extra day of margin. sort:updated-desc filters nothing; it makes
// a page-capped list keep the most recently updated PRs. There is deliberately
// no archived:false qualifier: open PRs in archived repositories must be listed.
func BuildSearch(now time.Time, closedDays int) string {
	date := now.UTC().AddDate(0, 0, -(closedDays + 1)).Format("2006-01-02")
	return "is:pr (author:@me OR assignee:@me) (is:open OR closed:>=" + date + ") sort:updated-desc"
}

// Wire structs. Everything nullable is a pointer; GraphQL nulls a list element
// when a nested field errors, so nodes may be nil.

type wireResp struct {
	Viewer    struct{ Login string }
	RateLimit struct {
		Cost      int
		Remaining int
		ResetAt   time.Time
	}
	Search struct {
		IssueCount int
		PageInfo   struct {
			HasNextPage bool
			EndCursor   string
		}
		Nodes []*wireNode
	}
}

type wireNode struct {
	Typename       string `json:"__typename"`
	ID             string
	Number         int
	Title          string
	URL            string
	IsDraft        bool
	State          string
	ClosedAt       time.Time
	MergedAt       time.Time
	UpdatedAt      time.Time
	Repository     struct{ NameWithOwner string }
	Author         *struct{ Login string }
	ReviewDecision *string
	// Only the opinionated-review connection is requested on purpose: in the
	// non-opinionated one a later COMMENTED review hides an earlier approval
	// (01-SPIKE.md Q1).
	LatestOpinionatedReviews struct {
		Nodes []*struct{ State string }
	}
	StatusCheckRollup *struct{ State string }
}

// maxPages caps discovery at 5 pages of 100 results (cost 1 each). When more
// remain, the snapshot is marked Truncated so the list is never shown as
// complete.
const maxPages = 5

// Fetch runs the discovery query page by page, de-duplicates by node id, keeps
// the PRs inside the closed window and orders them for display.
func (c *Client) Fetch(ctx context.Context) (domain.Snapshot, error) {
	now := c.now() // one instant for every page and the window filter
	vars := map[string]any{"q": BuildSearch(now, c.closedDays)}

	var snap domain.Snapshot
	seen := map[string]bool{}
	for page := 1; page <= maxPages; page++ {
		var resp wireResp
		err := c.gql.DoWithContext(ctx, discoveryQuery, vars, &resp)
		if err != nil {
			if page > 1 {
				if errors.Is(err, context.Canceled) {
					return domain.Snapshot{}, err
				}
				// Earlier pages stay on screen, and the list says it is partial.
				snap.Truncated = true
				snap.Warnings = addWarning(snap.Warnings, "Only part of the list could be loaded")
				break
			}
			// GitHub reports errors together with data: show the rows that loaded.
			var ge *api.GraphQLError
			if !errors.As(err, &ge) || (resp.Viewer.Login == "" && len(resp.Search.Nodes) == 0) {
				return domain.Snapshot{}, classify(err)
			}
			msg := "Some data could not be loaded"
			if len(ge.Errors) > 0 {
				if m := domain.Clean(ge.Errors[0].Message); m != "" {
					msg += ": " + m
				}
			}
			snap.Warnings = addWarning(snap.Warnings, msg)
		}
		pageSnap := mapResponse(resp)
		if page == 1 {
			snap.Viewer = pageSnap.Viewer
		}
		for _, pr := range pageSnap.PRs {
			if seen[pr.ID] {
				continue
			}
			seen[pr.ID] = true
			snap.PRs = append(snap.PRs, pr)
		}
		snap.RateLimit.Cost += pageSnap.RateLimit.Cost
		snap.RateLimit.Remaining = pageSnap.RateLimit.Remaining
		snap.RateLimit.ResetAt = pageSnap.RateLimit.ResetAt

		if !resp.Search.PageInfo.HasNextPage {
			break
		}
		if page == maxPages {
			snap.Truncated = true
			break
		}
		vars["after"] = resp.Search.PageInfo.EndCursor
	}

	snap.PRs = slices.DeleteFunc(snap.PRs, func(pr domain.PR) bool {
		return !domain.InWindow(pr, now, c.closedDays)
	})
	domain.Sort(snap.PRs)
	return snap, nil
}

// addWarning appends msg unless it is already present.
func addWarning(ws []string, msg string) []string {
	if slices.Contains(ws, msg) {
		return ws
	}
	return append(ws, msg)
}

func mapResponse(resp wireResp) domain.Snapshot {
	snap := domain.Snapshot{
		Viewer: domain.Clean(resp.Viewer.Login),
		RateLimit: domain.RateLimit{
			Cost:      resp.RateLimit.Cost,
			Remaining: resp.RateLimit.Remaining,
			ResetAt:   resp.RateLimit.ResetAt,
		},
	}
	for _, n := range resp.Search.Nodes {
		if n == nil || n.Typename != "PullRequest" {
			continue
		}
		author := "ghost"
		if n.Author != nil {
			author = domain.Clean(n.Author.Login)
		}
		decision := ""
		if n.ReviewDecision != nil {
			decision = *n.ReviewDecision
		}
		var reviewStates []string
		for _, r := range n.LatestOpinionatedReviews.Nodes {
			if r != nil {
				reviewStates = append(reviewStates, r.State)
			}
		}
		var rollup *string
		if n.StatusCheckRollup != nil {
			rollup = &n.StatusCheckRollup.State
		}
		snap.PRs = append(snap.PRs, domain.PR{
			ID:        n.ID,
			Ref:       domain.Ref{Repo: domain.Clean(n.Repository.NameWithOwner), Number: n.Number},
			Title:     domain.Clean(n.Title),
			URL:       n.URL,
			Author:    author,
			IsDraft:   n.IsDraft,
			State:     domain.ParseState(n.State),
			Review:    domain.DeriveReview(decision, reviewStates),
			CI:        domain.ClassifyRollup(rollup),
			UpdatedAt: n.UpdatedAt,
			ClosedAt:  n.ClosedAt,
		})
	}
	return snap
}

package github_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/auth"

	"github.com/sbartram/prv/internal/domain"
	"github.com/sbartram/prv/internal/github"
)

// pinSource is a github.WatchSource that returns fixed refs, so the live test
// takes the same saved-ref lookup path that real watched PRs use.
type pinSource []domain.Ref

func (p pinSource) Load() ([]domain.Ref, []string, error) { return p, nil, nil }

// livePin is one public PR and the domain values prv must map for it.
type livePin struct {
	ref      domain.Ref
	state    domain.State
	review   domain.Review
	decision string // raw reviewDecision recorded on 2026-10-13; "" for null
	ci       domain.CI
	notFound bool
	proves   string
}

// livePins holds the values verified on 2026-10-13. Every pin is merged or
// closed, so its MERGE cell is blank (MergeNone). Open PRs are deliberately not
// pinned: their values drift.
var livePins = []livePin{
	{
		ref: domain.Ref{Repo: "cli/cli", Number: 1}, state: domain.StateMerged,
		review: domain.ReviewNone, decision: "", ci: domain.CINone,
		proves: "stale DIRTY/CONFLICTING still gives a blank MERGE for a merged PR; a null decision with no reviews is never required; no checks is never passing",
	},
	{
		ref: domain.Ref{Repo: "cli/cli", Number: 6568}, state: domain.StateMerged,
		review: domain.ReviewApproved, decision: "APPROVED", ci: domain.CINone,
		proves: "decision APPROVED maps to approved",
	},
	{
		ref: domain.Ref{Repo: "komoot/photon", Number: 254}, state: domain.StateMerged,
		review: domain.ReviewApproved, decision: "", ci: domain.CINone,
		proves: "approved on a repo with no review rule (null decision, one APPROVED review) shows approved, never required (01-UAT #2)",
	},
	{
		ref: domain.Ref{Repo: "cli/cli", Number: 8525}, state: domain.StateClosed,
		review: domain.ReviewChangesRequested, decision: "CHANGES_REQUESTED", ci: domain.CINone,
		proves: "closed, not merged; changes requested",
	},
	{
		ref: domain.Ref{Repo: "cli/cli", Number: 11048}, state: domain.StateMerged,
		review: domain.ReviewRequired, decision: "REVIEW_REQUIRED", ci: domain.CINone,
		proves: "decision REVIEW_REQUIRED wins over the APPROVED review",
	},
	{
		ref: domain.Ref{Repo: "hashicorp/terraform", Number: 17465}, state: domain.StateMerged,
		review: domain.ReviewRequired, decision: "REVIEW_REQUIRED", ci: domain.CIPassing,
		proves: "passing from a legacy commit status that GitHub kept",
	},
	{
		ref: domain.Ref{Repo: "hashicorp/terraform", Number: 20036}, state: domain.StateMerged,
		review: domain.ReviewRequired, decision: "REVIEW_REQUIRED", ci: domain.CIRunning,
		proves: "a status stuck in PENDING maps to running",
	},
	{
		ref: domain.Ref{Repo: "hashicorp/terraform", Number: 5}, state: domain.StateMerged,
		review: domain.ReviewNone, decision: "", ci: domain.CINone,
		proves: "an oldest-style PR still maps",
	},
	{
		ref: domain.Ref{Repo: "cli/cli", Number: 14327}, state: domain.StateMerged,
		review: domain.ReviewApproved, decision: "APPROVED", ci: domain.CIFailing,
		proves: "failing from check runs (expected to expire around October 2027)",
	},
	{
		ref: domain.Ref{Repo: "cli/cli", Number: 14440}, state: domain.StateClosed,
		review: domain.ReviewChangesRequested, decision: "CHANGES_REQUESTED", ci: domain.CIPassing,
		proves: "passing on a recent closed PR (check runs; expires)",
	},
	{
		ref: domain.Ref{Repo: "cli/cli", Number: domain.MaxPRNumber}, notFound: true,
		proves: "the GraphQL Int maximum; no PR will have it, so the row is not found",
	},
}

type verdict int

const (
	verdictOK    verdict = iota // the mapped Review is the pinned one
	verdictDrift                // GitHub recomputed reviewDecision since the pin was recorded
	verdictBug                  // reviewDecision is unchanged, so prv's mapping changed
)

// reviewVerdict decides what a Review value means for a pin. raw is the
// reviewDecision read from GitHub now, "" for null.
func reviewVerdict(pin livePin, got domain.Review, raw string) verdict {
	switch {
	case got == pin.review:
		return verdictOK
	case raw != pin.decision:
		return verdictDrift
	default:
		return verdictBug
	}
}

// rawReviewQuery reads one field, so a Review mismatch can be told apart as
// GitHub drift or a prv bug. It is sent only on a mismatch.
const rawReviewQuery = `query RawReview($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) { reviewDecision }
  }
}`

// rawReviewDecision returns the PR's reviewDecision, "" for null.
func rawReviewDecision(ctx context.Context, gql *api.GraphQLClient, ref domain.Ref) (string, error) {
	owner, name, _ := strings.Cut(ref.Repo, "/")
	var resp struct {
		Repository struct {
			PullRequest *struct {
				ReviewDecision *string
			}
		}
	}
	vars := map[string]any{"owner": owner, "name": name, "number": ref.Number}
	if err := gql.DoWithContext(ctx, rawReviewQuery, vars, &resp); err != nil {
		return "", err
	}
	if resp.Repository.PullRequest == nil {
		return "", errors.New("pull request not found")
	}
	if d := resp.Repository.PullRequest.ReviewDecision; d != nil {
		return *d, nil
	}
	return "", nil
}

// orNull prints a null reviewDecision as "null".
func orNull(s string) string {
	if s == "" {
		return "null"
	}
	return s
}

// TestLive runs prv's real Fetch against github.com and checks the mapped
// state, review, CI and MERGE of 11 pinned public PRs. It is the data level of
// 01-UAT #2 and 03-UAT #4 on real responses.
//
// Run it with:
//
//	PRV_LIVE=1 go test ./internal/github -run TestLive -count=1 -v
//
// -count=1 skips the test cache. Without PRV_LIVE=1 the test skips before it
// looks for a token, so go test ./... stays offline. The token comes from
// GH_TOKEN, GITHUB_TOKEN or gh auth login, as in prv. The test is read-only. It
// costs about 4 API points, plus 1 for each pin whose Review differs.
//
// A skipped subtest names a pin that changed on GitHub: the PR was reopened,
// its reviewDecision was recomputed, or GitHub expired its check data. To
// re-pin, choose another merged or closed PR whose values you have checked on
// github.com or with
//
//	gh api graphql -f query='{ repository(owner: "cli", name: "cli") { pullRequest(number: 1) { reviewDecision } } }'
//
// and update the livePins table, its decision value included. Every Review is
// rule-sensitive, because GitHub computes reviewDecision from the current
// branch rules and reviewer access. A Review failure that says the raw value is
// unchanged is a prv bug, not drift.
//
// Open-PR MERGE states (mergeable, blocked, behind, conflicts, draft) drift, so
// they are left to the manual UAT. The test assumes that the CI rollup of a
// merged or closed PR never changes to a different non-empty value: only expiry
// to "no checks" is expected, and that is a skip.
func TestLive(t *testing.T) {
	if os.Getenv("PRV_LIVE") != "1" {
		t.Skip("set PRV_LIVE=1 to run against github.com (token from GH_TOKEN, GITHUB_TOKEN or gh auth login); read-only, about 4 API points")
	}
	token, _ := auth.TokenForHost("github.com")
	if token == "" {
		t.Fatal("PRV_LIVE=1 but no GitHub token (run gh auth login or set GH_TOKEN)")
	}
	// The same options as internal/github/client.go, so GH_DEBUG never prints
	// HTTP traffic. It sends nothing until a Review mismatch needs it.
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host: "github.com", AuthToken: token, Timeout: 20 * time.Second, LogIgnoreEnv: true,
	})
	if err != nil {
		t.Fatalf("creating the GraphQL client: %v", err)
	}

	refs := make(pinSource, 0, len(livePins))
	for _, p := range livePins {
		refs = append(refs, p.ref)
	}
	client, err := github.NewClient(github.Options{Token: token, ClosedDays: 14, Watch: refs})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	snap, err := client.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// Only the saved rows are checked: the discovery half returns the
	// developer's own PRs, which are not pinned.
	byKey := map[domain.Ref]domain.PR{}
	for _, pr := range snap.PRs {
		if pr.Manual {
			byKey[pr.Watched.Key()] = pr
		}
	}

	for _, pin := range livePins {
		t.Run(strings.ReplaceAll(pin.ref.String(), "/", "_"), func(t *testing.T) {
			pr, ok := byKey[pin.ref.Key()]
			if !ok {
				t.Fatalf("pin %s: no saved row in the snapshot", pin.ref)
			}
			if pin.notFound {
				if pr.Unavail.Kind != domain.UnavailNotFound || pr.Unavail.Reason != domain.NotFound().Reason {
					t.Fatalf("pin %s: Unavail = %v %q, want not found", pin.ref, pr.Unavail.Kind, pr.Unavail.Reason)
				}
				return
			}
			if pr.Unavail.Kind != domain.UnavailNone {
				t.Fatalf("pin %s: row is unavailable: %s", pin.ref, pr.Unavail.Reason)
			}

			// Read the raw decision first, only on a mismatch, so that every
			// skip comes before the first recorded error.
			var (
				v      = verdictOK
				raw    string
				rawErr error
			)
			if pr.Review != pin.review {
				raw, rawErr = rawReviewDecision(ctx, gql, pin.ref)
				if rawErr == nil {
					v = reviewVerdict(pin, pr.Review, raw)
				}
			}
			if pin.state == domain.StateClosed && pr.State != domain.StateClosed {
				t.Skipf("pin %s: reopened on GitHub (now %v); re-pin (see the TestLive doc comment)", pin.ref, pr.State)
			}
			if v == verdictDrift {
				t.Skipf("pin %s: reviewDecision is now %s (was %s on 2026-10-13), so the repository's review rule or a reviewer's access changed; re-pin (see the TestLive doc comment)",
					pin.ref, orNull(raw), orNull(pin.decision))
			}
			if pin.ci != domain.CINone && pr.CI == domain.CINone {
				t.Skipf("pin %s: GitHub expired the check data; re-pin (see the TestLive doc comment)", pin.ref)
			}

			if rawErr != nil {
				t.Errorf("pin %s: Review = %v, want %v (could not read the raw reviewDecision: %v)", pin.ref, pr.Review, pin.review, rawErr)
			}
			if v == verdictBug {
				t.Errorf("pin %s: Review = %v, want %v (reviewDecision is still %s, so prv's mapping changed)", pin.ref, pr.Review, pin.review, orNull(raw))
			}
			if pr.State != pin.state {
				t.Errorf("pin %s: State = %v, want %v", pin.ref, pr.State, pin.state)
			}
			if pr.Merge != domain.MergeNone {
				t.Errorf("pin %s: Merge = %v, want %v", pin.ref, pr.Merge, domain.MergeNone)
			}
			if pr.CI != pin.ci {
				t.Errorf("pin %s: CI = %v, want %v", pin.ref, pr.CI, pin.ci)
			}
		})
	}
}

// TestReviewVerdict proves, offline, how a Review mismatch is classified.
func TestReviewVerdict(t *testing.T) {
	pinFor := func(t *testing.T, repo string, n int) livePin {
		t.Helper()
		for _, p := range livePins {
			if p.ref.Repo == repo && p.ref.Number == n {
				return p
			}
		}
		t.Fatalf("no pin %s#%d", repo, n)
		return livePin{}
	}
	reviewName := map[domain.Review]string{
		domain.ReviewNone: "none", domain.ReviewApproved: "approved",
		domain.ReviewChangesRequested: "changes", domain.ReviewRequired: "required",
	}
	cases := []struct {
		repo string
		n    int
		got  domain.Review
		raw  string
		want verdict
	}{
		{"cli/cli", 1, domain.ReviewRequired, "REVIEW_REQUIRED", verdictDrift},
		{"hashicorp/terraform", 5, domain.ReviewRequired, "", verdictBug},
		{"cli/cli", 6568, domain.ReviewRequired, "REVIEW_REQUIRED", verdictDrift},
		{"komoot/photon", 254, domain.ReviewRequired, "REVIEW_REQUIRED", verdictDrift},
		{"komoot/photon", 254, domain.ReviewNone, "", verdictBug},
		{"cli/cli", 8525, domain.ReviewRequired, "REVIEW_REQUIRED", verdictDrift},
		{"cli/cli", 14440, domain.ReviewApproved, "APPROVED", verdictDrift},
		{"cli/cli", 11048, domain.ReviewApproved, "", verdictDrift},
		{"hashicorp/terraform", 17465, domain.ReviewNone, "", verdictDrift},
		{"hashicorp/terraform", 20036, domain.ReviewApproved, "REVIEW_REQUIRED", verdictBug},
		{"cli/cli", 14327, domain.ReviewApproved, "REVIEW_REQUIRED", verdictOK},
	}
	for _, c := range cases {
		pin := pinFor(t, c.repo, c.n)
		name := strings.ReplaceAll(pin.ref.String(), "/", "_") + "_" + reviewName[c.got] + "_" + orNull(c.raw)
		t.Run(name, func(t *testing.T) {
			if got := reviewVerdict(pin, c.got, c.raw); got != c.want {
				t.Errorf("reviewVerdict(%s, %v, %q) = %v, want %v", pin.ref, c.got, c.raw, got, c.want)
			}
		})
	}
}

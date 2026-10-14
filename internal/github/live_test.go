package github_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

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

// livePins holds the values verified on 2026-10-13.
var livePins = []livePin{
	{
		ref: domain.Ref{Repo: "cli/cli", Number: 1}, state: domain.StateMerged,
		review: domain.ReviewNone, decision: "", ci: domain.CINone,
		proves: "stale DIRTY/CONFLICTING still gives a blank MERGE for a merged PR; a null decision with no reviews is never required; no checks is never passing",
	},
}

// TestLive runs prv's real Fetch against github.com.
func TestLive(t *testing.T) {
	if os.Getenv("PRV_LIVE") != "1" {
		t.Skip("set PRV_LIVE=1 to run against github.com (token from GH_TOKEN, GITHUB_TOKEN or gh auth login); read-only, about 4 API points")
	}
	token, _ := auth.TokenForHost("github.com")
	if token == "" {
		t.Fatal("PRV_LIVE=1 but no GitHub token (run gh auth login or set GH_TOKEN)")
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
			if pr.State != pin.state {
				t.Errorf("pin %s: State = %v, want %v", pin.ref, pr.State, pin.state)
			}
			if pr.Merge != domain.MergeNone {
				t.Errorf("pin %s: Merge = %v, want %v", pin.ref, pr.Merge, domain.MergeNone)
			}
			if pr.Review != pin.review {
				t.Errorf("pin %s: Review = %v, want %v", pin.ref, pr.Review, pin.review)
			}
			if pr.CI != pin.ci {
				t.Errorf("pin %s: CI = %v, want %v", pin.ref, pr.CI, pin.ci)
			}
		})
	}
}

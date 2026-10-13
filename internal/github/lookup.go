package github

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/sbartram/prv/internal/domain"
)

// lookupChunk is how many aliases one Lookup request holds. A 25 or 50 alias
// request cost 1 point live (04-RESEARCH.md).
const lookupChunk = 25

// buildLookup builds the Lookup document for n refs. Only the integer alias
// index is formatted into it: the owner, the repository and the number travel
// as the variables $o<i>, $n<i> and $p<i>, never inside this text (T-04-01).
// The query is read-only.
func buildLookup(n int) string {
	decl := make([]string, 0, n)
	for i := 0; i < n; i++ {
		x := strconv.Itoa(i)
		decl = append(decl, "$o"+x+": String!, $n"+x+": String!, $p"+x+": Int!")
	}
	var b strings.Builder
	b.WriteString("query Lookup(" + strings.Join(decl, ", ") + ") {\n")
	b.WriteString("  rateLimit { limit cost remaining resetAt }\n")
	for i := 0; i < n; i++ {
		x := strconv.Itoa(i)
		b.WriteString("  r" + x + ": repository(owner: $o" + x + ", name: $n" + x + ") { pullRequest(number: $p" + x + ") { " + prFields + " } }\n")
	}
	b.WriteString("}")
	return b.String()
}

// lookupVars returns the variables of a Lookup request for refs: o<i> is the
// owner, n<i> the repository and p<i> the number.
func lookupVars(refs []domain.Ref) map[string]any {
	vars := make(map[string]any, 3*len(refs))
	for i, r := range refs {
		owner, name, _ := strings.Cut(r.Repo, "/")
		x := strconv.Itoa(i)
		vars["o"+x] = owner
		vars["n"+x] = name
		vars["p"+x] = r.Number
	}
	return vars
}

// wireLookupAlias is one alias of the Lookup response. Both the alias and its
// pullRequest are null when the PR cannot be read.
type wireLookupAlias struct {
	PullRequest *wireNode `json:"pullRequest"`
}

type wireRate struct {
	Limit     int
	Cost      int
	Remaining int
	ResetAt   time.Time
}

// runes returns s cut to at most n runes.
func runes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// unavailFor names why a saved PR did not load, from the errors GitHub gave for
// its alias (D-10), in this priority: an SSO failure, then not found, then the
// error GitHub gave. A FORBIDDEN error is an SSO failure when its extensions say
// saml_failure is true or its message mentions SAML (RESEARCH A1: the shape is
// constructed, not recorded). The organization is the owner of ref, never text
// read from a message. Reason and Help are built from fixed strings and cleaned
// GitHub text cut to 40 and 200 runes; no header, token or raw message reaches
// them.
func unavailFor(items []api.GraphQLErrorItem, ref domain.Ref) domain.Unavail {
	for _, it := range items {
		if it.Type != "FORBIDDEN" {
			continue
		}
		if saml, _ := it.Extensions["saml_failure"].(bool); saml || strings.Contains(strings.ToLower(it.Message), "saml") {
			owner, _, _ := strings.Cut(ref.Repo, "/")
			return domain.SSORequired(owner)
		}
	}
	for _, it := range items {
		if it.Type == "NOT_FOUND" {
			return domain.NotFound()
		}
	}
	if len(items) == 0 {
		return domain.LoadFailed("no data", "")
	}
	msg := domain.Clean(items[0].Message)
	short := runes(strings.ReplaceAll(strings.ToLower(domain.Clean(items[0].Type)), "_", " "), 40)
	if short == "" {
		short = runes(msg, 40)
	}
	if short == "" {
		short = "no data"
	}
	return domain.LoadFailed(short, runes(msg, 200))
}

// lookupRefs loads the saved PRs refs in chunks of lookupChunk and returns one
// PR per ref, in ref order. A ref that could not be read comes back as an
// unavailable PR (empty ID, Unavail set) and is never dropped. It sends nothing
// for no refs.
//
// Errors are attributed to an alias by errors[].path[0], so one bad ref never
// changes another ref's result. A null alias takes the reason from its own
// errors (unavailFor), else from the errors that name no alias. An error under a
// pull request that loaded keeps it loaded and adds a snapshot warning.
//
// A request that fails as a whole (network, 5xx, 401, a rate limit, a cancelled
// context, a GraphQL error with no data) returns the error and no PRs: the
// caller fails the whole refresh, so no row turns unavailable for it (D-12).
// Only the three rate-limit headers are ever read from a response (T-04-08).
func (c *Client) lookupRefs(ctx context.Context, refs []domain.Ref, snap *domain.Snapshot) ([]domain.PR, error) {
	var out []domain.PR
	for start := 0; start < len(refs); start += lookupChunk {
		chunk := refs[start:min(start+lookupChunk, len(refs))]

		var resp map[string]json.RawMessage
		var items []api.GraphQLErrorItem
		err := c.gql.DoWithContext(ctx, buildLookup(len(chunk)), lookupVars(chunk), &resp)
		if err != nil {
			err = classify(err, c.capture.latest())
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
			var ce *Error
			isErr := errors.As(err, &ce)
			if isErr && ce.Kind == KindRateLimited {
				return nil, err
			}
			// GitHub reports errors together with data: keep the aliases that loaded.
			var ge *api.GraphQLError
			if !isErr || ce.Kind != KindGraphQL || !errors.As(err, &ge) || len(resp) == 0 {
				return nil, err
			}
			items = ge.Errors
		}

		byAlias := map[string][]api.GraphQLErrorItem{}
		var unattributed []api.GraphQLErrorItem
		for _, it := range items {
			alias := ""
			if len(it.Path) > 0 {
				alias, _ = it.Path[0].(string)
			}
			if alias == "" {
				unattributed = append(unattributed, it)
				continue
			}
			byAlias[alias] = append(byAlias[alias], it)
		}

		var rate wireRate
		if raw, ok := resp["rateLimit"]; ok {
			_ = json.Unmarshal(raw, &rate)
		}
		snap.RateLimit.Cost += rate.Cost
		if rate.Limit > 0 {
			snap.RateLimit.Limit = rate.Limit
			snap.RateLimit.Remaining = rate.Remaining
			snap.RateLimit.ResetAt = rate.ResetAt
		}

		for i, ref := range chunk {
			alias := "r" + strconv.Itoa(i)
			var a *wireLookupAlias
			if raw, ok := resp[alias]; ok {
				_ = json.Unmarshal(raw, &a)
			}
			if a == nil || a.PullRequest == nil {
				own := byAlias[alias]
				if len(own) == 0 {
					own = unattributed
				}
				out = append(out, domain.PR{Ref: ref, Watched: ref, Manual: true, URL: ref.URL(), Unavail: unavailFor(own, ref)})
				continue
			}
			// An error under a PR that loaded (for example its check rollup)
			// does not make the PR unavailable.
			for _, it := range byAlias[alias] {
				msg := "Some data could not be loaded"
				if m := domain.Clean(it.Message); m != "" {
					msg += ": " + m
				}
				snap.Warnings = addWarning(snap.Warnings, msg)
			}
			pr := mapNode(a.PullRequest)
			pr.Manual = true
			pr.Watched = ref
			out = append(out, pr)
		}
	}
	return out, nil
}

// Lookup loads one PR by ref for the add prompt (D-02). A PR that cannot be
// read comes back with Unavail set; an error means the request failed as a
// whole. A PR with failing CI also gets its failing check named, so the new row
// looks like every other. It may run while Fetch runs: the two share only the
// GraphQL client and the header capture, which are safe for concurrent use.
func (c *Client) Lookup(ctx context.Context, ref domain.Ref) (domain.PR, error) {
	var scratch domain.Snapshot
	prs, err := c.lookupRefs(ctx, []domain.Ref{ref}, &scratch)
	if err != nil {
		return domain.PR{}, err
	}
	pr := prs[0]
	if pr.Unavail.Kind == domain.UnavailNone && pr.CI == domain.CIFailing {
		one := domain.Snapshot{PRs: []domain.PR{pr}}
		if err := c.fetchFailing(ctx, &one); err != nil {
			return domain.PR{}, err
		}
		pr = one.PRs[0]
	}
	return pr, nil
}

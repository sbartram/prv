package github

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/sbartram/prv/internal/domain"
)

// ErrorKind says what kind of failure GitHub or the network produced.
type ErrorKind int

const (
	KindAuth ErrorKind = iota
	KindForbidden
	KindServer
	KindNetwork
	KindGraphQL
	KindUnknown
	KindRateLimited
)

// Error is a failed fetch, described for the user. Its text is built only from
// the HTTP status, fixed strings and a cleaned API message: never from
// headers, request dumps or the token.
type Error struct {
	Kind    ErrorKind
	Status  int
	Message string
	err     error

	// RetryAfter and ResetAt are set for KindRateLimited when GitHub sent
	// them; zero means absent.
	RetryAfter time.Duration
	ResetAt    time.Time
}

// Error returns the description shown in the error panel.
func (e *Error) Error() string { return "GitHub request failed: " + e.Message }

// Unwrap returns the underlying error, for errors.Is and errors.As.
func (e *Error) Unwrap() error { return e.err }

// Hint says how to fix the failure.
func (e *Error) Hint() string {
	switch e.Kind {
	case KindAuth:
		return "GitHub rejected the token. Run `gh auth login`. If GH_TOKEN or GITHUB_TOKEN is set, it overrides the gh login: fix or unset it."
	case KindForbidden:
		return "GitHub refused the request. The API rate limit may be used up, or an organization may require SSO authorization for this token. Wait a minute, then press r."
	case KindServer:
		return "GitHub is having problems. Press r to retry."
	case KindNetwork:
		return "Could not reach GitHub. Check your connection, then press r to retry."
	case KindGraphQL:
		return "GitHub returned an error for the PR search. Press r to retry."
	case KindRateLimited:
		return "GitHub's API rate limit is used up. prv pauses and resumes on its own; the header shows when."
	}
	return "Press r to retry. If this keeps happening, run `gh auth status`."
}

// Short names the failure in one or two words for the header (D-07). The text
// is fixed per kind: it never holds the API message or the token.
func (e *Error) Short() string {
	switch e.Kind {
	case KindAuth:
		return "auth"
	case KindForbidden:
		return "forbidden"
	case KindServer:
		return "server"
	case KindNetwork:
		return "network"
	case KindGraphQL:
		return "GitHub error"
	case KindRateLimited:
		return "rate limit"
	}
	return "error"
}

// Backoff reports how long GitHub asked prv to wait (Retry-After), when the
// limit resets, and whether this failure is a rate limit at all. The ui reads
// it through its own interface.
func (e *Error) Backoff() (time.Duration, time.Time, bool) {
	return e.RetryAfter, e.ResetAt, e.Kind == KindRateLimited
}

var (
	_ interface{ Hint() string }  = (*Error)(nil)
	_ interface{ Short() string } = (*Error)(nil)
	_ interface {
		Backoff() (time.Duration, time.Time, bool)
	} = (*Error)(nil)
)

// classify turns a failed request into an *Error. A canceled context is
// returned unchanged: it means prv is quitting. A 401 is never retried with
// another token source: the hint names the variables instead. h holds the
// rate-limit headers of the latest response; a GraphQL error carries none of
// its own.
func classify(err error, h rateHeaders) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	var he *api.HTTPError
	if errors.As(err, &he) {
		kind := KindUnknown
		retryAfter := parseRetryAfter(he.Headers.Get("Retry-After"))
		switch {
		case he.StatusCode == 401:
			kind = KindAuth
		case he.StatusCode == 429:
			kind = KindRateLimited
		case he.StatusCode == 403 && (retryAfter > 0 || he.Headers.Get("X-RateLimit-Remaining") == "0"):
			kind = KindRateLimited
		case he.StatusCode == 403:
			kind = KindForbidden // no rate headers: SSO or permissions, no pause
		case he.StatusCode >= 500:
			kind = KindServer
		}
		msg := fmt.Sprintf("HTTP %d", he.StatusCode)
		if m := domain.Clean(he.Message); m != "" {
			msg += ": " + m
		}
		e := &Error{Kind: kind, Status: he.StatusCode, Message: msg, err: err}
		if kind == KindRateLimited {
			e.RetryAfter = retryAfter
			e.ResetAt = parseReset(he.Headers.Get("X-RateLimit-Reset"))
		}
		return e
	}
	var ge *api.GraphQLError
	if errors.As(err, &ge) {
		if rateLimitedGraphQL(ge, h) {
			return &Error{
				Kind:       KindRateLimited,
				Message:    "GitHub API rate limit exceeded",
				RetryAfter: parseRetryAfter(h.retryAfter),
				ResetAt:    parseReset(h.reset),
				err:        err,
			}
		}
		msg := "GitHub returned an error"
		if len(ge.Errors) > 0 {
			if m := domain.Clean(ge.Errors[0].Message); m != "" {
				msg = m
			}
		}
		return &Error{Kind: KindGraphQL, Message: msg, err: err}
	}
	var ne net.Error
	if errors.As(err, &ne) || errors.Is(err, context.DeadlineExceeded) {
		return &Error{Kind: KindNetwork, Message: "could not reach api.github.com", err: err}
	}
	return &Error{Kind: KindUnknown, Message: "unexpected error", err: err}
}

// rateLimitedGraphQL says whether an HTTP 200 GraphQL error is a rate limit.
// The headers decide first; the RATE_LIMITED error type is a second signal.
// The message text alone never decides.
func rateLimitedGraphQL(ge *api.GraphQLError, h rateHeaders) bool {
	if h.remaining == "0" || parseRetryAfter(h.retryAfter) > 0 {
		return true
	}
	for _, item := range ge.Errors {
		if item.Type == "RATE_LIMITED" {
			return true
		}
	}
	return false
}

package github

import (
	"context"
	"errors"
	"fmt"
	"net"

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
)

// Error is a failed fetch, described for the user. Its text is built only from
// the HTTP status, fixed strings and a cleaned API message: never from
// headers, request dumps or the token.
type Error struct {
	Kind    ErrorKind
	Status  int
	Message string
	err     error
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
	}
	return "Press r to retry. If this keeps happening, run `gh auth status`."
}

var _ interface{ Hint() string } = (*Error)(nil)

// classify turns a failed request into an *Error. A canceled context is
// returned unchanged: it means prv is quitting. A 401 is never retried with
// another token source: the hint names the variables instead.
func classify(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	var he *api.HTTPError
	if errors.As(err, &he) {
		kind := KindUnknown
		switch {
		case he.StatusCode == 401:
			kind = KindAuth
		case he.StatusCode == 403 || he.StatusCode == 429:
			kind = KindForbidden
		case he.StatusCode >= 500:
			kind = KindServer
		}
		msg := fmt.Sprintf("HTTP %d", he.StatusCode)
		if m := domain.Clean(he.Message); m != "" {
			msg += ": " + m
		}
		return &Error{Kind: kind, Status: he.StatusCode, Message: msg, err: err}
	}
	var ge *api.GraphQLError
	if errors.As(err, &ge) {
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

package github

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// rateHeaders holds the three response headers that describe a rate limit.
// It has exactly these fields on purpose: the token travels in a request
// header and no other header is ever kept (AUTH-03).
type rateHeaders struct {
	retryAfter string
	remaining  string
	reset      string
}

// captureTransport records the rate-limit headers of the latest response. go-gh
// returns a *GraphQLError for an HTTP 200 body with errors, and that error
// carries no headers, so the headers are taken here, below go-gh.
type captureTransport struct {
	base http.RoundTripper
	mu   sync.Mutex
	last rateHeaders
}

// RoundTrip sends the request and keeps the three headers of the response. It
// reads no request header.
func (t *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(r)
	var h rateHeaders
	if resp != nil {
		h = rateHeaders{
			retryAfter: resp.Header.Get("Retry-After"),
			remaining:  resp.Header.Get("X-RateLimit-Remaining"),
			reset:      resp.Header.Get("X-RateLimit-Reset"),
		}
	}
	t.mu.Lock()
	t.last = h
	t.mu.Unlock()
	return resp, err
}

// latest returns a copy of the headers of the latest response.
func (t *captureTransport) latest() rateHeaders {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.last
}

const maxRetryAfterSeconds = 1 << 31

// parseRetryAfter reads a Retry-After value as whole seconds. Anything else,
// such as an HTTP date or a negative number, counts as absent (zero).
func parseRetryAfter(s string) time.Duration {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return 0
	}
	// Saturate so a hostile value cannot overflow the Duration; plan 02-06
	// clamps the wait it accepts.
	return time.Duration(min(n, maxRetryAfterSeconds)) * time.Second
}

// parseReset reads an X-RateLimit-Reset value as epoch seconds. A value that is
// not a positive number counts as absent (the zero time).
func parseReset(s string) time.Time {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n <= 0 {
		return time.Time{}
	}
	return time.Unix(n, 0).UTC()
}

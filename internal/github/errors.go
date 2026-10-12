package github

import "errors"

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
// the HTTP status and a cleaned API message: never from headers, request
// dumps or the token.
type Error struct {
	Kind    ErrorKind
	Status  int
	Message string
	err     error
}

func (e *Error) Error() string { return "GitHub request failed: " + e.Message }

func (e *Error) Unwrap() error { return e.err }

// Hint says how to fix the failure.
func (e *Error) Hint() string { return "" }

var _ interface{ Hint() string } = (*Error)(nil)

func classify(err error) error {
	if err == nil {
		return nil
	}
	return errors.New("stub")
}

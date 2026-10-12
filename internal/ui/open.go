package ui

import (
	"errors"
	"net/url"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

// Opener opens a URL in the user's browser. cmd/prv supplies the go-gh browser;
// ui declares the interface so that it never imports that package.
type Opener interface{ Open(url string) error }

// openedMsg is the result of one open attempt.
type openedMsg struct{ err error }

// safeURL returns the URL to open and true, or "" and false. Only https URLs
// whose host is exactly github.com (any letter case) pass: no port, no user
// info, no space or control character. A prefix check on the raw string would
// accept github.com.evil.com and github.com@evil.com, and go-gh's Browse allows
// more schemes than https, so this allow-list is the gate.
func safeURL(raw string) (string, bool) {
	if strings.IndexFunc(raw, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || !strings.EqualFold(u.Hostname(), "github.com") {
		return "", false
	}
	return u.String(), true
}

// openCmd opens u in the browser. The closure captures only o and u, never the
// Model. A panic inside the opener becomes an error with a fixed message: the
// panic value is never shown.
func openCmd(o Opener, u string) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if recover() != nil {
				msg = openedMsg{err: errors.New("could not open the browser")}
			}
		}()
		return openedMsg{err: o.Open(u)}
	}
}

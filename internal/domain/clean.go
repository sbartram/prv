package domain

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// Clean makes a remote string safe to print in a terminal. It strips ANSI
// escape sequences, turns tabs into spaces, and drops C0/C1 control characters
// and the bidi override and isolate characters (U+202A..U+202E, U+2066..U+2069).
// Everything else, including ZWJ emoji sequences, passes through.
func Clean(s string) string {
	s = ansi.Strip(s)
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case unicode.IsControl(r),
			r >= 0x202A && r <= 0x202E,
			r >= 0x2066 && r <= 0x2069:
			return -1
		}
		return r
	}, s)
}

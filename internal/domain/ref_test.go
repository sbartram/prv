package domain

import (
	"errors"
	"testing"
)

func TestParseRef(t *testing.T) {
	ok := []struct {
		name, in string
		want     Ref
	}{
		{"plain url", "https://github.com/octo/tools/pull/42", Ref{"octo/tools", 42}},
		{"files tab and newline", "https://github.com/octo/tools/pull/42/files\n", Ref{"octo/tools", 42}},
		{"checks", "https://github.com/octo/tools/pull/42/checks", Ref{"octo/tools", 42}},
		{"commits", "https://github.com/octo/tools/pull/42/commits", Ref{"octo/tools", 42}},
		{"commit sha", "https://github.com/octo/tools/pull/42/commits/abc1234", Ref{"octo/tools", 42}},
		{"trailing slash", "https://github.com/octo/tools/pull/42/", Ref{"octo/tools", 42}},
		{"fragment", "https://github.com/octo/tools/pull/42#issuecomment-1", Ref{"octo/tools", 42}},
		{"query", "https://github.com/octo/tools/pull/42?diff=split", Ref{"octo/tools", 42}},
		{"no scheme", "github.com/octo/tools/pull/42", Ref{"octo/tools", 42}},
		{"www", "http://www.github.com/octo/tools/pull/42", Ref{"octo/tools", 42}},
		{"upper-case host", "HTTPS://GitHub.com/octo/tools/pull/42", Ref{"octo/tools", 42}},
		{"letter case kept", "https://github.com/ACME/Api/pull/7", Ref{"ACME/Api", 7}},
		{"dots and underscores", "https://github.com/o/my_repo.js/pull/1", Ref{"o/my_repo.js", 1}},
		{"short form", "octo/tools#42", Ref{"octo/tools", 42}},
		{"short form with space around", "  octo/tools#42\n", Ref{"octo/tools", 42}},
		{"largest number", "o/r#2147483647", Ref{"o/r", 2147483647}},
	}
	for _, c := range ok {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseRef(c.in)
			if err != nil {
				t.Fatalf("ParseRef(%q) error = %v", c.in, err)
			}
			if got != c.want {
				t.Errorf("ParseRef(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}

	bad := []struct{ name, in, reason string }{
		{"other host", "https://ghe.corp.com/o/r/pull/1", ReasonOtherHost},
		{"other host no scheme", "ghe.corp.com/o/r/pull/1", ReasonOtherHost},
		{"issue url", "https://github.com/o/r/issues/3", ReasonIssue},
		{"issue url fragment", "github.com/o/r/issues/3#issuecomment-9", ReasonIssue},
		{"zero", "o/r#0", ReasonNotRef},
		{"too big", "o/r#2147483648", ReasonNotRef},
		{"eleven digits", "o/r#99999999999", ReasonNotRef},
		{"negative", "o/r#-1", ReasonNotRef},
		{"no number", "o/r", ReasonNotRef},
		{"number only", "#12", ReasonNotRef},
		{"two refs", "o/r#1 o/r#2", ReasonNotRef},
		{"empty", "", ReasonNotRef},
		{"words", "hello", ReasonNotRef},
		{"dot repo", "o/.#1", ReasonNotRef},
		{"dotdot repo", "o/..#1", ReasonNotRef},
		{"escape", "o/r#1\x1b[31m", ReasonNotRef},
		{"repo url", "https://github.com/o/r", ReasonNotRef},
		{"unknown suffix", "https://github.com/o/r/pull/1/foo", ReasonNotRef},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseRef(c.in)
			var re *RefError
			if !errors.As(err, &re) {
				t.Fatalf("ParseRef(%q) error = %v, want a *RefError", c.in, err)
			}
			if re.Reason != c.reason {
				t.Errorf("ParseRef(%q) reason = %q, want %q", c.in, re.Reason, c.reason)
			}
		})
	}
}

func TestRefKeyAndURL(t *testing.T) {
	if got, want := (Ref{"ACME/Api", 7}).Key(), (Ref{"acme/api", 7}); got != want {
		t.Errorf("Key = %v, want %v", got, want)
	}
	if got, want := (Ref{"octo/tools", 42}).URL(), "https://github.com/octo/tools/pull/42"; got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
}

func TestValidRef(t *testing.T) {
	cases := []struct {
		ref  Ref
		want bool
	}{
		{Ref{"octo/tools", 42}, true},
		{Ref{"o/r", MaxPRNumber}, true},
		{Ref{"octo/tools", 0}, false},
		{Ref{"octo/tools", MaxPRNumber + 1}, false},
		{Ref{"octo", 1}, false},
		{Ref{"a/b/c", 1}, false},
		{Ref{"-a/b", 1}, false},
		{Ref{"o/..", 1}, false},
		{Ref{"o/r r", 1}, false},
		{Ref{"", 1}, false},
	}
	for _, c := range cases {
		if got := ValidRef(c.ref); got != c.want {
			t.Errorf("ValidRef(%v) = %v, want %v", c.ref, got, c.want)
		}
	}
}

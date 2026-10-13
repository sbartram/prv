package watchlist

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sbartram/prv/internal/domain"
)

func TestPath(t *testing.T) {
	abs := t.TempDir()
	t.Setenv("XDG_DATA_HOME", abs)
	got, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(abs, "prv", "watchlist.json"); got != want {
		t.Errorf("Path = %q, want %q", got, want)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "relative/dir") // not absolute: ignored
	got, err = Path()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".local", "share", "prv", "watchlist.json"); got != want {
		t.Errorf("Path = %q, want %q", got, want)
	}
}

func TestAddCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
	s := New(path)
	s.now = func() time.Time { return time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC) }

	added, err := s.Add(domain.Ref{Repo: "octo/tools", Number: 42})
	if err != nil || !added {
		t.Fatalf("Add = %v, %v, want true, nil", added, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"version": 1`, `"repo": "octo/tools"`, `"number": 42`, `"added_at": "2026-10-12T09:00:00Z"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("file missing %s:\n%s", want, data)
		}
	}
	if runtime.GOOS != "windows" {
		fi, _ := os.Stat(path)
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("file mode = %v, want 0600", fi.Mode().Perm())
		}
		di, _ := os.Stat(filepath.Dir(path))
		if di.Mode().Perm() != 0o700 {
			t.Errorf("dir mode = %v, want 0700", di.Mode().Perm())
		}
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("directory holds %d files, want only watchlist.json (no temp file left)", len(entries))
	}

	refs, warns, err := s.Load()
	if err != nil || len(warns) != 0 || len(refs) != 1 || refs[0] != (domain.Ref{Repo: "octo/tools", Number: 42}) {
		t.Errorf("Load = %v, %v, %v", refs, warns, err)
	}
}

func TestAddDuplicateIsNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watchlist.json")
	s := New(path)
	if _, err := s.Add(domain.Ref{Repo: "octo/tools", Number: 42}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)

	added, err := s.Add(domain.Ref{Repo: "OCTO/Tools", Number: 42})
	if err != nil || added {
		t.Fatalf("Add of a duplicate = %v, %v, want false, nil", added, err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Errorf("a duplicate Add rewrote the file:\n%s\n---\n%s", before, after)
	}
}

func TestLoadKeepsFileOrder(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "watchlist.json"))
	want := []domain.Ref{{Repo: "b/b", Number: 2}, {Repo: "a/a", Number: 9}, {Repo: "a/a", Number: 1}}
	for _, r := range want {
		if _, err := s.Add(r); err != nil {
			t.Fatal(err)
		}
	}
	got, _, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("Load = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Load[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestLoadMissingFile(t *testing.T) {
	refs, warns, err := New(filepath.Join(t.TempDir(), "nope", "watchlist.json")).Load()
	if err != nil || len(refs) != 0 || len(warns) != 0 {
		t.Errorf("Load of a missing file = %v, %v, %v, want empty and no error", refs, warns, err)
	}
}

func TestUnreadableFileIsNeverOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watchlist.json")
	const junk = "this is not json"
	if err := os.WriteFile(path, []byte(junk), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(path)
	if _, err := s.Add(domain.Ref{Repo: "o/r", Number: 1}); err == nil {
		t.Error("Add on an unreadable file returned no error")
	}
	if _, err := s.Remove(domain.Ref{Repo: "o/r", Number: 1}); err == nil {
		t.Error("Remove on an unreadable file returned no error")
	}
	if data, _ := os.ReadFile(path); string(data) != junk {
		t.Errorf("the unreadable file was changed: %q", data)
	}
}

func TestRemove(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "watchlist.json"))
	for _, r := range []domain.Ref{{Repo: "a/a", Number: 1}, {Repo: "b/b", Number: 2}} {
		if _, err := s.Add(r); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := s.Remove(domain.Ref{Repo: "A/A", Number: 1})
	if err != nil || !removed {
		t.Fatalf("Remove = %v, %v, want true, nil", removed, err)
	}
	removed, err = s.Remove(domain.Ref{Repo: "A/A", Number: 1})
	if err != nil || removed {
		t.Fatalf("second Remove = %v, %v, want false, nil", removed, err)
	}
	got, _, _ := s.Load()
	if len(got) != 1 || got[0] != (domain.Ref{Repo: "b/b", Number: 2}) {
		t.Errorf("Load after Remove = %v", got)
	}
}

func TestLoadSkipsInvalidEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watchlist.json")
	body := `{"version":1,"prs":[{"repo":"a/a","number":1},{"repo":"not a repo","number":2},{"repo":"A/A","number":1},{"repo":"c/c","number":0}]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(path)
	refs, warns, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0] != (domain.Ref{Repo: "a/a", Number: 1}) {
		t.Errorf("Load refs = %v, want only a/a#1", refs)
	}
	if len(warns) != 1 || warns[0] != "watchlist.json: 2 entries skipped (not a PR reference)" {
		t.Errorf("Load warnings = %v", warns)
	}

	// An Add keeps the hand-edited entries it does not understand.
	if _, err := s.Add(domain.Ref{Repo: "d/d", Number: 4}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "not a repo") {
		t.Errorf("Add deleted a hand-edited entry:\n%s", data)
	}
}

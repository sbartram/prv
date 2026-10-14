package watchlist

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 2 || names[0] != "watchlist.json" || names[1] != "watchlist.json.lock" {
		t.Errorf("directory holds %v, want only watchlist.json and watchlist.json.lock (no temp file left)", names)
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

// wlStore returns a Store on a fresh path in a temp directory.
func wlStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
	return New(path), path
}

// wlWrite writes body to path, creating the directory.
func wlWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// wlRead returns the file bytes.
func wlRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// wlBackups lists the watchlist.json.corrupt-* files next to path.
func wlBackups(t *testing.T, path string) []string {
	t.Helper()
	got, err := filepath.Glob(path + ".corrupt-*")
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// wlEntries builds a version 1 file body with n valid entries r/r<i>#1.
func wlEntries(n int) string {
	var b strings.Builder
	b.WriteString(`{"version":1,"prs":[`)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"repo":"o/r%d","number":1,"added_at":"2026-10-12T09:00:00Z"}`, i)
	}
	b.WriteString(`]}`)
	return b.String()
}

func TestLoadMissing(t *testing.T) {
	s, path := wlStore(t)
	refs, warns, err := s.Load()
	if err != nil || len(refs) != 0 || len(warns) != 0 {
		t.Errorf("Load = %v, %v, %v, want nil, nil, nil", refs, warns, err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Errorf("Load of a missing file created %s", filepath.Dir(path))
	}
}

func TestRoundTrip(t *testing.T) {
	s, path := wlStore(t)
	a := domain.Ref{Repo: "acme/api", Number: 1}
	b := domain.Ref{Repo: "octo/tools", Number: 42}
	for _, r := range []domain.Ref{a, b} {
		if added, err := s.Add(r); err != nil || !added {
			t.Fatalf("Add(%v) = %v, %v", r, added, err)
		}
	}
	got, warns, err := s.Load()
	if err != nil || len(warns) != 0 || len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("Load = %v, %v, %v, want [%v %v]", got, warns, err, a, b)
	}

	// Privacy: the file holds refs and the time they were added, nothing else.
	var top map[string]any
	if err := json.Unmarshal([]byte(wlRead(t, path)), &top); err != nil {
		t.Fatal(err)
	}
	if len(top) != 2 || top["version"] != float64(1) || top["prs"] == nil {
		t.Errorf("top-level keys = %v, want exactly version (1) and prs", top)
	}
	for _, e := range top["prs"].([]any) {
		m := e.(map[string]any)
		if len(m) != 3 || m["repo"] == nil || m["number"] == nil || m["added_at"] == nil {
			t.Errorf("entry keys = %v, want exactly repo, number and added_at", m)
		}
	}
}

func TestAddDuplicateCase(t *testing.T) {
	s, path := wlStore(t)
	if _, err := s.Add(domain.Ref{Repo: "acme/api", Number: 1}); err != nil {
		t.Fatal(err)
	}
	before := wlRead(t, path)
	added, err := s.Add(domain.Ref{Repo: "ACME/API", Number: 1})
	if err != nil || added {
		t.Fatalf("Add = %v, %v, want false, nil", added, err)
	}
	if after := wlRead(t, path); after != before {
		t.Errorf("a duplicate Add changed the file:\n%s\n---\n%s", before, after)
	}
}

func TestRemoveByKey(t *testing.T) {
	s, _ := wlStore(t)
	for _, r := range []domain.Ref{{Repo: "acme/api", Number: 1}, {Repo: "octo/tools", Number: 42}} {
		if _, err := s.Add(r); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := s.Remove(domain.Ref{Repo: "Acme/Api", Number: 1})
	if err != nil || !removed {
		t.Fatalf("Remove = %v, %v, want true, nil", removed, err)
	}
	got, _, _ := s.Load()
	if len(got) != 1 || got[0] != (domain.Ref{Repo: "octo/tools", Number: 42}) {
		t.Errorf("Load after Remove = %v", got)
	}
}

func TestRemoveMissingNoWrite(t *testing.T) {
	s, path := wlStore(t)
	r := domain.Ref{Repo: "acme/api", Number: 1}
	if _, err := s.Add(r); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Remove(r); err != nil {
		t.Fatal(err)
	}
	// Put the file in the past so a rewrite would show in the modification time.
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	before := wlRead(t, path)
	for i := 0; i < 2; i++ { // removed already, for example by another pane
		removed, err := s.Remove(r)
		if err != nil || removed {
			t.Fatalf("Remove #%d = %v, %v, want false, nil", i+1, removed, err)
		}
	}
	if wlRead(t, path) != before {
		t.Error("Remove of a missing ref changed the file")
	}
	if fi, _ := os.Stat(path); !fi.ModTime().Equal(old) {
		t.Errorf("Remove of a missing ref rewrote the file (mtime %v)", fi.ModTime())
	}
}

var corruptName = regexp.MustCompile(`\.corrupt-[0-9a-f]{8}$`)

// wlCheckBackup asserts that bad was copied byte for byte to exactly one
// watchlist.json.corrupt-HASH8 file and that Load reports ErrCorrupt.
func wlCheckBackup(t *testing.T, s *Store, path, bad string) {
	t.Helper()
	for i := 0; i < 2; i++ { // the second Load must not copy again
		_, _, err := s.Load()
		if !errors.Is(err, ErrCorrupt) {
			t.Fatalf("Load #%d error = %v, want one wrapping ErrCorrupt", i+1, err)
		}
		var ce *CorruptError
		if !errors.As(err, &ce) || ce.Backup == "" {
			t.Fatalf("Load error = %#v, want *CorruptError naming the backup", err)
		}
	}
	backups := wlBackups(t, path)
	if len(backups) != 1 {
		t.Fatalf("backups = %v, want exactly one", backups)
	}
	if !corruptName.MatchString(backups[0]) {
		t.Errorf("backup name %q, want watchlist.json.corrupt- plus 8 hex digits", backups[0])
	}
	if got := wlRead(t, backups[0]); got != bad {
		t.Errorf("backup differs from the bad file (len %d, want %d)", len(got), len(bad))
	}
	if got := wlRead(t, path); got != bad {
		t.Error("Load changed the bad file")
	}
}

func TestCorruptBackup(t *testing.T) {
	s, path := wlStore(t)
	const bad = "{not json"
	wlWrite(t, path, bad)
	wlCheckBackup(t, s, path, bad)
}

func TestUnknownVersion(t *testing.T) {
	s, path := wlStore(t)
	const bad = `{"version": 2, "prs": []}`
	wlWrite(t, path, bad)
	wlCheckBackup(t, s, path, bad)
}

func TestTooLarge(t *testing.T) {
	s, path := wlStore(t)
	bad := `{"version":1,"prs":[],"pad":"` + strings.Repeat("a", 1<<20) + `"}` // over 1 MiB
	wlWrite(t, path, bad)
	wlCheckBackup(t, s, path, bad)
}

func TestBadEntriesKept(t *testing.T) {
	s, path := wlStore(t)
	body := `{"version":1,"prs":[` +
		`{"repo":"a/a","number":1},{"repo":"evil/../x","number":1},{"repo":"o/r","number":0},` +
		`{"repo":"a b/c","number":1},{"repo":"b/b","number":2}]}`
	wlWrite(t, path, body)
	refs, warns, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 || refs[0] != (domain.Ref{Repo: "a/a", Number: 1}) || refs[1] != (domain.Ref{Repo: "b/b", Number: 2}) {
		t.Errorf("Load refs = %v, want a/a#1 and b/b#2", refs)
	}
	if len(warns) != 1 || warns[0] != "watchlist.json: 3 entries skipped (not a PR reference)" {
		t.Errorf("Load warnings = %v", warns)
	}
	if _, err := s.Add(domain.Ref{Repo: "c/c", Number: 3}); err != nil {
		t.Fatal(err)
	}
	got := wlRead(t, path)
	for _, want := range []string{"evil/../x", `"o/r"`, "a b/c"} {
		if !strings.Contains(got, want) {
			t.Errorf("Add deleted the hand-edited entry %s:\n%s", want, got)
		}
	}

	one := `{"version":1,"prs":[{"repo":"a/a","number":1},{"repo":"bad","number":1}]}`
	wlWrite(t, path, one)
	if _, warns, _ := s.Load(); len(warns) != 1 || warns[0] != "watchlist.json: 1 entry skipped (not a PR reference)" {
		t.Errorf("singular warning = %v", warns)
	}
}

func TestAddOverCorrupt(t *testing.T) {
	s, path := wlStore(t)
	const bad = "{not json"
	wlWrite(t, path, bad)
	added, err := s.Add(domain.Ref{Repo: "octo/tools", Number: 42})
	if err != nil || !added {
		t.Fatalf("Add = %v, %v, want true, nil", added, err)
	}
	backups := wlBackups(t, path)
	if len(backups) != 1 || wlRead(t, backups[0]) != bad {
		t.Fatalf("backups = %v, want one holding the old bytes", backups)
	}
	refs, warns, err := s.Load()
	if err != nil || len(warns) != 0 || len(refs) != 1 || refs[0] != (domain.Ref{Repo: "octo/tools", Number: 42}) {
		t.Errorf("Load after Add = %v, %v, %v, want only the new entry", refs, warns, err)
	}
	if !strings.Contains(wlRead(t, path), `"version": 1`) {
		t.Errorf("the new file is not version 1:\n%s", wlRead(t, path))
	}
}

func TestUnreadableNotOverwritten(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("file modes do not stop this user")
	}
	s, path := wlStore(t)
	body := wlEntries(1)
	wlWrite(t, path, body)
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o600) })
	if _, err := s.Add(domain.Ref{Repo: "new/one", Number: 1}); err == nil {
		t.Error("Add over an unreadable file returned no error")
	}
	if _, err := s.Remove(domain.Ref{Repo: "o/r0", Number: 1}); err == nil {
		t.Error("Remove over an unreadable file returned no error")
	}
	os.Chmod(path, 0o600)
	if got := wlRead(t, path); got != body {
		t.Errorf("the unreadable file was changed: %q", got)
	}
	if len(wlBackups(t, path)) != 0 {
		t.Error("an unreadable file is not corrupt, but a backup was made")
	}
}

func TestMaxEntries(t *testing.T) {
	s, path := wlStore(t)
	wlWrite(t, path, wlEntries(MaxEntries+1))
	refs, warns, err := s.Load()
	if err != nil || len(refs) != MaxEntries {
		t.Fatalf("Load = %d refs, %v, want %d", len(refs), err, MaxEntries)
	}
	if refs[0].Repo != "o/r0" || refs[MaxEntries-1].Repo != "o/r499" {
		t.Errorf("Load did not keep the first %d: %v ... %v", MaxEntries, refs[0], refs[MaxEntries-1])
	}
	if len(warns) != 1 || warns[0] != "watchlist.json: only the first 500 PRs are watched" {
		t.Errorf("warnings = %v", warns)
	}

	wlWrite(t, path, wlEntries(MaxEntries))
	added, err := s.Add(domain.Ref{Repo: "new/one", Number: 1})
	if added || err == nil || err.Error() != "the watch list is full (500 PRs)" {
		t.Errorf("Add past the cap = %v, %v, want false and the full error", added, err)
	}
	if got := wlRead(t, path); got != wlEntries(MaxEntries) {
		t.Error("a refused Add changed the file")
	}
}

func TestPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file modes")
	}
	s, path := wlStore(t)
	if _, err := s.Add(domain.Ref{Repo: "a/a", Number: 1}); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %v, want 0600", fi.Mode().Perm())
	}
	if di, _ := os.Stat(filepath.Dir(path)); di.Mode().Perm() != 0o700 {
		t.Errorf("directory mode = %v, want 0700", di.Mode().Perm())
	}
	if fi, err := os.Stat(path + ".lock"); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("lock file = %v, %v, want mode 0600", fi, err)
	}
}

func TestNoTempLeftovers(t *testing.T) {
	s, path := wlStore(t)
	if _, err := s.Add(domain.Ref{Repo: "a/a", Number: 1}); err != nil {
		t.Fatal(err)
	}
	before := wlRead(t, path)
	s.rename = func(string, string) error { return errors.New("disk on fire") }
	if _, err := s.Add(domain.Ref{Repo: "b/b", Number: 2}); err == nil {
		t.Fatal("Add returned no error when the rename failed")
	}
	if _, err := s.Remove(domain.Ref{Repo: "a/a", Number: 1}); err == nil {
		t.Fatal("Remove returned no error when the rename failed")
	}
	if wlRead(t, path) != before {
		t.Error("a failed write changed the old file")
	}
	left, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".watchlist-*.tmp"))
	if len(left) != 0 {
		t.Errorf("temp files left behind: %v", left)
	}
}

func TestTwoStoresInterleave(t *testing.T) {
	_, path := wlStore(t)
	s1, s2 := New(path), New(path)
	a := domain.Ref{Repo: "o/a", Number: 1}
	b := domain.Ref{Repo: "o/b", Number: 2}
	c := domain.Ref{Repo: "o/c", Number: 3}
	steps := []func() (bool, error){
		func() (bool, error) { return s1.Add(a) },
		func() (bool, error) { return s2.Add(b) },
		func() (bool, error) { return s1.Remove(a) },
		func() (bool, error) { return s2.Add(c) },
	}
	for i, step := range steps {
		if ok, err := step(); err != nil || !ok {
			t.Fatalf("step %d = %v, %v, want true, nil", i+1, ok, err)
		}
	}
	got, _, err := s1.Load()
	if err != nil || len(got) != 2 || got[0] != b || got[1] != c {
		t.Errorf("Load = %v, %v, want [%v %v]", got, err, b, c)
	}
}

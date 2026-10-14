package watchlist

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/sbartram/prv/internal/domain"
)

// helperHold is how long each child holds the critical section, so that without
// the lock all 8 children read before any of them writes (D-07).
const helperHold = 50 * time.Millisecond

// TestHelperAdd is the child process of TestMultiProcessAdd. A plain go test
// skips it. As a child it waits at a start barrier, then adds one ref.
func TestHelperAdd(t *testing.T) {
	path := os.Getenv("PRV_WATCHLIST_HELPER_PATH")
	if path == "" {
		t.Skip("helper process only")
	}
	n, err := strconv.Atoi(os.Getenv("PRV_WATCHLIST_HELPER_N"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "bad PRV_WATCHLIST_HELPER_N:", err)
		os.Exit(1)
	}
	sync := os.Getenv("PRV_WATCHLIST_HELPER_SYNC")
	if err := os.WriteFile(filepath.Join(sync, "ready-"+strconv.Itoa(n)), nil, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "ready file:", err)
		os.Exit(1)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(sync, "go")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			fmt.Fprintln(os.Stderr, "barrier timeout")
			os.Exit(1)
		}
		time.Sleep(time.Millisecond)
	}
	s := New(path)
	s.afterRead = func() { time.Sleep(helperHold) }
	added, err := s.Add(domain.Ref{Repo: "o/r", Number: n})
	if err != nil || !added {
		fmt.Fprintf(os.Stderr, "Add = %v, %v\n", added, err)
		os.Exit(1)
	}
}

// TestMultiProcessAdd is the proof of WR-01 (D-07): 8 real processes add 8
// different refs to one file at the same moment and none is lost.
func TestMultiProcessAdd(t *testing.T) {
	const n = 8
	path := filepath.Join(t.TempDir(), "prv", "watchlist.json")
	syncDir := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	outs := make([][]byte, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 1; i <= n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.CommandContext(t.Context(), exe, "-test.run=^TestHelperAdd$")
			cmd.Env = append(os.Environ(),
				"PRV_WATCHLIST_HELPER_PATH="+path,
				"PRV_WATCHLIST_HELPER_N="+strconv.Itoa(i),
				"PRV_WATCHLIST_HELPER_SYNC="+syncDir,
			)
			outs[i-1], errs[i-1] = cmd.CombinedOutput()
		}()
	}

	// Release all children together once all of them are ready.
	deadline := time.Now().Add(30 * time.Second)
	for {
		ready, _ := filepath.Glob(filepath.Join(syncDir, "ready-*"))
		if len(ready) == n {
			break
		}
		if time.Now().After(deadline) {
			t.Errorf("only %d of %d children reached the start barrier", len(ready), n)
			break
		}
		time.Sleep(time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(syncDir, "go"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Errorf("child %d: %v\n%s", i+1, errs[i], outs[i])
		}
	}

	refs, warns, err := New(path).Load()
	if err != nil || len(warns) != 0 {
		t.Fatalf("Load = %v, %v, %v", refs, warns, err)
	}
	got := make([]int, 0, len(refs))
	for _, r := range refs {
		if r.Repo != "o/r" {
			got = append(got, -1)
			continue
		}
		got = append(got, r.Number)
	}
	slices.Sort(got)
	if !slices.Equal(got, []int{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Errorf("Load = %d refs %v, want 8 (o/r#1..o/r#8)", len(refs), refs)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".watchlist-*.tmp")); len(left) != 0 {
		t.Errorf("temp files left behind: %v", left)
	}
}

func TestLockBusy(t *testing.T) {
	s1, path := wlStore(t)
	s2 := New(path)
	s2.lockWait = 50 * time.Millisecond
	unlock, err := s1.lock()
	if err != nil {
		t.Fatal(err)
	}
	released := false
	defer func() {
		if !released {
			unlock()
		}
	}()

	_, statErr := os.Stat(path)
	if !os.IsNotExist(statErr) {
		t.Fatalf("watchlist.json exists before any write: %v", statErr)
	}
	start := time.Now()
	if _, err := s2.Add(domain.Ref{Repo: "a/a", Number: 1}); !errors.Is(err, ErrBusy) {
		t.Errorf("Add under a held lock = %v, want ErrBusy", err)
	}
	if d := time.Since(start); d < 50*time.Millisecond {
		t.Errorf("Add gave up after %v, want at least the 50ms wait", d)
	}
	start = time.Now()
	if _, err := s2.Remove(domain.Ref{Repo: "a/a", Number: 1}); !errors.Is(err, ErrBusy) {
		t.Errorf("Remove under a held lock = %v, want ErrBusy", err)
	}
	if d := time.Since(start); d < 50*time.Millisecond {
		t.Errorf("Remove gave up after %v, want at least the 50ms wait", d)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a busy writer created watchlist.json: %v", err)
	}

	unlock()
	released = true
	if added, err := s2.Add(domain.Ref{Repo: "a/a", Number: 1}); err != nil || !added {
		t.Errorf("Add after the unlock = %v, %v, want true, nil", added, err)
	}

	// A busy writer leaves an existing file as it is.
	s1b := New(path)
	unlock2, err := s1b.lock()
	if err != nil {
		t.Fatal(err)
	}
	defer unlock2()
	before := wlRead(t, path)
	if _, err := s2.Add(domain.Ref{Repo: "b/b", Number: 2}); !errors.Is(err, ErrBusy) {
		t.Errorf("Add under a held lock = %v, want ErrBusy", err)
	}
	if wlRead(t, path) != before {
		t.Error("a busy writer changed the file")
	}
}

func TestLoadWhileLocked(t *testing.T) {
	s1, path := wlStore(t)
	wlWrite(t, path, wlEntries(2))
	unlock, err := s1.lock()
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	s2 := New(path)
	start := time.Now()
	refs, warns, err := s2.Load()
	if err != nil || len(warns) != 0 || len(refs) != 2 {
		t.Errorf("Load while locked = %v, %v, %v, want 2 refs", refs, warns, err)
	}
	if d := time.Since(start); d > 500*time.Millisecond {
		t.Errorf("Load waited %v for the lock, want no wait", d)
	}
}

func TestLockFileKept(t *testing.T) {
	s, path := wlStore(t)
	if _, err := s.Add(domain.Ref{Repo: "a/a", Number: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Remove(domain.Ref{Repo: "a/a", Number: 1}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path + ".lock")
	if err != nil {
		t.Fatalf("lock file is gone after Add and Remove: %v", err)
	}
	if runtime.GOOS != "windows" {
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("lock file mode = %v, want 0600", fi.Mode().Perm())
		}
		if di, _ := os.Stat(filepath.Dir(path)); di.Mode().Perm() != 0o700 {
			t.Errorf("directory mode = %v, want 0700", di.Mode().Perm())
		}
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"watchlist.json", "watchlist.json.lock"}) {
		t.Errorf("directory holds %v, want watchlist.json and watchlist.json.lock", names)
	}
}

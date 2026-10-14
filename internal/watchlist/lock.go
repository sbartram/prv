package watchlist

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

// ErrBusy means another prv held the watch list lock for longer than the wait
// (D-09). The file is unchanged.
var ErrBusy = errors.New("watch list is busy (another prv is writing it)")

// defaultLockWait is how long a writer waits for the lock. A real hold lasts
// milliseconds (one fsync).
const defaultLockWait = 3 * time.Second

// lockPoll is the pause between lock attempts.
const lockPoll = 10 * time.Millisecond

// lock takes an exclusive advisory flock on the sibling file watchlist.json.lock
// and returns the function that drops it. flock has no timeout, so it is polled
// with LOCK_NB until s.lockWait has passed (D-09), then it returns ErrBusy. The
// kernel drops the lock when the process dies, so there is no stale lock to clean
// up, and the lock file is never deleted (D-11): deleting it races with another
// locker. Only the writers (AddRef/Add and RemoveRef/Remove) call lock; reads
// never do (D-10). macOS and Linux only (D-08).
func (s *Store) lock() (unlock func(), err error) {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating %s: %w", dir, err)
	}
	f, err := os.OpenFile(s.path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}
	deadline := time.Now().Add(s.lockWait)
	for {
		err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			// Closing the only descriptor drops the lock. The closure keeps f
			// reachable until then.
			return func() { f.Close() }, nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EINTR) {
			f.Close()
			return nil, fmt.Errorf("locking watch list: %w", err)
		}
		if time.Now().After(deadline) {
			f.Close()
			return nil, ErrBusy
		}
		time.Sleep(lockPoll)
	}
}

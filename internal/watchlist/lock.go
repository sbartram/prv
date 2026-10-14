package watchlist

import (
	"errors"
	"time"
)

// ErrBusy means another prv held the watch list lock for longer than the wait
// (D-09).
var ErrBusy = errors.New("watch list is busy (another prv is writing it)")

// defaultLockWait is how long a writer waits for the lock. A real hold lasts
// milliseconds.
const defaultLockWait = 3 * time.Second

// lockPoll is the pause between lock attempts.
const lockPoll = 10 * time.Millisecond

// lock is a RED-phase stub: it takes no lock yet.
func (s *Store) lock() (unlock func(), err error) { return func() {}, nil }

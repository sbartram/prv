// Package watchlist stores the PRs the user added by hand as versioned JSON. It
// holds PR refs and the time each was added, never PR data or the token. Only
// Add and Remove write, and they write atomically (temp file, then rename). A
// read never changes the file. A file prv cannot use (not valid JSON, another
// version, over 1 MiB) is first copied byte for byte to watchlist.json.corrupt-
// plus 8 hex digits of its SHA-256, once; only then may an Add start a fresh
// file. A file prv cannot read at all is never written over. Entries that are
// not PR references stay in the file and are skipped when loading. At most
// MaxEntries PRs are watched. It prints nothing.
package watchlist

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sbartram/prv/internal/domain"
)

// Version is the file format version.
const Version = 1

// ErrCorrupt is wrapped by the error Load returns for a file prv cannot use.
var ErrCorrupt = errors.New("the watch list file is not valid")

// CorruptError says what is wrong with the file and where the copy is.
type CorruptError struct {
	Backup  string
	Problem string
	// Version is non-nil exactly when the file is a version mismatch.
	Version *VersionError
}

func (e *CorruptError) Error() string {
	return "watchlist.json " + e.Problem + " (a copy is in " + filepath.Base(e.Backup) + ")"
}

func (e *CorruptError) Unwrap() error { return ErrCorrupt }

// VersionError is a RED-phase stub.
type VersionError struct {
	Found   int
	Missing bool
}

func (e *VersionError) Error() string { return "stub" }

// ErrFull is a RED-phase stub.
var ErrFull = errors.New("stub")

// Result is a RED-phase stub.
type Result struct {
	Changed     bool
	StartedOver string
}

// AddRef is a RED-phase stub.
func (s *Store) AddRef(ref domain.Ref) (Result, error) {
	added, err := s.Add(ref)
	return Result{Changed: added}, err
}

// RemoveRef is a RED-phase stub.
func (s *Store) RemoveRef(ref domain.Ref) (Result, error) {
	removed, err := s.Remove(ref)
	return Result{Changed: removed}, err
}

// MaxEntries is the most PRs that are watched.
const MaxEntries = 500

// maxFileSize caps what is read: a watch list is a few hundred bytes.
const maxFileSize = 1 << 20

// Path returns the watch list path: $XDG_DATA_HOME/prv/watchlist.json when
// XDG_DATA_HOME is set and absolute, else ~/.local/share/prv/watchlist.json. It
// does not use the standard per-user config helper, which returns
// ~/Library/Application Support on macOS.
func Path() (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" && filepath.IsAbs(dir) {
		return filepath.Join(dir, "prv", "watchlist.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "prv", "watchlist.json"), nil
}

type fileShape struct {
	Version int     `json:"version"`
	PRs     []entry `json:"prs"`
}

type entry struct {
	Repo    string    `json:"repo"`
	Number  int       `json:"number"`
	AddedAt time.Time `json:"added_at"`
}

func (e entry) ref() domain.Ref { return domain.Ref{Repo: e.Repo, Number: e.Number} }

// Store is the watch list file. It keeps no state in memory: every call reads
// the file, so two prv instances and hand edits stay in sync (D-16).
type Store struct {
	path   string
	now    func() time.Time
	rename func(oldpath, newpath string) error
	// lockWait is how long a writer waits for the lock; tests shorten it.
	lockWait time.Duration
	// link publishes a finished backup (test seam, os.Link in New).
	link func(oldname, newname string) error
	// afterRead is a test hook. When non-nil, Add calls it inside the lock right
	// after readForWrite returns, so a test can hold the critical section open.
	// New leaves it nil.
	afterRead func()
}

// New returns a Store for path. It touches no file.
func New(path string) *Store {
	return &Store{path: path, now: time.Now, rename: os.Rename, lockWait: defaultLockWait, link: os.Link}
}

// Path returns the file path of the store.
func (s *Store) Path() string { return s.path }

// read returns the file contents. A missing file is an empty list. A file that
// is too large, is not valid JSON or is not version 1 is copied to a backup (once)
// and reported as a *CorruptError. A file that cannot be read, or whose backup
// cannot be made, is a plain error. In every error case the file is untouched, so
// that no caller writes over it by accident.
func (s *Store) read() (fileShape, error) {
	f, err := os.Open(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return fileShape{Version: Version}, nil
	}
	if err != nil {
		return fileShape{}, fmt.Errorf("reading %s: %w", s.path, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxFileSize+1))
	if err != nil {
		return fileShape{}, fmt.Errorf("reading %s: %w", s.path, err)
	}
	problem := ""
	var shape fileShape
	switch {
	case len(data) > maxFileSize:
		problem = "is larger than 1 MiB"
	case json.Unmarshal(data, &shape) != nil || shape.Version != Version:
		problem = "is not a valid watch list"
	default:
		return shape, nil
	}
	name, err := s.backup(data)
	if err != nil {
		return fileShape{}, fmt.Errorf("could not back up watchlist.json: %w", err)
	}
	return fileShape{}, &CorruptError{Backup: name, Problem: problem}
}

// backup copies the file to watchlist.json.corrupt-HASH8, where HASH8 is the
// first 8 hex digits of the SHA-256 of what was read. The copy is made once: an
// existing copy of the same bytes is left alone. The whole file is copied, also
// when it is larger than what read keeps in memory.
func (s *Store) backup(data []byte) (name string, err error) {
	sum := sha256.Sum256(data)
	name = s.path + ".corrupt-" + hex.EncodeToString(sum[:])[:8]
	dst, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return name, nil
	}
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			dst.Close()
			os.Remove(name) // never leave a partial copy that would count as done
		}
	}()
	src, err := os.Open(s.path)
	if err != nil {
		return "", err
	}
	defer src.Close()
	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}
	if err = dst.Sync(); err != nil {
		return "", err
	}
	if err = dst.Close(); err != nil {
		return "", err
	}
	return name, nil
}

// validRefs returns the entries that are PR references, in file order, without
// repeats (same Key), and how many entries were not PR references.
func validRefs(shape fileShape) (refs []domain.Ref, skipped int) {
	seen := map[domain.Ref]bool{}
	for _, e := range shape.PRs {
		r := e.ref()
		if !domain.ValidRef(r) {
			skipped++
			continue
		}
		if seen[r.Key()] {
			continue
		}
		seen[r.Key()] = true
		refs = append(refs, r)
	}
	return refs, skipped
}

// Load returns the saved refs in file order, at most MaxEntries. An entry that
// is not a valid PR ref, or repeats an earlier entry (same Key), is skipped;
// skipped invalid entries come back as one warning. Load never writes the file.
func (s *Store) Load() ([]domain.Ref, []string, error) {
	shape, err := s.read()
	if err != nil {
		return nil, nil, err
	}
	refs, skipped := validRefs(shape)
	var warns []string
	switch {
	case skipped == 1:
		warns = append(warns, "watchlist.json: 1 entry skipped (not a PR reference)")
	case skipped > 1:
		warns = append(warns, "watchlist.json: "+strconv.Itoa(skipped)+" entries skipped (not a PR reference)")
	}
	if len(refs) > MaxEntries {
		refs = refs[:MaxEntries]
		warns = append(warns, "watchlist.json: only the first "+strconv.Itoa(MaxEntries)+" PRs are watched")
	}
	return refs, warns, nil
}

// readForWrite reads the file for Add and Remove. A file that was copied to a
// backup starts over as an empty list (the backup holds the old bytes); any other
// error means nothing may be written.
func (s *Store) readForWrite() (fileShape, error) {
	shape, err := s.read()
	if errors.Is(err, ErrCorrupt) {
		return fileShape{Version: Version}, nil
	}
	return shape, err
}

// Add saves ref. It reports false, and writes nothing, when an entry with the
// same Key is already there. A file that cannot be read is an error and is left
// as it is. Entries that fail validation stay in the file: a hand edit is never
// deleted by an Add. It refuses to grow the list past MaxEntries.
func (s *Store) Add(ref domain.Ref) (bool, error) {
	if !domain.ValidRef(ref) {
		return false, errors.New("not a PR reference")
	}
	unlock, err := s.lock()
	if err != nil {
		return false, err
	}
	defer unlock()
	shape, err := s.readForWrite()
	if err != nil {
		return false, err
	}
	if s.afterRead != nil {
		s.afterRead()
	}
	for _, e := range shape.PRs {
		if e.ref().Key() == ref.Key() {
			return false, nil
		}
	}
	if refs, _ := validRefs(shape); len(refs) >= MaxEntries {
		return false, fmt.Errorf("the watch list is full (%d PRs)", MaxEntries)
	}
	shape.PRs = append(shape.PRs, entry{Repo: ref.Repo, Number: ref.Number, AddedAt: s.now().UTC()})
	if err := s.write(shape); err != nil {
		return false, err
	}
	return true, nil
}

// Remove drops every entry with the same Key as ref. It reports false, and
// writes nothing, when there was none (for example another pane removed it).
func (s *Store) Remove(ref domain.Ref) (bool, error) {
	unlock, err := s.lock()
	if err != nil {
		return false, err
	}
	defer unlock()
	shape, err := s.readForWrite()
	if err != nil {
		return false, err
	}
	kept := make([]entry, 0, len(shape.PRs))
	for _, e := range shape.PRs {
		if e.ref().Key() != ref.Key() {
			kept = append(kept, e)
		}
	}
	if len(kept) == len(shape.PRs) {
		return false, nil
	}
	shape.PRs = kept
	if err := s.write(shape); err != nil {
		return false, err
	}
	return true, nil
}

// write replaces the file: it writes a temp file in the same directory and
// renames it over the path, so a crash leaves the old file intact.
func (s *Store) write(f fileShape) (err error) {
	f.Version = Version
	if f.PRs == nil {
		f.PRs = []entry{}
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".watchlist-*.tmp") // mode 0600
	if err != nil {
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if _, err = tmp.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	if err = s.rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	return nil
}

// Package watchlist stores the PRs the user added by hand as versioned JSON. It
// holds PR refs and the time each was added, never PR data or the token. Only
// AddRef/Add and RemoveRef/Remove write. They hold an exclusive flock on
// watchlist.json.lock for the whole read-modify-write and write atomically (temp
// file, then rename). Reads never lock and never change the file. A file prv
// cannot parse (not valid JSON, not a JSON object, a version-1 object that does
// not decode, over 1 MiB) is first copied to watchlist.json.corrupt- plus 8 hex
// digits of its SHA-256, published only when complete; only then may an add
// start a fresh file, and the caller is told through Result.StartedOver. A JSON
// object whose version is not 1 (another version, version 0, or no version at
// all) is copied but never written over. A file prv cannot read at all is never
// written over. Entries that are not PR references stay in the file and are
// skipped when loading. At most MaxEntries PRs are watched. It prints nothing.
package watchlist

import (
	"bytes"
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
	"strings"
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
	// Version is non-nil exactly when the file is a version mismatch: nothing may
	// write over it. It is nil for a file that may start over (unparseable or
	// oversized). An int field would not do as the marker: version 0 is a
	// mismatch too.
	Version *VersionError
}

func (e *CorruptError) Error() string {
	return "watchlist.json " + e.Problem + " (a copy is in " + filepath.Base(e.Backup) + ")"
}

func (e *CorruptError) Unwrap() error { return ErrCorrupt }

// VersionError says the file is a JSON object whose version is not the one this
// prv writes. Found is the integer version it declares (0 is a real value).
// Missing is true when the version key is absent, null or not an integer. It does
// not wrap ErrCorrupt: nothing may start over from it.
type VersionError struct {
	Found   int
	Missing bool
}

func (e *VersionError) Error() string {
	if e.Missing {
		return fmt.Sprintf("watchlist.json has no version number; this prv writes version %d", Version)
	}
	return fmt.Sprintf("watchlist.json is version %d; this prv writes version %d", e.Found, Version)
}

// ErrFull is returned by AddRef (and Add) when the list holds MaxEntries PRs.
var ErrFull = fmt.Errorf("the watch list is full (%d PRs)", MaxEntries)

// Result is what AddRef and RemoveRef report. Changed means the file was
// written. StartedOver is the base name of the backup (watchlist.json.corrupt-
// XXXXXXXX) when this call replaced an unparseable or oversized file, and is
// empty otherwise.
type Result struct {
	Changed     bool
	StartedOver string
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
// the file, so two prv instances and hand edits stay in sync (D-16). Writers
// serialise through watchlist.json.lock; readers never wait for it.
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

// classify decides whether data is a usable version-1 list. It returns the shape
// and an empty problem when it is. Otherwise problem says why, and ve is non-nil
// when the file is a JSON object whose version is not 1: the version is read
// before the entries are decoded, so a future version whose entries this prv
// cannot decode is still recognised.
func classify(data []byte) (shape fileShape, problem string, ve *VersionError) {
	if len(data) > maxFileSize {
		return fileShape{}, "is larger than 1 MiB", nil
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(data, &obj) != nil || obj == nil { // null decodes to a nil map
		return fileShape{}, "is not a valid watch list", nil
	}
	var v *int
	if raw, ok := obj["version"]; !ok || json.Unmarshal(raw, &v) != nil || v == nil {
		ve = &VersionError{Missing: true}
	} else if *v != Version {
		ve = &VersionError{Found: *v}
	}
	if ve != nil {
		return fileShape{}, strings.TrimPrefix(ve.Error(), "watchlist.json "), ve
	}
	if json.Unmarshal(data, &shape) != nil {
		return fileShape{}, "is not a valid watch list", nil
	}
	return shape, "", nil
}

// read returns the file contents. A missing file is an empty list. A file that
// is too large, is not a JSON object, does not decode, or is not version 1 is
// copied to a backup (once) and reported as a *CorruptError; its Version field
// is set for a version mismatch. A file that cannot be read, or whose backup
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
	shape, problem, ve := classify(data)
	if problem == "" {
		return shape, nil
	}
	// f sits right after data, because the limited reader never reads past its
	// limit: an oversized file continues through the same handle.
	name, err := s.backup(data, f)
	if err != nil {
		return fileShape{}, fmt.Errorf("could not back up watchlist.json: %w", err)
	}
	return fileShape{}, &CorruptError{Backup: name, Problem: problem, Version: ve}
}

// backup copies data, followed by the rest of the same open handle, to a temp
// file next to the watch list. It syncs and closes the temp file and only then
// publishes it as watchlist.json.corrupt-HASH8 with a no-clobber hard link, so a
// partial backup never has the final name. HASH8 is the first 8 hex digits of the
// SHA-256 of exactly the bytes in the backup. An existing backup of that name
// counts as done. Any other link failure is an error: nothing is left behind and
// the caller writes nothing. The path is never re-opened.
func (s *Store) backup(data []byte, rest io.Reader) (name string, err error) {
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".corrupt-*.tmp") // mode 0600
	if err != nil {
		return "", err
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name()) // after a successful link the final name keeps the inode
	}()
	h := sha256.New()
	if _, err = io.Copy(io.MultiWriter(tmp, h), io.MultiReader(bytes.NewReader(data), rest)); err != nil {
		return "", err
	}
	if err = tmp.Sync(); err != nil {
		return "", err
	}
	if err = tmp.Close(); err != nil {
		return "", err
	}
	name = s.path + ".corrupt-" + hex.EncodeToString(h.Sum(nil))[:8]
	if err = s.link(tmp.Name(), name); err != nil && !errors.Is(err, fs.ErrExist) {
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

// readForWrite reads the file for AddRef and RemoveRef. A version mismatch comes
// back as its *VersionError and nothing may be written. An unparseable or
// oversized file that was copied to a backup starts over as an empty list; the
// backup's full path is returned so the caller can report it. Any other error
// means nothing may be written.
func (s *Store) readForWrite() (shape fileShape, backup string, err error) {
	shape, err = s.read()
	var ce *CorruptError
	if errors.As(err, &ce) {
		if ce.Version != nil {
			return fileShape{}, "", ce.Version
		}
		return fileShape{Version: Version}, ce.Backup, nil
	}
	return shape, "", err
}

// startedOver returns the notice for Result.StartedOver.
func startedOver(backup string) string {
	if backup == "" {
		return ""
	}
	return filepath.Base(backup)
}

// AddRef saves ref under the lock. It reports Changed false, and writes nothing,
// when an entry with the same Key is already there. A file that cannot be read,
// or that is a version mismatch, is an error and is left as it is. Entries that
// fail validation stay in the file: a hand edit is never deleted by an add. It
// returns ErrFull instead of growing the list past MaxEntries, and ErrBusy when
// another prv holds the lock for longer than the wait.
func (s *Store) AddRef(ref domain.Ref) (Result, error) {
	if !domain.ValidRef(ref) {
		return Result{}, errors.New("not a PR reference")
	}
	unlock, err := s.lock()
	if err != nil {
		return Result{}, err
	}
	defer unlock()
	shape, backup, err := s.readForWrite()
	if err != nil {
		return Result{}, err
	}
	if s.afterRead != nil {
		s.afterRead()
	}
	for _, e := range shape.PRs {
		if e.ref().Key() == ref.Key() {
			return Result{}, nil
		}
	}
	if refs, _ := validRefs(shape); len(refs) >= MaxEntries {
		return Result{}, ErrFull
	}
	shape.PRs = append(shape.PRs, entry{Repo: ref.Repo, Number: ref.Number, AddedAt: s.now().UTC()})
	if err := s.write(shape); err != nil {
		return Result{}, err
	}
	return Result{Changed: true, StartedOver: startedOver(backup)}, nil
}

// RemoveRef drops every entry with the same Key as ref, under the lock. It
// reports Changed false, and writes nothing, when there was none (for example
// another pane removed it, or the file was unparseable and only a backup was
// made).
func (s *Store) RemoveRef(ref domain.Ref) (Result, error) {
	unlock, err := s.lock()
	if err != nil {
		return Result{}, err
	}
	defer unlock()
	shape, backup, err := s.readForWrite()
	if err != nil {
		return Result{}, err
	}
	kept := make([]entry, 0, len(shape.PRs))
	for _, e := range shape.PRs {
		if e.ref().Key() != ref.Key() {
			kept = append(kept, e)
		}
	}
	if len(kept) == len(shape.PRs) {
		return Result{}, nil
	}
	shape.PRs = kept
	if err := s.write(shape); err != nil {
		return Result{}, err
	}
	return Result{Changed: true, StartedOver: startedOver(backup)}, nil
}

// Add saves ref like AddRef and reports only whether the file changed.
func (s *Store) Add(ref domain.Ref) (bool, error) {
	r, err := s.AddRef(ref)
	return r.Changed, err
}

// Remove drops ref like RemoveRef and reports only whether the file changed.
func (s *Store) Remove(ref domain.Ref) (bool, error) {
	r, err := s.RemoveRef(ref)
	return r.Changed, err
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

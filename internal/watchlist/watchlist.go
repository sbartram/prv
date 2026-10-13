// Package watchlist stores the PRs the user added by hand as versioned JSON. It
// holds PR refs only, never PR data or the token. Only Add and Remove write,
// and they write atomically. It prints nothing.
package watchlist

import (
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
	path string
	now  func() time.Time
}

// New returns a Store for path. It touches no file.
func New(path string) *Store { return &Store{path: path, now: time.Now} }

// Path returns the file path of the store.
func (s *Store) Path() string { return s.path }

// read returns the file contents. A missing file is an empty list. A file that
// cannot be read, is too large or is not a version 1 watch list is an error, so
// that no caller writes over it.
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
	if len(data) > maxFileSize {
		return fileShape{}, errors.New("watchlist.json is larger than 1 MiB")
	}
	var shape fileShape
	if err := json.Unmarshal(data, &shape); err != nil || shape.Version != Version {
		return fileShape{}, errors.New("watchlist.json is not a valid watch list")
	}
	return shape, nil
}

// Load returns the saved refs in file order. An entry that is not a valid PR
// ref, or repeats an earlier entry (same Key), is skipped; skipped invalid
// entries come back as one warning.
func (s *Store) Load() ([]domain.Ref, []string, error) {
	shape, err := s.read()
	if err != nil {
		return nil, nil, err
	}
	var refs []domain.Ref
	seen := map[domain.Ref]bool{}
	skipped := 0
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
	var warns []string
	if skipped > 0 {
		warns = append(warns, "watchlist.json: "+strconv.Itoa(skipped)+" entries skipped (not a PR reference)")
	}
	return refs, warns, nil
}

// Add saves ref. It reports false, and writes nothing, when an entry with the
// same Key is already there. A file that cannot be read is an error and is left
// as it is. Entries that fail validation stay in the file: a hand edit is never
// deleted by an Add.
func (s *Store) Add(ref domain.Ref) (bool, error) {
	if !domain.ValidRef(ref) {
		return false, errors.New("not a PR reference")
	}
	shape, err := s.read()
	if err != nil {
		return false, err
	}
	for _, e := range shape.PRs {
		if e.ref().Key() == ref.Key() {
			return false, nil
		}
	}
	shape.PRs = append(shape.PRs, entry{Repo: ref.Repo, Number: ref.Number, AddedAt: s.now().UTC()})
	if err := s.write(shape); err != nil {
		return false, err
	}
	return true, nil
}

// Remove drops every entry with the same Key as ref. It reports false, and
// writes nothing, when there was none.
func (s *Store) Remove(ref domain.Ref) (bool, error) {
	shape, err := s.read()
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
	if err = os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	return nil
}

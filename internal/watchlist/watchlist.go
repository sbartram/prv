// Package watchlist stores the PRs the user added by hand. RED stub.
package watchlist

import (
	"errors"
	"time"

	"github.com/sbartram/prv/internal/domain"
)

// Store is the watch list file. RED stub.
type Store struct {
	path string
	now  func() time.Time
}

// New returns a Store for path. RED stub.
func New(path string) *Store { return &Store{path: path, now: time.Now} }

// Path returns the default watch list path. RED stub.
func Path() (string, error) { return "", errors.New("not implemented") }

// Load returns the saved refs. RED stub.
func (s *Store) Load() ([]domain.Ref, []string, error) {
	return nil, nil, errors.New("not implemented")
}

// Add saves ref. RED stub.
func (s *Store) Add(ref domain.Ref) (bool, error) { return false, errors.New("not implemented") }

// Remove drops ref. RED stub.
func (s *Store) Remove(ref domain.Ref) (bool, error) { return false, errors.New("not implemented") }

package ui

import "github.com/sbartram/prv/internal/domain"

// Watchlist saves and removes manually added PRs. RED stub.
type Watchlist interface {
	Add(ref domain.Ref) (bool, error)
	Remove(ref domain.Ref) (bool, error)
}

package ui

import (
	"time"

	"charm.land/bubbles/v2/viewport"

	"github.com/sbartram/prv/internal/domain"
)

// detailState is the detail view of one pull request.
type detailState struct {
	open     bool
	ref      domain.Ref
	pr       domain.PR
	data     *domain.Detail
	err      error
	at       time.Time
	expanded bool
	vp       viewport.Model
}

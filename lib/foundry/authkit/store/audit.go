package store

import (
	"context"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/domain"
)

// AuditStore handles audit event persistence.
type AuditStore interface {
	// Record stores an audit event.
	Record(ctx context.Context, evt domain.Event) error
}
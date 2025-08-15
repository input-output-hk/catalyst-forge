package store

import (
	"context"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
)

// AuditStore handles audit event persistence.
type AuditStore interface {
	// Record stores an audit event.
	Record(ctx context.Context, evt domain.Event) error
}
package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
)

// AuditStore handles audit event persistence.
type AuditStore interface {
	// Record stores an audit event.
	Record(ctx context.Context, evt domain.Event) error

	// List returns audit events matching optional filters, ordered by created_at desc.
	// actorID/userID may be nil to ignore; types may be empty to ignore; supports limit/offset pagination.
	List(ctx context.Context, actorID *uuid.UUID, userID *uuid.UUID, types []string, since *time.Time, until *time.Time, limit, offset int) ([]domain.Event, int64, error)
}

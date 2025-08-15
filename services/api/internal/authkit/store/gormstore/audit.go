package gormstore

import (
	"context"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"
	"gorm.io/gorm"
)

// AuditStore is a GORM-based implementation of store.AuditStore.
type AuditStore struct {
	db *gorm.DB
}

// NewAuditStore creates a new GORM-based audit store.
func NewAuditStore(db *gorm.DB) *AuditStore {
	return &AuditStore{db: db}
}

// dbFor returns the appropriate database handle for the given context.
func (s *AuditStore) dbFor(ctx context.Context) *gorm.DB {
	if tx := repodb.TxFromContext(ctx); tx != nil {
		return tx
	}
	return s.db
}

// Record stores an audit event.
func (s *AuditStore) Record(ctx context.Context, evt domain.Event) error {
	dbEvent := &AuditEvent{
		ID:        evt.ID,
		Type:      string(evt.Type),
		UserID:    evt.UserID,
		ActorID:   evt.ActorID,
		IPAddress: evt.IPAddress,
		UserAgent: evt.UserAgent,
		Metadata:  JSON(evt.Metadata),
		CreatedAt: evt.CreatedAt,
	}

	return s.dbFor(ctx).WithContext(ctx).Create(dbEvent).Error
}
package gormstore

import (
	"context"
	"time"

	repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
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

// List returns audit events with filters and pagination
func (s *AuditStore) List(ctx context.Context, actorID *uuid.UUID, userID *uuid.UUID, types []string, since *time.Time, until *time.Time, limit, offset int) ([]domain.Event, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	db := s.dbFor(ctx).WithContext(ctx).Model(&AuditEvent{})
	if actorID != nil {
		db = db.Where("actor_id = ?", *actorID)
	}
	if userID != nil {
		db = db.Where("user_id = ?", *userID)
	}
	if len(types) > 0 {
		db = db.Where("type IN ?", types)
	}
	if since != nil {
		db = db.Where("created_at >= ?", *since)
	}
	if until != nil {
		db = db.Where("created_at <= ?", *until)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []AuditEvent
	if err := db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Event, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Event{
			ID:        r.ID,
			Type:      domain.EventType(r.Type),
			UserID:    r.UserID,
			ActorID:   r.ActorID,
			IPAddress: r.IPAddress,
			UserAgent: r.UserAgent,
			Metadata:  map[string]interface{}(r.Metadata),
			CreatedAt: r.CreatedAt,
		})
	}
	return out, total, nil
}

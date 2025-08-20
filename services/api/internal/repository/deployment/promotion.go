package deployment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	model "github.com/input-output-hk/catalyst-forge/services/api/internal/models/deployment"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
)

var (
	ErrPromotionNotFound = errors.New("promotion not found")
	ErrPromotionExists   = errors.New("promotion already exists")
)

// PromotionRepository defines the interface for promotion operations
type PromotionRepository interface {
	Create(ctx context.Context, p *model.Promotion) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Promotion, error)
	List(ctx context.Context, filter PromotionListFilter) ([]model.Promotion, int64, error)
	Update(ctx context.Context, p *model.Promotion) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.PromotionStatus, approverID *string, approvedAt *time.Time, reason *string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PromotionListFilter contains parameters for listing promotions
type PromotionListFilter struct {
	ProjectID  *uuid.UUID
	EnvID      *uuid.UUID
	ReleaseID  *uuid.UUID
	Status     *model.PromotionStatus
	Since      *time.Time
	Until      *time.Time
	Pagination *base.Pagination
	Sort       *base.Sort
}

type promotionRepositoryImpl struct {
	db *gorm.DB
}

// NewPromotionRepository creates a new promotion repository instance
func NewPromotionRepository(db *gorm.DB) PromotionRepository {
	return &promotionRepositoryImpl{db: db}
}

// Create inserts a new promotion
func (r *promotionRepositoryImpl) Create(ctx context.Context, p *model.Promotion) error {
	db := base.GetDB(ctx, r.db)
	if err := db.Create(p).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrPromotionExists
		}
		return err
	}
	return nil
}

// GetByID retrieves a promotion by ID
func (r *promotionRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*model.Promotion, error) {
	db := base.GetDB(ctx, r.db)
	var p model.Promotion
	if err := db.Where("id = ?", id).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPromotionNotFound
		}
		return nil, err
	}
	return &p, nil
}

// List retrieves promotions with filters and pagination
func (r *promotionRepositoryImpl) List(ctx context.Context, filter PromotionListFilter) ([]model.Promotion, int64, error) {
	db := base.GetDB(ctx, r.db)

	query := db.Model(&model.Promotion{})

	if filter.ProjectID != nil {
		query = query.Where("project_id = ?", *filter.ProjectID)
	}
	if filter.EnvID != nil {
		query = query.Where("env_id = ?", *filter.EnvID)
	}
	if filter.ReleaseID != nil {
		query = query.Where("release_id = ?", *filter.ReleaseID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Since != nil {
		query = query.Where("created_at >= ?", *filter.Since)
	}
	if filter.Until != nil {
		query = query.Where("created_at <= ?", *filter.Until)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = base.ApplySort(query, filter.Sort, "created_at DESC")
	query = base.ApplyPagination(query, filter.Pagination)

	var items []model.Promotion
	if err := query.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Update updates a promotion record
func (r *promotionRepositoryImpl) Update(ctx context.Context, p *model.Promotion) error {
	db := base.GetDB(ctx, r.db)
	result := db.Model(p).Where("id = ?", p.ID).Updates(p)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPromotionNotFound
	}
	return nil
}

// UpdateStatus updates status and optional approval info
func (r *promotionRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status model.PromotionStatus, approverID *string, approvedAt *time.Time, reason *string) error {
	db := base.GetDB(ctx, r.db)
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if approverID != nil {
		updates["approver_id"] = *approverID
	} else {
		updates["approver_id"] = nil
	}
	if approvedAt != nil {
		updates["approved_at"] = *approvedAt
	} else {
		updates["approved_at"] = nil
	}
	if reason != nil {
		updates["reason"] = *reason
	}

	result := db.Model(&model.Promotion{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPromotionNotFound
	}
	return nil
}

// Delete deletes a promotion by ID
func (r *promotionRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	db := base.GetDB(ctx, r.db)
	result := db.Where("id = ?", id).Delete(&model.Promotion{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPromotionNotFound
	}
	return nil
}

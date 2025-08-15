package trace

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/enums"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/trace"
	base "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
)

var (
	ErrTraceNotFound = errors.New("trace not found")
)

// Repository defines the interface for trace operations
type Repository interface {
	Create(ctx context.Context, trace *trace.Trace) error
	GetByID(ctx context.Context, id uuid.UUID) (*trace.Trace, error)
	List(ctx context.Context, filter ListFilter) ([]trace.Trace, int64, error)
	Update(ctx context.Context, trace *trace.Trace) error
}

// ListFilter contains filter parameters for listing traces
type ListFilter struct {
	RepoID         *uuid.UUID
	Purpose        *enums.TracePurpose
	RetentionClass *enums.RetentionClass
	Branch         *string
	CreatedBy      *string
	Since          *time.Time
	Until          *time.Time
	Pagination     *base.Pagination
	Sort           *base.Sort
}

// repositoryImpl implements Repository interface
type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

// Create creates a new trace
func (r *repositoryImpl) Create(ctx context.Context, tr *trace.Trace) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(tr).Error; err != nil {
		return err
	}
	
	return nil
}

// GetByID retrieves a trace by ID
func (r *repositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*trace.Trace, error) {
	db := base.GetDB(ctx, r.db)
	
	var tr trace.Trace
	if err := db.Where("id = ?", id).First(&tr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTraceNotFound
		}
		return nil, err
	}
	
	return &tr, nil
}

// List retrieves traces with filters and pagination
func (r *repositoryImpl) List(ctx context.Context, filter ListFilter) ([]trace.Trace, int64, error) {
	db := base.GetDB(ctx, r.db)
	
	query := db.Model(&trace.Trace{})
	
	// Apply filters
	if filter.RepoID != nil {
		query = query.Where("repo_id = ?", *filter.RepoID)
	}
	if filter.Purpose != nil {
		query = query.Where("purpose = ?", *filter.Purpose)
	}
	if filter.RetentionClass != nil {
		query = query.Where("retention_class = ?", *filter.RetentionClass)
	}
	if filter.Branch != nil {
		query = query.Where("branch = ?", *filter.Branch)
	}
	if filter.CreatedBy != nil {
		query = query.Where("created_by = ?", *filter.CreatedBy)
	}
	if filter.Since != nil {
		query = query.Where("created_at >= ?", *filter.Since)
	}
	if filter.Until != nil {
		query = query.Where("created_at <= ?", *filter.Until)
	}
	
	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// Apply sorting
	query = base.ApplySort(query, filter.Sort, "created_at DESC")
	
	// Apply pagination
	query = base.ApplyPagination(query, filter.Pagination)
	
	// Fetch results
	var traces []trace.Trace
	if err := query.Find(&traces).Error; err != nil {
		return nil, 0, err
	}
	
	return traces, total, nil
}

// Update updates a trace
func (r *repositoryImpl) Update(ctx context.Context, tr *trace.Trace) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Model(tr).Where("id = ?", tr.ID).Updates(tr)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrTraceNotFound
	}
	
	return nil
}
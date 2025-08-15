package trace

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/enums"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/trace"
	base "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
	traceRepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/trace"
)

var (
	ErrTraceNotFound = errors.New("trace not found")
)

// Service defines the interface for trace business logic
type Service interface {
	Create(ctx context.Context, req CreateRequest) (*trace.Trace, error)
	GetByID(ctx context.Context, id uuid.UUID) (*trace.Trace, error)
	List(ctx context.Context, filter ListFilter) ([]trace.Trace, int64, error)
}

// CreateRequest represents a request to create a trace
type CreateRequest struct {
	Purpose        enums.TracePurpose   `json:"purpose"`
	RetentionClass enums.RetentionClass `json:"retention_class"`
	RepoID         *uuid.UUID           `json:"repo_id,omitempty"`
	Branch         *string              `json:"branch,omitempty"`
	CreatedBy      *string              `json:"created_by,omitempty"`
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

// serviceImpl implements Service interface
type serviceImpl struct {
	traceRepo traceRepo.Repository
}

// NewService creates a new trace service
func NewService(traceRepo traceRepo.Repository) Service {
	return &serviceImpl{
		traceRepo: traceRepo,
	}
}

// Create creates a new trace for correlation
func (s *serviceImpl) Create(ctx context.Context, req CreateRequest) (*trace.Trace, error) {
	// Set default retention class if not provided
	retentionClass := req.RetentionClass
	if retentionClass == "" {
		retentionClass = enums.RetentionClassLong
	}
	
	// Create trace
	t := &trace.Trace{
		Purpose:        req.Purpose,
		RetentionClass: retentionClass,
		RepoID:         req.RepoID,
		Branch:         req.Branch,
		CreatedBy:      req.CreatedBy,
	}
	
	if err := s.traceRepo.Create(ctx, t); err != nil {
		return nil, err
	}
	
	return t, nil
}

// GetByID retrieves a trace by ID
func (s *serviceImpl) GetByID(ctx context.Context, id uuid.UUID) (*trace.Trace, error) {
	t, err := s.traceRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, traceRepo.ErrTraceNotFound) {
			return nil, ErrTraceNotFound
		}
		return nil, err
	}
	
	return t, nil
}

// List retrieves traces with filters
func (s *serviceImpl) List(ctx context.Context, filter ListFilter) ([]trace.Trace, int64, error) {
	repoFilter := traceRepo.ListFilter{
		RepoID:         filter.RepoID,
		Purpose:        filter.Purpose,
		RetentionClass: filter.RetentionClass,
		Branch:         filter.Branch,
		CreatedBy:      filter.CreatedBy,
		Since:          filter.Since,
		Until:          filter.Until,
		Pagination:     filter.Pagination,
		Sort:           filter.Sort,
	}
	
	return s.traceRepo.List(ctx, repoFilter)
}
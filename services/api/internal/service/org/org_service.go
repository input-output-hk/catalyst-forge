package org

import (
	"context"

	"github.com/google/uuid"
	model "github.com/input-output-hk/catalyst-forge/services/api/internal/models/org"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
	repo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/org"
)

// Service defines organization business operations.
type Service interface {
	Create(ctx context.Context, name string, isDefault bool) (*model.Organization, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error)
	GetDefault(ctx context.Context) (*model.Organization, error)
	List(ctx context.Context, pag *base.Pagination, sort *base.Sort) ([]model.Organization, int64, error)
	SetDefault(ctx context.Context, id uuid.UUID) error
	UpdateName(ctx context.Context, id uuid.UUID, name string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type serviceImpl struct{ r repo.Repository }

func NewService(r repo.Repository) Service { return &serviceImpl{r: r} }

func (s *serviceImpl) Create(ctx context.Context, name string, isDefault bool) (*model.Organization, error) {
	o := &model.Organization{Name: name, IsDefault: isDefault}
	if err := s.r.Create(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}
func (s *serviceImpl) GetByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	return s.r.GetByID(ctx, id)
}
func (s *serviceImpl) GetDefault(ctx context.Context) (*model.Organization, error) {
	return s.r.GetDefault(ctx)
}
func (s *serviceImpl) List(ctx context.Context, pag *base.Pagination, sort *base.Sort) ([]model.Organization, int64, error) {
	return s.r.List(ctx, pag, sort)
}
func (s *serviceImpl) SetDefault(ctx context.Context, id uuid.UUID) error {
	return s.r.SetDefault(ctx, id)
}
func (s *serviceImpl) UpdateName(ctx context.Context, id uuid.UUID, name string) error {
	return s.r.UpdateName(ctx, id, name)
}
func (s *serviceImpl) Delete(ctx context.Context, id uuid.UUID) error { return s.r.Delete(ctx, id) }

package policy

import (
	"context"

	"gorm.io/gorm"
)

// Repository provides CRUD access for APIPolicy.
type Repository interface {
	List(ctx context.Context) ([]APIPolicy, error)
	Upsert(ctx context.Context, p *APIPolicy) error
	Delete(ctx context.Context, id uint) error
}

type repo struct{ db *gorm.DB }

// NewRepository creates a new policy repository.
func NewRepository(db *gorm.DB) Repository { return &repo{db: db} }

func (r *repo) List(ctx context.Context) ([]APIPolicy, error) {
	var out []APIPolicy
	if err := r.db.WithContext(ctx).Order("priority DESC, id ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repo) Upsert(ctx context.Context, p *APIPolicy) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *repo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&APIPolicy{ID: id}).Error
}

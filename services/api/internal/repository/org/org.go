package org

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/org"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
)

var (
	ErrOrgNotFound   = errors.New("organization not found")
	ErrOrgExists     = errors.New("organization already exists")
	ErrDefaultExists = errors.New("default organization already set")
)

type Repository interface {
	Create(ctx context.Context, o *org.Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*org.Organization, error)
	GetDefault(ctx context.Context) (*org.Organization, error)
	List(ctx context.Context, pag *base.Pagination, sort *base.Sort) ([]org.Organization, int64, error)
	SetDefault(ctx context.Context, id uuid.UUID) error
	UpdateName(ctx context.Context, id uuid.UUID, name string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repositoryImpl struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &repositoryImpl{db: db} }

func (r *repositoryImpl) Create(ctx context.Context, o *org.Organization) error {
	db := base.GetDB(ctx, r.db)
	if o.IsDefault {
		// enforce only one default
		var count int64
		if err := db.Model(&org.Organization{}).Where("is_default = ?", true).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrDefaultExists
		}
	}
	if err := db.Create(o).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrOrgExists
		}
		return err
	}
	return nil
}

func (r *repositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*org.Organization, error) {
	var o org.Organization
	if err := base.GetDB(ctx, r.db).Where("id = ?", id).First(&o).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return &o, nil
}

func (r *repositoryImpl) GetDefault(ctx context.Context) (*org.Organization, error) {
	var o org.Organization
	if err := base.GetDB(ctx, r.db).Where("is_default = ?", true).First(&o).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return &o, nil
}

func (r *repositoryImpl) List(ctx context.Context, pag *base.Pagination, sort *base.Sort) ([]org.Organization, int64, error) {
	db := base.ApplyPagination(base.ApplySort(base.GetDB(ctx, r.db).Model(&org.Organization{}), sort, "created_at DESC"), pag)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []org.Organization
	if err := db.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *repositoryImpl) SetDefault(ctx context.Context, id uuid.UUID) error {
	return base.GetDB(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&org.Organization{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return err
		}
		res := tx.Model(&org.Organization{}).Where("id = ?", id).Update("is_default", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrOrgNotFound
		}
		return nil
	})
}

func (r *repositoryImpl) UpdateName(ctx context.Context, id uuid.UUID, name string) error {
	res := base.GetDB(ctx, r.db).Model(&org.Organization{}).Where("id = ?", id).Update("name", name)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
			return ErrOrgExists
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrOrgNotFound
	}
	return nil
}

func (r *repositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	res := base.GetDB(ctx, r.db).Where("id = ?", id).Delete(&org.Organization{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrOrgNotFound
	}
	return nil
}

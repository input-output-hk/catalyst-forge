package gormstore

import (
	"context"

	r "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/rbac"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BindingStore struct{ db *gorm.DB }

func NewBindingStore(db *gorm.DB) *BindingStore { return &BindingStore{db: db} }

// AutoMigrate migrates RBAC binding-related tables.
func (s *BindingStore) AutoMigrate() error {
	return s.db.AutoMigrate(&Binding{}, &PrincipalVersion{})
}

func (s *BindingStore) AddBinding(ctx context.Context, b r.Binding) error {
	m := Binding{
		ID:          b.ID,
		SubjectType: string(b.Subject.Type),
		SubjectID:   b.Subject.ID,
		RoleSlug:    b.RoleSlug,
		ScopeType:   string(b.ScopeType),
		ScopeID:     b.ScopeID,
		OrgID:       b.OrgID,
		CreatedAt:   b.CreatedAt,
	}
	return s.db.WithContext(ctx).Create(&m).Error
}

func (s *BindingStore) RemoveBinding(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&Binding{}).Error
}

func (s *BindingStore) ListBindings(ctx context.Context, subj r.Subject) ([]r.Binding, error) {
	var ms []Binding
	if err := s.db.WithContext(ctx).Where("subject_type = ? AND subject_id = ?", string(subj.Type), subj.ID).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]r.Binding, 0, len(ms))
	for _, m := range ms {
		out = append(out, r.Binding{
			ID:        m.ID,
			Subject:   subj,
			RoleSlug:  m.RoleSlug,
			ScopeType: r.ScopeType(m.ScopeType),
			ScopeID:   m.ScopeID,
			OrgID:     m.OrgID,
			CreatedAt: m.CreatedAt,
		})
	}
	return out, nil
}

func (s *BindingStore) ListBindingsByScope(ctx context.Context, scope r.ScopeType, scopeID string) ([]r.Binding, error) {
	var ms []Binding
	if err := s.db.WithContext(ctx).Where("scope_type = ? AND scope_id = ?", string(scope), scopeID).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]r.Binding, 0, len(ms))
	for _, m := range ms {
		out = append(out, r.Binding{
			ID:        m.ID,
			Subject:   r.Subject{Type: r.SubjectType(m.SubjectType), ID: m.SubjectID, OrgID: m.OrgID},
			RoleSlug:  m.RoleSlug,
			ScopeType: r.ScopeType(m.ScopeType),
			ScopeID:   m.ScopeID,
			OrgID:     m.OrgID,
			CreatedAt: m.CreatedAt,
		})
	}
	return out, nil
}

func (s *BindingStore) GetPrincipalVersion(ctx context.Context, subj r.Subject) (int64, error) {
	var pv PrincipalVersion
	err := s.db.WithContext(ctx).Where("subject_type = ? AND subject_id = ?", string(subj.Type), subj.ID).First(&pv).Error
	if err == gorm.ErrRecordNotFound {
		// initialize at zero
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return pv.Version, nil
}

func (s *BindingStore) BumpPrincipalVersion(ctx context.Context, subj r.Subject) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pv PrincipalVersion
		err := tx.Where("subject_type = ? AND subject_id = ?", string(subj.Type), subj.ID).First(&pv).Error
		if err == gorm.ErrRecordNotFound {
			pv = PrincipalVersion{SubjectType: string(subj.Type), SubjectID: subj.ID, Version: 1}
			return tx.Create(&pv).Error
		}
		if err != nil {
			return err
		}
		pv.Version++
		return tx.Model(&pv).UpdateColumn("version", pv.Version).Error
	})
}

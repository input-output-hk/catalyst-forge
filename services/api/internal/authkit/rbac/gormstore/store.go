package gormstore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	r "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac"
	"gorm.io/gorm"
)

// Store implements rbac.Store using GORM.
type Store struct{ db *gorm.DB }

func New(db *gorm.DB) *Store { return &Store{db: db} }

func (s *Store) AutoMigrate() error {
	return s.db.AutoMigrate(&Role{}, &RoleEntry{}, &Binding{}, &PrincipalVersion{})
}

// Roles
func (s *Store) CreateRole(ctx context.Context, role r.RoleDef) error {
	rid := role.ID
	if rid == uuid.Nil {
		rid = uuid.New()
	}
	m := Role{ID: rid, Slug: role.Slug, Name: role.Name, Description: role.Description, Color: role.Color, Version: role.Version}
	entries := make([]RoleEntry, 0, len(role.Entries))
	for _, e := range role.Entries {
		var conds json.RawMessage
		if len(e.Conditions) > 0 {
			b, _ := json.Marshal(e.Conditions)
			conds = b
		}
		entries = append(entries, RoleEntry{ID: uuid.New(), RoleID: rid, Effect: string(e.Effect), Permission: string(e.Permission), ResourceType: e.ResourceType, Conditions: conds})
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		if len(entries) > 0 {
			if err := tx.Create(&entries).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) GetRole(ctx context.Context, slug string) (*r.RoleDef, error) {
	var m Role
	if err := s.db.WithContext(ctx).Where("slug = ?", slug).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	var es []RoleEntry
	if err := s.db.WithContext(ctx).Where("role_id = ?", m.ID).Find(&es).Error; err != nil {
		return nil, err
	}
	d := r.RoleDef{ID: m.ID, Slug: m.Slug, Name: m.Name, Description: m.Description, Color: m.Color, Version: m.Version}
	for _, e := range es {
		var conds []r.Condition
		if len(e.Conditions) > 0 {
			_ = json.Unmarshal(e.Conditions, &conds)
		}
		d.Entries = append(d.Entries, r.RoleEntry{Effect: r.Effect(e.Effect), Permission: r.PermissionKey(e.Permission), ResourceType: e.ResourceType, Conditions: conds})
	}
	return &d, nil
}

func (s *Store) UpdateRole(ctx context.Context, role r.RoleDef) error {
	// Assume role exists; update basic fields and replace entries
	var existing Role
	if err := s.db.WithContext(ctx).Where("slug = ?", role.Slug).First(&existing).Error; err != nil {
		return err
	}
	existing.Name, existing.Description, existing.Color, existing.Version = role.Name, role.Description, role.Color, role.Version
	entries := make([]RoleEntry, 0, len(role.Entries))
	for _, e := range role.Entries {
		var conds json.RawMessage
		if len(e.Conditions) > 0 {
			b, _ := json.Marshal(e.Conditions)
			conds = b
		}
		entries = append(entries, RoleEntry{ID: uuid.New(), RoleID: existing.ID, Effect: string(e.Effect), Permission: string(e.Permission), ResourceType: e.ResourceType, Conditions: conds})
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&existing).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", existing.ID).Delete(&RoleEntry{}).Error; err != nil {
			return err
		}
		if len(entries) > 0 {
			if err := tx.Create(&entries).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ListRoles(ctx context.Context) ([]r.RoleDef, error) {
	var ms []Role
	if err := s.db.WithContext(ctx).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]r.RoleDef, 0, len(ms))
	for _, m := range ms {
		out = append(out, r.RoleDef{ID: m.ID, Slug: m.Slug, Name: m.Name, Description: m.Description, Color: m.Color, Version: m.Version})
	}
	return out, nil
}

func (s *Store) BumpRoleVersion(ctx context.Context, slug string) error {
	return s.db.WithContext(ctx).Model(&Role{}).Where("slug = ?", slug).UpdateColumn("version", gorm.Expr("version + 1")).Error
}

// Bindings
func (s *Store) AddBinding(ctx context.Context, b r.Binding) error {
	// Safety: reject unknown scope types at the storage boundary
	if !r.IsKnownScope(b.ScopeType) {
		return fmt.Errorf("rbac: unknown scope type: %s", b.ScopeType)
	}
	m := Binding{ID: b.ID, SubjectType: string(b.Subject.Type), SubjectID: b.Subject.ID, RoleSlug: b.RoleSlug, ScopeType: string(b.ScopeType), ScopeID: b.ScopeID, OrgID: b.OrgID, CreatedAt: b.CreatedAt}
	return s.db.WithContext(ctx).Create(&m).Error
}

func (s *Store) RemoveBinding(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&Binding{}).Error
}

func (s *Store) ListBindings(ctx context.Context, subj r.Subject) ([]r.Binding, error) {
	var ms []Binding
	if err := s.db.WithContext(ctx).Where("subject_type = ? AND subject_id = ?", string(subj.Type), subj.ID).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]r.Binding, 0, len(ms))
	for _, m := range ms {
		out = append(out, r.Binding{ID: m.ID, Subject: subj, RoleSlug: m.RoleSlug, ScopeType: r.ScopeType(m.ScopeType), ScopeID: m.ScopeID, OrgID: m.OrgID, CreatedAt: m.CreatedAt})
	}
	return out, nil
}

func (s *Store) ListBindingsByScope(ctx context.Context, scope r.ScopeType, scopeID string) ([]r.Binding, error) {
	if !r.IsKnownScope(scope) {
		return nil, fmt.Errorf("rbac: unknown scope type: %s", scope)
	}
	var ms []Binding
	if err := s.db.WithContext(ctx).Where("scope_type = ? AND scope_id = ?", string(scope), scopeID).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]r.Binding, 0, len(ms))
	for _, m := range ms {
		out = append(out, r.Binding{ID: m.ID, Subject: r.Subject{Type: r.SubjectType(m.SubjectType), ID: m.SubjectID, OrgID: m.OrgID}, RoleSlug: m.RoleSlug, ScopeType: r.ScopeType(m.ScopeType), ScopeID: m.ScopeID, OrgID: m.OrgID, CreatedAt: m.CreatedAt})
	}
	return out, nil
}

// Principal versions
func (s *Store) GetPrincipalVersion(ctx context.Context, subj r.Subject) (int64, error) {
	var pv PrincipalVersion
	err := s.db.WithContext(ctx).Where("subject_type = ? AND subject_id = ?", string(subj.Type), subj.ID).First(&pv).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return pv.Version, nil
}

func (s *Store) BumpPrincipalVersion(ctx context.Context, subj r.Subject) error {
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

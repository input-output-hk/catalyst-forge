package gormstore

import (
	"context"

	r "github.com/catalystgo/catalyst-forge/lib/foundry/authkit/rbac"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoleStore struct{ db *gorm.DB }

func NewRoleStore(db *gorm.DB) *RoleStore { return &RoleStore{db: db} }

// AutoMigrate migrates RBAC role-related tables.
func (s *RoleStore) AutoMigrate() error {
	return s.db.AutoMigrate(&Role{}, &RoleEntry{})
}

func (s *RoleStore) toModel(role r.RoleDef) (Role, []RoleEntry) {
	rid := role.ID
	if rid == uuid.Nil {
		rid = uuid.New()
	}
	m := Role{
		ID:          rid,
		Slug:        role.Slug,
		Name:        role.Name,
		Description: role.Description,
		Version:     role.Version,
	}
	var entries []RoleEntry
	for _, e := range role.Entries {
		entries = append(entries, RoleEntry{
			ID:           uuid.New(),
			RoleID:       rid,
			Effect:       string(e.Effect),
			Permission:   string(e.Permission),
			ResourceType: e.ResourceType,
			// Conditions: filled in Update/Create when persisted as JSON if needed
		})
	}
	return m, entries
}

func (s *RoleStore) toDomain(m Role, entries []RoleEntry) r.RoleDef {
	out := r.RoleDef{
		ID:          m.ID,
		Slug:        m.Slug,
		Name:        m.Name,
		Description: m.Description,
		Version:     m.Version,
	}
	for _, e := range entries {
		out.Entries = append(out.Entries, r.RoleEntry{
			Effect:       r.Effect(e.Effect),
			Permission:   r.PermissionKey(e.Permission),
			ResourceType: e.ResourceType,
			// Conditions: populated in later phases
		})
	}
	return out
}

func (s *RoleStore) CreateRole(ctx context.Context, role r.RoleDef) error {
	m, entries := s.toModel(role)
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

func (s *RoleStore) GetRole(ctx context.Context, slug string) (*r.RoleDef, error) {
	var m Role
	if err := s.db.WithContext(ctx).Where("slug = ?", slug).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	var entries []RoleEntry
	if err := s.db.WithContext(ctx).Where("role_id = ?", m.ID).Find(&entries).Error; err != nil {
		return nil, err
	}
	d := s.toDomain(m, entries)
	return &d, nil
}

func (s *RoleStore) UpdateRole(ctx context.Context, role r.RoleDef) error {
	m, entries := s.toModel(role)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Role{}).Where("slug = ?", role.Slug).Updates(&m).Error; err != nil {
			return err
		}
		// replace entries
		if err := tx.Where("role_id = ?", m.ID).Delete(&RoleEntry{}).Error; err != nil {
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

func (s *RoleStore) ListRoles(ctx context.Context) ([]r.RoleDef, error) {
	var roles []Role
	if err := s.db.WithContext(ctx).Find(&roles).Error; err != nil {
		return nil, err
	}
	out := make([]r.RoleDef, 0, len(roles))
	for _, m := range roles {
		// entries are omitted in list for efficiency; caller can GetRole for full details
		out = append(out, r.RoleDef{ID: m.ID, Slug: m.Slug, Name: m.Name, Description: m.Description, Version: m.Version})
	}
	return out, nil
}

func (s *RoleStore) BumpRoleVersion(ctx context.Context, slug string) error {
	return s.db.WithContext(ctx).Model(&Role{}).Where("slug = ?", slug).UpdateColumn("version", gorm.Expr("version + 1")).Error
}

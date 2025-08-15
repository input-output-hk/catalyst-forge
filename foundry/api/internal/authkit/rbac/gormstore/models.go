package gormstore

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Slug        string    `gorm:"uniqueIndex"`
	Name        string
	Description string
	Version     int64 `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RoleEntry struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey"`
	RoleID       uuid.UUID       `gorm:"index"`
	Effect       string          // "allow" | "deny"
	Permission   string          // app-defined permission key
	ResourceType string          // nullable
	Conditions   json.RawMessage // array of {name, params}
}

type Binding struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	SubjectType string     `gorm:"index"`
	SubjectID   string     `gorm:"index"`
	RoleSlug    string     `gorm:"index"`
	ScopeType   string     `gorm:"index"`
	ScopeID     string     `gorm:"index"`
	OrgID       *uuid.UUID `gorm:"index"`
	CreatedAt   time.Time
}

type PrincipalVersion struct {
	SubjectType string `gorm:"primaryKey"`
	SubjectID   string `gorm:"primaryKey"`
	Version     int64
	UpdatedAt   time.Time
}

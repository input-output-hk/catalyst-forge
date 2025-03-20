package models

import (
	"time"

	"gorm.io/gorm"
)

// ReleaseAlias represents an alias for a release
type ReleaseAlias struct {
	Name      string         `gorm:"primaryKey" json:"name"`
	ReleaseID string         `gorm:"not null;index" json:"release_id"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Release Release `gorm:"foreignKey:ReleaseID" json:"release,omitempty"`
}

// TableName specifies the table name for the ReleaseAlias model
func (ReleaseAlias) TableName() string {
	return "release_aliases"
}

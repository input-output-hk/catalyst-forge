package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository represents a source repository (host/org/name)
type Repository struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Host          string    `gorm:"not null" json:"host"`
	Org           string    `gorm:"not null" json:"org"`
	Name          string    `gorm:"not null" json:"name"`
	DefaultBranch string    `gorm:"not null;default:'main'" json:"default_branch"`
	CreatedAt     time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName specifies the table name
func (Repository) TableName() string {
	return "repository"
}

// BeforeCreate hook to set UUID if not provided
func (r *Repository) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
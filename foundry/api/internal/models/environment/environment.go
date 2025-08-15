package environment

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Environment represents deployment environments (dev, preprod, prod)
type Environment struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"not null;uniqueIndex:ux_environment_name" json:"name"`
	Cluster     string    `gorm:"not null" json:"cluster"`
	ArgoProject *string   `json:"argo_project,omitempty"`
	IsProtected bool      `gorm:"not null;default:false" json:"is_protected"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName specifies the table name
func (Environment) TableName() string {
	return "environment"
}

// BeforeCreate hook to set UUID if not provided
func (e *Environment) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
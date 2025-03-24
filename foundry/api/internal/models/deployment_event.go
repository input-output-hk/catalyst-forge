package models

import (
	"time"

	"gorm.io/gorm"
)

// DeploymentEvent represents an event that occurred during a deployment
type DeploymentEvent struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	DeploymentID string         `gorm:"not null;index" json:"deployment_id"`
	Name         string         `gorm:"not null" json:"name"`
	Message      string         `gorm:"not null" json:"message"`
	Timestamp    time.Time      `gorm:"not null;index" json:"timestamp"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	Deployment *ReleaseDeployment `gorm:"foreignKey:DeploymentID" json:"-"`
}

// TableName specifies the table name for the DeploymentEvent model
func (DeploymentEvent) TableName() string {
	return "deployment_events"
}

package models

import (
	"time"

	"gorm.io/gorm"
)

// DeploymentStatus type for deployment status
type DeploymentStatus string

// Possible deployment statuses
const (
	DeploymentStatusPending   DeploymentStatus = "pending"
	DeploymentStatusRunning   DeploymentStatus = "running"
	DeploymentStatusSucceeded DeploymentStatus = "succeeded"
	DeploymentStatusFailed    DeploymentStatus = "failed"
)

// ReleaseDeployment represents a point-in-time deployment of a specific release
type ReleaseDeployment struct {
	ID        string           `gorm:"primaryKey" json:"id"`
	ReleaseID string           `gorm:"not null;index" json:"release_id"`
	Timestamp time.Time        `gorm:"not null" json:"timestamp"`
	Status    DeploymentStatus `gorm:"not null;type:string;default:'pending'" json:"status"`
	Reason    string           `json:"reason,omitempty"`

	// Relationships
	Release Release `gorm:"foreignKey:ReleaseID" json:"release,omitempty"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the ReleaseDeployment model
func (ReleaseDeployment) TableName() string {
	return "release_deployments"
}

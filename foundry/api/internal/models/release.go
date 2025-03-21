package models

import (
	"time"

	"gorm.io/gorm"
)

// Release represents a point-in-time project release
type Release struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	SourceRepo   string    `gorm:"not null" json:"source_repo"`
	SourceCommit string    `gorm:"not null" json:"source_commit"`
	SourceBranch string    `json:"source_branch,omitempty"`
	Project      string    `gorm:"not null;index" json:"project"`
	ProjectPath  string    `gorm:"not null" json:"project_path"`
	Created      time.Time `gorm:"not null" json:"created"`
	Bundle       string    `gorm:"type:text;not null" json:"bundle"`

	// Relationships
	Deployments []ReleaseDeployment `gorm:"foreignKey:ReleaseID" json:"deployments,omitempty"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the Release model
func (Release) TableName() string {
	return "releases"
}

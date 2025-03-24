package client

import (
	"time"
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

// Release represents a point-in-time project release
type Release struct {
	ID           string    `json:"id"`
	SourceRepo   string    `json:"source_repo"`
	SourceCommit string    `json:"source_commit"`
	SourceBranch string    `json:"source_branch,omitempty"`
	Project      string    `json:"project"`
	ProjectPath  string    `json:"project_path"`
	Created      time.Time `json:"created"`
	Bundle       string    `json:"bundle"` // Base64-encoded source code

	// Relationships
	Deployments []ReleaseDeployment `json:"deployments,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeploymentEvent represents an event that occurred during a deployment
type DeploymentEvent struct {
	ID           uint      `json:"id"`
	DeploymentID string    `json:"deployment_id"`
	Name         string    `json:"name"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ReleaseDeployment represents a point-in-time deployment of a specific release
type ReleaseDeployment struct {
	ID        string           `json:"id"` // Generated from ReleaseID + timestamp
	ReleaseID string           `json:"release_id"`
	Timestamp time.Time        `json:"timestamp"`
	Status    DeploymentStatus `json:"status"`
	Reason    string           `json:"reason,omitempty"`
	Attempts  int              `json:"attempts"`

	// Relationships
	Release *Release          `json:"release,omitempty"`
	Events  []DeploymentEvent `json:"events,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReleaseAlias represents an alias for a release
type ReleaseAlias struct {
	Name      string    `json:"name"`
	ReleaseID string    `json:"release_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Release *Release `json:"release,omitempty"`
}

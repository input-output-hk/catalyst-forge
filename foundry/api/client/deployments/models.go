package deployments

import (
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/client/releases"
)

// ReleaseDeployment represents a point-in-time deployment of a specific release
type ReleaseDeployment struct {
	ID        string           `json:"id"` // Generated from ReleaseID + timestamp
	ReleaseID string           `json:"release_id"`
	Timestamp time.Time        `json:"timestamp"`
	Status    DeploymentStatus `json:"status"`
	Reason    string           `json:"reason,omitempty"`
	Attempts  int              `json:"attempts"`

	// Relationships
	Release *releases.Release `json:"release,omitempty"`
	Events  []DeploymentEvent `json:"events,omitempty"`

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

// DeploymentStatus type for deployment status
type DeploymentStatus string

// Possible deployment statuses
const (
	DeploymentStatusPending   DeploymentStatus = "pending"
	DeploymentStatusRunning   DeploymentStatus = "running"
	DeploymentStatusSucceeded DeploymentStatus = "succeeded"
	DeploymentStatusFailed    DeploymentStatus = "failed"
)

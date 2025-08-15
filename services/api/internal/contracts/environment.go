package contracts

import (
	"time"

	"github.com/google/uuid"
)

// EnvironmentCreate represents a request to create an environment
type EnvironmentCreate struct {
	ProjectID       string                 `json:"project_id" binding:"required,uuid4"`
	Name            string                 `json:"name" binding:"required"`
	EnvironmentType string                 `json:"environment_type" binding:"required,oneof=dev staging prod"`
	ClusterRef      *string                `json:"cluster_ref,omitempty"`
	Namespace       *string                `json:"namespace,omitempty"`
	Region          *string                `json:"region,omitempty"`
	CloudProvider   *string                `json:"cloud_provider,omitempty" binding:"omitempty,oneof=aws gcp azure other"`
	Config          map[string]interface{} `json:"config,omitempty"`
	Secrets         map[string]interface{} `json:"secrets,omitempty"`
	ProtectionRules map[string]interface{} `json:"protection_rules,omitempty"`
	Active          bool                   `json:"active"`
}

// EnvironmentUpdate represents a request to update an environment
type EnvironmentUpdate struct {
	Name            *string                `json:"name,omitempty"`
	EnvironmentType *string                `json:"environment_type,omitempty" binding:"omitempty,oneof=dev staging prod"`
	ClusterRef      *string                `json:"cluster_ref,omitempty"`
	Namespace       *string                `json:"namespace,omitempty"`
	Region          *string                `json:"region,omitempty"`
	CloudProvider   *string                `json:"cloud_provider,omitempty" binding:"omitempty,oneof=aws gcp azure other"`
	Config          map[string]interface{} `json:"config,omitempty"`
	Secrets         map[string]interface{} `json:"secrets,omitempty"`
	ProtectionRules map[string]interface{} `json:"protection_rules,omitempty"`
	Active          *bool                  `json:"active,omitempty"`
}

// EnvironmentResponse represents an environment response
type EnvironmentResponse struct {
	ID              string                 `json:"id"`
	ProjectID       string                 `json:"project_id"`
	Name            string                 `json:"name"`
	EnvironmentType string                 `json:"environment_type"`
	ClusterRef      *string                `json:"cluster_ref,omitempty"`
	Namespace       *string                `json:"namespace,omitempty"`
	Region          *string                `json:"region,omitempty"`
	CloudProvider   *string                `json:"cloud_provider,omitempty"`
	Config          map[string]interface{} `json:"config,omitempty"`
	Secrets         map[string]interface{} `json:"secrets,omitempty"`
	ProtectionRules map[string]interface{} `json:"protection_rules,omitempty"`
	Active          bool                   `json:"active"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// EnvironmentListFilter represents filters for listing environments
type EnvironmentListFilter struct {
	ProjectID       *string `json:"project_id,omitempty" form:"project_id" binding:"omitempty,uuid4"`
	Name            *string `json:"name,omitempty" form:"name"`
	EnvironmentType *string `json:"environment_type,omitempty" form:"environment_type" binding:"omitempty,oneof=dev staging prod"`
	ClusterRef      *string `json:"cluster_ref,omitempty" form:"cluster_ref"`
	Namespace       *string `json:"namespace,omitempty" form:"namespace"`
	Region          *string `json:"region,omitempty" form:"region"`
	CloudProvider   *string `json:"cloud_provider,omitempty" form:"cloud_provider" binding:"omitempty,oneof=aws gcp azure other"`
	Active          *bool   `json:"active,omitempty" form:"active"`
	TimeRange
	Pagination
	Sort
}

// EnvironmentIDParam represents an environment ID parameter
type EnvironmentIDParam struct {
	EnvironmentID uuid.UUID `uri:"environment_id" binding:"required,uuid4"`
}

// EnvironmentNameParam represents an environment name parameter
type EnvironmentNameParam struct {
	ProjectID uuid.UUID `uri:"project_id" binding:"required,uuid4"`
	Name      string    `uri:"name" binding:"required"`
}
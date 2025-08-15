package v2

import (
	"time"

	"github.com/google/uuid"
)

// ArtifactCreate represents a request to create an artifact
type ArtifactCreate struct {
	BuildID     string  `json:"build_id" binding:"required,uuid4"`
	ProjectID   string  `json:"project_id" binding:"required,uuid4"`
	ImageName   string  `json:"image_name" binding:"required"`
	ImageDigest string  `json:"image_digest" binding:"required"`
	Tag         *string `json:"tag,omitempty"`
	Repo        *string `json:"repo,omitempty"`
	Provider    *string `json:"provider,omitempty" binding:"omitempty,oneof=dockerhub gcr ecr quay ghcr other"`
	BuildArgs   map[string]interface{} `json:"build_args,omitempty"`
	BuildMeta   map[string]interface{} `json:"build_meta,omitempty"`
	ScanStatus  *string `json:"scan_status,omitempty" binding:"omitempty,oneof=pending passed failed skipped"`
	ScanResults map[string]interface{} `json:"scan_results,omitempty"`
	SignedBy    *string `json:"signed_by,omitempty"`
}

// ArtifactUpdate represents a request to update an artifact
type ArtifactUpdate struct {
	Tag         *string                `json:"tag,omitempty"`
	ScanStatus  *string                `json:"scan_status,omitempty" binding:"omitempty,oneof=pending passed failed skipped"`
	ScanResults map[string]interface{} `json:"scan_results,omitempty"`
	SignedBy    *string                `json:"signed_by,omitempty"`
	SignedAt    *time.Time             `json:"signed_at,omitempty"`
}

// ArtifactResponse represents an artifact response
type ArtifactResponse struct {
	ID          string                 `json:"id"`
	BuildID     string                 `json:"build_id"`
	ProjectID   string                 `json:"project_id"`
	ImageName   string                 `json:"image_name"`
	ImageDigest string                 `json:"image_digest"`
	Tag         *string                `json:"tag,omitempty"`
	Repo        *string                `json:"repo,omitempty"`
	Provider    *string                `json:"provider,omitempty"`
	BuildArgs   map[string]interface{} `json:"build_args,omitempty"`
	BuildMeta   map[string]interface{} `json:"build_meta,omitempty"`
	ScanStatus  *string                `json:"scan_status,omitempty"`
	ScanResults map[string]interface{} `json:"scan_results,omitempty"`
	SignedBy    *string                `json:"signed_by,omitempty"`
	SignedAt    *time.Time             `json:"signed_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ArtifactListFilter represents filters for listing artifacts
type ArtifactListFilter struct {
	BuildID     *string `json:"build_id,omitempty" form:"build_id" binding:"omitempty,uuid4"`
	ProjectID   *string `json:"project_id,omitempty" form:"project_id" binding:"omitempty,uuid4"`
	ImageName   *string `json:"image_name,omitempty" form:"image_name"`
	ImageDigest *string `json:"image_digest,omitempty" form:"image_digest"`
	Tag         *string `json:"tag,omitempty" form:"tag"`
	Repo        *string `json:"repo,omitempty" form:"repo"`
	Provider    *string `json:"provider,omitempty" form:"provider" binding:"omitempty,oneof=dockerhub gcr ecr quay ghcr other"`
	ScanStatus  *string `json:"scan_status,omitempty" form:"scan_status" binding:"omitempty,oneof=pending passed failed skipped"`
	SignedBy    *string `json:"signed_by,omitempty" form:"signed_by"`
	TimeRange
	Pagination
	Sort
}

// ArtifactIDParam represents an artifact ID parameter
type ArtifactIDParam struct {
	ArtifactID uuid.UUID `uri:"artifact_id" binding:"required,uuid4"`
}

// ArtifactDigestParam represents an artifact digest parameter
type ArtifactDigestParam struct {
	Digest string `uri:"digest" binding:"required"`
}
package contracts

import (
	"time"

	"github.com/google/uuid"
)

// PromotionCreate represents a request to create a promotion
type PromotionCreate struct {
	ProjectID     string                 `json:"project_id" binding:"required,uuid4"`
	ReleaseID     string                 `json:"release_id" binding:"required,uuid4"`
	EnvironmentID string                 `json:"environment_id" binding:"required,uuid4"`
	ApprovalMode  string                 `json:"approval_mode" binding:"required,oneof=manual auto"`
	RequestedBy   string                 `json:"requested_by" binding:"required"`
	Reason        *string                `json:"reason,omitempty"`
	PolicyResults map[string]interface{} `json:"policy_results,omitempty"`
}

// PromotionUpdate represents a request to update a promotion
type PromotionUpdate struct {
	Status           *string                `json:"status,omitempty" binding:"omitempty,oneof=requested approved submitted completed failed canceled superseded rejected"`
	Reason           *string                `json:"reason,omitempty"`
	ApproverID       *string                `json:"approver_id,omitempty"`
	ApprovedAt       *time.Time             `json:"approved_at,omitempty"`
	StepUpVerifiedAt *time.Time             `json:"step_up_verified_at,omitempty"`
	PolicyResults    map[string]interface{} `json:"policy_results,omitempty"`
	DeploymentID     *string                `json:"deployment_id,omitempty" binding:"omitempty,uuid4"`
	TraceID          *string                `json:"trace_id,omitempty" binding:"omitempty,uuid4"`
}

// PromotionResponse represents a promotion response
type PromotionResponse struct {
	ID               string                 `json:"id"`
	ProjectID        string                 `json:"project_id"`
	ReleaseID        string                 `json:"release_id"`
	EnvironmentID    string                 `json:"environment_id"`
	Status           string                 `json:"status"`
	ApprovalMode     string                 `json:"approval_mode"`
	RequestedBy      string                 `json:"requested_by"`
	RequestedAt      time.Time              `json:"requested_at"`
	Reason           *string                `json:"reason,omitempty"`
	ApproverID       *string                `json:"approver_id,omitempty"`
	ApprovedAt       *time.Time             `json:"approved_at,omitempty"`
	StepUpVerifiedAt *time.Time             `json:"step_up_verified_at,omitempty"`
	PolicyResults    map[string]interface{} `json:"policy_results,omitempty"`
	DeploymentID     *string                `json:"deployment_id,omitempty"`
	TraceID          *string                `json:"trace_id,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// PromotionListFilter represents filters for listing promotions
type PromotionListFilter struct {
	ProjectID     *string `json:"project_id,omitempty" form:"project_id" binding:"omitempty,uuid4"`
	EnvironmentID *string `json:"environment_id,omitempty" form:"environment_id" binding:"omitempty,uuid4"`
	ReleaseID     *string `json:"release_id,omitempty" form:"release_id" binding:"omitempty,uuid4"`
	Status        *string `json:"status,omitempty" form:"status" binding:"omitempty,oneof=requested approved submitted completed failed canceled superseded rejected"`
	TimeRange
	Pagination
	Sort
}

// PromotionIDParam represents a promotion ID parameter
type PromotionIDParam struct {
	PromotionID uuid.UUID `uri:"promotion_id" binding:"required,uuid4"`
}

// PromotionPageResult represents a paginated list of promotions
type PromotionPageResult struct {
	Items    []PromotionResponse `json:"items"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	Total    int64               `json:"total"`
}

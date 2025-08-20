package deployment

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PromotionStatus represents the lifecycle status of a promotion
type PromotionStatus string

const (
	PromotionStatusRequested  PromotionStatus = "requested"
	PromotionStatusApproved   PromotionStatus = "approved"
	PromotionStatusSubmitted  PromotionStatus = "submitted"
	PromotionStatusCompleted  PromotionStatus = "completed"
	PromotionStatusFailed     PromotionStatus = "failed"
	PromotionStatusCanceled   PromotionStatus = "canceled"
	PromotionStatusSuperseded PromotionStatus = "superseded"
	PromotionStatusRejected   PromotionStatus = "rejected"
)

// ApprovalMode represents how a promotion is approved
type ApprovalMode string

const (
	ApprovalModeManual ApprovalMode = "manual"
	ApprovalModeAuto   ApprovalMode = "auto"
)

// Promotion models a request/approval workflow to promote a release to an environment
type Promotion struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProjectID uuid.UUID `gorm:"type:uuid;not null;index:ix_promotions_proj_env,priority:1" json:"project_id"`
	ReleaseID uuid.UUID `gorm:"type:uuid;not null;index:ix_promotions_release" json:"release_id"`
	// Explicit column name to match manual index SQL in migrations (environment_id)
	EnvID uuid.UUID `gorm:"type:uuid;not null;column:environment_id;index:ix_promotions_proj_env,priority:2" json:"environment_id"`

	Status       PromotionStatus `gorm:"not null;index:ix_promotions_status" json:"status"`
	Reason       *string         `json:"reason,omitempty"`
	ApprovalMode ApprovalMode    `gorm:"not null" json:"approval_mode"`
	RequestedBy  string          `gorm:"not null" json:"requested_by"`
	RequestedAt  time.Time       `gorm:"not null;default:now()" json:"requested_at"`

	ApproverID       *string    `json:"approver_id,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	StepUpVerifiedAt *time.Time `json:"step_up_verified_at,omitempty"`
	PolicyResults    JSONB      `gorm:"type:jsonb;not null" json:"policy_results"`

	DeploymentID *uuid.UUID `gorm:"type:uuid" json:"deployment_id,omitempty"`
	TraceID      *uuid.UUID `gorm:"type:uuid" json:"trace_id,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now();index:ix_promotions_proj_env,priority:3,sort:desc" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName specifies the table name
func (Promotion) TableName() string {
	return "promotions"
}

// BeforeCreate hook to set UUID if not provided
func (p *Promotion) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

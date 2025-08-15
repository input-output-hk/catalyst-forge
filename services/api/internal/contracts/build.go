package contracts

import (
	"time"

	"github.com/google/uuid"
)

// BuildCreate represents a request to create a build
type BuildCreate struct {
	TraceID       *string                `json:"trace_id,omitempty" binding:"omitempty,uuid4"`
	RepoID        string                 `json:"repo_id" binding:"required,uuid4"`
	ProjectID     string                 `json:"project_id" binding:"required,uuid4"`
	CommitSHA     string                 `json:"commit_sha" binding:"required"`
	Branch        *string                `json:"branch,omitempty"`
	WorkflowRunID *string                `json:"workflow_run_id,omitempty"`
	Status        string                 `json:"status" binding:"required,oneof=queued running success failed canceled"`
	RunnerEnv     map[string]interface{} `json:"runner_env,omitempty"`
}

// BuildUpdate represents a request to update a build
type BuildUpdate struct {
	Status        *string                `json:"status,omitempty" binding:"omitempty,oneof=queued running success failed canceled"`
	WorkflowRunID *string                `json:"workflow_run_id,omitempty"`
	RunnerEnv     map[string]interface{} `json:"runner_env,omitempty"`
	FinishedAt    *time.Time             `json:"finished_at,omitempty"`
}

// BuildStatusUpdate represents a request to update only build status
type BuildStatusUpdate struct {
	Status string `json:"status" binding:"required,oneof=queued running success failed canceled"`
}

// BuildResponse represents a build response
type BuildResponse struct {
	ID            string                 `json:"id"`
	TraceID       *string                `json:"trace_id,omitempty"`
	RepoID        string                 `json:"repo_id"`
	ProjectID     string                 `json:"project_id"`
	CommitSHA     string                 `json:"commit_sha"`
	Branch        *string                `json:"branch,omitempty"`
	WorkflowRunID *string                `json:"workflow_run_id,omitempty"`
	Status        string                 `json:"status"`
	RunnerEnv     map[string]interface{} `json:"runner_env,omitempty"`
	FinishedAt    *time.Time             `json:"finished_at,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// BuildListFilter represents filters for listing builds
type BuildListFilter struct {
	TraceID       *string `json:"trace_id,omitempty" form:"trace_id" binding:"omitempty,uuid4"`
	RepoID        *string `json:"repo_id,omitempty" form:"repo_id" binding:"omitempty,uuid4"`
	ProjectID     *string `json:"project_id,omitempty" form:"project_id" binding:"omitempty,uuid4"`
	CommitSHA     *string `json:"commit_sha,omitempty" form:"commit_sha"`
	Branch        *string `json:"branch,omitempty" form:"branch"`
	WorkflowRunID *string `json:"workflow_run_id,omitempty" form:"workflow_run_id"`
	Status        *string `json:"status,omitempty" form:"status" binding:"omitempty,oneof=queued running success failed canceled"`
	TimeRange
	Pagination
	Sort
}

// BuildIDParam represents a build ID parameter
type BuildIDParam struct {
	BuildID uuid.UUID `uri:"build_id" binding:"required,uuid4"`
}
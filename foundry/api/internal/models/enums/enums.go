package enums

// TracePurpose represents the purpose of a trace
type TracePurpose string

const (
	TracePurposePRCheck       TracePurpose = "pr-check"
	TracePurposeMergeBuild    TracePurpose = "merge-build"
	TracePurposeTagRelease    TracePurpose = "tag-release"
	TracePurposeManualRelease TracePurpose = "manual-release"
	TracePurposeDeploy        TracePurpose = "deploy"
	TracePurposeRedeploy      TracePurpose = "redeploy"
	TracePurposeRollback      TracePurpose = "rollback"
	TracePurposePreview       TracePurpose = "preview"
)

// RetentionClass represents the retention class for traces
type RetentionClass string

const (
	RetentionClassShort RetentionClass = "short"
	RetentionClassLong  RetentionClass = "long"
)

// BuildStatus represents the status of a build
type BuildStatus string

const (
	BuildStatusQueued   BuildStatus = "queued"
	BuildStatusRunning  BuildStatus = "running"
	BuildStatusSuccess  BuildStatus = "success"
	BuildStatusFailed   BuildStatus = "failed"
	BuildStatusCanceled BuildStatus = "canceled"
)

// ReleaseStatus represents the status of a release
type ReleaseStatus string

const (
	ReleaseStatusDraft  ReleaseStatus = "draft"
	ReleaseStatusSealed ReleaseStatus = "sealed"
)

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	DeploymentStatusPending     DeploymentStatus = "pending"
	DeploymentStatusRendered    DeploymentStatus = "rendered"
	DeploymentStatusPushed      DeploymentStatus = "pushed"
	DeploymentStatusReconciling DeploymentStatus = "reconciling"
	DeploymentStatusHealthy     DeploymentStatus = "healthy"
	DeploymentStatusDegraded    DeploymentStatus = "degraded"
	DeploymentStatusFailed      DeploymentStatus = "failed"
	DeploymentStatusRolledBack  DeploymentStatus = "rolled_back"
)

// RenderJobStatus represents the status of a render job
type RenderJobStatus string

const (
	RenderJobStatusPending RenderJobStatus = "pending"
	RenderJobStatusRunning RenderJobStatus = "running"
	RenderJobStatusSuccess RenderJobStatus = "success"
	RenderJobStatusFailed  RenderJobStatus = "failed"
	RenderJobStatusCached  RenderJobStatus = "cached"
)
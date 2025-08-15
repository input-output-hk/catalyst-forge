package rbac

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PermissionKey string

type Effect string

const (
	Allow Effect = "allow"
	Deny  Effect = "deny"
)

type Condition struct {
	Name   string
	Params map[string]any
}

type RoleDef struct {
	ID          uuid.UUID
	Slug        string
	Name        string
	Description string
	Entries     []RoleEntry
	Version     int64
}

type RoleEntry struct {
	Effect       Effect
	Permission   PermissionKey
	ResourceType string
	Conditions   []Condition
}

type Binding struct {
	ID        uuid.UUID
	Subject   Subject
	RoleSlug  string
	ScopeType ScopeType
	ScopeID   string
	OrgID     *uuid.UUID
	CreatedAt time.Time
}

type ScopeType string

const (
	ScopeGlobal  ScopeType = "global"
	ScopeOrg     ScopeType = "org"
	ScopeProject ScopeType = "project"
	ScopeRes     ScopeType = "resource"
)

type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionDeny  Decision = "deny"
)

// Trace provides explanation details for a decision.
type Trace struct {
	CheckedAt  time.Time
	Subject    Subject
	Permission PermissionKey
	Resource   ResourceRef
	Steps      []TraceStep
	Outcome    Decision
}

type TraceStep struct {
	Scope      ScopeType
	ScopeID    string
	RoleSlug   string
	Entry      RoleEntry
	Conditions []ConditionEval
}

type ConditionEval struct {
	Name   string
	Passed bool
	Reason string
}

// ConditionEvaluator evaluates a named condition.
type ConditionEvaluator interface {
	Name() string
	Evaluate(ctx EvalContext, params map[string]any) (bool, string, error)
}

// EvalContext contains request-time context for condition evaluation.
type EvalContext struct {
	Now      time.Time
	Subject  Subject
	Resource ResourceRef
	// Optional: time until which step-up is valid for the subject.
	StepUpValidUntil time.Time
	// Optional: when the last step-up was performed; used with "within" windows.
	StepUpAt time.Time
	// Future: request metadata, IP, etc.
}

// Evaluator defines the RBAC decision engine.
type Evaluator interface {
	Check(ctx context.Context, subj Subject, action PermissionKey, res ResourceRef) (Decision, error)
	Explain(ctx context.Context, subj Subject, action PermissionKey, res ResourceRef) (Decision, Trace, error)
}

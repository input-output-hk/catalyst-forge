package rbac

import (
	"time"

	"github.com/google/uuid"
)

// SubjectRef identifies a subject for binding operations.
type SubjectRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// BindingCreateRequest is the payload for creating a binding.
type BindingCreateRequest struct {
	ID        string     `json:"id"`
	Subject   SubjectRef `json:"subject"`
	RoleSlug  string     `json:"role_slug"`
	ScopeType string     `json:"scope_type"`
	ScopeID   string     `json:"scope_id"`
	OrgID     string     `json:"org_id"`
}

// SubjectInput represents a subject in explain requests.
type SubjectInput struct {
	Type  string         `json:"type"`
	ID    string         `json:"id"`
	OrgID string         `json:"org_id"`
	Attrs map[string]any `json:"attrs"`
}

// ResourceInput represents a resource in explain requests.
type ResourceInput struct {
	Type   string         `json:"type"`
	ID     string         `json:"id"`
	OrgID  string         `json:"org_id"`
	Parent *ResourceInput `json:"parent,omitempty"`
	Attrs  map[string]any `json:"attrs"`
}

// ExplainRequest is the payload to request a decision explanation.
type ExplainRequest struct {
	Subject    SubjectInput  `json:"subject"`
	Permission string        `json:"permission"`
	Resource   ResourceInput `json:"resource"`
}

// ConditionsResponse lists registered condition names.
type ConditionsResponse struct {
	Conditions []string `json:"conditions"`
}

// ResourceTypesResponse lists canonical resource types.
type ResourceTypesResponse struct {
	Types []string `json:"types"`
}

// ResourceEndpointInfo describes how to list/browse a resource type via existing APIs.
type ResourceEndpointInfo struct {
	Method        string   `json:"method"`
	Path          string   `json:"path"`
	IDField       string   `json:"id_field"`
	LabelField    string   `json:"label_field"`
	QueryParams   []string `json:"query_params,omitempty"`
	ParentFilters []string `json:"parent_filters,omitempty"`
}

// ResourceTypeCatalogItem maps a canonical resource type to its browse endpoint info.
type ResourceTypeCatalogItem struct {
	Type     string               `json:"type"`
	Endpoint ResourceEndpointInfo `json:"endpoint"`
}

// ResourceTypeCatalogResponse returns the mapping for all supported resource types.
type ResourceTypeCatalogResponse struct {
	Types []ResourceTypeCatalogItem `json:"types"`
}

// PermissionsResponse lists registered permission keys.
type PermissionsResponse struct {
	Permissions []string `json:"permissions"`
}

// PermissionsCatalogResponse lists permission metadata.
type PermissionsCatalogResponse struct {
	Permissions []PermissionInfo `json:"permissions"`
}

type PermissionInfo struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
}

// Condition represents a condition attached to a role entry.
type Condition struct {
	Name   string         `json:"name"`
	Params map[string]any `json:"params"`
}

// RoleEntry represents a single permission grant/deny within a role.
type RoleEntry struct {
	Effect       string      `json:"effect"`
	Permission   string      `json:"permission"`
	ResourceType string      `json:"resource_type"`
	Conditions   []Condition `json:"conditions,omitempty"`
}

// Role is a flattened API DTO for roles.
type Role struct {
	ID          uuid.UUID   `json:"id"`
	Slug        string      `json:"slug"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Color       string      `json:"color"`
	Version     int64       `json:"version"`
	Entries     []RoleEntry `json:"entries,omitempty"`
}

// RolesListResponse lists role definitions.
type RolesListResponse struct {
	Roles []Role `json:"roles"`
}

// Binding is a flattened API DTO for bindings.
type Binding struct {
	ID          uuid.UUID  `json:"id"`
	SubjectType string     `json:"subject_type"`
	SubjectID   string     `json:"subject_id"`
	RoleSlug    string     `json:"role_slug"`
	ScopeType   string     `json:"scope_type"`
	ScopeID     string     `json:"scope_id"`
	OrgID       *uuid.UUID `json:"org_id"`
	CreatedAt   time.Time  `json:"created_at"`
}

// BindingsListResponse lists bindings.
type BindingsListResponse struct {
	Bindings []Binding `json:"bindings"`
}

// ExplainResponse is the response for an explanation request.
type ExplainResponse struct {
	Decision string `json:"decision"`
	Trace    any    `json:"trace"`
}

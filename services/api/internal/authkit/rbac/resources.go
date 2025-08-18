package rbac

import "sort"

// Canonical resource type identifiers. These must match values emitted by resolvers.
const (
	ResourceTypeOrg         = "org"
	ResourceTypeProject     = "project"
	ResourceTypeEnvironment = "environment"
	ResourceTypeRelease     = "release"
	ResourceTypeDeployment  = "deployment"
	ResourceTypeBuild       = "build"
	ResourceTypeArtifact    = "artifact"
	ResourceTypeRepository  = "repository"
)

// ResourceTypes returns the canonical list of resource types used by RBAC resolvers.
func ResourceTypes() []string {
	types := []string{
		ResourceTypeEnvironment,
		ResourceTypeRelease,
		ResourceTypeDeployment,
		ResourceTypeBuild,
		ResourceTypeArtifact,
		ResourceTypeRepository,
	}
	sort.Strings(types)
	return types
}

// ResourceEndpointInfo describes how to list/browse a resource type via existing APIs.
type ResourceEndpointInfo struct {
	Method        string
	Path          string
	IDField       string
	LabelField    string
	QueryParams   []string
	ParentFilters []string
}

// ResourceTypeCatalogItem maps a canonical resource type to its browse endpoint info.
type ResourceTypeCatalogItem struct {
	Type     string
	Endpoint ResourceEndpointInfo
}

// ResourceTypeCatalog returns the mapping for all supported instance resource types
// to their canonical list endpoints. This avoids duplicating list behavior and
// allows the frontend to reuse existing domain APIs.
func ResourceTypeCatalog() []ResourceTypeCatalogItem {
	items := []ResourceTypeCatalogItem{
		{Type: ResourceTypeEnvironment, Endpoint: ResourceEndpointInfo{Method: "GET", Path: "/api/v1/environments", IDField: "id", LabelField: "name"}},
		{Type: ResourceTypeRelease, Endpoint: ResourceEndpointInfo{Method: "GET", Path: "/api/v1/releases", IDField: "id", LabelField: "release_key", ParentFilters: []string{"project_id"}}},
		{Type: ResourceTypeDeployment, Endpoint: ResourceEndpointInfo{Method: "GET", Path: "/api/v1/deployments", IDField: "id", LabelField: "id", ParentFilters: []string{"project_id", "env_id", "release_id"}}},
		{Type: ResourceTypeBuild, Endpoint: ResourceEndpointInfo{Method: "GET", Path: "/api/v1/builds", IDField: "id", LabelField: "id", ParentFilters: []string{"project_id", "repo_id"}}},
		{Type: ResourceTypeArtifact, Endpoint: ResourceEndpointInfo{Method: "GET", Path: "/api/v1/artifacts", IDField: "id", LabelField: "name", ParentFilters: []string{"build_id"}}},
		{Type: ResourceTypeRepository, Endpoint: ResourceEndpointInfo{Method: "GET", Path: "/api/v1/repositories", IDField: "id", LabelField: "name", QueryParams: []string{"host", "org", "name"}}},
	}
	return items
}

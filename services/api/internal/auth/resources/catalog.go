package resources

import "sort"

type EndpointInfo struct {
	Method        string
	Path          string
	IDField       string
	LabelField    string
	QueryParams   []string
	ParentFilters []string
}

type TypeCatalogItem struct {
	Type     string
	Endpoint EndpointInfo
}

func Types() []string {
	types := []string{
		TypeEnvironment,
		TypeRelease,
		TypeDeployment,
		TypeBuild,
		TypeArtifact,
		TypeRepository,
	}
	sort.Strings(types)
	return types
}

func TypeCatalog() []TypeCatalogItem {
	return []TypeCatalogItem{
		{Type: TypeEnvironment, Endpoint: EndpointInfo{Method: "GET", Path: "/api/v1/environments", IDField: "id", LabelField: "name"}},
		{Type: TypeRelease, Endpoint: EndpointInfo{Method: "GET", Path: "/api/v1/releases", IDField: "id", LabelField: "release_key", ParentFilters: []string{"project_id"}}},
		{Type: TypeDeployment, Endpoint: EndpointInfo{Method: "GET", Path: "/api/v1/deployments", IDField: "id", LabelField: "id", ParentFilters: []string{"project_id", "env_id", "release_id"}}},
		{Type: TypeBuild, Endpoint: EndpointInfo{Method: "GET", Path: "/api/v1/builds", IDField: "id", LabelField: "id", ParentFilters: []string{"project_id", "repo_id"}}},
		{Type: TypeArtifact, Endpoint: EndpointInfo{Method: "GET", Path: "/api/v1/artifacts", IDField: "id", LabelField: "name", ParentFilters: []string{"build_id"}}},
		{Type: TypeRepository, Endpoint: EndpointInfo{Method: "GET", Path: "/api/v1/repositories", IDField: "id", LabelField: "name", QueryParams: []string{"host", "org", "name"}}},
	}
}

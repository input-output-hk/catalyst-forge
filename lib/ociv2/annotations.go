package ociv2

import "github.com/input-output-hk/catalyst-forge/lib/ociv2/utils"

// WithSource adds source repository and revision annotations
func (a Annotations) WithSource(repo, revision string) Annotations {
	utilsAnn := utils.Annotations(a)
	return Annotations(utilsAnn.WithSource(repo, revision))
}

// WithForgeKind adds Forge kind annotation
func (a Annotations) WithForgeKind(kind string) Annotations {
	utilsAnn := utils.Annotations(a)
	return Annotations(utilsAnn.WithForgeKind(kind))
}

// WithForgeProject adds Forge project annotation
func (a Annotations) WithForgeProject(project string) Annotations {
	utilsAnn := utils.Annotations(a)
	return Annotations(utilsAnn.WithForgeProject(project))
}

// WithForgeEnv adds Forge environment annotation
func (a Annotations) WithForgeEnv(env string) Annotations {
	utilsAnn := utils.Annotations(a)
	return Annotations(utilsAnn.WithForgeEnv(env))
}

// WithForgeRelease adds Forge release key annotation
func (a Annotations) WithForgeRelease(releaseKey string) Annotations {
	utilsAnn := utils.Annotations(a)
	return Annotations(utilsAnn.WithForgeRelease(releaseKey))
}

// WithTrace adds trace ID annotation for observability
func (a Annotations) WithTrace(traceID string) Annotations {
	utilsAnn := utils.Annotations(a)
	return Annotations(utilsAnn.WithTrace(traceID))
}

// WithBuildInfo adds build-related annotations
func (a Annotations) WithBuildInfo(buildID, buildNumber, buildURL string) Annotations {
	utilsAnn := utils.Annotations(a)
	return Annotations(utilsAnn.WithBuildInfo(buildID, buildNumber, buildURL))
}

// WithGitInfo adds git-related annotations
func (a Annotations) WithGitInfo(commit, branch, tag string, dirty bool) Annotations {
	utilsAnn := utils.Annotations(a)
	return Annotations(utilsAnn.WithGitInfo(commit, branch, tag, dirty))
}

// Merge combines annotations, with the provided annotations taking precedence
func (a Annotations) Merge(other Annotations) Annotations {
	utilsAnn := utils.Annotations(a)
	return Annotations(utilsAnn.Merge(utils.Annotations(other)))
}

// FilterForge returns only Forge-specific annotations
func (a Annotations) FilterForge() Annotations {
	utilsAnn := utils.Annotations(a)
	return Annotations(utilsAnn.FilterForge())
}

// FilterOCI returns only OCI standard annotations
func (a Annotations) FilterOCI() Annotations {
	utilsAnn := utils.Annotations(a)
	return Annotations(utilsAnn.FilterOCI())
}
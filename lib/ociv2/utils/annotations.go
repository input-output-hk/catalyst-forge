package utils

import (
	"maps"
	"time"
)

// Annotations is a helper type for OCI annotations (map[string]string)
type Annotations map[string]string

// Merge combines annotations, with the provided annotations taking precedence
func (a Annotations) Merge(other Annotations) Annotations {
	result := make(Annotations, len(a)+len(other))
	maps.Copy(result, a)
	maps.Copy(result, other)
	return result
}

// OCI standard annotation keys
const (
	// Standard OCI image annotations
	AnnSourceRepo     = "org.opencontainers.image.source"      // Source repository URL
	AnnSourceRev      = "org.opencontainers.image.revision"    // Source control revision
	AnnCreated        = "org.opencontainers.image.created"     // Creation timestamp (RFC 3339)
	AnnTitle          = "org.opencontainers.image.title"       // Human-readable title
	AnnDescription    = "org.opencontainers.image.description" // Human-readable description
	AnnAuthors        = "org.opencontainers.image.authors"     // Contact details of people/org
	AnnURL            = "org.opencontainers.image.url"         // URL to find more information
	AnnDocumentation  = "org.opencontainers.image.documentation" // URL to documentation
	AnnLicenses       = "org.opencontainers.image.licenses"    // License(s) under which contained software is distributed
	AnnVendor         = "org.opencontainers.image.vendor"      // Name of distributing entity
	AnnVersion        = "org.opencontainers.image.version"     // Version of the packaged software
	AnnBaseDigest     = "org.opencontainers.image.base.digest" // Digest of the base image
	AnnBaseName       = "org.opencontainers.image.base.name"   // Annotations of base image
)

// Forge-specific annotation keys
const (
	// Core Forge annotations
	AnnForgeKind    = "io.projectcatalyst.forge.kind"    // Artifact kind (release|rendered|sbom|...)
	AnnForgeProject = "io.projectcatalyst.forge.project" // Project name
	AnnForgeEnv     = "io.projectcatalyst.forge.env"     // Environment (dev|staging|prod)
	AnnForgeTrace   = "io.projectcatalyst.forge.trace"   // Trace ID for observability
	AnnForgeRelease = "io.projectcatalyst.forge.releaseKey" // Release key identifier
	
	// Build annotations
	AnnForgeBuildID     = "io.projectcatalyst.forge.build.id"        // Build ID
	AnnForgeBuildNumber = "io.projectcatalyst.forge.build.number"    // Build number
	AnnForgeBuildURL    = "io.projectcatalyst.forge.build.url"       // Build URL
	AnnForgeBuilder     = "io.projectcatalyst.forge.build.builder"   // Builder tool/version
	
	// Deployment annotations
	AnnForgeCluster    = "io.projectcatalyst.forge.cluster"      // Target cluster
	AnnForgeNamespace  = "io.projectcatalyst.forge.namespace"    // Target namespace
	AnnForgeDeployedBy = "io.projectcatalyst.forge.deployed.by"  // Who deployed
	AnnForgeDeployedAt = "io.projectcatalyst.forge.deployed.at"  // When deployed
	
	// Versioning annotations
	AnnForgeVersion     = "io.projectcatalyst.forge.version"         // Forge artifact version
	AnnForgeGitCommit   = "io.projectcatalyst.forge.git.commit"      // Git commit SHA
	AnnForgeGitBranch   = "io.projectcatalyst.forge.git.branch"      // Git branch
	AnnForgeGitTag      = "io.projectcatalyst.forge.git.tag"         // Git tag
	AnnForgeGitDirty    = "io.projectcatalyst.forge.git.dirty"       // Git working directory dirty
	
	// Signature annotations
	AnnForgeSigned     = "io.projectcatalyst.forge.signed"       // Whether artifact is signed
	AnnForgeSignedBy   = "io.projectcatalyst.forge.signed.by"    // Signer identity
	AnnForgeSignedAt   = "io.projectcatalyst.forge.signed.at"    // When signed
	AnnForgeSignature  = "io.projectcatalyst.forge.signature"    // Signature reference
)

// NewAnnotations creates a new Annotations map with standard values
func NewAnnotations() Annotations {
	return Annotations{
		AnnCreated: time.Now().UTC().Format(time.RFC3339),
	}
}

// WithSource adds source repository and revision annotations
func (a Annotations) WithSource(repo, revision string) Annotations {
	if repo != "" {
		a[AnnSourceRepo] = repo
	}
	if revision != "" {
		a[AnnSourceRev] = revision
	}
	return a
}

// WithForgeKind adds Forge kind annotation
func (a Annotations) WithForgeKind(kind string) Annotations {
	a[AnnForgeKind] = kind
	return a
}

// WithForgeProject adds Forge project annotation
func (a Annotations) WithForgeProject(project string) Annotations {
	a[AnnForgeProject] = project
	return a
}

// WithForgeEnv adds Forge environment annotation
func (a Annotations) WithForgeEnv(env string) Annotations {
	a[AnnForgeEnv] = env
	return a
}

// WithForgeRelease adds Forge release key annotation
func (a Annotations) WithForgeRelease(releaseKey string) Annotations {
	a[AnnForgeRelease] = releaseKey
	return a
}

// WithTrace adds trace ID annotation for observability
func (a Annotations) WithTrace(traceID string) Annotations {
	if traceID != "" {
		a[AnnForgeTrace] = traceID
	}
	return a
}

// WithBuildInfo adds build-related annotations
func (a Annotations) WithBuildInfo(buildID, buildNumber, buildURL string) Annotations {
	if buildID != "" {
		a[AnnForgeBuildID] = buildID
	}
	if buildNumber != "" {
		a[AnnForgeBuildNumber] = buildNumber
	}
	if buildURL != "" {
		a[AnnForgeBuildURL] = buildURL
	}
	return a
}

// WithGitInfo adds git-related annotations
func (a Annotations) WithGitInfo(commit, branch, tag string, dirty bool) Annotations {
	if commit != "" {
		a[AnnForgeGitCommit] = commit
	}
	if branch != "" {
		a[AnnForgeGitBranch] = branch
	}
	if tag != "" {
		a[AnnForgeGitTag] = tag
	}
	if dirty {
		a[AnnForgeGitDirty] = "true"
	}
	return a
}

// FilterForge returns only Forge-specific annotations
func (a Annotations) FilterForge() Annotations {
	result := make(Annotations)
	for k, v := range a {
		if len(k) > 24 && k[:24] == "io.projectcatalyst.forge" {
			result[k] = v
		}
	}
	return result
}

// FilterOCI returns only OCI standard annotations
func (a Annotations) FilterOCI() Annotations {
	result := make(Annotations)
	for k, v := range a {
		if len(k) > 18 && k[:18] == "org.opencontainers" {
			result[k] = v
		}
	}
	return result
}
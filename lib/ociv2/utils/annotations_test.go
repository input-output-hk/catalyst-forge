package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAnnotations(t *testing.T) {
	t.Parallel()
	
	ann := NewAnnotations()
	assert.NotNil(t, ann)
	assert.Len(t, ann, 1) // Should contain created timestamp
	assert.Contains(t, ann, AnnCreated)
	
	// Verify timestamp format is RFC3339
	createdTime, err := time.Parse(time.RFC3339, ann[AnnCreated])
	assert.NoError(t, err)
	assert.WithinDuration(t, time.Now(), createdTime, 5*time.Second)
}

func TestAnnotationsMerge(t *testing.T) {
	t.Parallel()
	
	base := Annotations{
		AnnTitle:       "Base Title",
		AnnDescription: "Base Description",
		AnnVersion:     "1.0.0",
	}
	
	override := Annotations{
		AnnDescription: "Override Description", // Should override
		AnnAuthors:     "John Doe",             // Should add
	}
	
	result := base.Merge(override)
	
	// Verify merge behavior
	assert.Equal(t, "Base Title", result[AnnTitle])        // Kept from base
	assert.Equal(t, "Override Description", result[AnnDescription]) // Overridden
	assert.Equal(t, "1.0.0", result[AnnVersion])          // Kept from base
	assert.Equal(t, "John Doe", result[AnnAuthors])       // Added from override
	
	// Verify original annotations are unchanged
	assert.Equal(t, "Base Description", base[AnnDescription])
	assert.NotContains(t, base, AnnAuthors)
}

func TestAnnotationBuilders(t *testing.T) {
	t.Parallel()
	
	t.Run("WithSource", func(t *testing.T) {
		ann := NewAnnotations().WithSource("https://github.com/org/repo", "abc123")
		
		assert.Equal(t, "https://github.com/org/repo", ann[AnnSourceRepo])
		assert.Equal(t, "abc123", ann[AnnSourceRev])
	})
	
	t.Run("WithTrace", func(t *testing.T) {
		ann := NewAnnotations().WithTrace("trace-123")
		
		assert.Equal(t, "trace-123", ann[AnnForgeTrace])
	})
	
	t.Run("WithTraceEmpty", func(t *testing.T) {
		ann := NewAnnotations().WithTrace("")
		
		// Empty trace should not be set
		assert.NotContains(t, ann, AnnForgeTrace)
	})
	
	t.Run("WithBuildInfo", func(t *testing.T) {
		ann := NewAnnotations().WithBuildInfo("build-123", "456", "https://build.example.com")
		
		assert.Equal(t, "build-123", ann[AnnForgeBuildID])
		assert.Equal(t, "456", ann[AnnForgeBuildNumber])
		assert.Equal(t, "https://build.example.com", ann[AnnForgeBuildURL])
	})
	
	t.Run("WithBuildInfoPartial", func(t *testing.T) {
		ann := NewAnnotations().WithBuildInfo("build-123", "", "")
		
		assert.Equal(t, "build-123", ann[AnnForgeBuildID])
		assert.NotContains(t, ann, AnnForgeBuildNumber)
		assert.NotContains(t, ann, AnnForgeBuildURL)
	})
	
	t.Run("WithGitInfo", func(t *testing.T) {
		ann := NewAnnotations().WithGitInfo("abc123", "main", "v1.0", true)
		
		assert.Equal(t, "abc123", ann[AnnForgeGitCommit])
		assert.Equal(t, "main", ann[AnnForgeGitBranch])
		assert.Equal(t, "v1.0", ann[AnnForgeGitTag])
		assert.Equal(t, "true", ann[AnnForgeGitDirty])
	})
	
	t.Run("WithGitInfoClean", func(t *testing.T) {
		ann := NewAnnotations().WithGitInfo("abc123", "main", "", false)
		
		assert.Equal(t, "abc123", ann[AnnForgeGitCommit])
		assert.Equal(t, "main", ann[AnnForgeGitBranch])
		assert.NotContains(t, ann, AnnForgeGitTag)
		assert.NotContains(t, ann, AnnForgeGitDirty)
	})
	
	t.Run("WithForgeRelease", func(t *testing.T) {
		ann := NewAnnotations().WithForgeRelease("release-123")
		
		assert.Equal(t, "release-123", ann[AnnForgeRelease])
	})
}

func TestForgeAnnotationBuilders(t *testing.T) {
	t.Parallel()
	
	t.Run("WithForgeKind", func(t *testing.T) {
		ann := NewAnnotations().WithForgeKind("release")
		
		assert.Equal(t, "release", ann[AnnForgeKind])
	})
	
	t.Run("WithForgeProject", func(t *testing.T) {
		ann := NewAnnotations().WithForgeProject("catalyst")
		
		assert.Equal(t, "catalyst", ann[AnnForgeProject])
	})
	
	t.Run("WithForgeEnv", func(t *testing.T) {
		ann := NewAnnotations().WithForgeEnv("production")
		
		assert.Equal(t, "production", ann[AnnForgeEnv])
	})
	
	t.Run("ManualAnnotations", func(t *testing.T) {
		// Test manual setting of annotations that don't have builder methods
		ann := NewAnnotations()
		ann[AnnTitle] = "Manual Title"
		ann[AnnDescription] = "Manual Description" 
		ann[AnnVersion] = "v1.2.3"
		ann[AnnAuthors] = "Test Author"
		
		assert.Equal(t, "Manual Title", ann[AnnTitle])
		assert.Equal(t, "Manual Description", ann[AnnDescription])
		assert.Equal(t, "v1.2.3", ann[AnnVersion])
		assert.Equal(t, "Test Author", ann[AnnAuthors])
	})
}

func TestSignatureAnnotations(t *testing.T) {
	t.Parallel()
	
	// Test manual setting of signature annotations
	ann := NewAnnotations()
	ann[AnnForgeSigned] = "true"
	ann[AnnForgeSignedBy] = "signer@example.com"
	ann[AnnForgeSignedAt] = time.Now().Format(time.RFC3339)
	
	assert.Equal(t, "true", ann[AnnForgeSigned])
	assert.Equal(t, "signer@example.com", ann[AnnForgeSignedBy])
	assert.NotEmpty(t, ann[AnnForgeSignedAt])
}

func TestDeploymentAnnotations(t *testing.T) {
	t.Parallel()
	
	// Test manual setting of deployment annotations
	ann := NewAnnotations()
	ann[AnnForgeCluster] = "prod-cluster"
	ann[AnnForgeNamespace] = "default"
	ann[AnnForgeDeployedBy] = "deploy-bot"
	ann[AnnForgeDeployedAt] = time.Now().Format(time.RFC3339)
	
	assert.Equal(t, "prod-cluster", ann[AnnForgeCluster])
	assert.Equal(t, "default", ann[AnnForgeNamespace])
	assert.Equal(t, "deploy-bot", ann[AnnForgeDeployedBy])
	assert.NotEmpty(t, ann[AnnForgeDeployedAt])
}

func TestAnnotationFilters(t *testing.T) {
	t.Parallel()
	
	// Create annotations with mixed OCI and Forge annotations
	ann := Annotations{
		// OCI standard annotations
		AnnTitle:       "Test Title",
		AnnDescription: "Test Description",
		AnnVersion:     "1.0.0",
		AnnAuthors:     "Test Author",
		
		// Forge annotations
		AnnForgeKind:    "release",
		AnnForgeProject: "catalyst",
		AnnForgeEnv:     "prod",
		
		// Custom annotations
		"custom.example.com/annotation": "custom value",
	}
	
	t.Run("FilterOCI", func(t *testing.T) {
		ociAnn := ann.FilterOCI()
		
		// Should contain OCI annotations
		assert.Contains(t, ociAnn, AnnTitle)
		assert.Contains(t, ociAnn, AnnDescription)
		assert.Contains(t, ociAnn, AnnVersion)
		assert.Contains(t, ociAnn, AnnAuthors)
		
		// Should not contain Forge or custom annotations
		assert.NotContains(t, ociAnn, AnnForgeKind)
		assert.NotContains(t, ociAnn, AnnForgeProject)
		assert.NotContains(t, ociAnn, "custom.example.com/annotation")
		
		// Should have 4 OCI annotations
		assert.Len(t, ociAnn, 4)
	})
	
	t.Run("FilterForge", func(t *testing.T) {
		forgeAnn := ann.FilterForge()
		
		// Should contain Forge annotations
		assert.Contains(t, forgeAnn, AnnForgeKind)
		assert.Contains(t, forgeAnn, AnnForgeProject)
		assert.Contains(t, forgeAnn, AnnForgeEnv)
		
		// Should not contain OCI or custom annotations
		assert.NotContains(t, forgeAnn, AnnTitle)
		assert.NotContains(t, forgeAnn, AnnDescription)
		assert.NotContains(t, forgeAnn, "custom.example.com/annotation")
		
		// Should have 3 Forge annotations
		assert.Len(t, forgeAnn, 3)
	})
	
	t.Run("CustomAnnotations", func(t *testing.T) {
		// Test that custom annotations are preserved
		assert.Contains(t, ann, "custom.example.com/annotation")
		assert.Equal(t, "custom value", ann["custom.example.com/annotation"])
	})
}

func TestAnnotationConstants(t *testing.T) {
	t.Parallel()
	
	// Test that all OCI annotation constants have correct prefixes
	ociAnnotations := []string{
		AnnSourceRepo, AnnSourceRev, AnnCreated, AnnTitle, AnnDescription,
		AnnAuthors, AnnURL, AnnDocumentation, AnnLicenses, AnnVendor,
		AnnVersion, AnnBaseDigest, AnnBaseName,
	}
	
	for _, ann := range ociAnnotations {
		assert.True(t, 
			strings.HasPrefix(ann, "org.opencontainers.image."),
			"OCI annotation %s should start with org.opencontainers.image.", ann)
	}
	
	// Test that all Forge annotation constants have correct prefixes
	forgeAnnotations := []string{
		AnnForgeKind, AnnForgeProject, AnnForgeEnv, AnnForgeTrace,
		AnnForgeRelease, AnnForgeBuildID, AnnForgeBuildNumber, AnnForgeBuildURL,
		AnnForgeBuilder, AnnForgeCluster, AnnForgeNamespace, AnnForgeDeployedBy,
		AnnForgeDeployedAt, AnnForgeVersion, AnnForgeGitCommit, AnnForgeGitBranch,
		AnnForgeGitTag, AnnForgeGitDirty, AnnForgeSigned, AnnForgeSignedBy,
		AnnForgeSignedAt, AnnForgeSignature,
	}
	
	for _, ann := range forgeAnnotations {
		assert.True(t,
			strings.HasPrefix(ann, "io.projectcatalyst.forge."),
			"Forge annotation %s should start with io.projectcatalyst.forge.", ann)
	}
}

func TestBuilderChaining(t *testing.T) {
	t.Parallel()
	
	// Test that builders can be chained together
	ann := NewAnnotations().
		WithSource("https://github.com/org/repo", "abc123").
		WithForgeKind("release").
		WithForgeProject("catalyst").
		WithForgeEnv("production").
		WithTrace("trace-123").
		WithBuildInfo("build-456", "789", "https://build.url")
	
	// Verify all values are set correctly
	assert.Equal(t, "https://github.com/org/repo", ann[AnnSourceRepo])
	assert.Equal(t, "abc123", ann[AnnSourceRev])
	assert.Equal(t, "release", ann[AnnForgeKind])
	assert.Equal(t, "catalyst", ann[AnnForgeProject])
	assert.Equal(t, "production", ann[AnnForgeEnv])
	assert.Equal(t, "trace-123", ann[AnnForgeTrace])
	assert.Equal(t, "build-456", ann[AnnForgeBuildID])
	assert.Equal(t, "789", ann[AnnForgeBuildNumber])
	assert.Equal(t, "https://build.url", ann[AnnForgeBuildURL])
	
	// Should also have the created timestamp
	assert.Contains(t, ann, AnnCreated)
}

func TestAnnotationsFromMap(t *testing.T) {
	t.Parallel()
	
	// Test that Annotations can be created from a regular map
	sourceMap := map[string]string{
		AnnTitle:        "Map Title",
		AnnDescription:  "Map Description",
		AnnForgeKind:    "rendered",
		"custom.key":    "custom value",
	}
	
	ann := Annotations(sourceMap)
	
	assert.Equal(t, "Map Title", ann[AnnTitle])
	assert.Equal(t, "Map Description", ann[AnnDescription])
	assert.Equal(t, "rendered", ann[AnnForgeKind])
	assert.Equal(t, "custom value", ann["custom.key"])
}

func TestAnnotationValidation(t *testing.T) {
	t.Parallel()
	
	// Test edge cases with empty and nil values
	t.Run("EmptyValues", func(t *testing.T) {
		ann := NewAnnotations().
			WithForgeKind("").
			WithForgeProject("").
			WithTrace("")
		
		// Empty values should still be set for some fields
		assert.Equal(t, "", ann[AnnForgeKind])
		assert.Equal(t, "", ann[AnnForgeProject])
		// Trace should not be set if empty
		assert.NotContains(t, ann, AnnForgeTrace)
	})
	
	t.Run("NilMerge", func(t *testing.T) {
		base := NewAnnotations().WithForgeKind("release")
		result := base.Merge(nil)
		
		// Should handle nil merge gracefully
		assert.Equal(t, "release", result[AnnForgeKind])
		assert.Contains(t, result, AnnCreated)
	})
}

func TestAllAnnotationConstants(t *testing.T) {
	t.Parallel()
	
	// Test a comprehensive set of annotations to ensure they're all defined
	ann := Annotations{
		// OCI annotations
		AnnSourceRepo:     "https://github.com/org/repo",
		AnnSourceRev:      "abc123",
		AnnCreated:        time.Now().Format(time.RFC3339),
		AnnTitle:          "Test Title",
		AnnDescription:    "Test Description",
		AnnAuthors:        "Test Author",
		AnnURL:            "https://example.com",
		AnnDocumentation:  "https://docs.example.com",
		AnnLicenses:       "MIT",
		AnnVendor:         "Example Corp",
		AnnVersion:        "1.0.0",
		AnnBaseDigest:     "sha256:abc123",
		AnnBaseName:       "base:latest",
		
		// Forge core annotations
		AnnForgeKind:      "release",
		AnnForgeProject:   "catalyst",
		AnnForgeEnv:       "production",
		AnnForgeTrace:     "trace-123",
		AnnForgeRelease:   "release-456",
		
		// Forge build annotations
		AnnForgeBuildID:     "build-123",
		AnnForgeBuildNumber: "456",
		AnnForgeBuildURL:    "https://build.example.com",
		AnnForgeBuilder:     "forge-builder:1.0",
		
		// Forge deployment annotations
		AnnForgeCluster:    "prod-cluster",
		AnnForgeNamespace:  "default",
		AnnForgeDeployedBy: "deploy-bot",
		AnnForgeDeployedAt: time.Now().Format(time.RFC3339),
		
		// Forge versioning annotations
		AnnForgeVersion:   "v1.2.3",
		AnnForgeGitCommit: "def456",
		AnnForgeGitBranch: "main",
		AnnForgeGitTag:    "v1.2.3",
		AnnForgeGitDirty:  "false",
		
		// Forge signature annotations
		AnnForgeSigned:    "true",
		AnnForgeSignedBy:  "signer@example.com",
		AnnForgeSignedAt:  time.Now().Format(time.RFC3339),
		AnnForgeSignature: "signature-ref",
	}
	
	// Verify all annotations are properly set
	assert.Len(t, ann, 35) // Should have all 35 defined annotations (13 OCI + 22 Forge)
	
	// Test filtering
	ociAnn := ann.FilterOCI()
	forgeAnn := ann.FilterForge()
	
	assert.Len(t, ociAnn, 13)  // All OCI annotations
	assert.Len(t, forgeAnn, 22) // All Forge annotations
}
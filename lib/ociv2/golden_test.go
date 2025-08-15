package ociv2

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Golden test files directory
const goldenDir = "testdata/golden"

// TestGoldenAnnotations tests annotation formats against golden files
func TestGoldenAnnotations(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name        string
		goldenFile  string
		annotations Annotations
	}{
		{
			name:       "basic_oci_annotations",
			goldenFile: "basic_oci_annotations.json",
			annotations: Annotations{
				AnnCreated:     "2025-01-01T00:00:00Z", // Fixed timestamp for golden tests
				AnnTitle:       "Test Application",
				AnnDescription: "A test application for golden tests",
				AnnVersion:     "1.0.0",
				AnnAuthors:     "Test Team <test@example.com>",
				AnnLicenses:    "MIT",
				AnnVendor:      "Example Corp",
				AnnURL:         "https://example.com",
				AnnDocumentation: "https://docs.example.com",
			},
		},
		{
			name:       "forge_annotations",
			goldenFile: "forge_annotations.json",
			annotations: func() Annotations {
				ann := Annotations{
					AnnCreated: "2025-01-01T00:00:00Z", // Fixed timestamp for golden tests
				}
				return ann.
					WithForgeKind("release").
					WithForgeProject("catalyst").
					WithForgeEnv("production").
					WithSource("https://github.com/input-output-hk/catalyst-forge", "abc123def456").
					WithBuildInfo("build-789", "123", "https://ci.example.com/build/789").
					WithGitInfo("abc123def456", "main", "v1.0.0", false)
			}(),
		},
		{
			name:       "comprehensive_annotations",
			goldenFile: "comprehensive_annotations.json",
			annotations: func() Annotations {
				ann := Annotations{
					AnnCreated: "2025-01-01T00:00:00Z", // Fixed timestamp for golden tests
				}
				ann[AnnTitle] = "Comprehensive Test"
				ann[AnnDescription] = "Contains all types of annotations"
				ann[AnnVersion] = "2.1.0"
				ann[AnnAuthors] = "Development Team"
				ann[AnnLicenses] = "Apache-2.0"
				ann[AnnVendor] = "Project Catalyst"
				ann[AnnURL] = "https://projectcatalyst.io"
				ann[AnnDocumentation] = "https://docs.projectcatalyst.io"
				
				// Add Forge annotations
				ann = ann.WithForgeKind("rendered").
					WithForgeProject("catalyst").
					WithForgeEnv("staging").
					WithTrace("trace-456789").
					WithForgeRelease("release-2024-001")
				
				// Add build info
				ann[AnnForgeBuildID] = "build-456"
				ann[AnnForgeBuildNumber] = "42"
				ann[AnnForgeBuildURL] = "https://ci.projectcatalyst.io/build/456"
				ann[AnnForgeBuilder] = "forge-builder:2.1.0"
				
				// Add deployment info
				ann[AnnForgeCluster] = "prod-cluster-eu"
				ann[AnnForgeNamespace] = "catalyst"
				ann[AnnForgeDeployedBy] = "deployment-bot"
				ann[AnnForgeDeployedAt] = "2024-01-15T10:30:00Z"
				
				// Add git info
				ann[AnnForgeGitCommit] = "def456789abc"
				ann[AnnForgeGitBranch] = "release/v2.1"
				ann[AnnForgeGitTag] = "v2.1.0"
				ann[AnnForgeGitDirty] = "false"
				
				// Add signature info
				ann[AnnForgeSigned] = "true"
				ann[AnnForgeSignedBy] = "release-bot@projectcatalyst.io"
				ann[AnnForgeSignedAt] = "2024-01-15T10:35:00Z"
				ann[AnnForgeSignature] = "sha256:signature123"
				
				return ann
			}(),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goldenPath := filepath.Join(goldenDir, tt.goldenFile)
			
			// Serialize current annotations
			actualJSON, err := json.MarshalIndent(tt.annotations, "", "  ")
			require.NoError(t, err)
			
			if *updateGolden {
				// Update golden file
				err := os.MkdirAll(goldenDir, 0755)
				require.NoError(t, err)
				
				err = os.WriteFile(goldenPath, actualJSON, 0644)
				require.NoError(t, err)
				
				t.Logf("Updated golden file: %s", goldenPath)
				return
			}
			
			// Read golden file
			expectedJSON, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				t.Fatalf("Golden file does not exist: %s. Run with -update-golden to create it.", goldenPath)
			}
			require.NoError(t, err)
			
			// Compare JSON content
			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

// TestGoldenMediaTypes tests media type constants against golden file
func TestGoldenMediaTypes(t *testing.T) {
	t.Parallel()
	
	mediaTypes := map[string]string{
		"ReleaseConfig":      MTReleaseConfig,
		"RenderedIndex":      MTRenderedIndex,
		"RenderedTarGz":      MTRenderedTarGz,
		"OCIEmptyJSON":       MTOCIEmptyJSON,
		"OCIImageManifest":   MTOCIImageManifest,
		"OCIImageIndex":      MTOCIImageIndex,
		"OCIArtifactManifest": MTOCIArtifactManifest,
	}
	
	goldenPath := filepath.Join(goldenDir, "media_types.json")
	
	// Serialize current media types
	actualJSON, err := json.MarshalIndent(mediaTypes, "", "  ")
	require.NoError(t, err)
	
	if *updateGolden {
		// Update golden file
		err := os.MkdirAll(goldenDir, 0755)
		require.NoError(t, err)
		
		err = os.WriteFile(goldenPath, actualJSON, 0644)
		require.NoError(t, err)
		
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}
	
	// Read golden file
	expectedJSON, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("Golden file does not exist: %s. Run with -update-golden to create it.", goldenPath)
	}
	require.NoError(t, err)
	
	// Compare JSON content
	assert.JSONEq(t, string(expectedJSON), string(actualJSON))
}

// TestGoldenAnnotationConstants tests annotation constants against golden file
func TestGoldenAnnotationConstants(t *testing.T) {
	t.Parallel()
	
	constants := map[string]string{
		// OCI standard annotations
		"AnnSourceRepo":     AnnSourceRepo,
		"AnnSourceRev":      AnnSourceRev,
		"AnnCreated":        AnnCreated,
		"AnnTitle":          AnnTitle,
		"AnnDescription":    AnnDescription,
		"AnnAuthors":        AnnAuthors,
		"AnnURL":            AnnURL,
		"AnnDocumentation":  AnnDocumentation,
		"AnnLicenses":       AnnLicenses,
		"AnnVendor":         AnnVendor,
		"AnnVersion":        AnnVersion,
		"AnnBaseDigest":     AnnBaseDigest,
		"AnnBaseName":       AnnBaseName,
		
		// Forge core annotations
		"AnnForgeKind":      AnnForgeKind,
		"AnnForgeProject":   AnnForgeProject,
		"AnnForgeEnv":       AnnForgeEnv,
		"AnnForgeTrace":     AnnForgeTrace,
		"AnnForgeRelease":   AnnForgeRelease,
		
		// Forge build annotations
		"AnnForgeBuildID":     AnnForgeBuildID,
		"AnnForgeBuildNumber": AnnForgeBuildNumber,
		"AnnForgeBuildURL":    AnnForgeBuildURL,
		"AnnForgeBuilder":     AnnForgeBuilder,
		
		// Forge deployment annotations
		"AnnForgeCluster":    AnnForgeCluster,
		"AnnForgeNamespace":  AnnForgeNamespace,
		"AnnForgeDeployedBy": AnnForgeDeployedBy,
		"AnnForgeDeployedAt": AnnForgeDeployedAt,
		
		// Forge versioning annotations
		"AnnForgeVersion":   AnnForgeVersion,
		"AnnForgeGitCommit": AnnForgeGitCommit,
		"AnnForgeGitBranch": AnnForgeGitBranch,
		"AnnForgeGitTag":    AnnForgeGitTag,
		"AnnForgeGitDirty":  AnnForgeGitDirty,
		
		// Forge signature annotations
		"AnnForgeSigned":    AnnForgeSigned,
		"AnnForgeSignedBy":  AnnForgeSignedBy,
		"AnnForgeSignedAt":  AnnForgeSignedAt,
		"AnnForgeSignature": AnnForgeSignature,
	}
	
	goldenPath := filepath.Join(goldenDir, "annotation_constants.json")
	
	// Serialize current constants
	actualJSON, err := json.MarshalIndent(constants, "", "  ")
	require.NoError(t, err)
	
	if *updateGolden {
		// Update golden file
		err := os.MkdirAll(goldenDir, 0755)
		require.NoError(t, err)
		
		err = os.WriteFile(goldenPath, actualJSON, 0644)
		require.NoError(t, err)
		
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}
	
	// Read golden file
	expectedJSON, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("Golden file does not exist: %s. Run with -update-golden to create it.", goldenPath)
	}
	require.NoError(t, err)
	
	// Compare JSON content
	assert.JSONEq(t, string(expectedJSON), string(actualJSON))
}

// TestGoldenErrorCategories tests error category constants
func TestGoldenErrorCategories(t *testing.T) {
	t.Parallel()
	
	categories := map[string]string{
		"Auth":       string(ErrorCategoryAuth),
		"Network":    string(ErrorCategoryNetwork),
		"Registry":   string(ErrorCategoryRegistry),
		"Validation": string(ErrorCategoryValidation),
		"Config":     string(ErrorCategoryConfig),
		"Cosign":     string(ErrorCategoryCosign),
		"Fallback":   string(ErrorCategoryFallback),
		"Unknown":    string(ErrorCategoryUnknown),
	}
	
	goldenPath := filepath.Join(goldenDir, "error_categories.json")
	
	// Serialize current categories
	actualJSON, err := json.MarshalIndent(categories, "", "  ")
	require.NoError(t, err)
	
	if *updateGolden {
		// Update golden file
		err := os.MkdirAll(goldenDir, 0755)
		require.NoError(t, err)
		
		err = os.WriteFile(goldenPath, actualJSON, 0644)
		require.NoError(t, err)
		
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}
	
	// Read golden file
	expectedJSON, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("Golden file does not exist: %s. Run with -update-golden to create it.", goldenPath)
	}
	require.NoError(t, err)
	
	// Compare JSON content
	assert.JSONEq(t, string(expectedJSON), string(actualJSON))
}

// Command line flag for updating golden files
var updateGolden = flag.Bool("update-golden", false, "update golden files")
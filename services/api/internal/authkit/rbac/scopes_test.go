package rbac

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildRef is a local alias to the shared helper for readability
func buildRef(orgID, projectID, envID, resType, resID string) ResourceRef {
	return BuildAncestryRef(orgID, projectID, envID, resType, resID)
}

func TestScopes_RegistryAndExtractors(t *testing.T) {
	t.Parallel()

	// IsKnownScope positive cases (drive from canonical registry)
	for _, s := range ListKnownScopes() {
		s := s
		t.Run("known/"+string(s), func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsKnownScope(s), "expected scope to be known: %s", s)
		})
	}

	// IsKnownScope negative
	t.Run("unknown/scope", func(t *testing.T) {
		t.Parallel()
		assert.False(t, IsKnownScope(ScopeType("bananas")), "unknown scope should be false")
	})

	// ValidateScopeNames with mix of known and unknown
	t.Run("validate/mixed", func(t *testing.T) {
		t.Parallel()
		known := ListKnownScopes()
		require.NotEmpty(t, known)
		err := ValidateScopeNames(known[0], ScopeType("zeta"), ScopeType("alpha"))
		require.Error(t, err, "expected error for unknown scopes")
		// The function sorts unknown names, so error string should list alpha before zeta
		assert.Contains(t, err.Error(), "alpha, zeta", "unknown scopes should be sorted in error message")
	})

	// GetScopeSpec and default extractors
	t.Run("spec/resource_extractor", func(t *testing.T) {
		t.Parallel()
		var matched []ScopeType
		for _, s := range ListKnownScopes() {
			spec, ok := GetScopeSpec(s)
			require.True(t, ok)
			id, okID := spec.DefaultExtractor(ResourceRef{Type: "deployment", ID: "res-123"})
			_, okEmpty := spec.DefaultExtractor(ResourceRef{Type: "deployment", ID: ""})
			if okID && id == "res-123" && !okEmpty {
				matched = append(matched, s)
			}
		}
		require.NotEmpty(t, matched, "expected at least one scope to act as a resource extractor")
	})

	t.Run("spec/global_extractor", func(t *testing.T) {
		t.Parallel()
		var found bool
		for _, s := range ListKnownScopes() {
			spec, ok := GetScopeSpec(s)
			require.True(t, ok)
			id, okID := spec.DefaultExtractor(ResourceRef{})
			if okID {
				assert.Equal(t, "", id, "global-like extractor should return empty ID for empty ref")
				found = true
				break
			}
		}
		require.True(t, found, "expected a global-like extractor to exist")
	})

	// parentTypeExtractor via registered defaults: environment and project
	t.Run("extractor/environment_and_project", func(t *testing.T) {
		t.Parallel()
		ref := buildRef("org-1", "proj-9", "env-7", "deployment", "dep-5")
		// Current chain: leaf -> environment -> project -> org
		envMatches := map[ScopeType]bool{}
		projMatches := map[ScopeType]bool{}
		for _, s := range ListKnownScopes() {
			spec, ok := GetScopeSpec(s)
			require.True(t, ok)
			id, okID := spec.DefaultExtractor(ref)
			if okID && id == "env-7" {
				envMatches[s] = true
			}
			if okID && id == "proj-9" {
				projMatches[s] = true
			}
		}
		require.NotEmpty(t, envMatches, "expected some extractor to locate environment ID")
		require.NotEmpty(t, projMatches, "expected some extractor to locate project ID")

		// Ensure same extractors fail when the corresponding ancestor is missing
		refNoEnv := buildRef("org-1", "proj-9", "", "deployment", "dep-5")
		for s := range envMatches {
			spec, _ := GetScopeSpec(s)
			_, okID := spec.DefaultExtractor(refNoEnv)
			assert.False(t, okID, "env extractor should fail when environment absent: %s", s)
		}
		refNoProj := buildRef("org-1", "", "env-7", "deployment", "dep-5")
		for s := range projMatches {
			spec, _ := GetScopeSpec(s)
			_, okID := spec.DefaultExtractor(refNoProj)
			assert.False(t, okID, "project extractor should fail when project absent: %s", s)
		}
	})

	// parentTypeExtractor not found for parent-less ref: only resource/global-like may match
	t.Run("extractor/parent_not_found", func(t *testing.T) {
		t.Parallel()
		refBare := buildRef("", "", "", "deployment", "dep-5")
		// Identify any extractors that match empty ref (treated as global-like)
		globalLike := map[ScopeType]bool{}
		for _, s := range ListKnownScopes() {
			spec, _ := GetScopeSpec(s)
			_, okID := spec.DefaultExtractor(ResourceRef{})
			if okID {
				globalLike[s] = true
			}
		}
		// Identify resource-like extractors which match the leaf's own ID when present
		resourceLike := map[ScopeType]bool{}
		for _, s := range ListKnownScopes() {
			spec, _ := GetScopeSpec(s)
			idBare, okBare := spec.DefaultExtractor(refBare)
			_, okEmpty := spec.DefaultExtractor(ResourceRef{Type: "deployment", ID: ""})
			if okBare && idBare == "dep-5" && !okEmpty {
				resourceLike[s] = true
			}
		}
		for _, s := range ListKnownScopes() {
			if globalLike[s] || resourceLike[s] {
				continue
			}
			spec, _ := GetScopeSpec(s)
			_, okID := spec.DefaultExtractor(refBare)
			assert.False(t, okID, "parent-based extractor should not match on bare leaf: %s", s)
		}
	})

	// orgExtractor from OrgID field and from ancestry
	t.Run("extractor/org_field_and_ancestry", func(t *testing.T) {
		t.Parallel()
		// Case 1: OrgID on ResourceRef
		orgUUID := uuid.New()
		leaf := ResourceRef{Type: "artifact", ID: "a1", OrgID: &orgUUID}
		var orgScopes []ScopeType
		for _, s := range ListKnownScopes() {
			spec, ok := GetScopeSpec(s)
			require.True(t, ok)
			id, okID := spec.DefaultExtractor(leaf)
			if okID && id == orgUUID.String() {
				orgScopes = append(orgScopes, s)
			}
		}
		require.NotEmpty(t, orgScopes, "expected some extractor to find OrgID on resource")

		// Case 2: Ancestry contains org node
		ref := buildRef("org-xyz", "proj-1", "env-2", "artifact", "a2")
		var foundAncestry bool
		for _, s := range orgScopes {
			spec, _ := GetScopeSpec(s)
			id2, ok2 := spec.DefaultExtractor(ref)
			if ok2 && id2 == "org-xyz" {
				foundAncestry = true
				break
			}
		}
		assert.True(t, foundAncestry, "org extractor should find org in ancestry")

		// Case 3: No org present
		refNoOrg := buildRef("", "proj-1", "env-2", "artifact", "a3")
		for _, s := range orgScopes {
			spec, _ := GetScopeSpec(s)
			_, ok3 := spec.DefaultExtractor(refNoOrg)
			assert.False(t, ok3, "org extractor should return false when org absent: %s", s)
		}
	})

	// ListKnownScopes returns scopes ordered by specificity (non-decreasing Specificity)
	t.Run("list_known_scopes_ordering", func(t *testing.T) {
		t.Parallel()
		got := ListKnownScopes()
		require.NotEmpty(t, got)
		prev := -1
		for i, s := range got {
			spec, ok := GetScopeSpec(s)
			require.True(t, ok)
			if i == 0 {
				prev = spec.Specificity
				continue
			}
			assert.GreaterOrEqual(t, spec.Specificity, prev, "scopes should be ordered by non-decreasing specificity")
			prev = spec.Specificity
		}
	})
}

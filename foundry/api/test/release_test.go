package test

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/foundry/api/client/releases"
)

func TestReleaseAPI(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	projectName := fmt.Sprintf("test-project-%d", time.Now().Unix())

	t.Run("CreateRelease", func(t *testing.T) {
		bundleStr := base64.StdEncoding.EncodeToString([]byte("test bundle data"))

		release := &releases.Release{
			SourceRepo:   "github.com/example/repo",
			SourceCommit: "abcdef123456",
			SourceBranch: "feature",
			Project:      projectName,
			ProjectPath:  "services/api",
			Bundle:       bundleStr,
		}

		createdRelease, err := c.Releases().Create(ctx, release, false)
		require.NoError(t, err)

		fmt.Printf("Created release ID: %+v\n", createdRelease)

		assert.NotEmpty(t, createdRelease.ID)
		assert.Equal(t, projectName, createdRelease.Project)
		assert.Equal(t, "github.com/example/repo", createdRelease.SourceRepo)
		assert.Equal(t, "abcdef123456", createdRelease.SourceCommit)
		assert.Equal(t, "feature", createdRelease.SourceBranch)
		assert.Equal(t, "services/api", createdRelease.ProjectPath)
		assert.Equal(t, bundleStr, createdRelease.Bundle)
		assert.NotZero(t, createdRelease.Created)

		t.Logf("Created release with ID: %s", createdRelease.ID)

		t.Run("GetRelease", func(t *testing.T) {
			fetchedRelease, err := c.Releases().Get(ctx, createdRelease.ID)
			require.NoError(t, err)

			assert.Equal(t, createdRelease.ID, fetchedRelease.ID)
			assert.Equal(t, createdRelease.Project, fetchedRelease.Project)
			assert.Equal(t, createdRelease.SourceRepo, fetchedRelease.SourceRepo)
			assert.Equal(t, createdRelease.SourceCommit, fetchedRelease.SourceCommit)
			assert.Equal(t, createdRelease.SourceBranch, fetchedRelease.SourceBranch)
			assert.Equal(t, createdRelease.ProjectPath, fetchedRelease.ProjectPath)
			assert.Equal(t, createdRelease.Bundle, fetchedRelease.Bundle)
		})

		t.Run("UpdateRelease", func(t *testing.T) {
			updatedRelease := *createdRelease
			updatedRelease.SourceCommit = "updated-commit-hash"

			result, err := c.Releases().Update(ctx, &updatedRelease)
			require.NoError(t, err)

			assert.Equal(t, "updated-commit-hash", result.SourceCommit)
			assert.Equal(t, createdRelease.ID, result.ID)

			fetchedRelease, err := c.Releases().Get(ctx, createdRelease.ID)
			require.NoError(t, err)
			assert.Equal(t, "updated-commit-hash", fetchedRelease.SourceCommit)
		})

		t.Run("CreateSecondRelease", func(t *testing.T) {
			release2 := &releases.Release{
				SourceRepo:   "github.com/example/repo",
				SourceCommit: "second-commit",
				Project:      projectName,
				ProjectPath:  "services/api",
				Bundle:       bundleStr,
			}

			createdRelease2, err := c.Releases().Create(ctx, release2, false)
			require.NoError(t, err)

			assert.NotEqual(t, createdRelease.ID, createdRelease2.ID)
			assert.Contains(t, createdRelease2.ID, projectName)

			t.Run("ListReleases", func(t *testing.T) {
				releases, err := c.Releases().List(ctx, projectName)
				require.NoError(t, err)

				assert.GreaterOrEqual(t, len(releases), 2)

				found1, found2 := false, false
				for _, r := range releases {
					if r.ID == createdRelease.ID {
						found1 = true
					}
					if r.ID == createdRelease2.ID {
						found2 = true
					}
				}

				assert.True(t, found1, "First release not found in list")
				assert.True(t, found2, "Second release not found in list")
			})
		})
	})
}

func TestReleaseWithDefaultBranch(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	projectName := fmt.Sprintf("test-project-default-branch-%d", time.Now().Unix())
	bundleStr := base64.StdEncoding.EncodeToString([]byte("test bundle data"))

	defaultBranchRelease := &releases.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "main-commit",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	// Create first release
	release1, err := c.Releases().Create(ctx, defaultBranchRelease, false)
	require.NoError(t, err)
	require.NotEmpty(t, release1.ID)

	// Create second release with same project
	release2, err := c.Releases().Create(ctx, defaultBranchRelease, false)
	require.NoError(t, err)
	require.NotEmpty(t, release2.ID)

	// Verify both releases have different IDs
	assert.NotEqual(t, release1.ID, release2.ID)

	// Verify both releases contain the project name
	assert.Contains(t, release1.ID, projectName)
	assert.Contains(t, release2.ID, projectName)

	// Verify the ID format follows the expected pattern
	idPattern := regexp.MustCompile(fmt.Sprintf(`^%s-\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`, projectName))
	assert.True(t, idPattern.MatchString(release1.ID), "Release ID format is incorrect")
	assert.True(t, idPattern.MatchString(release2.ID), "Release ID format is incorrect")
}

func TestReleaseWithBranch(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	projectName := fmt.Sprintf("test-project-branch-%d", time.Now().Unix())
	bundleStr := base64.StdEncoding.EncodeToString([]byte("test bundle data"))

	branchRelease := &releases.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "branch-commit",
		SourceBranch: "feature-branch",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	// Create first release
	branchRelease1, err := c.Releases().Create(ctx, branchRelease, false)
	require.NoError(t, err)
	require.NotEmpty(t, branchRelease1.ID)

	// Create second release with same project
	branchRelease2, err := c.Releases().Create(ctx, branchRelease, false)
	require.NoError(t, err)
	require.NotEmpty(t, branchRelease2.ID)

	// Verify both releases have different IDs
	assert.NotEqual(t, branchRelease1.ID, branchRelease2.ID)

	// Verify both releases contain the project name
	assert.Contains(t, branchRelease1.ID, projectName)
	assert.Contains(t, branchRelease2.ID, projectName)

	// Verify the ID format follows the expected pattern
	idPattern := regexp.MustCompile(fmt.Sprintf(`^%s-\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`, projectName))
	assert.True(t, idPattern.MatchString(branchRelease1.ID), "Release ID format is incorrect")
	assert.True(t, idPattern.MatchString(branchRelease2.ID), "Release ID format is incorrect")
}

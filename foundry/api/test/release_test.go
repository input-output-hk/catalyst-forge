package test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

func TestReleaseAPI(t *testing.T) {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	c := client.NewClient(apiURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	projectName := fmt.Sprintf("test-project-%d", time.Now().Unix())

	t.Run("CreateRelease", func(t *testing.T) {
		bundleStr := base64.StdEncoding.EncodeToString([]byte("test bundle data"))

		release := &client.Release{
			SourceRepo:   "github.com/example/repo",
			SourceCommit: "abcdef123456",
			SourceBranch: "feature",
			Project:      projectName,
			ProjectPath:  "services/api",
			Bundle:       bundleStr,
		}

		createdRelease, err := c.CreateRelease(ctx, release, false)
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
			fetchedRelease, err := c.GetRelease(ctx, createdRelease.ID)
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

			result, err := c.UpdateRelease(ctx, &updatedRelease)
			require.NoError(t, err)

			assert.Equal(t, "updated-commit-hash", result.SourceCommit)
			assert.Equal(t, createdRelease.ID, result.ID)

			fetchedRelease, err := c.GetRelease(ctx, createdRelease.ID)
			require.NoError(t, err)
			assert.Equal(t, "updated-commit-hash", fetchedRelease.SourceCommit)
		})

		t.Run("CreateSecondRelease", func(t *testing.T) {
			release2 := &client.Release{
				SourceRepo:   "github.com/example/repo",
				SourceCommit: "second-commit",
				Project:      projectName,
				ProjectPath:  "services/api",
				Bundle:       bundleStr,
			}

			createdRelease2, err := c.CreateRelease(ctx, release2, false)
			require.NoError(t, err)

			assert.NotEqual(t, createdRelease.ID, createdRelease2.ID)
			assert.Contains(t, createdRelease2.ID, projectName)

			t.Run("ListReleases", func(t *testing.T) {
				releases, err := c.ListReleases(ctx, projectName)
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

				assert.True(t, found1, "First created release not found in list")
				assert.True(t, found2, "Second created release not found in list")
			})
		})
	})
}

func TestReleaseWithDefaultBranch(t *testing.T) {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	c := client.NewClient(apiURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	projectName := fmt.Sprintf("test-default-branch-%d", time.Now().Unix())

	bundleStr := base64.StdEncoding.EncodeToString([]byte("test bundle data"))
	defaultBranchRelease := &client.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "main-commit",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	release1, err := c.CreateRelease(ctx, defaultBranchRelease, false)
	require.NoError(t, err)

	defaultBranchRelease.SourceCommit = "main-commit-2"
	release2, err := c.CreateRelease(ctx, defaultBranchRelease, false)
	require.NoError(t, err)

	defaultBranchPattern := regexp.MustCompile(fmt.Sprintf(`^%s-(\d+)$`, regexp.QuoteMeta(projectName)))

	assert.True(t, defaultBranchPattern.MatchString(release1.ID),
		"Release ID '%s' doesn't match default branch pattern: {project-name}-{counter}", release1.ID)
	assert.True(t, defaultBranchPattern.MatchString(release2.ID),
		"Release ID '%s' doesn't match default branch pattern: {project-name}-{counter}", release2.ID)

	idNumber1 := release1.ID[len(projectName)+1:]
	idNumber2 := release2.ID[len(projectName)+1:]
	assert.Greater(t, idNumber2, idNumber1, "Second release ID should be greater than first")

	branchRelease := &client.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "feature-commit",
		SourceBranch: "feature",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	branchRelease1, err := c.CreateRelease(ctx, branchRelease, false)
	require.NoError(t, err)

	branchRelease.SourceCommit = "feature-commit-2"
	branchRelease2, err := c.CreateRelease(ctx, branchRelease, false)
	require.NoError(t, err)

	assert.Contains(t, branchRelease1.ID, "-feature-", "Branch release ID should contain branch name")
	assert.Contains(t, branchRelease2.ID, "-feature-", "Branch release ID should contain branch name")

	branchPattern := regexp.MustCompile(fmt.Sprintf(`^%s-feature-(\d+)$`, regexp.QuoteMeta(projectName)))
	branchIdParts1 := branchPattern.FindStringSubmatch(branchRelease1.ID)
	branchIdParts2 := branchPattern.FindStringSubmatch(branchRelease2.ID)

	require.Len(t, branchIdParts1, 2, "Could not parse branch release ID correctly")
	require.Len(t, branchIdParts2, 2, "Could not parse branch release ID correctly")

	branchIdNumber1 := branchIdParts1[1]
	branchIdNumber2 := branchIdParts2[1]

	assert.Greater(t, branchIdNumber2, branchIdNumber1, "Second branch release ID should be greater than first")
}

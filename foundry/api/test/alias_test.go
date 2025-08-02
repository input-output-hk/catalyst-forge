package test

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/foundry/api/client/releases"
)

func TestAliasAPI(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	projectName := fmt.Sprintf("test-project-alias-%d", time.Now().Unix())

	bundleStr := base64.StdEncoding.EncodeToString([]byte("sample code for alias testing"))
	release := &releases.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "abcdef123456",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	createdRelease, err := c.Releases().Create(ctx, release, false)
	require.NoError(t, err)
	require.NotEmpty(t, createdRelease.ID)

	release2 := &releases.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "xyz789",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	createdRelease2, err := c.Releases().Create(ctx, release2, false)
	require.NoError(t, err)
	require.NotEmpty(t, createdRelease2.ID)

	t.Run("CreateAlias", func(t *testing.T) {
		aliasName := fmt.Sprintf("%s-latest", projectName)

		err := c.Aliases().Create(ctx, aliasName, createdRelease.ID)
		require.NoError(t, err)

		t.Run("GetReleaseByAlias", func(t *testing.T) {
			fetchedByAlias, err := c.Releases().GetByAlias(ctx, aliasName)
			require.NoError(t, err)

			assert.Equal(t, createdRelease.ID, fetchedByAlias.ID)
			assert.Equal(t, createdRelease.SourceCommit, fetchedByAlias.SourceCommit)
		})

		t.Run("ListAliases", func(t *testing.T) {
			aliases, err := c.Aliases().List(ctx, createdRelease.ID)
			require.NoError(t, err)

			foundAlias := false
			for _, a := range aliases {
				if a.Name == aliasName {
					foundAlias = true
					assert.Equal(t, createdRelease.ID, a.ReleaseID)
				}
			}
			assert.True(t, foundAlias, "Created alias not found in list")
		})

		t.Run("ReassignAlias", func(t *testing.T) {
			err := c.Aliases().Create(ctx, aliasName, createdRelease2.ID)
			require.NoError(t, err)

			fetchedByAlias, err := c.Releases().GetByAlias(ctx, aliasName)
			require.NoError(t, err)
			assert.Equal(t, createdRelease2.ID, fetchedByAlias.ID)
			assert.Equal(t, "xyz789", fetchedByAlias.SourceCommit)
		})

		t.Run("DeleteAlias", func(t *testing.T) {
			err := c.Aliases().Delete(ctx, aliasName)
			require.NoError(t, err)

			_, err = c.Releases().GetByAlias(ctx, aliasName)
			assert.Error(t, err, "Expected error when getting deleted alias")
		})
	})

	t.Run("MultipleAliases", func(t *testing.T) {
		alias1 := fmt.Sprintf("%s-prod", projectName)
		alias2 := fmt.Sprintf("%s-staging", projectName)

		err := c.Aliases().Create(ctx, alias1, createdRelease.ID)
		require.NoError(t, err)

		err = c.Aliases().Create(ctx, alias2, createdRelease.ID)
		require.NoError(t, err)

		aliases, err := c.Aliases().List(ctx, createdRelease.ID)
		require.NoError(t, err)

		found1, found2 := false, false
		for _, a := range aliases {
			if a.Name == alias1 {
				found1 = true
			}
			if a.Name == alias2 {
				found2 = true
			}
		}

		assert.True(t, found1, "First alias not found in list")
		assert.True(t, found2, "Second alias not found in list")

		err = c.Aliases().Delete(ctx, alias1)
		require.NoError(t, err)

		err = c.Aliases().Delete(ctx, alias2)
		require.NoError(t, err)
	})
}

package providers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v66/github"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	gh "github.com/input-output-hk/catalyst-forge/lib/providers/github"
	gm "github.com/input-output-hk/catalyst-forge/lib/providers/github/mocks"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sg "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubReleaserRelease(t *testing.T) {
	newProject := func(name, repoOwner, repoName, tag string, platforms []string) project.Project {
		return project.Project{
			Name: name,
			Blueprint: sb.Blueprint{
				Global: &sg.Global{
					Repo: &sg.Repo{
						Name: fmt.Sprintf("%s/%s", repoOwner, repoName),
					},
				},
				Project: &sp.Project{
					Ci: &sp.CI{
						Targets: map[string]sp.Target{
							"test": {
								Platforms: platforms,
							},
						},
					},
				},
			},
			Tag: &project.ProjectTag{
				Full: tag,
			},
		}
	}

	newRelease := func() sp.Release {
		return sp.Release{
			Target: "test",
		}
	}

	newAsset := func(name string) *github.ReleaseAsset {
		return &github.ReleaseAsset{
			Name: &name,
		}
	}

	workdir := "/tmp/catalyst-forge-123456"
	tests := []struct {
		name       string
		project    project.Project
		release    sp.Release
		ghRelease  github.RepositoryRelease
		config     GithubReleaserConfig
		files      map[string]string
		firing     bool
		force      bool
		runFail    bool
		createFail bool
		uploadFail bool
		validate   func(*testing.T, fs.Filesystem, map[string][]byte, bool, error)
	}{
		{
			name: "full",
			project: newProject(
				"project",
				"owner",
				"repo",
				"tag",
				[]string{"linux/amd64"},
			),
			release:   newRelease(),
			ghRelease: github.RepositoryRelease{},
			config: GithubReleaserConfig{
				Prefix: "project",
				Name:   "project/v1.0.0",
			},
			files: map[string]string{
				"linux/amd64/test": "test",
			},
			firing: true,
			validate: func(t *testing.T, fs fs.Filesystem, uploads map[string][]byte, created bool, err error) {
				assert.NoError(t, err)

				filename := "project-linux-amd64.tar.gz"
				exists, err := fs.Exists(filepath.Join(workdir, filename))
				require.NoError(t, err)
				assert.True(t, exists)

				assert.True(t, created)

				data, ok := uploads[filename]
				assert.True(t, ok)

				files := make(map[string]string)

				gzr, err := gzip.NewReader(bytes.NewBuffer(data))
				require.NoError(t, err)

				tr := tar.NewReader(gzr)
				for {
					header, err := tr.Next()
					if err == io.EOF {
						break
					}
					require.NoError(t, err)

					buf := make([]byte, header.Size)
					tr.Read(buf)
					files[header.Name] = string(buf)
				}

				contents, ok := files["test"]
				assert.True(t, ok)
				assert.Equal(t, "test", contents)
			},
		},
		{
			name: "multiple platforms",
			project: newProject(
				"project",
				"owner",
				"repo",
				"tag",
				[]string{"linux/amd64", "darwin/amd64"},
			),
			release:   newRelease(),
			ghRelease: github.RepositoryRelease{},
			config: GithubReleaserConfig{
				Prefix: "project",
				Name:   "project/v1.0.0",
			},
			files: map[string]string{
				"linux/amd64/test":  "test",
				"darwin/amd64/test": "test",
			},
			firing: true,
			validate: func(t *testing.T, fs fs.Filesystem, uploads map[string][]byte, created bool, err error) {
				assert.NoError(t, err)

				for _, platform := range []string{"linux-amd64", "darwin-amd64"} {
					filename := fmt.Sprintf("project-%s.tar.gz", platform)
					exists, err := fs.Exists(filepath.Join(workdir, filename))
					require.NoError(t, err)
					assert.True(t, exists)

					data, ok := uploads[filename]
					assert.True(t, ok)

					files := make(map[string]string)

					gzr, err := gzip.NewReader(bytes.NewBuffer(data))
					require.NoError(t, err)

					tr := tar.NewReader(gzr)
					for {
						header, err := tr.Next()
						if err == io.EOF {
							break
						}
						require.NoError(t, err)

						buf := make([]byte, header.Size)
						tr.Read(buf)
						files[header.Name] = string(buf)
					}

					contents, ok := files["test"]
					assert.True(t, ok)
					assert.Equal(t, "test", contents)
				}
			},
		},
		{
			name: "not firing",
			project: newProject(
				"project",
				"owner",
				"repo",
				"tag",
				[]string{"linux/amd64"},
			),
			release:   newRelease(),
			ghRelease: github.RepositoryRelease{},
			config:    GithubReleaserConfig{},
			files: map[string]string{
				"linux/amd64/test": "test",
			},
			firing: false,
			validate: func(t *testing.T, fs fs.Filesystem, uploads map[string][]byte, created bool, err error) {
				assert.NoError(t, err)
				assert.False(t, created)
			},
		},
		{
			name: "artifact not found",
			project: newProject(
				"project",
				"owner",
				"repo",
				"tag",
				[]string{"linux/amd64"},
			),
			release:   newRelease(),
			ghRelease: github.RepositoryRelease{},
			config:    GithubReleaserConfig{},
			files:     map[string]string{},
			firing:    true,
			validate: func(t *testing.T, fs fs.Filesystem, uploads map[string][]byte, created bool, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to validate artifacts")
			},
		},
		{
			name: "release already exists",
			project: newProject(
				"project",
				"owner",
				"repo",
				"tag",
				[]string{"linux/amd64"},
			),
			release: newRelease(),
			ghRelease: github.RepositoryRelease{
				ID: github.Int64(123456),
			},
			config: GithubReleaserConfig{
				Prefix: "project",
				Name:   "name",
			},
			files: map[string]string{
				"linux/amd64/test": "test",
			},
			firing: true,
			validate: func(t *testing.T, fs fs.Filesystem, uploads map[string][]byte, created bool, err error) {
				assert.NoError(t, err)
				assert.False(t, created)
			},
		},
		{
			name: "create release fail",
			project: newProject(
				"project",
				"owner",
				"repo",
				"tag",
				[]string{"linux/amd64"},
			),
			release:   newRelease(),
			ghRelease: github.RepositoryRelease{},
			config: GithubReleaserConfig{
				Prefix: "project",
				Name:   "project/v1.0.0",
			},
			files: map[string]string{
				"linux/amd64/test": "test",
			},
			firing:     true,
			createFail: true,
			validate: func(t *testing.T, fs fs.Filesystem, uploads map[string][]byte, created bool, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to create release")
			},
		},
		{
			name: "upload asset fail",
			project: newProject(
				"project",
				"owner",
				"repo",
				"tag",
				[]string{"linux/amd64"},
			),
			release:   newRelease(),
			ghRelease: github.RepositoryRelease{},
			config: GithubReleaserConfig{
				Prefix: "project",
				Name:   "project/v1.0.0",
			},
			files: map[string]string{
				"linux/amd64/test": "test",
			},
			firing:     true,
			uploadFail: true,
			validate: func(t *testing.T, fs fs.Filesystem, uploads map[string][]byte, created bool, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to upload asset")
			},
		},
		{
			name: "asset already exists",
			project: newProject(
				"project",
				"owner",
				"repo",
				"tag",
				[]string{"linux/amd64"},
			),
			release: newRelease(),
			ghRelease: github.RepositoryRelease{
				ID: github.Int64(123456),
				Assets: []*github.ReleaseAsset{
					newAsset("project-linux-amd64.tar.gz"),
				},
			},
			config: GithubReleaserConfig{
				Prefix: "project",
				Name:   "project/v1.0.0",
			},
			files: map[string]string{
				"linux/amd64/test": "test",
			},
			firing: true,
			validate: func(t *testing.T, fs fs.Filesystem, uploads map[string][]byte, created bool, err error) {
				assert.NoError(t, err)
				assert.NotContains(t, uploads, "/repos/owner/repo/releases/123456/assets?name=project-linux-amd64.tar.gz")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			files := make(map[string]string)
			for k := range tt.files {
				nk := filepath.Join(workdir, k)
				files[nk] = tt.files[k]
			}
			testutils.SetupFS(t, fs, files)

			var releaseCreated bool
			uploads := make(map[string][]byte)
			client := gm.GithubClientMock{
				GetReleaseByTagFunc: func(tag string) (*github.RepositoryRelease, error) {
					if tt.ghRelease.ID == nil {
						return nil, gh.ErrReleaseNotFound
					}
					return &tt.ghRelease, nil
				},
				CreateReleaseFunc: func(opts *github.RepositoryRelease) (*github.RepositoryRelease, error) {
					if tt.createFail {
						return nil, fmt.Errorf("failed to create release")
					}
					releaseCreated = true
					return &github.RepositoryRelease{
						ID: github.Int64(123456),
					}, nil
				},
				UploadReleaseAssetFunc: func(releaseID int64, path string) error {
					if tt.uploadFail {
						return fmt.Errorf("failed to upload asset")
					}
					assetName := filepath.Base(path)
					assetContent, err := fs.ReadFile(path)
					require.NoError(t, err)
					uploads[assetName] = assetContent
					return nil
				},
			}

			releaser := GithubReleaser{
				client:  &client,
				config:  tt.config,
				force:   tt.force,
				fs:      fs,
				handler: newReleaseEventHandlerMock(tt.firing),
				logger:  testutils.NewNoopLogger(),
				project: tt.project,
				release: tt.release,
				runner:  newProjectRunnerMock(tt.runFail),
				workdir: workdir,
			}

			err := releaser.Release()
			tt.validate(t, fs, uploads, releaseCreated, err)
		})
	}
}

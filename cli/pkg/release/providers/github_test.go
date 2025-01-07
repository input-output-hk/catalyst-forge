package providers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v66/github"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubReleaserRelease(t *testing.T) {
	newProject := func(name, repoOwner, repoName, tag string, platforms []string) project.Project {
		return project.Project{
			Name: name,
			Blueprint: schema.Blueprint{
				Global: schema.Global{
					Repo: schema.GlobalRepo{
						Name: fmt.Sprintf("%s/%s", repoOwner, repoName),
					},
				},
				Project: schema.Project{
					CI: schema.ProjectCI{
						Targets: map[string]schema.Target{
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

	newRelease := func() schema.Release {
		return schema.Release{
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
		release    schema.Release
		ghRelease  github.RepositoryRelease
		config     GithubReleaserConfig
		files      map[string]string
		firing     bool
		force      bool
		runFail    bool
		createFail bool
		uploadFail bool
		validate   func(*testing.T, afero.Fs, map[string][]byte, bool, error)
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
			validate: func(t *testing.T, fs afero.Fs, uploads map[string][]byte, created bool, err error) {
				assert.NoError(t, err)

				filename := "project-linux-amd64.tar.gz"
				exists, err := afero.Exists(fs, filepath.Join(workdir, filename))
				require.NoError(t, err)
				assert.True(t, exists)

				assert.True(t, created)

				url := fmt.Sprintf("/repos/owner/repo/releases/123456/assets?name=%s", filename)
				data, ok := uploads[url]
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
			validate: func(t *testing.T, fs afero.Fs, uploads map[string][]byte, created bool, err error) {
				assert.NoError(t, err)

				for _, platform := range []string{"linux-amd64", "darwin-amd64"} {
					filename := fmt.Sprintf("project-%s.tar.gz", platform)
					exists, err := afero.Exists(fs, filepath.Join(workdir, filename))
					require.NoError(t, err)
					assert.True(t, exists)

					url := fmt.Sprintf("/repos/owner/repo/releases/123456/assets?name=%s", filename)
					data, ok := uploads[url]
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
			validate: func(t *testing.T, fs afero.Fs, uploads map[string][]byte, created bool, err error) {
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
			validate: func(t *testing.T, fs afero.Fs, uploads map[string][]byte, created bool, err error) {
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
			validate: func(t *testing.T, fs afero.Fs, uploads map[string][]byte, created bool, err error) {
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
			validate: func(t *testing.T, fs afero.Fs, uploads map[string][]byte, created bool, err error) {
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
			validate: func(t *testing.T, fs afero.Fs, uploads map[string][]byte, created bool, err error) {
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
			validate: func(t *testing.T, fs afero.Fs, uploads map[string][]byte, created bool, err error) {
				assert.NoError(t, err)
				assert.NotContains(t, uploads, "/repos/owner/repo/releases/123456/assets?name=project-linux-amd64.tar.gz")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			files := make(map[string]string)
			for k := range tt.files {
				nk := filepath.Join(workdir, k)
				files[nk] = tt.files[k]
			}
			testutils.SetupFS(t, fs, files)

			var releaseCreated bool
			uploads := make(map[string][]byte)
			client := github.NewClient(
				mock.NewMockedHTTPClient(
					mock.WithRequestMatchHandler(
						mock.GetReposReleasesTagsByOwnerByRepoByTag,
						http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							body, _ := io.ReadAll(r.Body)
							uploads[r.URL.String()] = body

							if tt.ghRelease.ID == nil {
								mock.WriteError(
									w,
									http.StatusNotFound,
									"release not found",
								)
								return
							}

							w.Write(mock.MustMarshal(tt.ghRelease))
						}),
					),
					mock.WithRequestMatchHandler(
						mock.PostReposReleasesByOwnerByRepo,
						http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							if tt.createFail {
								mock.WriteError(
									w,
									http.StatusInternalServerError,
									"failed to create release",
								)
								return
							}

							releaseCreated = true
							w.Write(mock.MustMarshal(github.RepositoryRelease{
								ID: github.Int64(123456),
							}))
						}),
					),
					mock.WithRequestMatchHandler(
						mock.PostReposReleasesAssetsByOwnerByRepoByReleaseId,
						http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							body, _ := io.ReadAll(r.Body)
							uploads[r.URL.String()] = body

							if tt.uploadFail {
								mock.WriteError(
									w,
									http.StatusInternalServerError,
									"failed to upload asset",
								)
								return
							}

							w.Write(mock.MustMarshal(github.ReleaseAsset{}))
						}),
					),
				),
			)

			releaser := GithubReleaser{
				client:  client,
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

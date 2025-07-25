package providers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	earthlyMocks "github.com/input-output-hk/catalyst-forge/cli/pkg/earthly/mocks"
	eventsMocks "github.com/input-output-hk/catalyst-forge/cli/pkg/events/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/providers/aws"
	awsMocks "github.com/input-output-hk/catalyst-forge/lib/providers/aws/mocks"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsReleaserRelease(t *testing.T) {
	type testResult struct {
		localFs fs.Filesystem
		s3Fs    fs.Filesystem
		err     error
	}

	newProject := func(projectName, branch, bucket, docsPath string) project.Project {
		repo := testutils.NewTestRepo(t)
		require.NoError(t, repo.WriteFile("README.md", []byte("hello docs")))
		_, err := repo.Commit("initial commit")
		require.NoError(t, err)

		return project.Project{
			Name: projectName,
			Blueprint: sb.Blueprint{
				Global: &global.Global{
					Repo: &global.Repo{
						Name:          "owner/repo",
						DefaultBranch: branch,
					},
					Ci: &global.CI{
						Release: &global.Release{
							Docs: &global.DocsRelease{
								Bucket: bucket,
								Path:   docsPath,
								Url:    "https://docs.example.com/",
							},
						},
					},
				},
				Project: &sp.Project{
					Release: map[string]sp.Release{
						"docs": {
							Target: "docs",
						},
					},
				},
			},
			Repo: &repo,
		}
	}

	tests := []struct {
		name        string
		project     project.Project
		releaseName string
		files       map[string]string
		s3files     map[string]string
		validate    func(*testing.T, testResult)
	}{
		{
			name:        "full",
			project:     newProject("project", "master", "bucket", "prefix"),
			releaseName: "test",
			files: map[string]string{
				"index.html": "test docs",
			},
			s3files: map[string]string{
				"test.html": "test",
			},
			validate: func(t *testing.T, result testResult) {
				assert.NoError(t, result.err)

				result.s3Fs.Walk("/", func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					fmt.Println(path)
					return nil
				})

				exists, err := result.s3Fs.Exists("/bucket/prefix/test/index.html")
				require.NoError(t, err)
				assert.True(t, exists)

				exists, err = result.s3Fs.Exists("/bucket/prefix/test/test.html")
				require.NoError(t, err)
				assert.False(t, exists)

				content, err := result.s3Fs.ReadFile("/bucket/prefix/test/index.html")
				require.NoError(t, err)
				assert.Equal(t, "test docs", string(content))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3Fs := billy.NewInMemoryFs()
			for name, content := range tt.s3files {
				p := filepath.Join("/",
					tt.project.Blueprint.Global.Ci.Release.Docs.Bucket,
					tt.project.Blueprint.Global.Ci.Release.Docs.Path,
					tt.releaseName,
					name,
				)
				fmt.Println(p)
				require.NoError(t, s3Fs.WriteFile(p, []byte(content), 0o644))
			}

			mockAWSS3 := &awsMocks.AWSS3ClientMock{
				PutObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					bucket := *params.Bucket
					key := *params.Key

					bucketDir := "/" + bucket
					_ = s3Fs.MkdirAll(bucketDir, 0o755)
					filePath := filepath.Join(bucketDir, key)
					data, err := io.ReadAll(params.Body)
					if err != nil {
						return nil, err
					}

					if err := s3Fs.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
						return nil, err
					}

					if err := s3Fs.WriteFile(filePath, data, 0o644); err != nil {
						return nil, err
					}

					return &s3.PutObjectOutput{}, nil
				},
				DeleteObjectFunc: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
					bucket := *params.Bucket
					key := *params.Key
					bucketDir := "/" + bucket
					filePath := filepath.Join(bucketDir, key)

					_ = s3Fs.Remove(filePath)

					return &s3.DeleteObjectOutput{}, nil
				},
				ListObjectsV2Func: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
					bucket := *params.Bucket
					prefix := ""
					if params.Prefix != nil {
						prefix = *params.Prefix
					}
					bucketDir := "/" + bucket

					var contents []s3types.Object
					_ = s3Fs.Walk(bucketDir, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return nil
						}

						relPath, _ := filepath.Rel(bucketDir, path)
						if !info.IsDir() && strings.HasPrefix(relPath, prefix) {
							contents = append(contents, s3types.Object{Key: &relPath})
						}

						return nil
					})

					return &s3.ListObjectsV2Output{Contents: contents}, nil
				},
			}

			localFs := billy.NewInMemoryFs()
			for name, content := range tt.files {
				p := filepath.Join("/", earthly.GetBuildPlatform(), name)
				require.NoError(t, localFs.MkdirAll(filepath.Dir(p), 0o755))
				require.NoError(t, localFs.WriteFile(p, []byte(content), 0o644))
			}

			w := walker.NewCustomDefaultFSWalker(localFs, testutils.NewNoopLogger())
			s3Client := aws.NewCustomS3Client(mockAWSS3, &w, testutils.NewNoopLogger())
			releaser := &DocsReleaser{
				config:      DocsReleaserConfig{Name: tt.releaseName},
				force:       true,
				fs:          localFs,
				handler:     &eventsMocks.EventHandlerMock{FiringFunc: func(_ *project.Project, _ map[string]cue.Value) bool { return true }},
				logger:      testutils.NewNoopLogger(),
				project:     &tt.project,
				release:     sp.Release{Target: "docs"},
				releaseName: "docs",
				runner:      &earthlyMocks.ProjectRunnerMock{RunTargetFunc: func(string, ...earthly.EarthlyExecutorOption) error { return nil }},
				s3:          s3Client,
				workdir:     "/",
			}

			err := releaser.Release()
			tt.validate(t, testResult{
				localFs: localFs,
				s3Fs:    s3Fs,
				err:     err,
			})
		})
	}
}

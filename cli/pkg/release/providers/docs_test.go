package providers

import (
	"context"
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
	gh "github.com/input-output-hk/catalyst-forge/lib/providers/github"
	ghMocks "github.com/input-output-hk/catalyst-forge/lib/providers/github/mocks"
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
	type prPostResult struct {
		prNumber int
		body     string
	}

	type testResult struct {
		localFs fs.Filesystem
		s3Fs    fs.Filesystem
		err     error
		prPost  prPostResult
	}

	type branchFile struct {
		branch  string
		name    string
		content string
	}

	newProject := func(projectName, branch, bucket, docsPath string) project.Project {
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
			Repo: nil,
		}
	}

	tests := []struct {
		name          string
		projectName   string
		defaultBranch string
		bucket        string
		prefix        string
		releaseName   string
		prNumber      int
		curBranch     string
		files         map[string]string
		s3files       map[string]string
		branchFiles   []branchFile
		prComments    []gh.PullRequestComment
		branches      []gh.Branch
		inCI          bool
		isPR          bool
		validate      func(*testing.T, testResult)
	}{
		{
			name:          "default branch",
			projectName:   "project",
			defaultBranch: "master",
			bucket:        "bucket",
			prefix:        "prefix",
			releaseName:   "test",
			prNumber:      123,
			curBranch:     "master",
			files: map[string]string{
				"index.html": "test docs",
			},
			s3files: map[string]string{
				"test.html": "test",
			},
			branchFiles: []branchFile{
				{
					branch:  "mybranch",
					name:    "index.html",
					content: "test docs",
				},
			},
			prComments: []gh.PullRequestComment{},
			branches: []gh.Branch{
				{
					Name: "master",
				},
			},
			inCI: true,
			isPR: false,
			validate: func(t *testing.T, result testResult) {
				assert.NoError(t, result.err)

				exists, err := result.s3Fs.Exists("/bucket/prefix/test/master/index.html")
				require.NoError(t, err)
				assert.True(t, exists)

				exists, err = result.s3Fs.Exists("/bucket/prefix/test/master/test.html")
				require.NoError(t, err)
				assert.False(t, exists)

				exists, err = result.s3Fs.Exists("/bucket/prefix/test/mybranch/index.html")
				require.NoError(t, err)
				assert.False(t, exists)

				content, err := result.s3Fs.ReadFile("/bucket/prefix/test/master/index.html")
				require.NoError(t, err)
				assert.Equal(t, "test docs", string(content))

				assert.Equal(t, 0, result.prPost.prNumber)
				assert.Equal(t, "", result.prPost.body)
			},
		},
		{
			name:          "pr",
			projectName:   "project",
			defaultBranch: "master",
			bucket:        "bucket",
			prefix:        "prefix",
			releaseName:   "test",
			prNumber:      123,
			curBranch:     "mybranch",
			files: map[string]string{
				"index.html": "test docs",
			},
			s3files: map[string]string{
				"test.html": "test",
			},
			prComments: []gh.PullRequestComment{},
			branches:   []gh.Branch{},
			inCI:       true,
			isPR:       true,
			validate: func(t *testing.T, result testResult) {
				assert.NoError(t, result.err)

				exists, err := result.s3Fs.Exists("/bucket/prefix/test/mybranch/index.html")
				require.NoError(t, err)
				assert.True(t, exists)

				exists, err = result.s3Fs.Exists("/bucket/prefix/test/mybranch/test.html")
				require.NoError(t, err)
				assert.False(t, exists)

				content, err := result.s3Fs.ReadFile("/bucket/prefix/test/mybranch/index.html")
				require.NoError(t, err)
				assert.Equal(t, "test docs", string(content))

				expectedBody := `
<!-- forge:v1:docs-preview -->
## 📚 Docs Preview

The docs for this PR can be previewed at the following URL:

https://docs.example.com/test/b/mybranch
`

				assert.Equal(t, expectedBody, result.prPost.body)
			},
		},
		{
			name:          "pr comment exists",
			projectName:   "project",
			defaultBranch: "master",
			bucket:        "bucket",
			prefix:        "prefix",
			releaseName:   "test",
			prNumber:      123,
			curBranch:     "mybranch",
			files: map[string]string{
				"index.html": "test docs",
			},
			s3files: map[string]string{
				"test.html": "test",
			},
			prComments: []gh.PullRequestComment{
				{
					Author: "github-actions[bot]",
					Body:   "<!-- forge:v1:docs-preview -->",
				},
			},
			branches: []gh.Branch{},
			inCI:     true,
			isPR:     true,
			validate: func(t *testing.T, result testResult) {
				assert.NoError(t, result.err)

				assert.Equal(t, 0, result.prPost.prNumber)
				assert.Equal(t, "", result.prPost.body)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.inCI {
				t.Setenv("GITHUB_ACTIONS", "true")
			}

			prj := newProject(tt.projectName, tt.defaultBranch, tt.bucket, tt.prefix)

			s3Fs := billy.NewInMemoryFs()
			logger := testutils.NewNoopLogger()
			for name, content := range tt.s3files {
				p := filepath.Join("/",
					prj.Blueprint.Global.Ci.Release.Docs.Bucket,
					prj.Blueprint.Global.Ci.Release.Docs.Path,
					tt.releaseName,
					tt.curBranch,
					name,
				)
				require.NoError(t, s3Fs.WriteFile(p, []byte(content), 0o644))
			}

			for _, branchFile := range tt.branchFiles {
				p := filepath.Join("/",
					prj.Blueprint.Global.Ci.Release.Docs.Bucket,
					prj.Blueprint.Global.Ci.Release.Docs.Path,
					tt.releaseName,
					branchFile.branch,
					branchFile.name,
				)
				require.NoError(t, s3Fs.MkdirAll(filepath.Dir(p), 0o755))
				require.NoError(t, s3Fs.WriteFile(p, []byte(branchFile.content), 0o644))
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
					bucketDir := "/" + bucket

					if params.Delimiter != nil {
						var prefixes []s3types.CommonPrefix

						s3Fs.Walk("/", func(path string, info os.FileInfo, err error) error {
							if err != nil {
								return err
							}

							if !info.IsDir() {
								return nil
							}

							p1 := strings.TrimPrefix(path, "/"+tt.bucket+"/")
							if strings.HasPrefix(p1, *params.Prefix) {
								prefix := strings.TrimPrefix(p1, *params.Prefix)
								prefixes = append(prefixes, s3types.CommonPrefix{Prefix: &prefix})
							}
							return nil
						})

						truncated := false
						return &s3.ListObjectsV2Output{CommonPrefixes: prefixes, IsTruncated: &truncated}, nil
					}

					var contents []s3types.Object
					_ = s3Fs.Walk(filepath.Join(bucketDir, *params.Prefix), func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return nil
						}

						if !info.IsDir() {
							p := strings.TrimPrefix(path, bucketDir+"/")
							contents = append(contents, s3types.Object{Key: &p})
						}

						return nil
					})

					return &s3.ListObjectsV2Output{Contents: contents}, nil
				},
			}

			ghEnvMock := &ghMocks.GithubEnvMock{
				IsPRFunc: func() bool {
					return tt.isPR
				},
				GetBranchFunc: func() string {
					return tt.curBranch
				},
				GetPRNumberFunc: func() int {
					return tt.prNumber
				},
			}

			var prPost prPostResult
			ghMock := &ghMocks.GithubClientMock{
				EnvFunc: func() gh.GithubEnv {
					return ghEnvMock
				},
				ListBranchesFunc: func() ([]gh.Branch, error) {
					return tt.branches, nil
				},
				ListPullRequestCommentsFunc: func(prNumber int) ([]gh.PullRequestComment, error) {
					return tt.prComments, nil
				},
				PostPullRequestCommentFunc: func(prNumber int, body string) error {
					prPost.prNumber = prNumber
					prPost.body = body
					return nil
				},
			}

			localFs := billy.NewInMemoryFs()
			for name, content := range tt.files {
				p := filepath.Join("/", earthly.GetBuildPlatform(), name)
				require.NoError(t, localFs.MkdirAll(filepath.Dir(p), 0o755))
				require.NoError(t, localFs.WriteFile(p, []byte(content), 0o644))
			}

			w := walker.NewCustomDefaultFSWalker(localFs, logger)
			s3Client := aws.NewCustomS3Client(mockAWSS3, &w, logger)
			releaser := &DocsReleaser{
				config:      DocsReleaserConfig{Name: tt.releaseName},
				force:       true,
				fs:          localFs,
				ghClient:    ghMock,
				handler:     &eventsMocks.EventHandlerMock{FiringFunc: func(_ *project.Project, _ map[string]cue.Value) bool { return true }},
				logger:      logger,
				project:     &prj,
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
				prPost:  prPost,
			})
		})
	}
}

package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	gb "github.com/go-git/go-billy/v5"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage"
	tu "github.com/input-output-hk/catalyst-forge/lib/deployment/utils/test"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	pb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global/providers"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	rm "github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrewDeployer_Deploy(t *testing.T) {
	tests := []struct {
		name         string
		cfg          ReleaseConfig
		assets       map[string]string
		archiveFiles map[string][]byte
		validate     func(t *testing.T, workFs fs.Filesystem, gitFs fs.Filesystem, remote *rm.GitRemoteInteractorMock, err error)
	}{
		{
			name: "full brew release",
			cfg: ReleaseConfig{
				Prefix: "my-cli",
				Name:   "My CLI",
				Brew: &BrewConfig{
					Template:    "go-v1",
					Description: "A test CLI",
					BinaryName:  "my-cli",
					Tap: GitRepoConfig{
						Repository: "https://github.com/test-org/homebrew-tap.git",
						Branch:     "main",
					},
				},
			},
			assets: map[string]string{
				"darwin/amd64": "https://github.com/test-org/my-cli/releases/download/v0.1.0/my-cli-darwin-amd64.tar.gz",
				"darwin/arm64": "https://github.com/test-org/my-cli/releases/download/v0.1.0/my-cli-darwin-arm64.tar.gz",
				"linux/amd64":  "https://github.com/test-org/my-cli/releases/download/v0.1.0/my-cli-linux-amd64.tar.gz",
				"linux/arm64":  "https://github.com/test-org/my-cli/releases/download/v0.1.0/my-cli-linux-arm64.tar.gz",
			},
			archiveFiles: map[string][]byte{
				"/tmp/my-cli-darwin-amd64.tar.gz": []byte("darwin amd64 content"),
				"/tmp/my-cli-darwin-arm64.tar.gz": []byte("darwin arm64 content"),
				"/tmp/my-cli-linux-amd64.tar.gz":  []byte("linux amd64 content"),
				"/tmp/my-cli-linux-arm64.tar.gz":  []byte("linux arm64 content"),
			},
			validate: func(t *testing.T, workFs fs.Filesystem, gitFs fs.Filesystem, remote *rm.GitRemoteInteractorMock, err error) {
				require.NoError(t, err)

				// Verify the recipe file was created in the git filesystem
				recipePath := "/repo/Formula/my-cli.rb"
				exists, err := gitFs.Exists(recipePath)
				require.NoError(t, err)
				assert.True(t, exists, "recipe file should be written to the tap repo")

				// Verify the content of the recipe file
				content, err := gitFs.ReadFile(recipePath)
				require.NoError(t, err)
				recipeContent := string(content)

				// Check basic structure
				assert.Contains(t, recipeContent, `class My-Cli < Formula`) // Title case with hyphen preserved
				assert.Contains(t, recipeContent, `desc "A test CLI"`)
				assert.Contains(t, recipeContent, `homepage "https://github.com/test-org/my-cli"`)
				assert.Contains(t, recipeContent, `version "v0.1.0"`)

				// Check all assets are present
				assert.Contains(t, recipeContent, `"DarwinAMD64"`)
				assert.Contains(t, recipeContent, `url "https://github.com/test-org/my-cli/releases/download/v0.1.0/my-cli-darwin-amd64.tar.gz"`)

				assert.Contains(t, recipeContent, `"DarwinARM64"`)
				assert.Contains(t, recipeContent, `url "https://github.com/test-org/my-cli/releases/download/v0.1.0/my-cli-darwin-arm64.tar.gz"`)

				assert.Contains(t, recipeContent, `"LinuxAMD64"`)
				assert.Contains(t, recipeContent, `url "https://github.com/test-org/my-cli/releases/download/v0.1.0/my-cli-linux-amd64.tar.gz"`)

				assert.Contains(t, recipeContent, `"LinuxARM64"`)
				assert.Contains(t, recipeContent, `url "https://github.com/test-org/my-cli/releases/download/v0.1.0/my-cli-linux-arm64.tar.gz"`)

				// Check that sha256 hashes are present (they will be calculated from actual file content)
				assert.Contains(t, recipeContent, `sha256 "`)

				// Check install section
				assert.Contains(t, recipeContent, `bin.install "my-cli"`)

				// Verify that Clone was called with correct parameters
				assert.Equal(t, 1, len(remote.CloneCalls()), "Clone should be called once")
				cloneCall := remote.CloneCalls()[0]
				assert.Equal(t, "https://github.com/test-org/homebrew-tap.git", cloneCall.O.URL)
				assert.Equal(t, "main", cloneCall.O.ReferenceName.String())
				assert.Equal(t, 1, cloneCall.O.Depth)
				assert.NotNil(t, cloneCall.O.Auth, "authentication should be set")

				// Verify that Push was called
				assert.Equal(t, 1, len(remote.PushCalls()), "Push should be called once")
				pushCall := remote.PushCalls()[0]
				assert.NotNil(t, pushCall.O.Auth, "authentication should be set for push")
			},
		},
		{
			name: "brew release with custom template URL",
			cfg: ReleaseConfig{
				Prefix: "tool",
				Name:   "Tool",
				Brew: &BrewConfig{
					Template:     "custom-template",
					Description:  "Custom tool",
					BinaryName:   "tool",
					TemplatesUrl: "", // Will be set to test server URL
					Tap: GitRepoConfig{
						Repository: "https://github.com/org/homebrew-tap.git",
						Branch:     "develop",
					},
				},
			},
			assets: map[string]string{
				"darwin/amd64": "https://github.com/org/tool/releases/download/v1.0.0/tool-darwin-amd64.tar.gz",
			},
			archiveFiles: map[string][]byte{
				"/tmp/tool-darwin-amd64.tar.gz": []byte("tool archive content"),
			},
			validate: func(t *testing.T, workFs fs.Filesystem, gitFs fs.Filesystem, remote *rm.GitRemoteInteractorMock, err error) {
				require.NoError(t, err)

				// Verify the recipe file was created
				recipePath := "/repo/Formula/tool.rb"
				exists, err := gitFs.Exists(recipePath)
				require.NoError(t, err)
				assert.True(t, exists, "recipe file should be written to the tap repo")

				// Verify Clone was called with develop branch
				assert.Equal(t, 1, len(remote.CloneCalls()))
				cloneCall := remote.CloneCalls()[0]
				assert.Equal(t, "develop", cloneCall.O.ReferenceName.String())
			},
		},
		{
			name: "brew release with git template repository",
			cfg: ReleaseConfig{
				Prefix: "app",
				Name:   "App",
				Brew: &BrewConfig{
					Template:    "go-v1",
					Description: "App from git template",
					BinaryName:  "app",
					Templates: &GitRepoConfig{
						Repository: "https://github.com/input-output-hk/catalyst-forge.git",
						Branch:     "master",
					},
					Tap: GitRepoConfig{
						Repository: "https://github.com/org/homebrew-tap.git",
						Branch:     "main",
					},
				},
			},
			assets: map[string]string{
				"darwin/amd64": "https://github.com/org/app/releases/download/v2.0.0/app-darwin-amd64.tar.gz",
			},
			archiveFiles: map[string][]byte{
				"/tmp/app-darwin-amd64.tar.gz": []byte("app archive content"),
			},
			validate: func(t *testing.T, workFs fs.Filesystem, gitFs fs.Filesystem, remote *rm.GitRemoteInteractorMock, err error) {
				require.NoError(t, err)

				// Verify the recipe file was created
				recipePath := "/repo/Formula/app.rb"
				exists, err := gitFs.Exists(recipePath)
				require.NoError(t, err)
				assert.True(t, exists, "recipe file should be written to the tap repo")

				// Verify the content uses the correct template structure with .Assets
				content, err := gitFs.ReadFile(recipePath)
				require.NoError(t, err)
				recipeContent := string(content)
				assert.Contains(t, recipeContent, `class App < Formula`)
				assert.Contains(t, recipeContent, `desc "App from git template"`)
				assert.Contains(t, recipeContent, `bin.install "app"`)

				// Verify that Clone was called twice (once for template repo, once for tap repo)
				assert.Equal(t, 2, len(remote.CloneCalls()), "Clone should be called twice")

				// First call should be for template repository
				templateClone := remote.CloneCalls()[0]
				assert.Equal(t, "https://github.com/input-output-hk/catalyst-forge.git", templateClone.O.URL)
				assert.Equal(t, "master", templateClone.O.ReferenceName.String())

				// Second call should be for tap repository
				tapClone := remote.CloneCalls()[1]
				assert.Equal(t, "https://github.com/org/homebrew-tap.git", tapClone.O.URL)
				assert.Equal(t, "main", tapClone.O.ReferenceName.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup HTTP test server for template
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Return a template that uses all the fields
				fmt.Fprintln(w, `class {{ .Name | title }} < Formula
  desc "{{ .Description }}"
  homepage "{{ .Homepage }}"
  version "{{ .Version }}"
  
  {{- range $key, $asset := .Assets }}
  "{{ $key }}":
    url "{{ $asset.URL }}"
    sha256 "{{ $asset.SHA256 }}"
  {{- end }}
  
  def install
    bin.install "{{ .BinaryName }}"
  end
end`)
			}))
			defer ts.Close()

			// Set the template URL to the test server
			if tt.cfg.Brew.TemplatesUrl == "" {
				tt.cfg.Brew.TemplatesUrl = ts.URL
			}

			// Create separate filesystems for work and git
			workFs := billy.NewInMemoryFs()
			gitFs := billy.NewInMemoryFs()

			// Create the archive files in the work filesystem
			for path, content := range tt.archiveFiles {
				err := workFs.WriteFile(path, content, 0644)
				require.NoError(t, err)
			}

			// Setup secret store
			ss := tu.NewMockSecretStore(map[string]string{"token": "test-token"})
			logger := testutils.NewNoopLogger()

			// Track git operations
			var capturedRepo *gg.Repository
			var cloneCallCount int
			remote := &rm.GitRemoteInteractorMock{
				CloneFunc: func(s storage.Storer, worktree gb.Filesystem, o *gg.CloneOptions) (*gg.Repository, error) {
					repo, err := gg.Init(s, worktree)

					// For the git template test case, write the template file to the first clone (template repo)
					if tt.name == "brew release with git template repository" && cloneCallCount == 0 {
						// This is the template repository clone
						templateContent := `class {{ .Name | title }} < Formula
  desc "{{ .Description }}"
  homepage "{{ .Homepage }}"
  version "{{ .Version }}"
  
  # Support for multi-architecture builds
  on_macos do
    if Hardware::CPU.intel?
      url "{{ .Assets.DarwinAMD64.URL }}"
      sha256 "{{ .Assets.DarwinAMD64.SHA256 }}"
    elsif Hardware::CPU.arm?
      url "{{ .Assets.DarwinARM64.URL }}"
      sha256 "{{ .Assets.DarwinARM64.SHA256 }}"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "{{ .Assets.LinuxAMD64.URL }}"
      sha256 "{{ .Assets.LinuxAMD64.SHA256 }}"
    elsif Hardware::CPU.arm?
      url "{{ .Assets.LinuxARM64.URL }}"
      sha256 "{{ .Assets.LinuxARM64.SHA256 }}"
    end
  end

  def install
    # Installation instructions for the binary
    bin.install "{{ .BinaryName }}"
  end

  test do
    system "#{bin}/{{ .BinaryName }}", "--version"
  end
end`
						file, err := worktree.Create("go-v1.rb.tpl")
						if err != nil {
							return nil, err
						}
						_, err = file.Write([]byte(templateContent))
						if err != nil {
							return nil, err
						}
						file.Close()
					} else {
						capturedRepo = repo
					}

					cloneCallCount++
					return repo, err
				},
				PushFunc: func(r *gg.Repository, o *gg.PushOptions) error {
					// Verify this is the same repo we cloned for the tap (not the template)
					if capturedRepo != nil {
						assert.Equal(t, capturedRepo, r, "Push should be called on the same repository that was cloned")
					}
					return nil
				},
			}

			// Create project
			projectName := "my-cli"
			if tt.name == "brew release with custom template URL" {
				projectName = "tool"
			} else if tt.name == "brew release with git template repository" {
				projectName = "app"
			}
			p := project.Project{
				Name: projectName,
				Blueprint: blueprint.Blueprint{
					Global: &global.Global{
						Repo: &global.Repo{
							Name: fmt.Sprintf("test-org/%s", projectName),
						},
						Ci: &global.CI{
							Providers: &pb.Providers{
								Git: &pb.Git{
									Credentials: sc.Secret{
										Provider: "local",
										Path:     "token",
									},
								},
							},
						},
					},
				},
				Tag: &project.ProjectTag{
					Full: "v0.1.0",
				},
			}
			if tt.name == "brew release with custom template URL" {
				p.Blueprint.Global.Repo.Name = "org/tool"
				p.Tag.Full = "v1.0.0"
			} else if tt.name == "brew release with git template repository" {
				p.Blueprint.Global.Repo.Name = "org/app"
				p.Tag.Full = "v2.0.0"
			}

			// Create deployer with both filesystems using NewBrewDeployer
			deployer := NewBrewDeployer(
				&tt.cfg,
				"/tmp",
				WithFilesystem(workFs),
				WithGitFilesystem(gitFs),
				WithSecretsStore(ss),
				WithLogger(logger),
				WithRemote(remote),
			)
			deployer.project = p

			// Execute deployment
			err := deployer.Deploy("release", tt.assets)

			// Validate results
			tt.validate(t, workFs, gitFs, remote, err)
		})
	}
}

func TestBrewDeployer_calculateSHA256(t *testing.T) {
	tests := []struct {
		name        string
		fileContent []byte
		expectError bool
		validate    func(t *testing.T, result string, err error)
	}{
		{
			name:        "valid_file",
			fileContent: []byte("test content"),
			expectError: false,
			validate: func(t *testing.T, result string, err error) {
				require.NoError(t, err)
				// SHA256 of "test content"
				assert.Equal(t, "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72", result)
			},
		},
		{
			name:        "empty_file",
			fileContent: []byte(""),
			expectError: false,
			validate: func(t *testing.T, result string, err error) {
				require.NoError(t, err)
				// SHA256 of empty string
				assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", result)
			},
		},
		{
			name:        "non_existent_file",
			fileContent: nil,
			expectError: true,
			validate: func(t *testing.T, result string, err error) {
				assert.Error(t, err)
				assert.Equal(t, "", result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			deployer := &BrewDeployer{
				fs: fs,
			}

			testPath := "/test/file.tar.gz"
			if tt.fileContent != nil {
				err := fs.WriteFile(testPath, tt.fileContent, 0644)
				require.NoError(t, err)
			} else {
				testPath = "/non/existent/file.tar.gz"
			}

			result, err := deployer.calculateSHA256(testPath)
			tt.validate(t, result, err)
		})
	}
}

func TestBrewDeployer_platformToAssetKey(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		expected string
	}{
		{
			name:     "darwin_amd64",
			platform: "darwin/amd64",
			expected: "DarwinAMD64",
		},
		{
			name:     "darwin_arm64",
			platform: "darwin/arm64",
			expected: "DarwinARM64",
		},
		{
			name:     "linux_amd64",
			platform: "linux/amd64",
			expected: "LinuxAMD64",
		},
		{
			name:     "linux_arm64",
			platform: "linux/arm64",
			expected: "LinuxARM64",
		},
		{
			name:     "unknown_platform",
			platform: "windows/amd64",
			expected: "",
		},
		{
			name:     "invalid_format",
			platform: "invalid-platform",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployer := &BrewDeployer{}
			result := deployer.platformToAssetKey(tt.platform)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBrewDeployer_getTemplateData(t *testing.T) {
	tests := []struct {
		name     string
		assets   map[string]string
		files    map[string][]byte
		validate func(t *testing.T, result *BrewTemplateData, err error)
	}{
		{
			name: "single_platform",
			assets: map[string]string{
				"darwin/amd64": "https://github.com/test/app/releases/download/v1.0.0/app-darwin-amd64.tar.gz",
			},
			files: map[string][]byte{
				"/tmp/my-app-darwin-amd64.tar.gz": []byte("test content"),
			},
			validate: func(t *testing.T, result *BrewTemplateData, err error) {
				require.NoError(t, err)
				assert.Equal(t, "test-app", result.Name)
				assert.Equal(t, "A test app", result.Description)
				assert.Equal(t, "https://github.com/test/test-app", result.Homepage)
				assert.Equal(t, "v1.0.0", result.Version)
				assert.Equal(t, "test-app", result.BinaryName)
				assert.Len(t, result.Assets, 1)
				assert.Contains(t, result.Assets, "DarwinAMD64")
				assert.Equal(t, "https://github.com/test/app/releases/download/v1.0.0/app-darwin-amd64.tar.gz", result.Assets["DarwinAMD64"].URL)
				assert.Equal(t, "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72", result.Assets["DarwinAMD64"].SHA256)
			},
		},
		{
			name: "multiple_platforms",
			assets: map[string]string{
				"darwin/amd64": "https://github.com/test/app/releases/download/v1.0.0/app-darwin-amd64.tar.gz",
				"linux/arm64":  "https://github.com/test/app/releases/download/v1.0.0/app-linux-arm64.tar.gz",
			},
			files: map[string][]byte{
				"/tmp/my-app-darwin-amd64.tar.gz": []byte("darwin content"),
				"/tmp/my-app-linux-arm64.tar.gz":  []byte("linux content"),
			},
			validate: func(t *testing.T, result *BrewTemplateData, err error) {
				require.NoError(t, err)
				assert.Len(t, result.Assets, 2)
				assert.Contains(t, result.Assets, "DarwinAMD64")
				assert.Contains(t, result.Assets, "LinuxARM64")
			},
		},
		{
			name: "missing_file",
			assets: map[string]string{
				"darwin/amd64": "https://github.com/test/app/releases/download/v1.0.0/app-darwin-amd64.tar.gz",
			},
			files: map[string][]byte{},
			validate: func(t *testing.T, result *BrewTemplateData, err error) {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "failed to calculate SHA256")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()

			// Create test files
			for path, content := range tt.files {
				err := fs.WriteFile(path, content, 0644)
				require.NoError(t, err)
			}

			deployer := &BrewDeployer{
				fs:      fs,
				workdir: "/tmp",
				cfg: &ReleaseConfig{
					Prefix: "my-app",
					Brew: &BrewConfig{
						Description: "A test app",
						BinaryName:  "test-app",
					},
				},
				project: project.Project{
					Name: "test-app",
					Blueprint: blueprint.Blueprint{
						Global: &global.Global{
							Repo: &global.Repo{
								Name: "test/test-app",
							},
						},
					},
					Tag: &project.ProjectTag{
						Full: "v1.0.0",
					},
				},
			}

			result, err := deployer.getTemplateData(tt.assets)
			tt.validate(t, result, err)
		})
	}
}

func TestBrewDeployer_fetchTemplateFromGit(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		branch       string
		validate     func(t *testing.T, result []byte, err error)
	}{
		{
			name:         "successful_fetch",
			templateName: "go-v1",
			branch:       "main",
			validate: func(t *testing.T, result []byte, err error) {
				require.NoError(t, err)
				assert.Contains(t, string(result), "class {{ .Name | title }} < Formula")
				assert.Contains(t, string(result), "bin.install")
			},
		},
		{
			name:         "default_branch",
			templateName: "go-v1",
			branch:       "",
			validate: func(t *testing.T, result []byte, err error) {
				require.NoError(t, err)
				assert.Contains(t, string(result), "class {{ .Name | title }} < Formula")
			},
		},
		{
			name:         "missing_template_file",
			templateName: "nonexistent",
			branch:       "main",
			validate: func(t *testing.T, result []byte, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "could not read template file")
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutils.NewNoopLogger()

			// Create mock remote that simulates cloning a template repository
			remote := &rm.GitRemoteInteractorMock{
				CloneFunc: func(s storage.Storer, worktree gb.Filesystem, o *gg.CloneOptions) (*gg.Repository, error) {
					repo, err := gg.Init(s, worktree)
					if err != nil {
						return nil, err
					}

					// Only create the template file if it's the expected name
					if tt.templateName == "go-v1" {
						templateContent := `class {{ .Name | title }} < Formula
  desc "{{ .Description }}"
  homepage "{{ .Homepage }}"
  version "{{ .Version }}"
  
  def install
    bin.install "{{ .BinaryName }}"
  end
end`
						file, err := worktree.Create("go-v1.rb.tpl")
						if err != nil {
							return nil, err
						}
						_, err = file.Write([]byte(templateContent))
						if err != nil {
							return nil, err
						}
						file.Close()
					}

					return repo, nil
				},
			}

			deployer := &BrewDeployer{
				cfg: &ReleaseConfig{
					Brew: &BrewConfig{
						Template: tt.templateName,
						Templates: &GitRepoConfig{
							Repository: "https://github.com/input-output-hk/catalyst-forge.git",
							Branch:     tt.branch,
						},
					},
				},
				logger: logger,
				remote: remote,
			}

			result, err := deployer.fetchTemplateFromGit()
			tt.validate(t, result, err)
		})
	}
}

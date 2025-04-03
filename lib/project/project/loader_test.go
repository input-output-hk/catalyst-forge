package project

import (
	"os"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/project/injector"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultProjectLoaderLoad(t *testing.T) {
	ctx := cuecontext.New()

	earthfile := `
VERSION 0.8

foo:
	ARG foo

bar:
	ARG bar
`
	bp := `
version: "1.0"
global: {
  repo: {
    defaultBranch: "main"
    name: "foo"
	url: "bar"
  }
}
project: name: "foo"
`

	tests := []struct {
		name        string
		projectPath string
		files       map[string]string
		tag         string
		injectors   []injector.BlueprintInjector
		runtimes    func(fs.Filesystem) []RuntimeData
		env         map[string]string
		initGit     bool
		validate    func(*testing.T, Project, error)
	}{
		{
			name:        "full",
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile":     earthfile,
				"/project/blueprint.cue": bp,
			},
			tag:       "foo/v1.0.0",
			injectors: []injector.BlueprintInjector{},
			runtimes:  func(f fs.Filesystem) []RuntimeData { return nil },
			env:       map[string]string{},
			initGit:   true,
			validate: func(t *testing.T, p Project, err error) {
				require.NoError(t, err)
				assert.Equal(t, "/project", p.Path)
				assert.Equal(t, "/project", p.RepoRoot)
				assert.Equal(t, "foo", p.Name)
				assert.Equal(t, []string{"foo", "bar"}, p.Earthfile.Targets())

				require.NoError(t, err)
				assert.Equal(t, "foo/v1.0.0", p.Tag.Full)
				assert.Equal(t, "foo", p.Tag.Project)
				assert.Equal(t, "v1.0.0", string(p.Tag.Version))
			},
		},
		{
			name:        "non-project tag",
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile":     earthfile,
				"/project/blueprint.cue": bp,
			},
			tag:       "v1.0.0",
			injectors: []injector.BlueprintInjector{},
			runtimes:  func(f fs.Filesystem) []RuntimeData { return nil },
			env:       map[string]string{},
			initGit:   true,
			validate: func(t *testing.T, p Project, err error) {
				assert.Equal(t, "v1.0.0", p.Tag.Full)
				assert.Equal(t, "foo", p.Tag.Project)
				assert.Equal(t, "v1.0.0", string(p.Tag.Version))
			},
		},
		{
			name:        "with injectors",
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile": earthfile,
				"/project/blueprint.cue": `
		version: "1.0"
		global: {
		  repo: {
		    defaultBranch: "main"
		    name: "foo"
			url: "bar"
		  }
		}
		project: {
		  name: "foo"
		  ci: targets: foo: args: foo: _ @env(name="FOO",type="string")
		}
		`,
			},
			injectors: []injector.BlueprintInjector{
				injector.NewBlueprintEnvInjector(ctx, testutils.NewNoopLogger()),
			},
			runtimes: func(f fs.Filesystem) []RuntimeData { return nil },
			env: map[string]string{
				"FOO": "bar",
			},
			initGit: true,
			validate: func(t *testing.T, p Project, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "bar", p.Blueprint.Project.Ci.Targets["foo"].Args["foo"])
			},
		},
		{
			name:        "with runtime",
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile": earthfile,
				"/project/blueprint.cue": `
		version: "1.0"
		global: {
		  repo: {
		    defaultBranch: "main"
		    name: "foo"
			url: "bar"
		  }
		}
		project: {
		  name: "foo"
		  ci: targets: foo: args: foo: _ @forge(name="GIT_COMMIT_HASH")
		}
		`,
			},
			injectors: []injector.BlueprintInjector{
				injector.NewBlueprintEnvInjector(ctx, testutils.NewNoopLogger()),
			},
			runtimes: func(f fs.Filesystem) []RuntimeData {
				return []RuntimeData{NewCustomGitRuntime(f, testutils.NewNoopLogger())}
			},
			env:     map[string]string{},
			initGit: true,
			validate: func(t *testing.T, p Project, err error) {
				assert.NoError(t, err)
				head, err := p.Repo.Head()
				require.NoError(t, err)
				assert.Equal(t, head.Hash().String(), p.Blueprint.Project.Ci.Targets["foo"].Args["foo"])
			},
		},
		{
			name:        "no git",
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile":     earthfile,
				"/project/blueprint.cue": bp,
			},
			injectors: []injector.BlueprintInjector{},
			runtimes:  func(f fs.Filesystem) []RuntimeData { return nil },
			env:       map[string]string{},
			initGit:   false,
			validate: func(t *testing.T, p Project, err error) {
				assert.Error(t, err)
				assert.Equal(
					t,
					"failed to find git root: git root not found",
					err.Error(),
				)
			},
		},
		{
			name:        "invalid Earthfile",
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile":     "invalid",
				"/project/blueprint.cue": bp,
			},
			injectors: []injector.BlueprintInjector{},
			runtimes:  func(f fs.Filesystem) []RuntimeData { return nil },
			env:       map[string]string{},
			initGit:   true,
			validate: func(t *testing.T, p Project, err error) {
				assert.Error(t, err)
				assert.Equal(
					t,
					"failed to parse Earthfile: lexer error: Earthfile\nsyntax error: line 1:0: token recognition error at: 'invalid'",
					err.Error(),
				)
			},
		},
		{
			name:        "invalid blueprint",
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile":     earthfile,
				"/project/blueprint.cue": "invalid",
			},
			injectors: []injector.BlueprintInjector{},
			runtimes:  func(f fs.Filesystem) []RuntimeData { return nil },
			env:       map[string]string{},
			initGit:   true,
			validate: func(t *testing.T, p Project, err error) {
				assert.Error(t, err)
				assert.Equal(
					t,
					"failed to load blueprint: failed to load blueprint file: failed to compile CUE file: reference \"invalid\" not found",
					err.Error(),
				)
			},
		},
		{
			name:        "incomplete blueprint",
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile": earthfile,
				"/project/blueprint.cue": `
		version: "1.0"
		global: {
		  repo: {
		    defaultBranch: "main"
		    name: "foo"
			url: "bar"
		  }
		}
		project: {
		  name: "foo"
		  ci: targets: foo: args: foo: _ @env(name="INVALID",type="string")
		}
		`,
			},
			injectors: []injector.BlueprintInjector{},
			runtimes:  func(f fs.Filesystem) []RuntimeData { return nil },
			env:       map[string]string{},
			initGit:   true,
			validate: func(t *testing.T, p Project, err error) {
				assert.Error(t, err)
				assert.Equal(
					t,
					"failed to validate blueprint: #Blueprint.project.ci.targets.foo.args.foo: incomplete value string",
					err.Error(),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			logger := testutils.NewNoopLogger()

			defer func() {
				for k := range tt.env {
					require.NoError(t, os.Unsetenv(k))
				}
			}()

			for k, v := range tt.env {
				require.NoError(t, os.Setenv(k, v))
			}

			testutils.SetupFS(t, fs, tt.files)

			if tt.initGit {
				repo := testutils.NewTestRepoWithFS(t, tt.projectPath, fs)
				err := repo.StageFile("Earthfile")
				require.NoError(t, err)

				err = repo.StageFile("blueprint.cue")
				require.NoError(t, err)

				_, err = repo.Commit("Initial commit")
				require.NoError(t, err)

				head, err := repo.Head()
				require.NoError(t, err)
				if tt.tag != "" {
					_, err := repo.NewTag(head.Hash(), tt.tag, "Initial tag")
					require.NoError(t, err)
				}
			}

			bpLoader := blueprint.NewCustomBlueprintLoader(ctx, fs, logger)
			loader := DefaultProjectLoader{
				blueprintLoader: &bpLoader,
				ctx:             ctx,
				fs:              fs,
				injectors:       tt.injectors,
				logger:          logger,
				runtimes:        tt.runtimes(fs),
			}

			p, err := loader.Load(tt.projectPath)
			tt.validate(t, p, err)
		})
	}
}

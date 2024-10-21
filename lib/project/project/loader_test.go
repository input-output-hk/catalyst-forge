package project

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/project/injector"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	df "gopkg.in/jfontan/go-billy-desfacer.v0"
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
    name: "foo"
	defaultBranch: "main"
  }
}
project: name: "foo"
`

	tests := []struct {
		name        string
		fs          afero.Fs
		projectPath string
		files       map[string]string
		tag         string
		injectors   []injector.BlueprintInjector
		runtimes    []RuntimeData
		env         map[string]string
		initGit     bool
		validate    func(*testing.T, Project, error)
	}{
		{
			name:        "full",
			fs:          afero.NewMemMapFs(),
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile":     earthfile,
				"/project/blueprint.cue": bp,
			},
			tag:       "foo/v1.0.0",
			injectors: []injector.BlueprintInjector{},
			runtimes:  []RuntimeData{},
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
			name:        "with injectors",
			fs:          afero.NewMemMapFs(),
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile": earthfile,
				"/project/blueprint.cue": `
version: "1.0"
global: {
  repo: {
    name: "foo"
	defaultBranch: "main"
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
			runtimes: []RuntimeData{},
			env: map[string]string{
				"FOO": "bar",
			},
			initGit: true,
			validate: func(t *testing.T, p Project, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "bar", p.Blueprint.Project.CI.Targets["foo"].Args["foo"])
			},
		},
		{
			name:        "with runtime",
			fs:          afero.NewMemMapFs(),
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile": earthfile,
				"/project/blueprint.cue": `
version: "1.0"
global: {
  repo: {
    name: "foo"
	defaultBranch: "main"
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
			runtimes: []RuntimeData{
				NewGitRuntime(testutils.NewNoopLogger()),
			},
			env:     map[string]string{},
			initGit: true,
			validate: func(t *testing.T, p Project, err error) {
				assert.NoError(t, err)
				head, err := p.Repo.Head()
				require.NoError(t, err)
				assert.Equal(t, head.Hash().String(), p.Blueprint.Project.CI.Targets["foo"].Args["foo"])
			},
		},
		{
			name:        "no git",
			fs:          afero.NewMemMapFs(),
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile":     earthfile,
				"/project/blueprint.cue": bp,
			},
			injectors: []injector.BlueprintInjector{},
			runtimes:  []RuntimeData{},
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
			fs:          afero.NewMemMapFs(),
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile":     "invalid",
				"/project/blueprint.cue": bp,
			},
			injectors: []injector.BlueprintInjector{},
			runtimes:  []RuntimeData{},
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
			fs:          afero.NewMemMapFs(),
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile":     earthfile,
				"/project/blueprint.cue": "invalid",
			},
			injectors: []injector.BlueprintInjector{},
			runtimes:  []RuntimeData{},
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
			fs:          afero.NewMemMapFs(),
			projectPath: "/project",
			files: map[string]string{
				"/project/Earthfile": earthfile,
				"/project/blueprint.cue": `
version: "1.0"
global: {
  repo: {
    name: "foo"
	defaultBranch: "main"
  }
}
project: {
  name: "foo"
  ci: targets: foo: args: foo: _ @env(name="INVALID",type="string")
}
`,
			},
			injectors: []injector.BlueprintInjector{},
			runtimes:  []RuntimeData{},
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
			logger := testutils.NewNoopLogger()

			defer func() {
				for k := range tt.env {
					require.NoError(t, os.Unsetenv(k))
				}
			}()

			for k, v := range tt.env {
				require.NoError(t, os.Setenv(k, v))
			}

			testutils.SetupFS(t, tt.fs, tt.files)

			if tt.initGit {
				tt.fs.Mkdir(filepath.Join(tt.projectPath, ".git"), 0755)
				workdir := df.New(afero.NewBasePathFs(tt.fs, tt.projectPath))
				gitdir := df.New(afero.NewBasePathFs(tt.fs, filepath.Join(tt.projectPath, ".git")))
				storage := filesystem.NewStorage(gitdir, cache.NewObjectLRUDefault())
				r, err := gg.Init(storage, workdir)
				require.NoError(t, err)

				wt, err := r.Worktree()
				require.NoError(t, err)
				repo := testutils.InMemRepo{
					Fs:       workdir,
					Repo:     r,
					Worktree: wt,
				}

				repo.AddExistingFile(t, "Earthfile")
				repo.AddExistingFile(t, "blueprint.cue")
				repo.Commit(t, "Initial commit")

				head, err := repo.Repo.Head()
				require.NoError(t, err)
				if tt.tag != "" {
					repo.Tag(t, head.Hash(), tt.tag, "Initial tag")
				}
			}

			bpLoader := blueprint.NewCustomBlueprintLoader(ctx, tt.fs, logger)
			loader := DefaultProjectLoader{
				blueprintLoader: &bpLoader,
				ctx:             ctx,
				fs:              tt.fs,
				injectors:       tt.injectors,
				logger:          logger,
				runtimes:        tt.runtimes,
			}

			p, err := loader.Load(tt.projectPath)
			tt.validate(t, p, err)
		})
	}
}

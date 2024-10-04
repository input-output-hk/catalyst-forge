package project

import (
	"fmt"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/project/loader"
	"github.com/input-output-hk/catalyst-forge/lib/project/loader/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	df "gopkg.in/jfontan/go-billy-desfacer.v0"
)

type wrapfs struct {
	afero.Fs

	attempts  int
	failAfter int
	trigger   error
}

func (w *wrapfs) Open(name string) (afero.File, error) {
	w.attempts++
	if w.attempts == w.failAfter {
		return nil, w.trigger
	}
	return afero.Fs.Open(w.Fs, name)
}

func TestDefaultProjectLoader_Load(t *testing.T) {
	ctx := cuecontext.New()
	tests := []struct {
		name            string
		fs              afero.Fs
		path            string
		blueprint       blueprint.RawBlueprint
		files           map[string]string
		bpErr           error
		expectedName    string
		expectedTargets []string
		expectErr       bool
		expectedErr     string
	}{
		{
			name: "simple",
			fs:   afero.NewMemMapFs(),
			path: "/project",
			blueprint: blueprint.NewRawBlueprint(ctx.CompileString(`
version: "1.0"
project: name: "foo"
			`)),
			files: map[string]string{
				"/project/Earthfile": `
VERSION 0.8

foo:
	ARG foo

bar:
	ARG bar
`,
			},
			expectedName: "foo",
			expectedTargets: []string{
				"foo",
				"bar",
			},
			expectErr:   false,
			expectedErr: "",
		},
		{
			name:        "error loading blueprint",
			fs:          afero.NewMemMapFs(),
			path:        "/",
			blueprint:   blueprint.RawBlueprint{},
			files:       map[string]string{},
			bpErr:       fmt.Errorf("error"),
			expectErr:   true,
			expectedErr: "failed to load blueprint: error",
		},
		{
			name: "error decoding blueprint",
			fs:   afero.NewMemMapFs(),
			path: "/",
			blueprint: blueprint.NewRawBlueprint(ctx.CompileString(`
		version: 1.0
					`)),
			files: map[string]string{
				"/Earthfile": `
		VERSION 0.8
		`,
			},
			bpErr:       nil,
			expectErr:   true,
			expectedErr: "",
		},
		{
			name: "error reading Earthfile",
			fs: &wrapfs{
				Fs:        afero.NewMemMapFs(),
				trigger:   fmt.Errorf("error"),
				failAfter: 5,
			},
			path:      "/",
			blueprint: blueprint.RawBlueprint{},
			files: map[string]string{
				"/Earthfile": "",
			},
			bpErr:       nil,
			expectErr:   true,
			expectedErr: "failed to read Earthfile: error",
		},
		{
			name:      "error parsing Earthfile",
			fs:        afero.NewMemMapFs(),
			path:      "/",
			blueprint: blueprint.RawBlueprint{},
			files: map[string]string{
				"/Earthfile": "bad",
			},
			bpErr:       nil,
			expectErr:   true,
			expectedErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutils.SetupFS(t, tt.fs, tt.files)

			tt.fs.Mkdir(filepath.Join(tt.path, ".git"), 0755)
			workdir := afero.NewBasePathFs(tt.fs, tt.path)
			gitdir := afero.NewBasePathFs(tt.fs, filepath.Join(tt.path, ".git"))
			storage := filesystem.NewStorage(df.New(gitdir), cache.NewObjectLRUDefault())
			_, err := gg.Init(storage, df.New(workdir))
			assert.NoError(t, err)

			bpLoader := &mocks.BlueprintLoaderMock{
				LoadFunc: func(p, g string) (blueprint.RawBlueprint, error) {
					if tt.bpErr != nil {
						return blueprint.RawBlueprint{}, tt.bpErr
					}
					return tt.blueprint, nil
				},
				SetOverriderFunc: func(overrider loader.InjectorOverrider) {},
			}

			loader := DefaultProjectLoader{
				blueprintLoader: bpLoader,
				fs:              tt.fs,
				logger:          testutils.NewNoopLogger(),
			}

			got, err := loader.Load(tt.path)
			if testutils.AssertError(t, err, tt.expectErr, tt.expectedErr) {
				return
			}

			assert.Equal(t, tt.expectedName, got.Name)
			assert.Equal(t, tt.expectedTargets, got.Earthfile.Targets())
		})
	}
}

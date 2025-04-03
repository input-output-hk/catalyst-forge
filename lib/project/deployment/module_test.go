package deployment

import (
	"log/slog"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	sm "github.com/input-output-hk/catalyst-forge/lib/project/secrets/mocks"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleBundleDump(t *testing.T) {
	ctx := cuecontext.New()
	bundle := ModuleBundle{
		Raw: ctx.CompileString(`foo: "bar"`),
	}

	result, err := bundle.Dump()
	require.NoError(t, err)
	expected := `{
	foo: "bar"
}`
	require.Equal(t, expected, string(result))
}

func TestDumpModule(t *testing.T) {
	ctx := cuecontext.New()
	mod := sp.Module{
		Instance:  "foo",
		Namespace: "default",
		Type:      "kcl",
	}

	result, err := DumpModule(ctx, mod)
	require.NoError(t, err)
	expected := `{
	instance:  "foo"
	namespace: "default"
	type:      "kcl"
}`
	require.Equal(t, expected, string(result))
}

func TestDumpBundle(t *testing.T) {
	ctx := cuecontext.New()
	bundle := sp.ModuleBundle{
		Env: "test",
		Modules: map[string]sp.Module{
			"test": {
				Instance:  "foo",
				Namespace: "default",
				Type:      "kcl",
			},
		},
	}

	result, err := DumpBundle(ctx, bundle)
	require.NoError(t, err)
	expected := `{
	env: "test"
	modules: {
		test: {
			instance:  "foo"
			namespace: "default"
			type:      "kcl"
		}
	}
}`
	require.Equal(t, expected, string(result))
}

func TestFetchBundle(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		validate func(*testing.T, ModuleBundle, error)
	}{
		{
			name: "success",
			files: map[string]string{
				"project/blueprint.cue": makeBlueprint(),
			},
			validate: func(t *testing.T, bundle ModuleBundle, err error) {
				require.NoError(t, err)

				b := bundle.Bundle
				assert.Equal(t, "test", b.Env)
				assert.Len(t, b.Modules, 1)
				assert.Equal(t, "module", b.Modules["main"].Name)
				assert.Equal(t, "v1.0.0", b.Modules["main"].Version)
			},
		},
		{
			name: "no bundle",
			files: map[string]string{
				"project/blueprint.cue": `version: "1.0"`,
			},
			validate: func(t *testing.T, bundle ModuleBundle, err error) {
				require.Error(t, err)
				require.Equal(t, "project does not have a deployment bundle", err.Error())
			},
		},
		{
			name: "no project",
			files: map[string]string{
				"project1/blueprint.cue": `version: "1.0"`,
			},
			validate: func(t *testing.T, bundle ModuleBundle, err error) {
				require.Error(t, err)
				require.Equal(t, "project path does not exist: project", err.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			testutils.SetupFS(t, fs, tt.files)

			ss := secrets.NewSecretStore(
				map[secrets.Provider]func(*slog.Logger) (secrets.SecretProvider, error){
					secrets.ProviderLocal: func(logger *slog.Logger) (secrets.SecretProvider, error) {
						return &sm.SecretProviderMock{}, nil
					},
				},
			)

			r, err := repo.NewGitRepo("", testutils.NewNoopLogger(), repo.WithFS(fs))
			require.NoError(t, err)
			require.NoError(t, r.Init())

			bundle, err := FetchBundle(r, "project", ss, testutils.NewNoopLogger())
			tt.validate(t, bundle, err)
		})
	}
}

func TestParseBundle(t *testing.T) {
	bundle := `{
	env: "test"
	modules: {
		test: {
			instance:  "foo"
			namespace: "default"
			type:      "kcl"
		}
	}
}`
	ctx := cuecontext.New()
	result, err := ParseBundle(ctx, []byte(bundle))
	require.NoError(t, err)
	require.Equal(t, "test", result.Bundle.Env)
	require.Len(t, result.Bundle.Modules, 1)
	require.Equal(t, "foo", result.Bundle.Modules["test"].Instance)
	require.Equal(t, "default", result.Bundle.Modules["test"].Namespace)
	require.Equal(t, "kcl", result.Bundle.Modules["test"].Type)
}

func TestParseBundleValue(t *testing.T) {
	src := `{
		env: "test"
		modules: {
			test: {
				instance:  "foo"
				namespace: "default"
				type:      "kcl"
			}
		}
	}`
	ctx := cuecontext.New()
	bundle := ctx.CompileString(src)
	result, err := ParseBundleValue(bundle)
	require.NoError(t, err)
	require.Equal(t, "test", result.Bundle.Env)
	require.Len(t, result.Bundle.Modules, 1)
	require.Equal(t, "foo", result.Bundle.Modules["test"].Instance)
	require.Equal(t, "default", result.Bundle.Modules["test"].Namespace)
	require.Equal(t, "kcl", result.Bundle.Modules["test"].Type)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		mod  sp.Module
		err  bool
	}{
		{
			name: "valid",
			mod: sp.Module{
				Name:     "foo",
				Registry: "bar",
				Version:  "baz",
			},
			err: false,
		},
		{
			name: "no name",
			mod: sp.Module{
				Registry: "bar",
				Version:  "baz",
			},
			err: true,
		},
		{
			name: "no registry",
			mod: sp.Module{
				Name:    "foo",
				Version: "baz",
			},
			err: true,
		},
		{
			name: "no version",
			mod: sp.Module{
				Name:     "foo",
				Registry: "bar",
			},
			err: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.mod)
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func makeBlueprint() string {
	return `
		{
			version: "1.0"
			project: {
				name: "project"
				deployment: {
					on: {}
					bundle: {
						env: "test"
						modules: {
							main: {
								name: "module"
								version: "v1.0.0"
								values: {
									foo: "bar"
								}
							}
						}
					}
				}
			}
			global: {
				deployment: {
				    foundry: api: "https://foundry.com"
					registries: {
						containers: "registry.com"
						modules: "registry.com"
					}
					repo: {
						ref: "main"
						url: "github.com/org/repo"
					}
					root: "root"
				}
				repo: {
					defaultBranch: "master"
					name: "org/repo"
					url: "https://github.com/org/repo"
				}
			}
		}
	`
}

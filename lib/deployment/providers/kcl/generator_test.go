package kcl

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/external/kcl"
	"github.com/input-output-hk/catalyst-forge/lib/external/kcl/mocks"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKCLManifestGeneratorGenerate(t *testing.T) {
	type testResult struct {
		conf kcl.ModuleConfig
		err  error
		out  []byte
		path string
	}

	tests := []struct {
		name     string
		module   sp.Module
		env      string
		out      string
		files    map[string]string
		err      bool
		validate func(t *testing.T, result testResult)
	}{
		{
			name: "full",
			module: sp.Module{
				Instance:  "instance",
				Name:      "module",
				Namespace: "default",
				Registry:  "registry",
				Values:    "test",
				Version:   "1.0.0",
			},
			env: "test",
			out: "output",
			err: false,
			validate: func(t *testing.T, result testResult) {
				require.NoError(t, result.err)
				assert.Equal(t, kcl.ModuleConfig{
					Env:       "test",
					Instance:  "instance",
					Namespace: "default",
					Name:      "module",
					Values:    "test",
					Version:   "1.0.0",
				}, result.conf)
				assert.Equal(t, []byte("output"), result.out)
				assert.Equal(t, "oci://registry/module?tag=1.0.0", result.path)
			},
		},
		{
			name: "with local path",
			module: sp.Module{
				Instance:  "instance",
				Namespace: "default",
				Path:      "/mod",
				Values:    "test",
			},
			env: "test",
			out: "output",
			files: map[string]string{
				"/mod/kcl.mod": `
[package]
name = "module"
edition = "v0.11.0"
version = "1.0.0"
`,
			},
			err: false,
			validate: func(t *testing.T, result testResult) {
				require.NoError(t, result.err)
				assert.Equal(t, kcl.ModuleConfig{
					Env:       "test",
					Instance:  "instance",
					Name:      "module",
					Namespace: "default",
					Values:    "test",
					Version:   "1.0.0",
				}, result.conf)
				assert.Equal(t, []byte("output"), result.out)
				assert.Equal(t, "/mod", result.path)
			},
		},
		{
			name: "error",
			module: sp.Module{
				Instance:  "instance",
				Name:      "module",
				Namespace: "default",
				Registry:  "registry",
				Values:    "test",
				Version:   "1.0.0",
			},
			env: "test",
			out: "output",
			err: true,
			validate: func(t *testing.T, result testResult) {
				assert.Error(t, result.err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p string
			var c kcl.ModuleConfig
			m := &mocks.ClientMock{
				RunFunc: func(path string, conf kcl.ModuleConfig) (string, error) {
					p = path
					c = conf

					if tt.err {
						return "", fmt.Errorf("error")
					}

					return tt.out, nil
				},
			}

			fs := billy.NewInMemoryFs()
			if tt.files != nil {
				testutils.SetupFS(t, fs, tt.files)
			}

			g := &KCLManifestGenerator{
				client: m,
				fs:     fs,
				logger: testutils.NewNoopLogger(),
			}

			out, err := g.Generate(tt.module, getRaw(tt.module), tt.env)
			tt.validate(t, testResult{
				conf: c,
				err:  err,
				out:  out,
				path: p,
			})
		})
	}
}

func getRaw(m sp.Module) cue.Value {
	ctx := cuecontext.New()
	return ctx.Encode(m)
}

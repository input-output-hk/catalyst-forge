package injector

import (
	"os"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlueprintEnvInjectorInject(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name     string
		in       cue.Value
		env      map[string]string
		validate func(t *testing.T, out cue.Value)
	}{
		{
			name: "string",
			in: ctx.CompileString(`
{
	foo: _ @env(name="FOO",type="string")
}
			`),
			env: map[string]string{
				"FOO": "bar",
			},
			validate: func(t *testing.T, out cue.Value) {
				require.NoError(t, out.Validate(cue.Concrete(true)))

				v := out.LookupPath(cue.ParsePath("foo"))
				sv, err := v.String()
				require.NoError(t, err)
				assert.Equal(t, "bar", sv)
			},
		},
		{
			name: "int",
			in: ctx.CompileString(`
{
	foo: _ @env(name="FOO",type="int")
}
			`),
			env: map[string]string{
				"FOO": "1",
			},
			validate: func(t *testing.T, out cue.Value) {
				require.NoError(t, out.Validate(cue.Concrete(true)))

				v := out.LookupPath(cue.ParsePath("foo"))
				iv, err := v.Int64()
				require.NoError(t, err)
				assert.Equal(t, int64(1), iv)
			},
		},
		{
			name: "bool",
			in: ctx.CompileString(`
{
	foo: _ @env(name="FOO",type="bool")
}
			`),
			env: map[string]string{
				"FOO": "true",
			},
			validate: func(t *testing.T, out cue.Value) {
				require.NoError(t, out.Validate(cue.Concrete(true)))

				v := out.LookupPath(cue.ParsePath("foo"))
				bv, err := v.Bool()
				require.NoError(t, err)
				assert.Equal(t, true, bv)
			},
		},
		{
			name: "bad int",
			in: ctx.CompileString(`
{
	foo: _ @env(name="FOO",type="int")
}
			`),
			env: map[string]string{
				"FOO": "bar",
			},
			validate: func(t *testing.T, out cue.Value) {
				assert.Error(t, out.Validate(cue.Concrete(true)))
			},
		},
		{
			name: "bad type",
			in: ctx.CompileString(`
{
	foo: _ @env(name="FOO",type="foo")
}
			`),
			env: map[string]string{},
			validate: func(t *testing.T, out cue.Value) {
				assert.Error(t, out.Validate(cue.Concrete(true)))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				for k := range tt.env {
					require.NoError(t, os.Unsetenv(k))
				}
			}()

			for k, v := range tt.env {
				require.NoError(t, os.Setenv(k, v))
			}

			injector := NewBlueprintEnvInjector(ctx, testutils.NewNoopLogger())
			bp := blueprint.NewRawBlueprint(tt.in)
			out := injector.Inject(bp)

			tt.validate(t, out.Value())
		})
	}
}

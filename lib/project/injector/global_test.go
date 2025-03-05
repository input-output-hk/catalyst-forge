package injector

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlueprintGlobalInjectorInject(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name     string
		in       cue.Value
		validate func(t *testing.T, out cue.Value)
	}{
		{
			name: "simple",
			in: ctx.CompileString(`
{
	global: {
		foo: "bar"
	}
	foo: _ @global(name="foo")
}
			`),
			validate: func(t *testing.T, out cue.Value) {
				require.NoError(t, out.Validate(cue.Concrete(true)))

				v := out.LookupPath(cue.ParsePath("foo"))
				sv, err := v.String()
				require.NoError(t, err)
				assert.Equal(t, "bar", sv)
			},
		},
		{
			name: "nested",
			in: ctx.CompileString(`
{
	global: {
		foo: {
			bar: "baz"
		}
	}
	foo: _ @global(name="foo.bar")
}
			`),
			validate: func(t *testing.T, out cue.Value) {
				require.NoError(t, out.Validate(cue.Concrete(true)))

				v := out.LookupPath(cue.ParsePath("foo"))
				sv, err := v.String()
				require.NoError(t, err)
				assert.Equal(t, "baz", sv)
			},
		},
		{
			name: "complex type",
			in: ctx.CompileString(`
{
	global: {
		foo: {
			bar: [
				{
					baz: "baz"
				}
			]
		}
	}
	foo: _ @global(name="foo")
}
			`),
			validate: func(t *testing.T, out cue.Value) {
				require.NoError(t, out.Validate(cue.Concrete(true)))

				v := out.LookupPath(cue.ParsePath("foo.bar[0].baz"))
				sv, err := v.String()
				require.NoError(t, err)
				assert.Equal(t, "baz", sv)
			},
		},
		{
			name: "does not exist",
			in: ctx.CompileString(`
{
	global: {
		bar: "baz"
	}
	foo: _ @global(name="foo")
}
			`),
			validate: func(t *testing.T, out cue.Value) {
				require.Error(t, out.Validate(cue.Concrete(true)))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injector := NewBlueprintGlobalInjector(ctx, testutils.NewNoopLogger())
			bp := blueprint.NewRawBlueprint(tt.in)
			out := injector.Inject(bp)

			tt.validate(t, out.Value())
		})
	}
}

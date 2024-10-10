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

func TestBlueprintRuntimeInjectorInject(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name     string
		in       cue.Value
		data     map[string]cue.Value
		validate func(t *testing.T, out cue.Value)
	}{
		{
			name: "simple",
			in: ctx.CompileString(`
{
	foo: _ @forge(name="FOO")
}
			`),
			data: map[string]cue.Value{
				"FOO": ctx.CompileString(`"bar"`),
			},
			validate: func(t *testing.T, out cue.Value) {
				require.NoError(t, out.Validate(cue.Concrete(true)))

				v := out.LookupPath(cue.ParsePath("foo"))
				sv, err := v.String()
				require.NoError(t, err)
				assert.Equal(t, "bar", sv)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injector := NewBlueprintRuntimeInjector(ctx, tt.data, testutils.NewNoopLogger())
			bp := blueprint.NewRawBlueprint(tt.in)
			out := injector.Inject(bp)

			tt.validate(t, out.Value())
		})
	}
}

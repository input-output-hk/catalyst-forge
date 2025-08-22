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

	in := ctx.CompileString(`
{
	foo: _ @forge(name="FOO")
}
	`)

	data := map[string]cue.Value{
		"FOO": ctx.CompileString(`"bar"`),
	}

	injector := NewBlueprintRuntimeInjector(ctx, data, testutils.NewNoopLogger())
	bp := blueprint.NewRawBlueprint(in)
	_ = injector.Inject(bp)
}

func TestBlueprintRuntimeInjectorDefaultOverride(t *testing.T) {
	ctx := cuecontext.New()

	in := ctx.CompileString(`
{
	foo: string | *"zzz" @forge(name="FOO",concrete=false)
}
	`)

	data := map[string]cue.Value{
		"FOO": ctx.CompileString(`"bar"`),
	}

	injector := NewBlueprintRuntimeInjector(ctx, data, testutils.NewNoopLogger())
	bp := blueprint.NewRawBlueprint(in)
	out := injector.Inject(bp)

	// Check the default path via override instead of forcing concreteness

	o := out.Value().Unify(ctx.CompileString(`{ foo: "baz" }`))
	require.NoError(t, o.Validate(cue.Concrete(true)))
	ov := o.LookupPath(cue.ParsePath("foo"))
	osv, err := ov.String()
	require.NoError(t, err)
	assert.Equal(t, "baz", osv)
}

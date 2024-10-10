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

type mockBlueprintInjectorMap struct {
	data map[string]cue.Value
}

func (m *mockBlueprintInjectorMap) Get(ctx *cue.Context, name string, attrType AttrType) (cue.Value, error) {
	v, ok := m.data[name]
	if !ok {
		return cue.Value{}, ErrNotFound
	}
	return v, nil
}

func TestBaseInjectorInject(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name         string
		typeOptional bool
		in           cue.Value
		data         map[string]cue.Value
		validate     func(t *testing.T, out cue.Value)
	}{
		{
			name:         "simple",
			typeOptional: false,
			in: ctx.CompileString(`
{
	foo: _ @attrName(name="foo",type="string")
}
			`),
			data: map[string]cue.Value{
				"foo": ctx.CompileString(`"bar"`),
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
			name:         "no type",
			typeOptional: true,
			in: ctx.CompileString(`
{
	foo: _ @attrName(name="foo")
}
			`),
			data: map[string]cue.Value{
				"foo": ctx.CompileString(`"bar"`),
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
			name: "malformed",
			in: ctx.CompileString(`
{
	foo: _ @attrName(foo="bar")
}
			`),
			data: map[string]cue.Value{},
			validate: func(t *testing.T, out cue.Value) {
				require.Error(t, out.Err())
			},
		},
		{
			name:         "not found",
			typeOptional: true,
			in: ctx.CompileString(`
{
	foo: string | *"baz" @attrName(name="bar")
}
			`),
			data: map[string]cue.Value{},
			validate: func(t *testing.T, out cue.Value) {
				require.NoError(t, out.Validate(cue.Concrete(true)))

				v := out.LookupPath(cue.ParsePath("foo"))
				sv, err := v.String()
				require.NoError(t, err)
				assert.Equal(t, "baz", sv)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMap := &mockBlueprintInjectorMap{
				data: tt.data,
			}

			base := &BaseInjector{
				attrName:     "attrName",
				imap:         mockMap,
				logger:       testutils.NewNoopLogger(),
				typeOptional: tt.typeOptional,
			}

			bp := blueprint.NewRawBlueprint(ctx, tt.in)
			out := base.Inject(bp)

			tt.validate(t, out.Value())
		})
	}
}

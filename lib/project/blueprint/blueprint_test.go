package blueprint

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlueprintFilesUnify(t *testing.T) {
	ctx := cuecontext.New()
	compile := func(src string) cue.Value {
		return ctx.CompileString(src)
	}
	tests := []struct {
		name   string
		files  BlueprintFiles
		expect cue.Value
	}{
		{
			name:   "no file",
			files:  BlueprintFiles{},
			expect: compile("{}"),
		},
		{
			name: "single file",
			files: BlueprintFiles{
				{
					Value: compile("{a: 1}"),
				},
			},
			expect: compile("{a: 1}"),
		},
		{
			name: "multiple files",
			files: BlueprintFiles{
				{
					Value: compile("{a: 1}"),
				},
				{
					Value: compile("{b: 2}"),
				},
			},
			expect: compile("{a: 1, b: 2}"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := cuecontext.New()
			v, err := tt.files.Unify(ctx)
			require.NoError(t, err)

			expectSrc, err := format.Node(tt.expect.Syntax())
			require.NoError(t, err)

			gotSrc, err := format.Node(v.Syntax())
			require.NoError(t, err)

			assert.Equal(t, string(expectSrc), string(gotSrc))
		})
	}
}

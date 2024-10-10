package project

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitRuntimeLoad(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name     string
		tagInfo  TagInfo
		expected map[string]cue.Value
	}{
		{
			name: "with tag",
			tagInfo: TagInfo{
				Generated: "generated",
				Git:       "tag",
			},
			expected: map[string]cue.Value{
				"GIT_TAG_GENERATED": ctx.CompileString(`"generated"`),
				"GIT_TAG":           ctx.CompileString(`"tag"`),
				"GIT_IMAGE_TAG":     ctx.CompileString(`"tag"`),
			},
		},
		{
			name: "with no tag",
			tagInfo: TagInfo{
				Generated: "generated",
				Git:       "",
			},
			expected: map[string]cue.Value{
				"GIT_TAG_GENERATED": ctx.CompileString(`"generated"`),
				"GIT_IMAGE_TAG":     ctx.CompileString(`"generated"`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutils.NewNoopLogger()
			project := &Project{
				TagInfo:      tt.tagInfo,
				rawBlueprint: blueprint.NewRawBlueprint(ctx, ctx.CompileString("{}")),
			}

			runtime := NewGitRuntime(logger)
			data := runtime.Load(project)

			for key, expected := range tt.expected {
				actual, ok := data[key]
				require.True(t, ok, "missing value for key %q", key)

				sv, err := actual.String()
				require.NoError(t, err)

				ev, err := expected.String()
				require.NoError(t, err)

				assert.Equal(t, ev, sv)
			}
		})
	}
}

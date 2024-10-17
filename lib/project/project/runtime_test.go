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
		tagInfo  *TagInfo
		repoPath string
		prjPath  string
		expected map[string]cue.Value
	}{
		{
			name: "with tag",
			tagInfo: &TagInfo{
				Generated: "generated",
				Git:       "v1.0.0",
			},
			repoPath: "/repo",
			prjPath:  "/repo/project",
			expected: map[string]cue.Value{
				"GIT_TAG_GENERATED": ctx.CompileString(`"generated"`),
				"GIT_TAG":           ctx.CompileString(`"v1.0.0"`),
				"GIT_IMAGE_TAG":     ctx.CompileString(`"v1.0.0"`),
			},
		},
		{
			name: "with mono tag",
			tagInfo: &TagInfo{
				Generated: "generated",
				Git:       "project/v1.0.0",
			},
			repoPath: "/repo",
			prjPath:  "/repo/project",
			expected: map[string]cue.Value{
				"GIT_TAG_GENERATED": ctx.CompileString(`"generated"`),
				"GIT_TAG":           ctx.CompileString(`"v1.0.0"`),
				"GIT_IMAGE_TAG":     ctx.CompileString(`"v1.0.0"`),
			},
		},
		{
			name: "with non-matching tag",
			tagInfo: &TagInfo{
				Generated: "generated",
				Git:       "project1/v1.0.0",
			},
			repoPath: "/repo",
			prjPath:  "/repo/project",
			expected: map[string]cue.Value{
				"GIT_TAG_GENERATED": ctx.CompileString(`"generated"`),
				"GIT_IMAGE_TAG":     ctx.CompileString(`"generated"`),
			},
		},
		{
			name: "with no tag",
			tagInfo: &TagInfo{
				Generated: "generated",
				Git:       "",
			},
			expected: map[string]cue.Value{
				"GIT_TAG_GENERATED": ctx.CompileString(`"generated"`),
				"GIT_IMAGE_TAG":     ctx.CompileString(`"generated"`),
			},
		},
		{
			name:     "with no tag info",
			tagInfo:  nil,
			expected: map[string]cue.Value{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutils.NewNoopLogger()
			project := &Project{
				ctx:          ctx,
				RepoRoot:     tt.repoPath,
				Path:         tt.prjPath,
				RawBlueprint: blueprint.NewRawBlueprint(ctx.CompileString("{}")),
				TagInfo:      tt.tagInfo,
				logger:       logger,
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

				assert.Equal(t, ev, sv, "unexpected value for key %q", key)
			}
		})
	}
}

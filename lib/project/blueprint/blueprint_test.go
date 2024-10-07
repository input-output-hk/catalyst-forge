package blueprint

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"github.com/Masterminds/semver/v3"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
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

func TestBlueprintFilesValidateMajorVersions(t *testing.T) {
	tests := []struct {
		name      string
		files     BlueprintFiles
		expectErr bool
	}{
		{
			name: "same major versions",
			files: BlueprintFiles{
				{
					Version: semver.MustParse("1.0.0"),
				},
				{
					Version: semver.MustParse("1.1.0"),
				},
			},
			expectErr: false,
		},
		{
			name: "different major versions",
			files: BlueprintFiles{
				{
					Version: semver.MustParse("1.0.0"),
				},
				{
					Version: semver.MustParse("2.0.0"),
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.files.ValidateMajorVersions()
			testutils.AssertError(t, err, tt.expectErr, "")
		})
	}
}

func TestBlueprintFilesVersion(t *testing.T) {
	tests := []struct {
		name   string
		files  BlueprintFiles
		expect *semver.Version
	}{
		{
			name:   "no files",
			files:  BlueprintFiles{},
			expect: nil,
		},
		{
			name: "single file",
			files: BlueprintFiles{
				{
					Version: semver.MustParse("1.0.0"),
				},
			},
			expect: semver.MustParse("1.0.0"),
		},
		{
			name: "multiple files",
			files: BlueprintFiles{
				{
					Version: semver.MustParse("1.0.0"),
				},
				{
					Version: semver.MustParse("1.1.0"),
				},
				{
					Version: semver.MustParse("2.0.0"),
				},
			},
			expect: semver.MustParse("2.0.0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.files.Version()
			assert.Condition(t, func() bool {
				return (got == nil && tt.expect == nil) || got.Equal(tt.expect)
			})
		})
	}
}

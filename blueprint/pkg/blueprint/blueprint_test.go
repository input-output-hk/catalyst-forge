package blueprint

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"github.com/Masterminds/semver/v3"
	"github.com/input-output-hk/catalyst-forge/blueprint/internal/testutils"
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
			if err != nil {
				t.Fatalf("failed to unify: %v", err)
			}

			expectSrc, err := format.Node(tt.expect.Syntax())
			if err != nil {
				t.Fatalf("failed to format expect: %v", err)
			}

			gotSrc, err := format.Node(v.Syntax())
			if err != nil {
				t.Fatalf("failed to format got: %v", err)
			}

			if string(gotSrc) != string(expectSrc) {
				t.Errorf("got %s, want %s", gotSrc, expectSrc)
			}
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
			if r, err := testutils.CheckError(t, err, tt.expectErr, nil); r || err != nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
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
			if got == nil && tt.expect == nil {
				return
			} else if got == nil || tt.expect == nil {
				t.Fatalf("got %v, want %v", got, tt.expect)
			} else if !got.Equal(tt.expect) {
				t.Errorf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

package version

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/Masterminds/semver/v3"
	"github.com/input-output-hk/catalyst-forge/blueprint/internal/testutils"
	cuetools "github.com/input-output-hk/catalyst-forge/cuetools/pkg"
)

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expect    string
		expectErr bool
	}{
		{
			name: "version present",
			input: `
			version: "1.0"
			`,
			expect:    "1.0.0",
			expectErr: false,
		},
		{
			name: "version not present",
			input: `
			foo: "bar"
			`,
			expect:    "",
			expectErr: true,
		},
		{
			name: "version invalid",
			input: `
			version: "foobar"
			`,
			expect:    "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := cuetools.Compile(cuecontext.New(), []byte(tt.input))
			if err != nil {
				t.Fatalf("failed to compile cue: %v", err)
			}

			got, err := GetVersion(v)
			if r, err := testutils.CheckError(t, err, tt.expectErr, nil); r || err != nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if got.String() != tt.expect {
				t.Errorf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestValidateVersions(t *testing.T) {
	tests := []struct {
		name        string
		user        *semver.Version
		schema      *semver.Version
		expectErr   bool
		expectedErr error
	}{
		{
			name:        "blueprint major version greater than schema",
			user:        semver.MustParse("2.0.0"),
			schema:      semver.MustParse("1.0.0"),
			expectErr:   true,
			expectedErr: ErrMajorMismatch,
		},
		{
			name:        "blueprint minor version greater than schema",
			user:        semver.MustParse("1.1.0"),
			schema:      semver.MustParse("1.0.0"),
			expectErr:   true,
			expectedErr: ErrMinorMismatch,
		},
		{
			name:        "blueprint major version equal to schema",
			user:        semver.MustParse("1.0.0"),
			schema:      semver.MustParse("1.0.0"),
			expectErr:   false,
			expectedErr: nil,
		},
		{
			name:        "schema major greater than blueprint",
			user:        semver.MustParse("1.0.0"),
			schema:      semver.MustParse("2.0.0"),
			expectErr:   true,
			expectedErr: ErrMajorMismatch,
		},
		{
			name:        "schema minor greater than blueprint",
			user:        semver.MustParse("1.0.0"),
			schema:      semver.MustParse("1.1.0"),
			expectErr:   false,
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVersions(tt.user, tt.schema)
			if r, err := testutils.CheckError(t, err, tt.expectErr, tt.expectedErr); r || err != nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
		})
	}
}

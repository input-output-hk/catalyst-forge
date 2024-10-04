package version

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/Masterminds/semver/v3"
	cuetools "github.com/input-output-hk/catalyst-forge/lib/tools/pkg/cue"
	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.NoError(t, err, "unexpected error compiling CUE: %v", err)

			got, err := GetVersion(v)
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			assert.Equal(t, tt.expect, got.String())
		})
	}
}

func TestValidateVersions(t *testing.T) {
	tests := []struct {
		name        string
		user        *semver.Version
		schema      *semver.Version
		expectErr   bool
		expectedErr string
	}{
		{
			name:        "blueprint major version greater than schema",
			user:        semver.MustParse("2.0.0"),
			schema:      semver.MustParse("1.0.0"),
			expectErr:   true,
			expectedErr: ErrMajorMismatch.Error(),
		},
		{
			name:        "blueprint minor version greater than schema",
			user:        semver.MustParse("1.1.0"),
			schema:      semver.MustParse("1.0.0"),
			expectErr:   true,
			expectedErr: ErrMinorMismatch.Error(),
		},
		{
			name:        "blueprint major version equal to schema",
			user:        semver.MustParse("1.0.0"),
			schema:      semver.MustParse("1.0.0"),
			expectErr:   false,
			expectedErr: "",
		},
		{
			name:        "schema major greater than blueprint",
			user:        semver.MustParse("1.0.0"),
			schema:      semver.MustParse("2.0.0"),
			expectErr:   true,
			expectedErr: ErrMajorMismatch.Error(),
		},
		{
			name:        "schema minor greater than blueprint",
			user:        semver.MustParse("1.0.0"),
			schema:      semver.MustParse("1.1.0"),
			expectErr:   false,
			expectedErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVersions(tt.user, tt.schema)
			testutils.AssertError(t, err, tt.expectErr, tt.expectedErr)
		})
	}
}

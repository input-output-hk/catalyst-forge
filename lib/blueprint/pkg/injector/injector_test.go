package injector

import (
	"io"
	"log/slog"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/blueprint/pkg/injector/mocks"
	cuetools "github.com/input-output-hk/catalyst-forge/lib/tools/pkg/cue"
	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectEnv(t *testing.T) {
	tests := []struct {
		name          string
		raw           string
		env           map[string]string
		ovr           map[string]string
		path          string
		expectedType  envType
		expectedValue any
		expectErr     bool
	}{
		{
			name: "string env",
			raw: `
				foo: string | *"test" @env(name=FOO,type=string)
			`,
			env: map[string]string{
				"FOO": "bar",
			},
			ovr:           nil,
			path:          "foo",
			expectedType:  EnvTypeString,
			expectedValue: "bar",
			expectErr:     false,
		},
		{
			name: "int env",
			raw: `
				foo: int & >2 @env(name=FOO,type=int)
			`,
			env: map[string]string{
				"FOO": "3",
			},
			ovr:           nil,
			path:          "foo",
			expectedType:  EnvTypeInt,
			expectedValue: int64(3),
			expectErr:     false,
		},
		{
			name: "bool env",
			raw: `
				foo: bool @env(name=FOO,type=bool)
			`,
			env: map[string]string{
				"FOO": "true",
			},
			ovr:           nil,
			path:          "foo",
			expectedType:  EnvTypeBool,
			expectedValue: true,
			expectErr:     false,
		},
		{
			name: "override",
			raw: `
				foo: string | *"test" @env(name=FOO,type=string)
			`,
			env: map[string]string{
				"FOO": "bar",
			},
			ovr: map[string]string{
				"FOO": "baz",
			},
			path:          "foo",
			expectedType:  EnvTypeString,
			expectedValue: "baz",
			expectErr:     false,
		},
		{
			name: "override no env",
			raw: `
				foo: string | *"test" @env(name=FOO,type=string)
			`,
			ovr: map[string]string{
				"FOO": "baz",
			},
			path:          "foo",
			expectedType:  EnvTypeString,
			expectedValue: "baz",
			expectErr:     false,
		},
		{
			name: "bad int",
			raw: `
				foo: int @env(name=FOO,type=int)
			`,
			env: map[string]string{
				"FOO": "foo",
			},
			ovr:           nil,
			path:          "foo",
			expectedType:  EnvTypeInt,
			expectedValue: true,
			expectErr:     true,
		},
		{
			name: "mismatched types",
			raw: `
				foo: int @env(name=FOO,type=string)
			`,
			env: map[string]string{
				"FOO": "foo",
			},
			ovr:           nil,
			path:          "foo",
			expectedType:  EnvTypeString,
			expectedValue: true,
			expectErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := cuetools.Compile(cuecontext.New(), []byte(tt.raw))
			assert.NoError(t, err)

			i := Injector{
				logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
				getter: &mocks.EnvGetterMock{
					GetFunc: func(key string) (string, bool) {
						return tt.env[key], true
					},
				},
			}
			v = i.InjectEnv(v, tt.ovr)

			err = v.Validate(cue.Concrete(true))
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			switch tt.expectedType {
			case EnvTypeString:
				got, err := v.LookupPath(cue.ParsePath(tt.path)).String()
				assert.NoError(t, err, "expected value to be string")
				assert.Equal(t, tt.expectedValue, got)
			case EnvTypeInt:
				got, err := v.LookupPath(cue.ParsePath(tt.path)).Int64()
				assert.NoError(t, err, "expected value to be int")
				assert.Equal(t, tt.expectedValue, got)
			case EnvTypeBool:
				got, err := v.LookupPath(cue.ParsePath(tt.path)).Bool()
				assert.NoError(t, err, "expected value to be bool")
				assert.Equal(t, tt.expectedValue, got)
			}
		})
	}
}

func Test_findEnvAttr(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		path         string
		expectedBody string
	}{
		{
			name: "env attribute found",
			raw: `
				foo: string | *"test" @env(name=FOO,type=string)
			`,
			path:         "foo",
			expectedBody: `name=FOO,type=string`,
		},
		{
			name: "env attribute not found",
			raw: `
				foo: string | *"test"
			`,
			path:         "",
			expectedBody: "",
		},
		{
			name: "different attribute found",
			raw: `
				foo: string | *"test" @bar(name=FOO,type=string)
			`,
			path:         "",
			expectedBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := cuetools.Compile(cuecontext.New(), []byte(tt.raw))
			require.NoError(t, err)

			attr := findEnvAttr(v.LookupPath(cue.ParsePath(tt.path)))
			if attr == nil && tt.expectedBody != "" {
				t.Fatalf("expected to find env attribute")
			} else if attr == nil && tt.expectedBody == "" {
				return
			}

			assert.Equal(t, attr.Contents(), tt.expectedBody)
		})
	}
}

func Test_parseEnvAttr(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		expected  envAttr
		expectErr bool
	}{
		{
			name: "env attribute parsed",
			raw: `
				foo: string | *"test" @env(name=FOO,type=string)
			`,
			expected: envAttr{
				name:    "FOO",
				envType: "string",
			},
			expectErr: false,
		},
		{
			name: "malformed keys",
			raw: `
				foo: string | *"test" @env(names=FOO,types=string)
			`,
			expected:  envAttr{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := cuetools.Compile(cuecontext.New(), []byte(tt.raw))
			require.NoError(t, err)

			attr := findEnvAttr(v.LookupPath(cue.ParsePath("foo")))
			require.NotNil(t, attr, "expected to find env attribute")

			env, err := parseEnvAttr(attr)
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			assert.Equal(t, tt.expected.name, env.name)
			assert.Equal(t, tt.expected.envType, env.envType)
		})
	}
}

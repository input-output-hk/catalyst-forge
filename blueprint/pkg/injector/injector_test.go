package injector

import (
	"io"
	"log/slog"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cuetools "github.com/input-output-hk/catalyst-forge/cuetools/pkg"
)

func TestInjectEnv(t *testing.T) {
	tests := []struct {
		name          string
		raw           string
		env           map[string]string
		path          string
		expectedType  envType
		expectedValue any
		expectValid   bool
	}{
		{
			name: "string env",
			raw: `
				foo: string | *"test" @env(name=FOO,type=string)
			`,
			env: map[string]string{
				"FOO": "bar",
			},
			path:          "foo",
			expectedType:  EnvTypeString,
			expectedValue: "bar",
			expectValid:   true,
		},
		{
			name: "int env",
			raw: `
				foo: int & >2 @env(name=FOO,type=int)
			`,
			env: map[string]string{
				"FOO": "3",
			},
			path:          "foo",
			expectedType:  EnvTypeInt,
			expectedValue: int64(3),
			expectValid:   true,
		},
		{
			name: "bool env",
			raw: `
				foo: bool @env(name=FOO,type=bool)
			`,
			env: map[string]string{
				"FOO": "true",
			},
			path:          "foo",
			expectedType:  EnvTypeBool,
			expectedValue: true,
			expectValid:   true,
		},
		{
			name: "bad int",
			raw: `
				foo: int @env(name=FOO,type=int)
			`,
			env: map[string]string{
				"FOO": "foo",
			},
			path:          "foo",
			expectedType:  EnvTypeInt,
			expectedValue: true,
			expectValid:   false,
		},
		{
			name: "mismatched types",
			raw: `
				foo: int @env(name=FOO,type=string)
			`,
			env: map[string]string{
				"FOO": "foo",
			},
			path:          "foo",
			expectedType:  EnvTypeString,
			expectedValue: true,
			expectValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := cuetools.Compile(cuecontext.New(), []byte(tt.raw))
			if err != nil {
				t.Fatalf("failed to compile CUE: %v", err)
			}

			i := Injector{
				logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
				getter: &EnvGetterMock{
					GetFunc: func(key string) (string, bool) {
						return tt.env[key], true
					},
				},
			}
			v = i.InjectEnv(v)

			err = v.Validate(cue.Concrete(true))
			if err != nil && tt.expectValid {
				t.Fatalf("expected value to be invalid, got %v", err)
			} else if err == nil && !tt.expectValid {
				t.Fatalf("expected value to be valid, got none")
			} else if err != nil && !tt.expectValid {
				return
			}

			switch tt.expectedType {
			case EnvTypeString:
				got, err := v.LookupPath(cue.ParsePath(tt.path)).String()
				if err != nil {
					t.Fatal("expected value to be string")
				}

				if got != tt.expectedValue {
					t.Errorf("expected value to be %v, got %v", tt.expectedValue, got)
				}
			case EnvTypeInt:
				got, err := v.LookupPath(cue.ParsePath(tt.path)).Int64()
				if err != nil {
					t.Fatal("expected value to be int")
				}

				if got != tt.expectedValue {
					t.Fatalf("expected value to be %v, got %v", tt.expectedValue, got)
				}
			case EnvTypeBool:
				got, err := v.LookupPath(cue.ParsePath(tt.path)).Bool()
				if err != nil {
					t.Fatal("expected value to be bool")
				}

				if got != tt.expectedValue {
					t.Fatalf("expected value to be %v, got %v", tt.expectedValue, got)
				}
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
			if err != nil {
				t.Fatalf("failed to compile CUE: %v", err)
			}

			attr := findEnvAttr(v.LookupPath(cue.ParsePath(tt.path)))
			if attr == nil && tt.expectedBody != "" {
				t.Fatalf("expected to find env attribute")
			} else if attr == nil && tt.expectedBody == "" {
				return
			}

			if got := attr.Contents(); got != tt.expectedBody {
				t.Errorf("expected body to be %s, got %s", tt.expectedBody, got)
			}
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
			if err != nil {
				t.Fatalf("failed to compile CUE: %v", err)
			}

			attr := findEnvAttr(v.LookupPath(cue.ParsePath("foo")))
			if attr == nil {
				t.Fatalf("expected to find env attribute")
			}

			env, err := parseEnvAttr(attr)
			if err != nil && tt.expectErr {
				return
			} else if err != nil && !tt.expectErr {
				t.Fatalf("expected no error, got %v", err)
			} else if err == nil && tt.expectErr {
				t.Fatalf("expected error, got none")
			}

			if env.name != tt.expected.name {
				t.Errorf("expected name to be %s, got %s", tt.expected.name, env.name)
			}

			if env.envType != tt.expected.envType {
				t.Errorf("expected envType to be %s, got %s", tt.expected.envType, env.envType)
			}
		})
	}
}

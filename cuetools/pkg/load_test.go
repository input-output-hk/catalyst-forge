package pkg

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

func TestCompile(t *testing.T) {
	ctx := cuecontext.New()
	tests := []struct {
		name        string
		contents    string
		expectedVal cue.Value
		expectedErr string
	}{
		{
			name:        "valid contents",
			contents:    "{}",
			expectedVal: ctx.CompileString("{}"),
			expectedErr: "",
		},
		{
			name:        "invalid contents",
			contents:    "{a: b}",
			expectedVal: cue.Value{},
			expectedErr: "a: reference \"b\" not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := Compile(ctx, []byte(tt.contents))
			if err != nil && err.Error() != tt.expectedErr {
				t.Errorf("unexpected error: %v", err)
				return
			} else if err == nil && tt.expectedErr != "" {
				t.Errorf("expected error %q but got nil", tt.expectedErr)
				return
			} else if err != nil && err.Error() == tt.expectedErr {
				return
			}

			if !v.Equals(tt.expectedVal) {
				t.Errorf("expected value %v, got %v", tt.expectedVal, v)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	ctx := cuecontext.New()
	tests := []struct {
		name        string
		v           cue.Value
		expectedErr string
	}{
		{
			name:        "valid value",
			v:           ctx.CompileString("{}"),
			expectedErr: "",
		},
		{
			name:        "invalid value",
			v:           ctx.CompileString("{a: 1}").FillPath(cue.ParsePath("a"), fmt.Errorf("invalid value")),
			expectedErr: "failed to validate: invalid value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.v)
			if err != nil && err.Error() != tt.expectedErr {
				t.Errorf("unexpected error: %v", err)
			} else if err == nil && tt.expectedErr != "" {
				t.Errorf("expected error %q but got nil", tt.expectedErr)
			}
		})
	}
}

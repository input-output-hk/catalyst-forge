package cue

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/testutils"
	"github.com/stretchr/testify/assert"
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
			if testutils.AssertError(t, err, tt.expectedErr != "", tt.expectedErr) {
				return
			}

			assert.True(t, v.Equals(tt.expectedVal))
		})
	}
}

func TestValidate(t *testing.T) {
	ctx := cuecontext.New()
	tests := []struct {
		name        string
		v           cue.Value
		expectErr   bool
		expectedErr string
	}{
		{
			name:        "valid value",
			v:           ctx.CompileString("{}"),
			expectErr:   false,
			expectedErr: "",
		},
		{
			name:        "invalid value",
			v:           ctx.CompileString("{a: 1}").FillPath(cue.ParsePath("a"), fmt.Errorf("invalid value")),
			expectErr:   true,
			expectedErr: "failed to validate: invalid value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.v)
			testutils.AssertError(t, err, tt.expectErr, tt.expectedErr)
		})
	}
}

package cue

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
)

func TestDelete(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		path      string
		expected  string
		expectErr bool
	}{
		{
			name: "delete field",
			value: `
{
	a: 1
	b: 2
}`,
			path: "a",
			expected: `
{
	b: 2
}`,
		},
		{
			name: "delete nested field",
			value: `
{
	a: {
		b: 1
		c: 2
	}
}`,
			path: "a.b",
			expected: `
{
	a: {
		c: 2
	}
}`,
			expectErr: false,
		},
		{
			name: "delete list element",
			value: `
{
	a: [1, 2, 3]
}`,
			path: "a[1]",
			expected: `
{
	a: [1, ...] & [_, 3, ...] & {
		[...]
	}
}`,
			expectErr: false,
		},
		{
			name: "delete nested list element",
			value: `
{
	a: [
			{
				b: [1, 2, 3]
			}
		]
}`,
			path: "a[0].b[1]",
			expected: `
{
	a: [{
		b: [1, ...] & [_, 3, ...] & {
			[...]
		}
	}, ...] & {
		[...]
	}
}`,
			expectErr: false,
		},
		{
			name: "delete non-existent field",
			value: `
{
	a: 1
}`,
			path:      "b",
			expected:  "",
			expectErr: true,
		},
		{
			name: "delete non-existent index",
			value: `
{
	a: [1, 2, 3]
}`,
			path:      "a[3]",
			expected:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := cuecontext.New()
			v, err := Compile(ctx, []byte(tt.value))
			assert.NoError(t, err)

			final, err := Delete(ctx, v, tt.path)
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			src, err := format.Node(final.Syntax())
			assert.NoError(t, err)

			assert.Equal(t, fmt.Sprintf("\n%s", src), tt.expected)
		})
	}
}

func TestReplace(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		path      string
		replace   string
		expected  string
		expectErr bool
	}{
		{
			name: "replace field",
			value: `
{
	a: 1
	b: 2
}`,
			path:    "a",
			replace: "3",
			expected: `
{
	a: 3
	b: 2
}`,
			expectErr: false,
		},
		{
			name: "replace nested field",
			value: `
{
	a: {
		b: 1
		c: 2
	}
}`,
			path:    "a.b",
			replace: "3",
			expected: `
{
	a: {
		b: 3
		c: 2
	}
}`,
			expectErr: false,
		},
		{
			name: "replace list element",
			value: `
{
	a: [1, 2, 3]
}`,
			path:    "a[1]",
			replace: "4",
			expected: `
{
	a: [1, ...] & [_, 4, ...] & [_, _, 3, ...] & {
		[...]
	}
}`,
			expectErr: false,
		},
		{
			name: "replace nested list element",
			value: `
{
	a: [
			{
				b: [1, 2, 3]
			}
		]
}`,
			path:    "a[0].b[1]",
			replace: "4",
			expected: `
{
	a: [{
		b: [1, ...] & [_, 4, ...] & [_, _, 3, ...] & {
			[...]
		}
	}, ...] & {
		[...]
	}
}`,
			expectErr: false,
		},
		{
			name: "replace non-existent field",
			value: `
{
	a: 1
}`,
			path:      "b",
			replace:   "2",
			expected:  "",
			expectErr: true,
		},
		{
			name: "replace non-existent index",
			value: `
{
	a: [1, 2, 3]
}`,
			path:      "a[3]",
			replace:   "4",
			expected:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := cuecontext.New()
			v, err := Compile(ctx, []byte(tt.value))
			assert.NoError(t, err)

			replace, err := Compile(ctx, []byte(tt.replace))
			assert.NoError(t, err)

			final, err := Replace(ctx, v, tt.path, replace)

			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			src, err := format.Node(final.Syntax())
			assert.NoError(t, err)

			assert.Equal(t, fmt.Sprintf("\n%s", src), tt.expected)
		})
	}
}

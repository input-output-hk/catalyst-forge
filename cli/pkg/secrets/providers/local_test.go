package providers

import (
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestLocalClientGet(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		files       map[string]string
		expect      string
		expectErr   bool
		expectedErr string
	}{
		{
			name: "simple",
			key:  ".secrets",
			files: map[string]string{
				".secrets": "secret",
			},
			expect:      "secret",
			expectErr:   false,
			expectedErr: "",
		},
		{
			name: "file not found",
			key:  "foo",
			files: map[string]string{
				".secrets": "secret",
			},
			expect:      "",
			expectErr:   true,
			expectedErr: "open foo: file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			for k, v := range tt.files {
				afero.WriteFile(fs, k, []byte(v), 0644)
			}

			client := &LocalClient{
				fs:     fs,
				logger: testutils.NewNoopLogger(),
			}

			got, err := client.Get(tt.key)
			if testutils.AssertError(t, err, tt.expectErr, tt.expectedErr) {
				return
			}
			assert.Equal(t, tt.expect, got)
		})
	}
}

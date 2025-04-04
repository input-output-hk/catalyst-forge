package providers

import (
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
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
			expectedErr: "file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			for k, v := range tt.files {
				fs.WriteFile(k, []byte(v), 0644)
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

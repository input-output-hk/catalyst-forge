package providers

import (
	"fmt"
	"testing"

	"github.com/input-output-hk/catalyst-forge/forge/cli/internal/testutils"
	"github.com/spf13/afero"
)

func TestLocalClientGet(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		files       map[string]string
		expect      string
		expectErr   bool
		expectedErr error
	}{
		{
			name: "simple",
			key:  ".secrets",
			files: map[string]string{
				".secrets": "secret",
			},
			expect:      "secret",
			expectErr:   false,
			expectedErr: nil,
		},
		{
			name: "file not found",
			key:  "foo",
			files: map[string]string{
				".secrets": "secret",
			},
			expect:      "",
			expectErr:   true,
			expectedErr: fmt.Errorf("open foo: file does not exist"),
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

			ret, err := testutils.CheckError(t, err, tt.expectErr, tt.expectedErr)
			if err != nil {
				t.Error(err)
				return
			} else if ret {
				return
			}

			if got != tt.expect {
				t.Errorf("expected: %s, got: %s", tt.expect, got)
			}
		})
	}
}

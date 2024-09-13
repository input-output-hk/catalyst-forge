package providers

import (
	"os"
	"testing"

	"github.com/input-output-hk/catalyst-forge/tools/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestEnvClientGet(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		env         map[string]string
		expect      string
		expectErr   bool
		expectedErr string
	}{
		{
			name: "simple",
			key:  "FOO",
			env: map[string]string{
				"FOO": "secret",
			},
			expect:      "secret",
			expectErr:   false,
			expectedErr: "",
		},
		{
			name:        "secret not found",
			key:         "BAR",
			env:         map[string]string{},
			expect:      "",
			expectErr:   true,
			expectedErr: "enviroment variable BAR not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &EnvClient{
				logger: testutils.NewNoopLogger(),
			}

			for k, v := range tt.env {
				_ = os.Setenv(k, v)
			}

			got, err := client.Get(tt.key)
			if testutils.AssertError(t, err, tt.expectErr, tt.expectedErr) {
				return
			}
			assert.Equal(t, tt.expect, got)
		})
	}
}

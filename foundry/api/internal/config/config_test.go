package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBootstrapTokenValidation(t *testing.T) {
    t.Parallel()
	tests := []struct {
		name           string
		bootstrapToken string
		wantError      bool
		errorContains  string
	}{
		{
			name:           "no bootstrap token is valid",
			bootstrapToken: "",
			wantError:      false,
		},
		{
			name:           "bootstrap token with 32 chars is valid",
			bootstrapToken: strings.Repeat("a", 32),
			wantError:      false,
		},
		{
			name:           "bootstrap token with more than 32 chars is valid",
			bootstrapToken: strings.Repeat("a", 64),
			wantError:      false,
		},
		{
			name:           "bootstrap token with less than 32 chars is invalid",
			bootstrapToken: "short-token",
			wantError:      true,
			errorContains:  "at least 32 characters",
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			cfg := &Config{
				Database: DatabaseConfig{
					Password: "test-password", // Required field
				},
				Auth: AuthConfig{
					BootstrapToken: tt.bootstrapToken,
				},
			}

			err := cfg.Validate()
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetSafeBootstrapInfo(t *testing.T) {
    t.Parallel()
	tests := []struct {
		name           string
		bootstrapToken string
		expected       string
	}{
		{
			name:           "no token configured",
			bootstrapToken: "",
			expected:       "Bootstrap token not configured",
		},
		{
			name:           "token configured shows length",
			bootstrapToken: strings.Repeat("x", 32),
			expected:       "Bootstrap token configured (length: 32)",
		},
		{
			name:           "long token shows correct length",
			bootstrapToken: strings.Repeat("x", 64),
			expected:       "Bootstrap token configured (length: 64)",
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			cfg := &Config{
				Auth: AuthConfig{
					BootstrapToken: tt.bootstrapToken,
				},
			}

			result := cfg.GetSafeBootstrapInfo()
			assert.Equal(t, tt.expected, result)
			// Ensure the actual token value is never exposed
			if tt.bootstrapToken != "" {
				assert.NotContains(t, result, tt.bootstrapToken)
			}
		})
	}
}

func TestMaskSensitive(t *testing.T) {
    t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "<not set>",
		},
		{
			name:     "short string",
			input:    "abc",
			expected: "<set>",
		},
		{
			name:     "exactly 8 chars",
			input:    "12345678",
			expected: "<set>",
		},
		{
			name:     "long string shows first 4 chars",
			input:    "secret-token-value",
			expected: "secr****",
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			result := MaskSensitive(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

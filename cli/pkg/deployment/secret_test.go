package deployment

import (
	"log/slog"
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/project/pkg/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/pkg/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/project/pkg/secrets/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestGetGitToken(t *testing.T) {
	tests := []struct {
		name        string
		secretValue string
		expected    string
		expectErr   bool
		expectedErr string
	}{
		{
			name:        "valid secret",
			secretValue: `{"token":"foo"}`,
			expected:    "foo",
			expectErr:   false,
			expectedErr: "",
		},
		{
			name:        "invalid secret",
			secretValue: `{"foo":"bar"}`,
			expected:    "",
			expectErr:   true,
			expectedErr: "git provider token is empty",
		},
		{
			name:        "invalid JSON",
			secretValue: `invalid`,
			expected:    "",
			expectErr:   true,
			expectedErr: "could not unmarshal secret: invalid character 'i' looking for beginning of value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &mocks.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return tt.secretValue, nil
				},
			}
			store := secrets.NewSecretStore(map[secrets.Provider]func(*slog.Logger) (secrets.SecretProvider, error){
				secrets.Provider("mock"): func(logger *slog.Logger) (secrets.SecretProvider, error) {
					return provider, nil
				},
			})

			project := project.Project{
				Blueprint: schema.Blueprint{
					Global: schema.Global{
						CI: schema.GlobalCI{
							Providers: schema.Providers{
								Git: schema.ProviderGit{
									Credentials: &schema.Secret{
										Provider: "mock",
										Path:     "foo",
									},
								},
							},
						},
					},
				},
			}

			token, err := GetGitToken(&project, &store, testutils.NewNoopLogger())
			if testutils.AssertError(t, err, tt.expectErr, tt.expectedErr) {
				return
			}

			assert.Equal(t, tt.expected, token)
		})
	}
}

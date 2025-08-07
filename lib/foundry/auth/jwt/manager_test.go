package jwt_test

import (
	"os"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/keys"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestES256Manager(t *testing.T) {
	tests := []struct {
		name        string
		audiences   []string
		issuer      string
		permissions []auth.Permission
		validate    func(*testing.T, string, error, *jwt.ES256Manager)
	}{
		{
			name:        "default values",
			audiences:   nil,
			issuer:      "",
			permissions: []auth.Permission{auth.PermAliasRead},
			validate: func(t *testing.T, token string, err error, am *jwt.ES256Manager) {
				require.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := tokens.VerifyAuthToken(am, token)
				require.NoError(t, err)
				assert.Equal(t, "user_id", claims.UserID)
				assert.Equal(t, jwt.ISSUER, claims.Issuer)
				assert.Equal(t, []string{jwt.AUDIENCE}, []string(claims.Audience))
				assert.Equal(t, []auth.Permission{auth.PermAliasRead}, claims.Permissions)
				assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
				assert.True(t, claims.IssuedAt.Time.Before(time.Now().Add(time.Second)))
				assert.True(t, claims.NotBefore.Time.Before(time.Now().Add(time.Second)))

				assert.True(t, tokens.HasPermission(claims, auth.PermAliasRead))
				assert.False(t, tokens.HasPermission(claims, auth.PermAliasWrite))
			},
		},
		{
			name:        "custom audiences",
			audiences:   []string{"custom-audience", "another-audience"},
			issuer:      "",
			permissions: []auth.Permission{auth.PermDeploymentRead, auth.PermDeploymentWrite},
			validate: func(t *testing.T, token string, err error, am *jwt.ES256Manager) {
				require.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := tokens.VerifyAuthToken(am, token)
				require.NoError(t, err)
				assert.Equal(t, "user_id", claims.UserID)
				assert.Equal(t, jwt.ISSUER, claims.Issuer)
				assert.Equal(t, []string{"custom-audience", "another-audience"}, []string(claims.Audience))
				assert.Equal(t, []auth.Permission{auth.PermDeploymentRead, auth.PermDeploymentWrite}, claims.Permissions)
				assert.True(t, claims.ExpiresAt.Time.After(time.Now()))

				assert.True(t, tokens.HasPermission(claims, auth.PermDeploymentRead))
				assert.True(t, tokens.HasPermission(claims, auth.PermDeploymentWrite))
				assert.False(t, tokens.HasPermission(claims, auth.PermAliasRead))
			},
		},
		{
			name:        "custom issuer",
			audiences:   nil,
			issuer:      "custom-issuer.com",
			permissions: []auth.Permission{},
			validate: func(t *testing.T, token string, err error, am *jwt.ES256Manager) {
				require.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := tokens.VerifyAuthToken(am, token)
				require.NoError(t, err)
				assert.Equal(t, "user_id", claims.UserID)
				assert.Equal(t, "custom-issuer.com", claims.Issuer)
				assert.Equal(t, []string{jwt.AUDIENCE}, []string(claims.Audience))
				assert.Empty(t, claims.Permissions)
				assert.True(t, claims.ExpiresAt.Time.After(time.Now()))

				assert.False(t, tokens.HasPermission(claims, auth.PermAliasRead))
			},
		},
		{
			name:        "custom audiences and issuer",
			audiences:   []string{"test-audience"},
			issuer:      "test-issuer.org",
			permissions: []auth.Permission{auth.PermReleaseRead, auth.PermReleaseWrite, auth.PermDeploymentEventRead},
			validate: func(t *testing.T, token string, err error, am *jwt.ES256Manager) {
				require.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := tokens.VerifyAuthToken(am, token)
				require.NoError(t, err)
				assert.Equal(t, "user_id", claims.UserID)
				assert.Equal(t, "test-issuer.org", claims.Issuer)
				assert.Equal(t, []string{"test-audience"}, []string(claims.Audience))
				assert.Equal(t, []auth.Permission{auth.PermReleaseRead, auth.PermReleaseWrite, auth.PermDeploymentEventRead}, claims.Permissions)
				assert.True(t, claims.ExpiresAt.Time.After(time.Now()))

				assert.True(t, tokens.HasPermission(claims, auth.PermReleaseRead))
				assert.False(t, tokens.HasPermission(claims, auth.PermDeploymentEventWrite))
			},
		},
		{
			name:        "empty audiences",
			audiences:   []string{},
			issuer:      "",
			permissions: []auth.Permission{auth.PermAliasWrite},
			validate: func(t *testing.T, token string, err error, am *jwt.ES256Manager) {
				require.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := tokens.VerifyAuthToken(am, token)
				require.NoError(t, err)
				assert.Equal(t, "user_id", claims.UserID)
				assert.Equal(t, jwt.ISSUER, claims.Issuer)
				assert.Nil(t, claims.Audience)
				assert.Equal(t, []auth.Permission{auth.PermAliasWrite}, claims.Permissions)
				assert.True(t, claims.ExpiresAt.Time.After(time.Now()))

				assert.True(t, tokens.HasPermission(claims, auth.PermAliasWrite))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			am := newES256Manager(t, test.audiences, test.issuer)
			token, err := tokens.GenerateAuthToken(am, "user_id", test.permissions, time.Minute)
			test.validate(t, token, err, am)
		})
	}
}

func TestGenerateES256Keys(t *testing.T) {
	keyPair, err := keys.GenerateES256Keys()
	require.NoError(t, err)
	assert.NotNil(t, keyPair)
	assert.NotEmpty(t, keyPair.PrivateKeyPEM)
	assert.NotEmpty(t, keyPair.PublicKeyPEM)

	keyPair2, err := keys.GenerateES256Keys()
	require.NoError(t, err)
	assert.NotEqual(t, string(keyPair.PrivateKeyPEM), string(keyPair2.PrivateKeyPEM))
}

func newES256Manager(t *testing.T, audiences []string, issuer string) *jwt.ES256Manager {
	kp, err := keys.GenerateES256Keys()
	require.NoError(t, err)

	// Create temporary files for the keys
	privateKeyFile := "/tmp/test-private.pem"
	publicKeyFile := "/tmp/test-public.pem"

	// Write keys to files (ES256Manager loads from files)
	err = writeFile(privateKeyFile, kp.PrivateKeyPEM)
	require.NoError(t, err)
	err = writeFile(publicKeyFile, kp.PublicKeyPEM)
	require.NoError(t, err)

	// Clean up files when test completes
	t.Cleanup(func() {
		_ = removeFile(privateKeyFile)
		_ = removeFile(publicKeyFile)
	})

	var options []jwt.ManagerOption
	if audiences != nil {
		options = append(options, jwt.WithManagerAudiences(audiences))
	}
	if issuer != "" {
		options = append(options, jwt.WithManagerIssuer(issuer))
	}

	manager, err := jwt.NewES256Manager(privateKeyFile, publicKeyFile, options...)
	require.NoError(t, err)
	return manager
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}

func removeFile(path string) error {
	return os.Remove(path)
}

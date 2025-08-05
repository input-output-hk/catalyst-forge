package jwt

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager(t *testing.T) {
	tests := []struct {
		name        string
		audiences   []string
		issuer      string
		permissions []auth.Permission
		validate    func(*testing.T, string, error, *JWTManager)
	}{
		{
			name:        "default values",
			audiences:   nil,
			issuer:      "",
			permissions: []auth.Permission{auth.PermAliasRead},
			validate: func(t *testing.T, token string, err error, am *JWTManager) {
				require.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := am.ValidateToken(token)
				require.NoError(t, err)
				assert.Equal(t, "user_id", claims.UserID)
				assert.Equal(t, ISSUER, claims.Issuer)
				assert.Equal(t, []string{AUDIENCE}, []string(claims.Audience))
				assert.Equal(t, []auth.Permission{auth.PermAliasRead}, claims.Permissions)
				assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
				assert.True(t, claims.IssuedAt.Time.Before(time.Now().Add(time.Second)))
				assert.True(t, claims.NotBefore.Time.Before(time.Now().Add(time.Second)))

				hasPermission, err := am.HasPermission(token, auth.PermAliasRead)
				require.NoError(t, err)
				assert.True(t, hasPermission)

				hasPermission, err = am.HasPermission(token, auth.PermAliasWrite)
				require.NoError(t, err)
				assert.False(t, hasPermission)
			},
		},
		{
			name:        "custom audiences",
			audiences:   []string{"custom-audience", "another-audience"},
			issuer:      "",
			permissions: []auth.Permission{auth.PermDeploymentRead, auth.PermDeploymentWrite},
			validate: func(t *testing.T, token string, err error, am *JWTManager) {
				require.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := am.ValidateToken(token)
				require.NoError(t, err)
				assert.Equal(t, "user_id", claims.UserID)
				assert.Equal(t, ISSUER, claims.Issuer)
				assert.Equal(t, []string{"custom-audience", "another-audience"}, []string(claims.Audience))
				assert.Equal(t, []auth.Permission{auth.PermDeploymentRead, auth.PermDeploymentWrite}, claims.Permissions)
				assert.True(t, claims.ExpiresAt.Time.After(time.Now()))

				hasPermission, err := am.HasPermission(token, auth.PermDeploymentRead)
				require.NoError(t, err)
				assert.True(t, hasPermission)

				hasPermission, err = am.HasPermission(token, auth.PermDeploymentWrite)
				require.NoError(t, err)
				assert.True(t, hasPermission)

				hasPermission, err = am.HasPermission(token, auth.PermAliasRead)
				require.NoError(t, err)
				assert.False(t, hasPermission)
			},
		},
		{
			name:        "custom issuer",
			audiences:   nil,
			issuer:      "custom-issuer.com",
			permissions: []auth.Permission{},
			validate: func(t *testing.T, token string, err error, am *JWTManager) {
				require.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := am.ValidateToken(token)
				require.NoError(t, err)
				assert.Equal(t, "user_id", claims.UserID)
				assert.Equal(t, "custom-issuer.com", claims.Issuer)
				assert.Equal(t, []string{AUDIENCE}, []string(claims.Audience))
				assert.Empty(t, claims.Permissions)
				assert.True(t, claims.ExpiresAt.Time.After(time.Now()))

				hasPermission, err := am.HasPermission(token, auth.PermAliasRead)
				require.NoError(t, err)
				assert.False(t, hasPermission)
			},
		},
		{
			name:        "custom audiences and issuer",
			audiences:   []string{"test-audience"},
			issuer:      "test-issuer.org",
			permissions: []auth.Permission{auth.PermReleaseRead, auth.PermReleaseWrite, auth.PermDeploymentEventRead},
			validate: func(t *testing.T, token string, err error, am *JWTManager) {
				require.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := am.ValidateToken(token)
				require.NoError(t, err)
				assert.Equal(t, "user_id", claims.UserID)
				assert.Equal(t, "test-issuer.org", claims.Issuer)
				assert.Equal(t, []string{"test-audience"}, []string(claims.Audience))
				assert.Equal(t, []auth.Permission{auth.PermReleaseRead, auth.PermReleaseWrite, auth.PermDeploymentEventRead}, claims.Permissions)
				assert.True(t, claims.ExpiresAt.Time.After(time.Now()))

				hasPermission, err := am.HasPermission(token, auth.PermReleaseRead)
				require.NoError(t, err)
				assert.True(t, hasPermission)

				hasPermission, err = am.HasPermission(token, auth.PermDeploymentEventWrite)
				require.NoError(t, err)
				assert.False(t, hasPermission)
			},
		},
		{
			name:        "empty audiences",
			audiences:   []string{},
			issuer:      "",
			permissions: []auth.Permission{auth.PermAliasWrite},
			validate: func(t *testing.T, token string, err error, am *JWTManager) {
				require.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := am.ValidateToken(token)
				require.NoError(t, err)
				assert.Equal(t, "user_id", claims.UserID)
				assert.Equal(t, ISSUER, claims.Issuer)
				assert.Nil(t, claims.Audience)
				assert.Equal(t, []auth.Permission{auth.PermAliasWrite}, claims.Permissions)
				assert.True(t, claims.ExpiresAt.Time.After(time.Now()))

				hasPermission, err := am.HasPermission(token, auth.PermAliasWrite)
				require.NoError(t, err)
				assert.True(t, hasPermission)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			am := newJWTManager(t, test.audiences, test.issuer)
			token, err := am.GenerateToken("user_id", test.permissions, time.Minute)
			test.validate(t, token, err, am)
		})
	}
}

func TestGenerateES256Keys(t *testing.T) {
	keyPair, err := GenerateES256Keys()
	require.NoError(t, err)

	assert.NotNil(t, keyPair)
	assert.NotEmpty(t, keyPair.PrivateKeyPEM)
	assert.NotEmpty(t, keyPair.PublicKeyPEM)

	privateKeyBlock, _ := pem.Decode(keyPair.PrivateKeyPEM)
	require.NotNil(t, privateKeyBlock)
	assert.Equal(t, "EC PRIVATE KEY", privateKeyBlock.Type)

	privateKey, err := x509.ParseECPrivateKey(privateKeyBlock.Bytes)
	require.NoError(t, err)
	assert.Equal(t, "P-256", privateKey.Curve.Params().Name)

	publicKeyBlock, _ := pem.Decode(keyPair.PublicKeyPEM)
	require.NotNil(t, publicKeyBlock)
	assert.Equal(t, "PUBLIC KEY", publicKeyBlock.Type)

	publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	require.NoError(t, err)

	publicKey, ok := publicKeyInterface.(*ecdsa.PublicKey)
	require.True(t, ok)

	assert.True(t, publicKey.Equal(&privateKey.PublicKey))
	assert.Equal(t, "P-256", publicKey.Curve.Params().Name)

	keyPair2, err := GenerateES256Keys()
	require.NoError(t, err)

	assert.NotEqual(t, string(keyPair.PrivateKeyPEM), string(keyPair2.PrivateKeyPEM))
}

func newJWTManager(t *testing.T, audiences []string, issuer string) *JWTManager {
	fs := billy.NewInMemoryFs()
	kp, err := GenerateES256Keys()
	require.NoError(t, err)

	privateKeyBlock, _ := pem.Decode(kp.PrivateKeyPEM)
	require.NotNil(t, privateKeyBlock)

	privateKey, err := x509.ParseECPrivateKey(privateKeyBlock.Bytes)
	require.NoError(t, err)

	publicKeyBlock, _ := pem.Decode(kp.PublicKeyPEM)
	require.NotNil(t, publicKeyBlock)

	publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	require.NoError(t, err)

	publicKey, ok := publicKeyInterface.(*ecdsa.PublicKey)
	require.True(t, ok)

	// Set default values if not provided
	if audiences == nil {
		audiences = []string{AUDIENCE}
	}
	if issuer == "" {
		issuer = ISSUER
	}

	return &JWTManager{
		audiences:  audiences,
		issuer:     issuer,
		fs:         fs,
		logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		privateKey: privateKey,
		publicKey:  publicKey,
	}
}

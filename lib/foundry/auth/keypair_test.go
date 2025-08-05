package auth

import (
	"os"
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyPairSave(t *testing.T) {
	fs := billy.NewInMemoryFs()
	am := NewAuthManager(WithFilesystem(fs))

	kp, err := am.GenerateKeypair()
	require.NoError(t, err)

	err = kp.Save("/keys")
	require.NoError(t, err)

	exists, err := fs.Exists("/keys/public.pem")
	require.NoError(t, err)
	assert.True(t, exists)

	pubKeyPEM, err := fs.ReadFile("/keys/public.pem")
	require.NoError(t, err)
	pubKey, err := kp.encodePublicPEM()
	require.NoError(t, err)
	assert.Equal(t, pubKeyPEM, pubKey)

	pubKeyInfo, err := fs.Stat("/keys/public.pem")
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), pubKeyInfo.Mode())

	exists, err = fs.Exists("/keys/private.pem")
	require.NoError(t, err)
	assert.True(t, exists)

	privKeyPEM, err := fs.ReadFile("/keys/private.pem")
	require.NoError(t, err)
	privKey, err := kp.encodePrivatePEM()
	require.NoError(t, err)
	assert.Equal(t, privKeyPEM, privKey)

	privKeyInfo, err := fs.Stat("/keys/private.pem")
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), privKeyInfo.Mode())
}

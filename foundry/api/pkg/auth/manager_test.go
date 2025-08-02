package auth

// func TestAuthManager_GenerateKey(t *testing.T) {
// 	tests := []struct {
// 		name string
// 	}{
// 		{name: "success"},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			am := NewAuthManager()
// 			kp, err := am.GenerateKey()
// 			require.NoError(t, err)
// 			require.NotNil(t, kp)
// 			require.Len(t, kp.PublicKey, ed25519.PublicKeySize)
// 			require.Len(t, kp.PrivateKey, ed25519.PrivateKeySize)

// 			msg := []byte("hello")
// 			sig := ed25519.Sign(kp.PrivateKey, msg)
// 			assert.True(t, ed25519.Verify(kp.PublicKey, msg, sig))
// 		})
// 	}
// }

// func TestKeyPairSave(t *testing.T) {
// 	fs := billy.NewInMemoryFs()
// 	am := NewAuthManager(WithFilesystem(fs))

// 	kp, err := am.GenerateKey()
// 	require.NoError(t, err)

// 	err = kp.Save("/keys")
// 	require.NoError(t, err)

// 	exists, err := fs.Exists("/keys/public.key")
// 	require.NoError(t, err)
// 	assert.True(t, exists)

// 	pubKeyBytes, err := fs.ReadFile("/keys/public.key")
// 	require.NoError(t, err)
// 	pubKey := ed25519.PublicKey(pubKeyBytes)
// 	assert.Equal(t, kp.PublicKey, pubKey)

// 	privKeyBytes, err := fs.ReadFile("/keys/private.key")
// 	require.NoError(t, err)
// 	privKey := ed25519.PrivateKey(privKeyBytes)
// 	assert.Equal(t, kp.PrivateKey, privKey)
// }

package crypto

import (
	"testing"
	"time"
)

// CreateTestKeyManager provides a test-only helper for constructing a key manager
// with a generated ES256 key and current-quarter KID.
func CreateTestKeyManager(t *testing.T) KeyManager {
	t.Helper()
	manager := NewES256KeyManager()
	key, err := GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	pemKey, err := EncodePrivateKeyPEM(key)
	if err != nil {
		t.Fatalf("failed to encode key: %v", err)
	}
	kid := GenerateKeyIDAt(time.Now())
	if km, ok := manager.(*es256KeyManager); ok {
		if err := km.AddKey(kid, pemKey); err != nil {
			t.Fatalf("failed to add key: %v", err)
		}
	}
	return manager
}

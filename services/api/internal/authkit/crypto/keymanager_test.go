package crypto

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestKey generates a test P-256 key for testing.
func generateTestKey(t *testing.T) (*ecdsa.PrivateKey, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "failed to generate test key")

	// Marshal to PEM
	keyBytes, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err, "failed to marshal key")

	pemBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}

	pemBytes := pem.EncodeToMemory(pemBlock)
	return key, pemBytes
}

// generatePKCS8Key generates a test P-256 key in PKCS8 format.
func generatePKCS8Key(t *testing.T) (*ecdsa.PrivateKey, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "failed to generate test key")

	// Marshal to PKCS8
	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err, "failed to marshal key")

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	}

	pemBytes := pem.EncodeToMemory(pemBlock)
	return key, pemBytes
}

func TestES256KeyManager_AddKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) (string, []byte)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/ec_private_key",
			setup: func(t *testing.T) (string, []byte) {
				_, pemBytes := generateTestKey(t)
				return "test-key-1", pemBytes
			},
			wantErr: false,
		},
		{
			name: "ok/pkcs8_private_key",
			setup: func(t *testing.T) (string, []byte) {
				_, pemBytes := generatePKCS8Key(t)
				return "test-key-2", pemBytes
			},
			wantErr: false,
		},
		{
			name: "error/invalid_pem",
			setup: func(t *testing.T) (string, []byte) {
				return "bad-key", []byte("not a pem block")
			},
			wantErr: true,
			errMsg:  "failed to parse PEM block",
		},
		{
			name: "error/wrong_curve",
			setup: func(t *testing.T) (string, []byte) {
				// Generate P-384 key (wrong curve)
				key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
				require.NoError(t, err)

				keyBytes, err := x509.MarshalECPrivateKey(key)
				require.NoError(t, err)

				pemBlock := &pem.Block{
					Type:  "EC PRIVATE KEY",
					Bytes: keyBytes,
				}

				return "wrong-curve", pem.EncodeToMemory(pemBlock)
			},
			wantErr: true,
			errMsg:  "key must use P-256 curve for ES256",
		},
		{
			name: "error/unsupported_key_type",
			setup: func(t *testing.T) (string, []byte) {
				pemBlock := &pem.Block{
					Type:  "RSA PRIVATE KEY",
					Bytes: []byte("dummy"),
				}
				return "rsa-key", pem.EncodeToMemory(pemBlock)
			},
			wantErr: true,
			errMsg:  "unsupported key type",
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Use concrete type for setup methods
			km := NewES256KeyManager().(*es256KeyManager)
			kid, pemBytes := tc.setup(t)

			err := km.AddKey(kid, pemBytes)

			if tc.wantErr {
				require.Error(t, err, "expected error for %s", tc.name)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg, "error message mismatch")
				}
			} else {
				require.NoError(t, err, "unexpected error for %s", tc.name)
				assert.Equal(t, kid, km.CurrentKID(), "first key should be current")
			}
		})
	}
}

func TestES256KeyManager_GenerateKey(t *testing.T) {
	t.Parallel()

	// Use concrete type for setup methods
	km := NewES256KeyManager().(*es256KeyManager)

	// Generate first key
	err := km.GenerateKey("generated-1")
	require.NoError(t, err, "failed to generate first key")
	assert.Equal(t, "generated-1", km.CurrentKID(), "first key should be current")

	// Generate second key
	err = km.GenerateKey("generated-2")
	require.NoError(t, err, "failed to generate second key")
	assert.Equal(t, "generated-1", km.CurrentKID(), "current key should not change")

	// Switch to second key
	err = km.SetCurrentKID("generated-2")
	require.NoError(t, err, "failed to switch to second key")
	assert.Equal(t, "generated-2", km.CurrentKID(), "current key should update")

	// Try to switch to non-existent key
	err = km.SetCurrentKID("non-existent")
	require.Error(t, err, "should error on non-existent key")
	assert.Contains(t, err.Error(), "key not found")
}

func TestES256KeyManager_SignAndVerify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		claims    jwt.MapClaims
		wantErr   bool
		verifyErr bool
	}{
		{
			name: "ok/basic_claims",
			claims: jwt.MapClaims{
				"sub": "user-123",
				"iat": time.Now().Unix(),
				"exp": time.Now().Add(time.Hour).Unix(),
			},
			wantErr:   false,
			verifyErr: false,
		},
		{
			name: "ok/complex_claims",
			claims: jwt.MapClaims{
				"sub":   "user-456",
				"email": "test@example.com",
				"roles": []string{"admin", "user"},
				"iat":   time.Now().Unix(),
				"exp":   time.Now().Add(time.Hour).Unix(),
				"custom": map[string]interface{}{
					"feature": "enabled",
					"count":   42,
				},
			},
			wantErr:   false,
			verifyErr: false,
		},
		{
			name: "ok/expired_token",
			claims: jwt.MapClaims{
				"sub": "user-789",
				"iat": time.Now().Add(-2 * time.Hour).Unix(),
				"exp": time.Now().Add(-1 * time.Hour).Unix(), // Expired
			},
			wantErr:   false,
			verifyErr: true, // Verification should fail
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			km := NewES256KeyManager().(*es256KeyManager)

			// Generate a key
			err := km.GenerateKey("test-key")
			require.NoError(t, err, "failed to generate key")

			// Marshal claims to JSON
			payload, err := json.Marshal(tc.claims)
			require.NoError(t, err, "failed to marshal claims")

			// Sign the token
			token, err := km.SignJWT(ctx, payload, "")
			if tc.wantErr {
				require.Error(t, err, "expected signing error")
				return
			}
			require.NoError(t, err, "unexpected signing error")
			require.NotEmpty(t, token, "token should not be empty")

			// Verify token structure (header.payload.signature)
			parts := strings.Split(token, ".")
			assert.Len(t, parts, 3, "JWT should have 3 parts")

			// Verify the token
			claims, err := km.VerifyJWT(ctx, token)
			if tc.verifyErr {
				require.Error(t, err, "expected verification error")
				return
			}
			require.NoError(t, err, "unexpected verification error")

			// Check claims match (ignore time-based fields for comparison)
			assert.Equal(t, tc.claims["sub"], claims["sub"], "subject mismatch")
			if email, exists := tc.claims["email"]; exists {
				assert.Equal(t, email, claims["email"], "email mismatch")
			}
		})
	}
}

func TestES256KeyManager_AlgorithmConfusionAttack(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	km := NewES256KeyManager().(*es256KeyManager)

	// Generate a key
	err := km.GenerateKey("test-key")
	require.NoError(t, err, "failed to generate key")

	// Create a valid token
	claims := jwt.MapClaims{
		"sub": "user-123",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	payload, err := json.Marshal(claims)
	require.NoError(t, err, "failed to marshal claims")

	validToken, err := km.SignJWT(ctx, payload, "")
	require.NoError(t, err, "failed to sign token")

	tests := []struct {
		name        string
		modifyToken func(string) string
		wantErr     bool
		errContains string
	}{
		{
			name: "attack/none_algorithm",
			modifyToken: func(token string) string {
				// Try to use "none" algorithm
				parts := strings.Split(token, ".")
				// Decode header
				header := `{"alg":"none","kid":"test-key","typ":"JWT"}`
				encodedHeader := base64RawURLEncode([]byte(header))
				// Remove signature
				return encodedHeader + "." + parts[1] + "."
			},
			wantErr:     true,
			errContains: "unexpected alg",
		},
		{
			name: "attack/hs256_with_public_key",
			modifyToken: func(token string) string {
				// Try to use HS256 instead of ES256
				parts := strings.Split(token, ".")
				header := `{"alg":"HS256","kid":"test-key","typ":"JWT"}`
				encodedHeader := base64RawURLEncode([]byte(header))
				// Keep original payload and signature (will be invalid)
				return encodedHeader + "." + parts[1] + "." + parts[2]
			},
			wantErr:     true,
			errContains: "unexpected alg",
		},
		{
			name: "attack/rs256_algorithm",
			modifyToken: func(token string) string {
				parts := strings.Split(token, ".")
				header := `{"alg":"RS256","kid":"test-key","typ":"JWT"}`
				encodedHeader := base64RawURLEncode([]byte(header))
				return encodedHeader + "." + parts[1] + "." + parts[2]
			},
			wantErr:     true,
			errContains: "unexpected alg",
		},
		{
			name: "attack/missing_kid",
			modifyToken: func(token string) string {
				parts := strings.Split(token, ".")
				header := `{"alg":"ES256","typ":"JWT"}`
				encodedHeader := base64RawURLEncode([]byte(header))
				return encodedHeader + "." + parts[1] + "." + parts[2]
			},
			wantErr:     true,
			errContains: "missing kid",
		},
		{
			name: "attack/wrong_kid",
			modifyToken: func(token string) string {
				parts := strings.Split(token, ".")
				header := `{"alg":"ES256","kid":"wrong-key","typ":"JWT"}`
				encodedHeader := base64RawURLEncode([]byte(header))
				return encodedHeader + "." + parts[1] + "." + parts[2]
			},
			wantErr:     true,
			errContains: "verification key not found",
		},
		{
			name: "attack/tampered_payload",
			modifyToken: func(token string) string {
				parts := strings.Split(token, ".")
				// Modify payload
				tamperedClaims := jwt.MapClaims{
					"sub": "admin", // Changed from user-123
					"iat": time.Now().Unix(),
					"exp": time.Now().Add(time.Hour).Unix(),
				}
				tamperedPayload, _ := json.Marshal(tamperedClaims)
				encodedPayload := base64RawURLEncode(tamperedPayload)
				// Keep original header and signature
				return parts[0] + "." + encodedPayload + "." + parts[2]
			},
			wantErr:     true,
			errContains: "signature is invalid",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			maliciousToken := tc.modifyToken(validToken)

			_, err := km.VerifyJWT(ctx, maliciousToken)

			if tc.wantErr {
				require.Error(t, err, "expected verification to fail for %s", tc.name)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains, "error message mismatch")
				}
			} else {
				require.NoError(t, err, "unexpected verification error")
			}
		})
	}
}

func TestES256KeyManager_KeyRotation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	km := NewES256KeyManager().(*es256KeyManager)

	// Generate multiple keys (simulating quarterly rotation)
	keys := []string{"2024-q1", "2024-q2", "2024-q3", "2024-q4"}
	for _, kid := range keys {
		err := km.GenerateKey(kid)
		require.NoError(t, err, "failed to generate key %s", kid)
	}

	// Create tokens with different keys
	tokens := make(map[string]string)
	for _, kid := range keys {
		err := km.SetCurrentKID(kid)
		require.NoError(t, err, "failed to set current key to %s", kid)

		claims := jwt.MapClaims{
			"sub": "user-" + kid,
			"iat": time.Now().Unix(),
			"exp": time.Now().Add(time.Hour).Unix(),
		}

		payload, err := json.Marshal(claims)
		require.NoError(t, err)

		token, err := km.SignJWT(ctx, payload, "")
		require.NoError(t, err, "failed to sign with key %s", kid)
		tokens[kid] = token
	}

	// Verify all tokens can still be verified (simulating old tokens still valid)
	for kid, token := range tokens {
		claims, err := km.VerifyJWT(ctx, token)
		require.NoError(t, err, "failed to verify token signed with %s", kid)
		assert.Equal(t, "user-"+kid, claims["sub"], "subject mismatch for %s", kid)
	}
}

func TestES256KeyManager_JWKS(t *testing.T) {
	t.Parallel()

	km := NewES256KeyManager().(*es256KeyManager)

	// Generate multiple keys
	kids := []string{"key-1", "key-2", "key-3"}
	for _, kid := range kids {
		err := km.GenerateKey(kid)
		require.NoError(t, err, "failed to generate key %s", kid)
	}

	// Get JWKS
	jwksInterface := km.JWKS()
	require.NotNil(t, jwksInterface, "JWKS should not be nil")

	jwks, ok := jwksInterface.(map[string]interface{})
	require.True(t, ok, "JWKS should be a map")

	keysInterface, exists := jwks["keys"]
	require.True(t, exists, "JWKS should have 'keys' field")

	keys, ok := keysInterface.([]map[string]interface{})
	require.True(t, ok, "keys should be an array of maps")
	assert.Len(t, keys, 3, "should have 3 keys")

	// Verify each key has required fields
	for i, key := range keys {
		assert.Equal(t, "EC", key["kty"], "key type should be EC for key %d", i)
		assert.Contains(t, kids, key["kid"], "kid should be in our list for key %d", i)
		assert.Equal(t, "sig", key["use"], "use should be 'sig' for key %d", i)
		assert.Equal(t, "ES256", key["alg"], "algorithm should be ES256 for key %d", i)
		assert.Equal(t, "P-256", key["crv"], "curve should be P-256 for key %d", i)

		// Verify x and y coordinates exist and are base64url encoded
		x, hasX := key["x"].(string)
		y, hasY := key["y"].(string)
		assert.True(t, hasX, "key should have x coordinate for key %d", i)
		assert.True(t, hasY, "key should have y coordinate for key %d", i)

		// Verify coordinates are valid base64url (no padding)
		assert.NotContains(t, x, "=", "x coordinate should not have padding for key %d", i)
		assert.NotContains(t, y, "=", "y coordinate should not have padding for key %d", i)

		// Verify coordinate length (P-256 coordinates are 32 bytes, ~43 chars in base64)
		assert.GreaterOrEqual(t, len(x), 42, "x coordinate too short for key %d", i)
		assert.LessOrEqual(t, len(x), 44, "x coordinate too long for key %d", i)
		assert.GreaterOrEqual(t, len(y), 42, "y coordinate too short for key %d", i)
		assert.LessOrEqual(t, len(y), 44, "y coordinate too long for key %d", i)
	}
}

func TestES256KeyManager_Concurrency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	km := NewES256KeyManager().(*es256KeyManager)

	// Generate initial key
	err := km.GenerateKey("concurrent-key")
	require.NoError(t, err)

	// Run concurrent operations
	done := make(chan bool, 3)

	// Goroutine 1: Sign tokens
	go func() {
		for i := 0; i < 100; i++ {
			claims := jwt.MapClaims{
				"sub": "user",
				"seq": i,
			}
			payload, _ := json.Marshal(claims)
			_, err := km.SignJWT(ctx, payload, "")
			assert.NoError(t, err, "signing failed at iteration %d", i)
		}
		done <- true
	}()

	// Goroutine 2: Add keys
	go func() {
		for i := 0; i < 10; i++ {
			kid := fmt.Sprintf("new-key-%d", i)
			err := km.GenerateKey(kid)
			assert.NoError(t, err, "key generation failed for %s", kid)
		}
		done <- true
	}()

	// Goroutine 3: Get JWKS
	go func() {
		for i := 0; i < 50; i++ {
			jwks := km.JWKS()
			assert.NotNil(t, jwks, "JWKS was nil at iteration %d", i)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}

// base64RawURLEncode is a helper for creating test tokens
func base64RawURLEncode(data []byte) string {
	encoded := make([]byte, base64.RawURLEncoding.EncodedLen(len(data)))
	base64.RawURLEncoding.Encode(encoded, data)
	return string(encoded)
}

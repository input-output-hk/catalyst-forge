package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockKeyManager implements crypto.KeyManager for testing
type mockKeyManager struct {
	signFunc   func(ctx context.Context, payload []byte, kid string) (string, error)
	verifyFunc func(ctx context.Context, token string) (map[string]interface{}, error)
	jwksFunc   func() interface{}
	currentKID string
}

func (m *mockKeyManager) SignJWT(ctx context.Context, payload []byte, kid string) (string, error) {
	if m.signFunc != nil {
		return m.signFunc(ctx, payload, kid)
	}
	
	// Default implementation: create a simple JWT
	header := map[string]interface{}{
		"alg": "ES256",
		"kid": kid,
		"typ": "JWT",
	}
	
	headerJSON, _ := json.Marshal(header)
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	
	// Simple mock signature
	signature := base64.RawURLEncoding.EncodeToString([]byte("mock-signature"))
	
	return encodedHeader + "." + encodedPayload + "." + signature, nil
}

func (m *mockKeyManager) VerifyJWT(ctx context.Context, token string) (map[string]interface{}, error) {
	if m.verifyFunc != nil {
		return m.verifyFunc(ctx, token)
	}
	
	// Default implementation: decode and return payload
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}
	
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	
	// Validate expiration if present
	if exp, ok := claims["exp"].(float64); ok {
		if time.Unix(int64(exp), 0).Before(time.Now()) {
			return nil, jwt.ErrTokenExpired
		}
	}
	
	return claims, nil
}

func (m *mockKeyManager) JWKS() interface{} {
	if m.jwksFunc != nil {
		return m.jwksFunc()
	}
	
	// Default JWKS
	return map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "EC",
				"use": "sig",
				"alg": "ES256",
				"kid": "test-key",
				"crv": "P-256",
				"x":   "test-x-coordinate",
				"y":   "test-y-coordinate",
			},
		},
	}
}

func (m *mockKeyManager) CurrentKID() string {
	if m.currentKID != "" {
		return m.currentKID
	}
	return "test-key"
}

func setupTokenService(t *testing.T) (TokenService, *mockKeyManager) {
	t.Helper()
	
	keyManager := &mockKeyManager{
		currentKID: "test-key",
	}
	
	// Create secure random generator
	rand := crypto.NewSecureRand()
	
	// Create token service with 15 minute TTL
	tokenService := NewTokenService(
		keyManager,
		rand,
		"test-issuer",
		15*time.Minute,
	)
	
	return tokenService, keyManager
}

func TestNewTokenService(t *testing.T) {
	t.Parallel()
	
	t.Run("ok/valid_config", func(t *testing.T) {
		t.Parallel()
		
		keyManager := &mockKeyManager{}
		rand := crypto.NewSecureRand()
		
		assert.NotPanics(t, func() {
			_ = NewTokenService(keyManager, rand, "issuer", time.Hour)
		})
	})
	
	t.Run("panic/invalid_ttl", func(t *testing.T) {
		t.Parallel()
		
		keyManager := &mockKeyManager{}
		rand := crypto.NewSecureRand()
		
		assert.Panics(t, func() {
			_ = NewTokenService(keyManager, rand, "issuer", 0)
		})
		
		assert.Panics(t, func() {
			_ = NewTokenService(keyManager, rand, "issuer", -time.Hour)
		})
	})
}

func TestTokenService_SignAccess(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name      string
		claims    AccessClaims
		wantErr   bool
		errMsg    string
		validate  func(t *testing.T, token string)
	}{
		{
			name: "ok/basic_claims",
			claims: AccessClaims{
				Sub:            uuid.New().String(),
				Email:          "user@example.com",
				SessionVersion: 1,
			},
			wantErr: false,
			validate: func(t *testing.T, token string) {
				// JWT should have 3 parts
				parts := strings.Split(token, ".")
				assert.Len(t, parts, 3)
			},
		},
		{
			name: "ok/with_roles_and_permissions",
			claims: AccessClaims{
				Sub:            uuid.New().String(),
				Email:          "admin@example.com",
				Roles:          []string{"admin", "user"},
				Permissions:    []string{"read", "write", "delete"},
				SessionVersion: 2,
			},
			wantErr: false,
		},
		{
			name: "ok/with_step_up",
			claims: AccessClaims{
				Sub:            uuid.New().String(),
				Email:          "user@example.com",
				SessionVersion: 1,
				StepUpUntil:    time.Now().Add(time.Hour).Unix(),
			},
			wantErr: false,
		},
		{
			name: "ok/with_custom_jti",
			claims: AccessClaims{
				Sub:            uuid.New().String(),
				Email:          "user@example.com",
				SessionVersion: 1,
				JTI:            "custom-jti-123",
			},
			wantErr: false,
			validate: func(t *testing.T, token string) {
				// Decode payload and check JTI
				parts := strings.Split(token, ".")
				payload, err := base64.RawURLEncoding.DecodeString(parts[1])
				require.NoError(t, err)
				
				var claims map[string]interface{}
				err = json.Unmarshal(payload, &claims)
				require.NoError(t, err)
				assert.Equal(t, "custom-jti-123", claims["jti"])
			},
		},
		{
			name: "error/missing_session_version",
			claims: AccessClaims{
				Sub:   uuid.New().String(),
				Email: "user@example.com",
				// SessionVersion: 0, // Missing
			},
			wantErr: true,
			errMsg:  "session version is required",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests  
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			ctx := context.Background()
			tokenSvc, _ := setupTokenService(t)
			
			token, err := tokenSvc.SignAccess(ctx, tc.claims)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Empty(t, token)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, token)
				
				if tc.validate != nil {
					tc.validate(t, token)
				}
			}
		})
	}
}

func TestTokenService_ParseAccess(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	// Create a valid token for tests that need it
	validClaims := AccessClaims{
		Sub:            uuid.New().String(),
		Email:          "user@example.com",
		Roles:          []string{"user"},
		Permissions:    []string{"read"},
		SessionVersion: 1,
		StepUpUntil:    time.Now().Add(time.Hour).Unix(),
	}
	
	// Get a fresh token service for creating the valid token
	tokenSvcSetup, _ := setupTokenService(t)
	validToken, err := tokenSvcSetup.SignAccess(ctx, validClaims)
	require.NoError(t, err)
	
	tests := []struct {
		name     string
		token    string
		setup    func(t *testing.T, keyManager *mockKeyManager) string
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, claims *AccessClaims)
	}{
		{
			name:    "ok/valid_token",
			token:   validToken,
			wantErr: false,
			validate: func(t *testing.T, claims *AccessClaims) {
				assert.Equal(t, validClaims.Sub, claims.Sub)
				assert.Equal(t, validClaims.Email, claims.Email)
				assert.Equal(t, validClaims.Roles, claims.Roles)
				assert.Equal(t, validClaims.Permissions, claims.Permissions)
				assert.Equal(t, validClaims.SessionVersion, claims.SessionVersion)
				assert.Equal(t, validClaims.StepUpUntil, claims.StepUpUntil)
			},
		},
		{
			name: "ok/token_with_audience_array",
			setup: func(t *testing.T, keyManager *mockKeyManager) string {
				// Create token with audience as array
				claims := map[string]interface{}{
					"sub":   uuid.New().String(),
					"email": "user@example.com",
					"sv":    float64(1),
					"iss":   "test-issuer",
					"aud":   []interface{}{"test-issuer", "other-audience"},
					"iat":   time.Now().Unix(),
					"exp":   time.Now().Add(time.Hour).Unix(),
					"jti":   "test-jti",
				}
				
				payload, _ := json.Marshal(claims)
				token, _ := keyManager.SignJWT(ctx, payload, "")
				return token
			},
			wantErr: false,
			validate: func(t *testing.T, claims *AccessClaims) {
				assert.Equal(t, "test-issuer", claims.Audience)
			},
		},
		{
			name: "error/expired_token",
			setup: func(t *testing.T, keyManager *mockKeyManager) string {
				// Override verify func to return expired claims
				oldVerify := keyManager.verifyFunc
				keyManager.verifyFunc = func(ctx context.Context, token string) (map[string]interface{}, error) {
					return map[string]interface{}{
						"sub":   uuid.New().String(),
						"email": "user@example.com",
						"sv":    float64(1),
						"iss":   "test-issuer",
						"aud":   "test-issuer",
						"iat":   float64(time.Now().Add(-3 * time.Hour).Unix()),
						"exp":   float64(time.Now().Add(-2 * time.Hour).Unix()),
						"jti":   "test-jti",
					}, nil
				}
				t.Cleanup(func() { keyManager.verifyFunc = oldVerify })
				return "dummy-token"
			},
			wantErr: true,
			errMsg:  "token expired",
		},
		{
			name: "error/token_not_yet_valid",
			setup: func(t *testing.T, keyManager *mockKeyManager) string {
				// Override verify func to return future claims
				oldVerify := keyManager.verifyFunc
				keyManager.verifyFunc = func(ctx context.Context, token string) (map[string]interface{}, error) {
					return map[string]interface{}{
						"sub":   uuid.New().String(),
						"email": "user@example.com",
						"sv":    float64(1),
						"iss":   "test-issuer",
						"aud":   "test-issuer",
						"iat":   float64(time.Now().Add(2 * time.Hour).Unix()),
						"exp":   float64(time.Now().Add(3 * time.Hour).Unix()),
						"jti":   "test-jti",
					}, nil
				}
				t.Cleanup(func() { keyManager.verifyFunc = oldVerify })
				return "dummy-token"
			},
			wantErr: true,
			errMsg:  "token not yet valid",
		},
		{
			name: "error/wrong_issuer",
			setup: func(t *testing.T, keyManager *mockKeyManager) string {
				// Create token with wrong issuer
				claims := map[string]interface{}{
					"sub":   uuid.New().String(),
					"email": "user@example.com",
					"sv":    float64(1),
					"iss":   "wrong-issuer",
					"aud":   "test-issuer",
					"iat":   time.Now().Unix(),
					"exp":   time.Now().Add(time.Hour).Unix(),
					"jti":   "test-jti",
				}
				
				payload, _ := json.Marshal(claims)
				token, _ := keyManager.SignJWT(ctx, payload, "")
				return token
			},
			wantErr: true,
			errMsg:  "invalid issuer",
		},
		{
			name: "error/wrong_audience",
			setup: func(t *testing.T, keyManager *mockKeyManager) string {
				// Create token with wrong audience
				claims := map[string]interface{}{
					"sub":   uuid.New().String(),
					"email": "user@example.com",
					"sv":    float64(1),
					"iss":   "test-issuer",
					"aud":   "wrong-audience",
					"iat":   time.Now().Unix(),
					"exp":   time.Now().Add(time.Hour).Unix(),
					"jti":   "test-jti",
				}
				
				payload, _ := json.Marshal(claims)
				token, _ := keyManager.SignJWT(ctx, payload, "")
				return token
			},
			wantErr: true,
			errMsg:  "token audience does not match",
		},
		{
			name: "error/missing_subject",
			setup: func(t *testing.T, keyManager *mockKeyManager) string {
				// Create token without subject
				claims := map[string]interface{}{
					// "sub": missing
					"email": "user@example.com",
					"sv":    float64(1),
					"iss":   "test-issuer",
					"aud":   "test-issuer",
					"iat":   time.Now().Unix(),
					"exp":   time.Now().Add(time.Hour).Unix(),
					"jti":   "test-jti",
				}
				
				payload, _ := json.Marshal(claims)
				token, _ := keyManager.SignJWT(ctx, payload, "")
				return token
			},
			wantErr: true,
			errMsg:  "missing subject claim",
		},
		{
			name: "error/missing_session_version",
			setup: func(t *testing.T, keyManager *mockKeyManager) string {
				// Create token without session version
				claims := map[string]interface{}{
					"sub":   uuid.New().String(),
					"email": "user@example.com",
					// "sv": missing
					"iss": "test-issuer",
					"aud": "test-issuer",
					"iat": time.Now().Unix(),
					"exp": time.Now().Add(time.Hour).Unix(),
					"jti": "test-jti",
				}
				
				payload, _ := json.Marshal(claims)
				token, _ := keyManager.SignJWT(ctx, payload, "")
				return token
			},
			wantErr: true,
			errMsg:  "missing session version",
		},
		{
			name: "error/invalid_token_format",
			setup: func(t *testing.T, keyManager *mockKeyManager) string {
				// Override verify to return error
				oldVerify := keyManager.verifyFunc
				keyManager.verifyFunc = func(ctx context.Context, token string) (map[string]interface{}, error) {
					return nil, errors.New("invalid token format")
				}
				t.Cleanup(func() { keyManager.verifyFunc = oldVerify })
				return "not.a.valid.token"
			},
			wantErr: true,
		},
		{
			name:    "error/empty_token",
			token:   "",
			wantErr: true,
		},
	}
	
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			// Get a fresh token service for this test to avoid race conditions
			tokenSvc, keyManager := setupTokenService(t)
			
			token := tc.token
			if tc.setup != nil {
				token = tc.setup(t, keyManager)
			}
			
			claims, err := tokenSvc.ParseAccess(ctx, token)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, claims)
			} else {
				require.NoError(t, err)
				require.NotNil(t, claims)
				
				if tc.validate != nil {
					tc.validate(t, claims)
				}
			}
		})
	}
}

func TestTokenService_JWKS(t *testing.T) {
	t.Parallel()
	
	tokenSvc, _ := setupTokenService(t)
	
	jwks := tokenSvc.JWKS()
	require.NotNil(t, jwks)
	
	// Verify JWKS structure
	jwksMap, ok := jwks.(map[string]interface{})
	require.True(t, ok, "JWKS should be a map")
	
	keys, exists := jwksMap["keys"]
	require.True(t, exists, "JWKS should have 'keys' field")
	
	keysArray, ok := keys.([]map[string]interface{})
	require.True(t, ok, "keys should be an array")
	assert.NotEmpty(t, keysArray, "should have at least one key")
	
	// Verify key properties
	for _, key := range keysArray {
		assert.Equal(t, "EC", key["kty"], "key type should be EC")
		assert.Equal(t, "ES256", key["alg"], "algorithm should be ES256")
		assert.Equal(t, "sig", key["use"], "use should be sig")
		assert.NotEmpty(t, key["kid"], "should have key ID")
		assert.NotEmpty(t, key["x"], "should have x coordinate")
		assert.NotEmpty(t, key["y"], "should have y coordinate")
	}
}

func TestTokenService_ClockSkew(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	tokenSvc, keyManager := setupTokenService(t)
	
	// Test token with IssuedAt 30 seconds in the future (within clock skew)
	keyManager.verifyFunc = func(ctx context.Context, token string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"sub":   uuid.New().String(),
			"email": "user@example.com",
			"sv":    float64(1),
			"iss":   "test-issuer",
			"aud":   "test-issuer",
			"iat":   float64(time.Now().Add(30 * time.Second).Unix()),
			"exp":   float64(time.Now().Add(time.Hour).Unix()),
			"jti":   "test-jti",
		}, nil
	}
	
	// Should be accepted due to clock skew tolerance
	parsed, err := tokenSvc.ParseAccess(ctx, "dummy-token")
	require.NoError(t, err)
	assert.NotNil(t, parsed)
	
	// Test token with IssuedAt 90 seconds in the future (beyond clock skew)
	keyManager.verifyFunc = func(ctx context.Context, token string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"sub":   uuid.New().String(),
			"email": "user@example.com",
			"sv":    float64(1),
			"iss":   "test-issuer",
			"aud":   "test-issuer",
			"iat":   float64(time.Now().Add(90 * time.Second).Unix()),
			"exp":   float64(time.Now().Add(time.Hour).Unix()),
			"jti":   "test-jti",
		}, nil
	}
	
	// Should be rejected
	_, err = tokenSvc.ParseAccess(ctx, "dummy-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet valid")
}

func TestValidateAccessToken(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	tokenSvc, _ := setupTokenService(t)
	
	// Create valid token
	claims := AccessClaims{
		Sub:            uuid.New().String(),
		Email:          "user@example.com",
		SessionVersion: 1,
	}
	
	validToken, err := tokenSvc.SignAccess(ctx, claims)
	require.NoError(t, err)
	
	// Test valid token
	err = ValidateAccessToken(ctx, tokenSvc, validToken)
	assert.NoError(t, err)
	
	// Test invalid token
	err = ValidateAccessToken(ctx, tokenSvc, "invalid.token")
	assert.Error(t, err)
}

func TestTokenService_Concurrency(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	tokenSvc, _ := setupTokenService(t)
	
	// Run concurrent signing operations
	done := make(chan bool, 3)
	
	// Goroutine 1: Sign tokens
	go func() {
		for i := 0; i < 100; i++ {
			claims := AccessClaims{
				Sub:            uuid.New().String(),
				Email:          "user@example.com",
				SessionVersion: int64(i + 1),
			}
			_, err := tokenSvc.SignAccess(ctx, claims)
			assert.NoError(t, err)
		}
		done <- true
	}()
	
	// Goroutine 2: Parse tokens
	go func() {
		claims := AccessClaims{
			Sub:            uuid.New().String(),
			Email:          "user@example.com",
			SessionVersion: 1,
		}
		token, _ := tokenSvc.SignAccess(ctx, claims)
		
		for i := 0; i < 100; i++ {
			_, err := tokenSvc.ParseAccess(ctx, token)
			assert.NoError(t, err)
		}
		done <- true
	}()
	
	// Goroutine 3: Get JWKS
	go func() {
		for i := 0; i < 100; i++ {
			jwks := tokenSvc.JWKS()
			assert.NotNil(t, jwks)
		}
		done <- true
	}()
	
	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}
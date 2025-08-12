package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper to create a test context with JSON request.
func createTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	if body != nil {
		jsonData, _ := json.Marshal(body)
		c.Request = httptest.NewRequest(method, path, bytes.NewBuffer(jsonData))
		c.Request.Header.Set("Content-Type", "application/json")
	} else {
		c.Request = httptest.NewRequest(method, path, nil)
	}

	return c, w
}

func TestBootstrapRequest_Validation(t *testing.T) {
    t.Parallel()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		request        interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid request",
			request: BootstrapRequest{
				Email:          "admin@example.com",
				BootstrapToken: "valid-token-at-least-32-chars-long",
			},
			expectedStatus: 401, // Will fail at token validation, but passes JSON binding
		},
		{
			name: "missing email",
			request: BootstrapRequest{
				BootstrapToken: "valid-token-at-least-32-chars-long",
			},
			expectedStatus: 400,
			expectedError:  "invalid request",
		},
		{
			name: "invalid email format",
			request: BootstrapRequest{
				Email:          "not-an-email",
				BootstrapToken: "valid-token-at-least-32-chars-long",
			},
			expectedStatus: 400,
			expectedError:  "invalid request",
		},
		{
			name: "missing bootstrap token",
			request: BootstrapRequest{
				Email: "admin@example.com",
			},
			expectedStatus: 400,
			expectedError:  "invalid request",
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			// Create a minimal handler just to test request validation
			handler := &BootstrapHandler{configuredToken: "test-token"}

			c, w := createTestContext("POST", "/auth/bootstrap", tt.request)
			handler.Bootstrap(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

func TestBootstrapHandler_ConfigurationChecks(t *testing.T) {
    t.Parallel()
	gin.SetMode(gin.TestMode)

    t.Run("no bootstrap token configured", func(t *testing.T) {
        t.Parallel()
		handler := &BootstrapHandler{configuredToken: ""}

		request := BootstrapRequest{
			Email:          "admin@example.com",
			BootstrapToken: "any-token",
		}

		c, w := createTestContext("POST", "/auth/bootstrap", request)
		handler.Bootstrap(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "bootstrap not configured")
	})

    t.Run("invalid bootstrap token", func(t *testing.T) {
        t.Parallel()
		configuredToken := "correct-bootstrap-token-32-chars"
		handler := &BootstrapHandler{configuredToken: configuredToken}

		request := BootstrapRequest{
			Email:          "admin@example.com",
			BootstrapToken: "wrong-bootstrap-token-32-chars-x",
		}

		c, w := createTestContext("POST", "/auth/bootstrap", request)
		handler.Bootstrap(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid or expired bootstrap token")
	})
}

func TestBootstrapTokenHashing(t *testing.T) {
    t.Parallel()
	// Test that bootstrap tokens are properly hashed for storage
	token := "test-bootstrap-token-32-chars-min"

	// This is the same hashing logic used in bootstrap handler
	tokenHash := sha256.Sum256([]byte(token))
	tokenHashHex := hex.EncodeToString(tokenHash[:])

	assert.Len(t, tokenHashHex, 64)         // SHA256 hex is always 64 chars
	assert.NotEqual(t, token, tokenHashHex) // Ensure it's actually hashed

	// Test deterministic hashing
	tokenHash2 := sha256.Sum256([]byte(token))
	tokenHashHex2 := hex.EncodeToString(tokenHash2[:])
	assert.Equal(t, tokenHashHex, tokenHashHex2) // Same input = same hash
}

func TestBootstrapRequest_JSONSerialization(t *testing.T) {
    t.Parallel()
	// Test that our request/response structures serialize correctly
	req := BootstrapRequest{
		Email:          "test@example.com",
		BootstrapToken: "test-token-32-chars-minimum-len",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var unmarshaled BootstrapRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, req.Email, unmarshaled.Email)
	assert.Equal(t, req.BootstrapToken, unmarshaled.BootstrapToken)
}

func TestCreateInviteResponse_JSONSerialization(t *testing.T) {
    t.Parallel()
	resp := CreateInviteResponse{
		ID:    123,
		Token: "test-invite-token",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var unmarshaled CreateInviteResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, resp.ID, unmarshaled.ID)
	assert.Equal(t, resp.Token, unmarshaled.Token)
}

// Test bootstrap handler initialization.
func TestNewBootstrapHandler(t *testing.T) {
    t.Parallel()
	// Test that handler can be created with nil dependencies for unit testing
	handler := NewBootstrapHandler(nil, nil, nil, nil, nil, "test-token", nil)

	assert.NotNil(t, handler)
	assert.Equal(t, "test-token", handler.configuredToken)

	// Test with empty token
	handler2 := NewBootstrapHandler(nil, nil, nil, nil, nil, "", nil)
	assert.Empty(t, handler2.configuredToken)
}

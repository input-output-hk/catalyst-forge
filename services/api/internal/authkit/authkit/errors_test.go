package authkit

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSentinelErrors_Definitions(t *testing.T) {
	// Test that all sentinel errors are properly defined
	sentinelErrors := []struct {
		name string
		err  error
	}{
		// Authentication errors
		{"ErrUnauthorized", ErrUnauthorized},
		{"ErrTokenExpired", ErrTokenExpired},
		{"ErrTokenInvalid", ErrTokenInvalid},
		{"ErrSessionExpired", ErrSessionExpired},
		
		// Authorization errors
		{"ErrForbidden", ErrForbidden},
		{"ErrInsufficientRoles", ErrInsufficientRoles},
		{"ErrStepUpRequired", ErrStepUpRequired},
		
		// CSRF errors
		{"ErrCSRFInvalid", ErrCSRFInvalid},
		{"ErrCSRFMissing", ErrCSRFMissing},
		
		// Rate limiting errors
		{"ErrRateLimitExceeded", ErrRateLimitExceeded},
		
		// Invite errors
		{"ErrInviteNotFound", ErrInviteNotFound},
		{"ErrInviteExpired", ErrInviteExpired},
		{"ErrInviteLocked", ErrInviteLocked},
		{"ErrInviteRedeemed", ErrInviteRedeemed},
		
		// WebAuthn errors
		{"ErrChallengeNotFound", ErrChallengeNotFound},
		{"ErrChallengeExpired", ErrChallengeExpired},
		{"ErrCredentialInvalid", ErrCredentialInvalid},
		{"ErrHardwareKeyRequired", ErrHardwareKeyRequired},
		
		// Recovery errors
		{"ErrRecoveryCodeInvalid", ErrRecoveryCodeInvalid},
		{"ErrRecoveryCodeUsed", ErrRecoveryCodeUsed},
		
		// User errors
		{"ErrUserNotFound", ErrUserNotFound},
		{"ErrUserExists", ErrUserExists},
		
		// Refresh token errors
		{"ErrRefreshTokenInvalid", ErrRefreshTokenInvalid},
		{"ErrRefreshTokenExpired", ErrRefreshTokenExpired},
		{"ErrRefreshTokenRevoked", ErrRefreshTokenRevoked},
		{"ErrRefreshTokenReplay", ErrRefreshTokenReplay},
	}

	for _, test := range sentinelErrors {
		t.Run(test.name, func(t *testing.T) {
			assert.NotNil(t, test.err, "%s should not be nil", test.name)
			assert.NotEmpty(t, test.err.Error(), "%s should have a non-empty error message", test.name)
		})
	}
}

func TestSentinelErrors_Uniqueness(t *testing.T) {
	// Test that all error messages are unique
	errors := []error{
		ErrUnauthorized, ErrTokenExpired, ErrTokenInvalid, ErrSessionExpired,
		ErrForbidden, ErrInsufficientRoles, ErrStepUpRequired,
		ErrCSRFInvalid, ErrCSRFMissing,
		ErrRateLimitExceeded,
		ErrInviteNotFound, ErrInviteExpired, ErrInviteLocked, ErrInviteRedeemed,
		ErrChallengeNotFound, ErrChallengeExpired, ErrCredentialInvalid, ErrHardwareKeyRequired,
		ErrRecoveryCodeInvalid, ErrRecoveryCodeUsed,
		ErrUserNotFound, ErrUserExists,
		ErrRefreshTokenInvalid, ErrRefreshTokenExpired, ErrRefreshTokenRevoked, ErrRefreshTokenReplay,
	}

	seen := make(map[string]bool)
	for _, err := range errors {
		msg := err.Error()
		assert.False(t, seen[msg], "duplicate error message: %s", msg)
		seen[msg] = true
	}
}

func TestSentinelErrors_Identity(t *testing.T) {
	// Test error identity using errors.Is
	tests := []struct {
		name     string
		err      error
		target   error
		expected bool
	}{
		{
			name:     "same_error_identity",
			err:      ErrUnauthorized,
			target:   ErrUnauthorized,
			expected: true,
		},
		{
			name:     "different_error_identity",
			err:      ErrUnauthorized,
			target:   ErrForbidden,
			expected: false,
		},
		{
			name:     "wrapped_error_identity",
			err:      fmt.Errorf("context: %w", ErrTokenExpired),
			target:   ErrTokenExpired,
			expected: true,
		},
		{
			name:     "double_wrapped_error_identity",
			err:      fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", ErrTokenInvalid)),
			target:   ErrTokenInvalid,
			expected: true,
		},
		{
			name:     "nil_target",
			err:      ErrUnauthorized,
			target:   nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.Is(tt.err, tt.target)
			assert.Equal(t, tt.expected, result, "errors.Is check failed")
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test error wrapping and unwrapping
	baseErr := ErrUserNotFound
	wrappedErr := fmt.Errorf("database query failed: %w", baseErr)
	doubleWrappedErr := fmt.Errorf("operation failed: %w", wrappedErr)

	// Test unwrapping
	unwrapped := errors.Unwrap(wrappedErr)
	assert.Equal(t, baseErr, unwrapped, "should unwrap to base error")

	// Test double unwrapping
	unwrapped = errors.Unwrap(doubleWrappedErr)
	assert.Equal(t, wrappedErr, unwrapped, "should unwrap to wrapped error")

	// Test errors.Is with wrapped errors
	assert.True(t, errors.Is(wrappedErr, baseErr), "wrapped error should match base error")
	assert.True(t, errors.Is(doubleWrappedErr, baseErr), "double wrapped error should match base error")
	assert.True(t, errors.Is(doubleWrappedErr, wrappedErr), "double wrapped error should match wrapped error")

	// Test errors.As with wrapped errors - for this simple case, errors.Is is more appropriate
	// since all our errors are basic error types, not custom types with methods
	assert.True(t, errors.Is(wrappedErr, baseErr), "should find base error in wrapped chain")
}

func TestErrorCategorization(t *testing.T) {
	// Test error categorization by type
	authenticationErrors := []error{
		ErrUnauthorized, ErrTokenExpired, ErrTokenInvalid, ErrSessionExpired,
	}

	authorizationErrors := []error{
		ErrForbidden, ErrInsufficientRoles, ErrStepUpRequired,
	}

	csrfErrors := []error{
		ErrCSRFInvalid, ErrCSRFMissing,
	}

	inviteErrors := []error{
		ErrInviteNotFound, ErrInviteExpired, ErrInviteLocked, ErrInviteRedeemed,
	}

	webauthnErrors := []error{
		ErrChallengeNotFound, ErrChallengeExpired, ErrCredentialInvalid, ErrHardwareKeyRequired,
	}

	recoveryErrors := []error{
		ErrRecoveryCodeInvalid, ErrRecoveryCodeUsed,
	}

	userErrors := []error{
		ErrUserNotFound, ErrUserExists,
	}

	refreshTokenErrors := []error{
		ErrRefreshTokenInvalid, ErrRefreshTokenExpired, ErrRefreshTokenRevoked, ErrRefreshTokenReplay,
	}

	categories := []struct {
		name   string
		errors []error
	}{
		{"authentication", authenticationErrors},
		{"authorization", authorizationErrors},
		{"csrf", csrfErrors},
		{"invite", inviteErrors},
		{"webauthn", webauthnErrors},
		{"recovery", recoveryErrors},
		{"user", userErrors},
		{"refresh_token", refreshTokenErrors},
	}

	for _, category := range categories {
		t.Run(category.name+"_category", func(t *testing.T) {
			assert.NotEmpty(t, category.errors, "category should have errors defined")
			
			// Verify each error in category has unique message
			seen := make(map[string]bool)
			for _, err := range category.errors {
				msg := err.Error()
				assert.False(t, seen[msg], "duplicate message in %s category: %s", category.name, msg)
				seen[msg] = true
			}
		})
	}
}

func TestErrorResponse_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		response ErrorResponse
	}{
		{
			name: "complete_response",
			response: ErrorResponse{
				Error:   "unauthorized",
				Message: "Authentication required to access this resource",
			},
		},
		{
			name: "error_only",
			response: ErrorResponse{
				Error:   "forbidden",
				Message: "",
			},
		},
		{
			name: "message_only",
			response: ErrorResponse{
				Error:   "",
				Message: "Something went wrong",
			},
		},
		{
			name: "empty_response",
			response: ErrorResponse{
				Error:   "",
				Message: "",
			},
		},
		{
			name: "special_characters",
			response: ErrorResponse{
				Error:   "invalid_input",
				Message: "Input contains special characters: <>&\"'",
			},
		},
		{
			name: "unicode_response",
			response: ErrorResponse{
				Error:   "validation_failed",
				Message: "Les données fournies ne sont pas valides: ñoño",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.response)
			require.NoError(t, err, "marshaling should succeed")
			assert.NotEmpty(t, data, "marshaled data should not be empty")

			// Test unmarshaling
			var unmarshaled ErrorResponse
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err, "unmarshaling should succeed")

			// Verify fields match
			assert.Equal(t, tt.response.Error, unmarshaled.Error, "error field should match")
			assert.Equal(t, tt.response.Message, unmarshaled.Message, "message field should match")
		})
	}
}

func TestErrorResponse_ZeroValues(t *testing.T) {
	var response ErrorResponse

	// Test zero values
	assert.Empty(t, response.Error, "error field should be empty")
	assert.Empty(t, response.Message, "message field should be empty")

	// Test JSON marshaling of zero values
	data, err := json.Marshal(response)
	require.NoError(t, err, "marshaling zero values should succeed")

	expected := `{"error":"","message":""}`
	assert.JSONEq(t, expected, string(data), "zero values should marshal correctly")
}

func TestErrorResponse_FieldValidation(t *testing.T) {
	tests := []struct {
		name        string
		response    ErrorResponse
		description string
	}{
		{
			name: "long_error_code",
			response: ErrorResponse{
				Error:   "very_long_error_code_that_exceeds_normal_length_expectations",
				Message: "Short message",
			},
			description: "Long error codes should be allowed",
		},
		{
			name: "long_message",
			response: ErrorResponse{
				Error: "short",
				Message: "This is a very long error message that provides detailed information about what went wrong and how the user might be able to fix the issue. It contains multiple sentences and provides comprehensive context.",
			},
			description: "Long messages should be allowed",
		},
		{
			name: "whitespace_preservation",
			response: ErrorResponse{
				Error:   "  padded_error  ",
				Message: "\n\tFormatted\n\tmessage\n",
			},
			description: "Whitespace should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that fields can be set and retrieved
			assert.Equal(t, tt.response.Error, tt.response.Error, tt.description)
			assert.Equal(t, tt.response.Message, tt.response.Message, tt.description)

			// Test JSON round-trip preserves values
			data, err := json.Marshal(tt.response)
			require.NoError(t, err, "marshaling should succeed")

			var unmarshaled ErrorResponse
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err, "unmarshaling should succeed")

			assert.Equal(t, tt.response.Error, unmarshaled.Error, "error should be preserved")
			assert.Equal(t, tt.response.Message, unmarshaled.Message, "message should be preserved")
		})
	}
}

func TestErrorResponse_JSONStructure(t *testing.T) {
	response := ErrorResponse{
		Error:   "test_error",
		Message: "Test message",
	}

	data, err := json.Marshal(response)
	require.NoError(t, err, "marshaling should succeed")

	// Verify JSON structure
	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err, "unmarshaling to map should succeed")

	// Check required fields exist
	assert.Contains(t, jsonMap, "error", "JSON should contain error field")
	assert.Contains(t, jsonMap, "message", "JSON should contain message field")

	// Check field types
	assert.IsType(t, "", jsonMap["error"], "error field should be string")
	assert.IsType(t, "", jsonMap["message"], "message field should be string")

	// Check values
	assert.Equal(t, "test_error", jsonMap["error"], "error value should match")
	assert.Equal(t, "Test message", jsonMap["message"], "message value should match")
}

func TestErrorCombinations(t *testing.T) {
	// Test combining different error scenarios
	tests := []struct {
		name        string
		baseError   error
		wrapContext string
		expected    string
	}{
		{
			name:        "authentication_context",
			baseError:   ErrTokenExpired,
			wrapContext: "JWT validation failed",
			expected:    "JWT validation failed: token expired",
		},
		{
			name:        "authorization_context",
			baseError:   ErrInsufficientRoles,
			wrapContext: "access control check failed",
			expected:    "access control check failed: insufficient roles",
		},
		{
			name:        "invite_context",
			baseError:   ErrInviteExpired,
			wrapContext: "invite processing failed",
			expected:    "invite processing failed: invite expired",
		},
		{
			name:        "webauthn_context",
			baseError:   ErrCredentialInvalid,
			wrapContext: "WebAuthn assertion verification failed",
			expected:    "WebAuthn assertion verification failed: credential invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedErr := fmt.Errorf("%s: %w", tt.wrapContext, tt.baseError)
			
			assert.Equal(t, tt.expected, wrappedErr.Error(), "wrapped error message should match expected")
			assert.True(t, errors.Is(wrappedErr, tt.baseError), "wrapped error should be identifiable as base error")
			
			// Test unwrapping
			unwrapped := errors.Unwrap(wrappedErr)
			assert.Equal(t, tt.baseError, unwrapped, "should unwrap to base error")
		})
	}
}

func TestErrorResponseCreation(t *testing.T) {
	// Test creating ErrorResponse from various error types
	tests := []struct {
		name     string
		err      error
		expected ErrorResponse
	}{
		{
			name: "sentinel_error",
			err:  ErrUnauthorized,
			expected: ErrorResponse{
				Error:   "unauthorized",
				Message: "unauthorized",
			},
		},
		{
			name: "wrapped_error",
			err:  fmt.Errorf("authentication failed: %w", ErrTokenExpired),
			expected: ErrorResponse{
				Error:   "authentication failed: token expired",
				Message: "authentication failed: token expired",
			},
		},
		{
			name: "generic_error",
			err:  errors.New("custom error message"),
			expected: ErrorResponse{
				Error:   "custom error message",
				Message: "custom error message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create ErrorResponse from error
			response := ErrorResponse{
				Error:   tt.err.Error(),
				Message: tt.err.Error(),
			}
			
			assert.Equal(t, tt.expected.Error, response.Error, "error field should match")
			assert.Equal(t, tt.expected.Message, response.Message, "message field should match")
		})
	}
}
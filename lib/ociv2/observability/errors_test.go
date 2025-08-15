package observability

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorCategory_String(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		category ErrorCategory
		expected string
	}{
		{ErrorCategoryAuth, "auth"},
		{ErrorCategoryNetwork, "network"},
		{ErrorCategoryRegistry, "registry"},
		{ErrorCategoryValidation, "validation"},
		{ErrorCategoryConfig, "config"},
		{ErrorCategoryCosign, "cosign"},
		{ErrorCategoryFallback, "fallback"},
		{ErrorCategoryUnknown, "unknown"},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			// ErrorCategory is just a string, so we test the string value directly
			assert.Equal(t, tt.expected, string(tt.category))
		})
	}
}

func TestOCIError(t *testing.T) {
	t.Parallel()
	
	t.Run("BasicError", func(t *testing.T) {
		baseErr := errors.New("base error")
		ociErr := NewOCIError(baseErr, ErrorCategoryNetwork, "push")
		
		assert.Equal(t, baseErr, ociErr.Err)
		assert.Equal(t, ErrorCategoryNetwork, ociErr.Category)
		assert.Equal(t, "push", ociErr.Operation)
		assert.NotEmpty(t, ociErr.Code)
		assert.NotEmpty(t, ociErr.Message)
		assert.False(t, ociErr.Timestamp.IsZero())
		
		// Test Error() method
		assert.Contains(t, ociErr.Error(), ociErr.Message)
	})
	
	t.Run("ErrorWithMessage", func(t *testing.T) {
		ociErr := &OCIError{
			Message: "custom message",
		}
		
		assert.Equal(t, "custom message", ociErr.Error())
	})
	
	t.Run("ErrorWithNoMessage", func(t *testing.T) {
		baseErr := errors.New("base error")
		ociErr := &OCIError{
			Err: baseErr,
		}
		
		assert.Equal(t, "base error", ociErr.Error())
	})
	
	t.Run("ErrorWithCode", func(t *testing.T) {
		ociErr := &OCIError{
			Code: "TEST_ERROR",
		}
		
		assert.Equal(t, "oci error: TEST_ERROR", ociErr.Error())
	})
	
	t.Run("WithContext", func(t *testing.T) {
		ociErr := NewOCIError(errors.New("test"), ErrorCategoryAuth, "push")
		
		updated := ociErr.WithContext("key", "value")
		
		assert.Equal(t, ociErr, updated) // Should return same instance
		assert.Equal(t, "value", ociErr.Metadata["key"])
	})
	
	t.Run("WithDuration", func(t *testing.T) {
		ociErr := NewOCIError(errors.New("test"), ErrorCategoryAuth, "push")
		duration := 100 * time.Millisecond
		
		updated := ociErr.WithDuration(duration)
		
		assert.Equal(t, ociErr, updated) // Should return same instance
		assert.Equal(t, duration, ociErr.Duration)
	})
	
	t.Run("Unwrap", func(t *testing.T) {
		baseErr := errors.New("base error")
		causeErr := errors.New("cause error")
		
		// Test unwrapping Err
		ociErr := &OCIError{
			Err: baseErr,
		}
		assert.Equal(t, baseErr, ociErr.Unwrap())
		
		// Test unwrapping Cause (takes precedence)
		ociErr.Cause = causeErr
		assert.Equal(t, causeErr, ociErr.Unwrap())
	})
	
	t.Run("Is", func(t *testing.T) {
		targetErr := errors.New("target error")
		baseErr := errors.New("base error")
		causeErr := errors.New("cause error")
		
		ociErr := &OCIError{
			Err:   baseErr,
			Cause: causeErr,
		}
		
		// Should match wrapped errors
		assert.False(t, ociErr.Is(targetErr))
		assert.True(t, ociErr.Is(baseErr))
		assert.True(t, ociErr.Is(causeErr))
	})
}

func TestSpecificErrorConstructors(t *testing.T) {
	t.Parallel()
	
	t.Run("NewAuthError", func(t *testing.T) {
		cause := errors.New("invalid token")
		ociErr := NewAuthError("push", "example.com", cause)
		
		assert.Equal(t, ErrAuthError, ociErr.Err)
		assert.Equal(t, cause, ociErr.Cause)
		assert.Equal(t, ErrorCategoryAuth, ociErr.Category)
		assert.Equal(t, "push", ociErr.Operation)
		assert.Equal(t, "example.com", ociErr.Registry)
		assert.Equal(t, "AUTH_FAILED", ociErr.Code)
		assert.Contains(t, ociErr.Message, "authentication failed")
		assert.Contains(t, ociErr.Message, "example.com")
	})
	
	t.Run("NewNetworkError", func(t *testing.T) {
		cause := errors.New("connection timeout")
		ociErr := NewNetworkError("pull", "registry.example.com", cause)
		
		assert.Equal(t, ErrNetworkError, ociErr.Err)
		assert.Equal(t, cause, ociErr.Cause)
		assert.Equal(t, ErrorCategoryNetwork, ociErr.Category)
		assert.Equal(t, "pull", ociErr.Operation)
		assert.Equal(t, "registry.example.com", ociErr.Registry)
		assert.Equal(t, "NETWORK_FAILED", ociErr.Code)
		assert.Contains(t, ociErr.Message, "network error")
	})
	
	t.Run("NewRegistryError", func(t *testing.T) {
		cause := errors.New("not found")
		ociErr := NewRegistryError("resolve", "example.com", http.StatusNotFound, cause)
		
		assert.Equal(t, ErrRegistryError, ociErr.Err)
		assert.Equal(t, cause, ociErr.Cause)
		assert.Equal(t, ErrorCategoryRegistry, ociErr.Category)
		assert.Equal(t, "resolve", ociErr.Operation)
		assert.Equal(t, "example.com", ociErr.Registry)
		assert.Equal(t, http.StatusNotFound, ociErr.HTTPStatus)
		assert.Equal(t, "REGISTRY_404", ociErr.Code)
		assert.Contains(t, ociErr.Message, "registry error 404")
	})
	
	t.Run("NewValidationError", func(t *testing.T) {
		cause := errors.New("invalid reference format")
		ociErr := NewValidationError("push", "invalid:ref", cause)
		
		assert.Equal(t, ErrValidationError, ociErr.Err)
		assert.Equal(t, cause, ociErr.Cause)
		assert.Equal(t, ErrorCategoryValidation, ociErr.Category)
		assert.Equal(t, "push", ociErr.Operation)
		assert.Equal(t, "invalid:ref", ociErr.Reference)
		assert.Equal(t, "VALIDATION_FAILED", ociErr.Code)
		assert.Contains(t, ociErr.Message, "validation failed")
	})
	
	t.Run("NewCosignError", func(t *testing.T) {
		cause := errors.New("signing failed")
		ociErr := NewCosignError("sign", "example.com/repo:tag", cause)
		
		assert.Equal(t, ErrCosignError, ociErr.Err)
		assert.Equal(t, cause, ociErr.Cause)
		assert.Equal(t, ErrorCategoryCosign, ociErr.Category)
		assert.Equal(t, "sign", ociErr.Operation)
		assert.Equal(t, "example.com/repo:tag", ociErr.Reference)
		assert.Equal(t, "COSIGN_FAILED", ociErr.Code)
		assert.Contains(t, ociErr.Message, "cosign operation failed")
	})
	
	t.Run("NewFallbackError", func(t *testing.T) {
		artifactErr := errors.New("artifact manifest rejected")
		imageErr := errors.New("image manifest failed")
		ociErr := NewFallbackError("push", "example.com/repo:tag", artifactErr, imageErr)
		
		assert.Equal(t, ErrFallbackFailed, ociErr.Err)
		assert.Equal(t, ErrorCategoryFallback, ociErr.Category)
		assert.Equal(t, "push", ociErr.Operation)
		assert.Equal(t, "example.com/repo:tag", ociErr.Reference)
		assert.Equal(t, "FALLBACK_FAILED", ociErr.Code)
		assert.Contains(t, ociErr.Message, "both artifact and image manifest failed")
		
		// Check metadata
		require.NotNil(t, ociErr.Metadata)
		assert.Equal(t, "artifact manifest rejected", ociErr.Metadata["artifact_error"])
		assert.Equal(t, "image manifest failed", ociErr.Metadata["image_error"])
	})
}

func TestErrorCodeGeneration(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		err      error
		category ErrorCategory
		expected string
	}{
		{"NotFound", ErrNotFound, ErrorCategoryRegistry, "NOT_FOUND"},
		{"Unauthorized", ErrUnauthorized, ErrorCategoryAuth, "UNAUTHORIZED"},
		{"Forbidden", ErrForbidden, ErrorCategoryAuth, "FORBIDDEN"},
		{"Timeout", ErrTimeout, ErrorCategoryNetwork, "TIMEOUT"},
		{"Canceled", context.Canceled, ErrorCategoryNetwork, "CANCELED"},
		{"InvalidRef", ErrInvalidRef, ErrorCategoryValidation, "INVALID_REF"},
		{"InsecureRef", ErrInsecureRef, ErrorCategoryValidation, "INSECURE_REF"},
		{"MediaType", ErrMediaType, ErrorCategoryValidation, "MEDIA_TYPE"},
		{"Unsupported", ErrUnsupported, ErrorCategoryRegistry, "UNSUPPORTED"},
		{"ManifestFormat", ErrManifestFormat, ErrorCategoryValidation, "MANIFEST_FORMAT"},
		{"Generic", errors.New("generic error"), ErrorCategoryAuth, "AUTH_ERROR"},
		{"NilError", nil, ErrorCategoryNetwork, "NETWORK_UNKNOWN"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ociErr := NewOCIError(tt.err, tt.category, "test_op")
			assert.Equal(t, tt.expected, ociErr.Code)
		})
	}
}

func TestErrorMessageGeneration(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		category ErrorCategory
		contains string
	}{
		{ErrorCategoryAuth, "authentication failed"},
		{ErrorCategoryNetwork, "network error"},
		{ErrorCategoryRegistry, "registry error"},
		{ErrorCategoryValidation, "validation failed"},
		{ErrorCategoryConfig, "configuration error"},
		{ErrorCategoryCosign, "cosign error"},
		{ErrorCategoryFallback, "fallback failed"},
		{ErrorCategoryUnknown, "error during"},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			err := errors.New("test error")
			ociErr := NewOCIError(err, tt.category, "test_operation")
			assert.Contains(t, ociErr.Message, tt.contains)
			assert.Contains(t, ociErr.Message, "test_operation")
		})
	}
}

func TestIsRetryable(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"NetworkError", NewNetworkError("push", "example.com", errors.New("timeout")), true},
		{"ServerError", NewRegistryError("push", "example.com", 500, errors.New("server error")), true},
		{"RateLimit", NewRegistryError("push", "example.com", 429, errors.New("rate limited")), true},
		{"ClientError", NewRegistryError("push", "example.com", 400, errors.New("bad request")), false},
		{"AuthError", NewAuthError("push", "example.com", errors.New("unauthorized")), false},
		{"ValidationError", NewValidationError("push", "invalid:ref", errors.New("bad ref")), false},
		{"TimeoutError", ErrTimeout, true},
		{"NetworkErrorDirect", ErrNetworkError, true},
		{"ContextDeadline", context.DeadlineExceeded, true},
		{"GenericError", errors.New("generic"), false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.retryable, IsRetryable(tt.err))
		})
	}
}

func TestIsTemporary(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name      string
		err       error
		temporary bool
	}{
		{"NetworkError", NewNetworkError("push", "example.com", errors.New("timeout")), true},
		{"ServerError", NewRegistryError("push", "example.com", 500, errors.New("server error")), true},
		{"RateLimit", NewRegistryError("push", "example.com", 429, errors.New("rate limited")), true},
		{"ClientError", NewRegistryError("push", "example.com", 400, errors.New("bad request")), false},
		{"AuthError", NewAuthError("push", "example.com", errors.New("unauthorized")), false},
		{"ValidationError", NewValidationError("push", "invalid:ref", errors.New("bad ref")), false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.temporary, IsTemporary(tt.err))
		})
	}
}

func TestExtractHTTPStatus(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"OCIErrorWithStatus", NewRegistryError("push", "example.com", 404, errors.New("not found")), 404},
		{"ErrorWith404", errors.New("HTTP 404 not found"), 404},
		{"ErrorWith401", errors.New("received 401 unauthorized"), 401},
		{"ErrorWith403", errors.New("status: 403 forbidden"), 403},
		{"ErrorWith415", errors.New("error 415 unsupported media type"), 415},
		{"ErrorWith500", errors.New("internal server error 500"), 500},
		{"NoHTTPStatus", errors.New("generic error"), 0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := ExtractHTTPStatus(tt.err)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestGetErrorCategory(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{"OCIError", NewAuthError("push", "example.com", errors.New("auth failed")), ErrorCategoryAuth},
		{"Unauthorized", ErrUnauthorized, ErrorCategoryAuth},
		{"Forbidden", ErrForbidden, ErrorCategoryAuth},
		{"Timeout", ErrTimeout, ErrorCategoryNetwork},
		{"ContextDeadline", context.DeadlineExceeded, ErrorCategoryNetwork},
		{"InvalidRef", ErrInvalidRef, ErrorCategoryValidation},
		{"InsecureRef", ErrInsecureRef, ErrorCategoryValidation},
		{"NotFound", ErrNotFound, ErrorCategoryRegistry},
		{"Unsupported", ErrUnsupported, ErrorCategoryRegistry},
		{"GenericError", errors.New("generic"), ErrorCategoryUnknown},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := GetErrorCategory(tt.err)
			assert.Equal(t, tt.expected, category)
		})
	}
}

func TestErrorChaining(t *testing.T) {
	t.Parallel()
	
	// Test that error chains work correctly with errors.Is and errors.As
	baseErr := errors.New("base error")
	ociErr := NewNetworkError("push", "example.com", baseErr)
	
	// Test errors.Is
	assert.True(t, errors.Is(ociErr, ErrNetworkError))
	assert.True(t, errors.Is(ociErr, baseErr))
	assert.False(t, errors.Is(ociErr, ErrAuthError))
	
	// Test errors.As
	var targetOCIErr *OCIError
	assert.True(t, errors.As(ociErr, &targetOCIErr))
	assert.Equal(t, ErrorCategoryNetwork, targetOCIErr.Category)
}
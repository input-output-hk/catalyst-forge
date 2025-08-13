package httpkit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResponseWriter(t *testing.T) {
	t.Parallel()
	
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)
	
	assert.NotNil(t, rw)
	assert.Equal(t, w, rw.w)
}

func TestResponseWriter_JSON(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name           string
		status         int
		data           any
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "ok/simple_object",
			status:         http.StatusOK,
			data:           map[string]string{"message": "success"},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"success"}`,
		},
		{
			name:           "ok/array_data",
			status:         http.StatusOK,
			data:           []string{"item1", "item2"},
			expectedStatus: http.StatusOK,
			expectedBody:   `["item1","item2"]`,
		},
		{
			name:           "ok/string_data",
			status:         http.StatusCreated,
			data:           "simple string",
			expectedStatus: http.StatusCreated,
			expectedBody:   `"simple string"`,
		},
		{
			name:           "ok/nil_data",
			status:         http.StatusNoContent,
			data:           nil,
			expectedStatus: http.StatusNoContent,
			expectedBody:   "null",
		},
		{
			name:           "ok/number_data",
			status:         http.StatusOK,
			data:           42,
			expectedStatus: http.StatusOK,
			expectedBody:   "42",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			
			err := rw.JSON(tc.status, tc.data)
			
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestResponseWriter_Success(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name         string
		data         any
		expectedBody string
	}{
		{
			name:         "ok/with_data",
			data:         map[string]string{"result": "success"},
			expectedBody: `{"result":"success"}`,
		},
		{
			name:         "ok/nil_data",
			data:         nil,
			expectedBody: "null",
		},
		{
			name: "ok/complex_data",
			data: SuccessResponse{
				Data:    map[string]int{"count": 5},
				Message: "Operation completed",
			},
			expectedBody: `{"data":{"count":5},"message":"Operation completed"}`,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			
			err := rw.Success(tc.data)
			
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestResponseWriter_Created(t *testing.T) {
	t.Parallel()
	
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)
	
	data := map[string]string{"id": "123", "status": "created"}
	err := rw.Created(data)
	
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"id":"123","status":"created"}`, w.Body.String())
}

func TestResponseWriter_NoContent(t *testing.T) {
	t.Parallel()
	
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)
	
	rw.NoContent()
	
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestResponseWriter_Error(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name           string
		status         int
		code           ErrorCode
		message        string
		expectedStatus int
		expectedBody   ErrorResponseData
	}{
		{
			name:           "ok/basic_error",
			status:         http.StatusBadRequest,
			code:           ErrorInvalidRequest,
			message:        "Invalid data provided",
			expectedStatus: http.StatusBadRequest,
			expectedBody: ErrorResponseData{
				Error:   ErrorInvalidRequest,
				Message: "Invalid data provided",
			},
		},
		{
			name:           "ok/unauthorized_error",
			status:         http.StatusUnauthorized,
			code:           ErrorUnauthorized,
			message:        "Invalid credentials",
			expectedStatus: http.StatusUnauthorized,
			expectedBody: ErrorResponseData{
				Error:   ErrorUnauthorized,
				Message: "Invalid credentials",
			},
		},
		{
			name:           "ok/custom_error_code",
			status:         http.StatusInternalServerError,
			code:           ErrorCode("custom_error"),
			message:        "Custom error occurred",
			expectedStatus: http.StatusInternalServerError,
			expectedBody: ErrorResponseData{
				Error:   ErrorCode("custom_error"),
				Message: "Custom error occurred",
			},
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			
			err := rw.Error(tc.status, tc.code, tc.message)
			
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			
			var response ErrorResponseData
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedBody, response)
		})
	}
}

func TestResponseWriter_ErrorWithDetails(t *testing.T) {
	t.Parallel()
	
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)
	
	details := map[string]string{"field": "email", "reason": "invalid format"}
	err := rw.ErrorWithDetails(http.StatusBadRequest, ErrorInvalidRequest, "Validation failed", details)
	
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	var response ErrorResponseData
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, ErrorInvalidRequest, response.Error)
	assert.Equal(t, "Validation failed", response.Message)
	assert.Equal(t, details, response.Details)
}

func TestResponseWriter_Unauthorized(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name            string
		message         string
		expectedMessage string
	}{
		{
			name:            "ok/custom_message",
			message:         "Token expired",
			expectedMessage: "Token expired",
		},
		{
			name:            "ok/empty_message_uses_default",
			message:         "",
			expectedMessage: "Authentication required",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			
			err := rw.Unauthorized(tc.message)
			
			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			
			var response ErrorResponseData
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.Equal(t, ErrorUnauthorized, response.Error)
			assert.Equal(t, tc.expectedMessage, response.Message)
		})
	}
}

func TestResponseWriter_Forbidden(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name            string
		message         string
		expectedMessage string
	}{
		{
			name:            "ok/custom_message",
			message:         "Insufficient permissions",
			expectedMessage: "Insufficient permissions",
		},
		{
			name:            "ok/empty_message_uses_default",
			message:         "",
			expectedMessage: "Access denied",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			
			err := rw.Forbidden(tc.message)
			
			require.NoError(t, err)
			assert.Equal(t, http.StatusForbidden, w.Code)
			
			var response ErrorResponseData
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.Equal(t, ErrorForbidden, response.Error)
			assert.Equal(t, tc.expectedMessage, response.Message)
		})
	}
}

func TestResponseWriter_BadRequest(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name            string
		message         string
		expectedMessage string
	}{
		{
			name:            "ok/custom_message",
			message:         "Missing required field",
			expectedMessage: "Missing required field",
		},
		{
			name:            "ok/empty_message_uses_default",
			message:         "",
			expectedMessage: "Invalid request",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			
			err := rw.BadRequest(tc.message)
			
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			
			var response ErrorResponseData
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.Equal(t, ErrorInvalidRequest, response.Error)
			assert.Equal(t, tc.expectedMessage, response.Message)
		})
	}
}

func TestResponseWriter_NotFound(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name            string
		message         string
		expectedMessage string
	}{
		{
			name:            "ok/custom_message",
			message:         "User not found",
			expectedMessage: "User not found",
		},
		{
			name:            "ok/empty_message_uses_default",
			message:         "",
			expectedMessage: "Resource not found",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			
			err := rw.NotFound(tc.message)
			
			require.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, w.Code)
			
			var response ErrorResponseData
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.Equal(t, ErrorNotFound, response.Error)
			assert.Equal(t, tc.expectedMessage, response.Message)
		})
	}
}

func TestResponseWriter_RateLimited(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name       string
		retryAfter int
	}{
		{
			name:       "ok/short_retry",
			retryAfter: 60,
		},
		{
			name:       "ok/long_retry",
			retryAfter: 3600,
		},
		{
			name:       "ok/zero_retry",
			retryAfter: 0,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			
			err := rw.RateLimited(tc.retryAfter)
			
			require.NoError(t, err)
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
			assert.Equal(t, string(rune(tc.retryAfter)), w.Header().Get("Retry-After"))
			
			var response ErrorResponseData
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.Equal(t, ErrorRateLimitExceeded, response.Error)
			assert.Equal(t, "Too many attempts. Please try again later.", response.Message)
			assert.Equal(t, tc.retryAfter, response.RetryAfter)
		})
	}
}

func TestResponseWriter_StepUpRequired(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name            string
		message         string
		expectedMessage string
	}{
		{
			name:            "ok/custom_message",
			message:         "Admin action requires confirmation",
			expectedMessage: "Admin action requires confirmation",
		},
		{
			name:            "ok/empty_message_uses_default",
			message:         "",
			expectedMessage: "Please confirm with your passkey to continue",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			
			err := rw.StepUpRequired(tc.message)
			
			require.NoError(t, err)
			assert.Equal(t, http.StatusPreconditionRequired, w.Code)
			
			var response ErrorResponseData
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.Equal(t, ErrorStepUpRequired, response.Error)
			assert.Equal(t, tc.expectedMessage, response.Message)
		})
	}
}

func TestResponseWriter_InternalError(t *testing.T) {
	t.Parallel()
	
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)
	
	err := rw.InternalError()
	
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response ErrorResponseData
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, ErrorInternal, response.Error)
	assert.Equal(t, "An error occurred", response.Message)
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name           string
		status         int
		data           any
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "ok/object_data",
			status:         http.StatusOK,
			data:           map[string]string{"key": "value"},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"key":"value"}`,
		},
		{
			name:           "ok/array_data",
			status:         http.StatusCreated,
			data:           []int{1, 2, 3},
			expectedStatus: http.StatusCreated,
			expectedBody:   `[1,2,3]`,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			w := httptest.NewRecorder()
			
			err := WriteJSON(w, tc.status, tc.data)
			
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestWriteError(t *testing.T) {
	t.Parallel()
	
	w := httptest.NewRecorder()
	
	err := WriteError(w, http.StatusBadRequest, ErrorInvalidRequest, "Test error message")
	
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	var response ErrorResponseData
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, ErrorInvalidRequest, response.Error)
	assert.Equal(t, "Test error message", response.Message)
}

func TestWriteSuccess(t *testing.T) {
	t.Parallel()
	
	w := httptest.NewRecorder()
	data := map[string]string{"result": "success"}
	
	err := WriteSuccess(w, data)
	
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"result":"success"}`, w.Body.String())
}

func TestWriteNoContent(t *testing.T) {
	t.Parallel()
	
	w := httptest.NewRecorder()
	
	WriteNoContent(w)
	
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestSafeError(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ok/any_error_returns_safe_message",
			err:      assert.AnError,
			expected: "Invalid credentials",
		},
		{
			name:     "ok/nil_error_returns_safe_message",
			err:      nil,
			expected: "Invalid credentials",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			result := SafeError(tc.err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsAuthError(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		code     ErrorCode
		expected bool
	}{
		{
			name:     "ok/unauthorized_is_auth_error",
			code:     ErrorUnauthorized,
			expected: true,
		},
		{
			name:     "ok/token_expired_is_auth_error",
			code:     ErrorTokenExpired,
			expected: true,
		},
		{
			name:     "ok/token_invalid_is_auth_error",
			code:     ErrorTokenInvalid,
			expected: true,
		},
		{
			name:     "ok/session_expired_is_auth_error",
			code:     ErrorSessionExpired,
			expected: true,
		},
		{
			name:     "ok/forbidden_is_not_auth_error",
			code:     ErrorForbidden,
			expected: false,
		},
		{
			name:     "ok/invalid_request_is_not_auth_error",
			code:     ErrorInvalidRequest,
			expected: false,
		},
		{
			name:     "ok/rate_limit_is_not_auth_error",
			code:     ErrorRateLimitExceeded,
			expected: false,
		},
		{
			name:     "ok/webauthn_failed_is_not_auth_error",
			code:     ErrorWebAuthnFailed,
			expected: false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			result := IsAuthError(tc.code)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsValidationError(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		code     ErrorCode
		expected bool
	}{
		{
			name:     "ok/invalid_request_is_validation_error",
			code:     ErrorInvalidRequest,
			expected: true,
		},
		{
			name:     "ok/csrf_protection_is_validation_error",
			code:     ErrorCSRFProtection,
			expected: true,
		},
		{
			name:     "ok/rate_limit_is_validation_error",
			code:     ErrorRateLimitExceeded,
			expected: true,
		},
		{
			name:     "ok/unauthorized_is_not_validation_error",
			code:     ErrorUnauthorized,
			expected: false,
		},
		{
			name:     "ok/forbidden_is_not_validation_error",
			code:     ErrorForbidden,
			expected: false,
		},
		{
			name:     "ok/webauthn_failed_is_not_validation_error",
			code:     ErrorWebAuthnFailed,
			expected: false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			result := IsValidationError(tc.code)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestErrorResponse(t *testing.T) {
	t.Parallel()
	
	w := httptest.NewRecorder()
	
	ErrorResponse(w, http.StatusUnauthorized, "custom_error", "Custom error message")
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	var response ErrorResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, ErrorCode("custom_error"), response.Error)
	assert.Equal(t, "Custom error message", response.Message)
}

func TestErrorCodes_Constants(t *testing.T) {
	t.Parallel()
	
	// Verify error code constants are defined correctly
	assert.Equal(t, ErrorCode("unauthorized"), ErrorUnauthorized)
	assert.Equal(t, ErrorCode("token_expired"), ErrorTokenExpired)
	assert.Equal(t, ErrorCode("token_invalid"), ErrorTokenInvalid)
	assert.Equal(t, ErrorCode("session_expired"), ErrorSessionExpired)
	assert.Equal(t, ErrorCode("step_up_required"), ErrorStepUpRequired)
	assert.Equal(t, ErrorCode("forbidden"), ErrorForbidden)
	assert.Equal(t, ErrorCode("insufficient_roles"), ErrorInsufficientRoles)
	assert.Equal(t, ErrorCode("invalid_request"), ErrorInvalidRequest)
	assert.Equal(t, ErrorCode("csrf_protection"), ErrorCSRFProtection)
	assert.Equal(t, ErrorCode("rate_limit_exceeded"), ErrorRateLimitExceeded)
	assert.Equal(t, ErrorCode("webauthn_failed"), ErrorWebAuthnFailed)
	assert.Equal(t, ErrorCode("invalid_credential"), ErrorInvalidCredential)
	assert.Equal(t, ErrorCode("credential_exists"), ErrorCredentialExists)
	assert.Equal(t, ErrorCode("invite_expired"), ErrorInviteExpired)
	assert.Equal(t, ErrorCode("invite_invalid"), ErrorInviteInvalid)
	assert.Equal(t, ErrorCode("invite_redeemed"), ErrorInviteRedeemed)
	assert.Equal(t, ErrorCode("invite_locked"), ErrorInviteLocked)
	assert.Equal(t, ErrorCode("recovery_invalid"), ErrorRecoveryInvalid)
	assert.Equal(t, ErrorCode("recovery_expired"), ErrorRecoveryExpired)
	assert.Equal(t, ErrorCode("code_invalid"), ErrorCodeInvalid)
	assert.Equal(t, ErrorCode("internal_error"), ErrorInternal)
	assert.Equal(t, ErrorCode("not_found"), ErrorNotFound)
	assert.Equal(t, ErrorCode("method_not_allowed"), ErrorMethodNotAllowed)
}

func TestErrorResponseData_JSONSerialization(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		data     ErrorResponseData
		expected string
	}{
		{
			name: "ok/basic_error",
			data: ErrorResponseData{
				Error:   ErrorInvalidRequest,
				Message: "Test message",
			},
			expected: `{"error":"invalid_request","message":"Test message"}`,
		},
		{
			name: "ok/error_with_retry_after",
			data: ErrorResponseData{
				Error:      ErrorRateLimitExceeded,
				Message:    "Rate limited",
				RetryAfter: 60,
			},
			expected: `{"error":"rate_limit_exceeded","message":"Rate limited","retry_after":60}`,
		},
		{
			name: "ok/error_with_details",
			data: ErrorResponseData{
				Error:   ErrorInvalidRequest,
				Message: "Validation failed",
				Details: map[string]string{"field": "email"},
			},
			expected: `{"error":"invalid_request","message":"Validation failed","details":{"field":"email"}}`,
		},
		{
			name: "ok/error_with_all_fields",
			data: ErrorResponseData{
				Error:      ErrorCSRFProtection,
				Message:    "CSRF token invalid",
				RetryAfter: 30,
				Details:    []string{"token", "timestamp"},
			},
			expected: `{"error":"csrf_protection","message":"CSRF token invalid","retry_after":30,"details":["token","timestamp"]}`,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			jsonData, err := json.Marshal(tc.data)
			require.NoError(t, err)
			assert.JSONEq(t, tc.expected, string(jsonData))
			
			// Test round-trip
			var unmarshaled ErrorResponseData
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, tc.data, unmarshaled)
		})
	}
}

func TestSuccessResponse_JSONSerialization(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		data     SuccessResponse
		expected string
	}{
		{
			name: "ok/data_only",
			data: SuccessResponse{
				Data: map[string]int{"count": 5},
			},
			expected: `{"data":{"count":5}}`,
		},
		{
			name: "ok/message_only",
			data: SuccessResponse{
				Message: "Operation successful",
			},
			expected: `{"message":"Operation successful"}`,
		},
		{
			name: "ok/data_and_message",
			data: SuccessResponse{
				Data:    []string{"item1", "item2"},
				Message: "Items retrieved",
			},
			expected: `{"data":["item1","item2"],"message":"Items retrieved"}`,
		},
		{
			name: "ok/empty_response",
			data: SuccessResponse{},
			expected: `{}`,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			jsonData, err := json.Marshal(tc.data)
			require.NoError(t, err)
			assert.JSONEq(t, tc.expected, string(jsonData))
			
			// Test round-trip
			var unmarshaled SuccessResponse
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, tc.data, unmarshaled)
		})
	}
}

func TestResponseWriter_EnumerationSafety(t *testing.T) {
	t.Parallel()
	
	// Test that error responses don't leak sensitive information
	tests := []struct {
		name        string
		errorFunc   func(*ResponseWriter) error
		expectGeneric bool
	}{
		{
			name: "ok/internal_error_is_generic",
			errorFunc: func(rw *ResponseWriter) error {
				return rw.InternalError()
			},
			expectGeneric: true,
		},
		{
			name: "ok/safe_error_function",
			errorFunc: func(rw *ResponseWriter) error {
				// SafeError should always return generic message
				safeMsg := SafeError(assert.AnError)
				return rw.Unauthorized(safeMsg)
			},
			expectGeneric: true,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)
			
			err := tc.errorFunc(rw)
			require.NoError(t, err)
			
			var response ErrorResponseData
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			if tc.expectGeneric {
				// Verify response doesn't contain sensitive details
				assert.NotContains(t, strings.ToLower(response.Message), "database")
				assert.NotContains(t, strings.ToLower(response.Message), "sql")
				assert.NotContains(t, strings.ToLower(response.Message), "connection")
				assert.NotContains(t, strings.ToLower(response.Message), "server")
				assert.NotContains(t, strings.ToLower(response.Message), "file")
				assert.NotContains(t, strings.ToLower(response.Message), "path")
			}
		})
	}
}

func TestResponseWriter_JSONEncodingErrors(t *testing.T) {
	t.Parallel()
	
	// Test behavior with data that can't be JSON encoded
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)
	
	// Use a channel which can't be JSON encoded
	invalidData := make(chan int)
	
	err := rw.JSON(http.StatusOK, invalidData)
	
	// Should return an error for invalid JSON data
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "json")
}
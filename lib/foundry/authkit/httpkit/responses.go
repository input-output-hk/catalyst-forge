package httpkit

import (
	"encoding/json"
	"net/http"
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	// Authentication errors
	ErrorUnauthorized      ErrorCode = "unauthorized"
	ErrorTokenExpired      ErrorCode = "token_expired"
	ErrorTokenInvalid      ErrorCode = "token_invalid"
	ErrorSessionExpired    ErrorCode = "session_expired"
	ErrorStepUpRequired    ErrorCode = "step_up_required"
	
	// Authorization errors
	ErrorForbidden         ErrorCode = "forbidden"
	ErrorInsufficientRoles ErrorCode = "insufficient_roles"
	
	// Validation errors
	ErrorInvalidRequest    ErrorCode = "invalid_request"
	ErrorCSRFProtection    ErrorCode = "csrf_protection"
	ErrorRateLimitExceeded ErrorCode = "rate_limit_exceeded"
	
	// WebAuthn errors
	ErrorWebAuthnFailed    ErrorCode = "webauthn_failed"
	ErrorInvalidCredential ErrorCode = "invalid_credential"
	ErrorCredentialExists  ErrorCode = "credential_exists"
	
	// Invite errors
	ErrorInviteExpired     ErrorCode = "invite_expired"
	ErrorInviteInvalid     ErrorCode = "invite_invalid"
	ErrorInviteRedeemed    ErrorCode = "invite_redeemed"
	ErrorInviteLocked      ErrorCode = "invite_locked"
	
	// Recovery errors
	ErrorRecoveryInvalid   ErrorCode = "recovery_invalid"
	ErrorRecoveryExpired   ErrorCode = "recovery_expired"
	ErrorCodeInvalid       ErrorCode = "code_invalid"
	
	// Generic errors
	ErrorInternal          ErrorCode = "internal_error"
	ErrorNotFound          ErrorCode = "not_found"
	ErrorMethodNotAllowed  ErrorCode = "method_not_allowed"
)

// ErrorResponseData represents a standardized error response
type ErrorResponseData struct {
	Error      ErrorCode `json:"error"`
	Message    string    `json:"message"`
	RetryAfter int       `json:"retry_after,omitempty"` // For rate limiting (seconds)
	Details    any       `json:"details,omitempty"`      // Optional additional details
}

// SuccessResponse represents a successful response
type SuccessResponse struct {
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// ResponseWriter provides consistent JSON responses
type ResponseWriter struct {
	w http.ResponseWriter
}

// NewResponseWriter creates a new response writer
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{w: w}
}

// JSON writes a JSON response with the given status code
func (rw *ResponseWriter) JSON(status int, data any) error {
	rw.w.Header().Set("Content-Type", "application/json")
	rw.w.WriteHeader(status)
	return json.NewEncoder(rw.w).Encode(data)
}

// Success writes a successful response
func (rw *ResponseWriter) Success(data any) error {
	return rw.JSON(http.StatusOK, data)
}

// Created writes a created response
func (rw *ResponseWriter) Created(data any) error {
	return rw.JSON(http.StatusCreated, data)
}

// NoContent writes a no content response
func (rw *ResponseWriter) NoContent() {
	rw.w.WriteHeader(http.StatusNoContent)
}

// Error writes an error response
func (rw *ResponseWriter) Error(status int, code ErrorCode, message string) error {
	return rw.JSON(status, ErrorResponseData{
		Error:   code,
		Message: message,
	})
}

// ErrorWithDetails writes an error response with additional details
func (rw *ResponseWriter) ErrorWithDetails(status int, code ErrorCode, message string, details any) error {
	return rw.JSON(status, ErrorResponseData{
		Error:   code,
		Message: message,
		Details: details,
	})
}

// Unauthorized writes a 401 response
func (rw *ResponseWriter) Unauthorized(message string) error {
	if message == "" {
		message = "Authentication required"
	}
	return rw.Error(http.StatusUnauthorized, ErrorUnauthorized, message)
}

// Forbidden writes a 403 response
func (rw *ResponseWriter) Forbidden(message string) error {
	if message == "" {
		message = "Access denied"
	}
	return rw.Error(http.StatusForbidden, ErrorForbidden, message)
}

// BadRequest writes a 400 response
func (rw *ResponseWriter) BadRequest(message string) error {
	if message == "" {
		message = "Invalid request"
	}
	return rw.Error(http.StatusBadRequest, ErrorInvalidRequest, message)
}

// NotFound writes a 404 response
func (rw *ResponseWriter) NotFound(message string) error {
	if message == "" {
		message = "Resource not found"
	}
	return rw.Error(http.StatusNotFound, ErrorNotFound, message)
}

// RateLimited writes a 429 response with retry-after
func (rw *ResponseWriter) RateLimited(retryAfter int) error {
	rw.w.Header().Set("Retry-After", string(rune(retryAfter)))
	return rw.JSON(http.StatusTooManyRequests, ErrorResponseData{
		Error:      ErrorRateLimitExceeded,
		Message:    "Too many attempts. Please try again later.",
		RetryAfter: retryAfter,
	})
}

// StepUpRequired writes a 428 response
func (rw *ResponseWriter) StepUpRequired(message string) error {
	if message == "" {
		message = "Please confirm with your passkey to continue"
	}
	return rw.Error(http.StatusPreconditionRequired, ErrorStepUpRequired, message)
}

// InternalError writes a 500 response (enumeration-safe)
func (rw *ResponseWriter) InternalError() error {
	return rw.Error(http.StatusInternalServerError, ErrorInternal, "An error occurred")
}

// Helper functions for consistent responses without ResponseWriter

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, status int, code ErrorCode, message string) error {
	return WriteJSON(w, status, ErrorResponseData{
		Error:   code,
		Message: message,
	})
}

// WriteSuccess writes a success response
func WriteSuccess(w http.ResponseWriter, data any) error {
	return WriteJSON(w, http.StatusOK, data)
}

// WriteNoContent writes a no content response
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// SafeError returns an enumeration-safe error message
func SafeError(err error) string {
	// Return generic message to prevent enumeration attacks
	return "Invalid credentials"
}

// IsAuthError checks if the error code is authentication-related
func IsAuthError(code ErrorCode) bool {
	switch code {
	case ErrorUnauthorized, ErrorTokenExpired, ErrorTokenInvalid, ErrorSessionExpired:
		return true
	default:
		return false
	}
}

// IsValidationError checks if the error code is validation-related
func IsValidationError(code ErrorCode) bool {
	switch code {
	case ErrorInvalidRequest, ErrorCSRFProtection, ErrorRateLimitExceeded:
		return true
	default:
		return false
	}
}

// ErrorResponse writes an error response (compatibility function for middleware)
func ErrorResponse(w http.ResponseWriter, status int, code string, message string) {
	WriteJSON(w, status, ErrorResponseData{
		Error:   ErrorCode(code),
		Message: message,
	})
}
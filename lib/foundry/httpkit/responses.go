package httpkit

import (
    "encoding/json"
    "net/http"
)

// ErrorCode represents standardized, generic error codes.
type ErrorCode string

const (
    // Generic authn/z and request semantics (no domain-specific values here)
    ErrorUnauthorized     ErrorCode = "unauthorized"
    ErrorForbidden        ErrorCode = "forbidden"
    ErrorInvalidRequest   ErrorCode = "invalid_request"
    ErrorCSRFProtection   ErrorCode = "csrf_protection"
    ErrorRateLimitExceeded ErrorCode = "rate_limit_exceeded"
    ErrorNotFound         ErrorCode = "not_found"
    ErrorMethodNotAllowed ErrorCode = "method_not_allowed"
    ErrorInternal         ErrorCode = "internal_error"
)

// ErrorResponseData represents a standardized error response payload.
type ErrorResponseData struct {
    Error      ErrorCode `json:"error"`
    Message    string    `json:"message"`
    RetryAfter int       `json:"retry_after,omitempty"`
    Details    any       `json:"details,omitempty"`
}

// SuccessResponse represents a successful response payload.
type SuccessResponse struct {
    Data    any    `json:"data,omitempty"`
    Message string `json:"message,omitempty"`
}

// ResponseWriter provides consistent JSON responses
type ResponseWriter struct{ w http.ResponseWriter }

// NewResponseWriter creates a new response writer
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter { return &ResponseWriter{w: w} }

// JSON writes a JSON response with the given status code
func (rw *ResponseWriter) JSON(status int, data any) error {
    rw.w.Header().Set("Content-Type", "application/json")
    rw.w.WriteHeader(status)
    return json.NewEncoder(rw.w).Encode(data)
}

// Success writes a successful response
func (rw *ResponseWriter) Success(data any) error { return rw.JSON(http.StatusOK, data) }

// Created writes a created response
func (rw *ResponseWriter) Created(data any) error { return rw.JSON(http.StatusCreated, data) }

// NoContent writes a no content response
func (rw *ResponseWriter) NoContent() { rw.w.WriteHeader(http.StatusNoContent) }

// ErrorResponse writes an error response (compatibility function)
func ErrorResponse(w http.ResponseWriter, status int, code string, message string) {
    WriteJSON(w, status, ErrorResponseData{Error: ErrorCode(code), Message: message})
}

// WriteJSON writes a JSON response using the standard library
func WriteJSON(w http.ResponseWriter, status int, payload any) error {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    return json.NewEncoder(w).Encode(payload)
}


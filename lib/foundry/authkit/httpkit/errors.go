package httpkit

import (
	"fmt"
	"net/http"
)

// HTTPError represents an HTTP error with status code and error code
type HTTPError struct {
	Status  int
	Code    ErrorCode
	Message string
	Details any
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s - %s", e.Status, e.Code, e.Message)
}

// Write writes the error as an HTTP response
func (e *HTTPError) Write(w http.ResponseWriter) error {
	return WriteJSON(w, e.Status, ErrorResponseData{
		Error:   e.Code,
		Message: e.Message,
		Details: e.Details,
	})
}

// Common HTTP errors

// NewUnauthorizedError creates a 401 error
func NewUnauthorizedError(message string) *HTTPError {
	if message == "" {
		message = "Authentication required"
	}
	return &HTTPError{
		Status:  http.StatusUnauthorized,
		Code:    ErrorUnauthorized,
		Message: message,
	}
}

// NewForbiddenError creates a 403 error
func NewForbiddenError(message string) *HTTPError {
	if message == "" {
		message = "Access denied"
	}
	return &HTTPError{
		Status:  http.StatusForbidden,
		Code:    ErrorForbidden,
		Message: message,
	}
}

// NewBadRequestError creates a 400 error
func NewBadRequestError(message string) *HTTPError {
	if message == "" {
		message = "Invalid request"
	}
	return &HTTPError{
		Status:  http.StatusBadRequest,
		Code:    ErrorInvalidRequest,
		Message: message,
	}
}

// NewNotFoundError creates a 404 error
func NewNotFoundError(message string) *HTTPError {
	if message == "" {
		message = "Resource not found"
	}
	return &HTTPError{
		Status:  http.StatusNotFound,
		Code:    ErrorNotFound,
		Message: message,
	}
}

// NewRateLimitError creates a 429 error
func NewRateLimitError(retryAfter int) *HTTPError {
	return &HTTPError{
		Status:  http.StatusTooManyRequests,
		Code:    ErrorRateLimitExceeded,
		Message: "Too many attempts. Please try again later.",
		Details: map[string]int{"retry_after": retryAfter},
	}
}

// NewStepUpRequiredError creates a 428 error
func NewStepUpRequiredError(message string) *HTTPError {
	if message == "" {
		message = "Please confirm with your passkey to continue"
	}
	return &HTTPError{
		Status:  http.StatusPreconditionRequired,
		Code:    ErrorStepUpRequired,
		Message: message,
	}
}

// NewInternalError creates a 500 error (enumeration-safe)
func NewInternalError() *HTTPError {
	return &HTTPError{
		Status:  http.StatusInternalServerError,
		Code:    ErrorInternal,
		Message: "An error occurred",
	}
}

// NewCSRFError creates a CSRF validation error
func NewCSRFError(message string) *HTTPError {
	if message == "" {
		message = "Missing required security header"
	}
	return &HTTPError{
		Status:  http.StatusForbidden,
		Code:    ErrorCSRFProtection,
		Message: message,
	}
}

// ErrorHandler handles errors in a consistent way
type ErrorHandler func(w http.ResponseWriter, r *http.Request) error

// Wrap wraps an error handler to handle errors consistently
func Wrap(h ErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			// Check if it's an HTTPError
			if httpErr, ok := err.(*HTTPError); ok {
				_ = httpErr.Write(w)
				return
			}
			
			// Generic internal error for unknown errors
			_ = NewInternalError().Write(w)
		}
	}
}

// HandleError writes an appropriate error response based on the error type
func HandleError(w http.ResponseWriter, err error) {
	// Check if it's an HTTPError
	if httpErr, ok := err.(*HTTPError); ok {
		_ = httpErr.Write(w)
		return
	}
	
	// Check for specific error types
	switch err {
	case ErrCSRFTokenMissing, ErrCSRFTokenInvalid, ErrCSRFHeaderMissing:
		_ = NewCSRFError("").Write(w)
	default:
		// Generic internal error for unknown errors
		_ = NewInternalError().Write(w)
	}
}
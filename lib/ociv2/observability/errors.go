package observability

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Standard error variables for consistent error handling
var (
	// Registry errors
	ErrNotFound     = errors.New("oci: not found")
	ErrUnauthorized = errors.New("oci: unauthorized")
	ErrForbidden    = errors.New("oci: forbidden")
	ErrTimeout      = errors.New("oci: timeout")
	
	// Content errors
	ErrMediaType   = errors.New("oci: unexpected media type")
	ErrInsecureRef = errors.New("oci: insecure ref")
	ErrInvalidRef  = errors.New("oci: invalid reference")
	
	// Operation errors
	ErrUnsupported = errors.New("oci: unsupported")
	ErrCanceled    = errors.New("oci: canceled")
)

// Enhanced error variables for better categorization
var (
	ErrManifestFormat   = errors.New("oci: unsupported manifest format")
	ErrNetworkError     = errors.New("oci: network error")
	ErrRegistryError    = errors.New("oci: registry error")
	ErrAuthError        = errors.New("oci: authentication error")
	ErrValidationError  = errors.New("oci: validation error")
	ErrConfigError      = errors.New("oci: configuration error")
	ErrCosignError      = errors.New("oci: cosign error")
	ErrFallbackFailed   = errors.New("oci: fallback failed")
)

// ErrorCategory represents different categories of errors
type ErrorCategory string

const (
	ErrorCategoryAuth       ErrorCategory = "auth"
	ErrorCategoryNetwork    ErrorCategory = "network"
	ErrorCategoryRegistry   ErrorCategory = "registry"
	ErrorCategoryValidation ErrorCategory = "validation"
	ErrorCategoryConfig     ErrorCategory = "config"
	ErrorCategoryCosign     ErrorCategory = "cosign"
	ErrorCategoryFallback   ErrorCategory = "fallback"
	ErrorCategoryUnknown    ErrorCategory = "unknown"
)

// OCIError provides structured error information with context
type OCIError struct {
	// Core error information
	Err      error         `json:"error"`
	Category ErrorCategory `json:"category"`
	Code     string        `json:"code"`
	Message  string        `json:"message"`
	
	// Operation context
	Operation string `json:"operation"`
	Reference string `json:"reference,omitempty"`
	Registry  string `json:"registry,omitempty"`
	
	// HTTP context (if applicable)
	HTTPStatus int    `json:"http_status,omitempty"`
	HTTPMethod string `json:"http_method,omitempty"`
	
	// Additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	
	// Timing information
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
	
	// Error chain
	Cause error `json:"-"` // Original cause
}

// Error implements the error interface
func (e *OCIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("oci error: %s", e.Code)
}

// Unwrap implements error unwrapping for error chains
func (e *OCIError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return e.Err
}

// Is implements error comparison for errors.Is()
func (e *OCIError) Is(target error) bool {
	if e.Err != nil && errors.Is(e.Err, target) {
		return true
	}
	if e.Cause != nil && errors.Is(e.Cause, target) {
		return true
	}
	return false
}

// WithContext adds additional context to the error
func (e *OCIError) WithContext(key string, value interface{}) *OCIError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithDuration sets the operation duration
func (e *OCIError) WithDuration(d time.Duration) *OCIError {
	e.Duration = d
	return e
}

// NewOCIError creates a new structured OCI error
func NewOCIError(err error, category ErrorCategory, operation string) *OCIError {
	return &OCIError{
		Err:       err,
		Category:  category,
		Operation: operation,
		Timestamp: time.Now(),
		Code:      generateErrorCode(category, err),
		Message:   generateErrorMessage(category, operation, err),
	}
}

// NewAuthError creates an authentication error
func NewAuthError(operation string, registry string, cause error) *OCIError {
	return &OCIError{
		Err:       ErrAuthError,
		Cause:     cause,
		Category:  ErrorCategoryAuth,
		Operation: operation,
		Registry:  registry,
		Timestamp: time.Now(),
		Code:      "AUTH_FAILED",
		Message:   fmt.Sprintf("authentication failed for %s: %v", registry, cause),
	}
}

// NewNetworkError creates a network error
func NewNetworkError(operation string, registry string, cause error) *OCIError {
	return &OCIError{
		Err:       ErrNetworkError,
		Cause:     cause,
		Category:  ErrorCategoryNetwork,
		Operation: operation,
		Registry:  registry,
		Timestamp: time.Now(),
		Code:      "NETWORK_FAILED",
		Message:   fmt.Sprintf("network error for %s: %v", registry, cause),
	}
}

// NewRegistryError creates a registry-specific error
func NewRegistryError(operation string, registry string, httpStatus int, cause error) *OCIError {
	return &OCIError{
		Err:        ErrRegistryError,
		Cause:      cause,
		Category:   ErrorCategoryRegistry,
		Operation:  operation,
		Registry:   registry,
		HTTPStatus: httpStatus,
		Timestamp:  time.Now(),
		Code:       fmt.Sprintf("REGISTRY_%d", httpStatus),
		Message:    fmt.Sprintf("registry error %d for %s: %v", httpStatus, registry, cause),
	}
}

// NewValidationError creates a validation error
func NewValidationError(operation string, reference string, cause error) *OCIError {
	return &OCIError{
		Err:       ErrValidationError,
		Cause:     cause,
		Category:  ErrorCategoryValidation,
		Operation: operation,
		Reference: reference,
		Timestamp: time.Now(),
		Code:      "VALIDATION_FAILED",
		Message:   fmt.Sprintf("validation failed for %s: %v", reference, cause),
	}
}

// NewCosignError creates a Cosign-related error
func NewCosignError(operation string, reference string, cause error) *OCIError {
	return &OCIError{
		Err:       ErrCosignError,
		Cause:     cause,
		Category:  ErrorCategoryCosign,
		Operation: operation,
		Reference: reference,
		Timestamp: time.Now(),
		Code:      "COSIGN_FAILED",
		Message:   fmt.Sprintf("cosign operation failed for %s: %v", reference, cause),
	}
}

// NewFallbackError creates a fallback failure error
func NewFallbackError(operation string, reference string, artifactErr, imageErr error) *OCIError {
	return &OCIError{
		Err:       ErrFallbackFailed,
		Category:  ErrorCategoryFallback,
		Operation: operation,
		Reference: reference,
		Timestamp: time.Now(),
		Code:      "FALLBACK_FAILED",
		Message:   fmt.Sprintf("both artifact and image manifest failed for %s", reference),
		Metadata: map[string]interface{}{
			"artifact_error": artifactErr.Error(),
			"image_error":    imageErr.Error(),
		},
	}
}

// generateErrorCode creates a standardized error code
func generateErrorCode(category ErrorCategory, err error) string {
	if err == nil {
		return strings.ToUpper(string(category)) + "_UNKNOWN"
	}
	
	// Map common errors to codes
	switch {
	case errors.Is(err, ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, ErrUnauthorized):
		return "UNAUTHORIZED"
	case errors.Is(err, ErrForbidden):
		return "FORBIDDEN"
	case errors.Is(err, ErrTimeout):
		return "TIMEOUT"
	case errors.Is(err, context.Canceled):
		return "CANCELED"
	case errors.Is(err, ErrInvalidRef):
		return "INVALID_REF"
	case errors.Is(err, ErrInsecureRef):
		return "INSECURE_REF"
	case errors.Is(err, ErrMediaType):
		return "MEDIA_TYPE"
	case errors.Is(err, ErrUnsupported):
		return "UNSUPPORTED"
	case errors.Is(err, ErrManifestFormat):
		return "MANIFEST_FORMAT"
	default:
		return strings.ToUpper(string(category)) + "_ERROR"
	}
}

// generateErrorMessage creates a human-readable error message
func generateErrorMessage(category ErrorCategory, operation string, err error) string {
	if err == nil {
		return fmt.Sprintf("%s operation failed", operation)
	}
	
	switch category {
	case ErrorCategoryAuth:
		return fmt.Sprintf("authentication failed during %s: %v", operation, err)
	case ErrorCategoryNetwork:
		return fmt.Sprintf("network error during %s: %v", operation, err)
	case ErrorCategoryRegistry:
		return fmt.Sprintf("registry error during %s: %v", operation, err)
	case ErrorCategoryValidation:
		return fmt.Sprintf("validation failed during %s: %v", operation, err)
	case ErrorCategoryConfig:
		return fmt.Sprintf("configuration error during %s: %v", operation, err)
	case ErrorCategoryCosign:
		return fmt.Sprintf("cosign error during %s: %v", operation, err)
	case ErrorCategoryFallback:
		return fmt.Sprintf("fallback failed during %s: %v", operation, err)
	default:
		return fmt.Sprintf("error during %s: %v", operation, err)
	}
}

// IsRetryable determines if an error might be resolved by retrying
func IsRetryable(err error) bool {
	var ociErr *OCIError
	if errors.As(err, &ociErr) {
		switch ociErr.Category {
		case ErrorCategoryNetwork:
			return true
		case ErrorCategoryRegistry:
			// Some HTTP errors are retryable
			return ociErr.HTTPStatus >= 500 || ociErr.HTTPStatus == 429 // Server errors or rate limiting
		case ErrorCategoryAuth:
			return false // Auth errors typically require intervention
		case ErrorCategoryValidation:
			return false // Validation errors need fixes
		default:
			return false
		}
	}
	
	// Check for common retryable errors
	switch {
	case errors.Is(err, ErrTimeout):
		return true
	case errors.Is(err, ErrNetworkError):
		return true
	case errors.Is(err, context.DeadlineExceeded):
		return true
	default:
		return false
	}
}

// IsTemporary determines if an error is temporary
func IsTemporary(err error) bool {
	var ociErr *OCIError
	if errors.As(err, &ociErr) {
		switch ociErr.Category {
		case ErrorCategoryNetwork:
			return true
		case ErrorCategoryRegistry:
			return ociErr.HTTPStatus >= 500 || ociErr.HTTPStatus == 429
		default:
			return false
		}
	}
	
	return IsRetryable(err)
}

// ExtractHTTPStatus extracts HTTP status code from an error
func ExtractHTTPStatus(err error) int {
	var ociErr *OCIError
	if errors.As(err, &ociErr) {
		return ociErr.HTTPStatus
	}
	
	// Try to extract from error message
	errStr := err.Error()
	if strings.Contains(errStr, "404") {
		return http.StatusNotFound
	}
	if strings.Contains(errStr, "401") {
		return http.StatusUnauthorized
	}
	if strings.Contains(errStr, "403") {
		return http.StatusForbidden
	}
	if strings.Contains(errStr, "415") {
		return http.StatusUnsupportedMediaType
	}
	if strings.Contains(errStr, "500") {
		return http.StatusInternalServerError
	}
	
	return 0
}

// GetErrorCategory determines the category of an error
func GetErrorCategory(err error) ErrorCategory {
	var ociErr *OCIError
	if errors.As(err, &ociErr) {
		return ociErr.Category
	}
	
	// Check error message for common patterns
	errStr := err.Error()
	
	// Classify based on error type or message
	switch {
	case errors.Is(err, ErrUnauthorized), errors.Is(err, ErrForbidden),
		strings.Contains(errStr, "unauthorized"), strings.Contains(errStr, "forbidden"):
		return ErrorCategoryAuth
	case errors.Is(err, ErrTimeout), errors.Is(err, context.DeadlineExceeded),
		strings.Contains(errStr, "timeout"):
		return ErrorCategoryNetwork
	case errors.Is(err, ErrInvalidRef), errors.Is(err, ErrInsecureRef):
		return ErrorCategoryValidation
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrUnsupported),
		strings.Contains(errStr, "not found"), strings.Contains(errStr, "NAME_UNKNOWN"):
		return ErrorCategoryRegistry
	default:
		return ErrorCategoryUnknown
	}
}
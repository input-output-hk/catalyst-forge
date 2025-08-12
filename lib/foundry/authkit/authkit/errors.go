package authkit

import "errors"

// Common authentication errors
var (
	// Authentication errors
	ErrUnauthorized   = errors.New("unauthorized")
	ErrTokenExpired   = errors.New("token expired")
	ErrTokenInvalid   = errors.New("token invalid")
	ErrSessionExpired = errors.New("session expired")
	
	// Authorization errors
	ErrForbidden         = errors.New("forbidden")
	ErrInsufficientRoles = errors.New("insufficient roles")
	ErrStepUpRequired    = errors.New("step-up authentication required")
	
	// CSRF errors
	ErrCSRFInvalid = errors.New("CSRF token invalid")
	ErrCSRFMissing = errors.New("CSRF token missing")
	
	// Rate limiting errors
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	
	// Invite errors
	ErrInviteNotFound = errors.New("invite not found")
	ErrInviteExpired  = errors.New("invite expired")
	ErrInviteLocked   = errors.New("invite locked due to too many attempts")
	ErrInviteRedeemed = errors.New("invite already redeemed")
	
	// WebAuthn errors
	ErrChallengeNotFound = errors.New("challenge not found")
	ErrChallengeExpired  = errors.New("challenge expired")
	ErrCredentialInvalid = errors.New("credential invalid")
	ErrHardwareKeyRequired = errors.New("hardware security key required")
	
	// Recovery errors
	ErrRecoveryCodeInvalid = errors.New("recovery code invalid")
	ErrRecoveryCodeUsed    = errors.New("recovery code already used")
	
	// User errors
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
	
	// Refresh token errors
	ErrRefreshTokenInvalid = errors.New("refresh token invalid")
	ErrRefreshTokenExpired = errors.New("refresh token expired")
	ErrRefreshTokenRevoked = errors.New("refresh token revoked")
	ErrRefreshTokenReplay  = errors.New("refresh token replay detected")
)

// ErrorResponse represents a standardized error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
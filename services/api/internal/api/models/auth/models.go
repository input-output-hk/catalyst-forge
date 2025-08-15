package auth

import "time"

// PublicKeyOptionsResponse contains the WebAuthn publicKey options and a
// short-lived session key used to correlate begin/complete requests.
type PublicKeyOptionsResponse struct {
	PublicKey  interface{} `json:"publicKey"`
	SessionKey string      `json:"session_key"`
}

// LoginCompleteResponse is returned after a successful login, providing a
// compact user summary and the issued access token.
type LoginCompleteResponse struct {
	User        UserSummary `json:"user"`
	AccessToken string      `json:"access_token"`
}

// UserSummary is a minimal representation of a user included in auth flows.
type UserSummary struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

// AccessTokenResponse wraps an access token for endpoints that only issue a
// token without additional payload.
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// CredentialsListResponse lists the registered WebAuthn credentials for the
// authenticated user.
type CredentialsListResponse struct {
	Credentials []CredentialSummary `json:"credentials"`
}

// CredentialSummary describes a single registered credential.
type CredentialSummary struct {
	ID         string    `json:"id"`
	AAGUID     string    `json:"aaguid"`
	DeviceName string    `json:"device_name"`
	SignCount  uint32    `json:"sign_count"`
	LastUsedAt time.Time `json:"last_used_at"`
}

// MeResponse returns the authenticated user's identity and roles.
type MeResponse struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

// SessionResponse reports the current session state and step-up status.
type SessionResponse struct {
	Valid          bool      `json:"valid"`
	SessionVersion int64     `json:"session_version"`
	StepUpRequired bool      `json:"step_up_required"`
	StepUpUntil    time.Time `json:"step_up_until"`
}

// RecoveryInitResponse returns the identifier for a passwordless recovery flow.
type RecoveryInitResponse struct {
	FlowID string `json:"flow_id"`
}

// RecoveryVerifyResponse returns the verified flow and resolved user.
type RecoveryVerifyResponse struct {
	FlowID string `json:"flow_id"`
	UserID string `json:"user_id"`
}

// OnboardBeginResponse contains the WebAuthn options and session key for the
// onboarding flow, plus the user ID created/claimed during onboarding.
type OnboardBeginResponse struct {
	PublicKey  interface{} `json:"publicKey"`
	SessionKey string      `json:"session_key"`
	UserID     string      `json:"user_id"`
}

// LoginCompleteRequest is the payload for completing login.
type LoginCompleteRequest struct {
	SessionKey string      `json:"session_key"`
	Credential interface{} `json:"credential"`
}

// CredentialsAddBeginRequest starts a credential registration flow.
type CredentialsAddBeginRequest struct {
	DeviceName string `json:"device_name"`
}

// CredentialsAddCompleteRequest completes credential registration.
type CredentialsAddCompleteRequest struct {
	SessionKey string      `json:"session_key"`
	Credential interface{} `json:"credential"`
}

// StepUpCompleteRequest completes the step-up challenge.
type StepUpCompleteRequest struct {
	SessionKey string      `json:"session_key"`
	Credential interface{} `json:"credential"`
}

// OnboardBeginRequest starts onboarding using an invite.
type OnboardBeginRequest struct {
	InviteID   string `json:"invite_id"`
	Token      string `json:"token"`
	DeviceName string `json:"device_name"`
}

// OnboardCompleteRequest completes onboarding.
type OnboardCompleteRequest struct {
	SessionKey string      `json:"session_key"`
	Credential interface{} `json:"credential"`
	InviteID   string      `json:"invite_id"`
}

// RecoveryInitRequest begins a recovery flow.
type RecoveryInitRequest struct {
	Email string `json:"email"`
}

// RecoveryVerifyRequest verifies a recovery code.
type RecoveryVerifyRequest struct {
	FlowID string `json:"flow_id"`
	Code   string `json:"code"`
}

// RecoveryRegisterBeginRequest begins registering a new credential during recovery.
type RecoveryRegisterBeginRequest struct {
	FlowID     string `json:"flow_id"`
	DeviceName string `json:"device_name"`
}

// RecoveryRegisterCompleteRequest completes credential registration during recovery.
type RecoveryRegisterCompleteRequest struct {
	FlowID     string      `json:"flow_id"`
	SessionKey string      `json:"session_key"`
	Credential interface{} `json:"credential"`
}

// BootstrapRequest is the payload for the one-time admin bootstrap.
type BootstrapRequest struct {
	Email          string `json:"email"`
	BootstrapToken string `json:"bootstrap_token"`
}

// BootstrapResponse is returned after bootstrap succeeds.
type BootstrapResponse struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	TokenHash string `json:"token_hash"`
}

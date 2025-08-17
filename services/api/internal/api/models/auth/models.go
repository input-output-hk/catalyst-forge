package auth

import (
	"encoding/json"
	"time"
)

// PublicKeyOptionsResponse contains the WebAuthn publicKey options and a
// short-lived session key used to correlate begin/complete requests.
type PublicKeyOptionsResponse struct {
	PublicKey  json.RawMessage `json:"publicKey" swaggertype:"object"`
	SessionKey string          `json:"session_key"`
}

// LoginCompleteResponse is returned after a successful login, providing a
// compact user summary and the issued access token.
type LoginCompleteResponse struct {
	User        UserSummary `json:"user"`
	AccessToken string      `json:"access_token"`
}

// UserSummary is a minimal representation of a user included in auth flows.
type UserSummary struct {
	ID       string   `json:"id"`
	Email    string   `json:"email"`
	FullName string   `json:"full_name,omitempty"`
	Roles    []string `json:"roles"`
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
	ID       string   `json:"id"`
	Email    string   `json:"email"`
	FullName string   `json:"full_name,omitempty"`
	Roles    []string `json:"roles"`
}

// SessionResponse reports the current session state and step-up status.
type SessionResponse struct {
	Valid          bool      `json:"valid"`
	SessionVersion int64     `json:"session_version"`
	StepUpRequired bool      `json:"step_up_required"`
	StepUpUntil    time.Time `json:"step_up_until"`
}

// SessionsListResponse lists active sessions for the authenticated user.
type SessionsListResponse struct {
	Sessions []SessionSummary `json:"sessions"`
}

// SessionSummary summarizes a logical session (refresh token family).
type SessionSummary struct {
	ID             string    `json:"id"` // family_id
	Current        bool      `json:"current"`
	AMR            string    `json:"amr"` // webauthn | device_link
	DeviceID       *string   `json:"device_id,omitempty"`
	DeviceName     *string   `json:"device_name,omitempty"`
	UserAgent      string    `json:"user_agent,omitempty"`
	IPAddress      string    `json:"ip_address,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// RecoveryInitResponse returns the identifier for a passwordless recovery flow.
type RecoveryInitResponse struct {
	FlowID string `json:"flow_id"`
}

// RecoveryGenerateResponse returns the one-time-viewable recovery codes.
type RecoveryGenerateResponse struct {
	Codes []string `json:"codes"`
}

// RecoveryVerifyResponse returns the verified flow and resolved user.
type RecoveryVerifyResponse struct {
	FlowID string `json:"flow_id"`
	UserID string `json:"user_id"`
}

// OnboardBeginResponse contains the WebAuthn options and session key for the
// onboarding flow, plus the user ID created/claimed during onboarding.
type OnboardBeginResponse struct {
	PublicKey  json.RawMessage `json:"publicKey" swaggertype:"object"`
	SessionKey string          `json:"session_key"`
	UserID     string          `json:"user_id"`
}

// LoginCompleteRequest is the payload for completing login.
type LoginCompleteRequest struct {
	SessionKey string          `json:"session_key"`
	Credential json.RawMessage `json:"credential" swaggertype:"object"`
}

// CredentialsAddBeginRequest starts a credential registration flow.
type CredentialsAddBeginRequest struct {
	DeviceName string `json:"device_name"`
}

// CredentialsAddCompleteRequest completes credential registration.
type CredentialsAddCompleteRequest struct {
	SessionKey string          `json:"session_key"`
	Credential json.RawMessage `json:"credential" swaggertype:"object"`
}

// StepUpCompleteRequest completes the step-up challenge.
type StepUpCompleteRequest struct {
	SessionKey string          `json:"session_key"`
	Credential json.RawMessage `json:"credential" swaggertype:"object"`
}

// OnboardBeginRequest starts onboarding using an invite.
type OnboardBeginRequest struct {
	InviteID   string `json:"invite_id"`
	Token      string `json:"token"`
	DeviceName string `json:"device_name"`
}

// OnboardCompleteRequest completes onboarding.
type OnboardCompleteRequest struct {
	SessionKey string          `json:"session_key"`
	Credential json.RawMessage `json:"credential" swaggertype:"object"`
	InviteID   string          `json:"invite_id"`
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
	FlowID     string          `json:"flow_id"`
	SessionKey string          `json:"session_key"`
	Credential json.RawMessage `json:"credential" swaggertype:"object"`
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

// Admin users listing
type AdminUsersListResponse struct {
	Users []AdminUser `json:"users"`
}

type AdminUser struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	Roles          []string   `json:"roles"`
	ActiveSessions int        `json:"active_sessions"`
	LastActivityAt *time.Time `json:"last_activity_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Audit listing
type AuditListResponse struct {
	Events []AuditEvent `json:"events"`
	Total  int64        `json:"total"`
}

type AuditEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	UserID    *string                `json:"user_id,omitempty"`
	ActorID   *string                `json:"actor_id,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// Admin invite create
type AdminInviteCreateRequest struct {
	Email        string   `json:"email"`
	Roles        []string `json:"roles"`
	DaysToExpire int      `json:"days_to_expire"`
	EmailUser    bool     `json:"email_user"`
}

type AdminInviteCreateResponse struct {
	InviteID   string    `json:"invite_id"`
	InviteLink string    `json:"invite_link"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// InvitePreviewResponse returns non-sensitive info to display invite landing.
type InvitePreviewResponse struct {
	Valid     bool       `json:"valid"`
	Email     *string    `json:"email,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Reason    string     `json:"reason,omitempty"`
}

// AccessRequestCreateRequest is the public payload to submit a request.
type AccessRequestCreateRequest struct {
	Email  string `json:"email"`
	Reason string `json:"reason,omitempty"`
}

// AccessRequest represents a request in admin views.
type AccessRequest struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Reason    string     `json:"reason,omitempty"`
	Status    string     `json:"status"`
	Attempts  int        `json:"attempts"`
	DecidedAt *time.Time `json:"decided_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type AccessRequestListResponse struct {
	Requests []AccessRequest `json:"requests"`
	Total    int64           `json:"total"`
}

type AccessRequestDecideRequest struct {
	Approve bool   `json:"approve"`
	Note    string `json:"note,omitempty"`
}

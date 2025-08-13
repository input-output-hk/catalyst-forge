package domain

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of audit event.
type EventType string

const (
	// User lifecycle events
	EventUserCreated          EventType = "user.created"
	EventUserRolesUpdated     EventType = "user.roles_updated"
	EventUserSessionBumped    EventType = "user.session_bumped"
	EventUserDeleted          EventType = "user.deleted"
	EventUserSuspended        EventType = "user.suspended"
	EventUserReactivated      EventType = "user.reactivated"
	
	// Credential management events
	EventCredentialAdded      EventType = "credential.added"
	EventCredentialUsed       EventType = "credential.used"
	EventCredentialRemoved    EventType = "credential.removed"
	EventCredentialRevoked    EventType = "credential.revoked"
	EventCredentialUpdated    EventType = "credential.updated"
	
	// Invite events
	EventInviteCreated        EventType = "invite.created"
	EventInviteRedeemed       EventType = "invite.redeemed"
	EventInviteFailed         EventType = "invite.failed"
	EventInviteExpired        EventType = "invite.expired"
	EventInviteRevoked        EventType = "invite.revoked"
	EventInviteLocked         EventType = "invite.locked"
	
	// Authentication events
	EventLoginSuccess         EventType = "login.success"
	EventLoginFailed          EventType = "login.failed"
	EventLoginBegin           EventType = "login.begin"
	EventLogout               EventType = "logout"
	EventLogoutAll            EventType = "logout.all"
	EventLogoutForced         EventType = "logout.forced"
	
	// Registration events
	EventRegistrationBegin    EventType = "registration.begin"
	EventRegistrationSuccess  EventType = "registration.success"
	EventRegistrationFailed   EventType = "registration.failed"
	
	// Token events
	EventTokenRefresh         EventType = "token.refreshed"
	EventTokenRefreshFailed   EventType = "token.refresh_failed"
	EventTokenFamilyRevoked   EventType = "token.family_revoked"
	EventTokenReplayDetected  EventType = "token.replay_detected"
	EventTokenExpired         EventType = "token.expired"
	
	// Recovery events
	EventRecoveryInitiated    EventType = "recovery.initiated"
	EventRecoveryCodeUsed     EventType = "recovery.code_used"
	EventRecoveryCodeGenerated EventType = "recovery.codes_generated"
	EventRecoverySuccess      EventType = "recovery.success"
	EventRecoveryFailed       EventType = "recovery.failed"
	EventRecoveryCompleted    EventType = "recovery.completed"
	EventRecoveryExpired      EventType = "recovery.expired"
	
	// Step-up authentication events
	EventStepUpRequested      EventType = "stepup.requested"
	EventStepUpBegin          EventType = "stepup.begin"
	EventStepUpSuccess        EventType = "stepup.success"
	EventStepUpFailed         EventType = "stepup.failed"
	EventStepUpCompleted      EventType = "stepup.completed"
	EventStepUpExpired        EventType = "stepup.expired"
	
	// Security events
	EventRateLimitExceeded    EventType = "ratelimit.exceeded"
	EventAccessDenied         EventType = "access.denied"
	EventCSRFViolation        EventType = "csrf.violation"
	EventSuspiciousActivity   EventType = "security.suspicious"
	EventInvalidJWT           EventType = "jwt.invalid"
	EventJWTExpired           EventType = "jwt.expired"
	
	// WebAuthn specific events
	EventWebAuthnChallengeFailed EventType = "webauthn.challenge_failed"
	EventWebAuthnOriginMismatch  EventType = "webauthn.origin_mismatch"
	EventWebAuthnSignCountAnomaly EventType = "webauthn.signcount_anomaly"
	EventWebAuthnAttestationFailed EventType = "webauthn.attestation_failed"
	
	// Admin events
	EventAdminAction          EventType = "admin.action"
	EventAdminForceLogout     EventType = "admin.force_logout"
	EventAdminUserModified    EventType = "admin.user_modified"
	EventAdminCredentialRevoked EventType = "admin.credential_revoked"
)

// Event represents an audit event.
type Event struct {
	ID        uuid.UUID
	Type      EventType
	UserID    *uuid.UUID            // May be nil for anonymous events
	ActorID   *uuid.UUID            // Who performed the action (may differ from UserID)
	IPAddress string                // Client IP if available
	UserAgent string                // Client user agent
	Metadata  map[string]interface{} // Additional event-specific data
	CreatedAt time.Time
}
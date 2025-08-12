package domain

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of audit event.
type EventType string

const (
	EventUserCreated          EventType = "user.created"
	EventUserRolesUpdated     EventType = "user.roles_updated"
	EventUserSessionBumped    EventType = "user.session_bumped"
	EventCredentialAdded      EventType = "credential.added"
	EventCredentialUsed       EventType = "credential.used"
	EventCredentialRemoved    EventType = "credential.removed"
	EventCredentialRevoked    EventType = "credential.revoked"
	EventInviteCreated        EventType = "invite.created"
	EventInviteRedeemed       EventType = "invite.redeemed"
	EventInviteFailed         EventType = "invite.failed"
	EventLoginSuccess         EventType = "login.success"
	EventLoginFailed          EventType = "login.failed"
	EventLogout               EventType = "logout"
	EventLogoutAll            EventType = "logout.all"
	EventRegistrationSuccess  EventType = "registration.success"
	EventRegistrationFailed   EventType = "registration.failed"
	EventTokenRefresh         EventType = "token.refreshed"
	EventTokenRefreshFailed   EventType = "token.refresh_failed"
	EventTokenFamilyRevoked   EventType = "token.family_revoked"
	EventRecoveryInitiated    EventType = "recovery.initiated"
	EventRecoveryCodeUsed     EventType = "recovery.code_used"
	EventRecoverySuccess      EventType = "recovery.success"
	EventRecoveryFailed       EventType = "recovery.failed"
	EventRecoveryCompleted    EventType = "recovery.completed"
	EventStepUpRequested      EventType = "stepup.requested"
	EventStepUpSuccess        EventType = "stepup.success"
	EventStepUpFailed         EventType = "stepup.failed"
	EventStepUpCompleted      EventType = "stepup.completed"
	EventRateLimitExceeded    EventType = "ratelimit.exceeded"
	EventAccessDenied         EventType = "access.denied"
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
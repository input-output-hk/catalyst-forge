package service

import (
	"context"
	"strings"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
	"github.com/google/uuid"
)

// AuditLogger provides service-level audit logging capabilities.
type AuditLogger struct {
	store store.AuditStore
}

// NewAuditLogger creates a new service audit logger.
func NewAuditLogger(store store.AuditStore) *AuditLogger {
	return &AuditLogger{store: store}
}

// LogEvent records a generic audit event.
func (l *AuditLogger) LogEvent(ctx context.Context, eventType domain.EventType, userID, actorID *uuid.UUID, metadata map[string]interface{}) error {
	event := domain.Event{
		ID:        uuid.New(),
		Type:      eventType,
		UserID:    userID,
		ActorID:   actorID,
		Metadata:  metadata,
		CreatedAt: time.Now().UTC(),
	}
	
	return l.store.Record(ctx, event)
}

// LogUserCreated records a user creation event.
func (l *AuditLogger) LogUserCreated(ctx context.Context, userID uuid.UUID, email string, roles []string) error {
	metadata := map[string]interface{}{
		"email_domain": getDomain(email), // Store domain only, not full email
		"roles":        roles,
	}
	
	return l.LogEvent(ctx, domain.EventUserCreated, &userID, &userID, metadata)
}

// LogLoginAttempt records a login attempt.
func (l *AuditLogger) LogLoginAttempt(ctx context.Context, userID *uuid.UUID, success bool, reason string) error {
	eventType := domain.EventLoginSuccess
	if !success {
		eventType = domain.EventLoginFailed
	}
	
	metadata := map[string]interface{}{
		"success": success,
	}
	
	if !success && reason != "" {
		metadata["reason"] = reason
	}
	
	return l.LogEvent(ctx, eventType, userID, userID, metadata)
}

// LogCredentialUsed records credential usage during authentication.
func (l *AuditLogger) LogCredentialUsed(ctx context.Context, userID uuid.UUID, credentialID []byte, deviceName string) error {
	// Safely handle credential IDs shorter than 8 bytes
	n := 8
	if len(credentialID) < n {
		n = len(credentialID)
	}
	
	metadata := map[string]interface{}{
		"credential_id": toHex(credentialID[:n]), // Store only first n bytes as hex for privacy
		"device_name":   deviceName,
	}
	
	return l.LogEvent(ctx, domain.EventCredentialUsed, &userID, &userID, metadata)
}

// LogTokenRefresh records a refresh token usage.
func (l *AuditLogger) LogTokenRefresh(ctx context.Context, userID uuid.UUID, familyID uuid.UUID, success bool) error {
	eventType := domain.EventTokenRefresh
	if !success {
		eventType = domain.EventTokenRefreshFailed
	}
	
	metadata := map[string]interface{}{
		"family_id": familyID.String(),
		"success":   success,
	}
	
	return l.LogEvent(ctx, eventType, &userID, &userID, metadata)
}

// LogTokenReplay records a detected token replay attack.
func (l *AuditLogger) LogTokenReplay(ctx context.Context, userID uuid.UUID, familyID uuid.UUID) error {
	metadata := map[string]interface{}{
		"family_id": familyID.String(),
		"action":    "family_revoked",
	}
	
	return l.LogEvent(ctx, domain.EventTokenReplayDetected, &userID, &userID, metadata)
}

// LogRecoveryAttempt records a recovery code usage attempt.
func (l *AuditLogger) LogRecoveryAttempt(ctx context.Context, userID uuid.UUID, success bool) error {
	eventType := domain.EventRecoverySuccess
	if !success {
		eventType = domain.EventRecoveryFailed
	}
	
	metadata := map[string]interface{}{
		"success": success,
	}
	
	return l.LogEvent(ctx, eventType, &userID, &userID, metadata)
}

// LogStepUp records a step-up authentication event.
func (l *AuditLogger) LogStepUp(ctx context.Context, userID uuid.UUID, success bool, reason string) error {
	eventType := domain.EventStepUpSuccess
	if !success {
		eventType = domain.EventStepUpFailed
	}
	
	metadata := map[string]interface{}{
		"success": success,
		"reason":  reason,
	}
	
	return l.LogEvent(ctx, eventType, &userID, &userID, metadata)
}

// LogInviteCreated records invite creation.
func (l *AuditLogger) LogInviteCreated(ctx context.Context, actorID uuid.UUID, inviteID uuid.UUID, email string, roles []string) error {
	metadata := map[string]interface{}{
		"invite_id":    inviteID.String(),
		"email_domain": getDomain(email), // Store domain only
		"roles":        roles,
	}
	
	return l.LogEvent(ctx, domain.EventInviteCreated, nil, &actorID, metadata)
}

// LogInviteRedeemed records successful invite redemption.
func (l *AuditLogger) LogInviteRedeemed(ctx context.Context, userID uuid.UUID, inviteID uuid.UUID) error {
	metadata := map[string]interface{}{
		"invite_id": inviteID.String(),
	}
	
	return l.LogEvent(ctx, domain.EventInviteRedeemed, &userID, &userID, metadata)
}

// LogInviteFailed records failed invite redemption attempt.
func (l *AuditLogger) LogInviteFailed(ctx context.Context, inviteID uuid.UUID, reason string, attempts int) error {
	metadata := map[string]interface{}{
		"invite_id": inviteID.String(),
		"reason":    reason,
		"attempts":  attempts,
	}
	
	// Check if invite should be locked
	eventType := domain.EventInviteFailed
	if attempts >= 5 {
		eventType = domain.EventInviteLocked
		metadata["locked"] = true
	}
	
	return l.LogEvent(ctx, eventType, nil, nil, metadata)
}

// LogSessionInvalidated records session invalidation.
func (l *AuditLogger) LogSessionInvalidated(ctx context.Context, userID uuid.UUID, reason string, actorID *uuid.UUID) error {
	metadata := map[string]interface{}{
		"reason": reason,
	}
	
	eventType := domain.EventLogoutAll
	if reason == "forced" && actorID != nil && *actorID != userID {
		eventType = domain.EventLogoutForced
	}
	
	return l.LogEvent(ctx, eventType, &userID, actorID, metadata)
}

// LogWebAuthnAnomaly records WebAuthn security anomalies.
func (l *AuditLogger) LogWebAuthnAnomaly(ctx context.Context, userID uuid.UUID, anomalyType string, details map[string]interface{}) error {
	var eventType domain.EventType
	switch anomalyType {
	case "signcount":
		eventType = domain.EventWebAuthnSignCountAnomaly
	case "origin":
		eventType = domain.EventWebAuthnOriginMismatch
	case "challenge":
		eventType = domain.EventWebAuthnChallengeFailed
	case "attestation":
		eventType = domain.EventWebAuthnAttestationFailed
	default:
		eventType = domain.EventSuspiciousActivity
	}
	
	metadata := map[string]interface{}{
		"anomaly_type": anomalyType,
	}
	
	// Add details but sanitize any sensitive data
	for k, v := range details {
		if !isSensitiveField(k) {
			metadata[k] = v
		}
	}
	
	return l.LogEvent(ctx, eventType, &userID, &userID, metadata)
}

// LogAdminAction records administrative actions.
func (l *AuditLogger) LogAdminAction(ctx context.Context, actorID uuid.UUID, action string, targetUserID *uuid.UUID, details map[string]interface{}) error {
	metadata := map[string]interface{}{
		"action": action,
	}
	
	if targetUserID != nil {
		metadata["target_user_id"] = targetUserID.String()
	}
	
	// Add sanitized details
	for k, v := range details {
		if !isSensitiveField(k) {
			metadata[k] = v
		}
	}
	
	var eventType domain.EventType
	switch action {
	case "force_logout":
		eventType = domain.EventAdminForceLogout
	case "modify_user":
		eventType = domain.EventAdminUserModified
	case "revoke_credential":
		eventType = domain.EventAdminCredentialRevoked
	default:
		eventType = domain.EventAdminAction
	}
	
	return l.LogEvent(ctx, eventType, targetUserID, &actorID, metadata)
}

// Helper functions

// getDomain extracts and normalizes the domain from an email address.
func getDomain(email string) string {
	// Trim whitespace and find the @ symbol
	email = strings.TrimSpace(email)
	at := strings.LastIndexByte(email, '@')
	
	// Extract and normalize the domain
	if at >= 0 && at+1 < len(email) {
		domain := email[at+1:]
		// Domains are case-insensitive; normalize to lowercase and trim
		return strings.ToLower(strings.TrimSpace(domain))
	}
	return "unknown"
}

// toHex converts bytes to hexadecimal string.
func toHex(data []byte) string {
	const hexChars = "0123456789abcdef"
	result := make([]byte, len(data)*2)
	for i, b := range data {
		result[i*2] = hexChars[b>>4]
		result[i*2+1] = hexChars[b&0x0f]
	}
	return string(result)
}

// isSensitiveField checks if a field name contains sensitive data.
func isSensitiveField(field string) bool {
	sensitiveFields := []string{
		"password", "token", "secret", "key", "credential",
		"signature", "private", "clientdatajson", "authenticatordata",
		"publickey", "code", "hash",
	}
	
	fieldLower := strings.ToLower(field)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(fieldLower, sensitive) {
			return true
		}
	}
	return false
}
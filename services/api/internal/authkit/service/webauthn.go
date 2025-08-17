package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"bytes"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

// WebAuthnService handles WebAuthn ceremonies for registration and authentication.
type WebAuthnService interface {
	// BeginRegistration starts a WebAuthn registration ceremony.
	BeginRegistration(ctx context.Context, user *domain.User, deviceName string, requireHardwareKey bool) (creationOptions json.RawMessage, sessionKey string, err error)

	// FinishRegistration completes a WebAuthn registration ceremony.
	FinishRegistration(ctx context.Context, sessionKey string, clientResponse json.RawMessage) (*domain.Credential, error)

	// BeginLogin starts a WebAuthn authentication ceremony (username-less).
	BeginLogin(ctx context.Context, userHint string) (assertionOptions json.RawMessage, sessionKey string, err error)

	// FinishLogin completes a WebAuthn authentication ceremony.
	FinishLogin(ctx context.Context, sessionKey string, clientResponse json.RawMessage) (*domain.User, *domain.Credential, error)

	// BeginStepUp starts a step-up authentication ceremony for an authenticated user.
	BeginStepUp(ctx context.Context, user *domain.User, action string) (assertionOptions json.RawMessage, sessionKey string, err error)

	// FinishStepUp completes a step-up authentication ceremony.
	FinishStepUp(ctx context.Context, sessionKey string, clientResponse json.RawMessage) (*domain.User, error)
}

// webAuthnService implements WebAuthnService.
type webAuthnService struct {
	webauthn     *webauthn.WebAuthn
	users        store.UserStore
	credentials  store.CredentialStore
	challenges   store.ChallengeStore
	rand         crypto.Rand
	adminAAGUIDs map[string]bool
	challengeTTL time.Duration
}

// WebAuthnConfig holds configuration for the WebAuthn service.
type WebAuthnConfig struct {
	RPDisplayName        string
	RPID                 string
	RPOrigins            []string
	Users                store.UserStore
	Credentials          store.CredentialStore
	Challenges           store.ChallengeStore
	Rand                 crypto.Rand
	AdminAAGUIDAllowlist []string
	ChallengeTTL         time.Duration
}

// NewWebAuthnService creates a new WebAuthn service.
func NewWebAuthnService(cfg WebAuthnConfig) (WebAuthnService, error) {
	// Create WebAuthn config
	config := &webauthn.Config{
		RPDisplayName: cfg.RPDisplayName,
		RPID:          cfg.RPID,
		RPOrigins:     cfg.RPOrigins,
		// Always require user verification (biometric/PIN)
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationRequired,
		},
		// Default to no attestation for privacy (override for admins)
		AttestationPreference: protocol.PreferNoAttestation,
	}

	w, err := webauthn.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create webauthn: %w", err)
	}

	// Build admin AAGUID map for fast lookup (normalized to canonical form)
	adminAAGUIDs := make(map[string]bool, len(cfg.AdminAAGUIDAllowlist))
	for _, a := range cfg.AdminAAGUIDAllowlist {
		if u, err := uuid.Parse(a); err == nil {
			adminAAGUIDs[u.String()] = true // canonical lowercase
		}
	}

	return &webAuthnService{
		webauthn:     w,
		users:        cfg.Users,
		credentials:  cfg.Credentials,
		challenges:   cfg.Challenges,
		rand:         cfg.Rand,
		adminAAGUIDs: adminAAGUIDs,
		challengeTTL: cfg.ChallengeTTL,
	}, nil
}

// webAuthnUser adapts domain.User to webauthn.User interface.
type webAuthnUser struct {
	user        *domain.User
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte {
	id, _ := u.user.ID.MarshalBinary()
	return id
}

func (u *webAuthnUser) WebAuthnName() string {
	return u.user.Email
}

func (u *webAuthnUser) WebAuthnDisplayName() string {
	return u.user.Email
}

func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

func (u *webAuthnUser) WebAuthnIcon() string {
	return ""
}

// BeginRegistration starts a WebAuthn registration ceremony.
func (s *webAuthnService) BeginRegistration(ctx context.Context, user *domain.User, deviceName string, requireHardwareKey bool) (creationOptions json.RawMessage, sessionKey string, err error) {
	// Load existing credentials for exclusion list
	creds, err := s.credentials.GetByUser(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}

	// Convert to WebAuthn credentials
	webauthnCreds := make([]webauthn.Credential, len(creds))
	for i, cred := range creds {
		webauthnCreds[i] = webauthn.Credential{
			ID:              cred.ID[:],
			PublicKey:       cred.PublicKey,
			AttestationType: "none",
		}
	}

	// Create WebAuthn user
	webauthnUser := &webAuthnUser{
		user:        user,
		credentials: webauthnCreds,
	}

	// Generate registration options
	opts := []webauthn.RegistrationOption{
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			UserVerification:   protocol.VerificationRequired,
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			RequireResidentKey: boolPtr(true),
		}),
		webauthn.WithExtensions(protocol.AuthenticationExtensions{
			"credProps": true,
		}),
	}

	// Keep default attestation preference (PreferNoAttestation); rely on AAGUID checks instead

	options, session, err := s.webauthn.BeginRegistration(webauthnUser, opts...)
	if err != nil {
		return nil, "", err
	}

	// Marshal options to raw JSON for pass-through
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, "", err
	}

	// Generate session key
	sessionKeyBytes, err := s.rand.Bytes(32)
	if err != nil {
		return nil, "", err
	}
	sessionKey = base64.RawURLEncoding.EncodeToString(sessionKeyBytes)

	// Store session data in challenge store
	sessionData := map[string]interface{}{
		"type":                 "registration",
		"session":              session,
		"user_id":              user.ID.String(),
		"device_name":          deviceName,
		"require_hardware_key": requireHardwareKey,
	}
	sessionJSON, _ := json.Marshal(sessionData)

	if err := s.challenges.Set(ctx, sessionKey, sessionJSON, s.challengeTTL); err != nil {
		return nil, "", err
	}

	return json.RawMessage(optionsJSON), sessionKey, nil
}

// FinishRegistration completes a WebAuthn registration ceremony.
func (s *webAuthnService) FinishRegistration(ctx context.Context, sessionKey string, clientResponse json.RawMessage) (*domain.Credential, error) {
	// Retrieve session data
	sessionJSON, err := s.challenges.Get(ctx, sessionKey)
	if err != nil {
		return nil, errors.New("invalid or expired session")
	}
	// Ensure single-use: delete challenge on any terminal outcome
	defer func() { _ = s.challenges.Delete(ctx, sessionKey) }()

	var sessionData struct {
		Type               string                `json:"type"`
		Session            *webauthn.SessionData `json:"session"`
		UserID             string                `json:"user_id"`
		DeviceName         string                `json:"device_name"`
		RequireHardwareKey bool                  `json:"require_hardware_key"`
	}
	if err := json.Unmarshal(sessionJSON, &sessionData); err != nil {
		return nil, errors.New("invalid session data")
	}

	if sessionData.Type != "registration" {
		return nil, errors.New("invalid session type")
	}

	// Parse user ID
	userID, err := uuid.Parse(sessionData.UserID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Load user
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create WebAuthn user
	webauthnUser := &webAuthnUser{
		user:        user,
		credentials: []webauthn.Credential{},
	}

	// Parse the client response (base64url-aware)
	parsedCreation, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(clientResponse))
	if err != nil {
		return nil, fmt.Errorf("failed to parse credential response: %w", err)
	}

	// Verify the registration
	credential, err := s.webauthn.CreateCredential(webauthnUser, *sessionData.Session, parsedCreation)
	if err != nil {
		return nil, fmt.Errorf("registration verification failed: %w", err)
	}

	// Extract AAGUID from attestation object
	aaguid := extractAAGUID(credential.Authenticator.AAGUID)

	// If hardware key is required (admin), verify AAGUID only when an allowlist is configured.
	if sessionData.RequireHardwareKey {
		if len(s.adminAAGUIDs) > 0 && !s.adminAAGUIDs[aaguid] {
			return nil, errors.New("hardware security key not allowed for admin role")
		}
	}

	// Create domain credential
	domainCred := &domain.Credential{
		ID:         credential.ID,
		UserID:     user.ID,
		PublicKey:  credential.PublicKey,
		AAGUID:     aaguid,
		DeviceName: sessionData.DeviceName,
		SignCount:  credential.Authenticator.SignCount,
		RK:         true, // We always require resident keys
		CreatedAt:  time.Now().UTC(),
	}

	// Store credential
	if err := s.credentials.Add(ctx, domainCred); err != nil {
		return nil, err
	}

	return domainCred, nil
}

// BeginLogin starts a WebAuthn authentication ceremony (username-less).
func (s *webAuthnService) BeginLogin(ctx context.Context, userHint string) (assertionOptions json.RawMessage, sessionKey string, err error) {
	// For username-less login, we don't specify allowed credentials
	options, session, err := s.webauthn.BeginDiscoverableLogin(
		webauthn.WithUserVerification(protocol.VerificationRequired),
	)
	if err != nil {
		return nil, "", err
	}

	// Marshal options to raw JSON for pass-through
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, "", err
	}

	// Generate session key
	sessionKeyBytes, err := s.rand.Bytes(32)
	if err != nil {
		return nil, "", err
	}
	sessionKey = base64.RawURLEncoding.EncodeToString(sessionKeyBytes)

	// Store session data
	sessionData := map[string]interface{}{
		"type":      "login",
		"session":   session,
		"user_hint": userHint,
	}
	sessionJSON, _ := json.Marshal(sessionData)

	if err := s.challenges.Set(ctx, sessionKey, sessionJSON, s.challengeTTL); err != nil {
		return nil, "", err
	}

	return json.RawMessage(optionsJSON), sessionKey, nil
}

// FinishLogin completes a WebAuthn authentication ceremony.
func (s *webAuthnService) FinishLogin(ctx context.Context, sessionKey string, clientResponse json.RawMessage) (*domain.User, *domain.Credential, error) {
	// Retrieve session data
	sessionJSON, err := s.challenges.Get(ctx, sessionKey)
	if err != nil {
		return nil, nil, errors.New("invalid or expired session")
	}
	// Ensure single-use: delete challenge on any terminal outcome
	defer func() { _ = s.challenges.Delete(ctx, sessionKey) }()

	var sessionData struct {
		Type     string                `json:"type"`
		Session  *webauthn.SessionData `json:"session"`
		UserHint string                `json:"user_hint"`
	}
	if err := json.Unmarshal(sessionJSON, &sessionData); err != nil {
		return nil, nil, errors.New("invalid session data")
	}

	if sessionData.Type != "login" {
		return nil, nil, errors.New("invalid session type")
	}

	// Parse the client response (base64url-aware)
	parsedAssertion, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(clientResponse))
	if err != nil {
		slog.Default().Warn("webauthn: parse assertion failed", "err", err)
		return nil, nil, fmt.Errorf("failed to parse assertion response: %w", err)
	}

	// Resolve user ID, accommodating Safari which may omit or alter userHandle
	slog.Default().Debug("webauthn: assertion meta", "rawID_len", len(parsedAssertion.RawID), "userHandle_len", len(parsedAssertion.Response.UserHandle))
	var userID uuid.UUID
	if len(parsedAssertion.Response.UserHandle) == 16 {
		if uid, uerr := uuid.FromBytes(parsedAssertion.Response.UserHandle); uerr == nil {
			userID = uid
		} else {
			slog.Default().Warn("webauthn: userHandle parse failed, falling back to credential lookup", "err", uerr)
			cred, gerr := s.credentials.Get(ctx, parsedAssertion.RawID)
			if gerr != nil {
				slog.Default().Warn("webauthn: credential lookup failed", "err", gerr)
				return nil, nil, errors.New("authentication failed")
			}
			userID = cred.UserID
		}
	} else {
		slog.Default().Debug("webauthn: missing/non-uuid userHandle; using credential lookup")
		cred, gerr := s.credentials.Get(ctx, parsedAssertion.RawID)
		if gerr != nil {
			slog.Default().Warn("webauthn: credential lookup failed", "err", gerr)
			return nil, nil, errors.New("authentication failed")
		}
		userID = cred.UserID
	}

	// Load user and credentials
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, errors.New("authentication failed")
	}

	creds, err := s.credentials.GetByUser(ctx, user.ID)
	if err != nil {
		return nil, nil, errors.New("authentication failed")
	}

	// Convert to WebAuthn credentials
	webauthnCreds := make([]webauthn.Credential, len(creds))
	credMap := make(map[string]*domain.Credential)
	for i, cred := range creds {
		webauthnCreds[i] = webauthn.Credential{
			ID:              cred.ID[:],
			PublicKey:       cred.PublicKey,
			AttestationType: "none",
			Authenticator: webauthn.Authenticator{
				SignCount: cred.SignCount,
			},
		}
		credMap[string(cred.ID[:])] = &creds[i]
	}

	// Create WebAuthn user
	webauthnUser := &webAuthnUser{
		user:        user,
		credentials: webauthnCreds,
	}

	// Verify the assertion
	credential, err := s.webauthn.ValidateDiscoverableLogin(
		func(rawID, userHandle []byte) (webauthn.User, error) {
			return webauthnUser, nil
		},
		*sessionData.Session,
		parsedAssertion,
	)
	if err != nil {
		// Safari/iCloud-synced passkeys can trigger a backup-eligible flag mismatch even when
		// the assertion is otherwise valid. Treat this specific case as acceptable to avoid
		// false negatives during login.
		if strings.Contains(err.Error(), "Backup Eligible flag inconsistency") {
			slog.Default().Warn("webauthn: ignoring backup-eligible mismatch during discoverable login", "err", err)
			// Map the credential by raw ID and proceed
			domainCred := credMap[string(parsedAssertion.RawID)]
			if domainCred == nil {
				if c, gerr := s.credentials.Get(ctx, parsedAssertion.RawID); gerr == nil && c != nil {
					// ensure map hit for downstream
					domainCred = c
				} else {
					slog.Default().Warn("webauthn: could not resolve credential after ignoring backup flag", "lookup_err", gerr)
					return nil, nil, fmt.Errorf("authentication verification failed: %w", err)
				}
			}
			// Persist last-used timestamp without changing sign count
			if uerr := s.credentials.UpdateOnAssertion(ctx, domainCred.ID, domainCred.SignCount, time.Now().UTC()); uerr != nil {
				slog.Default().Warn("webauthn: failed to persist assertion update (ignored)", "err", uerr)
			}
			return user, domainCred, nil
		}

		slog.Default().Warn("webauthn: validate discoverable login failed", "err", err)
		return nil, nil, fmt.Errorf("authentication verification failed: %w", err)
	}

	// Find the domain credential used
	domainCred := credMap[string(credential.ID)]
	if domainCred == nil {
		return nil, nil, errors.New("credential not found")
	}

	// Enforce hardware key requirement for admin users only when an allowlist is configured.
	if isAdmin(user) && len(s.adminAAGUIDs) > 0 && !s.adminAAGUIDs[domainCred.AAGUID] {
		return nil, nil, errors.New("admin must authenticate with an allowed hardware security key")
	}

	// Update sign count and last-used time.
	// Equal counters are acceptable across authenticators; only strictly lower is suspicious.
	if credential.Authenticator.SignCount > 0 && credential.Authenticator.SignCount < domainCred.SignCount {
		return nil, nil, errors.New("potential credential clone detected")
	}

	if credential.Authenticator.SignCount > domainCred.SignCount {
		domainCred.SignCount = credential.Authenticator.SignCount
	}
	// Always persist last-used to keep admin UI accurate even when counter is unchanged
	if err := s.credentials.UpdateOnAssertion(ctx, domainCred.ID, domainCred.SignCount, time.Now().UTC()); err != nil {
		slog.Default().Warn("webauthn: failed to persist assertion update", "err", err)
	}

	return user, domainCred, nil
}

// BeginStepUp starts a step-up authentication ceremony for an authenticated user.
func (s *webAuthnService) BeginStepUp(ctx context.Context, user *domain.User, action string) (assertionOptions json.RawMessage, sessionKey string, err error) {
	// Load user credentials
	creds, err := s.credentials.GetByUser(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}

	// Convert to WebAuthn credentials
	webauthnCreds := make([]webauthn.Credential, len(creds))
	for i, cred := range creds {
		webauthnCreds[i] = webauthn.Credential{
			ID:              cred.ID[:],
			PublicKey:       cred.PublicKey,
			AttestationType: "none",
		}
	}

	// Create WebAuthn user
	webauthnUser := &webAuthnUser{
		user:        user,
		credentials: webauthnCreds,
	}

	// Generate authentication options
	options, session, err := s.webauthn.BeginLogin(webauthnUser,
		webauthn.WithUserVerification(protocol.VerificationRequired),
	)
	if err != nil {
		return nil, "", err
	}

	// Marshal options to raw JSON for pass-through
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, "", err
	}

	// Generate session key
	sessionKeyBytes, err := s.rand.Bytes(32)
	if err != nil {
		return nil, "", err
	}
	sessionKey = base64.RawURLEncoding.EncodeToString(sessionKeyBytes)

	// Store session data
	sessionData := map[string]interface{}{
		"type":    "step_up",
		"session": session,
		"user_id": user.ID.String(),
		"action":  action,
	}
	sessionJSON, _ := json.Marshal(sessionData)

	if err := s.challenges.Set(ctx, sessionKey, sessionJSON, s.challengeTTL); err != nil {
		return nil, "", err
	}

	return json.RawMessage(optionsJSON), sessionKey, nil
}

// FinishStepUp completes a step-up authentication ceremony.
func (s *webAuthnService) FinishStepUp(ctx context.Context, sessionKey string, clientResponse json.RawMessage) (*domain.User, error) {
	// Retrieve session data
	sessionJSON, err := s.challenges.Get(ctx, sessionKey)
	if err != nil {
		return nil, errors.New("invalid or expired session")
	}
	// Ensure single-use: delete challenge on any terminal outcome
	defer func() { _ = s.challenges.Delete(ctx, sessionKey) }()

	var sessionData struct {
		Type    string                `json:"type"`
		Session *webauthn.SessionData `json:"session"`
		UserID  string                `json:"user_id"`
		Action  string                `json:"action"`
	}
	if err := json.Unmarshal(sessionJSON, &sessionData); err != nil {
		return nil, errors.New("invalid session data")
	}

	if sessionData.Type != "step_up" {
		return nil, errors.New("invalid session type")
	}

	// Parse user ID
	userID, err := uuid.Parse(sessionData.UserID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Load user and credentials
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	creds, err := s.credentials.GetByUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// Convert to WebAuthn credentials
	webauthnCreds := make([]webauthn.Credential, len(creds))
	credMap := make(map[string]*domain.Credential)
	for i, cred := range creds {
		webauthnCreds[i] = webauthn.Credential{
			ID:              cred.ID[:],
			PublicKey:       cred.PublicKey,
			AttestationType: "none",
			Authenticator: webauthn.Authenticator{
				SignCount: cred.SignCount,
			},
		}
		credMap[string(cred.ID[:])] = &creds[i]
	}

	// Create WebAuthn user
	webauthnUser := &webAuthnUser{
		user:        user,
		credentials: webauthnCreds,
	}

	// Parse the client response (base64url-aware)
	parsedAssertion, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(clientResponse))
	if err != nil {
		return nil, fmt.Errorf("failed to parse assertion response: %w", err)
	}

	// Verify the assertion
	credential, err := s.webauthn.ValidateLogin(webauthnUser, *sessionData.Session, parsedAssertion)
	if err != nil {
		return nil, fmt.Errorf("step-up verification failed: %w", err)
	}

	// Find the domain credential used
	domainCred := credMap[string(credential.ID)]
	if domainCred == nil {
		return nil, errors.New("credential not found")
	}

	// Enforce hardware key requirement for admin users only when an allowlist is configured.
	if isAdmin(user) && len(s.adminAAGUIDs) > 0 && !s.adminAAGUIDs[domainCred.AAGUID] {
		return nil, errors.New("admin must authenticate with an allowed hardware security key")
	}

	// Update sign count
	if credential.Authenticator.SignCount > domainCred.SignCount {
		domainCred.SignCount = credential.Authenticator.SignCount
		_ = s.credentials.UpdateOnAssertion(ctx, domainCred.ID, domainCred.SignCount, time.Now().UTC())
	}

	return user, nil
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}

// extractAAGUID converts the AAGUID bytes to a canonical UUID string.
func extractAAGUID(aaguidBytes []byte) string {
	if len(aaguidBytes) != 16 {
		return ""
	}

	u, err := uuid.FromBytes(aaguidBytes)
	if err != nil {
		return ""
	}
	return u.String() // canonical 8-4-4-4-12, lowercase
}

// isAdmin checks if a user has admin role.
func isAdmin(user *domain.User) bool {
	for _, role := range user.Roles {
		if role == "admin" {
			return true
		}
	}
	return false
}

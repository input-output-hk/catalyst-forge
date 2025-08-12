package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/crypto"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/domain"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/store"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// WebAuthnService handles WebAuthn ceremonies for registration and authentication.
type WebAuthnService interface {
	// BeginRegistration starts a WebAuthn registration ceremony.
	BeginRegistration(ctx context.Context, user *domain.User, deviceName string, requireHardwareKey bool) (creationOptions interface{}, sessionKey string, err error)

	// FinishRegistration completes a WebAuthn registration ceremony.
	FinishRegistration(ctx context.Context, sessionKey string, clientResponse interface{}) (*domain.Credential, error)

	// BeginLogin starts a WebAuthn authentication ceremony (username-less).
	BeginLogin(ctx context.Context, userHint string) (assertionOptions interface{}, sessionKey string, err error)

	// FinishLogin completes a WebAuthn authentication ceremony.
	FinishLogin(ctx context.Context, sessionKey string, clientResponse interface{}) (*domain.User, *domain.Credential, error)

	// BeginStepUp starts a step-up authentication ceremony for an authenticated user.
	BeginStepUp(ctx context.Context, user *domain.User, action string) (assertionOptions interface{}, sessionKey string, err error)

	// FinishStepUp completes a step-up authentication ceremony.
	FinishStepUp(ctx context.Context, sessionKey string, clientResponse interface{}) (*domain.User, error)
}

// webAuthnService implements WebAuthnService.
type webAuthnService struct {
	webauthn       *webauthn.WebAuthn
	users          store.UserStore
	credentials    store.CredentialStore
	challenges     store.ChallengeStore
	rand           crypto.Rand
	adminAAGUIDs   map[string]bool
	challengeTTL   time.Duration
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
func (s *webAuthnService) BeginRegistration(ctx context.Context, user *domain.User, deviceName string, requireHardwareKey bool) (creationOptions interface{}, sessionKey string, err error) {
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
	
	// Use direct attestation for admin/hardware key requirements
	if requireHardwareKey {
		opts = append(opts, webauthn.WithConveyancePreference(protocol.PreferDirectAttestation))
	}
	
	options, session, err := s.webauthn.BeginRegistration(webauthnUser, opts...)
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
		"type":                "registration",
		"session":             session,
		"user_id":             user.ID.String(),
		"device_name":         deviceName,
		"require_hardware_key": requireHardwareKey,
	}
	sessionJSON, _ := json.Marshal(sessionData)
	
	if err := s.challenges.Set(ctx, sessionKey, sessionJSON, s.challengeTTL); err != nil {
		return nil, "", err
	}

	return options, sessionKey, nil
}

// FinishRegistration completes a WebAuthn registration ceremony.
func (s *webAuthnService) FinishRegistration(ctx context.Context, sessionKey string, clientResponse interface{}) (*domain.Credential, error) {
	// Retrieve session data
	sessionJSON, err := s.challenges.Get(ctx, sessionKey)
	if err != nil {
		return nil, errors.New("invalid or expired session")
	}
	// Ensure single-use: delete challenge on any terminal outcome
	defer func() { _ = s.challenges.Delete(ctx, sessionKey) }()

	var sessionData struct {
		Type              string                    `json:"type"`
		Session           *webauthn.SessionData     `json:"session"`
		UserID            string                    `json:"user_id"`
		DeviceName        string                    `json:"device_name"`
		RequireHardwareKey bool                      `json:"require_hardware_key"`
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

	// Parse the client response
	responseBytes, err := json.Marshal(clientResponse)
	if err != nil {
		return nil, errors.New("invalid client response")
	}

	var parsedResponse protocol.ParsedCredentialCreationData
	if err := json.Unmarshal(responseBytes, &parsedResponse); err != nil {
		return nil, errors.New("failed to parse credential response")
	}

	// Verify the registration
	credential, err := s.webauthn.CreateCredential(webauthnUser, *sessionData.Session, &parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("registration verification failed: %w", err)
	}

	// Extract AAGUID from attestation object
	aaguid := extractAAGUID(credential.Authenticator.AAGUID)

	// If hardware key is required (admin), verify AAGUID
	if sessionData.RequireHardwareKey {
		if !s.adminAAGUIDs[aaguid] {
			return nil, errors.New("hardware security key required for admin role")
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
func (s *webAuthnService) BeginLogin(ctx context.Context, userHint string) (assertionOptions interface{}, sessionKey string, err error) {
	// For username-less login, we don't specify allowed credentials
	options, session, err := s.webauthn.BeginDiscoverableLogin(
		webauthn.WithUserVerification(protocol.VerificationRequired),
	)
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

	return options, sessionKey, nil
}

// FinishLogin completes a WebAuthn authentication ceremony.
func (s *webAuthnService) FinishLogin(ctx context.Context, sessionKey string, clientResponse interface{}) (*domain.User, *domain.Credential, error) {
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

	// Parse the client response
	responseBytes, err := json.Marshal(clientResponse)
	if err != nil {
		return nil, nil, errors.New("invalid client response")
	}

	var parsedResponse protocol.ParsedCredentialAssertionData
	if err := json.Unmarshal(responseBytes, &parsedResponse); err != nil {
		return nil, nil, errors.New("failed to parse assertion response")
	}

	// Get user ID from response (userHandle)
	if len(parsedResponse.Response.UserHandle) == 0 {
		// Try to look up by credential ID if userHandle not provided
		cred, err := s.credentials.Get(ctx, parsedResponse.RawID)
		if err != nil {
			return nil, nil, errors.New("authentication failed")
		}
		parsedResponse.Response.UserHandle = cred.UserID[:]
	}

	// Parse user ID from userHandle
	userID, err := uuid.FromBytes(parsedResponse.Response.UserHandle)
	if err != nil {
		return nil, nil, errors.New("invalid user handle")
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
		&parsedResponse,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("authentication verification failed: %w", err)
	}

	// Find the domain credential used
	domainCred := credMap[string(credential.ID)]
	if domainCred == nil {
		return nil, nil, errors.New("credential not found")
	}
	
	// Enforce hardware key requirement for admin users
	if isAdmin(user) && !s.adminAAGUIDs[domainCred.AAGUID] {
		return nil, nil, errors.New("admin must authenticate with a hardware security key")
	}

	// Update sign count (important for clone detection)
	if credential.Authenticator.SignCount > domainCred.SignCount {
		domainCred.SignCount = credential.Authenticator.SignCount
		if err := s.credentials.UpdateOnAssertion(ctx, domainCred.ID, domainCred.SignCount, time.Now().UTC()); err != nil {
			// Log but don't fail authentication
			_ = err
		}
	} else if credential.Authenticator.SignCount > 0 && credential.Authenticator.SignCount <= domainCred.SignCount {
		// Potential credential clone detected
		// In production, you might want to flag this for security review
		return nil, nil, errors.New("potential credential clone detected")
	}

	return user, domainCred, nil
}

// BeginStepUp starts a step-up authentication ceremony for an authenticated user.
func (s *webAuthnService) BeginStepUp(ctx context.Context, user *domain.User, action string) (assertionOptions interface{}, sessionKey string, err error) {
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

	return options, sessionKey, nil
}

// FinishStepUp completes a step-up authentication ceremony.
func (s *webAuthnService) FinishStepUp(ctx context.Context, sessionKey string, clientResponse interface{}) (*domain.User, error) {
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

	// Parse the client response
	responseBytes, err := json.Marshal(clientResponse)
	if err != nil {
		return nil, errors.New("invalid client response")
	}

	var parsedResponse protocol.ParsedCredentialAssertionData
	if err := json.Unmarshal(responseBytes, &parsedResponse); err != nil {
		return nil, errors.New("failed to parse assertion response")
	}

	// Verify the assertion
	credential, err := s.webauthn.ValidateLogin(webauthnUser, *sessionData.Session, &parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("step-up verification failed: %w", err)
	}

	// Find the domain credential used
	domainCred := credMap[string(credential.ID)]
	if domainCred == nil {
		return nil, errors.New("credential not found")
	}
	
	// Enforce hardware key requirement for admin users
	if isAdmin(user) && !s.adminAAGUIDs[domainCred.AAGUID] {
		return nil, errors.New("admin must authenticate with a hardware security key")
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


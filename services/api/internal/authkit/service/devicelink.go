package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

// DeviceLinkService handles device-code linking flows for CLI authentication.
type DeviceLinkService interface {
	// BeginDeviceLink starts a new device linking flow.
	BeginDeviceLink(ctx context.Context, deviceName string, purpose string) (*DeviceLinkResponse, error)
	
	// AuthorizeDeviceLink authorizes a device link after user authentication.
	AuthorizeDeviceLink(ctx context.Context, deviceCode string, userID uuid.UUID) error
	
	// ExchangeDeviceCode exchanges a device code for tokens.
	ExchangeDeviceCode(ctx context.Context, deviceCode string) (*ExchangeResponse, error)
}

// DeviceLinkResponse contains the response for beginning a device link flow.
type DeviceLinkResponse struct {
	DeviceCode             string `json:"device_code"`
	UserCode               string `json:"user_code"`
	VerificationURI        string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn              int    `json:"expires_in"`
	Interval               int    `json:"interval"`
}

// ExchangeResponse contains the result of a device code exchange.
type ExchangeResponse struct {
	Status       string     `json:"status,omitempty"` // "authorization_pending", "slow_down", "expired_token"
	AccessToken  string     `json:"access_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	ExpiresIn    int        `json:"expires_in,omitempty"`
	RefreshExpiresIn int    `json:"refresh_expires_in,omitempty"`
	DeviceID     string     `json:"device_id,omitempty"`
	User         *UserInfo  `json:"user,omitempty"`
}

// UserInfo contains basic user information.
type UserInfo struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

// DeviceLinkConfig holds configuration for the device link service.
type DeviceLinkConfig struct {
	LinkStore     store.DeviceLinkStore
	DeviceStore   store.DeviceStore
	UserStore     store.UserStore
	RefreshStore  store.RefreshStore
	AuditStore    store.AuditStore  // For audit logging
	TokenService  TokenService
	RefreshService RefreshService
	Rand          crypto.Rand
	Origin        string        // Base URL for verification URIs
	LinkTTL       time.Duration // TTL for device link flows (default 10 min)
	PollInterval  int           // Polling interval in seconds (default 5)
	AccessTTL     time.Duration // Access token TTL
	RefreshTTL    time.Duration // Refresh token TTL
}

type deviceLinkService struct {
	linkStore      store.DeviceLinkStore
	deviceStore    store.DeviceStore
	userStore      store.UserStore
	refreshStore   store.RefreshStore
	auditStore     store.AuditStore
	tokenService   TokenService
	refreshService RefreshService
	rand           crypto.Rand
	origin         string
	linkTTL        time.Duration
	pollInterval   int
	accessTTL      time.Duration
	refreshTTL     time.Duration
}

// NewDeviceLinkService creates a new device link service.
func NewDeviceLinkService(cfg DeviceLinkConfig) DeviceLinkService {
	if cfg.LinkTTL <= 0 {
		cfg.LinkTTL = 10 * time.Minute
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5
	}
	return &deviceLinkService{
		linkStore:      cfg.LinkStore,
		deviceStore:    cfg.DeviceStore,
		userStore:      cfg.UserStore,
		refreshStore:   cfg.RefreshStore,
		auditStore:     cfg.AuditStore,
		tokenService:   cfg.TokenService,
		refreshService: cfg.RefreshService,
		rand:           cfg.Rand,
		origin:         cfg.Origin,
		linkTTL:        cfg.LinkTTL,
		pollInterval:   cfg.PollInterval,
		accessTTL:      cfg.AccessTTL,
		refreshTTL:     cfg.RefreshTTL,
	}
}

// BeginDeviceLink starts a new device linking flow.
func (s *deviceLinkService) BeginDeviceLink(ctx context.Context, deviceName string, purpose string) (*DeviceLinkResponse, error) {
	if deviceName == "" {
		deviceName = "CLI Device"
	}
	if purpose == "" {
		purpose = "login"
	}
	
	// Generate device code (32 bytes, base64url encoded)
	deviceCodeBytes, err := s.rand.Bytes(32)
	if err != nil {
		return nil, err
	}
	deviceCode := base64.RawURLEncoding.EncodeToString(deviceCodeBytes)
	deviceCodeHash := crypto.HashSHA256(deviceCodeBytes)
	
	// Generate user code (8 chars, format: XXXX-XX)
	userCode, err := s.generateUserCode()
	if err != nil {
		return nil, err
	}
	
	// Create the device link
	now := time.Now().UTC()
	link := &domain.DeviceLink{
		ID:         uuid.New(),
		DeviceCode: deviceCodeHash,
		UserCode:   userCode,
		DeviceName: deviceName,
		Purpose:    purpose,
		ExpiresAt:  now.Add(s.linkTTL),
		CreatedAt:  now,
		Interval:   s.pollInterval,
	}
	
	if err := s.linkStore.Create(ctx, link); err != nil {
		return nil, err
	}
	
	// Audit the event
	if s.auditStore != nil {
		event := domain.Event{
			ID:        uuid.New(),
			Type:      domain.EventDeviceLinkBegin,
			CreatedAt: now,
			Metadata: map[string]interface{}{
				"device_name": deviceName,
				"purpose":     purpose,
				"user_code":   userCode,
				"expires_at":  link.ExpiresAt,
			},
		}
		_ = s.auditStore.Record(ctx, event)
	}
	
	// Build response
	return &DeviceLinkResponse{
		DeviceCode:              deviceCode,
		UserCode:                userCode,
		VerificationURI:         fmt.Sprintf("%s/cli/link", s.origin),
		VerificationURIComplete: fmt.Sprintf("%s/cli/link?c=%s", s.origin, userCode),
		ExpiresIn:               int(s.linkTTL.Seconds()),
		Interval:                s.pollInterval,
	}, nil
}

// AuthorizeDeviceLink authorizes a device link after user authentication.
func (s *deviceLinkService) AuthorizeDeviceLink(ctx context.Context, userCode string, userID uuid.UUID) error {
	// Find the device link by user code
	link, err := s.linkStore.GetByUserCode(ctx, userCode)
	if err != nil {
		return err
	}
	if link == nil {
		return errors.New("invalid or expired device code")
	}
	
	// Check if already authorized
	if link.AuthorizedAt != nil {
		return errors.New("device already authorized")
	}
	
	// Authorize the link
	now := time.Now().UTC()
	err = s.linkStore.Authorize(ctx, link.ID, userID, now)
	if err != nil {
		return err
	}
	
	// Audit the event
	if s.auditStore != nil {
		event := domain.Event{
			ID:        uuid.New(),
			Type:      domain.EventDeviceLinkAuthorized,
			UserID:    &userID,
			CreatedAt: now,
			Metadata: map[string]interface{}{
				"device_name": link.DeviceName,
				"purpose":     link.Purpose,
				"user_code":   userCode,
			},
		}
		_ = s.auditStore.Record(ctx, event)
	}
	
	return nil
}

// ExchangeDeviceCode exchanges a device code for tokens.
func (s *deviceLinkService) ExchangeDeviceCode(ctx context.Context, deviceCode string) (*ExchangeResponse, error) {
	// Decode and hash the device code
	deviceCodeBytes, err := base64.RawURLEncoding.DecodeString(deviceCode)
	if err != nil || len(deviceCodeBytes) != 32 {
		return &ExchangeResponse{Status: "expired_token"}, nil
	}
	deviceCodeHash := crypto.HashSHA256(deviceCodeBytes)
	
	// Find the device link
	link, err := s.linkStore.GetByDeviceCode(ctx, deviceCodeHash)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return &ExchangeResponse{Status: "expired_token"}, nil
	}
	
	// Check if authorized
	if link.AuthorizedAt == nil || link.UserID == nil {
		return &ExchangeResponse{Status: "authorization_pending"}, nil
	}
	
	// Get the user
	user, err := s.userStore.GetByID(ctx, *link.UserID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}
	
	// Create a new device record
	device := &domain.Device{
		ID:         uuid.New(),
		UserID:     user.ID,
		DeviceName: link.DeviceName,
		CreatedAt:  time.Now().UTC(),
		LastUsedAt: time.Now().UTC(),
	}
	if err := s.deviceStore.Create(ctx, device); err != nil {
		return nil, err
	}
	
	// Issue refresh token with device ID
	refreshTokenBytes, err := s.rand.Bytes(32)
	if err != nil {
		return nil, err
	}
	refreshToken := base64.RawURLEncoding.EncodeToString(refreshTokenBytes)
	refreshTokenHash := crypto.HashSHA256(refreshTokenBytes)
	
	now := time.Now().UTC()
	expiresAt := now.Add(s.refreshTTL)
	_, _, err = s.refreshStore.CreateFamilyWithDevice(
		ctx,
		user.ID,
		device.ID,
		user.SessionVersion,
		refreshTokenHash,
		expiresAt,
	)
	if err != nil {
		return nil, err
	}
	
	// Issue access token
	claims := AccessClaims{
		Sub:            user.ID.String(),
		Email:          user.Email,
		Roles:          user.Roles,
		SessionVersion: user.SessionVersion,
		AMR:            []string{"device_link"}, // Authentication method reference
		DeviceID:       device.ID.String(),      // Include device ID for tracking
	}
	accessToken, err := s.tokenService.SignAccess(ctx, claims)
	if err != nil {
		return nil, err
	}
	
	// Delete the used device link
	_ = s.linkStore.Delete(ctx, link.ID)
	
	// Audit the successful exchange
	if s.auditStore != nil {
		event := domain.Event{
			ID:        uuid.New(),
			Type:      domain.EventDeviceLinkExchanged,
			UserID:    &user.ID,
			CreatedAt: now,
			Metadata: map[string]interface{}{
				"device_id":   device.ID.String(),
				"device_name": device.DeviceName,
			},
		}
		_ = s.auditStore.Record(ctx, event)
		
		// Also audit device creation
		deviceEvent := domain.Event{
			ID:        uuid.New(),
			Type:      domain.EventDeviceCreated,
			UserID:    &user.ID,
			CreatedAt: now,
			Metadata: map[string]interface{}{
				"device_id":   device.ID.String(),
				"device_name": device.DeviceName,
			},
		}
		_ = s.auditStore.Record(ctx, deviceEvent)
	}
	
	return &ExchangeResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresIn:        int(s.accessTTL.Seconds()),
		RefreshExpiresIn: int(s.refreshTTL.Seconds()),
		DeviceID:         device.ID.String(),
		User: &UserInfo{
			ID:    user.ID.String(),
			Email: user.Email,
			Roles: user.Roles,
		},
	}, nil
}

// generateUserCode generates a user-friendly code like "J7FQ-K9".
func (s *deviceLinkService) generateUserCode() (string, error) {
	// Use characters that are easy to distinguish
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	
	// Generate 7 random bytes
	randBytes, err := s.rand.Bytes(7)
	if err != nil {
		return "", err
	}
	
	// Map random bytes to charset
	codeBytes := make([]byte, 7)
	for i := 0; i < 7; i++ {
		codeBytes[i] = charset[int(randBytes[i])%len(charset)]
	}
	
	// Format as XXXX-XXX
	code := fmt.Sprintf("%s-%s", string(codeBytes[:4]), string(codeBytes[4:]))
	return code, nil
}

// VerifyDeviceCode checks if a device code is valid and returns the associated link.
// This is used by the browser UI to display device information.
func VerifyDeviceCode(ctx context.Context, linkStore store.DeviceLinkStore, userCode string) (*domain.DeviceLink, error) {
	link, err := linkStore.GetByUserCode(ctx, userCode)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return nil, errors.New("invalid or expired code")
	}
	return link, nil
}
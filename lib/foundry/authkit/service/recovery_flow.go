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
	"github.com/google/uuid"
)

var (
	// ErrRecoveryFlowNotFound is returned when a recovery flow doesn't exist.
	ErrRecoveryFlowNotFound = errors.New("recovery flow not found")
	// ErrRecoveryFlowExpired is returned when a recovery flow has expired.
	ErrRecoveryFlowExpired = errors.New("recovery flow expired")
	// ErrRecoveryFlowInvalid is returned when a recovery flow is invalid.
	ErrRecoveryFlowInvalid = errors.New("recovery flow invalid")
)

// RecoveryFlow represents an active account recovery flow.
type RecoveryFlow struct {
	ID        string
	UserID    uuid.UUID
	Email     string
	ExpiresAt time.Time
	Verified  bool
}

// RecoveryFlowService manages the account recovery process.
type RecoveryFlowService interface {
	// InitiateRecovery starts a recovery flow for a user by email.
	// Returns a flow ID to track the recovery process.
	InitiateRecovery(ctx context.Context, email string) (flowID string, err error)
	
	// VerifyRecoveryCode verifies a recovery code for the given flow.
	// Returns the user if successful.
	VerifyRecoveryCode(ctx context.Context, flowID string, code string) (*domain.User, error)
	
	// CompleteRecovery finishes the recovery process after new credentials are registered.
	CompleteRecovery(ctx context.Context, flowID string) error
	
	// GetFlow retrieves the current recovery flow.
	GetFlow(ctx context.Context, flowID string) (*RecoveryFlow, error)
}

// recoveryFlowService implements RecoveryFlowService.
type recoveryFlowService struct {
	userStore    store.UserStore
	recovery     RecoveryService
	kv           store.KV
	rand         crypto.Rand
	flowTTL      time.Duration
}

// NewRecoveryFlowService creates a new recovery flow service.
func NewRecoveryFlowService(
	userStore store.UserStore,
	recovery RecoveryService,
	kv store.KV,
	rand crypto.Rand,
) RecoveryFlowService {
	return &recoveryFlowService{
		userStore: userStore,
		recovery:  recovery,
		kv:        kv,
		rand:      rand,
		flowTTL:   15 * time.Minute, // 15-minute recovery flow validity
	}
}

// InitiateRecovery starts a recovery flow for a user by email.
func (s *recoveryFlowService) InitiateRecovery(ctx context.Context, email string) (string, error) {
	// Look up user by email
	user, err := s.userStore.GetByEmail(ctx, email)
	if err != nil || user == nil {
		// Always return success to prevent enumeration
		// Generate a fake flow ID so the response looks the same
		fakeID, _ := s.rand.String(16)
		// Log the actual error internally for debugging
		if err != nil {
			// In production, log this error for monitoring
			_ = err
		}
		return fakeID, nil
	}
	
	// Generate flow ID
	flowIDBytes, err := s.rand.Bytes(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate flow ID: %w", err)
	}
	flowID := base64.RawURLEncoding.EncodeToString(flowIDBytes)
	
	// Create flow data
	flow := RecoveryFlow{
		ID:        flowID,
		UserID:    user.ID,
		Email:     email,
		ExpiresAt: time.Now().UTC().Add(s.flowTTL),
		Verified:  false,
	}
	
	// Store flow in KV
	if err := s.storeFlow(ctx, &flow); err != nil {
		return "", fmt.Errorf("failed to store recovery flow: %w", err)
	}
	
	// In production, you would send an email here with instructions
	// The email would contain the flow ID and instructions to enter a recovery code
	
	return flowID, nil
}

// VerifyRecoveryCode verifies a recovery code for the given flow.
func (s *recoveryFlowService) VerifyRecoveryCode(ctx context.Context, flowID string, code string) (*domain.User, error) {
	// Get the flow
	flow, err := s.getFlow(ctx, flowID)
	if err != nil {
		return nil, err
	}
	
	// Check if flow is already verified (defense-in-depth)
	if flow.Verified {
		return nil, ErrRecoveryFlowInvalid
	}
	
	// Check if flow is expired
	if time.Now().UTC().After(flow.ExpiresAt) {
		return nil, ErrRecoveryFlowExpired
	}
	
	// Verify the recovery code
	if err := s.recovery.ValidateCode(ctx, flow.UserID, code); err != nil {
		if err == ErrRecoveryCodeInvalid {
			return nil, ErrRecoveryFlowInvalid
		}
		return nil, fmt.Errorf("failed to validate recovery code: %w", err)
	}
	
	// Mark flow as verified
	flow.Verified = true
	if err := s.storeFlow(ctx, flow); err != nil {
		return nil, fmt.Errorf("failed to update recovery flow: %w", err)
	}
	
	// Get and return the user
	user, err := s.userStore.GetByID(ctx, flow.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	// Bump session version to invalidate existing sessions
	if err := s.userStore.BumpSessionVersion(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to update user session version: %w", err)
	}
	
	return user, nil
}

// CompleteRecovery finishes the recovery process after new credentials are registered.
func (s *recoveryFlowService) CompleteRecovery(ctx context.Context, flowID string) error {
	// Get the flow
	flow, err := s.getFlow(ctx, flowID)
	if err != nil {
		return err
	}
	
	// Check if flow is verified
	if !flow.Verified {
		return ErrRecoveryFlowInvalid
	}
	
	// Check if flow is expired
	if time.Now().UTC().After(flow.ExpiresAt) {
		return ErrRecoveryFlowExpired
	}
	
	// Delete the flow
	key := s.makeFlowKey(flowID)
	if err := s.kv.Del(ctx, key); err != nil {
		return fmt.Errorf("failed to delete recovery flow: %w", err)
	}
	
	// Regenerate recovery codes since one was just used
	// This ensures the user always has a fresh set after recovery
	if _, err := s.recovery.RegenerateCodes(ctx, flow.UserID, 10); err != nil {
		// Log error but don't fail the recovery
		// User can regenerate codes manually later
		_ = err
	}
	
	return nil
}

// GetFlow retrieves the current recovery flow.
func (s *recoveryFlowService) GetFlow(ctx context.Context, flowID string) (*RecoveryFlow, error) {
	return s.getFlow(ctx, flowID)
}

// getFlow retrieves a flow from storage.
func (s *recoveryFlowService) getFlow(ctx context.Context, flowID string) (*RecoveryFlow, error) {
	key := s.makeFlowKey(flowID)
	
	data, err := s.kv.Get(ctx, key)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, ErrRecoveryFlowNotFound
		}
		return nil, fmt.Errorf("failed to get recovery flow: %w", err)
	}
	
	// Deserialize flow from JSON
	var flow RecoveryFlow
	if err := json.Unmarshal(data, &flow); err != nil {
		return nil, ErrRecoveryFlowInvalid
	}
	
	// Ensure the flow ID matches (defense in depth)
	flow.ID = flowID
	
	return &flow, nil
}

// storeFlow stores a flow in KV storage.
func (s *recoveryFlowService) storeFlow(ctx context.Context, flow *RecoveryFlow) error {
	key := s.makeFlowKey(flow.ID)
	
	// Serialize flow to JSON
	data, err := json.Marshal(flow)
	if err != nil {
		return fmt.Errorf("failed to serialize recovery flow: %w", err)
	}
	
	ttl := time.Until(flow.ExpiresAt)
	if ttl <= 0 {
		// Don't store expired flows - they should be deleted instead
		// If somehow we're trying to store an expired flow, use 1 second TTL
		// to ensure it's deleted quickly
		ttl = 1 * time.Second
	}
	
	return s.kv.Set(ctx, key, data, ttl)
}

// makeFlowKey creates a storage key for a recovery flow.
func (s *recoveryFlowService) makeFlowKey(flowID string) string {
	return fmt.Sprintf("recovery:flow:%s", flowID)
}
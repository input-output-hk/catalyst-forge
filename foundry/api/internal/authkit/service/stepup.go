package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
	"github.com/google/uuid"
)

var (
	// ErrStepUpRequired is returned when step-up authentication is required.
	ErrStepUpRequired = errors.New("step-up authentication required")
	// ErrStepUpExpired is returned when a step-up grant has expired.
	ErrStepUpExpired = errors.New("step-up authentication expired")
	// ErrStepUpNotFound is returned when a step-up grant is not found.
	ErrStepUpNotFound = errors.New("step-up authentication not found")
)

// StepUpService manages step-up authentication grants.
type StepUpService interface {
	// GrantStepUp records a successful step-up authentication.
	// The grant expires after the specified TTL.
	GrantStepUp(ctx context.Context, userID uuid.UUID, action string, ttl time.Duration) error
	
	// ValidateStepUp checks if a user has a valid step-up grant for an action.
	ValidateStepUp(ctx context.Context, userID uuid.UUID, action string) error
	
	// RevokeStepUp removes a step-up grant.
	RevokeStepUp(ctx context.Context, userID uuid.UUID, action string) error
	
	// RevokeAllStepUps removes all step-up grants for a user.
	RevokeAllStepUps(ctx context.Context, userID uuid.UUID) error
}

// stepUpService implements StepUpService.
type stepUpService struct {
	kv          store.KV
	defaultTTL  time.Duration
	maxTTL      time.Duration
}

// NewStepUpService creates a new step-up service.
func NewStepUpService(kv store.KV, defaultTTL time.Duration) StepUpService {
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute // Default 5-minute step-up validity
	}
	
	return &stepUpService{
		kv:         kv,
		defaultTTL: defaultTTL,
		maxTTL:     30 * time.Minute, // Maximum 30-minute step-up validity
	}
}

// GrantStepUp records a successful step-up authentication.
func (s *stepUpService) GrantStepUp(ctx context.Context, userID uuid.UUID, action string, ttl time.Duration) error {
	if s.kv == nil {
		// No KV store configured, can't grant step-up
		return errors.New("step-up service requires KV store")
	}
	
	// Use default TTL if not specified
	if ttl <= 0 {
		ttl = s.defaultTTL
	}
	
	// Cap TTL to maximum
	if ttl > s.maxTTL {
		ttl = s.maxTTL
	}
	
	// Create key based on user and action
	key := s.makeKey(userID, action)
	
	// Store the grant with expiry
	value := []byte(time.Now().UTC().Format(time.RFC3339))
	if err := s.kv.Set(ctx, key, value, ttl); err != nil {
		return fmt.Errorf("failed to grant step-up: %w", err)
	}
	
	return nil
}

// ValidateStepUp checks if a user has a valid step-up grant for an action.
func (s *stepUpService) ValidateStepUp(ctx context.Context, userID uuid.UUID, action string) error {
	if s.kv == nil {
		// No KV store configured, can't validate
		return ErrStepUpRequired
	}
	
	key := s.makeKey(userID, action)
	
	// Check if grant exists
	val, err := s.kv.Get(ctx, key)
	if err != nil {
		if err == store.ErrNotFound {
			return ErrStepUpRequired
		}
		return fmt.Errorf("failed to validate step-up: %w", err)
	}
	
	// Parse the timestamp to ensure it's valid
	grantTime, err := time.Parse(time.RFC3339, string(val))
	if err != nil {
		// Invalid timestamp, treat as expired
		return ErrStepUpExpired
	}
	
	// The KV store handles TTL, but double-check the grant isn't too old
	// This protects against KV store bugs or clock issues
	if time.Now().UTC().Sub(grantTime) > s.maxTTL {
		// Grant is too old, should have expired
		_ = s.kv.Del(ctx, key) // Clean up stale entry
		return ErrStepUpExpired
	}
	
	return nil
}

// RevokeStepUp removes a step-up grant.
func (s *stepUpService) RevokeStepUp(ctx context.Context, userID uuid.UUID, action string) error {
	if s.kv == nil {
		// No KV store configured
		return nil
	}
	
	key := s.makeKey(userID, action)
	return s.kv.Del(ctx, key)
}

// RevokeAllStepUps removes all step-up grants for a user.
func (s *stepUpService) RevokeAllStepUps(ctx context.Context, userID uuid.UUID) error {
	if s.kv == nil {
		// No KV store configured
		return nil
	}
	
	// Delete common action grants
	commonActions := []string{"", "sensitive", "admin", "delete", "credentials"}
	for _, action := range commonActions {
		key := s.makeKey(userID, action)
		_ = s.kv.Del(ctx, key) // Ignore errors for non-existent keys
	}
	
	// Note: This is a best-effort approach. In production, you might want to
	// use a KV store that supports prefix deletion or pattern matching.
	
	return nil
}

// makeKey creates a storage key for a step-up grant.
func (s *stepUpService) makeKey(userID uuid.UUID, action string) string {
	if action == "" {
		action = "default"
	}
	return fmt.Sprintf("stepup:%s:%s", userID.String(), action)
}

// StepUpGrant represents a step-up authentication grant with metadata.
type StepUpGrant struct {
	UserID    uuid.UUID
	Action    string
	GrantedAt time.Time
	ExpiresAt time.Time
}

// StepUpServiceWithMetadata extends StepUpService with metadata support.
type StepUpServiceWithMetadata interface {
	StepUpService
	
	// GrantStepUpWithMetadata records a step-up grant with additional metadata.
	GrantStepUpWithMetadata(ctx context.Context, grant StepUpGrant) error
	
	// GetGrant retrieves the current step-up grant if valid.
	GetGrant(ctx context.Context, userID uuid.UUID, action string) (*StepUpGrant, error)
}
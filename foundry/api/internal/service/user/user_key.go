package user

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks --out ./mocks/user_key.go . UserKeyService

// UserKeyService defines the interface for user key service operations
type UserKeyService interface {
	CreateUserKey(userKey *user.UserKey) error
	GetUserKeyByID(id uint) (*user.UserKey, error)
	GetUserKeyByKid(kid string) (*user.UserKey, error)
	GetUserKeysByUserID(userID uint) ([]user.UserKey, error)
	GetActiveUserKeysByUserID(userID uint) ([]user.UserKey, error)
	GetInactiveUserKeysByUserID(userID uint) ([]user.UserKey, error)
	GetInactiveUserKeys() ([]user.UserKey, error)
	UpdateUserKey(userKey *user.UserKey) error
	DeleteUserKey(id uint) error
	RevokeUserKey(id uint) error
	ListUserKeys() ([]user.UserKey, error)
}

// DefaultUserKeyService is the default implementation of UserKeyService
type DefaultUserKeyService struct {
	repo   userrepo.UserKeyRepository
	logger *slog.Logger
}

// NewUserKeyService creates a new user key service
func NewUserKeyService(repo userrepo.UserKeyRepository, logger *slog.Logger) *DefaultUserKeyService {
	return &DefaultUserKeyService{
		repo:   repo,
		logger: logger,
	}
}

// CreateUserKey creates a new user key
func (s *DefaultUserKeyService) CreateUserKey(userKey *user.UserKey) error {
	// Validate key data
	if err := s.validateUserKey(userKey); err != nil {
		return fmt.Errorf("invalid user key: %w", err)
	}

	// Check if kid already exists
	existing, err := s.repo.GetByKid(userKey.Kid)
	if err == nil && existing != nil {
		return fmt.Errorf("user key already exists with kid: %s", userKey.Kid)
	}

	// Set default status if not provided
	if userKey.Status == "" {
		userKey.Status = user.UserKeyStatusActive
	}

	s.logger.Info("Creating user key",
		"user_id", userKey.UserID,
		"kid", userKey.Kid,
		"status", userKey.Status)

	return s.repo.Create(userKey)
}

// GetUserKeyByID retrieves a user key by ID
func (s *DefaultUserKeyService) GetUserKeyByID(id uint) (*user.UserKey, error) {
	return s.repo.GetByID(id)
}

// GetUserKeyByKid retrieves a user key by kid (key ID)
func (s *DefaultUserKeyService) GetUserKeyByKid(kid string) (*user.UserKey, error) {
	return s.repo.GetByKid(kid)
}

// GetUserKeysByUserID retrieves all keys for a specific user
func (s *DefaultUserKeyService) GetUserKeysByUserID(userID uint) ([]user.UserKey, error) {
	return s.repo.GetByUserID(userID)
}

// GetActiveUserKeysByUserID retrieves all active keys for a specific user
func (s *DefaultUserKeyService) GetActiveUserKeysByUserID(userID uint) ([]user.UserKey, error) {
	return s.repo.GetActiveByUserID(userID)
}

// GetInactiveUserKeysByUserID retrieves all inactive keys for a specific user
func (s *DefaultUserKeyService) GetInactiveUserKeysByUserID(userID uint) ([]user.UserKey, error) {
	return s.repo.GetInactiveByUserID(userID)
}

// GetInactiveUserKeys retrieves all inactive keys
func (s *DefaultUserKeyService) GetInactiveUserKeys() ([]user.UserKey, error) {
	return s.repo.GetInactive()
}

// UpdateUserKey updates an existing user key
func (s *DefaultUserKeyService) UpdateUserKey(userKey *user.UserKey) error {
	// Validate key data
	if err := s.validateUserKey(userKey); err != nil {
		return fmt.Errorf("invalid user key: %w", err)
	}

	s.logger.Info("Updating user key",
		"id", userKey.ID,
		"user_id", userKey.UserID,
		"kid", userKey.Kid,
		"status", userKey.Status)

	return s.repo.Update(userKey)
}

// DeleteUserKey deletes a user key
func (s *DefaultUserKeyService) DeleteUserKey(id uint) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get user key: %w", err)
	}

	s.logger.Info("Deleting user key",
		"id", id,
		"user_id", existing.UserID,
		"kid", existing.Kid)

	return s.repo.Delete(id)
}

// RevokeUserKey revokes a user key by setting status to revoked
func (s *DefaultUserKeyService) RevokeUserKey(id uint) error {
	userKey, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get user key: %w", err)
	}

	userKey.Status = user.UserKeyStatusRevoked

	s.logger.Info("Revoking user key",
		"id", id,
		"user_id", userKey.UserID,
		"kid", userKey.Kid)

	return s.repo.Update(userKey)
}

// ListUserKeys retrieves all user keys
func (s *DefaultUserKeyService) ListUserKeys() ([]user.UserKey, error) {
	return s.repo.List()
}

// validateUserKey validates user key data
func (s *DefaultUserKeyService) validateUserKey(userKey *user.UserKey) error {
	if userKey.UserID == 0 {
		return fmt.Errorf("user_id cannot be empty")
	}
	if userKey.Kid == "" {
		return fmt.Errorf("kid cannot be empty")
	}
	if userKey.PubKeyB64 == "" {
		return fmt.Errorf("pubkey_b64 cannot be empty")
	}
	// Add more validation logic as needed
	return nil
}

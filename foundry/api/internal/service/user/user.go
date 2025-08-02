package user

import (
	"fmt"
	"log/slog"

	um "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks --out ./mocks/user.go . UserService

// UserService defines the interface for user service operations
type UserService interface {
	CreateUser(user *um.User) error
	GetUserByID(id uint) (*um.User, error)
	GetUserByEmail(email string) (*um.User, error)
	UpdateUser(user *um.User) error
	DeleteUser(id uint) error
	ListUsers() ([]um.User, error)
	GetPendingUsers() ([]um.User, error)
	ActivateUser(id uint) error
	DeactivateUser(id uint) error
}

// DefaultUserService is the default implementation of UserService
type DefaultUserService struct {
	repo   userrepo.UserRepository
	logger *slog.Logger
}

// NewUserService creates a new user service
func NewUserService(repo userrepo.UserRepository, logger *slog.Logger) *DefaultUserService {
	return &DefaultUserService{
		repo:   repo,
		logger: logger,
	}
}

// CreateUser creates a new user
func (s *DefaultUserService) CreateUser(user *um.User) error {
	// Validate email format
	if err := s.validateEmail(user.Email); err != nil {
		return fmt.Errorf("invalid email format: %w", err)
	}

	// Check if user already exists
	existing, err := s.repo.GetByEmail(user.Email)
	if err == nil && existing != nil {
		return fmt.Errorf("user already exists with email: %s", user.Email)
	}

	// Set default status if not provided
	if user.Status == "" {
		user.Status = um.UserStatusPending
	}

	s.logger.Info("Creating user",
		"email", user.Email,
		"status", user.Status)

	return s.repo.Create(user)
}

// GetUserByID retrieves a user by ID
func (s *DefaultUserService) GetUserByID(id uint) (*um.User, error) {
	return s.repo.GetByID(id)
}

// GetUserByEmail retrieves a user by email
func (s *DefaultUserService) GetUserByEmail(email string) (*um.User, error) {
	return s.repo.GetByEmail(email)
}

// UpdateUser updates an existing user
func (s *DefaultUserService) UpdateUser(user *um.User) error {
	// Validate email format if changed
	if err := s.validateEmail(user.Email); err != nil {
		return fmt.Errorf("invalid email format: %w", err)
	}

	s.logger.Info("Updating user",
		"id", user.ID,
		"email", user.Email,
		"status", user.Status)

	return s.repo.Update(user)
}

// DeleteUser deletes a user
func (s *DefaultUserService) DeleteUser(id uint) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	s.logger.Info("Deleting user",
		"id", id,
		"email", existing.Email)

	return s.repo.Delete(id)
}

// ListUsers retrieves all users
func (s *DefaultUserService) ListUsers() ([]um.User, error) {
	return s.repo.List()
}

// GetPendingUsers retrieves all users with pending status
func (s *DefaultUserService) GetPendingUsers() ([]um.User, error) {
	return s.repo.GetByStatus(um.UserStatusPending)
}

// ActivateUser activates a user
func (s *DefaultUserService) ActivateUser(id uint) error {
	u, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	u.Status = um.UserStatusActive

	s.logger.Info("Activating user",
		"id", id,
		"email", u.Email)

	return s.repo.Update(u)
}

// DeactivateUser deactivates a user
func (s *DefaultUserService) DeactivateUser(id uint) error {
	u, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	u.Status = um.UserStatusInactive

	s.logger.Info("Deactivating user",
		"id", id,
		"email", u.Email)

	return s.repo.Update(u)
}

// validateEmail validates email format
func (s *DefaultUserService) validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	// Add more email validation logic as needed
	return nil
}

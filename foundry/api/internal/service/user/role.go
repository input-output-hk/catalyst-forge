package user

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks --out ./mocks/role.go . RoleService

// RoleService defines the interface for role service operations
type RoleService interface {
	CreateRole(role *user.Role) error
	GetRoleByID(id uint) (*user.Role, error)
	GetRoleByName(name string) (*user.Role, error)
	UpdateRole(role *user.Role) error
	DeleteRole(id uint) error
	ListRoles() ([]user.Role, error)
}

// DefaultRoleService is the default implementation of RoleService
type DefaultRoleService struct {
	repo   userrepo.RoleRepository
	logger *slog.Logger
}

// NewRoleService creates a new role service
func NewRoleService(repo userrepo.RoleRepository, logger *slog.Logger) *DefaultRoleService {
	return &DefaultRoleService{
		repo:   repo,
		logger: logger,
	}
}

// CreateRole creates a new role
func (s *DefaultRoleService) CreateRole(role *user.Role) error {
	// Validate role name
	if err := s.validateRoleName(role.Name); err != nil {
		return fmt.Errorf("invalid role name: %w", err)
	}

	// Check if role already exists
	existing, err := s.repo.GetByName(role.Name)
	if err == nil && existing != nil {
		return fmt.Errorf("role already exists with name: %s", role.Name)
	}

	s.logger.Info("Creating role",
		"name", role.Name)

	return s.repo.Create(role)
}

// GetRoleByID retrieves a role by ID
func (s *DefaultRoleService) GetRoleByID(id uint) (*user.Role, error) {
	return s.repo.GetByID(fmt.Sprintf("%d", id))
}

// GetRoleByName retrieves a role by name
func (s *DefaultRoleService) GetRoleByName(name string) (*user.Role, error) {
	return s.repo.GetByName(name)
}

// UpdateRole updates an existing role
func (s *DefaultRoleService) UpdateRole(role *user.Role) error {
	// Validate role name
	if err := s.validateRoleName(role.Name); err != nil {
		return fmt.Errorf("invalid role name: %w", err)
	}

	s.logger.Info("Updating role",
		"id", role.ID,
		"name", role.Name)

	return s.repo.Update(role)
}

// DeleteRole deletes a role
func (s *DefaultRoleService) DeleteRole(id uint) error {
	existing, err := s.repo.GetByID(fmt.Sprintf("%d", id))
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	s.logger.Info("Deleting role",
		"id", id,
		"name", existing.Name)

	return s.repo.Delete(fmt.Sprintf("%d", id))
}

// ListRoles retrieves all roles
func (s *DefaultRoleService) ListRoles() ([]user.Role, error) {
	return s.repo.List()
}

// validateRoleName validates role name format
func (s *DefaultRoleService) validateRoleName(name string) error {
	if name == "" {
		return fmt.Errorf("role name cannot be empty")
	}
	// Add more role name validation logic as needed
	return nil
}

package user

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
)

// UserRoleService defines the interface for user role service operations
type UserRoleService interface {
	AssignUserToRole(userID, roleID uint) error
	RemoveUserFromRole(userID, roleID uint) error
	GetUserRoles(userID uint) ([]user.UserRole, error)
	GetRoleUsers(roleID uint) ([]user.UserRole, error)
	GetUserRole(userID, roleID uint) (*user.UserRole, error)
}

// DefaultUserRoleService is the default implementation of UserRoleService
type DefaultUserRoleService struct {
	repo   userrepo.UserRoleRepository
	logger *slog.Logger
}

// NewUserRoleService creates a new user role service
func NewUserRoleService(repo userrepo.UserRoleRepository, logger *slog.Logger) *DefaultUserRoleService {
	return &DefaultUserRoleService{
		repo:   repo,
		logger: logger,
	}
}

// AssignUserToRole assigns a user to a role
func (s *DefaultUserRoleService) AssignUserToRole(userID, roleID uint) error {
	userRole := &user.UserRole{
		UserID: userID,
		RoleID: roleID,
	}

	if err := s.repo.Create(userRole); err != nil {
		s.logger.Error("Failed to assign user to role", "error", err, "userID", userID, "roleID", roleID)
		return err
	}

	s.logger.Info("User assigned to role", "userID", userID, "roleID", roleID)
	return nil
}

// RemoveUserFromRole removes a user from a role
func (s *DefaultUserRoleService) RemoveUserFromRole(userID, roleID uint) error {
	if err := s.repo.DeleteByUserIDAndRoleID(fmt.Sprintf("%d", userID), fmt.Sprintf("%d", roleID)); err != nil {
		s.logger.Error("Failed to remove user from role", "error", err, "userID", userID, "roleID", roleID)
		return err
	}

	s.logger.Info("User removed from role", "userID", userID, "roleID", roleID)
	return nil
}

// GetUserRoles retrieves all roles for a specific user
func (s *DefaultUserRoleService) GetUserRoles(userID uint) ([]user.UserRole, error) {
	userRoles, err := s.repo.GetByUserID(fmt.Sprintf("%d", userID))
	if err != nil {
		s.logger.Error("Failed to get user roles", "error", err, "userID", userID)
		return nil, err
	}

	return userRoles, nil
}

// GetRoleUsers retrieves all users for a specific role
func (s *DefaultUserRoleService) GetRoleUsers(roleID uint) ([]user.UserRole, error) {
	userRoles, err := s.repo.GetByRoleID(fmt.Sprintf("%d", roleID))
	if err != nil {
		s.logger.Error("Failed to get role users", "error", err, "roleID", roleID)
		return nil, err
	}

	return userRoles, nil
}

// GetUserRole retrieves a specific user-role relationship
func (s *DefaultUserRoleService) GetUserRole(userID, roleID uint) (*user.UserRole, error) {
	userRoles, err := s.repo.GetByUserID(fmt.Sprintf("%d", userID))
	if err != nil {
		s.logger.Error("Failed to get user roles", "error", err, "userID", userID)
		return nil, err
	}

	for _, userRole := range userRoles {
		if userRole.RoleID == roleID {
			return &userRole, nil
		}
	}

	return nil, nil
}

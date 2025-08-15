package gormstore

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
	repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserStore is a GORM-based implementation of store.UserStore.
type UserStore struct {
	db *gorm.DB
}

// NewUserStore creates a new GORM-based user store.
func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

// dbFor returns the appropriate database handle for the given context.
// If a transaction is present in the context, it returns the transaction handle.
// Otherwise, it returns the default database handle.
func (s *UserStore) dbFor(ctx context.Context) *gorm.DB {
	if tx := repodb.TxFromContext(ctx); tx != nil {
		return tx
	}
	return s.db
}

// Create creates a new user with the given email and roles.
func (s *UserStore) Create(ctx context.Context, email string, roles []string) (*domain.User, error) {
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:             uuid.New(),
		Email:          email,
		RolesJSON:      string(rolesJSON),
		SessionVersion: 1,
	}

	if err := s.dbFor(ctx).WithContext(ctx).Create(user).Error; err != nil {
		return nil, err
	}

	return s.toDomain(user)
}

// GetByEmail retrieves a user by their email address.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user User
	if err := s.dbFor(ctx).WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return s.toDomain(&user)
}

// GetByID retrieves a user by their ID.
func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user User
	if err := s.dbFor(ctx).WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return s.toDomain(&user)
}

// UpdateRoles updates the roles for a user.
func (s *UserStore) UpdateRoles(ctx context.Context, id uuid.UUID, roles []string) error {
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return err
	}

	result := s.dbFor(ctx).WithContext(ctx).Model(&User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"roles":      string(rolesJSON),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// BumpSessionVersion increments the session version to invalidate all sessions.
func (s *UserStore) BumpSessionVersion(ctx context.Context, id uuid.UUID) error {
	result := s.dbFor(ctx).WithContext(ctx).Model(&User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"session_version": gorm.Expr("session_version + ?", 1),
			"updated_at":      time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// toDomain converts a database model to a domain entity.
func (s *UserStore) toDomain(user *User) (*domain.User, error) {
	var roles []string
	if err := json.Unmarshal([]byte(user.RolesJSON), &roles); err != nil {
		return nil, err
	}

	return &domain.User{
		ID:             user.ID,
		Email:          user.Email,
		Roles:          roles,
		SessionVersion: user.SessionVersion,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}, nil
}
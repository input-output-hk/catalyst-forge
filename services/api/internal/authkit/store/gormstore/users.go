package gormstore

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
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
		FullName:       "",
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

// UpdateFullName updates the user's display name.
func (s *UserStore) UpdateFullName(ctx context.Context, id uuid.UUID, fullName string) error {
	result := s.dbFor(ctx).WithContext(ctx).Model(&User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"full_name":  fullName,
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

// toDomain converts a database model to a domain entity.
func (s *UserStore) toDomain(user *User) (*domain.User, error) {
	var roles []string
	if err := json.Unmarshal([]byte(user.RolesJSON), &roles); err != nil {
		return nil, err
	}

	return &domain.User{
		ID:             user.ID,
		Email:          user.Email,
		FullName:       user.FullName,
		Roles:          roles,
		SessionVersion: user.SessionVersion,
		SuspendedAt:    user.SuspendedAt,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}, nil
}

// List returns users ordered by created_at desc with limit/offset.
func (s *UserStore) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var rows []User
	if err := s.dbFor(ctx).WithContext(ctx).Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.User, 0, len(rows))
	for i := range rows {
		du, err := s.toDomain(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, du)
	}
	return out, nil
}

// ListFiltered returns users filtered by q (email icontains) and role, ordered by created_at desc.
func (s *UserStore) ListFiltered(ctx context.Context, q string, role string, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	db := s.dbFor(ctx).WithContext(ctx).Model(&User{})
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("LOWER(email) LIKE LOWER(?)", like)
	}
	if role != "" {
		// roles stored as JSON text in column roles; use LIKE to find role token
		like := "%\"" + role + "\"%"
		db = db.Where("roles LIKE ?", like)
	}
	var rows []User
	if err := db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.User, 0, len(rows))
	for i := range rows {
		du, err := s.toDomain(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, du)
	}
	return out, nil
}

// CountFiltered returns total count for given filters
func (s *UserStore) CountFiltered(ctx context.Context, q string, role string) (int64, error) {
	db := s.dbFor(ctx).WithContext(ctx).Model(&User{})
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("LOWER(email) LIKE LOWER(?)", like)
	}
	if role != "" {
		like := "%\"" + role + "\"%"
		db = db.Where("roles LIKE ?", like)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// UpdateSuspended sets or clears the suspended_at timestamp.
func (s *UserStore) UpdateSuspended(ctx context.Context, id uuid.UUID, suspended bool, at time.Time) error {
	updates := map[string]interface{}{"updated_at": time.Now()}
	if suspended {
		updates["suspended_at"] = at
	} else {
		updates["suspended_at"] = nil
	}
	result := s.dbFor(ctx).WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

// Delete permanently removes a user and related auth data where appropriate.
func (s *UserStore) Delete(ctx context.Context, id uuid.UUID) error {
	db := s.dbFor(ctx).WithContext(ctx)
	// Delete related records first to avoid constraint errors
	if err := db.Where("user_id = ?", id).Delete(&Credential{}).Error; err != nil {
		return err
	}
	if err := db.Where("user_id = ?", id).Delete(&RefreshToken{}).Error; err != nil {
		return err
	}
	if err := db.Where("user_id = ?", id).Delete(&RecoveryCode{}).Error; err != nil {
		return err
	}
	if err := db.Where("user_id = ?", id).Delete(&Device{}).Error; err != nil {
		return err
	}
	// Finally delete user
	if err := db.Where("id = ?", id).Unscoped().Delete(&User{}).Error; err != nil {
		return err
	}
	return nil
}

// AccessRequestStore implements store.AccessRequestStore using GORM.
type AccessRequestStore struct{ db *gorm.DB }

func NewAccessRequestStore(db *gorm.DB) *AccessRequestStore { return &AccessRequestStore{db: db} }

func (s *AccessRequestStore) dbFor(ctx context.Context) *gorm.DB { return s.db }

func (s *AccessRequestStore) CreateOrBump(ctx context.Context, email string, reason string, now time.Time) (*domain.AccessRequest, error) {
	var ar AccessRequest
	tx := s.dbFor(ctx).WithContext(ctx)
	if err := tx.Where("email = ?", email).First(&ar).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ar = AccessRequest{ID: uuid.New(), Email: email, Reason: reason, Status: "pending", Attempts: 1, CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(&ar).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		if err := tx.Model(&AccessRequest{}).Where("id = ?", ar.ID).Updates(map[string]interface{}{"attempts": gorm.Expr("attempts + 1"), "reason": reason, "status": "pending", "updated_at": now}).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("id = ?", ar.ID).First(&ar).Error; err != nil {
			return nil, err
		}
	}
	return &domain.AccessRequest{ID: ar.ID, Email: ar.Email, Reason: ar.Reason, Status: ar.Status, Attempts: ar.Attempts, DecidedAt: ar.DecidedAt, CreatedAt: ar.CreatedAt, UpdatedAt: ar.UpdatedAt}, nil
}

func (s *AccessRequestStore) List(ctx context.Context, status string, q string, limit, offset int) ([]*domain.AccessRequest, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	db := s.dbFor(ctx).WithContext(ctx).Model(&AccessRequest{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("LOWER(email) LIKE LOWER(?) OR LOWER(reason) LIKE LOWER(?)", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []AccessRequest
	if err := db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*domain.AccessRequest, 0, len(rows))
	for i := range rows {
		r := rows[i]
		out = append(out, &domain.AccessRequest{ID: r.ID, Email: r.Email, Reason: r.Reason, Status: r.Status, Attempts: r.Attempts, DecidedAt: r.DecidedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return out, total, nil
}

func (s *AccessRequestStore) Decide(ctx context.Context, id uuid.UUID, approve bool, decidedBy uuid.UUID, note string, now time.Time) error {
	updates := map[string]interface{}{"status": func() string {
		if approve {
			return "approved"
		} else {
			return "rejected"
		}
	}(), "decided_at": now, "decided_by": decidedBy, "updated_at": now, "reason": note}
	res := s.dbFor(ctx).WithContext(ctx).Model(&AccessRequest{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("access request not found")
	}
	return nil
}

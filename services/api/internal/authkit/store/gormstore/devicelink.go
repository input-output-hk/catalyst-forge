package gormstore

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
	"gorm.io/gorm"
)

// deviceLinkStore implements store.DeviceLinkStore using GORM.
type deviceLinkStore struct {
	db *gorm.DB
}

// NewDeviceLinkStore creates a new GORM-backed device link store.
func NewDeviceLinkStore(db *gorm.DB) store.DeviceLinkStore {
	return &deviceLinkStore{db: db}
}

// Create stores a new device link flow.
func (s *deviceLinkStore) Create(ctx context.Context, link *domain.DeviceLink) error {
	dbLink := DeviceLink{
		ID:           link.ID,
		DeviceCode:   link.DeviceCode,
		UserCode:     link.UserCode,
		DeviceName:   link.DeviceName,
		Purpose:      link.Purpose,
		UserID:       link.UserID,
		AuthorizedAt: link.AuthorizedAt,
		ExpiresAt:    link.ExpiresAt,
		CreatedAt:    link.CreatedAt,
		Interval:     link.Interval,
	}
	return s.db.WithContext(ctx).Create(&dbLink).Error
}

// GetByDeviceCode retrieves a device link by hashed device code.
func (s *deviceLinkStore) GetByDeviceCode(ctx context.Context, deviceCodeHash []byte) (*domain.DeviceLink, error) {
	var dbLink DeviceLink
	err := s.db.WithContext(ctx).
		Where("device_code = ? AND expires_at > ?", deviceCodeHash, time.Now().UTC()).
		First(&dbLink).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toDomainDeviceLink(&dbLink), nil
}

// GetByUserCode retrieves a device link by user code.
func (s *deviceLinkStore) GetByUserCode(ctx context.Context, userCode string) (*domain.DeviceLink, error) {
	var dbLink DeviceLink
	err := s.db.WithContext(ctx).
		Where("user_code = ? AND expires_at > ?", userCode, time.Now().UTC()).
		First(&dbLink).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toDomainDeviceLink(&dbLink), nil
}

// Authorize marks a device link as authorized by a user.
func (s *deviceLinkStore) Authorize(ctx context.Context, id uuid.UUID, userID uuid.UUID, authorizedAt time.Time) error {
	return s.db.WithContext(ctx).
		Model(&DeviceLink{}).
		Where("id = ? AND authorized_at IS NULL", id).
		Updates(map[string]interface{}{
			"user_id":       userID,
			"authorized_at": authorizedAt,
		}).Error
}

// Delete removes an expired or completed device link.
func (s *deviceLinkStore) Delete(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Delete(&DeviceLink{}, id).Error
}

// DeleteExpired removes all expired device links.
func (s *deviceLinkStore) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	result := s.db.WithContext(ctx).
		Where("expires_at < ?", before).
		Delete(&DeviceLink{})
	return result.RowsAffected, result.Error
}

// toDomainDeviceLink converts a GORM model to domain model.
func toDomainDeviceLink(dbLink *DeviceLink) *domain.DeviceLink {
	return &domain.DeviceLink{
		ID:           dbLink.ID,
		DeviceCode:   dbLink.DeviceCode,
		UserCode:     dbLink.UserCode,
		DeviceName:   dbLink.DeviceName,
		Purpose:      dbLink.Purpose,
		UserID:       dbLink.UserID,
		AuthorizedAt: dbLink.AuthorizedAt,
		ExpiresAt:    dbLink.ExpiresAt,
		CreatedAt:    dbLink.CreatedAt,
		Interval:     dbLink.Interval,
	}
}

// deviceStore implements store.DeviceStore using GORM.
type deviceStore struct {
	db *gorm.DB
}

// NewDeviceStore creates a new GORM-backed device store.
func NewDeviceStore(db *gorm.DB) store.DeviceStore {
	return &deviceStore{db: db}
}

// Create stores a new device.
func (s *deviceStore) Create(ctx context.Context, device *domain.Device) error {
	dbDevice := Device{
		ID:         device.ID,
		UserID:     device.UserID,
		DeviceName: device.DeviceName,
		CreatedAt:  device.CreatedAt,
		LastUsedAt: device.LastUsedAt,
		RevokedAt:  device.RevokedAt,
	}
	return s.db.WithContext(ctx).Create(&dbDevice).Error
}

// GetByID retrieves a device by ID.
func (s *deviceStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Device, error) {
	var dbDevice Device
	err := s.db.WithContext(ctx).
		Where("id = ? AND revoked_at IS NULL", id).
		First(&dbDevice).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toDomainDevice(&dbDevice), nil
}

// GetByUser retrieves all devices for a user.
func (s *deviceStore) GetByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Device, error) {
	var dbDevices []Device
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Order("created_at DESC").
		Find(&dbDevices).Error
	if err != nil {
		return nil, err
	}
	
	devices := make([]*domain.Device, len(dbDevices))
	for i, dbDevice := range dbDevices {
		devices[i] = toDomainDevice(&dbDevice)
	}
	return devices, nil
}

// UpdateLastUsed updates the last used timestamp.
func (s *deviceStore) UpdateLastUsed(ctx context.Context, id uuid.UUID, lastUsedAt time.Time) error {
	return s.db.WithContext(ctx).
		Model(&Device{}).
		Where("id = ?", id).
		Update("last_used_at", lastUsedAt).Error
}

// Revoke marks a device as revoked.
func (s *deviceStore) Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) error {
	return s.db.WithContext(ctx).
		Model(&Device{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", revokedAt).Error
}

// Delete removes a device and cascades to refresh tokens.
func (s *deviceStore) Delete(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First revoke all refresh tokens for this device
		if err := tx.Model(&RefreshToken{}).
			Where("device_id = ?", id).
			Updates(map[string]interface{}{
				"revoked_at": time.Now().UTC(),
				"reason":     "device deleted",
			}).Error; err != nil {
			return err
		}
		
		// Then delete the device
		return tx.Delete(&Device{}, id).Error
	})
}

// toDomainDevice converts a GORM model to domain model.
func toDomainDevice(dbDevice *Device) *domain.Device {
	return &domain.Device{
		ID:         dbDevice.ID,
		UserID:     dbDevice.UserID,
		DeviceName: dbDevice.DeviceName,
		CreatedAt:  dbDevice.CreatedAt,
		LastUsedAt: dbDevice.LastUsedAt,
		RevokedAt:  dbDevice.RevokedAt,
	}
}
package user

import (
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"gorm.io/gorm"
)

type DeviceRepository interface {
	Create(device *dbmodel.Device) error
	GetByID(id uuid.UUID) (*dbmodel.Device, error)
	GetByJWKThumbprint(thumbprint string) (*dbmodel.Device, error)
	GetByUserID(userID uint) ([]dbmodel.Device, error)
	UpdateLastUsed(id uuid.UUID, timestamp time.Time) error
	RevokeDevice(id uuid.UUID) error
	GetActiveByUserID(userID uint) ([]dbmodel.Device, error)
}

type deviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) DeviceRepository {
	return &deviceRepository{db: db}
}

func (r *deviceRepository) Create(device *dbmodel.Device) error {
	return r.db.Create(device).Error
}

func (r *deviceRepository) GetByID(id uuid.UUID) (*dbmodel.Device, error) {
	var device dbmodel.Device
	if err := r.db.Where("id = ? AND status = ?", id, dbmodel.DeviceStatusActive).First(&device).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *deviceRepository) GetByJWKThumbprint(thumbprint string) (*dbmodel.Device, error) {
	var device dbmodel.Device
	if err := r.db.Where("jwk_thumbprint = ?", thumbprint).First(&device).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *deviceRepository) GetByUserID(userID uint) ([]dbmodel.Device, error) {
	var devices []dbmodel.Device
	if err := r.db.Where("user_id = ?", userID).Find(&devices).Error; err != nil {
		return nil, err
	}
	return devices, nil
}

func (r *deviceRepository) UpdateLastUsed(id uuid.UUID, timestamp time.Time) error {
	return r.db.Model(&dbmodel.Device{}).Where("id = ?", id).Update("last_used_at", timestamp).Error
}

func (r *deviceRepository) RevokeDevice(id uuid.UUID) error {
	return r.db.Model(&dbmodel.Device{}).Where("id = ?", id).Updates(map[string]any{
		"status":     dbmodel.DeviceStatusRevoked,
		"revoked_at": "NOW()",
	}).Error
}

func (r *deviceRepository) GetActiveByUserID(userID uint) ([]dbmodel.Device, error) {
	var devices []dbmodel.Device
	if err := r.db.Where("user_id = ? AND status = ?", userID, dbmodel.DeviceStatusActive).Find(&devices).Error; err != nil {
		return nil, err
	}
	return devices, nil
}

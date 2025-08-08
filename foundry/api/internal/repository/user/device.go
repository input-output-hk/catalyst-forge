package user

import (
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"gorm.io/gorm"
)

type DeviceRepository interface {
	Create(device *dbmodel.Device) error
	GetByUserAndFingerprint(userID uint, fingerprint string) (*dbmodel.Device, error)
}

type deviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) DeviceRepository { return &deviceRepository{db: db} }

func (r *deviceRepository) Create(device *dbmodel.Device) error { return r.db.Create(device).Error }

func (r *deviceRepository) GetByUserAndFingerprint(userID uint, fingerprint string) (*dbmodel.Device, error) {
	var d dbmodel.Device
	tx := r.db.First(&d, "user_id = ? AND fingerprint = ?", userID, fingerprint)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &d, nil
}

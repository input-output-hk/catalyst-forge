package user

import (
	"time"

	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"gorm.io/gorm"
)

type DeviceSessionRepository interface {
	Create(sess *dbmodel.DeviceSession) error
	GetByDeviceCode(code string) (*dbmodel.DeviceSession, error)
	GetByUserCode(code string) (*dbmodel.DeviceSession, error)
	ApproveByUserCode(userCode string, userID uint) error
	DenyByUserCode(userCode string) error
	TouchPoll(id uint, at time.Time) error
	UpdateStatus(id uint, status string) error
	UpdateInterval(id uint, intervalSeconds int) error
	IncrementPollCount(id uint) error
}

type deviceSessionRepository struct {
	db *gorm.DB
}

func NewDeviceSessionRepository(db *gorm.DB) DeviceSessionRepository {
	return &deviceSessionRepository{db: db}
}

func (r *deviceSessionRepository) Create(sess *dbmodel.DeviceSession) error {
	return r.db.Create(sess).Error
}

func (r *deviceSessionRepository) GetByDeviceCode(code string) (*dbmodel.DeviceSession, error) {
	var s dbmodel.DeviceSession
	tx := r.db.First(&s, "device_code = ?", code)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &s, nil
}

func (r *deviceSessionRepository) GetByUserCode(code string) (*dbmodel.DeviceSession, error) {
	var s dbmodel.DeviceSession
	tx := r.db.First(&s, "user_code = ?", code)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &s, nil
}

func (r *deviceSessionRepository) ApproveByUserCode(userCode string, userID uint) error {
	return r.db.Model(&dbmodel.DeviceSession{}).
		Where("user_code = ?", userCode).
		Updates(map[string]interface{}{
			"status":           "approved",
			"approved_user_id": userID,
		}).Error
}

func (r *deviceSessionRepository) DenyByUserCode(userCode string) error {
	return r.db.Model(&dbmodel.DeviceSession{}).
		Where("user_code = ?", userCode).
		Update("status", "denied").Error
}

func (r *deviceSessionRepository) TouchPoll(id uint, at time.Time) error {
	return r.db.Model(&dbmodel.DeviceSession{}).Where("id = ?", id).Update("last_polled_at", at).Error
}

func (r *deviceSessionRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&dbmodel.DeviceSession{}).Where("id = ?", id).Update("status", status).Error
}

func (r *deviceSessionRepository) UpdateInterval(id uint, intervalSeconds int) error {
	return r.db.Model(&dbmodel.DeviceSession{}).Where("id = ?", id).Update("interval_seconds", intervalSeconds).Error
}

func (r *deviceSessionRepository) IncrementPollCount(id uint) error {
	return r.db.Model(&dbmodel.DeviceSession{}).Where("id = ?", id).UpdateColumn("poll_count", gorm.Expr("poll_count + 1")).Error
}

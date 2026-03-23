package audit

import (
	adm "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/audit"
	"gorm.io/gorm"
)

type LogRepository interface {
	Create(entry *adm.Log) error
}

type logRepository struct{ db *gorm.DB }

func NewLogRepository(db *gorm.DB) LogRepository { return &logRepository{db: db} }

func (r *logRepository) Create(entry *adm.Log) error { return r.db.Create(entry).Error }

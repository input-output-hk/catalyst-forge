package audit

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Log represents an audit event persisted for admin review
type Log struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	EventType     string         `gorm:"not null;index" json:"event_type"`
	ActorUserID   *uint          `gorm:"index" json:"actor_user_id,omitempty"`
	SubjectUserID *uint          `gorm:"index" json:"subject_user_id,omitempty"`
	RequestIP     string         `json:"request_ip"`
	UserAgent     string         `json:"user_agent"`
	Metadata      datatypes.JSON `json:"metadata"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Log) TableName() string { return "audit_logs" }

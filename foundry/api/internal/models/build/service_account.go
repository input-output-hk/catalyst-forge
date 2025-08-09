package build

import "time"

// ServiceAccount represents an automation identity used for server cert issuance and CI
type ServiceAccount struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:255" json:"name"`
	Status    string    `gorm:"size:32;default:active" json:"status"`
	SAVer     int       `gorm:"default:1" json:"sa_ver"`
	CreatedAt time.Time `json:"created_at"`
}

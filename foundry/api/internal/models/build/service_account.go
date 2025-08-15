// ============================================================================
// DEPRECATED: This model is part of the v1 schema and will be removed.
// Please use the new v2 models located in the subdirectories:
// - internal/models/repository/
// - internal/models/project/
// - internal/models/release/
// - internal/models/deployment/
// etc.
// DO NOT add new functionality to this file.
// ============================================================================

package build

import "time"

// ServiceAccount represents an automation identity used for server cert issuance and CI.
type ServiceAccount struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:255" json:"name"`
	Status    string    `gorm:"size:32;default:active" json:"status"`
	SAVer     int       `gorm:"default:1" json:"sa_ver"`
	CreatedAt time.Time `json:"created_at"`
}

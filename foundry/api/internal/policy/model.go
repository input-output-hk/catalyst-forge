package policy

import (
	"time"

	"gorm.io/datatypes"
)

// APIPolicy represents a dynamic authorization policy for an HTTP method/path.
//
// PathPattern supports three forms:
// - Exact: "/api/v1/things"
// - Prefix: "/api/v1/things/*" (matches subtree with segment boundary)
// - Regex: "/^\\/api\\/v1\\/things\\/[0-9]+$/" (leading and trailing slashes denote regex)
type APIPolicy struct {
	ID                 uint           `gorm:"primaryKey"`
	Method             string         `gorm:"size:8;index:idx_method_pattern,priority:1;not null"`
	PathPattern        string         `gorm:"size:512;index:idx_method_pattern,priority:2;not null"`
	RequireAuth        bool           `gorm:"not null;default:false"`
	RequireStepUp      bool           `gorm:"not null;default:false"`
	AllowedRoles       datatypes.JSON `gorm:"type:jsonb"`
	AllowedPermissions datatypes.JSON `gorm:"type:jsonb"`
	Priority           int            `gorm:"not null;default:0"`
	Enabled            bool           `gorm:"not null;default:true"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

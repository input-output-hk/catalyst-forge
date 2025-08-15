package policy

import "gorm.io/gorm"

// AutoMigrate creates/updates policy tables.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&APIPolicy{})
}

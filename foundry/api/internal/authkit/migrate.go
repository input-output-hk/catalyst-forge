package authkit

import (
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store/gormstore"
	repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"
	"gorm.io/gorm"
)

// Migrator returns a db.Migrator for AuthKit tables using GORM AutoMigrate.
func Migrator() repodb.Migrator {
	return func(db *gorm.DB) error {
		return gormstore.AutoMigrate(db)
	}
}

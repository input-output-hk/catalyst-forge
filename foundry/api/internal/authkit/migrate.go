package authkit

import (
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/store/gormstore"
	repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"
	"gorm.io/gorm"
)

// Migrator returns a db.Migrator for AuthKit tables using GORM AutoMigrate.
func Migrator() repodb.Migrator {
	return func(db *gorm.DB) error {
		return gormstore.AutoMigrate(db)
	}
}

package db

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrator is a function that performs database migrations.
type Migrator func(db *gorm.DB) error

// RunMigrations executes a series of migration functions in order.
//
// Migrations are run sequentially, and the process stops at the first
// error. This allows modules to register their own migrations while
// keeping them coordinated at the application level.
func RunMigrations(db *gorm.DB, migrators ...Migrator) error {
	for i, m := range migrators {
		if err := m(db); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}
	return nil
}

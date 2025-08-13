// Package db provides a centralized database helper for GORM-based applications.
//
// It offers connection management with configurable pooling and timeouts,
// transaction support via the unit-of-work pattern, a migrations registry,
// and testing utilities. The package is designed to keep domain models and
// repositories in feature modules while providing centralized database
// configuration and management.
//
// Example:
//
//	cfg := db.Config{
//	    DSN:              "postgres://user:pass@localhost/dbname",
//	    MaxOpenConns:     25,
//	    MaxIdleConns:     5,
//	    ConnMaxLifetime:  time.Hour,
//	    StatementTimeout: 30 * time.Second,
//	}
//
//	store, err := db.Open(context.Background(), cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close(store)
//
//	// Use store.WithTx for transactions
//	err = store.WithTx(ctx, func(ctx context.Context) error {
//	    // All operations in this function share the same transaction
//	    return nil
//	})
package db

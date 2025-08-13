package db_test

import (
	"context"
	"fmt"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/db"
)

func ExampleOpen() {
	// This example shows how to open a database connection.
	// In real usage, provide a valid DSN for your database.
	
	ctx := context.Background()

	cfg := db.Config{
		DSN:              "postgres://user:pass@localhost/testdb",
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Hour,
		StatementTimeout: 30 * time.Second,
	}

	// In a real application:
	// store, err := db.Open(ctx, cfg)
	// if err != nil {
	//     log.Fatal(err)
	// }
	// defer db.Close(store)

	_ = ctx
	_ = cfg
	fmt.Println("See code comments for usage")
	// Output: See code comments for usage
}

func ExampleStore_WithTx() {
	// This example shows how to use transactions with Store.WithTx.
	// In real usage, store would be initialized with db.Open.
	
	// In a real application:
	// store, _ := db.Open(ctx, cfg)
	// defer db.Close(store)
	//
	// err := store.WithTx(ctx, func(txCtx context.Context) error {
	//     // All database operations here share the same transaction
	//     // The transaction handle can be retrieved with:
	//     // tx := db.TxFromContext(txCtx)
	//     
	//     // If this returns an error, the transaction rolls back
	//     // If it returns nil, the transaction commits
	//     return nil
	// })
	//
	// if err != nil {
	//     log.Printf("Transaction failed: %v", err)
	// }
	
	fmt.Println("See code comments for usage")
	// Output: See code comments for usage
}

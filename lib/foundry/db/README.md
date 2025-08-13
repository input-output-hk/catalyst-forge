# lib/foundry/db

A centralized database helper package providing connection management, transactions, migrations, and testing utilities for GORM-based applications.

## Features

- **Connection Management**: Configurable connection pooling, timeouts, and lifecycle management
- **Transaction Support**: Unit-of-work pattern with context-aware transaction propagation
- **Migrations Registry**: Centralized migration coordination across modules
- **Testing Utilities**: Mock and real database support for testing

## Usage

### Basic Setup

```go
import (
    "context"
    "time"
    "github.com/catalystgo/catalyst-forge/lib/foundry/db"
)

cfg := db.Config{
    // Include statement timeout in DSN for PostgreSQL
    DSN:              "postgres://user:pass@localhost/dbname?options=-c statement_timeout=30000",
    MaxOpenConns:     25,
    MaxIdleConns:     5,
    ConnMaxLifetime:  time.Hour,
    SlowThreshold:    200 * time.Millisecond, // Log slow queries
}

store, err := db.Open(context.Background(), cfg)
if err != nil {
    log.Fatal(err)
}
defer db.Close(store)
```

### Transactions (Unit of Work)

```go
// Basic transaction
err := store.WithTx(ctx, func(ctx context.Context) error {
    // All repository calls within this function will use the same transaction
    if err := userRepo.Create(ctx, user); err != nil {
        return err // Transaction will rollback
    }
    
    if err := auditRepo.Log(ctx, "user_created"); err != nil {
        return err // Transaction will rollback
    }
    
    return nil // Transaction will commit
})

// Transaction with custom options (if supported)
err = db.WithTxOptions(ctx, store, &sql.TxOptions{
    Isolation: sql.LevelSerializable,
    ReadOnly:  false,
}, func(ctx context.Context) error {
    // Transaction with serializable isolation level
    return nil
})
```

### Repository Integration

Repositories can be made transaction-aware by checking for an active transaction in the context:

```go
import repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"

type UserRepo struct {
    db *gorm.DB
}

func (r *UserRepo) dbFor(ctx context.Context) *gorm.DB {
    if tx := repodb.TxFromContext(ctx); tx != nil {
        return tx // Use transaction if available
    }
    return r.db // Use default connection
}

func (r *UserRepo) Create(ctx context.Context, user *User) error {
    return r.dbFor(ctx).WithContext(ctx).Create(user).Error
}
```

### Migrations

```go
// Define migrations in your modules
func UserMigrations(db *gorm.DB) error {
    return db.AutoMigrate(&User{}, &Role{})
}

// Run all migrations at startup
err := db.RunMigrations(store.Write(), 
    UserMigrations,
    AuditMigrations,
    // ... other module migrations
)
```

## Configuration

| Field | Description | Default | Notes |
| `DSN` | Database connection string | Required | Include timeouts here for PostgreSQL |
| `MaxOpenConns` | Maximum open connections | 0 (unlimited) |
| `MaxIdleConns` | Maximum idle connections | 0 |
| `ConnMaxLifetime` | Maximum connection lifetime | 0 (unlimited) |
| `ConnMaxIdleTime` | Maximum idle time | 0 (unlimited) |
| `StatementTimeout` | (Deprecated) Use DSN options instead | 0 | Not used; configure in DSN |
| `LogLevel` | GORM log level | Silent |
| `SlowThreshold` | Slow query threshold | 0 | Logs queries slower than this |

## Testing

See the `dbtest` package for testing utilities including mock database support and Testcontainers integration.

## Architecture

This package follows the repository pattern with transaction support via context propagation. It's designed to:

1. Keep domain models and repositories in feature modules
2. Provide centralized database configuration and management
3. Enable transparent transaction handling without changing repository interfaces
4. Support both unit and integration testing
5. Allow future extensions for read replicas without breaking changes

### Design Decisions

- **Statement Timeout**: Configure in DSN to ensure it applies to all pooled connections
- **Read/Write Separation**: Same handle now, but interface allows future read replica support
- **Transaction Options**: Available via `WithTxOptions` for custom isolation levels

For more details on the overall database architecture, see [.ai/api/DB.md](/.ai/api/DB.md).
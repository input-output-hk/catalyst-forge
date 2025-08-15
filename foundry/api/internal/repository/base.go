package repository

import (
	"context"

	"gorm.io/gorm"
)

// Pagination contains pagination request parameters
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// PageResult contains paginated results
type PageResult struct {
	Items    interface{} `json:"items"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Total    int64       `json:"total"`
}

// SortOrder defines sort direction
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// Sort defines sorting parameters
type Sort struct {
	Field string    `json:"field"`
	Order SortOrder `json:"order"`
}

// TxManager manages database transactions
type TxManager interface {
	// InTx executes a function within a transaction
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
	// DB returns the database instance for the current context
	DB(ctx context.Context) *gorm.DB
}

// txManager implementation
type txManager struct {
	db *gorm.DB
}

// NewTxManager creates a new transaction manager
func NewTxManager(db *gorm.DB) TxManager {
	return &txManager{db: db}
}

// InTx executes a function within a transaction
func (tm *txManager) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.db.Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, txKey{}, tx)
		return fn(ctx)
	})
}

// DB returns the database instance for the current context
func (tm *txManager) DB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return tm.db
}

// txKey is the context key for transaction
type txKey struct{}

// GetDB extracts the database from context or returns the default
func GetDB(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return defaultDB
}

// ApplyPagination applies pagination to a query
func ApplyPagination(query *gorm.DB, pagination *Pagination) *gorm.DB {
	if pagination == nil {
		return query
	}

	page := pagination.Page
	if page <= 0 {
		page = 1
	}

	pageSize := pagination.PageSize
	if pageSize <= 0 {
		pageSize = 20
	} else if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize
	return query.Offset(offset).Limit(pageSize)
}

// ApplySort applies sorting to a query
func ApplySort(query *gorm.DB, sort *Sort, defaultSort string) *gorm.DB {
	if sort == nil || sort.Field == "" {
		if defaultSort != "" {
			return query.Order(defaultSort)
		}
		return query
	}

	order := string(sort.Order)
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	return query.Order(sort.Field + " " + order)
}
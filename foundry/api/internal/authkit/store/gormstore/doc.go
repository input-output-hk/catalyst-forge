package gormstore

import "errors"

// Provide minimal error stubs to avoid importing gorm/sqlmock helpers into multiple tests.
var (
	errRecordNotFound = errors.New("record not found")
	anyError          = errors.New("any error")
)

// Package gormstore provides GORM-based implementations of all storage interfaces.
//
// These implementations are suitable for production use with PostgreSQL, MySQL, SQLite,
// and other databases supported by GORM. All models include proper indexes and constraints
// for optimal performance and data integrity.

// Package store defines the storage interfaces for the authentication system.
//
// These interfaces abstract the persistence layer, allowing for different
// implementations (in-memory, GORM, Redis, etc.) without changing the core
// authentication logic. All interfaces are designed to be testable and
// support both transactional and non-transactional operations.
package store
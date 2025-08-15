// Package authkit provides a complete passwordless authentication system using WebAuthn.
//
// The system implements device-based authentication without passwords or OAuth providers,
// using cryptographically secure invite links for onboarding and WebAuthn/Passkeys for
// all authentication. It includes support for session management with JWT access tokens
// and HttpOnly refresh tokens, step-up authentication for critical operations, and
// account recovery through single-use recovery codes.
//
// Key features:
//   - Passwordless authentication via WebAuthn/Passkeys
//   - No external OAuth dependencies
//   - Hardware security key enforcement for administrators
//   - JWT access tokens with refresh rotation
//   - Session versioning for immediate invalidation
//   - Step-up authentication for sensitive operations
//   - Rate limiting without IP addresses
//   - Comprehensive audit logging
//
// The package is designed with clean architecture principles, separating domain entities,
// storage interfaces, and service implementations. All auth logic is encapsulated within
// the package, requiring no changes to application handlers.
package authkit
// Package httpkit provides generic HTTP utilities that are not tied to
// authentication or any domain logic. It includes primitives for:
// - CSRF strategies (header-only, double-submit, memory, no-op)
// - Secure cookie helpers for CSRF tokens
// - JSON parsing, content-type validation, and security headers
// - CORS configuration and preflight handling
// - Request IDs, simple middleware composition, and recovery/timeout wrappers
// - Consistent JSON responses and a generic HTTPError type
//
// This package is designed to be imported by higher-level modules (e.g., authkit)
// without creating circular dependencies. Domain-specific codes or cookies should
// remain in their respective packages and may wrap the types provided here.
package httpkit


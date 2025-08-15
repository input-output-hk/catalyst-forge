// Package httpkit in the authkit module provides auth-domain utilities:
// - Auth-specific error codes and thin constructors wrapping lib/foundry/httpkit
// - Refresh/bootstrap cookie helpers for auth flows
//
// Generic HTTP primitives (CSRF, JSON parsing, CORS, security headers, generic
// errors/responses, middleware) live in the standalone module
// github.com/catalystgo/catalyst-forge/lib/foundry/httpkit to avoid circular
// dependencies and enable reuse across the repo.
package httpkit

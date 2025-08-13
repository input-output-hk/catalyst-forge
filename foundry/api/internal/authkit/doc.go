package authkit

// Package authkit wires the AuthKit library into the API.
//
// Responsibilities:
//   - Build AuthKit Config from API config.
//   - Construct Deps (stores, key manager, CSRF, limiter, clock, logger).
//   - Optionally run migrations for AuthKit GORM stores.
//   - Mount all /api/v1/auth endpoints and JWKS, and apply middlewares/policies.
//
// This package contains no domain logic; it only adapts API configuration and
// routing to the AuthKit library.

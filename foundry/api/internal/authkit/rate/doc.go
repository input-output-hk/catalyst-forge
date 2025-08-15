// Package rate provides rate limiting interfaces and utilities for the authentication system.
//
// Rate limiting is identity-based rather than IP-based, using email addresses,
// user IDs, and invite IDs as rate limit keys. This approach works well with
// load balancers that obscure client IPs.
package rate
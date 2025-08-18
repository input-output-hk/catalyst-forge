# internal/auth

Centralized application-specific authorization configuration.

This package hosts domain wiring for:
- permissions
- resource types
- scope chains and planner rules
- policy registry (route → permission)
- resource resolvers
- domain-specific conditions

The generic RBAC engine remains in `internal/authkit/rbac`.



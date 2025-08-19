## AuthKit package context (for LLM agents)

Purpose: High-signal map of the `authkit` package (and embedded `rbac`) to help agents quickly understand architecture, main entry points, flows, and security invariants.

### What this package is
- **Passwordless auth system** using WebAuthn/Passkeys (no OAuth).
- **Session model**: JWT access tokens + HttpOnly `__Host-refresh_token` with rotation and replay detection.
- **Step-up auth**: short window of elevated assurance for sensitive operations.
- **CSRF protection** for refresh flow via double submit tokens.
- **Audit-ready**: structured events around refresh, logout, rotation.
- **Embedded RBAC engine**: policy registry for path/method rules + resource-scoped permission checks.

### Layout overview
- `authkit/` (public API)
  - `config.go`: `Config` and defaults (RP, tokens, cookies, GitHub OIDC, step-up).
  - `deps.go`: dependency bundle `Deps` (stores, key manager, rand, CSRF, logger, KV, limiter).
  - `manager.go` + `manager_impl.go`: `Manager` facade and wiring.
  - `policy.go`: path policy registry and rule merging logic.
  - `routes.go`: canonical handlers for `/auth/*` routes and helper `Handlers` struct.
  - `context.go`: `AuthContext` carried in `gin.Context`.
  - `errors.go`: common error constants.
- `crypto/`: secure rand, JWT key manager (ES256), helpers.
- `domain/`: entity models (`User`, `Credential`, `Invite`, `RefreshToken`, `Device`, ...).
- `service/`: core services: WebAuthn ceremonies, tokens, refresh rotation, recovery, step-up, device-link.
- `store/`: interfaces and concrete stores (`gormstore`, `memstore`) for users, creds, refresh, invites, audit, etc.
- `httpkit/`: cookie helpers for `__Host-*` cookies and access/refresh management.
- `middleware/`: optional Gin middleware equivalents of the public `Manager` methods.
- `rbac/`: embedded role/permission engine (see below).
- `config.go` (package root): adapter from API config → `authkit.Config`.
- `deps.go` (package root): adapter to build `Deps` from infra (DB, key manager, logger, etc.).

### Public entry points
- `authkit.New(cfg Config, deps Deps) (Manager, error)` creates the system.
- `Manager` methods:
  - `RegisterRoutes(rg *gin.RouterGroup)` mounts `/auth/*` endpoints.
  - `RegisterJWKS(rg *gin.RouterGroup)` mounts `/.well-known/jwks.json`.
  - `Authenticate()` non-blocking authn (injects `AuthContext` if token valid).
  - `EnforcePolicies(provider PolicyProvider)` global policy enforcer.
  - `RequireAuth()` and `RequireStepUp()` per-group middleware.
  - `Handlers()` returns closures for apps wishing to mount routes themselves.

Minimal integration sketch
```go
cfg := authkit.DefaultConfig() // or BuildConfig(appCfg)
deps := authkit.Deps{ /* provide Stores, Keys, Rand, CSRF, KV, Logger */ }
m, _ := authkit.New(cfg, deps)

api := r.Group("/auth")
m.RegisterRoutes(api)

wellKnown := r.Group("/.well-known")
m.RegisterJWKS(wellKnown)

r.Use(m.Authenticate())            // non-blocking; injects AuthContext
r.Use(m.EnforcePolicies(registry)) // global policy
```

### Routes in `RegisterRoutes`
- Auth (browser and API):
  - `POST /auth/login/begin` → WebAuthn options
  - `POST /auth/login/complete` → issues refresh cookie + access token
  - `POST /auth/refresh` → CSRF-validated refresh rotation, returns access token
  - `POST /auth/logout` → revoke current refresh family, clear cookie
  - `POST /auth/logout-all` → revoke all refresh tokens for user (bumps session)
  - `GET  /auth/me` → id/email/roles
  - `GET  /auth/session` → validity, session version, step-up state
- Credentials:
  - `GET    /auth/credentials` → list
  - `POST   /auth/credentials/add/begin` → WebAuthn create options
  - `POST   /auth/credentials/add/complete`
  - `DELETE /auth/credentials/:id`
- Invites & onboarding:
  - `POST /auth/invites` → create invite (returns token)
  - `POST /auth/onboard/begin` → validate invite and return registration options
  - `POST /auth/onboard/complete` → redeem invite
- Recovery:
  - `POST /auth/recovery/init` → start recovery by email
  - `POST /auth/recovery/verify` → verify recovery code
  - `POST /auth/recovery/register/begin`
  - `POST /auth/recovery/register/complete`

### Tokens, cookies, sessions
- Access tokens: JWT with claims `sub`, `email`, `roles`, `permissions`, `sv`, `jti`, `iss`, `aud`, `iat`, `exp`, optional `su` (step-up until), `amr`, `device_id`.
- Refresh: HttpOnly cookie `__Host-refresh_token`; server stores SHA-256 of raw bytes; rotation creates new token and marks previous as rotated; replay detection revokes the family.
- Access cookie: `__Host-access_token` optionally set for browser UX; bearer token also supported.
- Session invalidation: bump `User.SessionVersion` → all access tokens invalid; refresh rotation enforces version consistency.
- CSRF: double-submit token validated on `/auth/refresh`.

Security invariants
- `__Host-*` cookies must be `Secure=true`, `Domain=""`, `Path="/"` (enforced in `httpkit`).
- Admins cannot bypass step-up: step-up is checked before roles/permissions.
- Admin role bypasses only RBAC role/permission checks in policy middleware, not step-up.
- Issuer and audience of tokens must match `cfg.Origin`.

### WebAuthn service (`service/webauthn.go`)
- Username-less login via discoverable credentials.
- Registration requires Resident Keys; hardware key enforcement for admins only if allowlist is configured (`AdminAAGUIDAllowlist`).
- Safari/iCloud passkey edge-cases handled (userHandle omissions, backup-eligible discrepancies).
- Step-up flows for sensitive endpoints return short-lived assurance window (`StepUpTTL`).

### Refresh service (rotation & replay)
- `Issue`: creates family and first token; audits event.
- `Rotate`: detects replays (if token already rotated, revokes family), checks expiry and session version, issues new access token and refresh cookie; audits rotation.
- `RevokeCurrent` and `RevokeAllUser`: for logout and global logout; audit and version bump.

### Auth context (`authkit/context.go`)
- Stored under key `auth_context` in `gin.Context`.
- Fields: `UserID`, `Email`, `FullName`, `Roles`, `Permissions`, `SessionVersion`, `StepUpValidUntil`, `TokenID`, `AMR`, `DeviceID`.
- Helpers: `IsAuthenticated`, `HasRole`, `HasAnyRole`, `HasPermission`, `HasAnyPermission`, `RequiresStepUp`.

### Policy registry (`authkit/policy.go`)
- Register path/method rules with: `RequireAuth`, `RequireStepUp`, `RequireRoles`, `RequirePermissions`, `AllowAnonymous`.
- Matching supports exact, `/*` prefix, and `/regex/` patterns.
- `MergeRules` keeps most restrictive: OR for auth/step-up, INTERSECT for roles/permissions.

### Middleware options
- Via `Manager`:
  - `Authenticate()` → optional auth; injects `AuthContext`.
  - `RequireAuth()` / `RequireStepUp()` per-route-groups.
  - `EnforcePolicies(provider)` uses registry and optional RBAC provider.
- Via `middleware/` package: `Authenticator`, `RequireAuth`, `RequireStepUp`, `PolicyEnforcer` utilities mirror above.

### Configuration
- `authkit.Config` controls WebAuthn (RP name/ID/origin, challenge TTL, `RequireUV`), token TTLs, cookie names/flags, step-up window, bootstrap token, GitHub OIDC, rate limiting, JWKS exposure.
- `services/api/internal/authkit/config.go` adapts the API’s app config into `authkit.Config` (sets SameSite, Origin from `PublicBaseURL`, TTLs, etc.).
- `services/api/internal/authkit/deps.go` builds concrete `Deps` using GORM stores by default, `StrictES256KeyManager`, CSRF setup, in-memory KV.

### Stores and persistence
- Interfaces under `store/` for: users, credentials, invites, refresh, recovery codes, access requests, audit, bootstrap, GitHub policies, device links/devices, KV, challenges.
- Implementations:
  - `store/gormstore`: persistent stores (users, credentials, invites, refresh, audit, bootstrap, etc.).
  - `store/memstore`: KV and challenge stores for ephemeral data.

### RBAC subpackage at a glance (`authkit/rbac`)
- Purpose: Resource-scoped permission checks with role definitions and bindings, plus pluggable condition evaluators.
- Key types:
  - `PermissionKey`, `Effect` (Allow/Deny), `RoleDef` (entries with optional conditions), `Binding` (subject→role at scope), `Decision`, `Trace`.
  - `Subject` (user/group/service), `ResourceRef` (type/id/org/parent/attrs).
  - Scopes: `global` → `org` → `project` → `environment` → `resource` (most-specific).
- Evaluation:
  - Denies are applied across scopes first; then allows in scope-planner order.
  - Conditions include: `requires_step_up`, `attr_equals`, `org_matches`, `time_window` (extensible via registry).
  - Planner (`planner.go`) maps resource type to scope chain; default preserves resource→project→org→global.
  - Cache (`MemoryCache`) for role entries and principal views with TTLs.
- Store:
  - Interface in `rbac/store.go`; GORM implementation in `rbac/gormstore` with models and simple migrations.
- Manager:
  - `rbac.New(cfg, deps)` returns `Manager` with `Check`, `Explain`, `RegisterResolver`, `Resolve`.
  - Integrates with policy middleware by implementing `PolicyProvider` tie-in and exact-path resource resolvers.
- Admin API and Gin middleware files are placeholders for future phases.

### JWKS and keys
- `Manager.RegisterJWKS` exposes `/.well-known/jwks.json` using the configured `KeyManager`.
- Use `StrictES256KeyManager` by default; supports adding keys via PEM and selecting current KID; prevents key ID reuse.

### GitHub OIDC (optional)
- Config fields under `Config.GitHub` allow verifying GH Actions OIDC tokens and exchanging for platform tokens; services/stores provide policy support (see `store/githubpolicies.go`).

### Typical app wiring (API integration)
```go
// Build using adapters provided in this repo
cfg := authkit.BuildConfig(appCfg)
deps := authkit.BuildDeps(ctx, appCfg, dbStore, gdb, nil, authkit.NewLogger(slog.Default()))
m, _ := authkit.New(cfg, deps)

r := gin.Default()
r.Use(m.Authenticate())
r.Use(m.EnforcePolicies(myRegistry))
m.RegisterRoutes(r.Group("/auth"))
if cfg.JWKSRoute { m.RegisterJWKS(r.Group("/.well-known")) }
```

### Gotchas and best practices
- Always enforce `__Host-*` cookie constraints (already enforced by helpers).
- Ensure `cfg.Origin` matches the public base URL; issuer/audience validation depends on it.
- Use `RequireStepUp` or policy rules for operations like credential management, destructive profile actions, and admin-only flows.
- Bump session version after bulk revocations to invalidate outstanding access tokens.
- Provide a persistent CSRF secret in production to avoid rotating CSRF cookies on restart.
- Admin hardware key enforcement only applies when an allowlist is configured; otherwise, passkeys in general are accepted.

### Where to look for specifics
- Public API: `authkit/manager.go`, `authkit/routes.go`, `authkit/policy.go`, `authkit/context.go`.
- Services: `service/webauthn.go`, `service/refresh.go`, `service/tokens.go`.
- Stores: `store/*` (interfaces), `store/gormstore/*`, `store/memstore/*`.
- Cookies & CSRF: `httpkit/cookies.go`, `lib/foundry/httpkit` for CSRF.
- RBAC: `rbac/*` (evaluator, planner, scopes, conditions, gormstore, resources).
- API adapters: package-root `config.go`, `deps.go`.

### Status
- RBAC: core evaluator, planner, store, and cache exist; Gin middleware and admin APIs are placeholders.
- Invites/recovery/credential flows are implemented; some routes note “add more flows” for future expansion.

This document is intentionally high-signal. For deeper prose documentation, check `docs/` and `.ai/*` files if present.



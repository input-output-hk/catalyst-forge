### AuthKit RBAC Integration Plan

This plan adds a generic, multi-tenant RBAC module to the existing `authkit` package, as specified in `.ai/api/PermissionsPkg.md` and aligned with `.ai/api/AuthPkg.md`.

The goal is to provide route-level authorization via `authkit.PolicyRegistry` that transparently consults RBAC for permission checks without modifying application handlers.

---

### Objectives

- Integrate an RBAC system into `lib/foundry/authkit` that is:
  - Generic: no hardcoded actions/entities; apps define permission keys.
  - Scoped: global → org → project → resource with inheritance.
  - Deterministic: explicit deny overrides allow (configurable), default deny.
  - Fast: versioned caches for roles and principals.
  - Auditable: decision tracing and audit events on admin changes.
  - Seamless: plugged into `PolicyRegistry` and middleware.

---

### Current State (Summary)

- `authkit` exists with: `PolicyRegistry`, `Authenticate`, `EnforcePolicies`, `AuthContext`, routes and services (tokens, refresh, step-up, recovery), middleware, stores.
- `PolicyEnforcer` currently checks `AuthContext` for roles/permissions (`HasAnyRole`, `HasAnyPermission`). There is no RBAC engine, resolver, or RBAC storage.

Implication: we will introduce a standalone `rbac` submodule and update policy enforcement to consult RBAC for permission checks. Because the service is not live, we will replace legacy user-role code with RBAC bindings rather than bridging it. Role checks remain short-term for coarse gates but should be phased out or backed by RBAC data.

---

### Deliverables

- New `rbac` package with API, evaluator, condition registry, caches, and stores (in-memory + GORM models).
- Bridge from `PolicyRegistry.RequirePermissions` to RBAC at middleware enforcement time, and replacement of legacy role membership handlers with RBAC bindings.
- Optional admin HTTP endpoints for managing roles/bindings.
- Tests: unit (evaluator, conditions, caching), in-memory end-to-end, and HTTP integration with `PolicyRegistry`.
- Minimal SQL/GORM migrations for RBAC tables.

---

### High-Level Architecture

- `authkit/rbac` package exposed via a `Manager` that:
  - Installs with `PolicyRegistry` so permission checks consult RBAC
  - Registers path-pattern-based `ResourceResolver` functions
  - Exposes programmatic checks (`Check`, `Explain`)
  - Optionally mounts admin routes under `/auth/rbac`

- `PolicyEnforcer` (middleware) will:
  - Evaluate `RequireAuth` and `RequireStepUp` using `AuthContext`
  - Evaluate `RequireRoles` using `AuthContext`
  - Evaluate `RequirePermissions` by calling RBAC with a `Subject` and a `ResourceRef` derived from the request via a registered resolver (or resource-agnostic if none).

---

### Work Breakdown Structure (Phased)

- Phase 0 — Package Skeleton & Configuration
  - Create `lib/foundry/authkit/rbac/` folder with files:
    - `rbac.go` (public facade, manager wiring)
    - `config.go` (module configuration)
    - `policy.go` (compiler, condition registry, evaluator)
    - `context.go` (subject/resource types, eval context)
    - `store.go` (DB-agnostic interfaces)
    - `cache.go` (role + principal caches)
    - `middleware.go` (integration helpers for `PolicyRegistry`)
    - `admin_api.go` (optional admin endpoints; can be toggled)
    - `gormstore/` (models + CRUD impl)
    - `testing/` (in-memory store + evaluation harness)
  - Define `rbac.Config` and `rbac.Deps` (Store, Clock, Logger, optional Cache, AuditStore).
  - Add package `doc.go` and README within the folder to document usage.

- Phase 1 — Domain & Storage Interfaces
  - Implement RBAC data types: `PermissionKey`, `Effect`, `Condition`, `RoleDef`, `RoleEntry`, `Binding`, `Decision`, `Trace`.
  - Define `Store` interface for roles, bindings, and per-subject principal versions.
  - Provide in-memory store in `rbac/testing` to unblock higher layers and tests.
  - Implement GORM models in `rbac/gormstore/models.go` and CRUD scaffolding (`roles.go`, `bindings.go`).
  - Add initial migrations (compatible with AutoMigrate) for:
    - `rbac_roles`, `rbac_role_entries`, `rbac_bindings`, `rbac_principal_versions`.

- Phase 2 — Evaluator & Conditions
  - Implement evaluation engine:
    - Gather subject bindings across scopes: global → org → project → resource
    - Expand to role entries (compiled from roles)
    - Apply precedence: explicit denies then allows
    - Evaluate ANDed conditions on winning entries
    - Default deny
  - Condition registry with built-ins:
    - `requires_step_up` (checks `AuthContext.StepUpValidUntil` vs `now` and window param)
    - `attr_equals` (subject/resource attribute comparison)
    - `org_matches` (tenancy safety)
    - `time_window` (business-hours windows)
  - Expose `RegisterCondition()` for custom evaluators.
  - Implement `Explain()` to return `Trace` and per-condition pass/fail details.

- Phase 3 — Caching & Versioning
  - Role cache: keyed by `role_slug + role.Version`; value: compiled entries.
  - Principal cache: keyed by `(subject) + principalVersion + optional org`.
  - Ensure store methods bump `role.Version` on role changes and increment principal version on binding changes.
  - Add TTLs in `rbac.Config` (default in-memory cache used if unspecified).

- Phase 4 — PolicyRegistry Bridge & Middleware Integration
  - Extend `authkit.PolicyRegistry` introspection to return matched patterns alongside rules (non-breaking helper):
    - Add `GetMatches(method, path) []Match` where `Match{Pattern string, Rule Rule}`.
    - Keep `GetRules` for backward compatibility.
  - RBAC Manager:
    - `WithPolicyRegistry(reg *authkit.PolicyRegistry) *authkit.PolicyRegistry` returns the same registry (for fluent wiring).
    - `RegisterResolver(pattern string, resolver ResourceResolver)` maps path patterns to a resolver that produces `ResourceRef` from `*gin.Context`.
  - Update `middleware.PolicyEnforcer.EnforcePolicies`:
    - If `RequirePermissions` is present, resolve the subject from `AuthContext` and resource via resolver if registered for the matched pattern. For each permission, call `rbac.Check`.
    - Map condition failures to 428 `step_up_required` where applicable; otherwise 403.
    - Maintain role checks (`RequireRoles`) via `AuthContext` to avoid regression.
    - Maintain behavior when no RBAC manager is installed: fallback to `AuthContext.HasAnyPermission` for backward compatibility (feature flag via `rbac.Manager` presence).

- Phase 5 — Admin & Maintenance APIs (Optional to Enable)
  - Mount under `/auth/rbac` (e.g., `POST /roles`, `PUT /roles/:slug`, `GET /roles`, `GET /roles/:slug`, `POST /bindings`, `DELETE /bindings/:id`, `GET /bindings`, `POST /explain`).
  - Protect endpoints using permissions defined by the application (e.g., `"rbac:admin"`). Do not hardcode.
  - Emit audit events via `store.AuditStore` on role and binding mutations.

- Phase 6 — Testing
  - Evaluator unit tests: deny-overrides-allow, scope precedence, condition pass/fail.
  - Condition tests for each built-in evaluator.
  - Cache invalidation tests (role and principal version changes reflect in decisions).
  - In-memory store end-to-end tests for `Check`/`Explain`.
  - HTTP integration tests: `PolicyRegistry` + resolvers + middleware enforcing permissions without touching handlers.
  - Golden tests for `Trace` output consistency.

- Phase 7 — Documentation & Examples
  - Add RBAC README with minimal boot integration snippet.
  - Document resolver registration patterns and multi-tenant safety (`org_matches`).
  - Update `FEEDBACK.md` or docs to reflect RBAC usage and migration steps.

---

### Detailed Tasks

- rbac package API
  - `Config{ SuperRoles []string; EnableDeny bool; RoleCacheTTL; PrincipalCacheTTL }`
  - `Deps{ Store Store; Clock authkit.Clock; Logger authkit.Logger; Audit store.AuditStore; Cache Cache }`
  - `Manager` interface:
    - `WithPolicyRegistry(reg *authkit.PolicyRegistry) *authkit.PolicyRegistry`
    - `RegisterResolver(pattern string, resolver ResourceResolver)`
    - `Check(ctx context.Context, subj Subject, action PermissionKey, res ResourceRef) (Decision, error)`
    - `Explain(ctx context.Context, subj Subject, action PermissionKey, res ResourceRef) (Decision, Trace, error)`
    - `RegisterAdminRoutes(rg *gin.RouterGroup)` (optional)

- Types
  - `Subject{ Type SubjectType; ID string; OrgID *uuid.UUID; Attrs map[string]any }`
  - `ResourceRef{ Type string; ID string; OrgID *uuid.UUID; Parent *ResourceRef; Attrs map[string]any }`
  - `ResourceResolver func(c *gin.Context) (ResourceRef, error)`

- Store
  - `CreateRole`, `GetRole`, `UpdateRole`, `ListRoles`, `BumpRoleVersion`
  - `AddBinding`, `RemoveBinding`, `ListBindings`, `ListBindingsByScope`
  - `GetPrincipalVersion`, `BumpPrincipalVersion`

- GORM Models
  - Tables: roles, role entries, bindings, principal versions; indexes as per spec.

- Evaluator
  - `Evaluate(subj, perm, res)` using compiled role entries and conditions.
  - Step-up enforcement is a condition (`requires_step_up`), not hardcoded.

- PolicyEnforcer changes
  - Add RBAC-aware permission path in `EnforcePolicies`:
    - When `len(rule.Permissions) > 0` and RBAC manager is present, iterate permissions and call `rbac.Check` with subject + resource.
    - Use `authkit.PolicyRegistry.GetMatches` to retrieve the concrete matched pattern to find the registered resolver.
    - On condition failures that indicate step-up, respond with 428 and standardized JSON error `{ "error": "step_up_required" }`.
    - Otherwise 403 for forbidden.
  - Keep existing behavior when RBAC manager is absent for compatibility during rollout.

---

### Integration Points & Wiring

- Application boot example:
  - Create RBAC manager via `rbac.New(cfg, deps)`
  - Build `PolicyRegistry` with `RequireAuth/RequireStepUp/RequirePermissions`
  - Register resource resolvers for permissioned routes:
    - `rbacMgr.RegisterResolver("/api/v1/infra/deploy/:projectID", resolver)`
  - Wrap registry: `auth.Use(auth.EnforcePolicies(rbacMgr.WithPolicyRegistry(reg)))`

- `AuthContext` remains a consumer-only facility for handlers and for role checks; permission checks move to RBAC.

---

### Migration (Replace legacy with RBAC)

- Since the service is not live, we will replace the legacy user-role code rather than bridge it.
- Action items:
  - Define RBAC `RoleDef` slugs that correspond to existing conceptual roles.
  - Implement a one-time migration/seed to create roles and convert any existing user↔role relationships into RBAC `Binding` records at `ScopeGlobal` (or appropriate scope) using user UUIDs.
  - Update endpoints under `foundry/api/internal/api/handlers/user` that manage user roles to call the RBAC store (`AddBinding`, `RemoveBinding`, `ListBindings`) or remove them if superseded by RBAC admin APIs.
  - Ensure middleware enforces permissions via RBAC; retire `AuthContext.Permissions` mutation paths.

---

### Security Notes

- Default deny; optional explicit deny overrides allow (configurable, default on).
- Prefer populating `OrgID` for both `Subject` and `ResourceRef` and include an `org_matches` condition by default in org-scoped roles.
- Step-up is enforced through conditions, not special cases in the enforcer.
- Audit all admin role/binding changes via `store.AuditStore`.

---

### Acceptance Criteria

- RBAC package compiles and exposes the documented API.
- Middleware enforces `RequirePermissions` via RBAC with resolvers; returns 428 for step-up condition failures and 403 otherwise.
- In-memory store passes evaluator and integration tests.
- GORM models migrate and CRUD works against a test DB.
- Role and principal cache invalidation verified by tests.
- Optional admin routes gated behind app-defined permissions.

---

### Risks & Mitigations

- Resolver/pattern sync with `PolicyRegistry`:
  - Mitigation: add `GetMatches` or attach pattern to rule matching; document usage.
- Performance under high cardinality bindings:
  - Mitigation: compiled role caches + principal caches, versioned invalidation.
- Race conditions on binding changes:
  - Mitigation: atomic version bumps and cache keying by versions.

---

### Timeline (Rough)

- Week 1: Phases 0–2 (skeleton, interfaces, evaluator, base conditions, in-memory store)
- Week 2: Phase 3–4 (caching, registry bridge, middleware wiring, initial tests)
- Week 3: Phase 5–6 (admin APIs, GORM adapter, full test matrix) + docs

---

### Open Questions

- Do we need group subjects in v1? If not, leave hooks but defer implementation.
- Should we surface `principalVersion` in access tokens for proactive client sync? Optional; can be added later.



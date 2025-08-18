### RBAC for AuthKit

This package provides a generic, fast, and auditable RBAC system integrated with AuthKit's policy enforcement.

It is designed to be:
- Generic: you define permission keys and resource types
- Scoped: global → org → project → resource with inheritance
- Deterministic: explicit deny can override allow (configurable)
- Fast: versioned role/principal caches
- Auditable: explainable decisions and pluggable audit

---

### Package layout

```
rbac/
  README.md                # this file
  config.go                # module configuration
  context.go               # Subject and ResourceRef types
  policy.go                # core types: roles, entries, conditions, decisions
  conditions.go            # condition registry + built-ins
  cache.go                 # in-memory cache implementation
  evaluator.go             # decision engine (deny-before-allow, scope-aware)
  rbac.go                  # public facade (Manager, New)
  manager_impl.go          # Manager implementation
  middleware.go            # future Gin helpers (reserved)
  admin_api.go             # future admin API (reserved)
  gormstore/
    models.go              # GORM models
    store.go               # GORM-backed Store implementation
  testing/
    inmemory_store.go      # in-memory Store for unit/integration tests
```

---

### Core concepts

- Subject: who acts (user/group/service). Minimal fields: `Type`, `ID`, optional `OrgID`, `Attrs`.
- ResourceRef: what is acted on. Fields: `Type`, `ID`, optional `OrgID`, `Parent` (for hierarchy), `Attrs` for conditions.
- PermissionKey: your authorization unit (opaque string, e.g., `"infra:deploy"`).
- RoleDef: named set of role entries.
- RoleEntry: Effect (allow/deny), PermissionKey, optional ResourceType, and ANDed conditions.
- Binding: links a subject to a role at a scope (global/org/project/resource), with hierarchical inheritance.

---

### Configuration and dependencies

```go
cfg := rbac.DefaultConfig()
cfg.EnableDeny = true                  // explicit deny overrides allow
cfg.RoleCacheTTL = 5 * time.Minute
cfg.PrincipalCacheTTL = 2 * time.Minute

deps := rbac.Deps{
    Store:  <rbac.Store>,             // GORM store or in-memory for tests
    Clock:  authkit.DefaultClock(),   // or custom
    Logger: <authkit.Logger>,
    Cache:  rbac.NewMemoryCache(),    // override if you want Redis-like
}

mgr, err := rbac.New(cfg, deps)
```

Store options:
- Use `rbac/gormstore.New(db)` with your GORM `*gorm.DB` (call `AutoMigrate` during boot).
- Use `rbac/testing.NewInMemoryStore()` for tests.

---

### Integration with PolicyRegistry and middleware

1) Create your `PolicyRegistry` with permission requirements.
```go
reg := authkit.NewPolicyRegistry().
  RequireAuth("*", "/api/*").
  RequirePermissions([]string{"infra:deploy"}, "POST", "/deploy")
```

2) Install the RBAC manager and attach to the enforcer.
```go
enforcer := middleware.NewPolicyEnforcer(reg).WithRBAC(mgr)
router.POST("/deploy", enforcer.EnforcePolicies(), handler)
```

3) Register resource resolvers per path pattern where resource context is needed (optional for resource-agnostic permissions).
```go
mgr.RegisterResolver("/deploy", func(c *gin.Context) (rbac.ResourceRef, error) {
  projectID := c.Query("project")
  return rbac.ResourceRef{ Type: "project", ID: projectID }, nil
})
```

At request time, the middleware will:
- Ensure authentication and (if configured) step-up
- Resolve the subject from `AuthContext` and resource via the registered resolver (or treat as resource-agnostic if none)
- Evaluate permissions by calling `mgr.Check(...)`
- Map step-up condition failures to `428 Precondition Required` and denials to `403 Forbidden`

---

### Programmatic checks

You can check permissions outside of HTTP middleware:
```go
subj := rbac.Subject{ Type: rbac.SubjectUser, ID: userID.String(), OrgID: &orgID }
res  := rbac.ResourceRef{ Type: "project", ID: projectID }

decision, err := mgr.Check(ctx, subj, rbac.PermissionKey("infra:deploy"), res)
// decision is rbac.DecisionAllow or rbac.DecisionDeny
```

---

### Conditions

Built-in conditions (plugin-based, you can register your own):
- `requires_step_up`: `{ "within": "5m" }` enforces recent step-up
- `attr_equals`: `{ "attr": "owner_id", "subject_field": "id" }`
- `org_matches`: `{}` ensures subject and resource belong to the same org if present
- `time_window`: `{ "start": "09:00Z", "end": "17:00Z", "days": ["Mon", ...] }`

Register custom conditions by implementing `ConditionEvaluator` and calling `rbac.RegisterCondition(...)`.

---

### Caching and invalidation

- Role cache: keyed by `role_slug + role.Version` (populated automatically on first use).
- Principal cache: keyed by `subject + principalVersion + action + resourceType`.
  - On permission changes for a principal (add/remove binding), call `Store.BumpPrincipalVersion` to invalidate.
  - TTLs are controlled by `Config`.

Roles/Bindings mutation guidance:
- After updating a role's entries, call `BumpRoleVersion(slug)`.
- After changing a subject's bindings, call `BumpPrincipalVersion(subject)`.

---

### Scopes and single-org setups

Supported scopes: global, org, project, resource. The evaluator:
- Applies deny across all applicable scopes first
- Applies allow by specificity: resource → project → org → global

Single-org model is fully supported:
- Use only global bindings (or a constant org ID) and avoid org/project/resource scoping
- Skip `org_matches` condition or set consistent OrgIDs

---

### Seeding roles and bindings

Define role slugs and entries during migrations/seeding:
```go
_ = store.CreateRole(ctx, rbac.RoleDef{
  Slug: "ops",
  Name: "Operators",
  Entries: []rbac.RoleEntry{
    {Effect: rbac.Allow, Permission: "infra:deploy", ResourceType: "project",
     Conditions: []rbac.Condition{{Name: "requires_step_up", Params: map[string]any{"within":"5m"}}}},
  },
})

_ = store.AddBinding(ctx, rbac.Binding{Subject: rbac.Subject{Type: rbac.SubjectUser, ID: userID},
  RoleSlug: "ops", ScopeType: rbac.ScopeProject, ScopeID: projectID})
```

---

### Testing

- Use `rbac/testing.NewInMemoryStore()` for unit tests and integration tests with `httptest`.
- GORM tests are provided under a build tag (`gormsqlite`). Run with:
```
go test -tags=gormsqlite ./lib/foundry/authkit/...
```

---

### Security notes

- Default deny; prefer allow-lists. Keep explicit denies for strategic carve-outs.
- Use `requires_step_up` for sensitive permissions.
- Prefer populating `OrgID` on both `Subject` and `ResourceRef` in multi-tenant setups and optionally add `org_matches`.
- Do not persist plaintext secrets in roles/conditions.

---

### Admin API (optional)

This package reserves `admin_api.go` for a future admin surface (CRUD for roles/bindings and `explain`). If enabled, secure it using your own permissions (e.g., `"rbac:admin"`).

---

### End-to-end examples (what happens and why)

These scenarios show concrete outcomes based on roles, bindings, conditions, and scopes.

Example A: Project deploy requires step-up

Setup
- Role `ops` includes an allow entry for `infra:deploy` on `project` with condition `requires_step_up: {within: "5m"}`.
- User U is bound to role `ops` at ScopeProject `p1`.
- Policy: `RequirePermissions(["infra:deploy"], "POST", "/deploy")` and resolver returns `ResourceRef{Type:"project", ID:"p1"}`.

Flow
1) U calls `POST /deploy?project=p1` without recent step-up
2) Enforcer resolves subject and resource; RBAC evaluates
3) Matching entry found (allow), condition evaluated; step-up missing
4) RBAC returns `ErrConditionStepUpRequired`
5) Middleware maps to HTTP 428 Precondition Required
Result: [Denied → 428 step_up_required]

6) U performs step-up (separate flow) and retries within 5 minutes
7) Condition passes (`StepUpValidUntil - now <= within`)
8) No deny entries apply; allow entry passes
Result: [Approved → 200]

Example B: Owner-only document read

Setup
- Role `owner` entry: allow `doc:read` on `doc` with `attr_equals: {attr:"owner_id", subject_field:"id"}`
- U bound to `owner` at global scope (no scoping needed; condition enforces ownership)
- Resolver for `/doc/:id` returns `ResourceRef{Type:"doc", ID:docID, Attrs:{"owner_id": <ownerUserID>}}`
- Policy: `RequirePermissions(["doc:read"], "GET", "/doc/:id")`

Flow
1) U requests `GET /doc/123`; resolver sets `owner_id` to U
2) RBAC finds allow entry; condition compares U.ID == resource.owner_id → true
3) No denies match → allow
Result: [Approved]

4) V (different user) requests same resource; resolver sets `owner_id` to U
5) Condition compares V.ID == U.ID → false
6) No other allows; default deny
Result: [Denied → 403]

Example C: Scope precedence (deny beats allow; specificity matters)

Setup
- Role `reader-global`: allow `data:read` (no ResourceType) — global scope binding
- Role `project-block`: deny `data:read` on `project` — project scope binding for `p1`
- ResourceRef for the request is `project p1`
- Policy requires `data:read`

Flow
1) U requests read on `project p1`
2) Applicable bindings: global allow, project deny
3) Evaluator order: first evaluate denies across all scopes → project deny matches and passes
4) Deny short-circuits; no further allows considered
Result: [Denied → 403]

Example D: Single-org, global-only permissions

Setup
- Role `auditor`: allow `logs:read` (no ResourceType)
- U bound to `auditor` at global scope
- No resolver registered (resource-agnostic)
- Policy requires `logs:read`

Flow
1) U requests `GET /logs`
2) No resolver → resource-agnostic evaluation; entry has empty ResourceType → matches
3) No denies; allow matches
Result: [Approved]



### RBAC seeding: extending permissions and roles

This package seeds a baseline permission taxonomy and roles (maintainer, developer, qa, viewer, ci_runner). Seeding is idempotent and will bump role versions when entries change to invalidate caches.

#### Where things live
- Permission keys and baseline roles: `seed.go`
- Invoked on boot: `cmd/api/bootstrap.go → initRBAC(...)`
- Config gate: `AUTH_RBAC_SEED_DEFAULTS=true|false` (default true)

#### Adding new API routes: what to change
1) Define new permission keys
   - Add constants to `seed.go`, grouped by domain.
   - Prefer verb-namespaced strings like `release:rollback`, `deploy:promote`, `env:update`.

2) Map routes → permissions
   - In the centralized policy registry (`internal/authkit/policy/registry.go`), require the appropriate permission(s) for the new route path/method.
   - If the route acts on a resource, register a resource resolver that sets a `ResourceRef{Type, ID, Parent, Attrs}` for scoping.

3) Decide scope and conditions
   - Set `ResourceType` on role entries to the target type (e.g., `"project"`, `"release"`, `"deployment"`).
   - Add safety conditions where needed:
     - `requires_step_up: { within: "5m" }` for sensitive operations
     - `org_matches` in multi-tenant setups
     - custom guards via `rbac.RegisterCondition`

4) Add to baseline roles (optional)
   - Update role definitions in `buildBaselineRoles()` to include the new permission in appropriate roles.
   - Keep `viewer` read-only; gate destructive operations behind `maintainer` with step-up.

5) Bump role versions on change
   - `SeedDefaultRoles(ctx, store, true)` will bump versions when entries differ, invalidating caches.
   - No manual calls needed—just adjust role entries and re-run.

6) Test
   - Add or update tests to cover the new permission → route mapping and expected allow/deny outcomes.
   - Consider edge conditions (step-up required, time windows, org isolation).

#### Patterns and tips
- Model permissions at the domain level; avoid per-endpoint keys. Multiple routes can reuse one permission (e.g., `deploy:read`).
- Favor project-scoped bindings for team roles; use org/global sparingly.
- Use explicit denies only for surgical carve-outs; rely on allow-lists + conditions for most cases.

#### Example: adding a release rollback route
1) Add key in `seed.go`:
```go
const PermReleaseRollback = rbac.PermissionKey("release:rollback")
```
2) Policy registry:
```go
reg.RequirePermissions([]string{"release:rollback"}, "POST", "/api/v1/releases/:id/rollback")
```
3) Resolver returns `ResourceRef{Type: "project", ID: <projectId>, Parent: orgRef}` for that release.
4) Update roles (e.g., only `maintainer`):
```go
allow(PermReleaseRollback, "project", stepUpCondition())
```
5) Rebuild/run; seeding will update roles and bump versions.



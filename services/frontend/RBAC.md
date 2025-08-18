## RBAC Admin Integration Plan

### Scope and goals
- **Goal**: Enable administrators to manage RBAC roles, entries, bindings, and inspect authorization decisions from the frontend.
- **Guardrails**: Admin UI gated by a server-enforced permission (e.g., `rbac:admin`). Server returns 403 for denials and 428 for step-up requirements.

### Backend API surface (to expose/confirm)
Add admin endpoints in the API service (backed by `rbac.Store` and `rbac.Manager`). Keep CRUD deterministic, version-aware, and auditable.

- **Roles**
  - GET `/rbac/roles` → list roles (omits entries for efficiency)
  - GET `/rbac/roles/{slug}` → role with entries
  - POST `/rbac/roles` → create role
  - PUT `/rbac/roles/{slug}` → update role (replace entries)
  - POST `/rbac/roles/{slug}/bump-version` → bump role version (invalidate role cache)
- **Bindings**
  - GET `/rbac/bindings?subject_type=&subject_id=` → bindings for a subject
  - GET `/rbac/bindings/by-scope?scope_type=&scope_id=` → bindings in a scope
  - POST `/rbac/bindings` → add binding
  - DELETE `/rbac/bindings/{id}` → remove binding
  - POST `/rbac/subjects/{subject_type}/{subject_id}/bump-version` → bump principal version (invalidate principal cache)
- **Explain**
  - POST `/rbac/explain` with `{subject, permission, resource}` → returns `Trace` with steps and condition evals
- **Conditions registry (optional, recommended)**
  - GET `/rbac/conditions` → list known condition names and param hints/schemas to drive the UI

All endpoints are protected by a policy like `RequirePermissions(["rbac:admin"])`.

### Frontend client plumbing
- Extend the OpenAPI spec with the RBAC endpoints and regenerate the TS client.
  - Use existing flow: `openapi-fetch` and `scripts/sync-vendored-client.sh`.
  - Place the generated client under `services/frontend/vendor` (follow project conventions).
- Create a small wrapper in `src/lib/api/rbac.ts` that:
  - Exposes typed functions for the endpoints above.
  - Normalizes errors (403 vs 428) and surfaces `step_up_required` distinctly.

### UI architecture
- Add an admin section at route base: `/admin/rbac`.
- File structure (suggested):
  - `src/routes/admin/rbac/index.tsx` → top-level tabs/nav
  - `src/routes/admin/rbac/roles.tsx` → roles list and role editor overlay
  - `src/routes/admin/rbac/bindings.tsx` → bindings by subject/scope
  - `src/routes/admin/rbac/explain.tsx` → decision explain tool
- Use shadcn/ui components, `@tanstack/react-query` for data, `react-hook-form` + `zod` for validation.

### Screens and flows
- **Roles**
  - List: table with `slug`, `name`, `description`, `version`, actions (view/edit).
  - Role editor: edit `name`, `description`, and manage `entries` array.
    - Entry fields: `effect` (allow/deny), `permission` (string), `resourceType` (optional), and conditions list.
    - Condition builder supports built-ins: `requires_step_up`, `attr_equals`, `org_matches`, `time_window`, and SAN-related: `dns_sans_suffix_in`, `uri_sans_prefix_in`, `ip_sans_in_cidrs`.
    - Parameter editors: schema-driven if `/rbac/conditions` exists; else provide a JSON editor for params.
    - Actions: Save (PUT), and separate "Bump version" CTA calling `/rbac/roles/{slug}/bump-version`.
- **Bindings**
  - By Subject tab: search by `subject_type` + `subject_id`; list bindings; add/remove.
  - By Scope tab: filter by `scope_type` + `scope_id`; list bindings within that scope; remove as needed.
  - Add binding form fields: role slug, scope type (global/org/project/resource), scope ID, optional OrgID.
  - After add/remove, call the principal bump endpoint for the subject.
- **Explain**
  - Form inputs:
    - Subject: `type`, `id`, optional `orgId`, optional `attrs` JSON.
    - Permission: key (string).
    - Resource: `type`, `id`, optional `orgId`, optional parent chain, optional `attrs` JSON.
  - Results: render `Trace.Steps[]` with scope, scope ID, role entry, and condition evals (pass/fail + reason). Display outcome prominently.

### State, validation, and UX
- **Data fetching**: `react-query` for queries/mutations with invalidation on successful writes; use optimistic updates where safe.
- **Validation**: `zod` schemas for RoleEntry and Condition params. For unknown conditions, validate as generic JSON with server-side feedback.
- **UX**: use `sonner` toasts, confirmation dialogs for destructive ops, loading skeletons, inline field errors.

### Security and gating
- Backend strictly enforces `rbac:admin` policy.
- Frontend shows the admin nav only if a lightweight probe (e.g., GET `/rbac/roles`) returns 200; do not rely solely on client-side assumptions.

### Caching and invalidation
- After role updates: present "Bump version" action to ensure role cache invalidation.
- After binding changes: call subject principal version bump endpoint.
- Explain in UI tooltips how role/principal versioning affects cache behavior.

### Testing
- Backend: unit tests for each endpoint and evaluator interactions, using the in-memory store or `gormsqlite` tagged tests.
- Frontend: component tests for forms and condition builders; integration tests using MSW to mock OpenAPI responses; include happy-path, edge, and failure (403/428) cases.

### Rollout steps
1. Implement/confirm backend endpoints and update OpenAPI spec; ensure `gormstore.AutoMigrate()` is called at boot.
2. Regenerate the frontend OpenAPI client and add `src/lib/api/rbac.ts` wrapper.
3. Scaffold `/admin/rbac` routes and build Roles, Bindings, and Explain screens.
4. Seed test roles/bindings; perform E2E checks; add audit logging where available.

### Nice-to-haves (incremental)
- Conditions registry endpoint returns param schemas and examples to render dynamic forms.
- Role cloning and version diff viewer.
- Bulk binding editor for project/org scopes; CSV import/export.



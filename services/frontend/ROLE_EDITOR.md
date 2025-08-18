# Role Editor UX Specification

This document specifies the end-to-end UX and API interactions for the Role Editor and Binding Builder used to manage RBAC roles and assignments.

The goals are:
- Ensure users can safely create/edit roles with clear permission semantics
- Prevent invalid ResourceType usage through canonical lists and validation
- Make binding setup intuitive across scopes (global, org, project, resource)

---

## Core Concepts (recap)

- **Permission**: string key describing an action (e.g., `deploy:create`).
- **Role**: named set of role entries granting or denying permissions, optionally tied to a resource type and conditions.
- **Role Entry**: `{ effect, permission, resource_type?, conditions[] }`
  - `effect`: `allow` or `deny`
  - `permission`: canonical key from the catalog
  - `resource_type` (optional): must be one of the canonical resource types; if empty, the entry is resource-agnostic
  - `conditions`: ANDed constraints evaluated at request time (e.g., step-up required)
- **Binding**: attaches a role to a subject at a scope with optional scope ID
  - Scopes: `global`, `org`, `project`, `resource`

---

## API Surface (used by the builder)

- Permissions metadata (canonical list)
  - GET `/api/v1/rbac/permissions/catalog`
  - Response:
    ```json
    { "permissions": [
      { "key": "deploy:create", "name": "Create Deployments", "description": "...", "domain": "deployments" },
      { "key": "artifact:read", "name": "Read Artifacts", "description": "...", "domain": "artifacts" }
    ]}
    ```

- Canonical resource types (canonical list)
  - GET `/api/v1/rbac/resource-types`
  - Response:
    ```json
    { "types": ["org","project","environment","release","deployment","build","artifact","repository"] }
    ```

- Resource browser (proposed; for instance selection when scope=resource)
  - GET `/api/v1/rbac/resources`
  - Query params:
    - `type` (required): one of canonical resource types
    - Hierarchy filters (optional, depending on type): `org_id`, `project_id`, `release_id`, `build_id`
    - Search/filtering: `q` (string), `limit` (int, default 20), `cursor` (opaque string)
  - Response:
    ```json
    { "items": [
      { "id": "p1", "type": "project", "label": "Project Alpha", "parent_type": "org", "parent_id": "o1" }
    ],
      "next_cursor": "..." }
    ```
  - Notes:
    - The backend maps natural keys to canonical IDs; frontend should treat `id` as opaque.
    - The list is filtered by parent hints when provided (e.g., only environments within a project).

- Role CRUD (already implemented)
  - POST `/api/v1/rbac/roles` (create)
  - PUT `/api/v1/rbac/roles/{slug}` (update)
  - GET `/api/v1/rbac/roles` (list)
  - GET `/api/v1/rbac/roles/{slug}` (get)
  - POST `/api/v1/rbac/roles/{slug}/bump-version`

- Bindings (already implemented)
  - GET `/api/v1/rbac/bindings` (by subject)
  - GET `/api/v1/rbac/bindings/by-scope` (by scope)
  - POST `/api/v1/rbac/bindings` (create)
  - DELETE `/api/v1/rbac/bindings/{id}` (delete)

---

## UX Flow: Role Editor (Role Entries)

1) Load permissions catalog and resource types in parallel
   - Fetch `/permissions/catalog` and `/resource-types`

2) Add/Edit role entry
   - Fields:
     - Effect: radio `Allow` | `Deny`
     - Permission: searchable dropdown from catalog (shows `Name` and `Domain`; stores `key`)
     - Resource type: dropdown populated from `/resource-types`
       - Include `Any` option to leave entry resource-agnostic (maps to empty `resource_type`)
     - Conditions (optional): UI for known conditions (e.g., `requires_step_up`, `time_window`)
   - Validation:
     - Permission must be one of the catalog keys
     - Resource type must be empty or in the canonical list
     - For `Deny` entries, warn if resource type is empty (deny may be too broad)

3) Save role
   - Sends role DTO with entries; backend validates and persists
   - On success, bump role version as needed

---

## UX Flow: Binding Builder

1) Select Subject
   - Subject type: `user` | `group` | `service`
   - Subject identifier: search/picker (the API for subjects is outside this spec; reuse existing endpoints)

2) Select Scope (critical for UX clarity)
   - Scope dropdown: `global`, `org`, `project`, `resource`

3) Scope-specific UI and validation
   - global
     - No additional pickers
     - Resource type dropdown is disabled (not applicable)
   - org
     - Show org picker; submit selected `org_id` as `ScopeID`
     - Resource type dropdown is disabled (not applicable)
   - project
     - Show project picker; submit selected `project_id` as `ScopeID`
     - Resource type dropdown is disabled (not applicable)
   - resource
     - Enable resource type dropdown (from `/resource-types`)
     - Show parent pickers as needed:
       - For `environment`, `release`, `build`, `deployment`, `artifact`: require project picker
       - For `repository`: require org picker (or project picker if repos are project-scoped in your domain)
     - Show resource instance picker driven by `/rbac/resources?type=...`
       - Pass parent filters (`project_id`, `org_id`) and optional `q`
       - On selection, set `ScopeType=resource` and `ScopeID=<chosen id>`

4) Submit binding
   - POST `/rbac/bindings` with `{ subject, role_slug, scope_type, scope_id, org_id? }`
   - On success, optionally prompt to bump principal version to invalidate caches

---

## Matching Semantics (why this UI matters)

- A role entry matches a request only if:
  - Permission keys are equal (case-insensitive compare in evaluator)
  - AND entry `resource_type` is empty (resource-agnostic) OR equals the `ResourceRef.Type` emitted by the resolver
  - AND conditions (if any) evaluate to true
  - AND the binding scope applies (resource/project/org/global) using ancestry from `ResourceRef.Parent`
- Consequences for UX:
  - If a route has no resolver, `ResourceRef.Type` is empty → only resource-agnostic entries can match for that route
  - A mismatched `resource_type` string in a role entry will never match; this is why we enforce canonical types and a resource browser

---

## Error States & Validation

- Role save/update
  - 400 if `resource_type` is not in canonical list
  - 400 if `permission` not in catalog
  - 409 if optimistic concurrency fails (role version mismatch), prompt to reload

- Binding create
  - 400 if `scope_type`/`scope_id` invalid or missing
  - 400 if `resource` type requires a parent but none provided
  - 404 if `scope_id` not found for the chosen `type`

- Authorization preview (optional future enhancement)
  - UI can call `POST /api/v1/rbac/explain` with the intended subject, permission, and resource to preview the decision trace

---

## Caching & Performance

- Cache permissions catalog and resource types for the session; refresh on page load
- Debounce search input to `/rbac/resources` (e.g., 250–400ms)
- Paginate via `limit` and `next_cursor`

---

## Accessibility & Internationalization

- Provide accessible labels with permission `Name` and concise descriptions; tooltips for extra detail
- Keep effect toggles keyboard-accessible; announce validation errors inline
- Avoid encoding IDs in visible text; always show human-friendly `label`

---

## Examples

- Project-scoped reader
  - Role entry: `allow` `artifact:read` (resource-agnostic)
  - Binding: scope=`project`, scope_id=`p1`

- Resource-scoped deployer
  - Role entry: `allow` `deploy:create` on `environment`
  - Binding: scope=`resource`, type=`environment`, scope_id=`env-123` (with project filter)

- Global auditor with time window
  - Role entry: `allow` `audit:read` (resource-agnostic) + condition `time_window`
  - Binding: scope=`global`

---

## Open Questions / Future Work

- Pattern-based resolvers (prefix/param support) to avoid exact path registration
- Server-side validation that ties `permission` ↔ allowed `resource_type` pairs (optional hardening)
- Bulk operations for bindings and role entries
- Inline test harness using `explain` to preview outcomes from the UI

---

## Reference

- Canonical endpoints used by the builder
  - GET `/api/v1/rbac/permissions/catalog`
  - GET `/api/v1/rbac/resource-types`
  - GET `/api/v1/rbac/resources` (proposed)
  - POST `/api/v1/rbac/bindings`
  - POST `/api/v1/rbac/explain` (optional preview)

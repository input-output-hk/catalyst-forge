# RBAC System: High‑Level Summary

This document explains how RBAC is modeled and configured so you can define permissions, roles, and bindings safely and predictably.

## Core Concepts

- **Permissions**
  - Canonical string keys (e.g., `release:create`, `deploy:promote`).
  - Catalog metadata: `key`, `name`, `description`, `domain`.
  - API: `GET /api/v1/rbac/permissions/catalog`.

- **Scopes**
  - Where a role binding applies:
    - `global`: applies everywhere
    - `org`: applies to an organization (Org ID)
    - `project`: applies to a project (Project ID)
    - `resource`: applies to a specific resource instance (Resource ID)
  - Specificity order: resource → project → org → global.
  - Denies are evaluated before allows across all applicable scopes.

- **Resource types (instance-level)**
  - Used when scope = `resource` to target a specific instance.
  - Canonical types: `environment`, `release`, `deployment`, `build`, `artifact`, `repository`.
  - API: `GET /api/v1/rbac/resource-types`.
  - Catalog mapping types to existing list endpoints: `GET /api/v1/rbac/resource-types/catalog` (gives method, path, id/label fields, parent/query params).
  - Note: `org` and `project` are scopes, not resource types.

- **Roles**
  - Named set of entries that grant or deny permissions.
  - Fields: `slug`, `name`, `description`, `color`, `version`, `entries`.
  - Role Entry fields:
    - `effect`: `allow` | `deny`
    - `permission`: permission key
    - `resource_type` (optional): empty = resource‑agnostic; otherwise one of the canonical resource types
    - `conditions` (optional): ANDed constraints (e.g., requires recent step‑up)
  - API:
    - `GET /api/v1/rbac/roles`, `GET /api/v1/rbac/roles/{slug}`
    - `POST /api/v1/rbac/roles`, `PUT /api/v1/rbac/roles/{slug}`
    - `POST /api/v1/rbac/roles/{slug}/bump-version`

- **Bindings**
  - Attach a role to a subject at a scope.
  - Subject: `type` (`user`|`group`|`service`) + `id`.
  - Scope: `global` (no ID), `org` (Org ID), `project` (Project ID), `resource` (Resource ID).
  - API:
    - `GET /api/v1/rbac/bindings?subject_type=&subject_id=`
    - `GET /api/v1/rbac/bindings/by-scope?scope_type=&scope_id=`
    - `POST /api/v1/rbac/bindings`, `DELETE /api/v1/rbac/bindings/{id}`

- **Conditions**
  - Optional constraints for entries (e.g., `requires_step_up`, time windows, org matches).
  - API: `GET /api/v1/rbac/conditions`.

- **Evaluation**
  - Inputs: Subject, Permission, ResourceRef.
  - Order:
    1) Collect entries from all bindings whose scopes apply (based on resource ancestry).
    2) Evaluate all denies first across scopes.
    3) Evaluate allows by specificity: resource → project → org → global.
    4) Default is deny if nothing matches or conditions fail.
  - Explain API: `POST /api/v1/rbac/explain` returns decision and trace.

- **Resource resolvers (HTTP integration)**
  - Routes can register a resolver that emits `ResourceRef{Type, ID, Parent, OrgID}`.
  - If an entry sets `resource_type`, the resolver must emit that exact type for a match.
  - Without a resolver, only resource‑agnostic entries can match that route.

## How to Configure RBAC

1) Define permissions
   - Use catalog keys; add new ones in backend if needed.
   - Reference keys directly in role entries.

2) Design roles
   - Group permissions by responsibility (e.g., `viewer`, `developer`, `maintainer`, `admin`).
   - Leave `resource_type` empty for resource‑agnostic behavior.
   - Use canonical `resource_type` for instance‑level control.
   - Add conditions for sensitive actions (e.g., step‑up).

3) Bind roles to subjects
   - Choose scope carefully:
     - `global`: broad; use sparingly.
     - `org`: applies within the org.
     - `project`: applies within the project.
     - `resource`: applies to an instance; pick type and item via the catalog‑guided endpoints.
   - Provide the `scope_id` when required.

4) Validate with Explain
   - Use `POST /api/v1/rbac/explain` to verify the decision path before rolling out changes.

## Admin & Safety

- Admin endpoints require `rbac:admin`.
- Fail‑closed: routes absent from the policy registry return 404.
- No super‑user bypass; all access goes through permissions.
- Caching:
  - Bump role: `POST /api/v1/rbac/roles/{slug}/bump-version`.
  - Bump principal: `POST /api/v1/rbac/subjects/{type}/{id}/bump-version`.

## Quick Reference

- Permissions: `GET /api/v1/rbac/permissions/catalog`
- Resource types: `GET /api/v1/rbac/resource-types`
- Type→endpoint catalog: `GET /api/v1/rbac/resource-types/catalog`
- Conditions: `GET /api/v1/rbac/conditions`
- Roles: list/get/create/update/bump
- Bindings: list by subject/scope, create, delete
- Explain: `POST /api/v1/rbac/explain`

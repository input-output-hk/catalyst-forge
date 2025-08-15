# Domain API Integration Tests Plan

This document enumerates the integration tests to add under `services/api/test/domain/`, following our established pattern (package-level TestMain with one Postgres container per package, per-test API env via snapshot restore) and the repository testing standards in `.ai/go/testing.md`.

We will create a `domain` test package with:
- `testmain_test.go`: start suite once, snapshot migrations, stop on exit
- `helpers_test.go`: `newDomainEnv(t)` helper returning per-test env (`*testutil.Env`)
- Table-driven tests for each resource: repositories, projects, builds, artifacts, environments, releases

General rules
- Use `require` for setup/preconditions and HTTP calls whose failure blocks subsequent assertions
- Use `assert` for additional checks
- Include positive and negative cases for each endpoint (success, not found, bad request, conflict, invalid transitions)
- Use `t.Parallel()` for tests that do NOT call `t.Setenv`; otherwise avoid parallel for those tests
- Seed minimal data via API endpoints where practical; otherwise, use direct DB helpers only when an API does not exist yet

## Repositories
Endpoints (from `handlers/repository.go`):
- GET /api/v1/repositories/{repo_id}
- GET /api/v1/repositories/by-path/{host}/{org}/{name}
- GET /api/v1/repositories

Tests:
- ok/get-by-id: create a repo via DB helper or dedicated seed, GET by id → 200, fields match
- error/get-by-id/invalid-id: non-UUID → 400
- error/get-by-id/not-found: random UUID → 404
- ok/get-by-path: with existing repo, GET by host/org/name → 200
- error/get-by-path/not-found → 404
- ok/list/defaults: without filters, returns 200 and page result
- ok/list/filters: host/org/name filters return expected

## Projects
Endpoints (from `handlers/project.go`):
- GET /api/v1/projects/{id}
- GET /api/v1/repositories/{repo_id}/projects/by-path?path=...
- GET /api/v1/projects

Tests:
- ok/get-by-id → 200
- error/get-by-id/invalid-id → 400
- error/get-by-id/not-found → 404
- ok/get-by-repo-and-path → 200
- error/get-by-repo-and-path/invalid-params → 400
- error/get-by-repo-and-path/not-found → 404
- ok/list/defaults → 200 with pagination fields
- ok/list/filters: repo_id/path/slug/status → 200

## Builds
Endpoints (from `handlers/build.go`):
- POST /api/v1/builds
- GET /api/v1/builds/{id}
- GET /api/v1/builds
- PATCH /api/v1/builds/{id}
- PATCH /api/v1/builds/{id}/status

Tests:
- ok/create/minimal → 201; then GET by id → 200
- error/create/missing-required → 400
- error/create/repo-not-found → 404
- error/create/project-not-found → 404
- ok/get-by-id → 200
- error/get-by-id/invalid-id → 400; not-found → 404
- ok/list/filters (trace_id/repo_id/project_id/commit_sha/branch/workflow_run_id/status/since/until) → 200
- ok/update/fields → 200; fields changed
- error/update/not-found → 404; invalid-status → 422
- ok/update-status/valid → 204
- error/update-status/not-found → 404; invalid-status → 422

## Artifacts
Endpoints (from `handlers/artifact.go`):
- POST /api/v1/artifacts
- GET /api/v1/artifacts/{id}
- GET /api/v1/artifacts/digest/{digest}
- GET /api/v1/artifacts
- PATCH /api/v1/artifacts/{id}
- DELETE /api/v1/artifacts/{id}

Tests:
- ok/create/container → 201; then GET by id → 200
- error/create/build-not-found → 404; invalid body → 400
- ok/get-by-digest → 200; not-found → 404; invalid format → 400
- ok/list/filters (build_id,image_name,image_digest) → 200
- ok/update/labels-metadata → 200; fields updated
- error/update/not-found → 404
- ok/delete → 204
- error/delete/not-found → 404; in-use → 409

## Environments
Endpoints (from `handlers/environment.go`):
- POST /api/v1/environments
- GET /api/v1/environments/{id}
- GET /api/v1/projects/{project_id}/environments/{name}
- GET /api/v1/environments
- PATCH /api/v1/environments/{id}
- DELETE /api/v1/environments/{id}

Tests:
- ok/create/minimal → 201
- error/create/invalid → 400; conflict on duplicate → 409
- ok/get-by-id → 200; not-found → 404; invalid-id → 400
- ok/get-by-project-and-name → 200; invalid params → 400; not-found → 404
- ok/list/defaults → 200; filters (name, cluster_ref, namespace, environment_type) → 200
- ok/update/fields → 200; conflict when protected → 409; not-found → 404
- ok/delete → 204; not-found → 404; conflict (in use/protected) → 409

## Releases
Endpoints (from `handlers/release.go`):
- POST /api/v1/releases
- GET /api/v1/releases/{id}
- GET /api/v1/releases
- PATCH /api/v1/releases/{id}
- DELETE /api/v1/releases/{id}
- GET /api/v1/releases/{id}/modules
- POST /api/v1/releases/{id}/modules
- DELETE /api/v1/releases/{release_id}/modules/{module_key}
- GET /api/v1/releases/{id}/injections
- POST /api/v1/releases/{id}/injections
- DELETE /api/v1/releases/{release_id}/injections/{injection_id}
- GET /api/v1/releases/{id}/artifacts
- POST /api/v1/releases/{id}/artifacts
- DELETE /api/v1/releases/{release_id}/artifacts/{artifact_id}

Tests:
- ok/create/minimal (modules/injections/artifacts empty) → 201
- error/create/invalid-body → 400
- error/create/project-not-found → 404; artifact-not-found → 404
- ok/get-by-id → 200; not-found → 404; invalid-id → 400
- ok/list/filters (project_id, release_key, status, oci_digest, tag, created_by, since, until) → 200
- ok/update/status-and-signature → 200
- error/update/not-found → 404; sealed → 409
- ok/delete → 204; not-found → 404; sealed → 409
- ok/modules/list → 200; add → 201; remove → 204
- ok/injections/list → 200; add → 201; remove → 204
- ok/artifacts/list → 200; attach → 201; detach → 204

## Test data setup
- When possible, create dependent resources via API (e.g., create repository → project → build → artifact → release) to exercise full stack.
- Where an API to create a dependency doesn’t exist yet, use DB helper functions scoped to tests (to be added under `testutil` if necessary) to seed minimal rows.

## Package layout
- `services/api/test/domain/testmain_test.go`: suite start/stop, snapshot
- `services/api/test/domain/helpers_test.go`: env helper, HTTP helpers reuse from `testutil`
- `services/api/test/domain/repositories_test.go`
- `services/api/test/domain/projects_test.go`
- `services/api/test/domain/builds_test.go`
- `services/api/test/domain/artifacts_test.go`
- `services/api/test/domain/environments_test.go`
- `services/api/test/domain/releases_test.go`

Each file will:
- Use table-driven tests with positive and negative cases
- Reuse `tu.DoJSON` for HTTP calls
- Use `t.Setenv` only when necessary; otherwise mark tests/subtests as parallel


# Test Suite Tasks — Make Tests Healthy Again

This document lists concrete, prioritized actions to remove duplication, modernize flows, improve coverage, and reduce flakiness across the integration tests in `test/`.

Tests can be run with:

- `just test-integration`

Or look at the `.justfile` to see the exact command and modify it to target specific tests.ss

## P0 — Remove Duplicates, Obsolete Code, Broken Flows

- [x] Consolidate Events coverage
  - [x] Keep Events API coverage in `test/events_test.go`; remove duplicate event checks in `test/deployment_test.go::TestDeploymentEvents` (or refactor that subtest to only test deployment-specific behavior).
- [x] Fold server certificate cases together
  - [x] Move “invalid: no SANs” case from `test/server_certificate_test.go` into `test/certificate_test.go`.
  - [x] Remove `test/server_certificate_test.go` once the unique case is ported.
- [x] Remove dead/unused helpers
  - [x] Delete `newTestClient()` and `getTestAPIURL()` from `test/common_test.go` (not used; all tests rely on `NewTestEnv(t)` + `AdminClient()`).
- [x] Replace or remove obsolete onboarding test
  - [x] Rewrite `test/onboarding_test.go` to use the device-keypair auth flow (`/auth/devices/*`) end-to-end and assert a protected call using the issued access token.
  - [x] If not feasible now, remove the test instead of skipping with `t.Skip`.

## P1 — Strengthen Weak Tests

- [x] JWKS validation
  - [x] Assert JWK fields for each key: `kty`, and either `crv`+`x`+`y` (EC) or `n`+`e` (RSA); optional `use`, `alg`, `kid` present.
  - [x] Optionally verify a sample JWT signature can be validated using the JWKS.
- [x] Build sessions
  - [x] Add `Get` and `List` assertions; validate TTL parsing and expiry semantics where exposed.
  - [x] Assert metadata persistence and retrieval.
  - [x] Negative inputs: invalid `owner_type`, malformed `ttl`.
- [x] Health check
  - [x] Decide to keep as smoke (simple 200) or drop as redundant since the harness already waits for `/healthz`.

## P1 — Add Authorization and Negative Tests

- [x] Authorization/permissions
  - [x] Attempt admin-only endpoints without a token → expect 401.
  - [x] Attempt admin-only endpoints with a limited role → expect 403.
  - [x] Verify permission enforcement on representative routes (users list/get, roles CRUD, releases create).
- [x] Invites
  - [x] Invalid/expired invite token returns an error.
  - [x] Reuse attempt of an invite token fails.
  - [x] TTL boundary tests (near expiration behaves as expected).
- [x] Device auth
  - [x] Registration: bad timestamp (skew), wrong origin/host, invalid proof format, mismatched device ID, invalid JWK values.
  - [x] Refresh/logout: wrong canonical components, replayed proof (same ts), expired ts.
- [x] Releases
  - [x] Invalid create/update: missing `Project`, invalid base64 `Bundle`, invalid `SourceBranch` format.
  - [x] Get non-existent release → error. Delete coverage if supported.
- [x] Aliases
  - [x] Duplicate alias name on same release → error.
  - [x] Invalid alias name format → error (if validated).
  - [x] Non-existent release ID when creating alias → error.
  - [x] Cross-project isolation: alias on project A must not leak to project B.
- [x] Deployments
  - [x] Invalid state transitions (if rules exist) → error.
  - [x] Get/Update with non-existent IDs → error.
- [x] Certificates
  - [x] Client certificate: parse returned cert, verify key usages/EKU appropriate for client, and SAN behavior.
  - [x] Invalid CSR: random bytes (non-PEM), wrong PEM type, malformed DER.
  - [x] TTL clamping: request excessive TTL and assert cap (if implemented) or error consistently.
- [x] Users/Roles/Keys
  - [x] Users: duplicate emails → error; invalid status → error; get/update/delete non-existent id → error.
  - [x] Roles: idempotent assign/remove; removing non-assigned user returns error (or is a no-op with clear response).
  - [x] Keys: invalid base64 pubkey → error; conflicting `kid` update → error; revoke idempotency behavior; list filters (active/inactive) behave consistently.

## P2 — Reduce Flakiness and Improve Test Infra

- [x] Centralize HTTP helpers
  - [x] De-duplicate `doJSON`: consolidate into `test/testutil` with optional `*http.Client` parameter; remove copy in `device_auth_test.go`.
  - [x] Remove `testingContext` in `device_auth_test.go` and consistently use `newTestContext()`.
- [x] Determinism and timing
  - [x] Prefer polling over fixed sleeps; where sleeps exist, replace with condition waits.
  - [x] Ensure Postgres readiness checks remain robust after snapshot restore (already present, keep).
  - [x] Rate-limiter: confirm `AUTH_RATELIMIT_DISABLE=1` in test env for stability; if specific per-principal limits still apply, vary principal identifiers in tests.
- [x] API server bootstrap
  - [x] Keep `StartAPIServer` fallback to `bin/foundry-api` or `go run`; document expectations in `README.md`.

## P2 — Structure, Style, and Consistency

- [x] Assertions
  - [x] Use `require` for critical setup; use `assert` for value checks. Prefer explicit error messages.
  - [x] Replace helper returns of `assert.AnError` with explicit `errors.New` for clarity.
- [x] Timeouts and cleanup
  - [x] Standardize context timeouts (e.g., 30s) and ensure all external resources are closed via `t.Cleanup`.
- [x] Naming and layout
  - [x] Ensure test names describe behavior (state/action/expectation). Group tests by domain (releases, aliases, etc.).

## CI and Docs

- [x] Document how to run integration tests locally (requires Docker/testcontainers) and in CI.
- [x] Ensure CI job sets build tag `integration` and required env for testcontainers.
- [x] Update `README.md` test section to reflect new device-keypair auth onboarding flow and how to bootstrap admin in tests.

## Acceptance Criteria

- P0 items complete: no duplicated suites, no skipped/obsolete onboarding test, no unused helpers.
- P1 items add meaningful negative/authorization coverage across major surfaces and strengthen weak tests (JWKS, build sessions).
- Test runs are deterministic and pass locally and in CI without flaky skips or intermittent rate-limit issues.
- Documentation updated to reflect the modernized test flow.


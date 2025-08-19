## AuthKit hardening goals (production readiness)

This document tracks the concrete goals to make `services/api/internal/authkit` production‑grade. It is a high‑signal checklist for the whole package, including the embedded `rbac` module.

References
- Code style: `.ai/go/style.md`
- Testing: `.ai/go/testing.md`

Global objectives
- Style: Conform to repo Go style and formatting; meaningful names; explicit types for exported APIs; no deep nesting; avoid unused/arcane patterns.
- Docs: Every package has a `doc.go`; all exported types/functions/methods have GoDoc; complex validation/security logic has short inline comments explaining the “why”.
- Testing: Per `.ai/go/testing.md`, add unit tests with: happy path, edge case, failure case. Ensure deterministic tests; avoid flakiness; use in‑memory fakes where feasible. Aim for high coverage of auth flows, token rotation, WebAuthn ceremonies, policy/rbac checks.
- Dead/duplicate code: Remove or consolidate; prefer single implementation paths; delete leftover stubs or TODO scaffolding if superseded.
- Performance: Eliminate obvious allocations/copies, reduce N+1 store calls, add caching where safe (RBAC cache exists), ensure zero/low allocations on hot paths (middlewares), and avoid unnecessary JSON marshal/unmarshal.
- Security: Enforce cookie invariants, CSRF on refresh, strict issuer/audience/time validation, replay detection, step‑up priority, admin rules, and safe cryptography defaults. Validate inputs; constant‑time compares where appropriate; single‑use challenges.
- Dependency hygiene: Avoid tight coupling to other internal packages beyond adapters; use interfaces; keep cross‑code pollution minimal. Adapters for app config/infra should live only in package‑root `config.go` and `deps.go`.

Per‑subpackage goals and checks

authkit/
- Ensure `doc.go` explains public API and integration model.
- Confirm `Manager` methods have GoDoc and examples (where helpful) and that `Handlers()` mirrors `RegisterRoutes` behavior consistently.
- `policy.go`: Document matcher semantics (exact/prefix/regex), `MergeRules` behavior, and admin/step‑up precedence. Add tests for matching/merging edge cases.
- `context.go`: Docs for `AuthContext` fields and helpers; tests for `RequiresStepUp`, role/permission helpers.
- Eliminate duplication between middleware and manager middlewares if any overlap remains.

service/
- WebAuthn: tests for registration/login/step‑up including Safari/iCloud edge cases; verify AAGUID allowlist logic; ensure single‑use challenge deletion on all terminal paths; document rationale on lenient backup‑eligible handling.
- Tokens: tests for Sign/Parse (issuer/audience/exp/leeway/sv/claims arrays). Avoid repeated marshal/unmarshal; ensure constant TTL handling; benchmark Parse in middleware context.
- Refresh: tests for issue/rotate/replay/revoke/session‑version mismatch; audit events; ensure rotation is atomic and safe; micro‑optimize base64 and hashing hot paths.
- Recovery/flows: tests for flow TTL, verify/complete; validate inputs.

store/
- Interfaces: verify completeness and cohesion; document semantics (e.g., rotation invariants, version bumps, single‑use guarantees).
- GORM stores: add tests (using sqlite in‑memory) for model constraints, indexes, and transactional behavior; check N+1 queries; ensure efficient lookups by keys/hashes; avoid scanning large blobs unnecessarily.
- memstore: validate TTL correctness; concurrency safety tests.

domain/
- Ensure entities document invariants (e.g., `SessionVersion`, credential counters, invite attempts/lockout). Add lightweight validation helpers if missing (and tests).

httpkit/
- Enforce `__Host-*` invariants; ensure SameSite/MaxAge/Expires correctness; tests for set/get/clear helpers.

middleware/
- Authenticate/RequireAuth/RequireStepUp/PolicyEnforcer: ensure consistent behavior with `Manager` middlewares; prioritize step‑up before roles/permissions; tests for bearer vs cookie tokens; failure fall‑through; 401/403/404/428 mapping; zero/low allocations under load.

crypto/
- Key manager strictness: no KID reuse; tests for key rotation and JWKS exposure; secure rand tests; ensure FIPS‑sane primitives; avoid weak hashes except for non‑security uses; constant‑time compares where used.

rate/
- Validate limiter interfaces and default no‑op; document guarantees; add basic tests.

handlers/
- Ensure any direct handlers are thin and reuse services; test JWKS and any additional endpoints.

rbac/
- `doc.go` describes model (roles, entries, conditions, scopes, planner, cache).
- Evaluator: tests for deny precedence, allow ordering by planner, conditions (step‑up, attr_equals, org_matches, time_window), and trace contents.
- Planner/scopes: tests for rule‑based chains, default chain, extractor overrides, unknown scopes ignored.
- Store (gorm): tests for roles/bindings lifecycle, principal versioning; migration coverage.
- Cache: TTL behavior; concurrency.
- Middleware/admin API: either implement minimal helpers or clearly mark as intentionally deferred; remove dead placeholders if superseded.

Cross‑cutting refactors and hygiene
- Remove duplicate logic between `routes.go` closures and `Handlers()` builder; consolidate where feasible.
- Ensure all input structs for HTTP endpoints validate required fields; return typed errors mapped to proper HTTP responses.
- Centralize constants (cookie names, header keys) in one place; avoid magic strings.
- Logging: use structured logs via provided Logger interface; avoid noisy logs on hot paths; add context where helpful.
- Time/clock: use `Deps.Clock` in places relying on time to improve testability.

Performance quick‑wins to evaluate
- Avoid copying large `json.RawMessage` unnecessarily; stream parse where possible.
- Reuse `base64`/`hex` buffers in hot paths; preallocate slices for credential listing.
- Minimize DB round‑trips on login/rotate by batching where possible.
- Cache compiled regexes and policy matchers (already done) and benchmark registry lookups.

Security review focus
- Cookie flags (`Secure`, `HttpOnly`, `SameSite`) and `__Host-*` rules enforced.
- CSRF check required for refresh; document threat model in code.
- Replay detection on refresh; family revoke is atomic and auditable.
- Issuer/audience/time validations with leeway; reject mismatches.
- Step‑up enforced before RBAC; admin bypass limited to roles/permissions only.
- WebAuthn challenge single‑use; constant‑time compares for tokens and invites.

Dependency hygiene
- Keep all app‑specific wiring in package‑root `config.go` and `deps.go`.
- Use interfaces for cross‑package boundaries; avoid importing non‑contract code from other services.
- Where `lib/foundry/httpkit` abstractions are used, keep thin adapters in this package; avoid deep coupling.

Acceptance criteria
- All checklists above satisfied for each subpackage.
- CI green with unit tests; coverage improved on critical paths (authn, rotation, policy, rbac evaluator).
- No dead code flagged by linters; no unused exports; docs present and accurate.
- Benchmarks for token parse and middlewares demonstrate acceptable latency/allocs.
- CONTEXT.md updated if architecture changes; this GOALS.md kept in sync during the hardening effort.

Tracking
- Add tasks to `.ai/TASKS.md` as work items (per subpackage and theme). Mark completed items and discoveries under “Discovered During Work”.



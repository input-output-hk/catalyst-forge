# Foundry API – Auth Overhaul + Certificates Tasks

This task list tracks two consecutive initiatives:
1) Auth overhaul (invite‑based, now baseline)
2) Certificates & build identity feature (current focus)

## Today

- [ ] Create branch `feat/invite-onboarding` and wire config flags (`AUTH_ACCESS_TTL`, `AUTH_REFRESH_TTL`, `INVITE_TTL`, `KET_TTL`, `REGISTRATION_MODE=invite_only`).
- [ ] Provision IAM role for SES (EKS IRSA)
  - [ ] Trust policy allowing `sts:AssumeRoleWithWebIdentity` for `system:serviceaccount:<ns>:<sa>`
  - [ ] Permissions: `ses:SendEmail` (SESv2) on `*` (or restricted identities)
  - [ ] Wire SA annotation `eks.amazonaws.com/role-arn`
  - [ ] Set env: `EMAIL_ENABLED=true`, `EMAIL_PROVIDER=ses`, `SES_REGION`, `EMAIL_SENDER`, `PUBLIC_BASE_URL`
  - [x] Branch created: `feat/invite-onboarding`
  - [x] `INVITE_TTL` wired (`Auth.InviteTTL` + handler default/override)
  - [x] `AUTH_ACCESS_TTL`
  - [ ] `AUTH_REFRESH_TTL`
  - [ ] `KET_TTL`
  - [ ] `REGISTRATION_MODE=invite_only`

## Phase 2 — Schema Migrations

- [x] Add `users.email_verified_at timestamptz`, `users.user_ver int default 1`.
- [x] Create `invites (email, roles, token_hash, expires_at, redeemed_at, created_by, created_at)` + indexes.
- [x] Create `devices (user_id, name, platform, fingerprint, created_at, last_seen_at)` + index.
- [ ] Alter `user_keys` → add `device_id`, `created_at`, `revoked_at`.
  - [x] `device_id`
  - [x] `created_at`
  - [ ] `revoked_at`
- [x] Create `refresh_tokens (user_id, device_id, token_hash, created_at, last_used_at, expires_at, replaced_by, revoked_at)` + indexes.
- [x] Create `revoked_jtis (jti, reason, revoked_at, expires_at)`.

## Phase 3 — Server Changes

- [ ] JWT manager / keys:
  - [x] Add claims `jti`, `user_ver`, `akid`
  - [ ] Reduce access TTL to 30m
  - [x] Implement JWKS endpoint with cache headers (ETag + Cache-Control)
  - [ ] Key rotation
- [x] Middleware: default RequireAll; add `RequireAny`, `RequireAllStepUp`; enforce `iss/aud/alg`, `jti` denylist, `user_ver` freshness.
- [ ] Invites: `POST /auth/invites` (admin) → create + email link.
  - [x] Create endpoint
  - [ ] Email link delivery (Amazon SES)
    - [x] SES email service scaffolding
    - [x] Construct and inject SES client at startup
- [x] Verify: `GET /verify?token=...&invite_id=...` → setup session; set user active (unless policy toggled).
- [ ] Key enrollment: `POST /auth/keys/bootstrap` → KET; `POST /auth/keys/register` with PoP (sign server nonce; verify).
  - [x] Bootstrap KET endpoint (issues short-lived token + nonce)
  - [x] Register with KET (verify KET + PoP, create key active)
- [ ] Device flow: `POST /device/init`, `POST /device/token` (polling with `interval`, `slow_down`).
- [x] Tokens: `POST /tokens/refresh` rotation+reuse detection
 - [x] Tokens: `POST /tokens/revoke` (optional)
- [x] Use HMAC-SHA256 for hashing invite tokens (`invites.token_hash`) and refresh tokens (`refresh_tokens.token_hash`) with a server secret; add config flag and rotateable secret.
  - [x] Invite HMAC via `INVITE_HASH_SECRET`
  - [x] Refresh HMAC via `REFRESH_HASH_SECRET`
  - [ ] Document and add rotation process
- [x] Remove legacy endpoints `POST /auth/users/register` and public `POST /auth/keys/register`.
 - [x] Rate limiting for `/auth/invites`, `/verify`, `/tokens/refresh`.
 - [ ] Rate limiting for `/device/*`.

## Infrastructure Notes

- Envoy/NLB currently masks client IPs in `X-Forwarded-For` with node IPs. The in-process per-IP rate limit is feature-flagged off by default. Plan to fix by adjusting NLB/Envoy (proxy protocol/XFF trust) or moving rate limits to edge.
- [ ] Emit audit events for invites, verification, key adds, token actions.

## Phase 4 — CLI & SDK

- [ ] CLI: implement `forge api device login` (display code/URL, wait for approval; poll with backoff/interval)).
- [ ] CLI: implement `forge api key add` using KET.
- [ ] CLI: implement refresh token storage via OS keyring; encrypted‑file fallback; `--token-store` flag/setting.
- [ ] Client library: implement device flow; token refresh; auto‑retry on `user_ver` mismatch.

## Phase 5 — Tests & Hardening

- [ ] Unit tests: middleware RequireAll semantics; KET lifecycle; token rotation/reuse detection; `user_ver` invalidation; denylist.
- [ ] Integration tests: end‑to‑end invite → verify → key add → token issuance → refresh → rotation → revocation.
- [ ] Security tests: rate‑limit coverage; replay/reuse; device `slow_down` handling.

## Follow‑ups (post‑cutover)

- [ ] Scaffold step‑up endpoints behind flags: `POST /mfa/webauthn/register`, `POST /mfa/webauthn/assert`; `RequireAllStepUp` returns `mfa_required` when enabled.
- [ ] Certificate permission hardening: IDNA normalization, label‑boundary checks, explicit wildcard depth rules, and tests.
- [ ] Improve invite email template (HTML + text): branding, accessibility, localization support, link preview meta, test rendering.

## Discovered During Work

- [ ] HMAC hashing config: `HASH_SECRET` or `INVITE_HASH_SECRET` and `REFRESH_HASH_SECRET`; plumb into repos/handlers.

## Notes

---

# Legacy certificate signing (DEPRECATED)

- The previous, legacy certificate signing flow is deprecated and will be removed.
- It was never deployed to production and is safe to delete.
- All certificate issuance should follow the new Step‑CA based design (clients/servers instances, provisioner JWTs, CSR validators, policy TTLs, auditing, and metrics) as outlined in the planning doc.

# Certificates & Build Identity – Tasks (Updated: AWS PCA)

Aligns with `.ai/feat/certs/OVERVIEW.md`.

## Phase 1 — Scaffolding
- [x] Add config keys for provisioner signer, step‑ca clients/servers, policy, OIDC, CA register (optional).
 - [x] Implement `internal/ca/stepca.go` (StepCAClient x2) with provisioner JWT signing. [scaffolded and wired in server context]
- [x] Implement CSR validators (client/server) with strict rules and helpful errors. [added `internal/ca/validators.go` + integrated in handler]
- [x] Add audit and metrics stubs. [audit repo wired; metrics TBD]

## Phase 2 — GitHub OIDC
- [x] JWKS fetcher with cache; verify `iss`, `aud`, `exp`, `nbf`. [lib client + cache running in server]
- [x] Parse claims (`repository`, `repository_owner`, `run_id`, `job_workflow_ref`, `ref`, `sha`).
- [x] Enforce policy (`GITHUB_ALLOWED_ORGS`, `GITHUB_ALLOWED_REPOS`, `GITHUB_PROTECTED_REFS`).
- [x] Mint job token (no refresh), TTL ≤ job timeout. [JWT expiry clamped to OIDC token expiry]

## Phase 3 — Build sessions
- [x] Migrations: `service_accounts`, `sa_keys`, `build_sessions` (idempotent).
- [x] Handler `POST /build/sessions`; enforce `SESSION_MAX_ACTIVE` per owner; emit audit; expose policy caps.

## Phase 4 — Client cert issuance
- [x] Handler `POST /certificates/sign` (user must have any `certificate:sign:*` permission).
- [x] Validate CSR (client/server rules) with `internal/ca/validators.go`.
- [x] Rate‑limit (per user/repo identity) using in‑memory limiter; configurable `certs_issuance_rate_hourly`.
- [x] TTL clamps (PCA):
  - [x] Client Dev: 90m
  - [x] Client CI: min(job_exp, 120m)
  - [x] Server: 6d
- [x] Call AWS PCA via `PCAClient` (adapter over AWS SDK v2 `acmpca`).
- [x] Emit metrics and audit (PCA):
  - [x] Metrics: `cert_issued_total`, `cert_issue_errors_total`, `pca_issue_latency_seconds`.
  - [x] Audit fields: include `ca_arn`, `template_arn`, `serial`, `sans`, `ttl`.

## Phase 5 — Server cert issuance (SA)
- [x] SA auth path; perms check `certificate:sign:*` (temporary; refine to specific SA perm as needed).
- [x] Handler `POST /ca/buildkit/server-certificates`; validate CSR (server rules); TTL clamp `SERVER_CERT_TTL`.
- [x] Call PCA servers CA; emit metrics and audit.

## Phase 6 — Optional ext_authz
- [x] `POST /build/gateway/authorize` (feature‑flagged). Simple allow based on SAN prefix; return 200/403.

## Phase 7 — Tests
- [x] Unit: validators (good/bad) for client/server CSR rules.
- [x] Unit: OIDC verifier (valid/empty/wrong aud/JWKS empty) in lib context (see `lib/foundry/auth/github`).
- [x] Unit: TTL clamps helper behavior (see `internal/ca/issuance_policy.go`).
- [x] Unit: In‑memory rate limiter behavior (see `internal/rate`).
- [ ] Integration tests (pre‑req: update client library):
  - [ ] Update `lib/foundry/client` to expose new flows/endpoints:
    - [ ] Invite flow: create invite (admin) and verify link handling.
    - [ ] Key Enrollment Token (KET): bootstrap + register.
    - [ ] Device flow: `device/init`, `device/token`, `device/approve` helpers.
    - [ ] GitHub OIDC exchange: helper to validate OIDC token and mint job token.
    - [ ] Build sessions: `POST /build/sessions` helper.
    - [ ] Certificates: client sign `POST /certificates/sign` and server sign `POST /ca/buildkit/server-certificates` helpers.
    - [ ] Optional ext_authz: `POST /build/gateway/authorize` helper (feature‑flagged).
  - [ ] Implement end‑to‑end integration tests using updated client:
    - [ ] Invite → verify → KET register → token issuance → refresh → rotation → revocation.
    - [ ] GHA OIDC exchange → job token → client cert signing (TTL clamp checked) → CA call mocked.
    - [ ] Server cert signing path with TTL clamp, metrics, and audit assertions (using mocks/DB checks).
  - [x] Replace Step‑CA tests with pure mock PCA returning CA and leaf; cover ok + error paths.

## Observability & Ops
- [ ] Metrics: issuance volume/errors/latency; active sessions.
- [x] Audit: include serial, SANs, TTL, provenance fields; structured JSON.
  - [ ] PCA audit fields: `ca_arn`, `template_arn`, `serial`, `sans`, `ttl`.

## Security checklist
- [ ] Provisioner JWT TTL ≤ `PROVISIONER_JWT_TTL` (≤120s).
- [ ] RequireAll semantics; keep any OR exceptions explicit.
- [ ] CSR sanitization enforced; reject client DNS/IP SANs.
- [ ] No PEMs logged; redact sensitive bodies.
- [ ] HL rate limits + per‑owner quotas.
- [ ] Clock skew tolerated; strict `iss/aud` checks.
- [ ] JWKS cache with rotation support.
- [ ] HTTP client timeouts (≤5–10s) and robust PCA SDK timeouts.
- [x] Remove all Step‑CA code and infra from dev/test env (docker-compose).
- Email delivery provider: Amazon SES for invite and verification emails (set up domain identity, DKIM, sandbox → production).
- Server-only hash secrets introduced for HMAC storage (not consumed by CLI): `INVITE_HASH_SECRET`, `REFRESH_HASH_SECRET`. Ensure env documentation and rotation guidance.


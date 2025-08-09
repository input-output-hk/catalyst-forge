# Foundry API – Certificates & Build Identity Plan

This plan defines the next phase for the Foundry API: secure, policy‑driven X.509 issuance for developers and CI, backed by AWS ACM PCA (replacing step‑ca), with provenance binding and audited issuance. It builds on the completed invite‑based auth overhaul.

References:
- `.ai/feat/certs/OVERVIEW.md` (authoritative scope and contracts)
- `.ai/api/registration_overhaul.md`, `.ai/api/response.md` (auth baseline)

## TL;DR

- GitHub OIDC exchange → mint job‑scoped access token (no refresh).
- Build sessions tracked server‑side with concurrency caps and provenance.
- Client cert issuance (Dev/CI) via AWS PCA clients CA; server cert issuance (gateway) via AWS PCA servers CA.
- Strict CSR validators, rate limits/quotas, auditing, and metrics.
- Optional Envoy ext_authz hook for gateway admission.
- Two step‑ca instances (clients/servers) with provisioner JWTs signed by the API.

## Project structure (new/updated packages)

Internal packages to add/extend:
- `internal/config` – env → typed config for certs, OIDC, policy, CA endpoints.
- `internal/security` – tokens, perms, context (user_ver/sa_ver), GH OIDC verify, RequireAll helpers.
- `internal/ca` – StepCAClient (JWT provisioner), CSR validation, signers (clients/servers).
- `internal/http` – handlers/routing for CI exchange, sessions, cert endpoints, optional ext_authz.
- `internal/rate` – limiter interface/impl (in‑mem for dev; pluggable prod).
- `internal/audit` – event emitter (JSON), integrates with existing audit repo model.
- `internal/metrics` – histogram/counters (Prometheus).
- `internal/store` – `service_accounts`, `sa_keys`, `build_sessions` repositories.

## Configuration (env → config)

Add keys per Overview: API base URL, provisioner signer (KID, ES256 key, TTL), step‑ca bases/provisioners, timeouts, policy (TTLs/limits), GH OIDC (iss/aud/allowlists/jwks cache), optional CA register (S3/DDB). Validate at startup; require explicit prod values.

## Data model & migrations

New tables (idempotent):
- `service_accounts (id, name, org_id, status, sa_ver, created_at)`
- `sa_keys (id, sa_id, akid, alg, pubkey_b64, status, created_at, unique(sa_id, akid))`
- `build_sessions (id uuid, owner_type, owner_id, org_id, source, created_at, expires_at, metadata jsonb)` + index on owner

## Security & middleware

- AuthN: verify access/job tokens; attach `sub`, `org_id`, `perms`, and `user_ver` or `sa_ver`.
- AuthZ: RequireAll default. Handlers annotate required perms per endpoint.
- CI tokens: minted by API (no refresh), TTL ≤ job timeout.
- Scrub sensitive headers/bodies (JWTs/CSRs/PEMs) from logs.

## PCA adapter & CSR validation

- `internal/service/pca` provides a `PCAClient` interface and AWS SDK v2 implementation (Issue/Get/GetCA).
- CSR validators:
  - Client (Dev/CI): reject DNS/IP SANs; allow URI SANs with `spiffe://forge/dev|ci/...` patterns; optional email SAN; length limits.
  - Server (gateway): require DNS or IP SAN; optional IDNA normalization.
  - Return 400 with precise reason on invalid CSR.

## HTTP endpoints

- `POST /ci/auth/github/exchange` – verify OIDC, policy checks, mint job token.
- `POST /build/sessions` – create/tracks sessions; enforce per‑owner concurrency.
- `POST /certificates/sign` – client cert issuance (Dev/CI); validate CSR; rate‑limit; TTL clamp; call PCA; audit/metrics.
- `POST /ca/buildkit/server-certificates` – server cert issuance (SA only) via PCA servers CA; CSR rules; TTL clamp.
- (Optional) `POST /build/gateway/authorize` – ext_authz admission.

## Permissions (RequireAll)

| Endpoint                           | Required perms                                         |
| ---------------------------------- | ------------------------------------------------------ |
| `/ci/auth/github/exchange`         | none (OIDC)                                            |
| `/build/sessions`                  | `build:run`                                            |
| `/ca/buildkit/certificates`        | `build:run` AND `certificate:sign:buildkit-client`     |
| `/ca/buildkit/server-certificates` | `certificate:sign:buildkit-server` (SA only)           |
| `/build/gateway/authorize`         | internal allowlist                                     |

## Rate limits & quotas

- Keyed limiter (`issue:user:<id>` or `issue:repo:<org>/<repo>`) with `ISSUANCE_RATE_HOURLY` per 1h window.
- Enforce `SESSION_MAX_ACTIVE` per owner.

## Metrics & auditing

- Metrics: `cert_issued_total`, `cert_issue_errors_total{reason}`, `pca_issue_latency_seconds`, `build_sessions_open`.
- Audit: `ci.oidc.exchanged`, `build.session.created`, `cert.issued`, `servercert.issued` (include serial, SANs, ttl, provenance).

## Implementation phases

1) Scaffolding – config keys; StepCAClient x2; CSR validators; audit/metrics stubs.
2) GitHub OIDC – JWKS cache verify; policy checks; job tokens (no refresh).
3) Build sessions – migration + handler; per‑owner cap.
4) Client cert issuance – handler; provenance; TTL clamp; rate‑limit; step‑ca call; metrics/audit.
5) Server cert issuance – SA auth; CSR rules; TTL clamp; step‑ca servers call; metrics/audit.
6) Optional ext_authz – admission hook; feature flag.
7) Tests – unit (validators, TTLs, policy, limiter); integration with mocked PCA.

## PCA TTL policy

- Client Dev: default 90m (config `certs_client_cert_ttl_dev`)
- Client CI: min(job token exp, 120m default or `certs_client_cert_ttl_ci_max`)
- Server: default 6d (config `certs_server_cert_ttl`)

## PCA audit fields

Include `ca_arn`, `template_arn`, `signing_algo`, `serial`, `sans`, `ttl`, `not_after` in `cert.issued` and `servercert.issued` events.

## Status – Auth overhaul (completed baseline)

- Invite‑based onboarding, KET, device flow, refresh rotation, JWKS, RequireAll middleware delivered. Legacy endpoints removed. See prior sections for details and tests.


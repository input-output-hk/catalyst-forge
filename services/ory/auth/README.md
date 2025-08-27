# Auth Orchestrator

Small Gin-based service that orchestrates Ory Hydra login/consent against Ory Kratos and provides a token hook for mint-time claim enrichment (e.g., GitHub Actions JWT-Bearer).

## Endpoints

- `GET /api/v1/oauth2/login` — handles Hydra `login_challenge`; verifies Kratos session (`/sessions/whoami`), accepts login, or redirects to Kratos login UI
- `GET /api/v1/oauth2/consent` — handles Hydra `consent_challenge`; fail-closed if no Kratos session; maps traits → token claims; accepts consent
- `POST /api/v1/consent` — placeholder for form-based consent (optional)
- `POST /api/v1/hydra/token-hook` — enriches access token claims for mint/refresh (JWT‑Bearer: normalizes GitHub claims)
- `GET /api/v1/health` — liveness
- `GET /api/v1/metrics` — Prometheus metrics

## Run locally

```bash
cd services/ory/auth
go run ./cmd/auth
# Service listens on :8080 by default
```

### With configuration

```bash
AUTH_SERVER_ADDR=":8080" \
AUTH_HYDRA_ADMIN_URL="http://localhost:4445" \
AUTH_KRATOS_PUBLIC_URL="http://localhost:4433" \
go run ./cmd/auth run
```

## Configuration

The service uses Cobra/Viper. Values can be provided via flags or environment variables (prefix `AUTH_`).

- Flags:
  - `--server.addr` (default `:8080`)
  - `--hydra.admin_url` (default `http://hydra-admin:4445`)
  - `--kratos.public_url` (default `http://kratos-public:4433`)
  - `--consent.remember_for` (default `300`)
  - `--consent.scopes` (default `openid,offline`)
  - `--consent.audience` (default empty)

- Environment variables:
  - `AUTH_SERVER_ADDR`
  - `AUTH_HYDRA_ADMIN_URL`
  - `AUTH_KRATOS_PUBLIC_URL`
  - `AUTH_CONSENT_REMEMBER_FOR`
  - `AUTH_CONSENT_SCOPES`
  - `AUTH_CONSENT_AUDIENCE`

### Validation and defaults

- URLs are validated on startup; the service exits on invalid values.
- Consent defaults: `remember_for=300s`, `scopes=[openid, offline]` unless provided.

## Integration with Hydra and Kratos

1) Configure Hydra to call the orchestrator for login/consent and token hook.

- Using ORY CLI (patches):

```bash
ory patch oauth2-config \
  --replace '/urls/login="https://auth.local/api/v1/oauth2/login"' \
  --replace '/urls/consent="https://auth.local/api/v1/oauth2/consent"' \
  --add '/oauth2/token_hook/url="https://auth.local/api/v1/hydra/token-hook"'
```

2) Point the orchestrator at Hydra Admin and Kratos Public:

- Example config: see `examples/auth-orchestrator.example.toml`.

3) Networking/security best practice:

- Expose orchestrator internally; place Istio policy so only Hydra can call `/api/v1/hydra/token-hook` and `/api/v1/oauth2/*`.
- For APIs, prefer Oathkeeper `jwt` authenticator against Hydra JWKS.

## Error responses and correlation

- Unified error shape:

```json
{ "error": "message", "correlation": { "request_id": "...", "login_challenge": "...", "consent_challenge": "..." } }
```

- The request ID is echoed via `X-Request-ID` response header. A per-request log line includes method, URL, status, latency, and correlation IDs.

## Metrics

- Prometheus endpoint: `GET /api/v1/metrics`.
- Counters:
  - `auth_orchestrator_login_accept_total`
  - `auth_orchestrator_consent_accept_total`
  - `auth_orchestrator_login_failure_total`
  - `auth_orchestrator_consent_failure_total`
  - `auth_orchestrator_tokenhook_failure_total`

## Examples

- `examples/auth-orchestrator.example.toml` — orchestrator config sample
- `examples/hydra.example.yaml` — Hydra config snippet mapping URLs and token hook

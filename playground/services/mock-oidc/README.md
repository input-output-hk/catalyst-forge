# Mock OIDC Provider

A lightweight, multi-issuer OpenID Connect server for local development and CI. It is configurable via YAML, supports public and confidential clients, PKCE, multiple personas, and emits realistic discovery, token, userinfo, and JWKS endpoints.

## Features

- Multiple providers under a single base URL (e.g. `/google`, `/default`).
- Public and confidential clients (basic/post client auth for confidential).
- Authorization code flow (with PKCE S256; optional plain), implicit ID token for testing.
- Scope-gated userinfo (`email`, `profile`).
- Deterministic JWKS when a PEM signing key is provided.
- Personas with arbitrary extra claims (e.g., `hd`) merged into ID tokens and userinfo.
- Optional refresh token issuance (see policy).

## Quick start

Prerequisites: Docker, Earthly, and (optionally) `just`.

```bash
# From playground/services/mock-oidc
just up          # build container via Earthly and start with docker compose
just logs        # tail logs
just down        # stop
```

Health check: `GET /healthz`

## Configuration

The server reads YAML from the path in `CONFIG_PATH`.

Example (`mock-oidc.config.example.yaml`):

```yaml
listen_addr: ":8080"
base_url: "http://localhost:8080"
global_secret: "0123456789abcdef0123456789abcdef" # 32 bytes
allow_pkce_plain: false
providers:
  - id: "default"
    public: true
    client:
      id: "kratos-client"
      secret: "kratos-secret"
      redirect_uris: ["http://localhost:18080/callback"]
    personas:
      default:
        sub: "00000000-0000-0000-0000-000000000001"
        claims:
          email: "test@example.com"
          email_verified: true
          name: "Test User"
          hd: "example.com"
  - id: "google"
    public: false
    client:
      id: "google-client"
      secret: "google-secret"
      redirect_uris: ["http://localhost:18080/callback"]
    personas:
      acme:
        sub: "11111111-1111-1111-1111-111111111111"
        claims:
          email: "user@acme.com"
          email_verified: true
          name: "G Suite User"
          hd: "acme.com"
```

Notes:
- `global_secret` must be exactly 32 bytes.
- `base_url` is used to render discovery and endpoint URLs.
- Each provider lives under `/{id}`.

## Endpoints

For provider `{id}` at `{base_url}`:
- Discovery: `{base_url}/{id}/.well-known/openid-configuration`
- Authorization: `{base_url}/{id}/authorize`
- Token: `{base_url}/{id}/token`
- Userinfo: `{base_url}/{id}/userinfo`
- JWKS: `{base_url}/{id}/jwks`

Global convenience redirects:
- `/.well-known/openid-configuration` → first provider’s discovery
- `/.well-known/jwks.json` → first provider’s JWKS

## Personas and extra claims

- Define personas and claims per provider in YAML.
- The server merges persona claims into the ID token’s `extra` and exposes them from `/userinfo` when appropriate.
- Example: the `hd` (hosted domain) claim is included in ID tokens and returned by `/userinfo` when scope includes `profile`.

## PKCE and client types

- PKCE S256 is supported by default; plain can be toggled globally and/or per-provider via config.
- Public clients: discovery advertises `token_endpoint_auth_methods_supported: ["none"]`; token endpoint accepts `client_id` only.
- Confidential clients: discovery advertises basic and post methods; token endpoint requires client auth.

## Refresh token policy

- Public clients: never receive refresh tokens.
- Confidential clients: receive refresh tokens only when scope includes `offline_access`.
- The server enforces the policy when writing token responses.

## Kratos integration

Point Kratos to the provider’s issuer (discovery resolves all endpoints):

- Issuer: `https://oidc.projectcatalyst.dev/google`
- Discovery: `https://oidc.projectcatalyst.dev/google/.well-known/openid-configuration`

## Testing

Run the test suite:

```bash
go test ./integration -v
```
- Global well-known redirects
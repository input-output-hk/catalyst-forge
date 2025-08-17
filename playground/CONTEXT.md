## Catalyst Forge – Local Integration Context

This playground composes the Frontend and API behind a local HTTPS edge to mimic production routing.

### Topology
- Edge (Caddy): `https://forge.localhost`
  - Routes `/api/*` → API service (strip `/api` prefix)
  - Routes all other paths → Frontend
- API: listens on `:5050` inside the network
- Frontend: served behind the edge (or Vite dev server in dev profile)
- Postgres + pgAdmin for API storage

### Auth model (API)
- Endpoints under `/api/v1/auth/*`:
  - `login/begin`, `login/complete` (WebAuthn/Passkeys)
  - `refresh` (HttpOnly `__Host-refresh_token` cookie + double‑submit CSRF)
  - `logout`, `logout-all`, `credentials/*`, `step-up/*`, `me`
- Responses include `access_token` (JWT). Refresh cookie is rotated on each refresh.
- CSRF header: `X-CSRF-Token` must match `__Host-csrf_token` cookie for browser refresh.

### TypeScript client (`services/clients/ts`)
- `FoundryClient` wraps `openapi-fetch` with an auto‑auth fetch:
  - Sends `Authorization: Bearer <access_token>` from in‑memory store
  - On 401, POSTs to `/api/v1/auth/refresh` with CSRF header/cookie, updates token, retries safe requests
- WebAuthn helpers call `/api/v1/auth/login/*`, `/credentials/*`, `/step-up/*`.

### Base URL and routing
- Use `VITE_API_URL = https://forge.localhost` so absolute API paths like `/api/v1/...` resolve via the edge to the API.
- The edge strips the `/api` prefix and proxies to the API container on `api:5050`.

### Profiles
- `prod` profile: edge + API + built frontend
- `dev` profile: edge‑dev + API + Vite dev server (`frontend-dev`)

### Commands
- See `playground/README.md` and `.justfile` for:
  - `just up` (bring up stack)
  - `just down` (tear down)
  - `just logs` (tail logs)

### Certificates and hosts
- Generate certs with `mkcert` into `playground/.certs/`: `mkcert forge.localhost`
- Add hosts entry: `127.0.0.1 forge.localhost forge`

### Integration expectations
- Frontend reads `import.meta.env.VITE_API_URL` to construct the client.
- Access token is memory‑only; refresh is cookie‑based; CSRF handled automatically by the client.
- API health: `GET https://forge.localhost/healthz` (edge proxies to API root)




# Catalyst Forge Frontend

A fast SvelteKit (Svelte 5) app for Project Catalyst. This app currently uses mocked data and a mocked auth flow; real API integration is deferred (see ADR below).

## Quick start

- Prerequisites
  - Node 22 (we use nodenv locally). Example:
    - `nodenv install -s 22.12.0 && nodenv local 22.12.0 && nodenv rehash`
  - npm (bundled with Node)

- Install
  - `npm install`

- Develop
  - `npm run dev`
  - Open `http://localhost:5173`
  - For API dev (TLS proxy recommended):
    - Generate certs: `cd foundry/api/.certs && mkcert api.localhost`
    - Start compose with proxy profile: `docker compose -f foundry/api/docker-compose.yml --profile proxy up -d`
    - Set `VITE_API_URL=https://api.localhost`

- Lint & format
  - `npm run format`
  - `npm run lint`

- Tests
  - End-to-end (recommended): `npm run e2e:dev`
  - Note: browser unit tests are being stabilized; E2E is the source of truth for now.

- Analyze bundle (gzip budgets)
  - `npm run analyze`
  - Budgets: JS ≤ 80KB, CSS ≤ 10KB (gzip) for initial route

## Runtime configuration

- API base URL
  - The frontend reads the API base URL at runtime from `FORGE_API_BASE_URL`.
  - Default (for local docker-compose): `http://127.0.0.1:5050`.
  - You can also provide `VITE_API_URL` for compatibility, but prefer `FORGE_API_BASE_URL` in container/k8s deployments to avoid build-time coupling.
  - Example (macOS/Linux): `FORGE_API_BASE_URL=http://127.0.0.1:5050 npm run dev`

## Structure

- `src/routes/`
  - `/` dashboard with KPI cards and recent activity (real API with mock fallbacks)
  - `/login`, `/auth/callback`, `/logout` (mocked passwordless flow)
  - `/releases` list (server-side loaded; mock fallback on 401/403)
  - `/releases/[id]` detail + deployments (mock fallback on 401/403)
  - `/verify` invite verification (real endpoint)
  - `/auth/device` and `/auth/device/approve` (device authorization flow; partial wiring)
  - `/tokens` brand tokens visualization
- `src/lib/ui/` minimal UI primitives (Button, Input, Card, Modal, Drawer, Toast)
- `src/lib/mocks/` deterministic mock client
- `src/lib/query.ts` TanStack Query client

## Security & cookies (mock)

- Mock auth sets a non-sensitive `cf_session` cookie with an email. Real tokens/cookies will be `HttpOnly`, `Secure`, `SameSite=Lax` and set in the server callback.

## Command palette

- Press Cmd/Ctrl+K to open a simple command palette (Dashboard, Releases, Profile, Approve Device).

## ADRs

- `docs/decisions/ADR-0001-defer-api-integration.md`
- `docs/decisions/ADR-0002-real-api-and-releases.md`

## Notes

- Svelte 5 runes are used in components/routes. Some legacy warnings may appear in dev builds; these do not affect functionality.

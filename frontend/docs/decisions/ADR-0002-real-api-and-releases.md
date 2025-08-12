# ADR-0002: Real API integration (runtime-configured) and Projects→Releases rename

Status: Accepted

Date: 2025-08-10

Context

- Initial frontend shipped with mocked data and a "Projects" concept.
- The backend API (OpenAPI at `foundry/api/docs/swagger.yaml`) uses Releases and Deployments.
- We need to progressively integrate the real API while maintaining developer velocity.

Decision

1. Runtime API client
   - Use `openapi-typescript` to generate `src/lib/api/types.ts`.
   - Use `openapi-fetch` to create a typed client in `src/lib/api/client.ts`.
   - Base URL resolved at runtime via `getApiBaseUrl()` (`FORGE_API_BASE_URL`, default `http://127.0.0.1:5050`).
2. Auth flows (progressive)
   - Keep mock login for local productivity.
   - Implement verify and device-auth routes that call real endpoints; when unauthorized (401/403), show clear UI and proceed with mock fallbacks.
   - Session cookie will be `HttpOnly; SameSite=Lax; Secure` in non-dev once access tokens land.
3. Rename Projects → Releases
   - `/releases` replaces `/projects` using real `GET /releases` with a server-side loader.
   - Detail route `/releases/[id]` shows deployments via `GET /release/{id}/deployments`.
4. Security
   - CSP configured in `svelte.config.js`.
   - Server security headers added in `src/hooks.server.ts`.

Consequences

- Local dev uses server-side `load` to avoid CORS and to provide mocked fallbacks on 401/403.
- E2E tests use the mock login to satisfy route guards while we complete the auth flows.
- Clear TODO comments and `mocked: true` flags mark all temporary code for removal.

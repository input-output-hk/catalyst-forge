## Frontend TODO (from code review)

Actionable items to improve DRYness, best practices, organization, readability, and consistency.

### DRY: extract reusable logic from `src/pages/Users.tsx`
- [ ] Move small UI pieces into components:
  - [ ] `StatusBadge`, `RoleBadge`
  - [ ] `Confirm` dialog, `RowActions`
  - [ ] Header underline element (generic, e.g. `SectionUnderline`)
  - [ ] Table toolbar (filters/search/density), pagination bar
  - [ ] Right-side user details sheet (credentials, security, audit)
- [ ] Move helpers to feature utils:
  - [ ] `firstLast`, `makeId`, CSV export
- [ ] Create feature folder: `src/features/users/{components,hooks,utils,api,types}.ts`

### Data fetching and state
- [ ] Standardize on TanStack Query for server data
  - [ ] Users list (filters/paging/sort)
  - [ ] User credentials and audit
  - [ ] Invites and access-requests
  - [ ] Define query keys and mutations; use optimistic updates where appropriate
- [ ] Keep `store/app-store.tsx` for app/session/flags only; avoid storing server lists there

### API client and auth
- [x] Use generated client via `forge-client` alias; add sync script to vendor runtime and types
- [x] Import OpenAPI types from vendored client and replace inline assertions
- [ ] Remove remaining `apiFetch` usages and hard-coded URLs (migrate to `forge.raw`)

### Toasts (pick one system)
- [ ] Decide between Radix toast (`components/ui/toast`, `Toaster`) and Sonner (`components/ui/sonner`, `Sonner`)
- [ ] Remove the unused one; update imports and providers in `src/App.tsx`

### Page titles and metadata
- [ ] Ensure every page uses `usePageTitle(title, description?, path?)` like `Dashboard`

### Consistency
- [ ] Build API URLs uniformly via `new URL(path, getApiBaseUrl())`
- [ ] Align domain data: use API + Query for real pages; keep `fixtures` for dev/mock only

### Users feature split (proposed structure)
- [ ] `src/features/users/components/StatusBadge.tsx`
- [ ] `src/features/users/components/RoleBadge.tsx`
- [ ] `src/features/users/components/Confirm.tsx`
- [ ] `src/features/users/components/RowActions.tsx`
- [ ] `src/features/users/components/UserDetailsSheet.tsx`
- [ ] `src/features/users/components/UsersToolbar.tsx`
- [ ] `src/features/users/components/UsersTable.tsx`
- [ ] `src/features/users/hooks/useUsersFilters.ts`
- [ ] `src/features/users/hooks/useUsersSelection.ts`
- [ ] `src/features/users/hooks/useUsersShortcuts.ts`
- [ ] `src/features/users/api/queries.ts`
- [ ] `src/features/users/types.ts`
- [ ] Update `src/pages/Users.tsx` to compose from the above

### Minor improvements
- [ ] Extract keyboard shortcut logic from `Users` into `useUsersShortcuts`
- [ ] Convert complex inline async handlers to named functions
- [ ] Consider a shared `DataTable` abstraction for selection/sort/pagination

### Unify frontend API usage with generated client
- Context: use the generated/wrapper client at `services/clients/ts/src/api/client.ts` (exposed to the frontend via Vite alias `forge-client` or a vendored bundle) instead of ad‑hoc `fetch` in `src/lib/api.ts`.

- [ ] Decide client import path for the browser
  - [ ] Prefer `import { ForgeClient } from "forge-client"` via Vite alias to `vendor/forge-client/index.mjs`
  - [ ] Ensure build/publish step keeps `vendor/forge-client/index.mjs` up to date with `services/clients/ts` output
  - [ ] If alias not available in some envs, add fallback import notes in `README.md`

- [ ] Add a single client instance
  - [ ] Create `src/lib/client.ts` that exports a singleton `forge`:
    - Base URL from `getApiBaseUrl()`
    - `autoAuth: true` with `credentials: 'include'` (handled by client’s `createAutoAuthFetch`)
    - Include `errorHandlingMiddleware` (already default) and optional `loggingMiddleware` in development
  - [ ] Provide helper wrappers: `get`, `post`, `patch`, `del` that call `forge.raw.METHOD(path, { params, body })` and return typed data

- [ ] Replace ad‑hoc API helpers
  - [ ] Migrate `src/lib/api.ts` to keep only `getApiBaseUrl()` and cookie utilities (or remove entirely if no longer needed)
  - [x] Replace many `apiFetch` usages with `forge.raw.*` calls returning typed results (Profile, RegisterRequestForm, Bootstrap partial)
  - [x] Replace `apiFetch` usages in `InviteLanding` and `AuditLog`
  - [ ] Replace remaining `apiFetch` usages in `Profile`
  - [ ] Replace manual `logoutEverywhere` with `forge.raw.POST('/api/v1/auth/logout')`

- [ ] Centralize auth/me and refresh logic
  - [ ] Use `forge.raw.GET('/api/v1/auth/me')` in `RequireAuth` with proper response types from OpenAPI
  - [ ] Rely on the client’s AutoAuth to perform refresh when 401 occurs, rather than manual retry logic in components
  - [ ] On explicit login flows that yield an access token, call `forge.setAccessToken(token)` (if token-based mode is used)

- [ ] Type safety via OpenAPI types
  - [ ] Import `type paths` from `forge-client` and use `paths`-derived inference for inputs/outputs
  - [ ] Remove `any` in `RequireAuth` and `Users` mapping; use exact response shapes
  - [ ] Audit and replace inline type assertions with generated OpenAPI schema types
    - Prefer `components["schemas"][...]` or `paths["/route"]["method"]["responses"][status]["content"]["application/json"]`
    - Prefer vendored types via `import type { paths } from "forge-client"`; avoid `as any` / `as unknown as`
    - If mismatches are found, update the OpenAPI schema or server/DTOs so the schema remains the source of truth

- [ ] React Query integration
  - [x] Introduce auth query wrappers in `src/features/auth/api/queries.ts`
  - [ ] Introduce feature query functions for Users; remove manual `useEffect` fetches
  - [ ] Add a global `QueryClient` `onError` handler to surface `(response as any).errorMessage` via the chosen toast system

- [ ] Remove hard-coded paths scattered in pages
  - [ ] `Users` list → `GET /api/v1/admin/users` with query params (q, role, limit, offset)
  - [ ] User suspend/enable → `PATCH /api/v1/admin/users/{id}`
  - [ ] Credentials → `GET /api/v1/admin/users/{id}/credentials`
  - [ ] Audit → `GET /api/v1/admin/audit?user_id=...&limit=...`
  - [ ] Invites → `POST /api/v1/admin/invites`
  - [ ] Access requests → `GET /api/v1/admin/access-requests`, `PATCH /api/v1/admin/access-requests/{id}`

- [ ] Tooling and build
  - [ ] Add a script/Justfile task to (re)build and sync the TS client into `services/frontend/vendor/forge-client/index.mjs`
  - [ ] Document usage in `services/frontend/README.md` and note the Vite alias in `vite.config.ts`
  - [ ] Optional: ESLint rule or code search check to prevent new usages of `fetch("/api/...")` or `apiFetch(` in `src/`

- [ ] Incremental migration plan
  - [ ] Introduce client and migrate `Users` feature first (list, details, audit, invites)
  - [ ] Migrate `AuthFlows`, `Profile`, `Settings`, etc.
  - [ ] Remove deprecated helpers after all usages are migrated

### Standardize on OpenAPI auth endpoints (per @schema.d.ts)
- Use `ForgeClient` with `paths` types for all auth calls; remove hard-coded strings and manual `fetch`.

- [ ] Session + identity
  - [ ] `RequireAuth.tsx`: replace manual `apiFetch` with `forge.raw.GET('/api/v1/auth/me')` using `components["schemas"]["auth.MeResponse"]`
  - [ ] Global refresh: rely on AutoAuth; if explicit needed, `forge.raw.POST('/api/v1/auth/refresh', { body: {} })`
  - [ ] Logout: `forge.raw.POST('/api/v1/auth/logout')`; logout-all (if used): `forge.raw.POST('/api/v1/auth/logout-all')`

- [ ] Login / Step-up (WebAuthn)
  - [ ] Begin login: `POST /api/v1/auth/login/begin` → `auth.PublicKeyOptionsResponse`
  - [ ] Complete login: `POST /api/v1/auth/login/complete` → `auth.LoginCompleteResponse`
  - [ ] Step-up begin: `POST /api/v1/auth/step-up/begin` → `auth.PublicKeyOptionsResponse`
  - [ ] Step-up complete: `POST /api/v1/auth/step-up/complete` → `auth.AccessTokenResponse`

- [ ] Onboarding / invites
  - [ ] Bootstrap admin: `POST /api/v1/auth/bootstrap` → `auth.BootstrapResponse` (page: `Bootstrap.tsx`)
  - [ ] Onboard begin: `POST /api/v1/auth/onboard/begin` → `auth.OnboardBeginResponse`
  - [ ] Onboard complete: `POST /api/v1/auth/onboard/complete` → 204
  - [ ] Invite preview (public): `GET /api/v1/admin/invites/preview` → `auth.InvitePreviewResponse`
  - [ ] Invite create (admin): `POST /api/v1/admin/invites` → `auth.AdminInviteCreateResponse` (used in `Users`)

- [ ] Recovery codes
  - [ ] Self-generate: `POST /api/v1/auth/recovery/codes/generate` → `auth.RecoveryGenerateResponse` (e.g., `Profile`)
  - [ ] Admin generate for a user: `POST /api/v1/admin/users/{id}/recovery/codes/generate` (used in `Users` sheet)
  - [ ] Recovery flow: init `POST /api/v1/auth/recovery/init` → `auth.RecoveryInitResponse`, verify `POST /api/v1/auth/recovery/verify` → `auth.RecoveryVerifyResponse`, register begin/complete endpoints

- [ ] Credentials (WebAuthn devices)
  - [ ] List: `GET /api/v1/auth/credentials` → `auth.CredentialsListResponse`
  - [ ] Delete: `DELETE /api/v1/auth/credentials/{credentialId}`
  - [ ] Rename: `PATCH /api/v1/auth/credentials/{id}`
  - [ ] Add begin: `POST /api/v1/auth/credentials/add/begin` → `auth.PublicKeyOptionsResponse`
  - [ ] Add complete: `POST /api/v1/auth/credentials/add/complete` → 204

- [ ] Devices and device link (for CLI)
  - [ ] Devices list/get/delete: `/api/v1/auth/devices`, `/api/v1/auth/devices/{id}`
  - [ ] Device link begin/exchange/verify/authorize: `/api/v1/auth/device-link/*`

- [ ] Access requests (admin)
  - [ ] List: `GET /api/v1/admin/access-requests` → `auth.AccessRequestListResponse` (used in `Users` Requests tab)
  - [ ] Decide: `PATCH /api/v1/admin/access-requests/{id}` with `auth.AccessRequestDecideRequest`

- [ ] Implementation tasks
  - [ ] Create `src/features/auth/api/queries.ts` with typed wrappers around the above paths
  - [ ] Update `AuthFlows.tsx`, `Profile.tsx`, `SettingsProfile.tsx`, `RegisterRequestForm.tsx`, and `RequireAuth.tsx` to use these wrappers
  - [ ] Remove remaining references to `apiFetch('/api/v1/auth/...')` and inline URL construction
  - [ ] Add ESLint rule/search guard to block new `'/api/v1/auth/'` string literals in `src/`

### References (files touched)
- `src/pages/Users.tsx`
- `src/components/auth/RequireAuth.tsx`
- `src/lib/api.ts`
- `src/App.tsx`
- `src/hooks/use-page-title.tsx` (ensure usage across pages)

### Notes
- Vite config, routing guards, UI component patterns, and overall structure are solid. The largest wins are consolidating to a single toast system, adopting React Query for server data, centralizing 401 handling, strengthening typings, and splitting the oversized `Users` page into a coherent feature module.



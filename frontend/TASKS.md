### IMPORTANT: Validation flow

1. User instructs to proceed with tasks
2. Agent FULLY finishes task as described in doc
3. Agent VERIFIES task has been completed
4. Agent ASKS user for review
5. User confirms
6. Agent CHECKS OFF task and continues with the next task
7. Repeat

---

### Frontend Auth Bootstrap — Task Plan (Today)

**Goal**: In local test environment, use a known bootstrap token to create an admin invite, complete device registration, and obtain an access token + refresh cookie so the frontend can access protected API endpoints using the real auth flow.

#### Scope for today
- [x] Add `/bootstrap` page with a simple form to submit the bootstrap token
- [x] Call `POST /auth/bootstrap` with configured bootstrap email and provided token
- [x] Start device flow: `POST /auth/devices/init` with `{ token, invite_id }`
- [ ] Generate ES256 device key (non-extractable, WebCrypto) and persist in IndexedDB
- [ ] Build registration proof (ASN.1 DER ECDSA over canonical string) and call `POST /auth/devices/register`
- [ ] Store access token in memory; rely on refresh cookie (HttpOnly)
- [x] Implement refresh using device-bound proof headers; logout similarly
- [ ] Provide a minimal protected call from the UI to verify access (e.g., list users)
- [ ] Add unit tests for device key + proof builders + refresh wrapper
- [ ] Add minimal docs (README snippet) and configuration notes
  - [ ] Update README with dev proxy and bootstrap steps

---

### Endpoints and contracts (backend-aligned)
- `/auth/bootstrap` (POST) → body: `{ email, bootstrap_token }` → returns `{ id, token }`
- `/auth/devices/init` (POST) → body: `{ token, invite_id }` → returns `{ device_id, challenge, alg, expires_at }`
- `/auth/devices/register` (POST) → body: `{ device_id, device_name, public_key_jwk, device_proof, timestamp }` → sets RT cookie; returns `{ access_token, expires_in, user }`
- `/auth/refresh` (POST) → headers: `X-Device-Id`, `X-Device-Proof` (timestamp.base64url(r||s)) → returns `{ access_token, expires_in }`
- `/auth/logout` (POST) → headers: `X-Device-Id`, `X-Device-Proof` (over POST /auth/logout) → 204

---

### Detailed tasks

#### 1) Page: `/bootstrap`
- [x] Form: input for bootstrap token (string) and admin email; submit button
- [ ] On submit:
  - [x] POST `/auth/bootstrap` with `{ email: <form email>, bootstrap_token: <form token> }`
  - [x] Receive `{ id, token }` → proceed to device init
  - [x] POST `/auth/devices/init` with `{ token, invite_id: id }`
  - [ ] Trigger device registration SDK (below)

#### 2) SDK: storage + keys (IndexedDB)
- [ ] Tiny IndexedDB helper (`forge-auth` db, `device` store)
- [ ] `deviceKey.getOrCreate(nameHint)`
  - Generate ES256 non-extractable private key; export pub JWK
  - Persist `{ deviceId, key, pubJwk, name }`

#### 3) SDK: proof builders
- [ ] Registration proof (ASN.1 DER ECDSA):
  - Canonical: `"DEVICE-REGISTER\n<ts>\n<device_id>\n<challenge>"`
  - Build `timestamp.base64url(der(sig))`
- [ ] Refresh/logout proof (raw r||s):
  - Canonical: `"AUTH-REFRESH\n<ts>\n<device_id>\n<origin-host>\n<method> <path>"`
  - Build `timestamp.base64url(r||s)` (64 bytes)
  - Use host (e.g., `new URL(API_BASE).host`) for origin portion

#### 4) SDK: flows
- [ ] `initFromBootstrapToken(bootstrapToken: string)`
  - Calls `/auth/bootstrap` then `/auth/devices/init`
  - Invokes device register; stores AT in memory; leaves RT in cookie
- [ ] `registerDevice(initResp)`
  - Get/create device; build proof; POST `/auth/devices/register`
  - Save tokens in memory `{ accessToken, exp }`
- [ ] `ensureAccessToken()`
  - [x] If exp is near, refresh: POST `/auth/refresh` with headers `X-Device-Id` and `X-Device-Proof`
  - [x] On success, rotate AT in memory
- [ ] `withAuth(fetch)` wrapper
  - [x] Attach `Authorization: Bearer <AT>`; on 401, try one refresh then retry once
- [ ] `logout({ revokeDevice? })`
  - [x] POST `/auth/logout` with proof; clear memory; optionally clear IndexedDB record

#### 5) Minimal protected call (verification)
- [x] Add small UI on success (post-register) to call a protected endpoint (e.g., `GET /auth/users`) using `withAuth(fetch)` and render a count/list

#### 6) Configuration & environment
- [ ] `NEXT_PUBLIC_API_BASE` → API base URL
- [ ] `NEXT_PUBLIC_BOOTSTRAP_EMAIL` → email to pass to `/auth/bootstrap`
- [ ] Ensure all cookie-reliant calls use `credentials: 'include'`

#### 7) Tests (today)
- [ ] Unit: device key generation and `signCanonical` variants produce base64url and differ with input
- [ ] Unit: `ensureAccessToken()`
  - Valid AT returns without network
  - Expired AT + successful refresh sets new AT
  - Expired AT + failed refresh returns null
- [ ] Mock network calls for init/register/refresh/logout

#### 8) Docs
- [ ] Add README section: local bootstrap + device registration steps, required env vars, limitations

---

### Acceptance criteria
- After entering a valid bootstrap token on `/bootstrap`, the app:
  - Creates an admin invite
  - Completes device registration
  - Stores AT in memory and can successfully call at least one protected endpoint using `withAuth` (refresh-capable)
- Keys are non-extractable; RT remains only in HttpOnly cookie; all cookie-dependent calls pass `credentials: 'include'`

### Out of scope (today)
- Magic link UI and email code entry
- Multi-device management UI
- Deep permission-aware UI; we’ll only verify protected access minimally

---

### Notes
- Aligns with backend tests: registration proof uses ASN.1 DER; refresh/logout use raw r||s and host component in canonical string
- Invite `id` and `token` come from `/auth/bootstrap` response



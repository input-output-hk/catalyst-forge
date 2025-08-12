# Authentication Package v1 Implementation Tasks

This task list outlines the complete implementation path for the v1 authentication package based on the specifications in `.ai/api/AuthSpec.md` and `.ai/api/AuthPkg.md`.

## Implementation Order

Tasks should be completed in sequence as each builds upon the previous. Each high-level task includes acceptance criteria that must be validated before proceeding to the next phase.

---

## 📦 1. Set up package structure and core interfaces ✅

**Acceptance Criteria**: All interfaces compile, domain models match spec, no external dependencies in interfaces

- [x] Create /authkit module with go.mod
- [x] Set up directory structure (/authkit, /domain, /service, /store, /crypto, /httpkit, /rate)
- [x] Define domain entities (User, Credential, Invite, RecoveryCode, RefreshToken)
- [x] Create storage interfaces (UserStore, CredentialStore, InviteStore, RecoveryCodeStore, RefreshStore, AuditStore, ChallengeStore, KV)
- [x] Define Config and Deps structures
- [x] Create Manager interface and AuthContext

---

## 🗄️ 2. Implement storage layer with in-memory and GORM adapters ✅

**Acceptance Criteria**: All stores have both in-memory and GORM implementations, unit tests pass, proper indexing on GORM models

- [x] Create in-memory implementations for all stores (testing/inmemory)
- [x] Define GORM models matching domain entities
- [x] Implement GORM UserStore with session versioning
- [x] Implement GORM CredentialStore with sign count tracking
- [x] Implement GORM InviteStore with attempt tracking and HMAC validation
- [x] Implement GORM RefreshStore with family tracking and rotation
- [x] Implement GORM RecoveryCodeStore with SHA256 hashing
- [x] Create challenge store with TTL support (memory/Redis)

---

## 🔐 3. Implement crypto and token services ✅

**Acceptance Criteria**: JWT signing/verification works, JWKS exposes public keys, refresh rotation prevents replay attacks

- [x] Create KeyManager for ES256 JWT signing/verification
- [x] Implement JWKS endpoint support with key rotation
- [x] Create TokenService with AccessClaims structure
- [x] Implement secure random generator (Rand interface)
- [x] Create hash utilities for HMAC-SHA256 and SHA256
- [x] Implement RefreshService with token rotation and family tracking
- [x] Add replay detection for refresh tokens

---

## 🔑 4. Implement WebAuthn service ✅

**Acceptance Criteria**: WebAuthn ceremonies complete successfully, admin AAGUID enforcement works, challenges expire correctly

- [x] Integrate go-webauthn library
- [x] Implement BeginRegistration with attestation handling (none for users, direct for admins)
- [x] Implement FinishRegistration with AAGUID validation for admins
- [x] Implement BeginLogin (username-less flow)
- [x] Implement FinishLogin with sign count and UV flag validation
- [x] Implement step-up authentication flow (BeginStepUp/FinishStepUp)
- [x] Add challenge storage with TTL enforcement

---

## 🍪 5. Implement HTTP utilities and cookie management ✅

**Acceptance Criteria**: Cookies have correct security flags, CSRF protection blocks requests without header

- [x] Create cookie helpers with __Host- prefix support
- [x] Implement CSRF protection with X-Requested-With header
- [x] Create consistent JSON response helpers
- [x] Add enumeration-safe error responses

---

## 🛣️ 6. Implement auth routes and handlers

**Acceptance Criteria**: All routes respond correctly, proper HTTP status codes, enumeration-safe responses

- [ ] Create onboarding routes (POST /auth/onboard/begin, /auth/onboard/complete)
- [ ] Create login routes (POST /auth/login/begin, /auth/login/complete)
- [ ] Create refresh route with CSRF validation (POST /auth/refresh)
- [ ] Create logout routes (POST /auth/logout, /auth/logout-all)
- [ ] Create step-up routes (POST /auth/stepup/begin, /auth/stepup/complete)
- [ ] Create invite management route (POST /auth/invites)
- [ ] Add credential management routes (POST /auth/credentials/add/begin, /auth/credentials/add/complete)
- [ ] Implement JWKS route (GET /.well-known/jwks.json)

---

## 🛡️ 7. Implement middleware and policy system ✅

**Acceptance Criteria**: Policies enforce correctly, 401/403/428 responses as appropriate, AuthContext available in handlers

- [x] Create Authenticate middleware (JWT parsing, AuthContext injection)
- [x] Implement PolicyRegistry with path/method matching
- [x] Create EnforcePolicies middleware with rule evaluation
- [x] Add RequireAuth helper middleware
- [x] Add RequireStepUp helper middleware
- [x] Implement session version validation in middleware

---

## 🔄 8. Implement recovery and step-up services ✅

**Acceptance Criteria**: Recovery codes work once only, step-up grants expire correctly

- [x] Create RecoveryService for code generation
- [x] Implement recovery code consumption with single-use enforcement
- [x] Create StepUpService with TTL-based grants
- [x] Add recovery flow routes (POST /auth/recovery/init, /auth/recovery/verify, /auth/recovery/register/*)

---

## ⚡ 9. Implement rate limiting ✅

**Acceptance Criteria**: Rate limits enforce correctly, 429 responses with retry-after header

- [x] Create Limiter interface with Allow method
- [x] Implement memory-based rate limiter
- [x] Add Redis-based rate limiter (optional for v1)
- [x] Integrate rate limiting into auth routes per spec:
  - Login: 5 attempts per 15 minutes
  - Invite: 5 attempts per lifetime
  - Recovery init: 3 attempts per hour
  - Recovery verify: 5 attempts per hour
  - Refresh: 100 attempts per hour
  - Credentials add: 10 attempts per hour

---

## 📊 10. Add audit logging

**Acceptance Criteria**: All security events logged, no PII in logs

- [ ] Define audit event types
- [ ] Implement AuditStore
- [ ] Add audit logging to all auth operations

---

## ✅ 11. Integration testing and validation

**Acceptance Criteria**: All user flows from spec work end-to-end, security features enforce correctly, no handler modifications needed

- [ ] Create integration test harness
- [ ] Test complete onboarding flow (invite → registration → login)
- [ ] Test refresh token rotation and replay detection
- [ ] Test session invalidation (logout-all, version bump)
- [ ] Test step-up authentication flow
- [ ] Test recovery flow
- [ ] Test admin hardware key enforcement
- [ ] Test rate limiting behavior
- [ ] Test CSRF protection

---

## Key Implementation Notes

### Security Requirements
- All tokens must be time-limited
- Refresh tokens stored as SHA256 hash only
- Constant-time comparisons for all secret validation
- Access tokens never stored in persistent storage (memory only)
- CSRF protection via SameSite=Strict + custom headers
- WebAuthn provides phishing resistance by design

### Configuration Requirements
- `INVITE_HASH_SECRET`: HMAC key for invite token hashing (REQUIRED)
- `JWT_SIGNING_KEY`: ES256 private key for signing JWTs (REQUIRED)
- `WEBAUTHN_RP_ID`: Relying party ID (e.g., "foundry.example.com")
- `WEBAUTHN_ORIGIN`: Expected origin for WebAuthn ceremonies

### Browser Requirements
- Chrome 67+ (2018)
- Firefox 60+ (2018)
- Safari 14+ (2020)
- Edge 18+ (2018)
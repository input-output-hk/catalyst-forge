# CLI Authentication Implementation Tasks

## Overview
Implement device-code linking flow for CLI authentication, allowing the CLI to authenticate via browser-based WebAuthn without handling cryptographic operations directly.

## Phase 1: Core Backend Implementation ✅

### 1. Domain Models & Storage ✅
- [x] Create device-link domain entities
  - [x] `DeviceLink` struct with fields: `device_code`, `user_code`, `device_name`, `purpose`, `expires_at`, `authorized_at`, `user_id`
  - [x] Add to `domain/entities.go`
- [x] Create device management entities  
  - [x] `Device` struct with fields: `device_id`, `user_id`, `device_name`, `created_at`, `last_used_at`
  - [x] Reuse/adapt existing device management if present
- [x] Implement storage interfaces
  - [x] `DeviceLinkStore` interface in `store/` directory
  - [x] `DeviceStore` interface (or adapt existing)
- [x] Implement storage backends
  - [x] GORM implementation for `DeviceLinkStore`
  - [x] GORM implementation for `DeviceStore`
  - [x] In-memory implementation for testing
- [x] Create database migrations
  - [x] Migration for `device_links` table (via AutoMigrate)
  - [x] Migration for `devices` table (via AutoMigrate)

### 2. Service Layer ✅
- [x] Create `DeviceLinkService` in `service/devicelink.go`
  - [x] `BeginDeviceLink(ctx, deviceName, purpose)` - generates codes, stores link
  - [x] `AuthorizeDeviceLink(ctx, deviceCode, userID)` - marks as authorized after WebAuthn
  - [x] `ExchangeDeviceCode(ctx, deviceCode)` - exchanges for tokens
  - [x] Code generation logic (user-friendly 8-char codes like "J7FQ-K9")
  - [x] TTL management (10 min default)
  - [ ] Rate limiting per device_code and IP (to be added in Phase 2)
- [x] Extend `RefreshService` for CLI support
  - [x] Add support for non-cookie refresh tokens (header/body)
  - [x] Maintain backward compatibility with browser cookie mode
  - [x] Device ID binding for refresh tokens

### 3. API Handlers ✅
- [x] Create handler package `handlers/auth/devicelink.go`
  - [x] `BeginDeviceLinkHandler` - POST `/api/v1/auth/device-link/begin`
  - [x] `AuthorizeDeviceLinkHandler` - POST `/api/v1/auth/device-link/authorize`
  - [x] `ExchangeDeviceCodeHandler` - POST `/api/v1/auth/device-link/exchange`
- [x] Modify existing refresh handler
  - [x] Detect `X-CLI: 1` header
  - [x] Support `Authorization: Refresh <token>` header
  - [x] Support JSON body `{"refresh_token": "<token>"}`
  - [x] Skip CSRF validation for CLI mode
  - [x] Return new refresh token in response (optional rotation)

### 4. Route Registration ✅
- [x] Register new device-link routes in `routes/authkit.go`
  - [x] Mount at `/api/v1/auth/device-link/*`
  - [x] Wire up with appropriate middlewares
- [x] Update Swagger annotations
  - [x] Document new endpoints
  - [x] Update existing `/refresh` endpoint docs

### 5. Step-Up Authentication Integration ✅
- [x] Modify policy enforcer to return proper step-up hints
  - [x] Return 428 status with `verification_uri` for CLI
  - [x] Include device-link flow information
- [x] Allow reuse of device-link flow for step-up
  - [x] Support `purpose: "step_up"` in device-link begin
  - [ ] Create step-up grants bound to device_id (to be added in Phase 2)

## Phase 2: Security & Validation ✅

### 6. Security Hardening ✅
- [x] Implement rate limiting
  - [x] Per device_code (prevent brute force)
  - [x] Per IP for exchange endpoint
  - [x] Polling interval enforcement (5s minimum)
- [x] Add audit logging
  - [x] `device_link_begin` event
  - [x] `device_link_authorized` event  
  - [x] `device_link_exchange` event
  - [x] `refresh_reuse_detected` event for CLI tokens
- [x] Validate step-up requirement
  - [x] Require fresh WebAuthn for device-link authorize
  - [x] Integrate with existing step-up service

### 7. Token Management ✅
- [x] Implement device-bound refresh tokens
  - [x] Link refresh token families to device_id
  - [x] Track device_id in refresh token metadata
- [x] Add device revocation
  - [x] Revoke all tokens when device is deleted
  - [x] Cascade deletion of refresh families
- [x] JWT claims enhancement
  - [x] Add `"amr": ["webauthn", "device_link"]` to access tokens
  - [x] Include device_id in token metadata

## Phase 3: Frontend Support

### 8. Browser UI Pages
- [ ] Create `/cli/link` page
  - [ ] Accept and display user_code
  - [ ] Show device information
  - [ ] WebAuthn step-up flow
  - [ ] Call authorize endpoint after authentication
- [ ] Create `/cli/confirm` page for step-up
  - [ ] Similar flow but for step-up confirmation
  - [ ] Show action being confirmed
- [ ] Device management UI
  - [ ] List CLI devices at `/auth/devices`
  - [ ] Show last used timestamps
  - [ ] Revoke/delete functionality

## Phase 4: Testing

### 9. Unit Tests
- [ ] Service layer tests
  - [ ] `DeviceLinkService` test coverage
  - [ ] Modified `RefreshService` tests
  - [ ] Token rotation and replay detection
- [ ] Handler tests
  - [ ] Mock services for handler testing
  - [ ] Error case coverage
  - [ ] Header/body parsing validation

### 10. Integration Tests
- [ ] End-to-end device-link flow test
  - [ ] Begin → Authorize → Exchange sequence
  - [ ] Token refresh with CLI mode
  - [ ] Session invalidation
- [ ] Step-up authentication flow test
  - [ ] Trigger step-up requirement
  - [ ] Complete via device-link
  - [ ] Verify grant application
- [ ] Security tests
  - [ ] Replay attack detection
  - [ ] Rate limiting verification
  - [ ] Token family revocation

## Phase 5: CLI Implementation

### 11. CLI Commands (in separate CLI repo)
- [ ] Implement `forge auth login`
  - [ ] Call begin endpoint
  - [ ] Display user code
  - [ ] Open browser to verification URI
  - [ ] Poll exchange endpoint
  - [ ] Store refresh token in OS keychain
- [ ] Implement `forge auth logout`
  - [ ] Call logout endpoint
  - [ ] Clear keychain entry
- [ ] Implement `forge auth status`
  - [ ] Show current authentication status
  - [ ] Display token expiry
- [ ] Token refresh logic
  - [ ] Auto-refresh on 401
  - [ ] Handle refresh token rotation
  - [ ] Reuse detection handling

### 12. CLI Token Storage
- [ ] OS keychain integration
  - [ ] macOS Keychain
  - [ ] Windows Credential Manager
  - [ ] Linux Secret Service
- [ ] Fallback to encrypted file storage
- [ ] Device ID persistence

## Phase 6: Documentation & Polish

### 13. Documentation
- [ ] API documentation
  - [ ] Update OpenAPI/Swagger specs
  - [ ] Document error responses
  - [ ] Rate limit documentation
- [ ] Developer guide
  - [ ] CLI authentication flow diagram
  - [ ] Step-by-step usage guide
  - [ ] Troubleshooting section
- [ ] Security documentation
  - [ ] Threat model
  - [ ] Security considerations

### 14. Observability
- [ ] Metrics
  - [ ] Device link success/failure rates
  - [ ] Exchange polling patterns
  - [ ] Token refresh rates
- [ ] Logging
  - [ ] Structured logging for all operations
  - [ ] Security event highlighting
- [ ] Monitoring alerts
  - [ ] High failure rates
  - [ ] Replay attack detection
  - [ ] Unusual polling patterns

## Phase 7: Deployment

### 15. Configuration & Deployment
- [ ] Configuration parameters
  - [ ] Device code TTL (default 10 min)
  - [ ] Polling interval (default 5s)
  - [ ] User code format/length
- [ ] Feature flags
  - [ ] Enable/disable CLI auth
  - [ ] Enable/disable GitHub OIDC alongside
- [ ] Migration plan
  - [ ] Database migrations
  - [ ] Backward compatibility
  - [ ] Rollback strategy

## Success Criteria

- [ ] CLI can authenticate without handling WebAuthn
- [ ] Browser handles all cryptographic operations
- [ ] Refresh tokens work seamlessly in CLI mode
- [ ] Step-up authentication works via browser
- [ ] Device management UI shows CLI devices
- [ ] Security: replay detection, rate limiting, audit logging
- [ ] All tests passing
- [ ] Documentation complete

## Notes

- Priority: Phase 1-2 are critical path
- Phase 3 (Frontend) can be developed in parallel
- Phase 4 (Testing) should accompany each phase
- Phase 5 (CLI) can begin after Phase 1 handlers are ready
- Estimated timeline: 2-3 weeks for full implementation
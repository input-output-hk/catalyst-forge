# AuthKit Unit Testing Tasks

This task list focuses on comprehensive unit testing for the AuthKit authentication package. All tests must comply with `.ai/go/testing.md` specifications.

## Testing Standards
- ✅ Use `testify/assert` and `testify/require` for all assertions
- ✅ Table-driven tests with subtests for multiple scenarios
- ✅ Both positive and negative test cases
- ✅ Same-package testing (not `package_test`)
- ✅ Focus on correctness, not implementation details
- ✅ Extensive testing for crypto/security code
- ✅ Use `t.Parallel()` where safe
- ✅ Descriptive failure messages

## Priority: Critical Security Components

### 🔐 1. Crypto Package Tests ✅
**Critical**: Cryptographic code requires extensive testing with test vectors and misuse cases.

#### crypto/keymanager_test.go ✅
- [x] Test JWT signing with ES256 algorithm
- [x] Test JWT verification with valid/invalid signatures
- [x] Test key rotation scenarios
- [x] Test JWKS generation and format
- [x] Test expired JWT handling
- [x] Test malformed JWT rejection
- [x] Test algorithm confusion attacks (e.g., none, HS256 with ES256 key)
- [x] Property tests: sign/verify round-trip

#### crypto/hash_test.go ✅
- [x] Test SHA256 hashing with known vectors
- [x] Test HMAC-SHA256 with RFC test vectors
- [x] Test constant-time comparison
- [x] Test empty input handling
- [x] Test large input handling
- [x] Misuse: verify wrong-length comparisons fail safely

#### crypto/rand_test.go ✅
- [x] Test random byte generation (statistical properties)
- [x] Test base64url encoding correctness
- [x] Test concurrent access safety
- [x] Property: output length matches request
- [x] Property: no obvious patterns in output

---

### 🔑 2. Service Layer Tests ✅
**Critical**: Core authentication logic and security flows.

#### service/tokens_test.go ✅
- [x] Test access token generation with all claims
- [x] Test token parsing and validation
- [x] Test session version mismatch detection
- [x] Test step-up claim handling
- [x] Test audience/issuer validation
- [x] Test clock skew tolerance
- [x] Negative: expired tokens, wrong signing key, tampered payload

#### service/webauthn_test.go ✅
- [x] Test registration challenge generation
- [x] Test registration completion with valid attestation
- [x] Test login challenge generation  
- [x] Test login completion with valid assertion
- [x] Test challenge expiry enforcement
- [x] Test origin validation
- [x] Test AAGUID enforcement for admins
- [x] Test sign count tracking
- [x] Test UV flag validation
- [x] Negative: replay attacks, wrong origin, expired challenges

#### service/refresh_test.go ✅
- [x] Test initial refresh token issuance
- [x] Test token rotation mechanics
- [x] Test family tracking
- [x] Test replay detection and family revocation
- [x] Test session version validation
- [x] Test expiry handling
- [x] Property: each token used exactly once
- [x] Concurrent rotation attempts

#### service/recovery_test.go ✅
- [x] Test recovery code generation (entropy, format)
- [x] Test code consumption (single-use)
- [x] Test code verification with SHA256
- [x] Test batch replacement
- [x] Negative: reuse attempts, wrong codes, timing attacks

#### service/stepup_test.go ✅
- [x] Test step-up grant creation
- [x] Test TTL enforcement
- [x] Test concurrent grant checks
- [x] Test expiry handling

---

### 🛡️ 3. Store Layer Tests ✅
**Important**: Data persistence and integrity.

#### store/gormstore/*_test.go ✅
For each store (users, credentials, invites, refresh, recovery, audit):
- [x] Test Create operations
- [x] Test Read operations (by ID, by unique field)
- [x] Test Update operations
- [x] Test Delete/soft-delete operations
- [x] Test query filters and pagination
- [x] Test concurrent access/race conditions
- [x] Test constraint violations (unique, foreign key)
- [x] Test transaction rollback scenarios

#### testing/inmemory/*_test.go
- [ ] Validate in-memory stores match interface contracts
- [ ] Test thread-safety with concurrent operations
- [ ] Test data isolation between operations

---

### 🌐 4. HTTP/Middleware Tests ✅

#### httpkit/cookies_test.go ✅
- [x] Existing tests validated against spec
- [x] Add tests for __Host- prefix enforcement
- [x] Add tests for all security flags

#### httpkit/csrf_test.go ✅
- [x] Existing tests validated against spec
- [x] Add tests for SameSite + custom header combo
- [x] Add timing attack resistance tests

#### httpkit/responses_test.go ✅
- [x] Test JSON response formatting
- [x] Test error response consistency
- [x] Test enumeration-safe responses
- [x] Test status code mappings

#### middleware/authenticate_test.go ✅
- [x] Test JWT extraction from Authorization header
- [x] Test AuthContext injection
- [x] Test missing/invalid token handling
- [x] Test session version validation
- [x] Performance: measure overhead

#### middleware/policies_test.go ✅
- [x] Test path pattern matching (exact, wildcard, regex)
- [x] Test method-specific rules
- [x] Test role/permission checks
- [x] Test step-up enforcement
- [x] Test policy precedence
- [x] Test performance with large rule sets

#### middleware/ratelimit_test.go ✅
- [x] Test rate limit enforcement
- [x] Test identity-based keys
- [x] Test sliding window mechanics
- [x] Test 429 response format
- [x] Test retry-after header
- [x] Concurrent request handling

#### middleware/audit_test.go ✅
- [x] Existing tests validated against spec
- [x] Add performance tests for async logging

---

### 🔧 5. Domain and Core Tests

#### domain/entities_test.go
- [ ] Test entity validation rules
- [ ] Test field constraints
- [ ] Test JSON marshaling/unmarshaling
- [ ] Test zero values behavior

#### authkit/context_test.go
- [ ] Test context storage/retrieval
- [ ] Test authentication checks
- [ ] Test role/permission helpers
- [ ] Test step-up validation
- [ ] Test concurrent access

#### authkit/policy_test.go
- [ ] Test policy registry construction
- [ ] Test rule compilation
- [ ] Test pattern matching algorithms
- [ ] Test rule priority/conflicts
- [ ] Benchmark: large policy sets

#### authkit/errors_test.go
- [ ] Test error types and messages
- [ ] Test error wrapping/unwrapping
- [ ] Test sentinel error comparisons

---

### 📊 6. Integration Tests

#### integration/auth_flow_test.go
- [ ] Complete onboarding flow (invite → register → login)
- [ ] Complete login flow (challenge → assertion → tokens)
- [ ] Token refresh flow with rotation
- [ ] Recovery flow (init → verify → register)
- [ ] Step-up flow (request → complete)
- [ ] Logout flows (single, all devices)
- [ ] Session invalidation scenarios
- [ ] Rate limiting across flows

#### integration/security_test.go
- [ ] CSRF protection validation
- [ ] WebAuthn replay attack prevention
- [ ] Token replay detection
- [ ] Session fixation prevention
- [ ] Timing attack resistance
- [ ] Concurrent operation safety

---

## Test Coverage Goals

### Must Have 90%+ Coverage
- crypto/* (all cryptographic operations)
- service/tokens.go (JWT handling)
- service/webauthn.go (authentication ceremonies)
- service/refresh.go (token rotation)
- middleware/authenticate.go (security boundary)

### Should Have 80%+ Coverage  
- service/* (business logic)
- store/gormstore/* (data integrity)
- middleware/* (request processing)
- httpkit/* (security helpers)

### Nice to Have 70%+ Coverage
- authkit/* (public API)
- domain/* (data models)
- rate/* (rate limiting)

---

## Special Testing Considerations

### Cryptographic Testing
- Use official test vectors (NIST, RFC, etc.)
- Test constant-time operations with timing measurements
- Include misuse cases (wrong key sizes, bad IVs, etc.)
- Verify no secret leakage in errors/logs

### Concurrency Testing
- Use `-race` flag for all tests
- Test with `GOMAXPROCS=1` and `GOMAXPROCS=4`
- Use sync.WaitGroup for coordination
- Test mutex/lock contention scenarios

### Security Testing
- Test all authentication bypass attempts
- Verify enumeration resistance
- Check for timing attacks
- Test injection attacks (SQL, NoSQL, command)
- Verify no PII in logs/errors

### Performance Testing
- Benchmark critical paths
- Test with realistic data volumes
- Measure memory allocations
- Profile CPU usage for hot paths

---

## Testing Utilities to Create

### test/fixtures/
- [ ] Valid/invalid JWT samples
- [ ] WebAuthn challenge/response fixtures
- [ ] X.509 certificates for testing
- [ ] JWKS fixtures

### test/helpers/
- [ ] Clock mock for time-based tests
- [ ] Random source mock for deterministic tests
- [ ] HTTP client mock for integration tests
- [ ] Database test helpers (setup/teardown)

### test/vectors/
- [ ] HMAC-SHA256 test vectors
- [ ] JWT signature test vectors
- [ ] WebAuthn assertion vectors
- [ ] Recovery code test cases

---

## Validation Checklist for Existing Tests

For each existing test file, verify:
- [ ] Uses testify/assert and testify/require
- [ ] Has table-driven structure with subtests
- [ ] Includes both positive and negative cases
- [ ] Has descriptive test names and failure messages
- [ ] Uses t.Parallel() where appropriate
- [ ] No time.Sleep() or non-deterministic behavior
- [ ] Focuses on behavior, not implementation

---

## Execution Order

1. **Week 1**: Critical crypto components (highest risk)
2. **Week 2**: Service layer (core business logic)
3. **Week 3**: Store layer and middleware
4. **Week 4**: Integration tests and security validation
5. **Week 5**: Performance testing and optimization

---

## Success Metrics

- Zero security vulnerabilities in tested code
- All crypto operations validated against test vectors
- 90%+ coverage on security-critical paths
- All tests pass with -race flag
- Deterministic test execution
- CI runs complete in < 5 minutes
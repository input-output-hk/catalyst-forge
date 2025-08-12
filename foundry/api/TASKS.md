# Lint Violations (golangci-lint)
Generated: Mon Aug 11 21:50:43 PDT 2025

## Summary by linter
- : 356
- revive: 50
- paralleltest: 50
- camel: 50
- t: 30
- github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserKeyRepository: 11
- github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRepository: 9
- github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.DeploymentRepository: 7
- github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.RoleRepository: 6
- db: 6
- unused: 5
- testifylint: 5
- github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRoleRepository: 5
- id: 4
- userKey: 3
- userID: 3
- t,: 3
- github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.ReleaseRepository: 3
- c: 3
- user: 2
- unconvert: 2
- u: 2
- role: 2
- privateKeyPath: 2
- github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.EventRepository: 2
- fmt.Sprintf: 2
- device.Status: 2
- ctx,: 2
- *gorm.io/gorm.DB: 2
- verifier: 1
- um.UserStatusPending: 1
- time.Now: 1
- t,: 1
- t,: 1
- t: 1
- signer: 1
- rl: 1
- requested: 1
- prealloc: 1
- NewSource: 1
- name: 1
- l: 1
- kid: 1
- h: 1
- h: 1
- h: 1
- github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.GithubAuthRepository: 1
- g: 1
- g: 1
- email: 1
- ctx,: 1
- ctx,: 1
- ctx,: 1
- capacity: 1
- c: 1
- c: 1
- auth: 1
- a,: 1
- *DeviceRegistrationHandler: 1
- *DeviceRefreshHandler: 1
- *DeviceLoginHandler: 1
- "verify: 1
- "Unsupported: 1
- "Missing: 1
- "ensure: 1
- "Different: 1

## Details
```
internal/api/handlers/device_registration_challenge_test.go:71:2: Function TestDeviceRegistrationHandler_ChallengeLookup missing the call to method parallel in the test run (paralleltest)
	t.Run("ensure O(1) lookup performance", func(t *testing.T) {
	^
internal/api/handlers/device_registration_challenge_test.go:93:2: Function TestDeviceRegistrationHandler_ChallengeLookup missing the call to method parallel in the test run (paralleltest)
	t.Run("verify we never use proof as lookup key", func(t *testing.T) {
	^
internal/api/handlers/device_registration_test.go:307:2: Function TestGenerateJWKThumbprint_RFC7638_Compliance missing the call to method parallel in the test run (paralleltest)
	t.Run("Different keys produce different thumbprints", func(t *testing.T) {
	^
internal/api/handlers/device_registration_test.go:332:2: Function TestGenerateJWKThumbprint_RFC7638_Compliance missing the call to method parallel in the test run (paralleltest)
	t.Run("Missing required fields", func(t *testing.T) {
	^
internal/api/handlers/device_registration_test.go:366:2: Function TestGenerateJWKThumbprint_RFC7638_Compliance missing the call to method parallel in the test run (paralleltest)
	t.Run("Unsupported key type", func(t *testing.T) {
	^
internal/api/middleware/auth_rate_limit_test.go:89:1: Function TestAuthRateLimitMiddleware_DeviceRegistration missing the call to method parallel (paralleltest)
func TestAuthRateLimitMiddleware_DeviceRegistration(t *testing.T) {
^
internal/auth/cookie_manager_test.go:35:1: Function TestNewCookieManager missing the call to method parallel (paralleltest)
func TestNewCookieManager(t *testing.T) {
^
internal/auth/cookie_manager_test.go:46:1: Function TestCookieManager_SetRefreshTokenCookie missing the call to method parallel (paralleltest)
func TestCookieManager_SetRefreshTokenCookie(t *testing.T) {
^
internal/auth/cookie_manager_test.go:78:1: Function TestCookieManager_ParseRefreshTokenCookie missing the call to method parallel (paralleltest)
func TestCookieManager_ParseRefreshTokenCookie(t *testing.T) {
^
internal/auth/cookie_manager_test.go:125:2: Range statement for test TestCookieManager_ParseRefreshTokenCookie missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/auth/cookie_manager_test.go:146:1: Function TestCookieManager_ParseRefreshTokenValue missing the call to method parallel (paralleltest)
func TestCookieManager_ParseRefreshTokenValue(t *testing.T) {
^
internal/auth/cookie_manager_test.go:199:2: Range statement for test TestCookieManager_ParseRefreshTokenValue missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/auth/cookie_manager_test.go:217:1: Function TestCookieManager_ClearRefreshTokenCookie missing the call to method parallel (paralleltest)
func TestCookieManager_ClearRefreshTokenCookie(t *testing.T) {
^
internal/auth/cookie_manager_test.go:243:1: Function TestCookieManager_SetAccessTokenCookie missing the call to method parallel (paralleltest)
func TestCookieManager_SetAccessTokenCookie(t *testing.T) {
^
internal/auth/cookie_manager_test.go:263:1: Function TestCookieManager_ShouldUseSecureFlag missing the call to method parallel (paralleltest)
func TestCookieManager_ShouldUseSecureFlag(t *testing.T) {
^
internal/auth/cookie_manager_test.go:308:2: Range statement for test TestCookieManager_ShouldUseSecureFlag missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/auth/cookie_manager_test.go:324:1: Function TestCookieManager_GetCookieDomain missing the call to method parallel (paralleltest)
func TestCookieManager_GetCookieDomain(t *testing.T) {
^
internal/auth/cookie_manager_test.go:369:2: Range statement for test TestCookieManager_GetCookieDomain missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/auth/cookie_manager_test.go:385:1: Function TestCookieManager_ValidateRefreshTokenCookieFormat missing the call to method parallel (paralleltest)
func TestCookieManager_ValidateRefreshTokenCookieFormat(t *testing.T) {
^
internal/auth/cookie_manager_test.go:437:2: Range statement for test TestCookieManager_ValidateRefreshTokenCookieFormat missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/auth/cookie_manager_test.go:451:1: Function TestCookieManager_GetRefreshTokenCookieName missing the call to method parallel (paralleltest)
func TestCookieManager_GetRefreshTokenCookieName(t *testing.T) {
^
internal/auth/cookie_manager_test.go:461:1: Function TestCookieManager_EndToEndFlow missing the call to method parallel (paralleltest)
func TestCookieManager_EndToEndFlow(t *testing.T) {
^
internal/auth/device_proof_test.go:135:1: Function TestDeviceProofVerifier_ParseDeviceProofHeaders missing the call to method parallel (paralleltest)
func TestDeviceProofVerifier_ParseDeviceProofHeaders(t *testing.T) {
^
internal/auth/device_proof_test.go:176:2: Range statement for test TestDeviceProofVerifier_ParseDeviceProofHeaders missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/auth/device_proof_test.go:194:1: Function TestDeviceProofVerifier_ParseDeviceProof missing the call to method parallel (paralleltest)
func TestDeviceProofVerifier_ParseDeviceProof(t *testing.T) {
^
internal/auth/device_proof_test.go:230:2: Range statement for test TestDeviceProofVerifier_ParseDeviceProof missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/auth/device_proof_test.go:247:1: Function TestDeviceProofVerifier_GenerateCanonicalString missing the call to method parallel (paralleltest)
func TestDeviceProofVerifier_GenerateCanonicalString(t *testing.T) {
^
internal/auth/device_proof_test.go:264:1: Function TestDeviceProofVerifier_VerifyTimestamp missing the call to method parallel (paralleltest)
func TestDeviceProofVerifier_VerifyTimestamp(t *testing.T) {
^
internal/auth/device_proof_test.go:306:2: Range statement for test TestDeviceProofVerifier_VerifyTimestamp missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/auth/device_proof_test.go:320:1: Function TestDeviceProofVerifier_GetDevicePublicKey missing the call to method parallel (paralleltest)
func TestDeviceProofVerifier_GetDevicePublicKey(t *testing.T) {
^
internal/auth/device_proof_test.go:399:2: Range statement for test TestDeviceProofVerifier_GetDevicePublicKey missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/auth/device_proof_test.go:420:1: Function TestDeviceProofVerifier_VerifySignature missing the call to method parallel (paralleltest)
func TestDeviceProofVerifier_VerifySignature(t *testing.T) {
^
internal/auth/device_proof_test.go:458:2: Range statement for test TestDeviceProofVerifier_VerifySignature missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/auth/device_proof_test.go:472:1: Function TestDeviceProofVerifier_VerifyDeviceProof_EndToEnd missing the call to method parallel (paralleltest)
func TestDeviceProofVerifier_VerifyDeviceProof_EndToEnd(t *testing.T) {
^
internal/auth/device_proof_test.go:507:1: Function TestDeviceProofVerifier_parseECDSAFromJWK missing the call to method parallel (paralleltest)
func TestDeviceProofVerifier_parseECDSAFromJWK(t *testing.T) {
^
internal/auth/device_proof_test.go:566:2: Range statement for test TestDeviceProofVerifier_parseECDSAFromJWK missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/auth/device_proof_test.go:583:1: Function TestDeviceProofVerifier_VerifyDeviceProofWithKey_AlgorithmValidation missing the call to method parallel (paralleltest)
func TestDeviceProofVerifier_VerifyDeviceProofWithKey_AlgorithmValidation(t *testing.T) {
^
internal/auth/device_proof_test.go:645:2: Range statement for test TestDeviceProofVerifier_VerifyDeviceProofWithKey_AlgorithmValidation missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/auth/device_proof_test.go:659:1: Function TestDeviceProofVerifier_validateAlgorithm missing the call to method parallel (paralleltest)
func TestDeviceProofVerifier_validateAlgorithm(t *testing.T) {
^
internal/auth/device_proof_test.go:859:2: Range statement for test TestDeviceProofVerifier_validateAlgorithm missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
internal/ca/issuance_policy_test.go:8:1: Function TestClampTTL missing the call to method parallel (paralleltest)
func TestClampTTL(t *testing.T) {
^
internal/ca/issuance_policy_test.go:20:2: Range statement for test TestClampTTL missing the call to method parallel in test Run (paralleltest)
	for _, tc := range tests {
	^
internal/ca/validators_test.go:37:1: Function TestValidateClientCSR_Success_NoDNSNoIP missing the call to method parallel (paralleltest)
func TestValidateClientCSR_Success_NoDNSNoIP(t *testing.T) {
^
internal/ca/validators_test.go:44:1: Function TestValidateClientCSR_Fails_WithDNS missing the call to method parallel (paralleltest)
func TestValidateClientCSR_Fails_WithDNS(t *testing.T) {
^
internal/ca/validators_test.go:51:1: Function TestValidateClientCSR_Fails_WithIP missing the call to method parallel (paralleltest)
func TestValidateClientCSR_Fails_WithIP(t *testing.T) {
^
internal/ca/validators_test.go:58:1: Function TestValidateServerCSR_Success_WithDNS missing the call to method parallel (paralleltest)
func TestValidateServerCSR_Success_WithDNS(t *testing.T) {
^
internal/ca/validators_test.go:65:1: Function TestValidateServerCSR_Success_WithIP missing the call to method parallel (paralleltest)
func TestValidateServerCSR_Success_WithIP(t *testing.T) {
^
internal/ca/validators_test.go:72:1: Function TestValidateServerCSR_Fails_NoSANs missing the call to method parallel (paralleltest)
func TestValidateServerCSR_Fails_NoSANs(t *testing.T) {
^
internal/config/config_test.go:10:1: Function TestBootstrapTokenValidation missing the call to method parallel (paralleltest)
func TestBootstrapTokenValidation(t *testing.T) {
^
internal/config/config_test.go:40:2: Range statement for test TestBootstrapTokenValidation missing the call to method parallel in test Run (paralleltest)
	for _, tt := range tests {
	^
cmd/api/auth.go:82:2: Consider pre-allocating `perms` (prealloc)
	var perms []libauth.Permission
	^
cmd/api/auth/auth.go:4:6: exported: type name will be used as auth.AuthCmd by other packages, and that stutters; consider calling this Cmd (revive)
type AuthCmd struct {
     ^
cmd/api/auth/generate.go:12:6: exported: exported type GenerateCmd should have comment or be unexported (revive)
type GenerateCmd struct {
     ^
cmd/api/auth/generate.go:20:1: exported: exported method GenerateCmd.Run should have comment or be unexported (revive)
func (g *GenerateCmd) Run() error {
^
cmd/api/auth/init.go:11:6: exported: exported type InitCmd should have comment or be unexported (revive)
type InitCmd struct {
     ^
cmd/api/auth/validate.go:10:6: exported: exported type ValidateCmd should have comment or be unexported (revive)
type ValidateCmd struct {
     ^
cmd/api/auth/validate.go:15:1: exported: exported method ValidateCmd.Run should have comment or be unexported (revive)
func (g *ValidateCmd) Run() error {
^
internal/api/handlers/user/user.go:14:6: exported: type name will be used as user.UserHandler by other packages, and that stutters; consider calling this Handler (revive)
type UserHandler struct {
     ^
internal/api/handlers/user/user_key.go:21:6: exported: type name will be used as user.UserKeyHandler by other packages, and that stutters; consider calling this KeyHandler (revive)
type UserKeyHandler struct {
     ^
internal/api/handlers/user/user_key.go:65:6: exported: exported type BootstrapKETRequest should have comment or be unexported (revive)
type BootstrapKETRequest struct {
     ^
internal/api/handlers/user/user_key.go:69:6: exported: exported type BootstrapKETResponse should have comment or be unexported (revive)
type BootstrapKETResponse struct {
     ^
internal/api/handlers/user/user_key.go:74:6: exported: exported type RegisterWithKETRequest should have comment or be unexported (revive)
type RegisterWithKETRequest struct {
     ^
internal/api/handlers/user/user_role.go:14:6: exported: type name will be used as user.UserRoleHandler by other packages, and that stutters; consider calling this RoleHandler (revive)
type UserRoleHandler struct {
     ^
internal/api/handlers/user/user_role.go:29:6: exported: type name will be used as user.UserRole by other packages, and that stutters; consider calling this Role (revive)
type UserRole struct {
     ^
internal/api/middleware/ratelimit.go:19:6: exported: exported type RateLimiter should have comment or be unexported (revive)
type RateLimiter struct {
     ^
internal/api/middleware/ratelimit.go:26:1: exported: exported function NewRateLimiter should have comment or be unexported (revive)
func NewRateLimiter(capacity int, refill time.Duration) *RateLimiter {
^
internal/api/middleware/ratelimit.go:30:1: exported: exported method RateLimiter.Allow should have comment or be unexported (revive)
func (rl *RateLimiter) Allow(key string) bool {
^
internal/api/middleware/ratelimit.go:53:1: redefines-builtin-id: redefinition of the built-in function min (revive)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
internal/ca/issuance_policy.go:8:40: redefines-builtin-id: redefinition of the built-in function cap (revive)
func ClampTTL(requested time.Duration, cap time.Duration) time.Duration {
                                       ^
internal/config/config.go:25:2: var-naming: struct field HttpPort should be HTTPPort (revive)
	HttpPort       int           `kong:"help='HTTP port to listen on',default=8080,name='http-port',env='HTTP_PORT'"`
	^
internal/models/build/build_session.go:8:6: exported: type name will be used as build.BuildSession by other packages, and that stutters; consider calling this Session (revive)
type BuildSession struct {
     ^
internal/models/user/user.go:10:6: exported: type name will be used as user.UserStatus by other packages, and that stutters; consider calling this Status (revive)
type UserStatus string
     ^
internal/models/user/user_key.go:15:6: exported: type name will be used as user.UserKeyStatus by other packages, and that stutters; consider calling this KeyStatus (revive)
type UserKeyStatus string
     ^
internal/models/user/user_key.go:25:6: exported: type name will be used as user.UserKey by other packages, and that stutters; consider calling this Key (revive)
type UserKey struct {
     ^
internal/models/user/user_role.go:10:6: exported: type name will be used as user.UserRole by other packages, and that stutters; consider calling this Role (revive)
type UserRole struct {
     ^
internal/rate/limiter.go:1:1: package-comments: should have a package comment (revive)
package rate
^
internal/rate/limiter.go:20:1: exported: exported function NewInMemoryLimiter should have comment or be unexported (revive)
func NewInMemoryLimiter() *InMemoryLimiter {
^
internal/rate/limiter.go:24:1: exported: exported method InMemoryLimiter.Allow should have comment or be unexported (revive)
func (l *InMemoryLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, error) {
^
internal/repository/audit/log.go:1:1: package-comments: should have a package comment (revive)
package audit
^
internal/repository/audit/log.go:8:6: exported: exported type LogRepository should have comment or be unexported (revive)
type LogRepository interface {
     ^
internal/repository/audit/log.go:14:1: exported: exported function NewLogRepository should have comment or be unexported (revive)
func NewLogRepository(db *gorm.DB) LogRepository { return &logRepository{db: db} }
^
internal/repository/build/build_session.go:1:1: package-comments: should have a package comment (revive)
package buildrepo
^
internal/repository/build/build_session.go:10:6: exported: exported type BuildSessionRepository should have comment or be unexported (revive)
type BuildSessionRepository interface {
     ^
internal/repository/build/build_session.go:17:1: exported: exported function NewBuildSessionRepository should have comment or be unexported (revive)
func NewBuildSessionRepository(db *gorm.DB) BuildSessionRepository {
^
internal/repository/user/device.go:11:6: exported: exported type DeviceRepository should have comment or be unexported (revive)
type DeviceRepository interface {
     ^
internal/repository/user/device.go:25:1: exported: exported function NewDeviceRepository should have comment or be unexported (revive)
func NewDeviceRepository(db *gorm.DB) DeviceRepository {
^
internal/repository/user/invite.go:10:6: exported: exported type InviteRepository should have comment or be unexported (revive)
type InviteRepository interface {
     ^
internal/repository/user/invite.go:24:1: exported: exported function NewInviteRepository should have comment or be unexported (revive)
func NewInviteRepository(db *gorm.DB) InviteRepository { return &inviteRepository{db: db} }
^
internal/repository/user/invite_repository_test.go:22:56: unused-parameter: parameter 'db' seems to be unused, consider removing or renaming it as _ (revive)
			validate: func(t *testing.T, repo InviteRepository, db *gorm.DB) {
			                                                    ^
internal/repository/user/refresh_token.go:12:6: exported: exported type RefreshTokenRepository should have comment or be unexported (revive)
type RefreshTokenRepository interface {
     ^
internal/repository/user/refresh_token.go:34:1: exported: exported function NewRefreshTokenRepository should have comment or be unexported (revive)
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
^
internal/repository/user/revoked_jti.go:10:6: exported: exported type RevokedJTIRepository should have comment or be unexported (revive)
type RevokedJTIRepository interface {
     ^
internal/repository/user/revoked_jti.go:18:1: exported: exported function NewRevokedJTIRepository should have comment or be unexported (revive)
func NewRevokedJTIRepository(db *gorm.DB) RevokedJTIRepository {
^
internal/repository/user/user.go:9:6: exported: type name will be used as user.UserRepository by other packages, and that stutters; consider calling this Repository (revive)
type UserRepository interface {
     ^
internal/repository/user/user_key.go:9:6: exported: type name will be used as user.UserKeyRepository by other packages, and that stutters; consider calling this KeyRepository (revive)
type UserKeyRepository interface {
     ^
internal/repository/user/user_role.go:9:6: exported: type name will be used as user.UserRoleRepository by other packages, and that stutters; consider calling this RoleRepository (revive)
type UserRoleRepository interface {
     ^
internal/service/user/user.go:14:6: exported: type name will be used as user.UserService by other packages, and that stutters; consider calling this Service (revive)
type UserService interface {
     ^
internal/service/user/user_key.go:14:6: exported: type name will be used as user.UserKeyService by other packages, and that stutters; consider calling this KeyService (revive)
type UserKeyService interface {
     ^
internal/service/user/user_role.go:12:6: exported: type name will be used as user.UserRoleService by other packages, and that stutters; consider calling this RoleService (revive)
type UserRoleService interface {
     ^
internal/utils/context.go:1:9: var-naming: avoid meaningless package names (revive)
package utils
        ^
pkg/utils/id.go:1:9: var-naming: avoid meaningless package names (revive)
package utils
        ^
cmd/api/mockdata.go:60:2: SA1019: mrand.Seed has been deprecated since Go 1.20 and an alternative has been available since Go 1.0: As of Go 1.20 there is no reason to call Seed with a random value. Programs that call Seed with a known value to get a specific sequence of results should use New(NewSource(seed)) to obtain a local random generator. (staticcheck)
	mrand.Seed(time.Now().UnixNano())
	^
internal/models/alias.go:12:27: json(camel): got 'release_id' want 'releaseId' (tagliatelle)
	ReleaseID string         `gorm:"not null;index" json:"release_id"`
	                         ^
internal/models/audit/log.go:13:31: json(camel): got 'event_type' want 'eventType' (tagliatelle)
	EventType     string         `gorm:"not null;index" json:"event_type"`
	                             ^
internal/models/audit/log.go:14:31: json(camel): got 'actor_user_id' want 'actorUserId' (tagliatelle)
	ActorUserID   *uint          `gorm:"index" json:"actor_user_id,omitempty"`
	                             ^
internal/models/audit/log.go:15:31: json(camel): got 'subject_user_id' want 'subjectUserId' (tagliatelle)
	SubjectUserID *uint          `gorm:"index" json:"subject_user_id,omitempty"`
	                             ^
internal/models/audit/log.go:16:31: json(camel): got 'request_ip' want 'requestIp' (tagliatelle)
	RequestIP     string         `json:"request_ip"`
	                             ^
internal/models/audit/log.go:17:31: json(camel): got 'user_agent' want 'userAgent' (tagliatelle)
	UserAgent     string         `json:"user_agent"`
	                             ^
internal/models/deployment.go:23:29: json(camel): got 'release_id' want 'releaseId' (tagliatelle)
	ReleaseID string           `gorm:"not null;index" json:"release_id"`
	                           ^
internal/models/deployment_event.go:12:30: json(camel): got 'deployment_id' want 'deploymentId' (tagliatelle)
	DeploymentID string         `gorm:"not null;index" json:"deployment_id"`
	                            ^
internal/models/github.go:19:29: json(camel): got 'created_by' want 'createdBy' (tagliatelle)
	CreatedBy   string         `gorm:"not null" json:"created_by"`
	                           ^
internal/models/github.go:20:29: json(camel): got 'updated_by' want 'updatedBy' (tagliatelle)
	UpdatedBy   string         `gorm:"not null" json:"updated_by"`
	                           ^
internal/models/release.go:12:25: json(camel): got 'source_repo' want 'sourceRepo' (tagliatelle)
	SourceRepo   string    `gorm:"not null" json:"source_repo"`
	                       ^
internal/models/release.go:13:25: json(camel): got 'source_commit' want 'sourceCommit' (tagliatelle)
	SourceCommit string    `gorm:"not null" json:"source_commit"`
	                       ^
internal/models/release.go:14:25: json(camel): got 'source_branch' want 'sourceBranch' (tagliatelle)
	SourceBranch string    `json:"source_branch,omitempty"`
	                       ^
internal/models/release.go:16:25: json(camel): got 'project_path' want 'projectPath' (tagliatelle)
	ProjectPath  string    `gorm:"not null" json:"project_path"`
	                       ^
internal/models/user/bootstrap_token.go:13:29: json(camel): got 'used_at' want 'usedAt' (tagliatelle)
	UsedAt      time.Time      `gorm:"not null" json:"used_at"`
	                           ^
internal/models/user/bootstrap_token.go:14:29: json(camel): got 'used_by_email' want 'usedByEmail' (tagliatelle)
	UsedByEmail string         `gorm:"not null" json:"used_by_email"`
	                           ^
internal/models/user/bootstrap_token.go:15:29: json(camel): got 'invite_id' want 'inviteId' (tagliatelle)
	InviteID    uint           `gorm:"not null" json:"invite_id"` // Reference to the created invite
	                           ^
internal/models/user/bootstrap_token.go:16:29: json(camel): got 'created_at' want 'createdAt' (tagliatelle)
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	                           ^
internal/models/user/device.go:13:31: json(camel): got 'user_id' want 'userId' (tagliatelle)
	UserID        uint           `gorm:"not null;index" json:"user_id"`
	                             ^
internal/models/user/device.go:18:31: json(camel): got 'created_at' want 'createdAt' (tagliatelle)
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	                             ^
internal/models/user/device.go:19:31: json(camel): got 'last_used_at' want 'lastUsedAt' (tagliatelle)
	LastUsedAt    *time.Time     `json:"last_used_at,omitempty"`
	                             ^
internal/models/user/device.go:20:31: json(camel): got 'revoked_at' want 'revokedAt' (tagliatelle)
	RevokedAt     *time.Time     `json:"revoked_at,omitempty"`
	                             ^
internal/models/user/device_session.go:13:29: json(camel): got 'user_code' want 'userCode' (tagliatelle)
	UserCode        string     `gorm:"not null;uniqueIndex" json:"user_code"`
	                           ^
internal/models/user/device_session.go:14:29: json(camel): got 'expires_at' want 'expiresAt' (tagliatelle)
	ExpiresAt       time.Time  `gorm:"not null" json:"expires_at"`
	                           ^
internal/models/user/device_session.go:15:29: json(camel): got 'interval_seconds' want 'intervalSeconds' (tagliatelle)
	IntervalSeconds int        `gorm:"not null" json:"interval_seconds"`
	                           ^
internal/models/user/device_session.go:17:29: json(camel): got 'approved_user_id' want 'approvedUserId' (tagliatelle)
	ApprovedUserID  *uint      `gorm:"index" json:"approved_user_id,omitempty"`
	                           ^
internal/models/user/device_session.go:18:29: json(camel): got 'last_polled_at' want 'lastPolledAt' (tagliatelle)
	LastPolledAt    *time.Time `json:"last_polled_at,omitempty"`
	                           ^
internal/models/user/device_session.go:19:29: json(camel): got 'poll_count' want 'pollCount' (tagliatelle)
	PollCount       int        `gorm:"not null;default:0" json:"poll_count"`
	                           ^
internal/models/user/device_session.go:20:29: json(camel): got 'completed_at' want 'completedAt' (tagliatelle)
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	                           ^
internal/models/user/device_session.go:25:29: json(camel): got 'created_at' want 'createdAt' (tagliatelle)
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	                           ^
internal/models/user/invite.go:16:33: json(camel): got 'expires_at' want 'expiresAt' (tagliatelle)
	ExpiresAt       time.Time      `gorm:"not null" json:"expires_at"`
	                               ^
internal/models/user/invite.go:17:33: json(camel): got 'redeemed_at' want 'redeemedAt' (tagliatelle)
	RedeemedAt      *time.Time     `json:"redeemed_at,omitempty"`
	                               ^
internal/models/user/invite.go:18:33: json(camel): got 'created_by' want 'createdBy' (tagliatelle)
	CreatedBy       uint           `gorm:"not null" json:"created_by"`
	                               ^
internal/models/user/refresh_token.go:12:24: json(camel): got 'user_id' want 'userId' (tagliatelle)
	UserID     uint       `gorm:"not null;index" json:"user_id"`
	                      ^
internal/models/user/refresh_token.go:13:24: json(camel): got 'device_id' want 'deviceId' (tagliatelle)
	DeviceID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"device_id"`
	                      ^
internal/models/user/refresh_token.go:14:24: json(camel): got 'family_id' want 'familyId' (tagliatelle)
	FamilyID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"family_id"`
	                      ^
internal/models/user/refresh_token.go:15:24: json(camel): got 'parent_id' want 'parentId' (tagliatelle)
	ParentID   *uuid.UUID `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	                      ^
internal/models/user/refresh_token.go:18:24: json(camel): got 'expires_at' want 'expiresAt' (tagliatelle)
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	                      ^
internal/models/user/refresh_token.go:19:24: json(camel): got 'revoked_at' want 'revokedAt' (tagliatelle)
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	                      ^
internal/models/user/refresh_token.go:20:24: json(camel): got 'rotated_at' want 'rotatedAt' (tagliatelle)
	RotatedAt  *time.Time `gorm:"index" json:"rotated_at,omitempty"` // When this token was rotated/used
	                      ^
internal/models/user/revoked_jti.go:9:22: json(camel): got 'revoked_at' want 'revokedAt' (tagliatelle)
	RevokedAt time.Time `gorm:"not null;autoCreateTime" json:"revoked_at"`
	                    ^
internal/models/user/role.go:20:27: json(camel): got 'updated_at' want 'updatedAt' (tagliatelle)
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	                         ^
internal/models/user/user.go:25:29: json(camel): got 'email_verified_at' want 'emailVerifiedAt' (tagliatelle)
	EmailVerifiedAt *time.Time `gorm:"index" json:"email_verified_at,omitempty"`
	                           ^
internal/models/user/user.go:26:29: json(camel): got 'user_ver' want 'userVer' (tagliatelle)
	UserVer         int        `gorm:"not null;default:1" json:"user_ver"`
	                           ^
internal/models/user/user.go:30:27: json(camel): got 'updated_at' want 'updatedAt' (tagliatelle)
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	                         ^
internal/models/user/user_key.go:27:26: json(camel): got 'user_id' want 'userId' (tagliatelle)
	UserID    uint          `gorm:"not null;index" json:"user_id"`
	                        ^
internal/models/user/user_key.go:29:26: json(camel): got 'pubkey_b64' want 'pubkeyB64' (tagliatelle)
	PubKeyB64 string        `gorm:"not null" json:"pubkey_b64"`
	                        ^
internal/models/user/user_key.go:33:17: json(camel): got 'device_id' want 'deviceId' (tagliatelle)
	DeviceID *uint `gorm:"index" json:"device_id,omitempty"`
	               ^
internal/models/user/user_key.go:40:27: json(camel): got 'updated_at' want 'updatedAt' (tagliatelle)
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	                         ^
internal/models/user/user_role.go:13:14: json(camel): got 'role_id' want 'roleId' (tagliatelle)
	RoleID uint `gorm:"not null;index" json:"role_id"`
	            ^
internal/api/handlers/device_logout_test.go:197:2: empty: use assert.NotEmpty (testifylint)
	assert.Positive(t, len(deviceID.String()))
	^
internal/api/handlers/device_registration_challenge_test.go:109:3: useless-assert: meaningless assertion (testifylint)
		assert.True(t, true, "Type safety prevents using device proof as lookup key")
		^
internal/api/handlers/device_registration_test.go:341:3: require-error: for error assertions use require (testifylint)
		assert.Error(t, err)
		^
internal/api/handlers/device_registration_test.go:352:3: require-error: for error assertions use require (testifylint)
		assert.Error(t, err)
		^
internal/api/handlers/device_registration_test.go:362:3: require-error: for error assertions use require (testifylint)
		assert.Error(t, err)
		^
internal/api/handlers/device_management.go:117:22: unnecessary conversion (unconvert)
			Status:     string(device.Status),
			                  ^
internal/api/handlers/device_management_test.go:311:21: unnecessary conversion (unconvert)
		Status:     string(device.Status),
		                  ^
internal/api/handlers/device_login.go:318:73: (*DeviceLoginHandler).generateSecretHash - result 1 (error) is always nil (unparam)
func (h *DeviceLoginHandler) generateSecretHash(secret string) (string, error) {
                                                                        ^
internal/api/handlers/device_refresh.go:258:75: (*DeviceRefreshHandler).generateSecretHash - result 1 (error) is always nil (unparam)
func (h *DeviceRefreshHandler) generateSecretHash(secret string) (string, error) {
                                                                          ^
internal/api/handlers/device_registration.go:560:80: (*DeviceRegistrationHandler).generateSecretHash - result 1 (error) is always nil (unparam)
func (h *DeviceRegistrationHandler) generateSecretHash(secret string) (string, error) {
                                                                               ^
internal/api/handlers/cookies.go:13:6: func setAccessCookie is unused (unused)
func setAccessCookie(c *gin.Context, jwt string, ttl time.Duration) {
     ^
internal/api/handlers/cookies.go:18:6: func setRefreshCookie is unused (unused)
func setRefreshCookie(c *gin.Context, opaque string, ttl time.Duration) {
     ^
internal/api/handlers/cookies.go:23:6: func clearAccessCookie is unused (unused)
func clearAccessCookie(c *gin.Context) {
     ^
internal/api/handlers/cookies.go:27:6: func clearRefreshCookie is unused (unused)
func clearRefreshCookie(c *gin.Context) {
     ^
internal/api/handlers/cookies.go:31:6: func cookieDomain is unused (unused)
func cookieDomain(c *gin.Context) string {
     ^
cmd/api/auth/generate.go:24:10: error returned from external package is unwrapped: sig: func github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt.NewES256Manager(privateKeyPath string, publicKeyPath string, opts ...github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt.ManagerOption) (*github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt.ES256Manager, error) (wrapcheck)
		return err
		       ^
cmd/api/auth/generate.go:51:10: error returned from external package is unwrapped: sig: func github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens.GenerateAuthToken(signer github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt.JWTSigner, subject string, permissions []github.com/input-output-hk/catalyst-forge/lib/foundry/auth.Permission, expiration time.Duration, opts ...github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt.TokenOption) (string, error) (wrapcheck)
		return err
		       ^
cmd/api/auth/validate.go:18:10: error returned from external package is unwrapped: sig: func github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt.NewES256Manager(privateKeyPath string, publicKeyPath string, opts ...github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt.ManagerOption) (*github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt.ES256Manager, error) (wrapcheck)
		return err
		       ^
cmd/api/auth/validate.go:23:10: error returned from external package is unwrapped: sig: func github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens.VerifyAuthToken(verifier github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt.JWTVerifier, tokenString string) (*github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens.AuthClaims, error) (wrapcheck)
		return err
		       ^
internal/repository/counter.go:54:14: error returned from external package is unwrapped: sig: func (*gorm.io/gorm.DB).Transaction(fc func(tx *gorm.io/gorm.DB) error, opts ...*database/sql.TxOptions) (err error) (wrapcheck)
		return "", err
		           ^
internal/service/deployment.go:61:15: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.ReleaseRepository).GetByID(ctx context.Context, id string) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.Release, error) (wrapcheck)
		return nil, err
		            ^
internal/service/deployment.go:83:11: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.DeploymentRepository).Create(ctx context.Context, deployment *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.ReleaseDeployment) error (wrapcheck)
			return err
			       ^
internal/service/deployment.go:98:15: error returned from external package is unwrapped: sig: func (*gorm.io/gorm.DB).Transaction(fc func(tx *gorm.io/gorm.DB) error, opts ...*database/sql.TxOptions) (err error) (wrapcheck)
		return nil, err
		            ^
internal/service/deployment.go:109:15: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.DeploymentRepository).GetByID(ctx context.Context, id string) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.ReleaseDeployment, error) (wrapcheck)
		return nil, err
		            ^
internal/service/deployment.go:114:15: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.ReleaseRepository).GetByID(ctx context.Context, id string) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.Release, error) (wrapcheck)
		return nil, err
		            ^
internal/service/deployment.go:130:10: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.DeploymentRepository).GetByID(ctx context.Context, id string) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.ReleaseDeployment, error) (wrapcheck)
		return err
		       ^
internal/service/deployment.go:135:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.DeploymentRepository).Update(ctx context.Context, deployment *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.ReleaseDeployment) error (wrapcheck)
	return s.deploymentRepo.Update(ctx, deployment)
	       ^
internal/service/deployment.go:142:15: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.ReleaseRepository).GetByID(ctx context.Context, id string) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.Release, error) (wrapcheck)
		return nil, err
		            ^
internal/service/deployment.go:145:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.DeploymentRepository).ListByReleaseID(ctx context.Context, releaseID string) ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.ReleaseDeployment, error) (wrapcheck)
	return s.deploymentRepo.ListByReleaseID(ctx, releaseID)
	       ^
internal/service/deployment.go:155:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.DeploymentRepository).GetLatestByReleaseID(ctx context.Context, releaseID string) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.ReleaseDeployment, error) (wrapcheck)
	return s.deploymentRepo.GetLatestByReleaseID(ctx, releaseID)
	       ^
internal/service/deployment.go:162:10: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.DeploymentRepository).GetByID(ctx context.Context, id string) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.ReleaseDeployment, error) (wrapcheck)
		return err
		       ^
internal/service/deployment.go:172:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.EventRepository).AddEvent(ctx context.Context, event *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.DeploymentEvent) error (wrapcheck)
	return s.eventRepo.AddEvent(ctx, event)
	       ^
internal/service/deployment.go:182:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.EventRepository).ListEventsByDeploymentID(ctx context.Context, deploymentID string) ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.DeploymentEvent, error) (wrapcheck)
	return s.eventRepo.ListEventsByDeploymentID(ctx, deploymentID)
	       ^
internal/service/gha.go:56:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository.GithubAuthRepository).Create(auth *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models.GithubRepositoryAuth) error (wrapcheck)
	return s.repo.Create(auth)
	       ^
internal/service/user/role.go:53:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.RoleRepository).Create(role *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.Role) error (wrapcheck)
	return s.repo.Create(role)
	       ^
internal/service/user/role.go:58:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.RoleRepository).GetByID(id string) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.Role, error) (wrapcheck)
	return s.repo.GetByID(fmt.Sprintf("%d", id))
	       ^
internal/service/user/role.go:63:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.RoleRepository).GetByName(name string) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.Role, error) (wrapcheck)
	return s.repo.GetByName(name)
	       ^
internal/service/user/role.go:77:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.RoleRepository).Update(role *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.Role) error (wrapcheck)
	return s.repo.Update(role)
	       ^
internal/service/user/role.go:91:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.RoleRepository).Delete(id string) error (wrapcheck)
	return s.repo.Delete(fmt.Sprintf("%d", id))
	       ^
internal/service/user/role.go:96:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.RoleRepository).List() ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.Role, error) (wrapcheck)
	return s.repo.List()
	       ^
internal/service/user/user.go:62:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRepository).Create(user *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.User) error (wrapcheck)
	return s.repo.Create(user)
	       ^
internal/service/user/user.go:67:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRepository).GetByID(id uint) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.User, error) (wrapcheck)
	return s.repo.GetByID(id)
	       ^
internal/service/user/user.go:72:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRepository).GetByEmail(email string) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.User, error) (wrapcheck)
	return s.repo.GetByEmail(email)
	       ^
internal/service/user/user.go:87:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRepository).Update(user *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.User) error (wrapcheck)
	return s.repo.Update(user)
	       ^
internal/service/user/user.go:101:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRepository).Delete(id uint) error (wrapcheck)
	return s.repo.Delete(id)
	       ^
internal/service/user/user.go:106:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRepository).List() ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.User, error) (wrapcheck)
	return s.repo.List()
	       ^
internal/service/user/user.go:111:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRepository).GetByStatus(status github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserStatus) ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.User, error) (wrapcheck)
	return s.repo.GetByStatus(um.UserStatusPending)
	       ^
internal/service/user/user.go:127:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRepository).Update(user *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.User) error (wrapcheck)
	return s.repo.Update(u)
	       ^
internal/service/user/user.go:143:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRepository).Update(user *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.User) error (wrapcheck)
	return s.repo.Update(u)
	       ^
internal/service/user/user_key.go:65:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserKeyRepository).Create(userKey *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserKey) error (wrapcheck)
	return s.repo.Create(userKey)
	       ^
internal/service/user/user_key.go:70:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserKeyRepository).GetByID(id uint) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserKey, error) (wrapcheck)
	return s.repo.GetByID(id)
	       ^
internal/service/user/user_key.go:75:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserKeyRepository).GetByKid(kid string) (*github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserKey, error) (wrapcheck)
	return s.repo.GetByKid(kid)
	       ^
internal/service/user/user_key.go:80:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserKeyRepository).GetByUserID(userID uint) ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserKey, error) (wrapcheck)
	return s.repo.GetByUserID(userID)
	       ^
internal/service/user/user_key.go:85:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserKeyRepository).GetActiveByUserID(userID uint) ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserKey, error) (wrapcheck)
	return s.repo.GetActiveByUserID(userID)
	       ^
internal/service/user/user_key.go:90:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserKeyRepository).GetInactiveByUserID(userID uint) ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserKey, error) (wrapcheck)
	return s.repo.GetInactiveByUserID(userID)
	       ^
internal/service/user/user_key.go:95:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserKeyRepository).GetInactive() ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserKey, error) (wrapcheck)
	return s.repo.GetInactive()
	       ^
internal/service/user/user_key.go:111:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserKeyRepository).Update(userKey *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserKey) error (wrapcheck)
	return s.repo.Update(userKey)
	       ^
internal/service/user/user_key.go:126:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserKeyRepository).Delete(id uint) error (wrapcheck)
	return s.repo.Delete(id)
	       ^
internal/service/user/user_key.go:143:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserKeyRepository).Update(userKey *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserKey) error (wrapcheck)
	return s.repo.Update(userKey)
	       ^
internal/service/user/user_key.go:148:9: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserKeyRepository).List() ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserKey, error) (wrapcheck)
	return s.repo.List()
	       ^
internal/service/user/user_role.go:43:10: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRoleRepository).Create(userRole *github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserRole) error (wrapcheck)
		return err
		       ^
internal/service/user/user_role.go:54:10: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRoleRepository).DeleteByUserIDAndRoleID(userID string, roleID string) error (wrapcheck)
		return err
		       ^
internal/service/user/user_role.go:66:15: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRoleRepository).GetByUserID(userID string) ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserRole, error) (wrapcheck)
		return nil, err
		            ^
internal/service/user/user_role.go:77:15: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRoleRepository).GetByRoleID(roleID string) ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserRole, error) (wrapcheck)
		return nil, err
		            ^
internal/service/user/user_role.go:88:15: error returned from interface method should be wrapped: sig: func (github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user.UserRoleRepository).GetByUserID(userID string) ([]github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user.UserRole, error) (wrapcheck)
		return nil, err
		            ^
217 issues:
* paralleltest: 50
* prealloc: 1
* revive: 50
* staticcheck: 1
* tagliatelle: 50
* testifylint: 5
* unconvert: 2
* unparam: 3
* unused: 5
* wrapcheck: 50
```

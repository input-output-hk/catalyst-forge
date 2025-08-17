package auth

import (
	"github.com/gin-gonic/gin"
	service "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
)

// The functions in this file exist solely to carry Swagger annotations.
// They are not registered as handlers at runtime. The real bindings happen
// in the various Register* functions in this package. Swaggo requires
// annotation blocks to be attached to top-level functions to generate paths.

// @Summary Begin login
// @Tags auth
// @Produce json
// @Success 200 {object} auth.PublicKeyOptionsResponse
// @Router /api/v1/auth/login/begin [post]
func DocAuthLoginBegin(c *gin.Context) {}

// @Summary Complete login
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.LoginCompleteRequest true "Login complete request"
// @Success 200 {object} auth.LoginCompleteResponse
// @Router /api/v1/auth/login/complete [post]
func DocAuthLoginComplete(c *gin.Context) {}

// @Summary List credentials
// @Tags auth
// @Produce json
// @Success 200 {object} auth.CredentialsListResponse
// @Router /api/v1/auth/credentials [get]
func DocAuthCredentialsList(c *gin.Context) {}

// @Summary Add credential (begin)
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.CredentialsAddBeginRequest true "Begin credential add"
// @Success 200 {object} auth.PublicKeyOptionsResponse
// @Router /api/v1/auth/credentials/add/begin [post]
func DocAuthCredentialsAddBegin(c *gin.Context) {}

// @Summary Add credential (complete)
// @Tags auth
// @Accept json
// @Param request body auth.CredentialsAddCompleteRequest true "Complete credential add"
// @Success 204
// @Router /api/v1/auth/credentials/add/complete [post]
func DocAuthCredentialsAddComplete(c *gin.Context) {}

// @Summary Delete credential
// @Tags auth
// @Param credentialId path string true "Credential ID"
// @Success 204
// @Router /api/v1/auth/credentials/{credentialId} [delete]
func DocAuthCredentialsDelete(c *gin.Context) {}

// @Summary Update credential device name
// @Tags auth
// @Param id path string true "Credential ID (base64url)"
// @Accept json
// @Produce json
// @Success 204
// @Router /api/v1/auth/credentials/{id} [patch]
func DocAuthCredentialsUpdate(c *gin.Context) {}

// @Summary Step-up (begin)
// @Tags auth
// @Produce json
// @Success 200 {object} auth.PublicKeyOptionsResponse
// @Router /api/v1/auth/step-up/begin [post]
func DocAuthStepUpBegin(c *gin.Context) {}

// @Summary Step-up (complete)
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.StepUpCompleteRequest true "Step-up complete request"
// @Success 200 {object} auth.AccessTokenResponse
// @Router /api/v1/auth/step-up/complete [post]
func DocAuthStepUpComplete(c *gin.Context) {}

// @Summary Refresh access token
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} auth.AccessTokenResponse
// @Router /api/v1/auth/refresh [post]
func DocAuthRefresh(c *gin.Context) {}

// @Summary Logout current session
// @Tags auth
// @Success 204
// @Router /api/v1/auth/logout [post]
func DocAuthLogout(c *gin.Context) {}

// @Summary Logout all sessions
// @Tags auth
// @Success 204
// @Router /api/v1/auth/logout-all [post]
func DocAuthLogoutAll(c *gin.Context) {}

// @Summary Me
// @Tags auth
// @Produce json
// @Success 200 {object} auth.MeResponse
// @Router /api/v1/auth/me [get]
func DocAuthMeGet(c *gin.Context) {}

// @Summary Update profile (full name)
// @Tags auth
// @Accept json
// @Produce json
// @Router /api/v1/auth/me [patch]
func DocAuthMePatch(c *gin.Context) {}

// @Summary Session
// @Tags auth
// @Produce json
// @Success 200 {object} auth.SessionResponse
// @Router /api/v1/auth/session [get]
func DocAuthSession(c *gin.Context) {}

// @Summary List active sessions
// @Tags auth
// @Produce json
// @Success 200 {object} auth.SessionsListResponse
// @Router /api/v1/auth/sessions [get]
func DocAuthSessionsList(c *gin.Context) {}

// @Summary Revoke a session
// @Tags auth
// @Param family_id path string true "Family ID"
// @Success 204
// @Router /api/v1/auth/sessions/{family_id} [delete]
func DocAuthSessionsDelete(c *gin.Context) {}

// @Summary Onboard begin
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.OnboardBeginRequest true "Onboard begin request"
// @Success 200 {object} auth.OnboardBeginResponse
// @Router /api/v1/auth/onboard/begin [post]
func DocAuthOnboardBegin(c *gin.Context) {}

// @Summary Onboard complete
// @Tags auth
// @Accept json
// @Param request body auth.OnboardCompleteRequest true "Onboard complete request"
// @Success 204
// @Router /api/v1/auth/onboard/complete [post]
func DocAuthOnboardComplete(c *gin.Context) {}

// @Summary Admin bootstrap
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.BootstrapRequest true "Bootstrap request"
// @Success 201 {object} auth.BootstrapResponse
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/auth/bootstrap [post]
func DocAuthBootstrap(c *gin.Context) {}

// @Summary Exchange GitHub OIDC token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ghaExchangeRequest true "GitHub OIDC exchange request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/auth/oidc/github/exchange [post]
func DocAuthGithubExchange(c *gin.Context) {}

// @Summary Create GitHub policy
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.GithubPolicyCreateRequest true "Create policy"
// @Success 201 {object} auth.GithubPolicyResponse
// @Router /api/v1/auth/oidc/github/policies [post]
func DocAuthGithubPolicyCreate(c *gin.Context) {}

// @Summary List GitHub policies
// @Tags auth
// @Produce json
// @Success 200 {array} auth.GithubPolicyResponse
// @Router /api/v1/auth/oidc/github/policies [get]
func DocAuthGithubPolicyList(c *gin.Context) {}

// @Summary Get GitHub policy
// @Tags auth
// @Produce json
// @Param id path string true "Policy ID"
// @Success 200 {object} auth.GithubPolicyResponse
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/auth/oidc/github/policies/{id} [get]
func DocAuthGithubPolicyGet(c *gin.Context) {}

// @Summary Update GitHub policy
// @Tags auth
// @Accept json
// @Param id path string true "Policy ID"
// @Param request body auth.GithubPolicyUpdateRequest true "Update policy"
// @Success 204
// @Router /api/v1/auth/oidc/github/policies/{id} [put]
func DocAuthGithubPolicyUpdate(c *gin.Context) {}

// @Summary Delete GitHub policy
// @Tags auth
// @Param id path string true "Policy ID"
// @Success 204
// @Router /api/v1/auth/oidc/github/policies/{id} [delete]
func DocAuthGithubPolicyDelete(c *gin.Context) {}

// @Summary Recovery init
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.RecoveryInitRequest true "Recovery init request"
// @Success 200 {object} auth.RecoveryInitResponse
// @Router /api/v1/auth/recovery/init [post]
func DocAuthRecoveryInit(c *gin.Context) {}

// @Summary Recovery verify
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.RecoveryVerifyRequest true "Recovery verify request"
// @Success 200 {object} auth.RecoveryVerifyResponse
// @Router /api/v1/auth/recovery/verify [post]
func DocAuthRecoveryVerify(c *gin.Context) {}

// @Summary Recovery register begin
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.RecoveryRegisterBeginRequest true "Recovery register begin request"
// @Success 200 {object} auth.PublicKeyOptionsResponse
// @Router /api/v1/auth/recovery/register/begin [post]
func DocAuthRecoveryRegisterBegin(c *gin.Context) {}

// @Summary Recovery register complete
// @Tags auth
// @Accept json
// @Param request body auth.RecoveryRegisterCompleteRequest true "Recovery register complete request"
// @Success 204
// @Router /api/v1/auth/recovery/register/complete [post]
func DocAuthRecoveryRegisterComplete(c *gin.Context) {}

// @Summary Generate new recovery codes (one-time view)
// @Tags auth
// @Produce json
// @Success 200 {object} auth.RecoveryGenerateResponse
// @Router /api/v1/auth/recovery/codes/generate [post]
func DocAuthRecoveryCodesGenerate(c *gin.Context) {}

// Device-link endpoints

// @Summary Begin device link flow
// @Description Initiates a device authorization flow for CLI/device authentication
// @Tags auth
// @Accept json
// @Produce json
// @Param request body BeginDeviceLinkRequest true "Device link request"
// @Success 200 {object} service.DeviceLinkResponse
// @Router /api/v1/auth/device-link/begin [post]
func DocAuthDeviceLinkBegin(c *gin.Context) {}

// @Summary Authorize device link
// @Description Authorizes a pending device link request (requires authentication and step-up)
// @Tags auth
// @Accept json
// @Produce json
// @Param request body AuthorizeDeviceLinkRequest true "Authorization request"
// @Success 204
// @Router /api/v1/auth/device-link/authorize [post]
func DocAuthDeviceLinkAuthorize(c *gin.Context) {}

// @Summary Exchange device code for tokens
// @Description Exchanges a device code for access and refresh tokens (polling endpoint)
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ExchangeDeviceCodeRequest true "Exchange request"
// @Success 200 {object} service.ExchangeResponse
// @Router /api/v1/auth/device-link/exchange [post]
func DocAuthDeviceLinkExchange(c *gin.Context) {}

// @Summary Verify device link code
// @Description Verifies a user code and returns device information for display in the browser UI
// @Tags auth
// @Accept json
// @Produce json
// @Param code query string true "User code to verify"
// @Success 200 {object} auth.DeviceLinkVerificationResponse
// @Router /api/v1/auth/device-link/verify [get]
func DocAuthDeviceLinkVerify(c *gin.Context) {}

// Device management endpoints

// @Summary List user devices
// @Description Lists all devices registered for the authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} auth.DeviceListResponse
// @Router /api/v1/auth/devices [get]
func DocAuthDevicesList(c *gin.Context) {}

// @Summary Get device details
// @Description Gets details of a specific device
// @Tags auth
// @Accept json
// @Produce json
// @Param id path string true "Device ID"
// @Success 200 {object} auth.DeviceResponse
// @Router /api/v1/auth/devices/{id} [get]
func DocAuthDevicesGet(c *gin.Context) {}

// @Summary Revoke device
// @Description Revokes a device and all associated refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param id path string true "Device ID"
// @Success 204
// @Router /api/v1/auth/devices/{id} [delete]
func DocAuthDevicesDelete(c *gin.Context) {}

// Admin endpoints (documentation stubs)

// @Summary Preview invite (public)
// @Tags admin
// @Produce json
// @Param token query string false "Invite token (base64url)"
// @Param id query string false "Invite ID (UUID)"
// @Success 200 {object} auth.InvitePreviewResponse
// @Router /api/v1/admin/invites/preview [get]
func DocAdminInvitePreview(c *gin.Context) {}

// @Summary Create invite (admin)
// @Tags admin
// @Accept json
// @Produce json
// @Param request body auth.AdminInviteCreateRequest true "Invite create request"
// @Success 201 {object} auth.AdminInviteCreateResponse
// @Router /api/v1/admin/invites [post]
func DocAdminInviteCreate(c *gin.Context) {}

// @Summary List users (admin)
// @Tags admin
// @Produce json
// @Param q query string false "Search query (email or ID)"
// @Param role query string false "Filter by role (admin|member)"
// @Param limit query integer false "Max results (1-200)" default(50)
// @Param offset query integer false "Offset for pagination" default(0)
// @Success 200 {object} auth.AdminUsersListResponse
// @Router /api/v1/admin/users [get]
func DocAdminUsersList(c *gin.Context) {}

// @Summary List user credentials (admin)
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} auth.CredentialsListResponse
// @Router /api/v1/admin/users/{id}/credentials [get]
func DocAdminUserCredentials(c *gin.Context) {}

// @Summary Generate recovery codes for a user (admin)
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} auth.RecoveryGenerateResponse
// @Router /api/v1/admin/users/{id}/recovery/codes/generate [post]
func DocAdminUserRecoveryCodes(c *gin.Context) {}

// @Summary List audit events (admin)
// @Tags admin
// @Produce json
// @Param actor_id query string false "Filter by actor ID (UUID)"
// @Param user_id query string false "Filter by user ID (UUID)"
// @Param types query string false "Comma-separated event types"
// @Param since query string false "Filter events created at or after this RFC3339 timestamp"
// @Param until query string false "Filter events created before this RFC3339 timestamp"
// @Param limit query integer false "Max results (1-200)" default(50)
// @Param offset query integer false "Offset for pagination" default(0)
// @Success 200 {object} auth.AuditListResponse
// @Router /api/v1/admin/audit [get]
func DocAdminAuditList(c *gin.Context) {}

// @Summary List access requests (admin)
// @Tags admin
// @Produce json
// @Param status query string false "Filter by status (pending|approved|rejected)"
// @Param q query string false "Search query (email or reason)"
// @Param limit query integer false "Max results (1-200)" default(50)
// @Param offset query integer false "Offset for pagination" default(0)
// @Success 200 {object} auth.AccessRequestListResponse
// @Router /api/v1/admin/access-requests [get]
func DocAdminAccessRequestsList(c *gin.Context) {}

// @Summary Decide access request (admin)
// @Tags admin
// @Accept json
// @Param id path string true "Access Request ID"
// @Param request body auth.AccessRequestDecideRequest true "Decision"
// @Success 204
// @Router /api/v1/admin/access-requests/{id} [patch]
func DocAdminAccessRequestDecide(c *gin.Context) {}

// @Summary Update a user (admin)
// @Tags admin
// @Accept json
// @Param id path string true "User ID"
// @Success 204
// @Router /api/v1/admin/users/{id} [patch]
func DocAdminUserUpdate(c *gin.Context) {}

// @Summary Delete a user (admin)
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Success 204
// @Router /api/v1/admin/users/{id} [delete]
func DocAdminUserDelete(c *gin.Context) {}

// Silence unused import warnings for documentation-only type references
var _ = []interface{}{
	(*service.DeviceLinkResponse)(nil),
	(*service.ExchangeResponse)(nil),
}

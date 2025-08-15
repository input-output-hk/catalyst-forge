package routes

import (
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	authhandlers "github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers/auth"
	apiauth "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit"
	akservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/service"
)

// bindAuthKitRoutes binds swagger-annotated API routes to concrete API handlers.
func bindAuthKitRoutes(
	r *gin.Engine,
	refresh akservice.RefreshService,
	csrf httpkit.CSRF,
	cookie httpkit.CookieConfig,
	wa akservice.WebAuthnService,
	tokens akservice.TokenService,
	refreshTTL time.Duration,
	stepUpTTL time.Duration,
	bootstrapToken string,
	origin string,
	ghEnabled bool,
	ghIssuer string,
	ghAudiences []string,
	ghExchangeTTL time.Duration,
) {
	_ = r.Group("/api/v1/auth")

	// @Summary Begin login
	// @Tags auth
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/login/begin [post]
	authhandlers.RegisterLoginBegin(r, wa)

	// @Summary Complete login
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/login/complete [post]
	authhandlers.RegisterLoginComplete(r, wa, refresh, tokens, cookie, refreshTTL)

	// @Summary List credentials
	// @Tags auth
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/credentials [get]
	authhandlers.RegisterCredentials(r, authhandlers.CredentialsDeps{Credentials: apiauth.BuildDepsCached().Stores.Credentials})

	// @Summary Add credential (begin)
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/credentials/add/begin [post]
	{
		deps := apiauth.BuildDepsCached()
		authhandlers.RegisterCredentialsAdd(r, authhandlers.CredentialsAddDeps{Users: deps.Stores.Users, WebAuthn: wa})
	}

	// @Summary Add credential (complete)
	// @Tags auth
	// @Accept json
	// @Success 204
	// @Router /api/v1/auth/credentials/add/complete [post]
	// bound via RegisterCredentialsAdd

	// @Summary Delete credential
	// @Tags auth
	// @Param credentialId path string true "Credential ID"
	// @Success 204
	// @Router /api/v1/auth/credentials/{credentialId} [delete]
	// bound via RegisterCredentials

	// @Summary Step-up (begin)
	// @Tags auth
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/step-up/begin [post]
	authhandlers.RegisterStepUpBegin(r, wa, apiauth.BuildDepsCached().Stores.Users)

	// @Summary Step-up (complete)
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/step-up/complete [post]
	authhandlers.RegisterStepUpComplete(r, wa, tokens, stepUpTTL)

	// @Summary Refresh access token
	// @Tags auth
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/refresh [post]
	authhandlers.RegisterRefresh(r, authhandlers.RefreshDeps{CSRF: csrf, Service: refresh, Cookie: cookie})

	// @Summary Logout current session
	// @Tags auth
	// @Success 204
	// @Router /api/v1/auth/logout [post]
	authhandlers.RegisterLogout(r, authhandlers.LogoutDeps{Service: refresh, Cookie: cookie})

	// @Summary Logout all sessions
	// @Tags auth
	// @Success 204
	// @Router /api/v1/auth/logout-all [post]
	authhandlers.RegisterLogoutAll(r, authhandlers.LogoutAllDeps{Service: refresh})

	// @Summary Me
	// @Tags auth
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/me [get]
	authhandlers.RegisterMeSession(r)

	// @Summary Create invite
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 201 {object} map[string]interface{}
	// @Router /api/v1/auth/invites [post]
	// invites TBD

	// @Summary Onboard begin
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/onboard/begin [post]
	{
		deps := apiauth.BuildDepsCached()
		authhandlers.RegisterOnboarding(r, authhandlers.OnboardDeps{Invites: deps.Stores.Invites, Users: deps.Stores.Users, Refresh: refresh, Tokens: tokens}, wa, cookie, refreshTTL)
	}

	// Bootstrap admin (one-time). Returns 201 on success; 404 on invalid/used token
	// @Summary Bootstrap admin account
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 201 {object} map[string]interface{}
	// @Router /api/v1/auth/bootstrap [post]
	{
		deps := apiauth.BuildDepsCached()
		bs := akservice.NewBootstrapService(bootstrapToken, deps.Stores.Users, deps.Stores.Bootstrap)
		authhandlers.RegisterBootstrap(r, bs, tokens, refresh)
	}

	// GitHub OIDC exchange (public)
	// @Summary Exchange GitHub OIDC ID token for API access token
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/oidc/github/exchange [post]
	{
		deps := apiauth.BuildDepsCached()
		v := akservice.NewGHAVerifier(akservice.GHAVerifierConfig{Issuer: ghIssuer, Audiences: ghAudiences, JWKSCacheTTL: 15 * time.Minute})
		s := akservice.NewGHAExchangeService(akservice.GHAConfig{Enabled: ghEnabled, ExchangeTTL: ghExchangeTTL}, v, deps.Stores.GithubPolicies, tokens)
		authhandlers.RegisterGithubExchange(r, s)
	}

	// GitHub OIDC policy management (admin, guard to be added via policies)
	{
		deps := apiauth.BuildDepsCached()
		authhandlers.RegisterGithubPolicies(r, authhandlers.GithubPolicyDeps{Store: deps.Stores.GithubPolicies})
	}

	// @Summary Onboard complete
	// @Tags auth
	// @Accept json
	// @Success 204
	// @Router /api/v1/auth/onboard/complete [post]
	// bound via RegisterOnboarding

	// @Summary Recovery init
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/recovery/init [post]
	{
		deps := apiauth.BuildDepsCached()
		authhandlers.RegisterRecovery(r, authhandlers.RecoveryDeps{Flow: akservice.NewRecoveryFlowService(deps.Stores.Users, akservice.NewRecoveryService(deps.Stores.RecoveryCodes, deps.Rand), deps.KV, deps.Rand)}, wa)
	}

	// @Summary Recovery verify
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/recovery/verify [post]
	// bound via RegisterRecovery

	// @Summary Recovery register begin
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/recovery/register/begin [post]
	// bound via RegisterRecovery

	// @Summary Recovery register complete
	// @Tags auth
	// @Accept json
	// @Success 204
	// @Router /api/v1/auth/recovery/register/complete [post]
	// bound via RegisterRecovery

	// Device-link endpoints for CLI authentication
	// @Summary Begin device link flow
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/device-link/begin [post]
	{
		deps := apiauth.BuildDepsCached()
		deviceLinkService := akservice.NewDeviceLinkService(akservice.DeviceLinkConfig{
			LinkStore:      deps.Stores.DeviceLinks,
			DeviceStore:    deps.Stores.Devices,
			UserStore:      deps.Stores.Users,
			RefreshStore:   deps.Stores.Refresh,
			TokenService:   tokens,
			RefreshService: refresh,
			Rand:           deps.Rand,
			Origin:         origin,
			LinkTTL:        10 * time.Minute,
			PollInterval:   5,
			AccessTTL:      30 * time.Minute,
			RefreshTTL:     refreshTTL,
		})
		authhandlers.RegisterDeviceLink(r, authhandlers.DeviceLinkDeps{
			DeviceLinkService: deviceLinkService,
			UserStore:         deps.Stores.Users,
		})
		authhandlers.RegisterDeviceLinkVerify(r, deps.Stores.DeviceLinks)
	}

	// @Summary Authorize device link
	// @Tags auth
	// @Accept json
	// @Success 204
	// @Router /api/v1/auth/device-link/authorize [post]
	// bound via RegisterDeviceLink

	// @Summary Exchange device code for tokens
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/device-link/exchange [post]
	// bound via RegisterDeviceLink

	// @Summary Verify device code
	// @Tags auth
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/device-link/verify [get]
	// bound via RegisterDeviceLinkVerify

	// Device management endpoints
	// @Summary List user devices
	// @Tags auth
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/devices [get]
	{
		deps := apiauth.BuildDepsCached()
		authhandlers.RegisterDeviceManagement(r, authhandlers.DeviceManagementDeps{
			DeviceStore:  deps.Stores.Devices,
			RefreshStore: deps.Stores.Refresh,
			AuditStore:   deps.Stores.Audit,
		})
	}

	// @Summary Get device details
	// @Tags auth
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/devices/{id} [get]
	// bound via RegisterDeviceManagement

	// @Summary Revoke device
	// @Tags auth
	// @Success 204
	// @Router /api/v1/auth/devices/{id} [delete]
	// bound via RegisterDeviceManagement
}

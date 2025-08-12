package api

import (
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/rate"
	auditrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/audit"
	buildrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/build"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
	emailsvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/email"
	pca "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/pca"

	userservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	ghauth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/github"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// SetupRouter configures the Gin router.
func SetupRouter(
	releaseService service.ReleaseService,
	deploymentService service.DeploymentService,
	userService userservice.UserService,
	roleService userservice.RoleService,
	userRoleService userservice.UserRoleService,
	userKeyService userservice.UserKeyService,
	am *middleware.AuthMiddleware,
	db *gorm.DB,
	logger *slog.Logger,
	jwtManager jwt.JWTManager,
	ghaOIDCClient ghauth.GithubActionsOIDCClient,
	ghaAuthService service.GithubAuthService,

	emailService emailsvc.Service,
	sessionMaxActive int,
	enablePerIPRateLimit bool,
	pcaClient pca.PCAClient,
	authConfig *config.Config, // Add authConfig parameter
) *gin.Engine {
	r := gin.New()

	// Middleware Setup //

	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))
	// General CORS: allow credentials and specific origins when set via PUBLIC_BASE_URL
	// (Auth endpoints use specialized auth CORS middleware)
	r.Use(func(c *gin.Context) {
		// Skip CORS for auth endpoints - they use specialized auth CORS middleware
		if strings.HasPrefix(c.Request.URL.Path, "/auth/") {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		base := c.GetString("public_base_url")
		if base == "" {
			base = "" // keep default no CORS if not configured
		}
		// Only set CORS headers if origin matches the configured base URL
		if origin != "" && base != "" && origin == base {
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
			// Set the origin only if it matches our configured base
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-CLI")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		} else if origin != "" && base != "" {
			// Origin doesn't match configured base - set Vary header but no CORS headers
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == "OPTIONS" {
			c.Status(204)
			c.Abort()
			return
		}
		c.Next()
	})
	r.Use(func(c *gin.Context) {
		c.Set("releaseService", releaseService)
		c.Set("deploymentService", deploymentService)
		c.Set("userService", userService)
		c.Next()
	})

	releaseHandler := handlers.NewReleaseHandler(releaseService, logger)
	deploymentHandler := handlers.NewDeploymentHandler(deploymentService, logger)
	healthHandler := handlers.NewHealthHandler(db, logger)

	// User handlers
	userHandler := user.NewUserHandler(userService, logger)
	roleHandler := user.NewRoleHandler(roleService, logger)
	userRoleHandler := user.NewUserRoleHandler(userRoleService, logger)
	userKeyHandler := user.NewUserKeyHandler(userKeyService, logger, jwtManager)

	// Legacy Ed25519 auth handler removed in Task 5.2 - replaced by device-keypair authentication

	// Invite handler
	inviteRepo := userrepo.NewInviteRepository(db)
	// email service is optional and passed from server main
	inviteHandler := handlers.NewInviteHandler(
		inviteRepo,
		userService,
		roleService,
		userRoleService,
		72*60*60*1e9,
		emailService,
		authConfig.Auth.InviteMaxAttempts,
		authConfig.Auth.InviteLockDuration,
	)

	// Bootstrap handler
	bootstrapRepo := userrepo.NewBootstrapTokenRepository(db)
	bootstrapHandler := handlers.NewBootstrapHandler(
		inviteHandler,
		userService,
		roleService,
		bootstrapRepo,
		inviteRepo,
		authConfig.Auth.BootstrapToken,
		db,
	)

	// GitHub handler
	githubHandler := handlers.NewGithubHandler(jwtManager, ghaOIDCClient, ghaAuthService, logger)

	// Certificate handler
	certificateHandler := handlers.NewCertificateHandler(jwtManager)
	if pcaClient != nil {
		certificateHandler = certificateHandler.WithPCA(pcaClient)
	}
	// JWKS handler (public)
	jwksHandler := handlers.NewJWKSHandler(jwtManager)
	// Legacy device handler removed in Task 5.1 - replaced by device-keypair authentication

	// Device registration handler (new device-keypair authentication)
	deviceRepo := userrepo.NewDeviceRepository(db)
	deviceRefreshRepo := userrepo.NewRefreshTokenRepository(db)
	deviceRegistrationHandler := handlers.NewDeviceRegistrationHandler(
		inviteRepo,
		deviceRepo,
		deviceRefreshRepo,
		userService,
		roleService,
		userRoleService,
		jwtManager,
		&authConfig.Auth,
		logger,
	)

	// Device refresh handler (new token refresh with device proof)
	deviceRefreshHandler := handlers.NewDeviceRefreshHandler(
		deviceRepo,
		deviceRefreshRepo,
		userService,
		roleService,
		userRoleService,
		jwtManager,
		&authConfig.Auth,
		logger,
	)

	// Device logout handler (new device-keypair logout)
	deviceLogoutHandler := handlers.NewDeviceLogoutHandler(
		deviceRepo,
		deviceRefreshRepo,
		&authConfig.Auth,
		logger,
	)

	// Device returning-login handler (sign challenge with stored device key)
	deviceLoginHandler := handlers.NewDeviceLoginHandler(
		deviceRepo,
		deviceRefreshRepo,
		userService,
		roleService,
		userRoleService,
		jwtManager,
		&authConfig.Auth,
		logger,
	)

	// Device management handler (new device management endpoints)
	deviceManagementHandler := handlers.NewDeviceManagementHandler(
		deviceRepo,
		deviceRefreshRepo,
		userService,
		jwtManager,
		&authConfig.Auth,
		logger,
	)

	// Auth CORS middleware for /auth/* endpoints
	authCORSMiddleware := middleware.NewAuthCORSMiddleware(&authConfig.Auth, logger)

	// Auth rate limiting middleware for /auth/* endpoints
	rateLimiter := rate.NewInMemoryLimiter()
	authRateLimitMiddleware := middleware.NewAuthRateLimitMiddleware(rateLimiter, &authConfig.Auth, logger)

	// Legacy session handler removed in Task 5.1 - replaced by device-keypair authentication
	// Audit repo (set in context for handlers that choose to log)
	auditRepo := auditrepo.NewLogRepository(db)
	// Build session handler
	buildSessRepo := buildrepo.NewBuildSessionRepository(db)
	buildHandler := handlers.NewBuildHandler(buildSessRepo, sessionMaxActive, auditRepo)
	r.Use(func(c *gin.Context) { c.Set("auditRepo", auditRepo); c.Next() })

	// Health check endpoint
	r.GET("/healthz", healthHandler.CheckHealth)

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Public JWKS endpoint for token verification
	r.GET("/.well-known/jwks.json", jwksHandler.GetJWKS)

	// Route Setup //

	// Release endpoints
	r.POST("/release", am.ValidatePermissions([]auth.Permission{auth.PermReleaseWrite}), releaseHandler.CreateRelease)
	r.GET("/release/:id", am.ValidatePermissions([]auth.Permission{auth.PermReleaseRead}), releaseHandler.GetRelease)
	r.PUT("/release/:id", am.ValidatePermissions([]auth.Permission{auth.PermReleaseWrite}), releaseHandler.UpdateRelease)
	r.GET("/releases", am.ValidatePermissions([]auth.Permission{auth.PermReleaseRead}), releaseHandler.ListReleases)

	// Release aliases
	r.GET("/release/alias/:name", am.ValidatePermissions([]auth.Permission{auth.PermReleaseRead}), releaseHandler.GetReleaseByAlias)
	r.POST("/release/alias/:name", am.ValidatePermissions([]auth.Permission{auth.PermReleaseWrite}), releaseHandler.CreateAlias)
	r.DELETE("/release/alias/:name", am.ValidatePermissions([]auth.Permission{auth.PermReleaseWrite}), releaseHandler.DeleteAlias)
	r.GET("/release/:id/aliases", am.ValidatePermissions([]auth.Permission{auth.PermReleaseRead}), releaseHandler.ListAliases)

	// Deployment endpoints
	r.POST("/release/:id/deploy", am.ValidatePermissions([]auth.Permission{auth.PermDeploymentWrite}), deploymentHandler.CreateDeployment)
	r.GET("/release/:id/deploy/:deployId", am.ValidatePermissions([]auth.Permission{auth.PermDeploymentRead}), deploymentHandler.GetDeployment)
	r.PUT("/release/:id/deploy/:deployId", am.ValidatePermissions([]auth.Permission{auth.PermDeploymentWrite}), deploymentHandler.UpdateDeployment)
	r.GET("/release/:id/deployments", am.ValidatePermissions([]auth.Permission{auth.PermDeploymentRead}), deploymentHandler.ListDeployments)
	r.GET("/release/:id/deploy/latest", am.ValidatePermissions([]auth.Permission{auth.PermDeploymentRead}), deploymentHandler.GetLatestDeployment)

	// Deployment event endpoints
	r.POST("/release/:id/deploy/:deployId/events", am.ValidatePermissions([]auth.Permission{auth.PermDeploymentEventWrite}), deploymentHandler.AddDeploymentEvent)
	r.GET("/release/:id/deploy/:deployId/events", am.ValidatePermissions([]auth.Permission{auth.PermDeploymentEventRead}), deploymentHandler.GetDeploymentEvents)

	// GitHub authentication management endpoints (requires auth)
	r.POST("/auth/github", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthWrite}), githubHandler.CreateAuth)
	r.GET("/auth/github", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthRead}), githubHandler.ListAuths)
	r.GET("/auth/github/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthRead}), githubHandler.GetAuth)
	r.GET("/auth/github/repository/:repository", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthRead}), githubHandler.GetAuthByRepository)
	r.PUT("/auth/github/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthWrite}), githubHandler.UpdateAuth)
	r.DELETE("/auth/github/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthWrite}), githubHandler.DeleteAuth)

	// Registration endpoints (legacy) removed in single-org invite model

	// Authentication endpoints
	// Legacy Ed25519 challenge-response endpoints removed in Task 5.2 - replaced by device-keypair authentication
	r.POST("/auth/github/login", authCORSMiddleware.Handle(), githubHandler.ValidateToken)
	// Legacy session endpoints removed in Task 5.1 - replaced by /auth/refresh and /auth/logout

	// Bootstrap endpoint (unprotected, one-time use, rate-limited)
	r.POST("/auth/bootstrap", authCORSMiddleware.Handle(), authRateLimitMiddleware.Handle(), bootstrapHandler.Bootstrap)

	// Invite endpoints
	r.POST("/auth/invites", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), inviteHandler.CreateInvite)
	r.GET("/verify", inviteHandler.Verify)

	// Device registration endpoints (new device-keypair authentication)
	r.POST("/auth/devices/init", authCORSMiddleware.Handle(), authRateLimitMiddleware.Handle(), deviceRegistrationHandler.InitDeviceRegistration)
	r.POST("/auth/devices/register", authCORSMiddleware.Handle(), authRateLimitMiddleware.Handle(), deviceRegistrationHandler.RegisterDevice)

	// Returning device login endpoints (no invite/email)
	r.POST("/auth/devices/login/init", authCORSMiddleware.Handle(), authRateLimitMiddleware.Handle(), deviceLoginHandler.InitLogin)
	r.POST("/auth/devices/login", authCORSMiddleware.Handle(), authRateLimitMiddleware.Handle(), deviceLoginHandler.Login)

	// Device-bound refresh endpoint (new device-keypair authentication)
	r.POST("/auth/refresh", authCORSMiddleware.Handle(), authRateLimitMiddleware.Handle(), deviceRefreshHandler.RefreshToken)

	// Device-bound logout endpoint (new device-keypair authentication)
	r.POST("/auth/logout", authCORSMiddleware.Handle(), deviceLogoutHandler.Logout)

	// Device management endpoints (require JWT authentication)
	r.GET("/auth/devices", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserRead}), deviceManagementHandler.ListDevices)
	r.DELETE("/auth/devices/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), deviceManagementHandler.DeleteDevice)

	// Legacy device flow endpoints removed in Task 5.1 - replaced by device-keypair authentication

	// Build sessions
	r.POST("/build/sessions", am.ValidatePermissions([]auth.Permission{auth.PermDeploymentWrite}), buildHandler.CreateBuildSession)

	// Pending endpoints
	r.GET("/auth/pending/users", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserRead}), userHandler.GetPendingUsers)
	r.GET("/auth/pending/keys", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.GetInactiveUserKeys)

	// User endpoints
	r.POST("/auth/users", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), userHandler.CreateUser)
	r.GET("/auth/users", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserRead}), userHandler.ListUsers)
	r.GET("/auth/users/email/:email", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserRead}), userHandler.GetUserByEmail)
	r.GET("/auth/users/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserRead}), userHandler.GetUser)
	r.PUT("/auth/users/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), userHandler.UpdateUser)
	r.DELETE("/auth/users/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), userHandler.DeleteUser)
	r.POST("/auth/users/:id/activate", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), userHandler.ActivateUser)
	r.POST("/auth/users/:id/deactivate", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), userHandler.DeactivateUser)

	// User key endpoints
	r.POST("/auth/keys", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserKeyWrite}), userKeyHandler.CreateUserKey)
	r.POST("/auth/keys/bootstrap", authCORSMiddleware.Handle(), userKeyHandler.BootstrapKET)
	r.POST("/auth/keys/register", authCORSMiddleware.Handle(), userKeyHandler.RegisterWithKET)
	r.GET("/auth/keys", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.ListUserKeys)
	r.GET("/auth/keys/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.GetUserKey)
	r.GET("/auth/keys/kid/:kid", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.GetUserKeyByKid)
	r.PUT("/auth/keys/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserKeyWrite}), userKeyHandler.UpdateUserKey)
	r.DELETE("/auth/keys/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserKeyWrite}), userKeyHandler.DeleteUserKey)
	r.POST("/auth/keys/:id/revoke", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserKeyWrite}), userKeyHandler.RevokeUserKey)
	r.GET("/auth/keys/user/:user_id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.GetUserKeysByUserID)
	r.GET("/auth/keys/user/:user_id/active", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.GetActiveUserKeysByUserID)
	r.GET("/auth/keys/user/:user_id/inactive", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.GetInactiveUserKeysByUserID)

	// Role endpoints
	r.POST("/auth/roles", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermRoleWrite}), roleHandler.CreateRole)
	r.GET("/auth/roles", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermRoleRead}), roleHandler.ListRoles)
	r.GET("/auth/roles/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermRoleRead}), roleHandler.GetRole)
	r.GET("/auth/roles/name/:name", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermRoleRead}), roleHandler.GetRoleByName)
	r.PUT("/auth/roles/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermRoleWrite}), roleHandler.UpdateRole)
	r.DELETE("/auth/roles/:id", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermRoleWrite}), roleHandler.DeleteRole)

	// User-role endpoints
	r.POST("/auth/user-roles", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserWrite, auth.PermRoleWrite}), userRoleHandler.AssignUserToRole)
	r.DELETE("/auth/user-roles", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserWrite, auth.PermRoleWrite}), userRoleHandler.RemoveUserFromRole)
	r.GET("/auth/user-roles", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserRead, auth.PermRoleRead}), userRoleHandler.GetUserRoles)
	r.GET("/auth/role-users", authCORSMiddleware.Handle(), am.ValidatePermissions([]auth.Permission{auth.PermUserRead, auth.PermRoleRead}), userRoleHandler.GetRoleUsers)

	// Certificate endpoints
	r.POST("/certificates/sign", am.ValidateAnyCertificatePermission(), certificateHandler.SignCertificate)
	r.POST("/ca/buildkit/server-certificates", am.ValidatePermissions([]auth.Permission{auth.PermCertificateSignAll}), certificateHandler.SignServerCertificate)
	r.GET("/certificates/root", certificateHandler.GetRootCertificate)

	// Optional ext_authz (feature-flagged)
	r.POST("/build/gateway/authorize", certificateHandler.AuthorizeBuildGateway)

	return r
}

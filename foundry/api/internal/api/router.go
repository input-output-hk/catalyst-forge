package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	ca "github.com/input-output-hk/catalyst-forge/foundry/api/internal/ca"
	auditrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/audit"
	buildrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/build"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
	emailsvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/email"
	pca "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/pca"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/stepca"
	userservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	ghauth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/github"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// SetupRouter configures the Gin router
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
	stepCAClient *stepca.Client,
	emailService emailsvc.Service,
	sessionMaxActive int,
	enablePerIPRateLimit bool,
	clientsProv *ca.StepCAClient,
	serversProv *ca.StepCAClient,
	pcaClient pca.PCAClient,
) *gin.Engine {
	r := gin.New()

	// Middleware Setup //

	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))
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

	// Auth handler
	authManager := auth.NewAuthManager()
	authHandler := handlers.NewAuthHandler(userKeyService, userService, userRoleService, roleService, authManager, jwtManager, logger)

	// Invite handler
	inviteRepo := userrepo.NewInviteRepository(db)
	// email service is optional and passed from server main
	inviteHandler := handlers.NewInviteHandler(inviteRepo, userService, roleService, userRoleService, 72*60*60*1e9, emailService)

	// GitHub handler
	githubHandler := handlers.NewGithubHandler(jwtManager, ghaOIDCClient, ghaAuthService, logger)

	// Certificate handler
	certificateHandler := handlers.NewCertificateHandler(jwtManager, stepCAClient, clientsProv, serversProv)
	if pcaClient != nil {
		certificateHandler = certificateHandler.WithPCA(pcaClient)
	}
	// JWKS handler (public)
	jwksHandler := handlers.NewJWKSHandler(jwtManager)
	// Device handler
	deviceSessRepo := userrepo.NewDeviceSessionRepository(db)
	deviceRepo := userrepo.NewDeviceRepository(db)
	deviceRefreshRepo := userrepo.NewRefreshTokenRepository(db)
	deviceHandler := handlers.NewDeviceHandler(deviceSessRepo, deviceRepo, deviceRefreshRepo, userService, roleService, userRoleService, jwtManager, logger)
	// Token handler
	refreshRepo := userrepo.NewRefreshTokenRepository(db)
	tokenHandler := handlers.NewTokenHandler(refreshRepo, userService, roleService, userRoleService, jwtManager)
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
	r.POST("/auth/github", am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthWrite}), githubHandler.CreateAuth)
	r.GET("/auth/github", am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthRead}), githubHandler.ListAuths)
	r.GET("/auth/github/:id", am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthRead}), githubHandler.GetAuth)
	r.GET("/auth/github/repository/:repository", am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthRead}), githubHandler.GetAuthByRepository)
	r.PUT("/auth/github/:id", am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthWrite}), githubHandler.UpdateAuth)
	r.DELETE("/auth/github/:id", am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthWrite}), githubHandler.DeleteAuth)

	// Registration endpoints (legacy) removed in single-org invite model

	// Authentication endpoints
	r.POST("/auth/challenge", authHandler.CreateChallenge)
	r.POST("/auth/login", authHandler.Login)
	r.POST("/auth/github/login", githubHandler.ValidateToken)
	r.POST("/tokens/refresh", tokenHandler.Refresh)
	r.POST("/tokens/revoke", tokenHandler.Revoke)

	// Invite endpoints
	r.POST("/auth/invites", am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), inviteHandler.CreateInvite)
	r.GET("/verify", inviteHandler.Verify)

	// Device flow endpoints
	r.POST("/device/init", deviceHandler.Init)
	// Build sessions
	r.POST("/build/sessions", am.ValidatePermissions([]auth.Permission{auth.PermDeploymentWrite}), buildHandler.CreateBuildSession)
	r.POST("/device/token", deviceHandler.Token)
	r.POST("/device/approve", am.ValidatePermissions([]auth.Permission{}), deviceHandler.Approve)

	// Pending endpoints
	r.GET("/auth/pending/users", am.ValidatePermissions([]auth.Permission{auth.PermUserRead}), userHandler.GetPendingUsers)
	r.GET("/auth/pending/keys", am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.GetInactiveUserKeys)

	// User endpoints
	r.POST("/auth/users", am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), userHandler.CreateUser)
	r.GET("/auth/users", am.ValidatePermissions([]auth.Permission{auth.PermUserRead}), userHandler.ListUsers)
	r.GET("/auth/users/email/:email", am.ValidatePermissions([]auth.Permission{auth.PermUserRead}), userHandler.GetUserByEmail)
	r.GET("/auth/users/:id", am.ValidatePermissions([]auth.Permission{auth.PermUserRead}), userHandler.GetUser)
	r.PUT("/auth/users/:id", am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), userHandler.UpdateUser)
	r.DELETE("/auth/users/:id", am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), userHandler.DeleteUser)
	r.POST("/auth/users/:id/activate", am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), userHandler.ActivateUser)
	r.POST("/auth/users/:id/deactivate", am.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), userHandler.DeactivateUser)

	// User key endpoints
	r.POST("/auth/keys", am.ValidatePermissions([]auth.Permission{auth.PermUserKeyWrite}), userKeyHandler.CreateUserKey)
	r.POST("/auth/keys/bootstrap", userKeyHandler.BootstrapKET)
	r.POST("/auth/keys/register", userKeyHandler.RegisterWithKET)
	r.GET("/auth/keys", am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.ListUserKeys)
	r.GET("/auth/keys/:id", am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.GetUserKey)
	r.GET("/auth/keys/kid/:kid", am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.GetUserKeyByKid)
	r.PUT("/auth/keys/:id", am.ValidatePermissions([]auth.Permission{auth.PermUserKeyWrite}), userKeyHandler.UpdateUserKey)
	r.DELETE("/auth/keys/:id", am.ValidatePermissions([]auth.Permission{auth.PermUserKeyWrite}), userKeyHandler.DeleteUserKey)
	r.POST("/auth/keys/:id/revoke", am.ValidatePermissions([]auth.Permission{auth.PermUserKeyWrite}), userKeyHandler.RevokeUserKey)
	r.GET("/auth/keys/user/:user_id", am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.GetUserKeysByUserID)
	r.GET("/auth/keys/user/:user_id/active", am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.GetActiveUserKeysByUserID)
	r.GET("/auth/keys/user/:user_id/inactive", am.ValidatePermissions([]auth.Permission{auth.PermUserKeyRead}), userKeyHandler.GetInactiveUserKeysByUserID)

	// Role endpoints
	r.POST("/auth/roles", am.ValidatePermissions([]auth.Permission{auth.PermRoleWrite}), roleHandler.CreateRole)
	r.GET("/auth/roles", am.ValidatePermissions([]auth.Permission{auth.PermRoleRead}), roleHandler.ListRoles)
	r.GET("/auth/roles/:id", am.ValidatePermissions([]auth.Permission{auth.PermRoleRead}), roleHandler.GetRole)
	r.GET("/auth/roles/name/:name", am.ValidatePermissions([]auth.Permission{auth.PermRoleRead}), roleHandler.GetRoleByName)
	r.PUT("/auth/roles/:id", am.ValidatePermissions([]auth.Permission{auth.PermRoleWrite}), roleHandler.UpdateRole)
	r.DELETE("/auth/roles/:id", am.ValidatePermissions([]auth.Permission{auth.PermRoleWrite}), roleHandler.DeleteRole)

	// User-role endpoints
	r.POST("/auth/user-roles", am.ValidatePermissions([]auth.Permission{auth.PermUserWrite, auth.PermRoleWrite}), userRoleHandler.AssignUserToRole)
	r.DELETE("/auth/user-roles", am.ValidatePermissions([]auth.Permission{auth.PermUserWrite, auth.PermRoleWrite}), userRoleHandler.RemoveUserFromRole)
	r.GET("/auth/user-roles", am.ValidatePermissions([]auth.Permission{auth.PermUserRead, auth.PermRoleRead}), userRoleHandler.GetUserRoles)
	r.GET("/auth/role-users", am.ValidatePermissions([]auth.Permission{auth.PermUserRead, auth.PermRoleRead}), userRoleHandler.GetRoleUsers)

	// Certificate endpoints
	r.POST("/certificates/sign", am.ValidateAnyCertificatePermission(), certificateHandler.SignCertificate)
	r.POST("/ca/buildkit/server-certificates", am.ValidatePermissions([]auth.Permission{auth.PermCertificateSignAll}), certificateHandler.SignServerCertificate)
	r.GET("/certificates/root", certificateHandler.GetRootCertificate)

	// Optional ext_authz (feature-flagged)
	r.POST("/build/gateway/authorize", certificateHandler.AuthorizeBuildGateway)

	return r
}

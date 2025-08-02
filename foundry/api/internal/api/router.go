package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
	userservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
	ghauth "github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth/github"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth/jwt"
	"github.com/redis/go-redis/v9"
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
	jwtManager *jwt.JWTManager,
	ghaOIDCClient ghauth.GithubActionsOIDCClient,
	ghaAuthService service.GithubAuthService,
	redisClient *redis.Client,
) *gin.Engine {
	r := gin.New()

	// Middleware Setup //

	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))
	r.Use(func(c *gin.Context) {
		c.Set("releaseService", releaseService)
		c.Set("deploymentService", deploymentService)
		c.Set("userService", userService)
		c.Set("redisClient", redisClient)
		c.Next()
	})

	releaseHandler := handlers.NewReleaseHandler(releaseService, logger)
	deploymentHandler := handlers.NewDeploymentHandler(deploymentService, logger)
	healthHandler := handlers.NewHealthHandler(db, logger)

	// User handlers
	userHandler := user.NewUserHandler(userService, logger)
	roleHandler := user.NewRoleHandler(roleService, logger)
	userRoleHandler := user.NewUserRoleHandler(userRoleService, logger)
	userKeyHandler := user.NewUserKeyHandler(userKeyService, logger)

	// Auth handler
	authManager := auth.NewAuthManager(auth.WithRedis(redisClient))
	authHandler := handlers.NewAuthHandler(userKeyService, userService, userRoleService, roleService, authManager, jwtManager, logger)

	// GitHub handler
	githubHandler := handlers.NewGithubHandler(jwtManager, ghaOIDCClient, ghaAuthService, logger)

	// Health check endpoint
	r.GET("/healthz", healthHandler.CheckHealth)

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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

	// Registration endpoints
	r.POST("/auth/users/register", userHandler.RegisterUser)
	r.POST("/auth/keys/register", userKeyHandler.RegisterUserKey)

	// Authentication endpoints
	r.POST("/auth/challenge", authHandler.CreateChallenge)
	r.POST("/auth/login", authHandler.Login)
	r.POST("/auth/github/login", githubHandler.ValidateToken)

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

	return r
}

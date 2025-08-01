package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// SetupRouter configures the Gin router
func SetupRouter(
	releaseService service.ReleaseService,
	deploymentService service.DeploymentService,
	am *middleware.AuthMiddleware,
	db *gorm.DB,
	logger *slog.Logger,
	ghaHandler *handlers.GHAHandler,
) *gin.Engine {
	r := gin.New()

	// Middleware Setup //

	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))
	r.Use(func(c *gin.Context) {
		c.Set("releaseService", releaseService)
		c.Set("deploymentService", deploymentService)
		c.Next()
	})

	releaseHandler := handlers.NewReleaseHandler(releaseService, logger)
	deploymentHandler := handlers.NewDeploymentHandler(deploymentService, logger)
	healthHandler := handlers.NewHealthHandler(db, logger)

	// Health check endpoint
	r.GET("/healthz", healthHandler.CheckHealth)

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Route Setup //

	// GHA token validation endpoint (no auth required)
	r.POST("/gha/validate", ghaHandler.ValidateToken)

	// GHA authentication management endpoints (requires auth)
	r.POST("/gha/auth", am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthWrite}), ghaHandler.CreateAuth)
	r.GET("/gha/auth", am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthRead}), ghaHandler.ListAuths)
	r.GET("/gha/auth/:id", am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthRead}), ghaHandler.GetAuth)
	r.GET("/gha/auth/repository/:repository", am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthRead}), ghaHandler.GetAuthByRepository)
	r.PUT("/gha/auth/:id", am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthWrite}), ghaHandler.UpdateAuth)
	r.DELETE("/gha/auth/:id", am.ValidatePermissions([]auth.Permission{auth.PermGHAAuthWrite}), ghaHandler.DeleteAuth)

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

	return r
}

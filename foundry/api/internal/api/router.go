package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/auth"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
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

	// Route Setup //

	// GHA token validation endpoint
	r.POST("/auth/gha/validate", ghaHandler.ValidateToken)

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

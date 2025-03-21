package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
	"gorm.io/gorm"
)

// SetupRouter configures the Gin router
func SetupRouter(
	releaseService service.ReleaseService,
	deploymentService service.DeploymentService,
	db *gorm.DB,
	logger *slog.Logger,
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

	// Release endpoints
	r.POST("/release", releaseHandler.CreateRelease)
	r.GET("/release/:id", releaseHandler.GetRelease)
	r.PUT("/release/:id", releaseHandler.UpdateRelease)
	r.GET("/releases", releaseHandler.ListReleases)

	// Release aliases
	r.GET("/release/alias/:name", releaseHandler.GetReleaseByAlias)
	r.POST("/release/alias/:name", releaseHandler.CreateAlias)
	r.DELETE("/release/alias/:name", releaseHandler.DeleteAlias)
	r.GET("/release/:id/aliases", releaseHandler.ListAliases)

	// Deployment endpoints
	r.POST("/release/:id/deploy", deploymentHandler.CreateDeployment)
	r.GET("/release/:id/deploy/:deployId", deploymentHandler.GetDeployment)
	r.PUT("/release/:id/deploy/:deployId/status", deploymentHandler.UpdateDeploymentStatus)
	r.GET("/release/:id/deployments", deploymentHandler.ListDeployments)
	r.GET("/release/:id/deploy/latest", deploymentHandler.GetLatestDeployment)

	return r
}

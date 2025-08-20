package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/api/middleware"
	apiroutes "github.com/input-output-hk/catalyst-forge/services/api/internal/api/routes"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/config"
	emailsvc "github.com/input-output-hk/catalyst-forge/services/api/internal/service/email"
	pca "github.com/input-output-hk/catalyst-forge/services/api/internal/service/pca"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// SetupRouter configures the Gin router.
func SetupRouter(
	db *gorm.DB,
	logger *slog.Logger,
	emailService emailsvc.Service,
	sessionMaxActive int,
	enablePerIPRateLimit bool,
	pcaClient pca.PCAClient,
	authConfig *config.Config, // Add authConfig parameter
) *gin.Engine {
	r := gin.New()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))
	//r.Use(middleware.CORSMiddleware())

	// Handlers
	healthHandler := handlers.NewHealthHandler(db, logger)
	apiroutes.RegisterDomainRoutes(r, apiroutes.DomainDeps{
		DB:     db,
		Logger: logger,
	})
	apiroutes.RegisterPublic(r, healthHandler.CheckHealth)

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}

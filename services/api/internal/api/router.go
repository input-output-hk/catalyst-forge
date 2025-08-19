package api

import (
	"context"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/api/middleware"
	apiroutes "github.com/input-output-hk/catalyst-forge/services/api/internal/api/routes"
	libauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
	certkit "github.com/input-output-hk/catalyst-forge/services/api/internal/certkit/certkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/config"
	emailsvc "github.com/input-output-hk/catalyst-forge/services/api/internal/service/email"
	pca "github.com/input-output-hk/catalyst-forge/services/api/internal/service/pca"

	apiauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
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

	// Setup AuthKit
	setupAuthKit(authConfig, r, db, logger)

	// Middleware
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORSMiddleware())
	// Test auth bypass (no-op unless TEST_AUTH_BYPASS=1)
	r.Use(middleware.TestAuthBypass())

	// Handlers
	healthHandler := handlers.NewHealthHandler(db, logger)
	certificateHandler := handlers.NewCertificateHandler()
	if pcaClient != nil {
		certificateHandler = certificateHandler.WithPCA(pcaClient)
	}

	// Routes
	apiroutes.RegisterCertificates(r, apiroutes.CertificatesDeps{H: certificateHandler})
	apiroutes.RegisterRBACAdmin(r, db, logger)

	// Mount certkit under /pki when CA ARN is configured
	if authConfig != nil && authConfig.Certs.PCAClientCAArn != "" {
		var pcaCli certkit.PCAClient
		if os.Getenv("CERTS_PCA_FAKE") == "1" {
			pcaCli = newPKIFake()
		} else {
			if cli, err := certkit.BuildPCAFromRegion(context.Background(), authConfig.Certs.CARegion); err == nil {
				pcaCli = cli
			}
		}
		if pcaCli != nil {
			if ck, e2 := certkit.New(certkit.Config{
				CAArn:        authConfig.Certs.PCAClientCAArn,
				Region:       authConfig.Certs.CARegion,
				DefaultTTL:   authConfig.Certs.ClientCertTTLDev,
				MaxTTL:       authConfig.Certs.ClientCertTTLCIMax,
				PollInterval: authConfig.Certs.PCATimeout / 10,
				MaxWait:      authConfig.Certs.PCATimeout,
			}, certkit.Deps{PCA: pcaCli}); e2 == nil {
				ck.RegisterRoutes(r.Group("/pki"))
			}
		}
	}
	apiroutes.RegisterDomainRoutes(r, apiroutes.DomainDeps{
		DB:     db,
		Logger: logger,
	})
	apiroutes.RegisterPublic(r, healthHandler.CheckHealth)

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}

// setupAuthKit builds and mounts AuthKit on the router.
func setupAuthKit(authConfig *config.Config, r *gin.Engine, db *gorm.DB, logger *slog.Logger) {
	akCfg := apiauth.BuildConfig(authConfig)
	if logger != nil {
		logger.Info("AuthKit config loaded",
			"bootstrap_token_len", len(akCfg.BootstrapToken),
		)
	}
	// deps := apiauth.BuildDeps(context.Background(), authConfig, nil, db, nil, apiauth.NewLogger(logger))

	reg := libauth.NewPolicyRegistry()
	reg.AllowAnonymous("GET", "/health", "/swagger/*")

	// TODO: Replace with a dedicated JWKS handler
	// Mount JWKS via Manager; API binds its own /api/v1/auth routes
	// if m, err := libauth.New(akCfg, deps); err == nil {
	// 	m.RegisterJWKS(r.Group("/.well-known"))
	// }

	// Wire canonical middleware (auth + policy) and bind API's auth routes
	apiroutes.RegisterAuthKit(r, akCfg, reg)
}

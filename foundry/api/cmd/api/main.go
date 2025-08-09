package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	kongtoml "github.com/alecthomas/kong-toml"
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/cmd/api/auth"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	metrics "github.com/input-output-hk/catalyst-forge/foundry/api/internal/metrics"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
	emailsvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/email"

	// step-ca (type reference only)
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/stepca"
	userservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s/mocks"
	ghauth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/github"

	// gorm imported via helpers

	_ "github.com/input-output-hk/catalyst-forge/foundry/api/docs"
)

var version = "dev"

// @title           Catalyst Foundry API
// @version         1.0
// @description     API for managing releases and deployments in the Catalyst Foundry system.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:5050
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

var mockK8sClient = mocks.ClientMock{
	CreateDeploymentFunc: func(ctx context.Context, deployment *models.ReleaseDeployment) error {
		return nil
	},
}

// CLI represents the command-line interface structure
type CLI struct {
	Run     RunCmd       `kong:"cmd,help='Start the API server'"`
	Version VersionCmd   `kong:"cmd,help='Show version information'"`
	Auth    auth.AuthCmd `kong:"cmd,help='Authentication management commands'"`
	Seed    SeedCmd      `kong:"cmd,help='Seed default data (admin user/role)'"`
	// --config=/path/to/config.toml support (TOML via kong-toml loader)
	Config kong.ConfigFlag `kong:"help='Load configuration from a TOML file',name='config'"`
}

// RunCmd represents the run subcommand
type RunCmd struct {
	config.Config `kong:"embed"`
}

// VersionCmd represents the version subcommand
type VersionCmd struct{}

// Run executes the version subcommand
func (v *VersionCmd) Run() error {
	fmt.Printf("foundry api version %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
	return nil
}

// Run executes the run subcommand
func (r *RunCmd) Run() error {
	// Validate configuration
	if err := r.Validate(); err != nil {
		return err
	}

	// Initialize logger
	logger, err := r.GetLogger()
	if err != nil {
		return err
	}

	// Connect to the database
	db, err := openDB(r.Config)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		return err
	}

	// Run migrations
	logger.Info("Running database migrations")
	err = runMigrations(db)
	if err != nil {
		logger.Error("Failed to run migrations", "error", err)
		return err
	}

	// Context reserved for future init steps (kept to match structure)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = ctx
	cancel()

	// Initialize Kubernetes client if enabled
	var k8sClient k8s.Client
	if r.Kubernetes.Enabled {
		logger.Info("Initializing Kubernetes client", "namespace", r.Kubernetes.Namespace)
		k8sClient, err = initK8sClient(r.Kubernetes, logger)
		if err != nil {
			logger.Error("Failed to initialize Kubernetes client", "error", err)
			return err
		}
	} else {
		k8sClient = &mockK8sClient
		logger.Info("Kubernetes integration is disabled")
	}

	// Initialize repositories
	releaseRepo := repository.NewReleaseRepository(db)
	deploymentRepo := repository.NewDeploymentRepository(db)
	counterRepo := repository.NewIDCounterRepository(db)
	aliasRepo := repository.NewAliasRepository(db)
	eventRepo := repository.NewEventRepository(db)
	ghaAuthRepo := repository.NewGithubAuthRepository(db)

	// Initialize user repositories
	userRepo := userrepo.NewUserRepository(db)
	roleRepo := userrepo.NewRoleRepository(db)
	userRoleRepo := userrepo.NewUserRoleRepository(db)
	userKeyRepo := userrepo.NewUserKeyRepository(db)

	// Initialize services
	releaseService := service.NewReleaseService(releaseRepo, aliasRepo, counterRepo, deploymentRepo)
	deploymentService := service.NewDeploymentService(deploymentRepo, releaseRepo, eventRepo, k8sClient, db, logger)
	ghaAuthService := service.NewGithubAuthService(ghaAuthRepo, logger)

	// Initialize user services
	userService := userservice.NewUserService(userRepo, logger)
	roleService := userservice.NewRoleService(roleRepo, logger)
	userRoleService := userservice.NewUserRoleService(userRoleRepo, logger)
	userKeyService := userservice.NewUserKeyService(userKeyRepo, logger)

	// Initialize middleware
	jwtManagerImpl, err := initJWTManager(r.Auth, logger)
	if err != nil {
		logger.Error("Failed to initialize JWT manager", "error", err)
		return err
	}
	jwtManager := jwtManagerImpl
	revokedRepo := userrepo.NewRevokedJTIRepository(db)
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, logger, userService, revokedRepo)

	// Step-CA removed with PCA migration; keep nil client for root endpoint fallback
	var stepCAClient *stepca.Client = nil

	// Initialize GitHub Actions OIDC client
	ghaOIDCClient, err := ghauth.NewDefaultGithubActionsOIDCClient(context.Background(), "/tmp/gha-jwks-cache")
	if err != nil {
		logger.Error("Failed to initialize GHA OIDC client", "error", err)
		return err
	}

	// Start the GHA OIDC cache
	if err := ghaOIDCClient.StartCache(); err != nil {
		logger.Error("Failed to start GHA OIDC cache", "error", err)
		return err
	}
	defer ghaOIDCClient.StopCache()

	// Setup router
	// Optionally construct SES email service
	var emailService emailsvc.Service
	emailService, _ = initEmailService(r.Email, r.Server.PublicBaseURL)
	// Initialize Prometheus metrics
	metrics.InitDefault()

	clientsCA2, serversCA2 := buildProvisionerClients(r.Config)
	// Initialize PCA if configured
	pcaCli, _ := initPCAClient(r.Certs)
	router := api.SetupRouter(
		releaseService,
		deploymentService,
		userService,
		roleService,
		userRoleService,
		userKeyService,
		authMiddleware,
		db,
		logger,
		jwtManager,
		ghaOIDCClient,
		ghaAuthService,
		stepCAClient,
		emailService,
		r.Certs.SessionMaxActive,
		r.Security.EnableNaivePerIPRateLimit,
		clientsCA2,
		serversCA2,
		pcaCli,
	)
	// Inject defaults into request context (policy, email, github, etc.)
	injectDefaultContext(router, r.Config, emailService, clientsCA2, serversCA2)
	// Attach PCA client to certificate handler if available
	if pcaCli != nil {
		// Router constructed the handler; re-create and replace with PCA attached requires refactor.
		// Simpler: set PCA config in context and handlers already read it; PCA client stored globally here.
		// For now, set a global in gin context via middleware
		router.Use(func(c *gin.Context) { c.Set("pca_client_present", true); c.Next() })
	}
	// Expose cert TTL clamps
	router.Use(func(c *gin.Context) {
		c.Set("certs_client_cert_ttl_dev", r.Certs.ClientCertTTLDev)
		c.Set("certs_client_cert_ttl_ci_max", r.Certs.ClientCertTTLCIMax)
		c.Set("certs_server_cert_ttl", r.Certs.ServerCertTTL)
		c.Next()
	})
	// Ensure Step-CA provisioner clients are present on every request context
	router.Use(func(c *gin.Context) {
		c.Set("stepca_clients_ca", clientsCA2)
		c.Set("stepca_servers_ca", serversCA2)
		c.Next()
	})

	// Initialize server
	server := api.NewServer(r.GetServerAddr(), router, logger)

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Error("Failed to start server", "error", err)
			quit <- syscall.SIGTERM
		}
	}()

	logger.Info("API server started", "addr", r.GetServerAddr())

	// Wait for shutdown signal
	<-quit
	logger.Info("Shutting down server...")

	// Create a deadline for graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exiting")
	return nil
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("foundry-api"),
		kong.Description("Catalyst Foundry API Server"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		// Load configuration from TOML files if present; CLI flags override
		kong.Configuration(kongtoml.Loader,
			"/etc/foundry/foundry-api.toml",
			"/etc/foundry-api.toml",
			"~/.config/foundry/api.toml",
			"./config.toml",
		),
	)

	// Execute the selected subcommand
	err := ctx.Run()
	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

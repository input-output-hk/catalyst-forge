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
	"github.com/input-output-hk/catalyst-forge/foundry/api/cmd/api/auth"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
	am "github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
	ghauth "github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth/github"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s/mocks"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var version = "dev"

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
	db, err := gorm.Open(postgres.Open(r.GetDSN()), &gorm.Config{})
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		return err
	}

	// Run migrations
	logger.Info("Running database migrations")
	err = db.AutoMigrate(
		&models.Release{},
		&models.ReleaseDeployment{},
		&models.IDCounter{},
		&models.ReleaseAlias{},
		&models.DeploymentEvent{},
		&models.GHARepositoryAuth{},
	)
	if err != nil {
		logger.Error("Failed to run migrations", "error", err)
		return err
	}

	// Initialize Kubernetes client if enabled
	var k8sClient k8s.Client
	if r.Kubernetes.Enabled {
		logger.Info("Initializing Kubernetes client", "namespace", r.Kubernetes.Namespace)
		k8sClient, err = k8s.New(r.Kubernetes.Namespace, logger)
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
	ghaAuthRepo := repository.NewGHAAuthRepository(db)

	// Initialize services
	releaseService := service.NewReleaseService(releaseRepo, aliasRepo, counterRepo, deploymentRepo)
	deploymentService := service.NewDeploymentService(deploymentRepo, releaseRepo, eventRepo, k8sClient, db, logger)
	ghaAuthService := service.NewGHAAuthService(ghaAuthRepo, logger)

	// Initialize middleware
	authManager, err := am.NewAuthManager(r.Auth.PrivateKey, r.Auth.PublicKey, am.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to initialize auth manager", "error", err)
		return err
	}
	authMiddleware := middleware.NewAuthMiddleware(authManager, logger)

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

	// Initialize GHA handler
	ghaHandler := handlers.NewGHAHandler(authManager, ghaOIDCClient, ghaAuthService, logger)

	// Setup router
	router := api.SetupRouter(releaseService, deploymentService, authMiddleware, db, logger, ghaHandler)

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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
	)

	// Execute the selected subcommand
	err := ctx.Run()
	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

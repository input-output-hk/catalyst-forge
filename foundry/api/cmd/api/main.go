package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s/mocks"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var mockK8sClient = mocks.ClientMock{
	CreateDeploymentFunc: func(ctx context.Context, deployment *models.ReleaseDeployment) error {
		return nil
	},
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger, err := cfg.GetLogger()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Connect to the database
	db, err := gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{})
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Run migrations
	logger.Info("Running database migrations")
	err = db.AutoMigrate(
		&models.Release{},
		&models.ReleaseDeployment{},
		&models.IDCounter{},
		&models.ReleaseAlias{},
		&models.DeploymentEvent{},
	)
	if err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize Kubernetes client if enabled
	var k8sClient k8s.Client
	if cfg.Kubernetes.Enabled {
		logger.Info("Initializing Kubernetes client", "namespace", cfg.Kubernetes.Namespace)
		k8sClient, err = k8s.New(cfg.Kubernetes.Namespace, logger)
		if err != nil {
			logger.Error("Failed to initialize Kubernetes client", "error", err)
			os.Exit(1)
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

	// Initialize services
	releaseService := service.NewReleaseService(releaseRepo, aliasRepo, counterRepo, deploymentRepo)
	deploymentService := service.NewDeploymentService(deploymentRepo, releaseRepo, eventRepo, k8sClient, db, logger)

	// Setup router
	router := api.SetupRouter(releaseService, deploymentService, db, logger)

	// Initialize server
	server := api.NewServer(cfg.GetServerAddr(), router, logger)

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

	logger.Info("API server started", "addr", cfg.GetServerAddr())

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
}

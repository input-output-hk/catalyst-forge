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

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	metrics "github.com/input-output-hk/catalyst-forge/foundry/api/internal/metrics"
	emailsvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/email"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	_ "github.com/input-output-hk/catalyst-forge/foundry/api/docs"
)

var (
	version = "dev"
	cfgFile string
	cfg     config.Config
)

// var mockK8sClient = mocks.ClientMock{
// 	CreateDeploymentFunc: func(ctx context.Context, deployment *models.ReleaseDeployment) error {
// 		return nil
// 	},
// }

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "foundry-api",
	Short: "Catalyst Foundry API Server",
	Long:  `API for managing releases and deployments in the Catalyst Foundry system.`,
}

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the API server",
	Long:  `Start the Catalyst Foundry API server with the configured settings.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Ensure Viper is initialized before any flag default that consults Viper
		initConfig()
		// Only override from flag if explicitly provided
		if f := cmd.Flags().Lookup("bootstrap-token"); f != nil && f.Changed {
			if val, err := cmd.Flags().GetString("bootstrap-token"); err == nil {
				viper.Set("auth.bootstraptoken", val)
			}
		}
		return nil
	},
	RunE: runServer,
}

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("foundry api version %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/foundry/foundry-api.toml)")

	// Add subcommands
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)

	// Define flags via helper
	addRunFlags(runCmd)

	// Bind flags to viper
	bindRunFlags()
}

// bindRunFlags is defined in flags.go

func initConfig() { initViper(cfgFile) }

func loadConfig() error { return loadConfigFromViper() }

func runServer(cmd *cobra.Command, args []string) error {
	// Load configuration
	if err := loadConfig(); err != nil {
		return err
	}

	// Initialize logger
	logger, err := cfg.GetLogger()
	if err != nil {
		return err
	}

	// Connect to the database
	db, err := openDB(cfg, logger)
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

	// Initialize RBAC (automigrate + seed defaults if enabled)
	initRBAC(context.Background(), db, cfg, logger)

	// Context reserved for future init steps
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = ctx
	cancel()

	// Initialize Kubernetes client if enabled
	// var k8sClient k8s.Client
	// if cfg.Kubernetes.Enabled {
	// 	logger.Info("Initializing Kubernetes client", "namespace", cfg.Kubernetes.Namespace)
	// 	k8sClient, err = initK8sClient(cfg.Kubernetes, logger)
	// 	if err != nil {
	// 		logger.Error("Failed to initialize Kubernetes client", "error", err)
	// 		return err
	// 	}
	// } else {
	// 	k8sClient = &mockK8sClient
	// 	logger.Info("Kubernetes integration is disabled")
	// }

	// Setup router
	// Optionally construct SES email service
	var emailService emailsvc.Service
	emailService, _ = initEmailService(cfg.Email, cfg.Server.PublicBaseURL)

	// Initialize Prometheus metrics
	metrics.InitDefault()

	// Initialize PCA if configured
	router := api.SetupRouter(
		db,
		logger,
		emailService,
		cfg.Certs.SessionMaxActive,
		cfg.Security.EnableNaivePerIPRateLimit,
		nil,
		&cfg,
	)

	// Inject defaults into request context
	injectDefaultContext(router, cfg, emailService)

	// Expose cert TTL clamps
	router.Use(func(c *gin.Context) {
		c.Set("certs_client_cert_ttl_dev", cfg.Certs.ClientCertTTLDev)
		c.Set("certs_client_cert_ttl_ci_max", cfg.Certs.ClientCertTTLCIMax)
		c.Set("certs_server_cert_ttl", cfg.Certs.ServerCertTTL)
		c.Next()
	})

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
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

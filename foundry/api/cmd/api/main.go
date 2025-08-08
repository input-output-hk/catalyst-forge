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
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/cmd/api/auth"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	adm "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/audit"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
	emailsvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/email"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/stepca"
	userservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s/mocks"
	ghauth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/github"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/input-output-hk/catalyst-forge/foundry/api/docs"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
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
		&models.GithubRepositoryAuth{},
		&user.User{},
		&user.Role{},
		&user.UserRole{},
		&user.UserKey{},
		&user.Device{},
		&user.RefreshToken{},
		&user.DeviceSession{},
		&user.RevokedJTI{},
		&user.Invite{},
		&adm.Log{},
	)
	if err != nil {
		logger.Error("Failed to run migrations", "error", err)
		return err
	}

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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
	jwtManagerImpl, err := jwt.NewES256Manager(
		r.Auth.PrivateKey,
		r.Auth.PublicKey,
		jwt.WithManagerLogger(logger),
		jwt.WithMaxAuthTokenTTL(r.Auth.AccessTTL),
	)
	if err != nil {
		logger.Error("Failed to initialize JWT manager", "error", err)
		return err
	}
	var jwtManager jwt.JWTManager = jwtManagerImpl
	revokedRepo := userrepo.NewRevokedJTIRepository(db)
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, logger, userService, revokedRepo)

	// Initialize step-ca client (for certificate signing)
	var rootCA []byte
	if r.StepCA.RootCA != "" {
		rootCA, err = os.ReadFile(r.StepCA.RootCA)
		if err != nil {
			logger.Error("Failed to read step-ca root certificate", "error", err, "path", r.StepCA.RootCA)
			return err
		}
	}

	fmt.Printf("Root CA: %s\n", rootCA)
	stepCAClient, err := stepca.NewClient(stepca.Config{
		BaseURL:            r.StepCA.BaseURL,
		InsecureSkipVerify: r.StepCA.InsecureSkipVerify,
		Timeout:            r.StepCA.ClientTimeout,
		RootCA:             rootCA,
	})
	if err != nil {
		logger.Error("Failed to initialize step-ca client", "error", err)
		return err
	}

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
	if r.Email.Enabled && r.Email.Provider == "ses" {
		ses, err := emailsvc.NewSES(context.Background(), emailsvc.SESOptions{
			Region:  r.Email.SESRegion,
			Sender:  r.Email.Sender,
			BaseURL: r.Server.PublicBaseURL,
		})
		if err != nil {
			logger.Error("Failed to initialize SES email service", "error", err)
		} else {
			emailService = ses
		}
	}

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
	)
	// Inject default invite TTL and optional email service into router context
	router.Use(func(c *gin.Context) {
		c.Set("invite_default_ttl", r.Auth.InviteTTL)
		if r.Email.Enabled && r.Email.Provider == "ses" {
			c.Set("email_provider", "ses")
			c.Set("email_sender", r.Email.Sender)
			c.Set("public_base_url", r.Server.PublicBaseURL)
			c.Set("email_region", r.Email.SESRegion)
		}
		c.Set("enable_per_ip_ratelimit", r.Security.EnableNaivePerIPRateLimit)
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
	)

	// Execute the selected subcommand
	err := ctx.Run()
	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

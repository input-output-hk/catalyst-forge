package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"time"

	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	adm "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/audit"
	buildmodels "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/build"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	emailsvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/email"
	pcaclient "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/pca"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openDB(cfg config.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{})
}

func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
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
		// Build identity models
		&buildmodels.ServiceAccount{},
		&buildmodels.ServiceAccountKey{},
		&buildmodels.BuildSession{},
	)
}

func initK8sClient(cfg config.KubernetesConfig, logger *slog.Logger) (k8s.Client, error) {
	if cfg.Enabled {
		return k8s.New(cfg.Namespace, logger)
	}
	return nil, nil
}

func initJWTManager(authCfg config.AuthConfig, logger *slog.Logger) (jwt.JWTManager, error) {
	manager, err := jwt.NewES256Manager(
		authCfg.PrivateKey,
		authCfg.PublicKey,
		jwt.WithManagerLogger(logger),
		jwt.WithMaxAuthTokenTTL(authCfg.AccessTTL),
	)
	if err != nil {
		return nil, err
	}
	return manager, nil
}

// initGHAClient reserved for future extraction if needed
//
//lint:ignore U1000 kept intentionally to preserve API surface
func initGHAClient() (Start func() error, Stop func(), clientCtx context.Context, err error) {
	// Kept in main for logging; this wrapper reserved for future extraction if needed.
	return nil, nil, nil, nil
}

func initEmailService(cfg config.EmailConfig, publicBaseURL string) (emailsvc.Service, error) {
	if cfg.Enabled && cfg.Provider == "ses" {
		return emailsvc.NewSES(context.Background(), emailsvc.SESOptions{
			Region:  cfg.SESRegion,
			Sender:  cfg.Sender,
			BaseURL: publicBaseURL,
		})
	}
	return nil, nil
}

// parseProvisionerSigner retained for legacy dev paths
//
//lint:ignore U1000 unused after PCA migration
func parseProvisionerSigner(path string) *ecdsa.PrivateKey {
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil
	}
	if pk, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if ec, ok := pk.(*ecdsa.PrivateKey); ok {
			return ec
		}
	}
	if ec, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return ec
	}
	return nil
}

func injectDefaultContext(r *gin.Engine, cfg config.Config, emailSvc emailsvc.Service) {
	r.Use(func(c *gin.Context) {
		c.Set("invite_default_ttl", cfg.Auth.InviteTTL)
		if emailSvc != nil && cfg.Email.Enabled && cfg.Email.Provider == "ses" {
			c.Set("email_provider", "ses")
			c.Set("email_sender", cfg.Email.Sender)
			c.Set("public_base_url", cfg.Server.PublicBaseURL)
			c.Set("email_region", cfg.Email.SESRegion)
		}
		c.Set("enable_per_ip_ratelimit", cfg.Security.EnableNaivePerIPRateLimit)
		// GitHub OIDC policy
		c.Set("github_expected_iss", cfg.Certs.GhOIDCIssuer)
		c.Set("github_expected_aud", cfg.Certs.GhOIDCAudience)
		c.Set("github_allowed_orgs", cfg.Certs.GhAllowedOrgs)
		c.Set("github_allowed_repos", cfg.Certs.GhAllowedRepos)
		c.Set("github_protected_refs", cfg.Certs.GhProtectedRefs)
		c.Set("github_job_token_default_ttl", cfg.Certs.JobTokenDefaultTTL)
		// PCA configuration keys for handlers
		clientArn := cfg.Certs.PCAClientCAArn
		serverArn := cfg.Certs.PCAServerCAArn
		if clientArn == "" {
			clientArn = "arn:mock:client"
		}
		if serverArn == "" {
			serverArn = "arn:mock:server"
		}
		c.Set("certs_pca_client_ca_arn", clientArn)
		c.Set("certs_pca_server_ca_arn", serverArn)
		c.Set("certs_pca_client_template_arn", cfg.Certs.PCAClientTemplateArn)
		c.Set("certs_pca_server_template_arn", cfg.Certs.PCAServerTemplateArn)
		c.Set("certs_pca_signing_algo_client", cfg.Certs.PCASigningAlgoClient)
		c.Set("certs_pca_signing_algo_server", cfg.Certs.PCASigningAlgoServer)
		// Feature flags
		c.Set("feature_ext_authz_enabled", cfg.Certs.ExtAuthzEnabled)
		c.Next()
	})
}

// initPCAClient optionally initializes an ACM-PCA client wrapper when ARNs are provided
func initPCAClient(cfg config.CertsConfig) (pcaclient.PCAClient, error) {
	if cfg.PCAClientCAArn == "" && cfg.PCAServerCAArn == "" {
		// Dev/local: return a mock PCA so cert flows work in integration tests without AWS
		return &pcaclient.Mock{}, nil
	}
	return pcaclient.NewAWS(pcaclient.Options{Timeout: cfg.PCATimeout})
}

// Utility: short timeout context
// newTimeoutCtx helper (currently unused)
//
//lint:ignore U1000 reserved for future use
func newTimeoutCtx(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

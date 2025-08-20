package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/config"
	argomodels "github.com/input-output-hk/catalyst-forge/services/api/internal/models/argo"
	artifactmodels "github.com/input-output-hk/catalyst-forge/services/api/internal/models/artifact"
	adm "github.com/input-output-hk/catalyst-forge/services/api/internal/models/audit"
	buildmodels "github.com/input-output-hk/catalyst-forge/services/api/internal/models/build"
	deploymentmodels "github.com/input-output-hk/catalyst-forge/services/api/internal/models/deployment"
	environmentmodels "github.com/input-output-hk/catalyst-forge/services/api/internal/models/environment"
	gitopsmodels "github.com/input-output-hk/catalyst-forge/services/api/internal/models/gitops"
	projectmodels "github.com/input-output-hk/catalyst-forge/services/api/internal/models/project"
	releasemodels "github.com/input-output-hk/catalyst-forge/services/api/internal/models/release"
	repositorymodels "github.com/input-output-hk/catalyst-forge/services/api/internal/models/repository"
	tracemodels "github.com/input-output-hk/catalyst-forge/services/api/internal/models/trace"
	emailsvc "github.com/input-output-hk/catalyst-forge/services/api/internal/service/email"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openDB(cfg config.Config, logger *slog.Logger) (*gorm.DB, error) {
	// Retry loop so the server waits for the DB to come up instead of crash-looping.
	// In Kubernetes this keeps the container in an unhealthy/non-ready state until DB is reachable.
	timeout := 60 * time.Second
	if v := os.Getenv("DB_CONNECT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		}
	}
	interval := 500 * time.Millisecond
	if v := os.Getenv("DB_CONNECT_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

	dsn := cfg.GetDSN()
	deadline := time.Now().Add(timeout)
	var lastErr error
	if logger != nil {
		logger.Info("Connecting to database",
			"host", cfg.Database.Host,
			"port", cfg.Database.DbPort,
			"name", cfg.Database.Name,
			"timeout", timeout,
		)
	}
	for time.Now().Before(deadline) {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			if sqlDB, err2 := db.DB(); err2 == nil {
				if errPing := sqlDB.Ping(); errPing == nil {
					if logger != nil {
						logger.Info("Connected to database",
							"host", cfg.Database.Host,
							"port", cfg.Database.DbPort,
							"name", cfg.Database.Name,
						)
					}
					return db, nil
				} else {
					lastErr = errPing
				}
			} else {
				lastErr = err2
			}
		} else {
			lastErr = err
		}
		if logger != nil {
			logger.Debug("Database not ready yet; retrying",
				"error", lastErr,
				"retry_in", interval,
			)
		}
		time.Sleep(interval)
	}
	return nil, fmt.Errorf("database not ready within %s: %w", timeout, lastErr)
}

func runMigrations(db *gorm.DB) error {
	// Core API models - All new models from Phase 1-4 implementation
	if err := db.AutoMigrate(
		// Audit models
		&adm.Log{},

		// Repository and Project models
		&repositorymodels.Repository{},
		&projectmodels.Project{},

		// Trace models
		&tracemodels.Trace{},

		// Build models
		&buildmodels.Build{},

		// Artifact models
		&artifactmodels.Artifact{},

		// Release models
		&releasemodels.Release{},
		&releasemodels.ReleaseModule{},
		&releasemodels.ReleaseArtifact{},
		&releasemodels.RenderedRelease{},

		// Environment models
		&environmentmodels.Environment{},

		// Deployment models
		&deploymentmodels.Deployment{},
		&deploymentmodels.Promotion{},

		// GitOps models
		&gitopsmodels.GitOpsChange{},

		// GitOps sync models
		&argomodels.GitOpsSync{},
	); err != nil {
		return err
	}
	// Ensure indexes for promotions
	if db.Migrator().HasTable("promotions") {
		if err := db.Exec(`CREATE INDEX IF NOT EXISTS ix_promotions_proj_env ON promotions(project_id, environment_id, created_at DESC)`).Error; err != nil {
			return err
		}
		if err := db.Exec(`CREATE INDEX IF NOT EXISTS ix_promotions_release ON promotions(release_id)`).Error; err != nil {
			return err
		}
		if err := db.Exec(`CREATE INDEX IF NOT EXISTS ix_promotions_status ON promotions(status)`).Error; err != nil {
			return err
		}
	}

	// Ensure conditional indexes exist for nullable digest columns
	if db.Migrator().HasTable("release") {
		if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS ux_release_oci_digest ON "release" (oci_digest) WHERE oci_digest IS NOT NULL`).Error; err != nil {
			return err
		}
	}

	return nil
}

// func initK8sClient(cfg config.KubernetesConfig, logger *slog.Logger) (k8s.Client, error) {
// 	if cfg.Enabled {
// 		return k8s.New(cfg.Namespace, logger)
// 	}
// 	return nil, nil
// }

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
		c.Next()
	})
}

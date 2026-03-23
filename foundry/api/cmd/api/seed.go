package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
)

// SeedCmd seeds default data (admin user and optional admin role)
type SeedCmd struct {
	// Email for the admin account
	Email string `kong:"help='Admin email to seed',default='admin@foundry.dev'"`
	// Also create an admin role and assign
	WithRole bool `kong:"help='Create admin role with all permissions and assign to user',default=true"`
}

func (s *SeedCmd) Run() error {
	cfg := configFromEnv()
	db, err := openDB(cfg)
	if err != nil {
		return err
	}

	// Ensure schema exists
	if err := runMigrations(db); err != nil {
		return fmt.Errorf("seed: run migrations: %w", err)
	}

	ur := userrepo.NewUserRepository(db)
	rr := userrepo.NewRoleRepository(db)
	urr := userrepo.NewUserRoleRepository(db)

	u, _ := ur.GetByEmail(s.Email)
	if u == nil {
		now := &time.Time{}
		*now = time.Now()
		u = &user.User{Email: s.Email, Status: user.UserStatusActive, EmailVerifiedAt: now, UserVer: 1}
		if err := ur.Create(u); err != nil {
			return fmt.Errorf("seed: create user: %w", err)
		}
	}

	if s.WithRole {
		// Create or get admin role with all permissions
		r, _ := rr.GetByName("admin")
		if r == nil {
			r = &user.Role{Name: "admin"}
			r.SetPermissions(auth.AllPermissions)
			if err := rr.Create(r); err != nil {
				return fmt.Errorf("seed: create role: %w", err)
			}
		}
		// Assign if not already assigned
		// Simple insert; duplicates will error silently if unique constraint exists; otherwise duplicates are acceptable in dev
		_ = urr.Create(&user.UserRole{UserID: u.ID, RoleID: r.ID})
	}

	fmt.Printf("Seed complete: user=%s (id=%d)\n", u.Email, u.ID)
	return nil
}

// configFromEnv builds minimal DB config from envs used by entrypoint/compose
func configFromEnv() config.Config {
	var cfg config.Config
	// Server unused here
	// Database config via env
	cfg.Database.Host = getenv("DB_HOST", "postgres")
	cfg.Database.DbPort = mustAtoi(getenv("DB_PORT", "5432"))
	cfg.Database.User = getenv("DB_USER", "foundry")
	cfg.Database.Password = getenv("DB_PASSWORD", "changeme")
	cfg.Database.Name = getenv("DB_NAME", "foundry")
	cfg.Database.SSLMode = getenv("DB_SSLMODE", "disable")
	return cfg
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

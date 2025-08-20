//go:build integration

package testutil

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type APIServer struct {
	Cmd     *exec.Cmd
	BaseURL string
}

// randReader wraps crypto/rand.Read; separate for vet/lint clarity.
type randReader struct{}

func (randReader) Read(p []byte) (int, error) { return cryptoRandRead(p) }

// indirection to avoid importing crypto/rand at the top-level export section
var cryptoRandRead = func(p []byte) (int, error) { return rand.Read(p) }

// buildServerEnv creates the environment variables for the API server.
func buildServerEnv(cfg *Config, pg *PG, ctx context.Context) ([]string, error) {
	host, port, user, pass, db, ssl, err := pg.DSN(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DSN: %w", err)
	}

	env := os.Environ()
	add := func(k, v string) { env = append(env, fmt.Sprintf("%s=%s", k, v)) }

	// Server config
	add("SERVER_HTTPPORT", fmt.Sprintf("%d", cfg.HTTPPort))
	add("SERVER_PUBLICBASEURL", cfg.PublicBaseURL)
	add("SERVER_COOKIESAMESITE", "Strict")

	// Database config (Viper keys)
	add("DATABASE_HOST", host)
	add("DATABASE_DBPORT", port)
	add("DATABASE_USER", user)
	add("DATABASE_PASSWORD", pass)
	add("DATABASE_NAME", db)
	add("DATABASE_SSLMODE", ssl)

	// Logging
	if os.Getenv("TEST_LOG") == "1" {
		add("LOGGING_LEVEL", "debug")
		add("LOGGING_FORMAT", "text")
	} else {
		add("LOGGING_LEVEL", "error")
		add("LOGGING_FORMAT", "json")
	}

	return env, nil
}

// createServerCommand builds the exec.Cmd for starting the API server.
// Passes explicit flags to avoid relying on config files or fragile env decoding.
func createServerCommand(ctx context.Context, env []string, cfg *Config, host, port, user, pass, dbName, ssl string) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	bin := os.Getenv("FOUNDRY_API_BIN")
	if bin == "" {
		if root, e := findRepoRoot(); e == nil {
			candidate := filepath.Join(root, "bin", "foundry-api")
			if _, statErr := os.Stat(candidate); statErr == nil {
				bin = candidate
			}
		}
	}

	// Build common args we want to always pass explicitly to the server
	args := []string{
		"run",
		"--http-port", fmt.Sprintf("%d", cfg.HTTPPort),
		"--public-base-url", cfg.PublicBaseURL,
		"--db-host", host,
		"--db-port", port,
		"--db-user", user,
		"--db-password", pass,
		"--db-name", dbName,
		"--db-sslmode", ssl,
	}

	// Prefer human logs when TEST_LOG=1
	if os.Getenv("TEST_LOG") == "1" {
		args = append(args, "--log-level", "debug", "--log-format", "text")
	} else {
		args = append(args, "--log-level", "error", "--log-format", "json")
	}

	if bin != "" {
		cmd = exec.CommandContext(ctx, bin, args...)
	} else {
		// Fallback to `go run` if binary not available
		cmd = exec.CommandContext(ctx, "go", append([]string{"run", "./cmd/api"}, args...)...)
		if root, e := findRepoRoot(); e == nil {
			cmd.Dir = root
		}
	}

	cmd.Env = env
	// Inherit stdio when verbose; otherwise discard
	if os.Getenv("TEST_LOG") == "1" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd, nil
}

// StartAPIServer launches the API via `go run ./cmd/api run` with test env.
func StartAPIServer(ctx context.Context, cfg *Config, pg *PG) (*APIServer, error) {
	// Ensure Postgres is ready before starting API (helps after snapshot restore)
	host, port, user, pass, db, ssl, err := pg.DSN(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DSN: %w", err)
	}
	if err := waitForPostgres(host, port, user, pass, db, ssl, 60*time.Second); err != nil {
		return nil, fmt.Errorf("postgres not ready: %w", err)
	}

	// Build server environment
	env, err := buildServerEnv(cfg, pg, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build environment: %w", err)
	}

	// Create and configure command with explicit flags
	cmd, err := createServerCommand(ctx, env, cfg, host, port, user, pass, db, ssl)
	if err != nil {
		return nil, fmt.Errorf("failed to create command: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	srv := &APIServer{Cmd: cmd, BaseURL: cfg.PublicBaseURL}
	// Wait for healthz (up to 60s to account for DB migration + warmup)
	if err := waitForHealthy(cfg.PublicBaseURL, 60*time.Second); err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("server failed to become healthy: %w", err)
	}
	return srv, nil
}

func (s *APIServer) Stop() {
	if s == nil || s.Cmd == nil || s.Cmd.Process == nil {
		return
	}
	_ = s.Cmd.Process.Kill()
	// Best-effort wait
	_ = s.Cmd.Wait()
}

func waitForHealthy(base string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := strings.TrimRight(base, "/") + "/healthz"
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == 200 {
			_ = resp.Body.Close()
			return nil
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("server not healthy at %s within %s", url, timeout)
}

// waitForPostgres attempts to connect and ping the DB until ready or timeout.
func waitForPostgres(host, port, user, pass, db, ssl string, timeout time.Duration) error {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", host, port, user, pass, db, ssl)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := sql.Open("postgres", dsn)
		if err == nil {
			conn.SetConnMaxIdleTime(5 * time.Second)
			if err = conn.Ping(); err == nil {
				_ = conn.Close()
				return nil
			}
			_ = conn.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("postgres not reachable at %s:%s within %s", host, port, timeout)
}

// findRepoRoot walks up from CWD to locate the repository root (directory containing go.mod).
func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for i := 0; i < 6; i++ { // limit to a few levels
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("repo root not found")
}

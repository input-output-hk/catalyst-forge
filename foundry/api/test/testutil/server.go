//go:build integration

package testutil

import (
    "context"
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/x509"
    "database/sql"
    "encoding/pem"
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

// KeyPaths holds paths to generated JWT signing keys.
type KeyPaths struct {
    PrivatePath string
    PublicPath  string
    TempDir     string
}

// GenerateJWTKeys creates ephemeral ECDSA keys for JWT signing and returns their paths.
func GenerateJWTKeys() (*KeyPaths, error) {
    tmp, err := os.MkdirTemp("", "api-keys-*")
    if err != nil {
        return nil, fmt.Errorf("failed to create temp dir: %w", err)
    }
    
    // Generate ECDSA P-256 keys using Go stdlib
    key, err := ecdsa.GenerateKey(elliptic.P256(), randReader{})
    if err != nil {
        return nil, fmt.Errorf("failed to generate key: %w", err)
    }
    
    // Private key (EC PRIVATE KEY for compatibility with ES256 manager)
    privDER, err := x509.MarshalECPrivateKey(key)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal private key: %w", err)
    }
    privPath := filepath.Join(tmp, "private.pem")
    if err := os.WriteFile(privPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER}), 0600); err != nil {
        return nil, fmt.Errorf("failed to write private key: %w", err)
    }
    
    // Public key (PKIX)
    pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal public key: %w", err)
    }
    pubPath := filepath.Join(tmp, "public.pem")
    if err := os.WriteFile(pubPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}), 0644); err != nil {
        return nil, fmt.Errorf("failed to write public key: %w", err)
    }
    
    return &KeyPaths{
        PrivatePath: privPath,
        PublicPath:  pubPath,
        TempDir:     tmp,
    }, nil
}

// randReader wraps crypto/rand.Read; separate for vet/lint clarity.
type randReader struct{}
func (randReader) Read(p []byte) (int, error) { return cryptoRandRead(p) }

// indirection to avoid importing crypto/rand at the top-level export section
var cryptoRandRead = func(p []byte) (int, error) { return rand.Read(p) }

// buildServerEnv creates the environment variables for the API server.
func buildServerEnv(cfg *Config, keys *KeyPaths, pg *PG, ctx context.Context) ([]string, error) {
    host, port, user, pass, db, ssl, err := pg.DSN(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get DSN: %w", err)
    }
    
    env := os.Environ()
    add := func(k, v string) { env = append(env, fmt.Sprintf("%s=%s", k, v)) }
    
    add("HTTP_PORT", fmt.Sprintf("%d", cfg.HTTPPort))
    add("PUBLIC_BASE_URL", cfg.PublicBaseURL)
    add("AUTH_PRIVATE_KEY", keys.PrivatePath)
    add("AUTH_PUBLIC_KEY", keys.PublicPath)
    add("REFRESH_HASH_SECRET", cfg.RefreshHashSecret)
    add("AUTH_RATE_LIMIT_BURST", fmt.Sprintf("%d", cfg.RateLimitBurst))
    add("AUTH_RATE_LIMIT_WINDOW", fmt.Sprintf("%ds", cfg.RateLimitWindowSec))
    // Disable rate limiter for test stability; explicit rate limit tests still validate behavior
    add("AUTH_RATELIMIT_DISABLE", "1")
    // Disable secure cookies for HTTP test environment
    add("AUTH_REFRESH_COOKIE_SECURE", "false")
    add("BOOTSTRAP_TOKEN", cfg.BootstrapToken)
    add("USE_TC", "1")
    add("DB_HOST", host)
    add("DB_PORT", port)
    add("DB_USER", user)
    add("DB_PASSWORD", pass)
    add("DB_NAME", db)
    add("DB_SSLMODE", ssl)
    
    // Quiet logs unless TEST_LOG=1
    if os.Getenv("TEST_LOG") == "1" {
        add("LOG_LEVEL", "debug")
        add("LOG_FORMAT", "text")
    } else {
        add("LOG_LEVEL", "error")
        add("LOG_FORMAT", "json")
    }
    
    return env, nil
}

// createServerCommand builds the exec.Cmd for starting the API server.
func createServerCommand(ctx context.Context, env []string) (*exec.Cmd, error) {
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
    
    if bin != "" {
        cmd = exec.CommandContext(ctx, bin, "run")
    } else {
        // Fallback to `go run` if binary not available
        cmd = exec.CommandContext(ctx, "go", "run", "./cmd/api", "run")
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

    // Generate JWT keys
    keys, err := GenerateJWTKeys()
    if err != nil {
        return nil, fmt.Errorf("failed to generate keys: %w", err)
    }
    
    if os.Getenv("TEST_LOG") == "1" {
        if b, e := os.ReadFile(keys.PrivatePath); e == nil {
            if p, _ := pem.Decode(b); p != nil {
                fmt.Println("[testutil] private key PEM type:", p.Type)
            }
        }
        fmt.Println("[testutil] private:", keys.PrivatePath, " public:", keys.PublicPath)
    }

    // Build server environment
    env, err := buildServerEnv(cfg, keys, pg, ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to build environment: %w", err)
    }

    // Create and configure command
    cmd, err := createServerCommand(ctx, env)
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
    if s == nil || s.Cmd == nil || s.Cmd.Process == nil { return }
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
        if resp != nil { _ = resp.Body.Close() }
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
    if err != nil { return "", err }
    dir := wd
    for i := 0; i < 6; i++ { // limit to a few levels
        if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
            return dir, nil
        }
        parent := filepath.Dir(dir)
        if parent == dir { break }
        dir = parent
    }
    return "", errors.New("repo root not found")
}

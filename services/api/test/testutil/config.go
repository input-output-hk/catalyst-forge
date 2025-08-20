//go:build integration

package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
)

// Config holds minimal knobs for starting the API under test.
type Config struct {
	HTTPPort      int
	PublicBaseURL string
}

// RandomPort allocates a free TCP port on localhost and returns it.
func RandomPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	addr := l.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// RandomHex generates a random hex string of the specified byte length.
func RandomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// parseIntEnv parses an integer from environment variable with fallback.
func parseIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var result int
		if n, err := fmt.Sscanf(v, "%d", &result); n == 1 && err == nil {
			return result
		}
	}
	return fallback
}

// getStringEnv gets string from environment with fallback generator.
func getStringEnv(key string, fallback func() string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback()
}

// DefaultTestConfig builds a reasonable default config for tests.
func DefaultTestConfig() (*Config, error) {
	// Allow tests to pin port/base via env; fallback to ephemeral
	httpPort := parseIntEnv("SERVER_HTTPPORT", 0)
	var err error
	if httpPort == 0 {
		httpPort, err = RandomPort()
		if err != nil {
			return nil, fmt.Errorf("failed to allocate port: %w", err)
		}
	}

	base := os.Getenv("SERVER_PUBLICBASEURL")
	if base == "" {
		base = fmt.Sprintf("http://127.0.0.1:%d", httpPort)
	}

	return &Config{
		HTTPPort:      httpPort,
		PublicBaseURL: base,
	}, nil
}

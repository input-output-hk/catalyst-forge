package config_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/tools/github-job-checker/internal/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func resetConfig() {
	// Reset flags and viper before each test
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	viper.Reset()
	os.Clearenv()
	os.Args = os.Args[:1] // Keep only the program name
}

func TestLoadConfig_FromFlags(t *testing.T) {
	resetConfig()

	os.Args = []string{
		"cmd",
		"--owner=test-owner",
		"--repo=test-repo",
		"--ref=test-ref",
		"--token=test-token",
		"--check-interval=15s",
		"--timeout=600s",
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Owner != "test-owner" {
		t.Errorf("expected owner 'test-owner', got '%s'", cfg.Owner)
	}
	if cfg.Repo != "test-repo" {
		t.Errorf("expected repo 'test-repo', got '%s'", cfg.Repo)
	}
	if cfg.Ref != "test-ref" {
		t.Errorf("expected ref 'test-ref', got '%s'", cfg.Ref)
	}
	if cfg.Token != "test-token" {
		t.Errorf("expected token 'test-token', got '%s'", cfg.Token)
	}
	if cfg.CheckInterval != 15*time.Second {
		t.Errorf("expected check-interval 15s, got %v", cfg.CheckInterval)
	}
	if cfg.Timeout != 600*time.Second {
		t.Errorf("expected timeout 600s, got %v", cfg.Timeout)
	}
}

func TestLoadConfig_FromEnv(t *testing.T) {
	resetConfig()

	os.Setenv("GITHUB_OWNER", "env-owner")
	os.Setenv("GITHUB_REPO", "env-repo")
	os.Setenv("GITHUB_REF", "env-ref")
	os.Setenv("GITHUB_TOKEN", "env-token")
	os.Setenv("GITHUB_CHECK_INTERVAL", "20s")
	os.Setenv("GITHUB_TIMEOUT", "500s")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Owner != "env-owner" {
		t.Errorf("expected owner 'env-owner', got '%s'", cfg.Owner)
	}
	if cfg.Repo != "env-repo" {
		t.Errorf("expected repo 'env-repo', got '%s'", cfg.Repo)
	}
	if cfg.Ref != "env-ref" {
		t.Errorf("expected ref 'env-ref', got '%s'", cfg.Ref)
	}
	if cfg.Token != "env-token" {
		t.Errorf("expected token 'env-token', got '%s'", cfg.Token)
	}
	if cfg.CheckInterval != 20*time.Second {
		t.Errorf("expected check-interval 20s, got %v", cfg.CheckInterval)
	}
	if cfg.Timeout != 500*time.Second {
		t.Errorf("expected timeout 500s, got %v", cfg.Timeout)
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	resetConfig()

	// Create a temporary config file
	configContent := `owner: file-owner
repo: file-repo
ref: file-ref
token: file-token
check_interval: 25s
timeout: 400s
`
	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("failed to write to temp config file: %v", err)
	}
	tmpFile.Close()

	viper.SetConfigFile(tmpFile.Name())

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Owner != "file-owner" {
		t.Errorf("expected owner 'file-owner', got '%s'", cfg.Owner)
	}
	if cfg.Repo != "file-repo" {
		t.Errorf("expected repo 'file-repo', got '%s'", cfg.Repo)
	}
	if cfg.Ref != "file-ref" {
		t.Errorf("expected ref 'file-ref', got '%s'", cfg.Ref)
	}
	if cfg.Token != "file-token" {
		t.Errorf("expected token 'file-token', got '%s'", cfg.Token)
	}
	if cfg.CheckInterval != 25*time.Second {
		t.Errorf("expected check-interval 25s, got %v", cfg.CheckInterval)
	}
	if cfg.Timeout != 400*time.Second {
		t.Errorf("expected timeout 400s, got %v", cfg.Timeout)
	}
}

func TestLoadConfig_DefaultsAndRequiredFields(t *testing.T) {
	resetConfig()

	// Set only the required fields
	os.Setenv("GITHUB_OWNER", "default-owner")
	os.Setenv("GITHUB_REPO", "default-repo")
	os.Setenv("GITHUB_REF", "default-ref")
	os.Setenv("GITHUB_TOKEN", "default-token")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.CheckInterval != 10*time.Second {
		t.Errorf("expected default check-interval 10s, got %v", cfg.CheckInterval)
	}
	if cfg.Timeout != 300*time.Second {
		t.Errorf("expected default timeout 300s, got %v", cfg.Timeout)
	}
}

func TestLoadConfig_MissingRequiredFields(t *testing.T) {
	testCases := []struct {
		name          string
		env           map[string]string
		expectedError string
	}{
		{
			name: "missing owner",
			env: map[string]string{
				"GITHUB_REPO":  "repo",
				"GITHUB_REF":   "ref",
				"GITHUB_TOKEN": "token",
			},
			expectedError: "owner is required",
		},
		{
			name: "missing repo",
			env: map[string]string{
				"GITHUB_OWNER": "owner",
				"GITHUB_REF":   "ref",
				"GITHUB_TOKEN": "token",
			},
			expectedError: "repo is required",
		},
		{
			name: "missing ref",
			env: map[string]string{
				"GITHUB_OWNER": "owner",
				"GITHUB_REPO":  "repo",
				"GITHUB_TOKEN": "token",
			},
			expectedError: "ref is required",
		},
		{
			name: "missing token",
			env: map[string]string{
				"GITHUB_OWNER": "owner",
				"GITHUB_REPO":  "repo",
				"GITHUB_REF":   "ref",
			},
			expectedError: "token is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetConfig()

			// Set environment variables for the test case
			for key, value := range tc.env {
				os.Setenv(key, value)
			}

			_, err := config.LoadConfig()
			if err == nil || !strings.Contains(err.Error(), tc.expectedError) {
				t.Fatalf("expected error containing '%s', got '%v'", tc.expectedError, err)
			}
		})
	}
}

package cmd

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/tools/github-job-checker/internal/config"
)

// Run initializes the checker and starts the process.
func Run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	checker := &Checker{
		Owner:         cfg.Owner,
		Repo:          cfg.Repo,
		Ref:           cfg.Ref,
		CheckInterval: cfg.CheckInterval,
		Timeout:       cfg.Timeout,
		Client:        NewGitHubAPIClient(cfg.Token),
	}

	ctx := context.Background()
	return checker.Run(ctx)
}

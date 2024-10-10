package run

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
)

//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/runner.go . ProjectRunner

type ProjectRunner interface {
	RunTarget(target string, opts ...earthly.EarthlyExecutorOption) (map[string]earthly.EarthlyExecutionResult, error)
}

type DefaultProjectRunner struct {
	ctx      RunContext
	exectuor executor.Executor
	logger   *slog.Logger
	project  *project.Project
	store    secrets.SecretStore
}

// RunTarget runs the given Earthly target.
func (p *DefaultProjectRunner) RunTarget(
	target string,
	opts ...earthly.EarthlyExecutorOption,
) (map[string]earthly.EarthlyExecutionResult, error) {
	return earthly.NewEarthlyExecutor(
		p.project.Path,
		target,
		p.exectuor,
		p.store,
		p.logger,
		append(p.generateOpts(target), opts...)...,
	).Run()
}

// generateOpts generates the options for the Earthly executor.
func (p *DefaultProjectRunner) generateOpts(target string) []earthly.EarthlyExecutorOption {
	var opts []earthly.EarthlyExecutorOption

	if _, ok := p.project.Blueprint.Project.CI.Targets[target]; ok {
		targetConfig := p.project.Blueprint.Project.CI.Targets[target]

		if len(targetConfig.Args) > 0 {
			var args []string
			for k, v := range targetConfig.Args {
				args = append(args, fmt.Sprintf("--%s", k), v)
			}

			opts = append(opts, earthly.WithTargetArgs(args...))
		}

		// We only run multiple platforms in CI mode to avoid issues with local builds.
		if targetConfig.Platforms != nil && p.ctx.CI {
			opts = append(opts, earthly.WithPlatforms(targetConfig.Platforms...))
		}

		if targetConfig.Privileged != nil && *targetConfig.Privileged {
			opts = append(opts, earthly.WithPrivileged())
		}

		if targetConfig.Retries != nil {
			opts = append(opts, earthly.WithRetries(*targetConfig.Retries))
		}

		if len(targetConfig.Secrets) > 0 {
			opts = append(opts, earthly.WithSecrets(targetConfig.Secrets))
		}
	}

	if p.project.Blueprint.Global.CI.Providers.Earthly.Satellite != nil && !p.ctx.Local {
		opts = append(opts, earthly.WithSatellite(*p.project.Blueprint.Global.CI.Providers.Earthly.Satellite))
	}

	if len(p.project.Blueprint.Global.CI.Secrets) > 0 {
		opts = append(opts, earthly.WithSecrets(p.project.Blueprint.Global.CI.Secrets))
	}

	return opts
}

func NewDefaultProjectRunner(
	ctx RunContext,
	project *project.Project,
) DefaultProjectRunner {
	e := executor.NewLocalExecutor(
		ctx.Logger,
		executor.WithRedirect(),
	)

	return DefaultProjectRunner{
		ctx:      ctx,
		exectuor: e,
		logger:   ctx.Logger,
		project:  project,
		store:    ctx.SecretStore,
	}
}

func NewCustomDefaultProjectRunner(
	ctx RunContext,
	exec executor.Executor,
	logger *slog.Logger,
	project *project.Project,
	store secrets.SecretStore,
) DefaultProjectRunner {
	return DefaultProjectRunner{
		ctx:      ctx,
		exectuor: exec,
		logger:   logger,
		project:  project,
		store:    store,
	}
}

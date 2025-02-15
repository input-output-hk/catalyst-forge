package run

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/schema"
)

//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/runner.go . ProjectRunner

type ProjectRunner interface {
	RunTarget(target string, opts ...earthly.EarthlyExecutorOption) error
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
) error {
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

	if schema.HasProjectCiDefined(p.project.Blueprint) {
		if _, ok := p.project.Blueprint.Project.Ci.Targets[target]; ok {
			targetConfig := p.project.Blueprint.Project.Ci.Targets[target]

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

			if targetConfig.Privileged {
				opts = append(opts, earthly.WithPrivileged())
			}

			if targetConfig.Retries > 0 {
				opts = append(opts, earthly.WithRetries(int(targetConfig.Retries)))
			}

			if len(targetConfig.Secrets) > 0 {
				opts = append(opts, earthly.WithSecrets(targetConfig.Secrets))
			}
		}
	}

	if schema.HasEarthlyProviderDefined(p.project.Blueprint) {
		if p.project.Blueprint.Global.Ci.Providers.Earthly.Satellite != "" && !p.ctx.Local {
			opts = append(opts, earthly.WithSatellite(p.project.Blueprint.Global.Ci.Providers.Earthly.Satellite))
		}
	}

	if schema.HasGlobalCIDefined(p.project.Blueprint) {
		if len(p.project.Blueprint.Global.Ci.Secrets) > 0 {
			opts = append(opts, earthly.WithSecrets(p.project.Blueprint.Global.Ci.Secrets))
		}
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

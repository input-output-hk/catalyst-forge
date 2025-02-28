package earthly

import (
	"fmt"
	"log/slog"
	"regexp"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/schema"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
)

var (
	ErrNoMatchingTargets = fmt.Errorf("no matching targets found")
)

//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/runner.go . ProjectRunner

// ProjectRunner is an interface for running Earthly targets for a project.
type ProjectRunner interface {
	RunTarget(target string, opts ...EarthlyExecutorOption) error
}

// DefaultProjectRunner is the default implementation of the ProjectRunner interface.
type DefaultProjectRunner struct {
	ctx      run.RunContext
	exectuor executor.Executor
	logger   *slog.Logger
	project  *project.Project
	store    secrets.SecretStore
}

// RunTarget runs the given Earthly target.
func (p *DefaultProjectRunner) RunTarget(
	target string,
	opts ...EarthlyExecutorOption,
) error {
	popts, err := p.generateOpts(target)
	if err != nil {
		return err
	}

	return NewEarthlyExecutor(
		p.project.Path,
		target,
		p.exectuor,
		p.store,
		p.logger,
		append(popts, opts...)...,
	).Run()
}

// generateOpts generates the options for the Earthly executor.
func (p *DefaultProjectRunner) generateOpts(target string) ([]EarthlyExecutorOption, error) {
	var opts []EarthlyExecutorOption

	if schema.HasProjectCiDefined(p.project.Blueprint) {
		targetConfig, err := p.unifyTargets(p.project.Blueprint.Project.Ci.Targets, target)
		if err != nil && err != ErrNoMatchingTargets {
			return nil, err
		} else if err != ErrNoMatchingTargets {
			if len(targetConfig.Args) > 0 {
				var args []string
				for k, v := range targetConfig.Args {
					args = append(args, fmt.Sprintf("--%s", k), v)
				}

				opts = append(opts, WithTargetArgs(args...))
			}

			// We only run multiple platforms in CI mode to avoid issues with local builds.
			if targetConfig.Platforms != nil && p.ctx.CI {
				opts = append(opts, WithPlatforms(targetConfig.Platforms...))
			}

			if targetConfig.Privileged {
				opts = append(opts, WithPrivileged())
			}

			if targetConfig.Retries > 0 {
				opts = append(opts, WithRetries(int(targetConfig.Retries)))
			}

			if len(targetConfig.Secrets) > 0 {
				opts = append(opts, WithSecrets(targetConfig.Secrets))
			}
		}
	}

	if schema.HasEarthlyProviderDefined(p.project.Blueprint) {
		if p.project.Blueprint.Global.Ci.Providers.Earthly.Satellite != "" && !p.ctx.Local {
			opts = append(opts, WithSatellite(p.project.Blueprint.Global.Ci.Providers.Earthly.Satellite))
		}
	}

	if schema.HasGlobalCIDefined(p.project.Blueprint) {
		if len(p.project.Blueprint.Global.Ci.Secrets) > 0 {
			opts = append(opts, WithSecrets(p.project.Blueprint.Global.Ci.Secrets))
		}
	}

	return opts, nil
}

// unifyTargets unifies the targets that match the given name.
func (p *DefaultProjectRunner) unifyTargets(
	Targets map[string]sp.Target,
	name string,
) (sp.Target, error) {
	var targets []string
	for target := range Targets {
		filter, err := regexp.Compile(target)
		if err != nil {
			return sp.Target{}, fmt.Errorf("failed to compile target name '%s' to regex: %w", name, err)
		}

		if filter.MatchString(name) {
			targets = append(targets, target)
		}
	}

	fmt.Printf("targets: %v\n", targets)

	if len(targets) == 0 {
		return sp.Target{}, ErrNoMatchingTargets
	}

	var rt cue.Value
	ctx := cuecontext.New()
	for _, target := range targets {
		rt = rt.Unify(ctx.Encode(Targets[target]))
	}

	if rt.Err() != nil {
		return sp.Target{}, fmt.Errorf("failed to unify targets: %w", rt.Err())
	}

	var target sp.Target
	if err := rt.Decode(&target); err != nil {
		return sp.Target{}, fmt.Errorf("failed to decode unified targets: %w", err)
	}

	return target, nil
}

// NewDefaultProjectRunner creates a new DefaultProjectRunner instance.
func NewDefaultProjectRunner(
	ctx run.RunContext,
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

// NewCustomDefaultProjectRunner creates a new DefaultProjectRunner instance with custom dependencies.
func NewCustomDefaultProjectRunner(
	ctx run.RunContext,
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

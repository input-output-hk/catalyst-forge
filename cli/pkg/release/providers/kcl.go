package providers

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/release/providers/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/providers/aws"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/executor"
)

const (
	KCL_BINARY = "kcl"
)

type KCLReleaserConfig struct {
	Container string `json:"container"`
}

type KCLReleaser struct {
	config      KCLReleaserConfig
	ecr         aws.ECRClient
	force       bool
	handler     events.EventHandler
	kcl         executor.WrappedExecuter
	logger      *slog.Logger
	project     project.Project
	release     sp.Release
	releaseName string
}

func (r *KCLReleaser) Release() error {
	if !r.handler.Firing(&r.project, r.project.GetReleaseEvents(r.releaseName)) && !r.force {
		r.logger.Info("No release event is firing, skipping release")
		return nil
	}

	registries := r.project.Blueprint.Global.Ci.Providers.Kcl.Registries
	if len(registries) == 0 {
		return fmt.Errorf("must specify at least one KCL registry")
	}

	for _, registry := range registries {
		container := project.GenerateContainerName(&r.project, r.config.Container, registry)
		path, err := r.project.GetRelativePath()
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		if common.IsECRRegistry(registry) {
			r.logger.Info("Detected ECR registry, checking if repository exists", "repository", container)
			if err := common.CreateECRRepoIfNotExists(r.ecr, &r.project, container, r.logger); err != nil {
				return fmt.Errorf("failed to create ECR repository: %w", err)
			}
		}

		r.logger.Info("Publishing module", "path", path, "container", container)
		out, err := r.kcl.Execute("mod", "push", fmt.Sprintf("oci://%s", container))
		if err != nil {
			r.logger.Error("Failed to push module", "module", container, "error", err, "output", string(out))
			return fmt.Errorf("failed to push module: %w", err)
		}
	}

	return nil
}

// NewKCLReleaser creates a new KCL release provider.
func NewKCLReleaser(ctx run.RunContext,
	project project.Project,
	name string,
	force bool,
) (*KCLReleaser, error) {
	release, ok := project.Blueprint.Project.Release[name]
	if !ok {
		return nil, fmt.Errorf("unknown release: %s", name)
	}

	exec := executor.NewLocalExecutor(ctx.Logger, executor.WithWorkdir(project.Path))
	if _, ok := exec.LookPath(KCL_BINARY); ok != nil {
		return nil, fmt.Errorf("failed to find KCL binary: %w", ok)
	}

	var config KCLReleaserConfig
	err := common.ParseConfig(&project, name, &config)
	if err != nil && err != common.ErrConfigNotFound {
		return nil, fmt.Errorf("failed to parse release config: %w", err)
	}

	ecr, err := aws.NewECRClient(ctx.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create ECR client: %w", err)
	}

	kcl := executor.NewWrappedLocalExecutor(exec, "kcl")
	handler := events.NewDefaultEventHandler(ctx.Logger)
	return &KCLReleaser{
		config:      config,
		ecr:         ecr,
		force:       force,
		handler:     &handler,
		logger:      ctx.Logger,
		kcl:         kcl,
		project:     project,
		release:     release,
		releaseName: name,
	}, nil
}

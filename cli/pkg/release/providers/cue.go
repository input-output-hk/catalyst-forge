package providers

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

const CUE_BINARY = "cue"

type CueReleaserConfig struct {
	Version string `json:"version"`
}

type CueReleaser struct {
	config      CueReleaserConfig
	cue         executor.WrappedExecuter
	force       bool
	handler     events.EventHandler
	logger      *slog.Logger
	project     project.Project
	release     schema.Release
	releaseName string
}

func (r *CueReleaser) Release() error {
	if !r.handler.Firing(&r.project, r.project.GetReleaseEvents(r.releaseName)) && !r.force {
		r.logger.Info("No release event is firing, skipping release")
		return nil
	}

	registry := r.project.Blueprint.Global.CI.Providers.CUE.Registry
	if registry == nil {
		return fmt.Errorf("must specify at least one CUE registry")
	}

	if r.config.Version == "" {
		return fmt.Errorf("no version specified")
	}

	var fullRegistry string
	prefix := r.project.Blueprint.Global.CI.Providers.CUE.RegistryPrefix
	if prefix != nil {
		fullRegistry = fmt.Sprintf("%s/%s", *registry, *prefix)
	} else {
		fullRegistry = *registry
	}

	os.Setenv("CUE_REGISTRY", fullRegistry)
	defer os.Unsetenv("CUE_REGISTRY")

	path, err := r.project.GetRelativePath()
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	r.logger.Info("Publishing module", "path", path, "registry", fullRegistry, "version", r.config.Version)
	out, err := r.cue.Execute("mod", "publish", r.config.Version)
	if err != nil {
		r.logger.Error("Failed to publish module", "error", err, "output", string(out))
		return fmt.Errorf("failed to publish module: %w", err)
	}

	return nil
}

func NewCueReleaser(ctx run.RunContext,
	project project.Project,
	name string,
	force bool,
) (*CueReleaser, error) {
	release, ok := project.Blueprint.Project.Release[name]
	if !ok {
		return nil, fmt.Errorf("unknown release: %s", name)
	}

	exec := executor.NewLocalExecutor(ctx.Logger)
	if _, ok := exec.LookPath(CUE_BINARY); ok != nil {
		return nil, fmt.Errorf("failed to find cue binary: %w", ok)
	}

	var config CueReleaserConfig
	if err := parseConfig(&project, name, &config); err != nil {
		return nil, fmt.Errorf("failed to parse release config: %w", err)
	}

	cue := executor.NewLocalWrappedExecutor(exec, CUE_BINARY)
	handler := events.NewDefaultEventHandler(ctx.Logger)
	return &CueReleaser{
		config:      config,
		cue:         cue,
		force:       force,
		handler:     &handler,
		logger:      ctx.Logger,
		project:     project,
		release:     release,
		releaseName: name,
	}, nil
}

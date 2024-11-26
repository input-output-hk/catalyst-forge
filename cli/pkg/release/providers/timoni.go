package providers

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

const (
	TIMONI_BINARY = "timoni"
)

type TimoniReleaserConfig struct {
	Container string `json:"container"`
	Tag       string `json:"tag"`
}

type TimoniReleaser struct {
	config      TimoniReleaserConfig
	force       bool
	handler     events.EventHandler
	logger      *slog.Logger
	project     project.Project
	release     schema.Release
	releaseName string
	timoni      executor.WrappedExecuter
}

func (r *TimoniReleaser) Release() error {
	if !r.handler.Firing(&r.project, r.project.GetReleaseEvents(r.releaseName)) && !r.force {
		r.logger.Info("No release event is firing, skipping release")
		return nil
	}

	registries := r.project.Blueprint.Global.CI.Providers.Timoni.Registries
	if len(registries) == 0 {
		return fmt.Errorf("must specify at least one Timoni registry")
	}

	container := r.config.Container
	if container == "" {
		r.logger.Debug("Defaulting container name")
		container = fmt.Sprintf("%s-%s", r.project.Name, "deployment")
	}

	var tag string
	if r.project.Tag != nil {
		tag = strings.TrimPrefix(r.project.Tag.Version, "v")
	} else if r.config.Tag != "" {
		tag = r.config.Tag
	} else {
		return fmt.Errorf("no tag found")
	}

	for _, registry := range registries {
		fullContainer := fmt.Sprintf("oci://%s/%s:%s", registry, container, tag)
		path, err := r.project.GetRelativePath()
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		r.logger.Info("Publishing module", "path", path, "module", fullContainer)
		out, err := r.timoni.Execute("mod", "push", "--version", tag, "--latest=false", path, fullContainer)
		if err != nil {
			r.logger.Error("Failed to push module", "module", fullContainer, "error", err, "output", string(out))
			return fmt.Errorf("failed to push module: %w", err)
		}
	}

	return nil
}

// NewTimoniReleaser creates a new Timoni release provider.
func NewTimoniReleaser(ctx run.RunContext,
	project project.Project,
	name string,
	force bool,
) (*TimoniReleaser, error) {
	release, ok := project.Blueprint.Project.Release[name]
	if !ok {
		return nil, fmt.Errorf("unknown release: %s", name)
	}

	exec := executor.NewLocalExecutor(ctx.Logger)
	if _, ok := exec.LookPath(TIMONI_BINARY); ok != nil {
		return nil, fmt.Errorf("failed to find Timoni binary: %w", ok)
	}

	var config TimoniReleaserConfig
	if err := parseConfig(&project, name, &config); err != nil {
		return nil, fmt.Errorf("failed to parse release config: %w", err)
	}

	timoni := executor.NewLocalWrappedExecutor(exec, "timoni")
	handler := events.NewDefaultEventHandler(ctx.Logger)
	return &TimoniReleaser{
		config:      config,
		force:       force,
		handler:     &handler,
		logger:      ctx.Logger,
		project:     project,
		release:     release,
		releaseName: name,
		timoni:      timoni,
	}, nil
}

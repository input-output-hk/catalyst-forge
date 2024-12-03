package providers

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

const CUE_BINARY = "cue"

type CueReleaserConfig struct {
	Container string `json:"container"`
	Tag       string `json:"tag"`
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

	cue := executor.NewLocalWrappedExecutor(exec, "timoni")
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

package providers

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/publish/providers/common"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/executor"
)

const (
	TIMONI_BINARY = "timoni"
)

type TimoniPublisherConfig struct {
	Container string `json:"container"`
	Tag       string `json:"tag"`
}

type TimoniPublisher struct {
	config        TimoniPublisherConfig
	force         bool
	handler       events.EventHandler
	logger        *slog.Logger
	project       project.Project
	publisher     sp.Publisher
	publisherName string
	timoni        executor.WrappedExecuter
}

func (r *TimoniPublisher) Publish() error {
	if !r.handler.Firing(&r.project, r.project.GetPublisherEvents(r.publisherName)) && !r.force {
		r.logger.Info("No publisher event is firing, skipping publish")
		return nil
	}

	registries := r.project.Blueprint.Global.Ci.Providers.Timoni.Registries
	if len(registries) == 0 {
		return fmt.Errorf("must specify at least one Timoni registry")
	}

	container := r.config.Container
	if container == "" {
		r.logger.Debug("Defaulting container name")
		container = fmt.Sprintf("%s-%s", r.project.Name, "deployment")
	}

	tag := strings.TrimPrefix(r.config.Tag, "v")
	if tag == "" {
		return fmt.Errorf("no tag specified")
	}

	for _, registry := range registries {
		fullContainer := fmt.Sprintf("oci://%s/%s", registry, container)
		path, err := r.project.GetRelativePath()
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		r.logger.Info("Publishing module", "path", path, "container", fullContainer, "tag", tag)
		out, err := r.timoni.Execute("mod", "push", "--version", tag, "--latest=false", path, fullContainer)
		if err != nil {
			r.logger.Error("Failed to push module", "module", fullContainer, "error", err, "output", string(out))
			return fmt.Errorf("failed to push module: %w", err)
		}
	}

	return nil
}

// NewTimoniPublisher creates a new Timoni publish provider.
func NewTimoniPublisher(ctx run.RunContext,
	project project.Project,
	name string,
	force bool,
) (*TimoniPublisher, error) {
	publisher, ok := project.Blueprint.Project.Publishers[name]
	if !ok {
		return nil, fmt.Errorf("unknown publisher: %s", name)
	}

	exec := executor.NewLocalExecutor(ctx.Logger)
	if _, ok := exec.LookPath(TIMONI_BINARY); ok != nil {
		return nil, fmt.Errorf("failed to find Timoni binary: %w", ok)
	}

	var config TimoniPublisherConfig
	if err := common.ParseConfig(&project, name, &config); err != nil {
		return nil, fmt.Errorf("failed to parse publish config: %w", err)
	}

	timoni := executor.NewWrappedLocalExecutor(exec, "timoni")
	handler := events.NewDefaultEventHandler(ctx.Logger)
	return &TimoniPublisher{
		config:        config,
		force:         force,
		handler:       &handler,
		logger:        ctx.Logger,
		project:       project,
		publisher:     publisher,
		publisherName: name,
		timoni:        timoni,
	}, nil
}

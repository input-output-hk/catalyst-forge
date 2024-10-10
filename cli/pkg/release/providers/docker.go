package providers

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

const (
	DOCKER_BINARY  = "docker"
	CONTAINER_NAME = "container"
	TAG_NAME       = "tag"
)

type DockerReleaser struct {
	ctx     run.RunContext
	docker  executor.WrappedExecuter
	logger  *slog.Logger
	project project.Project
	release schema.Release
}

func (r *DockerReleaser) Release() error {
	if _, err := exec.LookPath(DOCKER_BINARY); err != nil {
		return fmt.Errorf("failed to find Docker binary: %w", err)
	}

	// if err := r.run(); err != nil {
	// 	return fmt.Errorf("failed to run release target: %w", err)
	// }

	if err := r.validateImages(); err != nil {
		return fmt.Errorf("failed to validate images: %w", err)
	}

	return nil
}

// imageExists checks if the image exists in the Docker daemon.
func (r *DockerReleaser) imageExists(image string) bool {
	r.logger.Info("Validating image exists", "image", image)
	out, err := r.docker.Execute("inspect", image)
	if err != nil {
		r.logger.Error("Failed to inspect image", "image", image, "error", err)
		r.logger.Error(string(out))
		return false
	}

	return true
}

// getPlatforms returns the platforms present in the release target, if any.
func (r *DockerReleaser) getPlatforms() []string {
	if _, ok := r.project.Blueprint.Project.CI.Targets[r.release.Target]; ok {
		if len(r.project.Blueprint.Project.CI.Targets[r.release.Target].Platforms) > 1 {
			return r.project.Blueprint.Project.CI.Targets[r.release.Target].Platforms
		}
	}

	return nil
}

// run runs the release target.
func (r *DockerReleaser) run() error {
	runner := run.NewProjectRunner(r.ctx, &r.project)
	_, err := runner.RunTarget(
		r.release.Target,
		earthly.WithTargetArgs("--container", CONTAINER_NAME, "--tag", TAG_NAME),
	)

	return err
}

// validateImages validates that the expected images exist in the Docker daemon.
func (r *DockerReleaser) validateImages() error {
	platforms := r.getPlatforms()
	if len(platforms) > 0 {
		for _, platform := range platforms {
			image := fmt.Sprintf("%s:%s_%s", CONTAINER_NAME, TAG_NAME, strings.Replace(platform, "/", "_", -1))
			if !r.imageExists(image) {
				return fmt.Errorf("image %s does not exist", image)
			}
		}
	} else {
		image := fmt.Sprintf("%s:%s", CONTAINER_NAME, TAG_NAME)
		if !r.imageExists(image) {
			return fmt.Errorf("image %s does not exist", image)
		}
	}

	return nil
}

// NewDockerReleaser creates a new Docker releaser.
func NewDockerReleaser(ctx run.RunContext, project project.Project, release schema.Release) (*DockerReleaser, error) {
	docker := executor.NewLocalWrappedExecutor(executor.NewLocalExecutor(ctx.Logger), "docker")
	return &DockerReleaser{
		ctx:     ctx,
		docker:  docker,
		logger:  ctx.Logger,
		project: project,
		release: release,
	}, nil
}

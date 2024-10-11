package providers

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/release/events"
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
	docker  executor.WrappedExecuter
	force   bool
	handler events.ReleaseEventHandler
	logger  *slog.Logger
	project project.Project
	release schema.Release
	runner  run.ProjectRunner
}

func (r *DockerReleaser) Release() error {
	r.logger.Info("Running release target", "project", r.project.Name, "target", r.release.Target)
	if err := r.run(); err != nil {
		return fmt.Errorf("failed to run release target: %w", err)
	}

	if err := r.validateImages(); err != nil {
		return fmt.Errorf("failed to validate images: %w", err)
	}

	if !r.handler.Firing(r.release.On) && !r.force {
		r.logger.Info("No release event is firing, skipping release")
		return nil
	}

	if r.project.Blueprint.Project.Container == "" {
		return fmt.Errorf("no container name found")
	} else if len(r.project.Blueprint.Global.CI.Registries) == 0 {
		return fmt.Errorf("no registries found")
	}

	container := r.project.Blueprint.Project.Container
	registries := r.project.Blueprint.Global.CI.Registries

	var imageTag string
	if r.project.TagInfo.Git != "" {
		if r.project.TagInfo.Git.IsMono() {
			imageTag = string(r.project.TagInfo.Git)
		} else {
			imageTag = string(r.project.TagInfo.Git)
		}
	} else {
		imageTag = string(r.project.TagInfo.Generated)
	}

	if len(r.getPlatforms()) > 0 {
		for _, registry := range registries {
			var pushed []string

			for _, platform := range r.getPlatforms() {
				platformSuffix := strings.Replace(platform, "/", "_", -1)
				curImage := fmt.Sprintf("%s:%s_%s", CONTAINER_NAME, TAG_NAME, platformSuffix)
				newImage := fmt.Sprintf("%s/%s:%s_%s", registry, container, imageTag, platformSuffix)

				r.logger.Debug("Tagging image", "tag", newImage)
				if err := r.tagImage(curImage, newImage); err != nil {
					return fmt.Errorf("failed to tag image: %w", err)
				}

				r.logger.Info("Pushing image", "image", newImage)
				if err := r.pushImage(newImage); err != nil {
					return fmt.Errorf("failed to push image: %w", err)
				}

				pushed = append(pushed, newImage)
			}

			mutliPlatformImage := fmt.Sprintf("%s/%s:%s", registry, container, imageTag)
			r.logger.Info("Pushing multi-platform image", "image", mutliPlatformImage)
			if err := r.pushMultiPlatformImage(mutliPlatformImage, pushed...); err != nil {
				return fmt.Errorf("failed to push multi-platform image: %w", err)
			}
		}
	} else {
		for _, registry := range registries {
			curImage := fmt.Sprintf("%s:%s", CONTAINER_NAME, TAG_NAME)
			newImage := fmt.Sprintf("%s/%s:%s", registry, container, imageTag)

			r.logger.Info("Tagging image", "old", curImage, "new", newImage)
			if err := r.tagImage(curImage, newImage); err != nil {
				return fmt.Errorf("failed to tag image: %w", err)
			}

			r.logger.Info("Pushing image", "image", newImage)
			if err := r.pushImage(newImage); err != nil {
				return fmt.Errorf("failed to push image: %w", err)
			}
		}
	}

	r.logger.Info("Release complete")
	return nil
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

// pushImage pushes the image to the Docker registry.
func (r *DockerReleaser) pushImage(image string) error {
	out, err := r.docker.Execute("push", image)
	if err != nil {
		r.logger.Error("Failed to push image", "image", image, "error", err)
		r.logger.Error(string(out))
		return err
	}

	return nil
}

func (r *DockerReleaser) pushMultiPlatformImage(image string, images ...string) error {
	cmd := []string{"buildx", "imagetools", "create", "--tag", image}
	cmd = append(cmd, images...)
	out, err := r.docker.Execute(cmd...)
	if err != nil {
		r.logger.Error("Failed to push multi-platform image", "image", image, "error", err)
		r.logger.Error(string(out))
		return err
	}

	return nil
}

// run runs the release target.
func (r *DockerReleaser) run() error {
	_, err := r.runner.RunTarget(
		r.release.Target,
		earthly.WithTargetArgs("--container", CONTAINER_NAME, "--tag", TAG_NAME),
	)

	return err
}

// tagImage tags the image with the given tag.
func (r *DockerReleaser) tagImage(image, tag string) error {
	r.logger.Info("Tagging image", "image", image, "tag", tag)
	out, err := r.docker.Execute("tag", image, tag)
	if err != nil {
		r.logger.Error("Failed to tag image", "image", image, "tag", tag, "error", err)
		r.logger.Error(string(out))
		return err
	}

	return nil
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
func NewDockerReleaser(
	ctx run.RunContext,
	project project.Project,
	release schema.Release,
	force bool,
) (*DockerReleaser, error) {
	exec := executor.NewLocalExecutor(ctx.Logger)
	if _, ok := exec.LookPath(DOCKER_BINARY); ok != nil {
		return nil, fmt.Errorf("failed to find Docker binary: %w", ok)
	} else if project.TagInfo == nil {
		return nil, fmt.Errorf("cannot publish without tag information")
	}

	docker := executor.NewLocalWrappedExecutor(exec, "docker")
	handler := events.NewDefaultReleaseEventHandler(&project, ctx.Logger)
	runner := run.NewDefaultProjectRunner(ctx, &project)
	return &DockerReleaser{
		docker:  docker,
		force:   force,
		handler: &handler,
		logger:  ctx.Logger,
		project: project,
		release: release,
		runner:  &runner,
	}, nil
}

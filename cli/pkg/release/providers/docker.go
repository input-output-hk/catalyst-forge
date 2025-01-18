package providers

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/providers/aws"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

const (
	DOCKER_BINARY  = "docker"
	CONTAINER_NAME = "container"
	TAG_NAME       = "tag"
)

type DockerReleaserConfig struct {
	Tag string `json:"tag"`
}

type DockerReleaser struct {
	config      DockerReleaserConfig
	docker      executor.WrappedExecuter
	ecr         aws.ECRClient
	force       bool
	handler     events.EventHandler
	logger      *slog.Logger
	project     project.Project
	release     schema.Release
	releaseName string
	runner      run.ProjectRunner
}

func (r *DockerReleaser) Release() error {
	r.logger.Info("Running release target", "project", r.project.Name, "target", r.release.Target)
	if err := r.run(); err != nil {
		return fmt.Errorf("failed to run release target: %w", err)
	}

	if err := r.validateImages(); err != nil {
		return fmt.Errorf("failed to validate images: %w", err)
	}

	if !r.handler.Firing(&r.project, r.project.GetReleaseEvents(r.releaseName)) && !r.force {
		r.logger.Info("No release event is firing, skipping release")
		return nil
	}

	if r.project.Blueprint.Project.Container == "" {
		return fmt.Errorf("no container name found")
	} else if len(r.project.Blueprint.Global.CI.Registries) == 0 {
		return fmt.Errorf("no registries found")
	}

	registries := r.project.Blueprint.Global.CI.Registries
	imageTag := r.config.Tag
	if imageTag == "" {
		return fmt.Errorf("no image tag specified")
	}

	platforms := getPlatforms(&r.project, r.release.Target)
	if len(platforms) > 0 {
		for _, registry := range registries {
			var pushed []string

			container := project.GenerateContainerName(&r.project, r.project.Blueprint.Project.Container, registry)
			if isECRRegistry(registry) {
				r.logger.Info("Detected ECR registry, checking if repository exists", "repository", container)
				if err := createECRRepoIfNotExists(r.ecr, &r.project, container, r.logger); err != nil {
					return fmt.Errorf("failed to create ECR repository: %w", err)
				}
			}

			for _, platform := range platforms {
				platformSuffix := strings.Replace(platform, "/", "_", -1)
				curImage := fmt.Sprintf("%s:%s_%s", CONTAINER_NAME, TAG_NAME, platformSuffix)
				newImage := fmt.Sprintf("%s:%s_%s", container, imageTag, platformSuffix)

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
			container := project.GenerateContainerName(&r.project, r.project.Blueprint.Project.Container, registry)
			if isECRRegistry(registry) {
				r.logger.Info("Detected ECR registry, checking if repository exists", "repository", container)
				if err := createECRRepoIfNotExists(r.ecr, &r.project, container, r.logger); err != nil {
					return fmt.Errorf("failed to create ECR repository: %w", err)
				}
			}

			curImage := fmt.Sprintf("%s:%s", CONTAINER_NAME, TAG_NAME)
			newImage := fmt.Sprintf("%s:%s", container, imageTag)

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
	return r.runner.RunTarget(
		r.release.Target,
		earthly.WithTargetArgs("--container", CONTAINER_NAME, "--tag", TAG_NAME),
	)
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
	platforms := getPlatforms(&r.project, r.release.Target)
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
	name string,
	force bool,
) (*DockerReleaser, error) {
	release, ok := project.Blueprint.Project.Release[name]
	if !ok {
		return nil, fmt.Errorf("unknown release: %s", name)
	}

	exec := executor.NewLocalExecutor(ctx.Logger)
	if _, ok := exec.LookPath(DOCKER_BINARY); ok != nil {
		return nil, fmt.Errorf("failed to find Docker binary: %w", ok)
	}

	var config DockerReleaserConfig
	if err := parseConfig(&project, name, &config); err != nil {
		return nil, fmt.Errorf("failed to parse release config: %w", err)
	}

	ecr, err := aws.NewECRClient(ctx.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create ECR client: %w", err)
	}

	docker := executor.NewLocalWrappedExecutor(exec, "docker")
	handler := events.NewDefaultEventHandler(ctx.Logger)
	runner := run.NewDefaultProjectRunner(ctx, &project)
	return &DockerReleaser{
		config:      config,
		docker:      docker,
		ecr:         ecr,
		force:       force,
		handler:     &handler,
		logger:      ctx.Logger,
		project:     project,
		release:     release,
		releaseName: name,
		runner:      &runner,
	}, nil
}

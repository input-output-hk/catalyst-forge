package release

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/release/providers"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

type ReleaserType string

const (
	ReleaserTypeDocker ReleaserType = "docker"
	ReleaserTypeGithub ReleaserType = "github"
)

type Releaser interface {
	Release() error
}

type ReleaserFactory func(run.RunContext, project.Project, string, bool) (Releaser, error)

type ReleaserStore struct {
	releasers map[ReleaserType]ReleaserFactory
}

func (r *ReleaserStore) GetReleaser(
	rtype ReleaserType,
	ctx run.RunContext,
	project project.Project,
	name string,
	force bool,
) (Releaser, error) {
	releaser, ok := r.releasers[rtype]
	if !ok {
		return nil, fmt.Errorf("unsupported releaser type: %s", rtype)
	}

	return releaser(ctx, project, name, force)
}

func NewDefaultReleaserStore() *ReleaserStore {
	return &ReleaserStore{
		releasers: map[ReleaserType]ReleaserFactory{
			ReleaserTypeDocker: func(ctx run.RunContext, project project.Project, name string, force bool) (Releaser, error) {
				return providers.NewDockerReleaser(ctx, project, name, force)
			},
			ReleaserTypeGithub: func(ctx run.RunContext, project project.Project, name string, force bool) (Releaser, error) {
				return providers.NewGithubReleaser(ctx, project, name, force)
			},
		},
	}
}

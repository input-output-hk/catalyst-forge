package release

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/release/providers"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

type ReleaserType string

const (
	ReleaserTypeCue    ReleaserType = "cue"
	ReleaserTypeDocker ReleaserType = "docker"
	ReleaserTypeDocs   ReleaserType = "docs"
	ReleaserTypeGithub ReleaserType = "github"
	ReleaserTypeKCL    ReleaserType = "kcl"
	ReleaserTypeTimoni ReleaserType = "timoni"
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
			ReleaserTypeCue: func(ctx run.RunContext, project project.Project, name string, force bool) (Releaser, error) {
				return providers.NewCueReleaser(ctx, project, name, force)
			},
			ReleaserTypeDocker: func(ctx run.RunContext, project project.Project, name string, force bool) (Releaser, error) {
				return providers.NewDockerReleaser(ctx, project, name, force)
			},
			ReleaserTypeDocs: func(ctx run.RunContext, project project.Project, name string, force bool) (Releaser, error) {
				return providers.NewDocsReleaser(ctx, project, name, force)
			},
			ReleaserTypeGithub: func(ctx run.RunContext, project project.Project, name string, force bool) (Releaser, error) {
				return providers.NewGithubReleaser(ctx, project, name, force)
			},
			ReleaserTypeKCL: func(ctx run.RunContext, project project.Project, name string, force bool) (Releaser, error) {
				return providers.NewKCLReleaser(ctx, project, name, force)
			},
			ReleaserTypeTimoni: func(ctx run.RunContext, project project.Project, name string, force bool) (Releaser, error) {
				return providers.NewTimoniReleaser(ctx, project, name, force)
			},
		},
	}
}

package publish

import (
    "fmt"

    "github.com/input-output-hk/catalyst-forge/cli/pkg/publish/providers"
    githubpub "github.com/input-output-hk/catalyst-forge/cli/pkg/publish/providers/github"
    "github.com/input-output-hk/catalyst-forge/cli/pkg/run"
    "github.com/input-output-hk/catalyst-forge/lib/project/project"
)

type PublisherType string

const (
    PublisherTypeCue    PublisherType = "cue"
    PublisherTypeDocker PublisherType = "docker"
    PublisherTypeDocs   PublisherType = "docs"
    PublisherTypeGithub PublisherType = "github"
    PublisherTypeKCL    PublisherType = "kcl"
    PublisherTypeTimoni PublisherType = "timoni"
)

type Publisher interface {
    Publish() error
}

type PublisherFactory func(run.RunContext, project.Project, string, bool) (Publisher, error)

type PublisherStore struct {
    publishers map[PublisherType]PublisherFactory
}

func (r *PublisherStore) GetPublisher(
    ptype PublisherType,
    ctx run.RunContext,
    project project.Project,
    name string,
    force bool,
) (Publisher, error) {
    publisher, ok := r.publishers[ptype]
    if !ok {
        return nil, fmt.Errorf("unsupported publisher type: %s", ptype)
    }

    return publisher(ctx, project, name, force)
}

func NewDefaultPublisherStore() *PublisherStore {
    return &PublisherStore{
        publishers: map[PublisherType]PublisherFactory{
            PublisherTypeCue: func(ctx run.RunContext, project project.Project, name string, force bool) (Publisher, error) {
                return providers.NewCuePublisher(ctx, project, name, force)
            },
            PublisherTypeDocker: func(ctx run.RunContext, project project.Project, name string, force bool) (Publisher, error) {
                return providers.NewDockerPublisher(ctx, project, name, force)
            },
            PublisherTypeDocs: func(ctx run.RunContext, project project.Project, name string, force bool) (Publisher, error) {
                return providers.NewDocsPublisher(ctx, project, name, force)
            },
            PublisherTypeGithub: func(ctx run.RunContext, project project.Project, name string, force bool) (Publisher, error) {
                return githubpub.NewPublisher(ctx, project, name, force)
            },
            PublisherTypeKCL: func(ctx run.RunContext, project project.Project, name string, force bool) (Publisher, error) {
                return providers.NewKCLPublisher(ctx, project, name, force)
            },
            PublisherTypeTimoni: func(ctx run.RunContext, project project.Project, name string, force bool) (Publisher, error) {
                return providers.NewTimoniPublisher(ctx, project, name, force)
            },
        },
    }
}


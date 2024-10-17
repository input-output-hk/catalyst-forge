package providers

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	evmocks "github.com/input-output-hk/catalyst-forge/cli/pkg/release/events/mocks"
	rmocks "github.com/input-output-hk/catalyst-forge/cli/pkg/run/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

func newReleaseEventHandlerMock(firing bool) *evmocks.ReleaseEventHandlerMock {
	return &evmocks.ReleaseEventHandlerMock{
		FiringFunc: func(p *project.Project, releaseName string) bool {
			return firing
		},
	}
}

func newProjectRunnerMock(fail bool) *rmocks.ProjectRunnerMock {
	return &rmocks.ProjectRunnerMock{
		RunTargetFunc: func(target string, opts ...earthly.EarthlyExecutorOption) error {
			if fail {
				return fmt.Errorf("failed to run release target")
			}
			return nil
		},
	}
}

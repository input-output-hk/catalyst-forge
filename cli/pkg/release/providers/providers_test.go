package providers

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	evmocks "github.com/input-output-hk/catalyst-forge/cli/pkg/release/events/mocks"
	rmocks "github.com/input-output-hk/catalyst-forge/cli/pkg/run/mocks"
)

func newReleaseEventHandlerMock(firing bool) *evmocks.ReleaseEventHandlerMock {
	return &evmocks.ReleaseEventHandlerMock{
		FiringFunc: func(events []string) bool {
			return firing
		},
	}
}

func newProjectRunnerMock(fail bool) *rmocks.ProjectRunnerMock {
	return &rmocks.ProjectRunnerMock{
		RunTargetFunc: func(target string, opts ...earthly.EarthlyExecutorOption) (map[string]earthly.EarthlyExecutionResult, error) {
			if fail {
				return nil, fmt.Errorf("failed to run release target")
			}
			return nil, nil
		},
	}
}

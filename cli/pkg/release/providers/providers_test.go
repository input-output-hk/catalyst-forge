package providers

import (
	"fmt"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	evmocks "github.com/input-output-hk/catalyst-forge/cli/pkg/events/mocks"
	rmocks "github.com/input-output-hk/catalyst-forge/cli/pkg/run/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

func newReleaseEventHandlerMock(firing bool) *evmocks.EventHandlerMock {
	return &evmocks.EventHandlerMock{
		FiringFunc: func(p *project.Project, events map[string]cue.Value) bool {
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

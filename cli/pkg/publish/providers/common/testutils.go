package common

import (
	"fmt"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	emocks "github.com/input-output-hk/catalyst-forge/cli/pkg/earthly/mocks"
	evmocks "github.com/input-output-hk/catalyst-forge/cli/pkg/events/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

func NewPublisherEventHandlerMock(firing bool) *evmocks.EventHandlerMock {
	return &evmocks.EventHandlerMock{
		FiringFunc: func(p *project.Project, events map[string]cue.Value) bool { return firing },
	}
}

func NewProjectRunnerMock(fail bool) *emocks.ProjectRunnerMock {
	return &emocks.ProjectRunnerMock{
		RunTargetFunc: func(target string, opts ...earthly.EarthlyExecutorOption) error {
			if fail {
				return fmt.Errorf("failed to run publisher target")
			}
			return nil
		},
	}
}

// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mocks

import (
	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"sync"
)

// Ensure, that ProjectRunnerMock does implement earthly.ProjectRunner.
// If this is not the case, regenerate this file with moq.
var _ earthly.ProjectRunner = &ProjectRunnerMock{}

// ProjectRunnerMock is a mock implementation of earthly.ProjectRunner.
//
//	func TestSomethingThatUsesProjectRunner(t *testing.T) {
//
//		// make and configure a mocked earthly.ProjectRunner
//		mockedProjectRunner := &ProjectRunnerMock{
//			RunTargetFunc: func(target string, opts ...earthly.EarthlyExecutorOption) error {
//				panic("mock out the RunTarget method")
//			},
//		}
//
//		// use mockedProjectRunner in code that requires earthly.ProjectRunner
//		// and then make assertions.
//
//	}
type ProjectRunnerMock struct {
	// RunTargetFunc mocks the RunTarget method.
	RunTargetFunc func(target string, opts ...earthly.EarthlyExecutorOption) error

	// calls tracks calls to the methods.
	calls struct {
		// RunTarget holds details about calls to the RunTarget method.
		RunTarget []struct {
			// Target is the target argument value.
			Target string
			// Opts is the opts argument value.
			Opts []earthly.EarthlyExecutorOption
		}
	}
	lockRunTarget sync.RWMutex
}

// RunTarget calls RunTargetFunc.
func (mock *ProjectRunnerMock) RunTarget(target string, opts ...earthly.EarthlyExecutorOption) error {
	if mock.RunTargetFunc == nil {
		panic("ProjectRunnerMock.RunTargetFunc: method is nil but ProjectRunner.RunTarget was just called")
	}
	callInfo := struct {
		Target string
		Opts   []earthly.EarthlyExecutorOption
	}{
		Target: target,
		Opts:   opts,
	}
	mock.lockRunTarget.Lock()
	mock.calls.RunTarget = append(mock.calls.RunTarget, callInfo)
	mock.lockRunTarget.Unlock()
	return mock.RunTargetFunc(target, opts...)
}

// RunTargetCalls gets all the calls that were made to RunTarget.
// Check the length with:
//
//	len(mockedProjectRunner.RunTargetCalls())
func (mock *ProjectRunnerMock) RunTargetCalls() []struct {
	Target string
	Opts   []earthly.EarthlyExecutorOption
} {
	var calls []struct {
		Target string
		Opts   []earthly.EarthlyExecutorOption
	}
	mock.lockRunTarget.RLock()
	calls = mock.calls.RunTarget
	mock.lockRunTarget.RUnlock()
	return calls
}

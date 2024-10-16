// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mocks

import (
	"github.com/input-output-hk/catalyst-forge/cli/pkg/release/events"
	"sync"
)

// Ensure, that ReleaseEventHandlerMock does implement events.ReleaseEventHandler.
// If this is not the case, regenerate this file with moq.
var _ events.ReleaseEventHandler = &ReleaseEventHandlerMock{}

// ReleaseEventHandlerMock is a mock implementation of events.ReleaseEventHandler.
//
//	func TestSomethingThatUsesReleaseEventHandler(t *testing.T) {
//
//		// make and configure a mocked events.ReleaseEventHandler
//		mockedReleaseEventHandler := &ReleaseEventHandlerMock{
//			FiringFunc: func(events []string) bool {
//				panic("mock out the Firing method")
//			},
//		}
//
//		// use mockedReleaseEventHandler in code that requires events.ReleaseEventHandler
//		// and then make assertions.
//
//	}
type ReleaseEventHandlerMock struct {
	// FiringFunc mocks the Firing method.
	FiringFunc func(events []string) bool

	// calls tracks calls to the methods.
	calls struct {
		// Firing holds details about calls to the Firing method.
		Firing []struct {
			// Events is the events argument value.
			Events []string
		}
	}
	lockFiring sync.RWMutex
}

// Firing calls FiringFunc.
func (mock *ReleaseEventHandlerMock) Firing(events []string) bool {
	if mock.FiringFunc == nil {
		panic("ReleaseEventHandlerMock.FiringFunc: method is nil but ReleaseEventHandler.Firing was just called")
	}
	callInfo := struct {
		Events []string
	}{
		Events: events,
	}
	mock.lockFiring.Lock()
	mock.calls.Firing = append(mock.calls.Firing, callInfo)
	mock.lockFiring.Unlock()
	return mock.FiringFunc(events)
}

// FiringCalls gets all the calls that were made to Firing.
// Check the length with:
//
//	len(mockedReleaseEventHandler.FiringCalls())
func (mock *ReleaseEventHandlerMock) FiringCalls() []struct {
	Events []string
} {
	var calls []struct {
		Events []string
	}
	mock.lockFiring.RLock()
	calls = mock.calls.Firing
	mock.lockFiring.RUnlock()
	return calls
}
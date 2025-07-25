// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mocks

import (
	"sync"
)

// GithubEnvMock is a mock implementation of github.GithubEnv.
//
//	func TestSomethingThatUsesGithubEnv(t *testing.T) {
//
//		// make and configure a mocked github.GithubEnv
//		mockedGithubEnv := &GithubEnvMock{
//			GetBranchFunc: func() string {
//				panic("mock out the GetBranch method")
//			},
//			GetEventPayloadFunc: func() (any, error) {
//				panic("mock out the GetEventPayload method")
//			},
//			GetEventTypeFunc: func() string {
//				panic("mock out the GetEventType method")
//			},
//			GetPRNumberFunc: func() int {
//				panic("mock out the GetPRNumber method")
//			},
//			GetTagFunc: func() string {
//				panic("mock out the GetTag method")
//			},
//			HasEventFunc: func() bool {
//				panic("mock out the HasEvent method")
//			},
//			IsPRFunc: func() bool {
//				panic("mock out the IsPR method")
//			},
//		}
//
//		// use mockedGithubEnv in code that requires github.GithubEnv
//		// and then make assertions.
//
//	}
type GithubEnvMock struct {
	// GetBranchFunc mocks the GetBranch method.
	GetBranchFunc func() string

	// GetEventPayloadFunc mocks the GetEventPayload method.
	GetEventPayloadFunc func() (any, error)

	// GetEventTypeFunc mocks the GetEventType method.
	GetEventTypeFunc func() string

	// GetPRNumberFunc mocks the GetPRNumber method.
	GetPRNumberFunc func() int

	// GetTagFunc mocks the GetTag method.
	GetTagFunc func() string

	// HasEventFunc mocks the HasEvent method.
	HasEventFunc func() bool

	// IsPRFunc mocks the IsPR method.
	IsPRFunc func() bool

	// calls tracks calls to the methods.
	calls struct {
		// GetBranch holds details about calls to the GetBranch method.
		GetBranch []struct {
		}
		// GetEventPayload holds details about calls to the GetEventPayload method.
		GetEventPayload []struct {
		}
		// GetEventType holds details about calls to the GetEventType method.
		GetEventType []struct {
		}
		// GetPRNumber holds details about calls to the GetPRNumber method.
		GetPRNumber []struct {
		}
		// GetTag holds details about calls to the GetTag method.
		GetTag []struct {
		}
		// HasEvent holds details about calls to the HasEvent method.
		HasEvent []struct {
		}
		// IsPR holds details about calls to the IsPR method.
		IsPR []struct {
		}
	}
	lockGetBranch       sync.RWMutex
	lockGetEventPayload sync.RWMutex
	lockGetEventType    sync.RWMutex
	lockGetPRNumber     sync.RWMutex
	lockGetTag          sync.RWMutex
	lockHasEvent        sync.RWMutex
	lockIsPR            sync.RWMutex
}

// GetBranch calls GetBranchFunc.
func (mock *GithubEnvMock) GetBranch() string {
	if mock.GetBranchFunc == nil {
		panic("GithubEnvMock.GetBranchFunc: method is nil but GithubEnv.GetBranch was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetBranch.Lock()
	mock.calls.GetBranch = append(mock.calls.GetBranch, callInfo)
	mock.lockGetBranch.Unlock()
	return mock.GetBranchFunc()
}

// GetBranchCalls gets all the calls that were made to GetBranch.
// Check the length with:
//
//	len(mockedGithubEnv.GetBranchCalls())
func (mock *GithubEnvMock) GetBranchCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetBranch.RLock()
	calls = mock.calls.GetBranch
	mock.lockGetBranch.RUnlock()
	return calls
}

// GetEventPayload calls GetEventPayloadFunc.
func (mock *GithubEnvMock) GetEventPayload() (any, error) {
	if mock.GetEventPayloadFunc == nil {
		panic("GithubEnvMock.GetEventPayloadFunc: method is nil but GithubEnv.GetEventPayload was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetEventPayload.Lock()
	mock.calls.GetEventPayload = append(mock.calls.GetEventPayload, callInfo)
	mock.lockGetEventPayload.Unlock()
	return mock.GetEventPayloadFunc()
}

// GetEventPayloadCalls gets all the calls that were made to GetEventPayload.
// Check the length with:
//
//	len(mockedGithubEnv.GetEventPayloadCalls())
func (mock *GithubEnvMock) GetEventPayloadCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetEventPayload.RLock()
	calls = mock.calls.GetEventPayload
	mock.lockGetEventPayload.RUnlock()
	return calls
}

// GetEventType calls GetEventTypeFunc.
func (mock *GithubEnvMock) GetEventType() string {
	if mock.GetEventTypeFunc == nil {
		panic("GithubEnvMock.GetEventTypeFunc: method is nil but GithubEnv.GetEventType was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetEventType.Lock()
	mock.calls.GetEventType = append(mock.calls.GetEventType, callInfo)
	mock.lockGetEventType.Unlock()
	return mock.GetEventTypeFunc()
}

// GetEventTypeCalls gets all the calls that were made to GetEventType.
// Check the length with:
//
//	len(mockedGithubEnv.GetEventTypeCalls())
func (mock *GithubEnvMock) GetEventTypeCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetEventType.RLock()
	calls = mock.calls.GetEventType
	mock.lockGetEventType.RUnlock()
	return calls
}

// GetPRNumber calls GetPRNumberFunc.
func (mock *GithubEnvMock) GetPRNumber() int {
	if mock.GetPRNumberFunc == nil {
		panic("GithubEnvMock.GetPRNumberFunc: method is nil but GithubEnv.GetPRNumber was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetPRNumber.Lock()
	mock.calls.GetPRNumber = append(mock.calls.GetPRNumber, callInfo)
	mock.lockGetPRNumber.Unlock()
	return mock.GetPRNumberFunc()
}

// GetPRNumberCalls gets all the calls that were made to GetPRNumber.
// Check the length with:
//
//	len(mockedGithubEnv.GetPRNumberCalls())
func (mock *GithubEnvMock) GetPRNumberCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetPRNumber.RLock()
	calls = mock.calls.GetPRNumber
	mock.lockGetPRNumber.RUnlock()
	return calls
}

// GetTag calls GetTagFunc.
func (mock *GithubEnvMock) GetTag() string {
	if mock.GetTagFunc == nil {
		panic("GithubEnvMock.GetTagFunc: method is nil but GithubEnv.GetTag was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetTag.Lock()
	mock.calls.GetTag = append(mock.calls.GetTag, callInfo)
	mock.lockGetTag.Unlock()
	return mock.GetTagFunc()
}

// GetTagCalls gets all the calls that were made to GetTag.
// Check the length with:
//
//	len(mockedGithubEnv.GetTagCalls())
func (mock *GithubEnvMock) GetTagCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetTag.RLock()
	calls = mock.calls.GetTag
	mock.lockGetTag.RUnlock()
	return calls
}

// HasEvent calls HasEventFunc.
func (mock *GithubEnvMock) HasEvent() bool {
	if mock.HasEventFunc == nil {
		panic("GithubEnvMock.HasEventFunc: method is nil but GithubEnv.HasEvent was just called")
	}
	callInfo := struct {
	}{}
	mock.lockHasEvent.Lock()
	mock.calls.HasEvent = append(mock.calls.HasEvent, callInfo)
	mock.lockHasEvent.Unlock()
	return mock.HasEventFunc()
}

// HasEventCalls gets all the calls that were made to HasEvent.
// Check the length with:
//
//	len(mockedGithubEnv.HasEventCalls())
func (mock *GithubEnvMock) HasEventCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockHasEvent.RLock()
	calls = mock.calls.HasEvent
	mock.lockHasEvent.RUnlock()
	return calls
}

// IsPR calls IsPRFunc.
func (mock *GithubEnvMock) IsPR() bool {
	if mock.IsPRFunc == nil {
		panic("GithubEnvMock.IsPRFunc: method is nil but GithubEnv.IsPR was just called")
	}
	callInfo := struct {
	}{}
	mock.lockIsPR.Lock()
	mock.calls.IsPR = append(mock.calls.IsPR, callInfo)
	mock.lockIsPR.Unlock()
	return mock.IsPRFunc()
}

// IsPRCalls gets all the calls that were made to IsPR.
// Check the length with:
//
//	len(mockedGithubEnv.IsPRCalls())
func (mock *GithubEnvMock) IsPRCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockIsPR.RLock()
	calls = mock.calls.IsPR
	mock.lockIsPR.RUnlock()
	return calls
}

// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mocks

import (
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"sync"
)

// Ensure, that BlueprintLoaderMock does implement blueprint.BlueprintLoader.
// If this is not the case, regenerate this file with moq.
var _ blueprint.BlueprintLoader = &BlueprintLoaderMock{}

// BlueprintLoaderMock is a mock implementation of blueprint.BlueprintLoader.
//
//	func TestSomethingThatUsesBlueprintLoader(t *testing.T) {
//
//		// make and configure a mocked blueprint.BlueprintLoader
//		mockedBlueprintLoader := &BlueprintLoaderMock{
//			LoadFunc: func(projectPath string, gitRootPath string) (blueprint.RawBlueprint, error) {
//				panic("mock out the Load method")
//			},
//		}
//
//		// use mockedBlueprintLoader in code that requires blueprint.BlueprintLoader
//		// and then make assertions.
//
//	}
type BlueprintLoaderMock struct {
	// LoadFunc mocks the Load method.
	LoadFunc func(projectPath string, gitRootPath string) (blueprint.RawBlueprint, error)

	// calls tracks calls to the methods.
	calls struct {
		// Load holds details about calls to the Load method.
		Load []struct {
			// ProjectPath is the projectPath argument value.
			ProjectPath string
			// GitRootPath is the gitRootPath argument value.
			GitRootPath string
		}
	}
	lockLoad sync.RWMutex
}

// Load calls LoadFunc.
func (mock *BlueprintLoaderMock) Load(projectPath string, gitRootPath string) (blueprint.RawBlueprint, error) {
	if mock.LoadFunc == nil {
		panic("BlueprintLoaderMock.LoadFunc: method is nil but BlueprintLoader.Load was just called")
	}
	callInfo := struct {
		ProjectPath string
		GitRootPath string
	}{
		ProjectPath: projectPath,
		GitRootPath: gitRootPath,
	}
	mock.lockLoad.Lock()
	mock.calls.Load = append(mock.calls.Load, callInfo)
	mock.lockLoad.Unlock()
	return mock.LoadFunc(projectPath, gitRootPath)
}

// LoadCalls gets all the calls that were made to Load.
// Check the length with:
//
//	len(mockedBlueprintLoader.LoadCalls())
func (mock *BlueprintLoaderMock) LoadCalls() []struct {
	ProjectPath string
	GitRootPath string
} {
	var calls []struct {
		ProjectPath string
		GitRootPath string
	}
	mock.lockLoad.RLock()
	calls = mock.calls.Load
	mock.lockLoad.RUnlock()
	return calls
}

// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mocks

import (
	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/blueprint"
	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/loader"
	"sync"
)

// Ensure, that BlueprintLoaderMock does implement loader.BlueprintLoader.
// If this is not the case, regenerate this file with moq.
var _ loader.BlueprintLoader = &BlueprintLoaderMock{}

// BlueprintLoaderMock is a mock implementation of loader.BlueprintLoader.
//
//	func TestSomethingThatUsesBlueprintLoader(t *testing.T) {
//
//		// make and configure a mocked loader.BlueprintLoader
//		mockedBlueprintLoader := &BlueprintLoaderMock{
//			LoadFunc: func(projectPath string, gitRootPath string) (blueprint.RawBlueprint, error) {
//				panic("mock out the Load method")
//			},
//			SetOverriderFunc: func(overrider loader.InjectorOverrider)  {
//				panic("mock out the SetOverrider method")
//			},
//		}
//
//		// use mockedBlueprintLoader in code that requires loader.BlueprintLoader
//		// and then make assertions.
//
//	}
type BlueprintLoaderMock struct {
	// LoadFunc mocks the Load method.
	LoadFunc func(projectPath string, gitRootPath string) (blueprint.RawBlueprint, error)

	// SetOverriderFunc mocks the SetOverrider method.
	SetOverriderFunc func(overrider loader.InjectorOverrider)

	// calls tracks calls to the methods.
	calls struct {
		// Load holds details about calls to the Load method.
		Load []struct {
			// ProjectPath is the projectPath argument value.
			ProjectPath string
			// GitRootPath is the gitRootPath argument value.
			GitRootPath string
		}
		// SetOverrider holds details about calls to the SetOverrider method.
		SetOverrider []struct {
			// Overrider is the overrider argument value.
			Overrider loader.InjectorOverrider
		}
	}
	lockLoad         sync.RWMutex
	lockSetOverrider sync.RWMutex
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

// SetOverrider calls SetOverriderFunc.
func (mock *BlueprintLoaderMock) SetOverrider(overrider loader.InjectorOverrider) {
	if mock.SetOverriderFunc == nil {
		panic("BlueprintLoaderMock.SetOverriderFunc: method is nil but BlueprintLoader.SetOverrider was just called")
	}
	callInfo := struct {
		Overrider loader.InjectorOverrider
	}{
		Overrider: overrider,
	}
	mock.lockSetOverrider.Lock()
	mock.calls.SetOverrider = append(mock.calls.SetOverrider, callInfo)
	mock.lockSetOverrider.Unlock()
	mock.SetOverriderFunc(overrider)
}

// SetOverriderCalls gets all the calls that were made to SetOverrider.
// Check the length with:
//
//	len(mockedBlueprintLoader.SetOverriderCalls())
func (mock *BlueprintLoaderMock) SetOverriderCalls() []struct {
	Overrider loader.InjectorOverrider
} {
	var calls []struct {
		Overrider loader.InjectorOverrider
	}
	mock.lockSetOverrider.RLock()
	calls = mock.calls.SetOverrider
	mock.lockSetOverrider.RUnlock()
	return calls
}

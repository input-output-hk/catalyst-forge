// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mocks

import (
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"sync"
)

// BlueprintInjectorMock is a mock implementation of injector.BlueprintInjector.
//
//	func TestSomethingThatUsesBlueprintInjector(t *testing.T) {
//
//		// make and configure a mocked injector.BlueprintInjector
//		mockedBlueprintInjector := &BlueprintInjectorMock{
//			InjectFunc: func(bp blueprint.RawBlueprint) blueprint.RawBlueprint {
//				panic("mock out the Inject method")
//			},
//		}
//
//		// use mockedBlueprintInjector in code that requires injector.BlueprintInjector
//		// and then make assertions.
//
//	}
type BlueprintInjectorMock struct {
	// InjectFunc mocks the Inject method.
	InjectFunc func(bp blueprint.RawBlueprint) blueprint.RawBlueprint

	// calls tracks calls to the methods.
	calls struct {
		// Inject holds details about calls to the Inject method.
		Inject []struct {
			// Bp is the bp argument value.
			Bp blueprint.RawBlueprint
		}
	}
	lockInject sync.RWMutex
}

// Inject calls InjectFunc.
func (mock *BlueprintInjectorMock) Inject(bp blueprint.RawBlueprint) blueprint.RawBlueprint {
	if mock.InjectFunc == nil {
		panic("BlueprintInjectorMock.InjectFunc: method is nil but BlueprintInjector.Inject was just called")
	}
	callInfo := struct {
		Bp blueprint.RawBlueprint
	}{
		Bp: bp,
	}
	mock.lockInject.Lock()
	mock.calls.Inject = append(mock.calls.Inject, callInfo)
	mock.lockInject.Unlock()
	return mock.InjectFunc(bp)
}

// InjectCalls gets all the calls that were made to Inject.
// Check the length with:
//
//	len(mockedBlueprintInjector.InjectCalls())
func (mock *BlueprintInjectorMock) InjectCalls() []struct {
	Bp blueprint.RawBlueprint
} {
	var calls []struct {
		Bp blueprint.RawBlueprint
	}
	mock.lockInject.RLock()
	calls = mock.calls.Inject
	mock.lockInject.RUnlock()
	return calls
}

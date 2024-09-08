// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mocks

import (
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/executor"
	"sync"
)

// Ensure, that ExecutorMock does implement executor.Executor.
// If this is not the case, regenerate this file with moq.
var _ executor.Executor = &ExecutorMock{}

// ExecutorMock is a mock implementation of executor.Executor.
//
//	func TestSomethingThatUsesExecutor(t *testing.T) {
//
//		// make and configure a mocked executor.Executor
//		mockedExecutor := &ExecutorMock{
//			ExecuteFunc: func(command string, args []string) ([]byte, error) {
//				panic("mock out the Execute method")
//			},
//		}
//
//		// use mockedExecutor in code that requires executor.Executor
//		// and then make assertions.
//
//	}
type ExecutorMock struct {
	// ExecuteFunc mocks the Execute method.
	ExecuteFunc func(command string, args []string) ([]byte, error)

	// calls tracks calls to the methods.
	calls struct {
		// Execute holds details about calls to the Execute method.
		Execute []struct {
			// Command is the command argument value.
			Command string
			// Args is the args argument value.
			Args []string
		}
	}
	lockExecute sync.RWMutex
}

// Execute calls ExecuteFunc.
func (mock *ExecutorMock) Execute(command string, args []string) ([]byte, error) {
	if mock.ExecuteFunc == nil {
		panic("ExecutorMock.ExecuteFunc: method is nil but Executor.Execute was just called")
	}
	callInfo := struct {
		Command string
		Args    []string
	}{
		Command: command,
		Args:    args,
	}
	mock.lockExecute.Lock()
	mock.calls.Execute = append(mock.calls.Execute, callInfo)
	mock.lockExecute.Unlock()
	return mock.ExecuteFunc(command, args)
}

// ExecuteCalls gets all the calls that were made to Execute.
// Check the length with:
//
//	len(mockedExecutor.ExecuteCalls())
func (mock *ExecutorMock) ExecuteCalls() []struct {
	Command string
	Args    []string
} {
	var calls []struct {
		Command string
		Args    []string
	}
	mock.lockExecute.RLock()
	calls = mock.calls.Execute
	mock.lockExecute.RUnlock()
	return calls
}
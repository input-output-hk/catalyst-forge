// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mocks

import (
	"context"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s"
	"sync"
)

// Ensure, that ClientMock does implement k8s.Client.
// If this is not the case, regenerate this file with moq.
var _ k8s.Client = &ClientMock{}

// ClientMock is a mock implementation of k8s.Client.
//
//	func TestSomethingThatUsesClient(t *testing.T) {
//
//		// make and configure a mocked k8s.Client
//		mockedClient := &ClientMock{
//			CreateDeploymentFunc: func(ctx context.Context, deployment *models.ReleaseDeployment) error {
//				panic("mock out the CreateDeployment method")
//			},
//		}
//
//		// use mockedClient in code that requires k8s.Client
//		// and then make assertions.
//
//	}
type ClientMock struct {
	// CreateDeploymentFunc mocks the CreateDeployment method.
	CreateDeploymentFunc func(ctx context.Context, deployment *models.ReleaseDeployment) error

	// calls tracks calls to the methods.
	calls struct {
		// CreateDeployment holds details about calls to the CreateDeployment method.
		CreateDeployment []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Deployment is the deployment argument value.
			Deployment *models.ReleaseDeployment
		}
	}
	lockCreateDeployment sync.RWMutex
}

// CreateDeployment calls CreateDeploymentFunc.
func (mock *ClientMock) CreateDeployment(ctx context.Context, deployment *models.ReleaseDeployment) error {
	if mock.CreateDeploymentFunc == nil {
		panic("ClientMock.CreateDeploymentFunc: method is nil but Client.CreateDeployment was just called")
	}
	callInfo := struct {
		Ctx        context.Context
		Deployment *models.ReleaseDeployment
	}{
		Ctx:        ctx,
		Deployment: deployment,
	}
	mock.lockCreateDeployment.Lock()
	mock.calls.CreateDeployment = append(mock.calls.CreateDeployment, callInfo)
	mock.lockCreateDeployment.Unlock()
	return mock.CreateDeploymentFunc(ctx, deployment)
}

// CreateDeploymentCalls gets all the calls that were made to CreateDeployment.
// Check the length with:
//
//	len(mockedClient.CreateDeploymentCalls())
func (mock *ClientMock) CreateDeploymentCalls() []struct {
	Ctx        context.Context
	Deployment *models.ReleaseDeployment
} {
	var calls []struct {
		Ctx        context.Context
		Deployment *models.ReleaseDeployment
	}
	mock.lockCreateDeployment.RLock()
	calls = mock.calls.CreateDeployment
	mock.lockCreateDeployment.RUnlock()
	return calls
}

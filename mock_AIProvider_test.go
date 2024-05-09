// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package nit

import (
	"sync"
)

// Ensure, that AIProviderMock does implement AIProvider.
// If this is not the case, regenerate this file with moq.
var _ AIProvider = &AIProviderMock{}

// AIProviderMock is a mock implementation of AIProvider.
//
//	func TestSomethingThatUsesAIProvider(t *testing.T) {
//
//		// make and configure a mocked AIProvider
//		mockedAIProvider := &AIProviderMock{
//			CreateCompletetionFunc: func(req *completionRequest) (*CompletionResponse, error) {
//				panic("mock out the CreateCompletetion method")
//			},
//		}
//
//		// use mockedAIProvider in code that requires AIProvider
//		// and then make assertions.
//
//	}
type AIProviderMock struct {
	// CreateCompletetionFunc mocks the CreateCompletetion method.
	CreateCompletetionFunc func(req *completionRequest) (*CompletionResponse, error)

	// calls tracks calls to the methods.
	calls struct {
		// CreateCompletetion holds details about calls to the CreateCompletetion method.
		CreateCompletetion []struct {
			// Req is the req argument value.
			Req *completionRequest
		}
	}
	lockCreateCompletetion sync.RWMutex
}

// CreateCompletetion calls CreateCompletetionFunc.
func (mock *AIProviderMock) CreateCompletetion(req *completionRequest) (*CompletionResponse, error) {
	if mock.CreateCompletetionFunc == nil {
		panic("AIProviderMock.CreateCompletetionFunc: method is nil but AIProvider.CreateCompletetion was just called")
	}
	callInfo := struct {
		Req *completionRequest
	}{
		Req: req,
	}
	mock.lockCreateCompletetion.Lock()
	mock.calls.CreateCompletetion = append(mock.calls.CreateCompletetion, callInfo)
	mock.lockCreateCompletetion.Unlock()
	return mock.CreateCompletetionFunc(req)
}

// CreateCompletetionCalls gets all the calls that were made to CreateCompletetion.
// Check the length with:
//
//	len(mockedAIProvider.CreateCompletetionCalls())
func (mock *AIProviderMock) CreateCompletetionCalls() []struct {
	Req *completionRequest
} {
	var calls []struct {
		Req *completionRequest
	}
	mock.lockCreateCompletetion.RLock()
	calls = mock.calls.CreateCompletetion
	mock.lockCreateCompletetion.RUnlock()
	return calls
}

// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package nit

import (
	goanthropic "github.com/madebywelch/anthropic-go/v2/pkg/anthropic"
	"sync"
)

// Ensure, that anthropicMock does implement anthropic.
// If this is not the case, regenerate this file with moq.
var _ anthropic = &anthropicMock{}

// anthropicMock is a mock implementation of anthropic.
//
//	func TestSomethingThatUsesanthropic(t *testing.T) {
//
//		// make and configure a mocked anthropic
//		mockedanthropic := &anthropicMock{
//			MessageFunc: func(req *goanthropic.MessageRequest) (*goanthropic.MessageResponse, error) {
//				panic("mock out the Message method")
//			},
//		}
//
//		// use mockedanthropic in code that requires anthropic
//		// and then make assertions.
//
//	}
type anthropicMock struct {
	// MessageFunc mocks the Message method.
	MessageFunc func(req *goanthropic.MessageRequest) (*goanthropic.MessageResponse, error)

	// calls tracks calls to the methods.
	calls struct {
		// Message holds details about calls to the Message method.
		Message []struct {
			// Req is the req argument value.
			Req *goanthropic.MessageRequest
		}
	}
	lockMessage sync.RWMutex
}

// Message calls MessageFunc.
func (mock *anthropicMock) Message(req *goanthropic.MessageRequest) (*goanthropic.MessageResponse, error) {
	if mock.MessageFunc == nil {
		panic("anthropicMock.MessageFunc: method is nil but anthropic.Message was just called")
	}
	callInfo := struct {
		Req *goanthropic.MessageRequest
	}{
		Req: req,
	}
	mock.lockMessage.Lock()
	mock.calls.Message = append(mock.calls.Message, callInfo)
	mock.lockMessage.Unlock()
	return mock.MessageFunc(req)
}

// MessageCalls gets all the calls that were made to Message.
// Check the length with:
//
//	len(mockedanthropic.MessageCalls())
func (mock *anthropicMock) MessageCalls() []struct {
	Req *goanthropic.MessageRequest
} {
	var calls []struct {
		Req *goanthropic.MessageRequest
	}
	mock.lockMessage.RLock()
	calls = mock.calls.Message
	mock.lockMessage.RUnlock()
	return calls
}

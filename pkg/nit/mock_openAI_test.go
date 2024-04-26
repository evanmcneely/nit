// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package review

import (
	"context"
	"github.com/sashabaranov/go-openai"
	"sync"
)

// Ensure, that openAIMock does implement openAI.
// If this is not the case, regenerate this file with moq.
var _ openAI = &openAIMock{}

// openAIMock is a mock implementation of openAI.
//
//	func TestSomethingThatUsesopenAI(t *testing.T) {
//
//		// make and configure a mocked openAI
//		mockedopenAI := &openAIMock{
//			CreateChatCompletionFunc: func(ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
//				panic("mock out the CreateChatCompletion method")
//			},
//		}
//
//		// use mockedopenAI in code that requires openAI
//		// and then make assertions.
//
//	}
type openAIMock struct {
	// CreateChatCompletionFunc mocks the CreateChatCompletion method.
	CreateChatCompletionFunc func(ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)

	// calls tracks calls to the methods.
	calls struct {
		// CreateChatCompletion holds details about calls to the CreateChatCompletion method.
		CreateChatCompletion []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Request is the request argument value.
			Request openai.ChatCompletionRequest
		}
	}
	lockCreateChatCompletion sync.RWMutex
}

// CreateChatCompletion calls CreateChatCompletionFunc.
func (mock *openAIMock) CreateChatCompletion(ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	if mock.CreateChatCompletionFunc == nil {
		panic("openAIMock.CreateChatCompletionFunc: method is nil but openAI.CreateChatCompletion was just called")
	}
	callInfo := struct {
		Ctx     context.Context
		Request openai.ChatCompletionRequest
	}{
		Ctx:     ctx,
		Request: request,
	}
	mock.lockCreateChatCompletion.Lock()
	mock.calls.CreateChatCompletion = append(mock.calls.CreateChatCompletion, callInfo)
	mock.lockCreateChatCompletion.Unlock()
	return mock.CreateChatCompletionFunc(ctx, request)
}

// CreateChatCompletionCalls gets all the calls that were made to CreateChatCompletion.
// Check the length with:
//
//	len(mockedopenAI.CreateChatCompletionCalls())
func (mock *openAIMock) CreateChatCompletionCalls() []struct {
	Ctx     context.Context
	Request openai.ChatCompletionRequest
} {
	var calls []struct {
		Ctx     context.Context
		Request openai.ChatCompletionRequest
	}
	mock.lockCreateChatCompletion.RLock()
	calls = mock.calls.CreateChatCompletion
	mock.lockCreateChatCompletion.RUnlock()
	return calls
}

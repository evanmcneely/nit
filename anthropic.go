package nit

import (
	goanthropic "github.com/madebywelch/anthropic-go/v2/pkg/anthropic"
)

//go:generate moq -out mock_anthropic_test.go . anthropic
type anthropic interface {
	Message(req *goanthropic.MessageRequest) (*goanthropic.MessageResponse, error)
}

type anthropicProvider struct {
	Client anthropic
}

func NewAnthropic(key string) *anthropicProvider {
	// only errors if the key is "", don't do that
	client, _ := goanthropic.NewClient(key)
	return &anthropicProvider{
		Client: client,
	}
}

func (a *anthropicProvider) CreateCompletetion(req *CompletionRequest) (*CompletionResponse, error) {
	model := a.getModel(req.Model)

	request := goanthropic.NewMessageRequest(
		[]goanthropic.MessagePartRequest{{Role: "user", Content: []goanthropic.ContentBlock{goanthropic.NewTextContentBlock(req.Prompt)}}},
		goanthropic.WithModel[goanthropic.MessageRequest](model),
		goanthropic.WithTemperature[goanthropic.MessageRequest](0),
		goanthropic.WithMaxTokens[goanthropic.MessageRequest](4096), // this is the maximum
	)

	completion, err := a.Client.Message(request)
	if err != nil {
		return nil, err
	}

	resp := &CompletionResponse{
		Completion: completion.Content[0].Text,
		Tokens:     completion.Usage.InputTokens + completion.Usage.OutputTokens,
	}

	return resp, nil
}

func (a *anthropicProvider) getModel(model string) goanthropic.Model {
	switch model {
	case modelCheap:
		return goanthropic.Claude3Haiku
	case modelGood:
		return goanthropic.Claude3Opus
	default:
		return goanthropic.Claude3Opus
	}
}

package nit

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

//go:generate moq -out mock_openAI_test.go . openAI
type openAI interface {
	CreateChatCompletion(ctx context.Context, request openai.ChatCompletionRequest) (response openai.ChatCompletionResponse, err error)
}

type openAIProvider struct {
	Client openAI
}

func NewOpenAI(key string) *openAIProvider {
	return &openAIProvider{
		Client: openai.NewClient(key),
	}
}

func (o *openAIProvider) CreateCompletetion(req *CompletionRequest) (*CompletionResponse, error) {
	model := o.getModel(req.Model)

	openAiRequest := openai.ChatCompletionRequest{
		Model:       model,
		Temperature: 0,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: req.Prompt,
			},
		},
	}

	if req.Format != "" {
		openAiRequest.ResponseFormat = &openai.ChatCompletionResponseFormat{Type: o.getCompletionFormat(req.Format)}
	}

	completion, err := o.Client.CreateChatCompletion(
		context.Background(),
		openAiRequest,
	)
	if err != nil {
		return nil, err
	}

	resp := &CompletionResponse{
		Completion: completion.Choices[0].Message.Content,
		Tokens:     completion.Usage.TotalTokens,
	}

	return resp, nil
}

func (o *openAIProvider) getModel(model string) string {
	switch model {
	case modelCheap:
		return openai.GPT3Dot5Turbo0125
	case modelGood:
		return openai.GPT4Turbo0125
	default:
		return openai.GPT4Turbo0125
	}
}

func (o *openAIProvider) getCompletionFormat(format string) openai.ChatCompletionResponseFormatType {
	switch format {
	case formatJSON:
		return openai.ChatCompletionResponseFormatTypeJSONObject
	case formatText:
		return openai.ChatCompletionResponseFormatTypeText
	default:
		return openai.ChatCompletionResponseFormatTypeText
	}
}

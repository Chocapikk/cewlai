package ai

import (
	"context"
	"fmt"

	"github.com/Chocapikk/cewlai/crawler"
	"github.com/openai/openai-go"
	openaiopt "github.com/openai/openai-go/option"
)

type openaiProvider struct {
	client openai.Client
	model  string
}

var openaiModels = map[string]string{
	"gpt-4.1-mini": string(openai.ChatModelGPT4_1Mini),
	"gpt-4.1":      string(openai.ChatModelGPT4_1),
	"gpt-4.1-nano": string(openai.ChatModelGPT4_1Nano),
	"gpt-4o-mini":  string(openai.ChatModelGPT4oMini),
	"gpt-4o":       string(openai.ChatModelGPT4o),
	"o3-mini":      string(openai.ChatModelO3Mini),
	"o3":           string(openai.ChatModelO3),
	"o4-mini":      string(openai.ChatModelO4Mini),
}

func newOpenAIProvider(apiKey, model, baseURL string) *openaiProvider {
	var opts []openaiopt.RequestOption
	if apiKey != "" {
		opts = append(opts, openaiopt.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, openaiopt.WithBaseURL(baseURL))
	}
	if resolved, ok := openaiModels[model]; ok {
		model = resolved
	} else if model == "" {
		model = openaiModels["gpt-4o-mini"]
	}
	return &openaiProvider{client: openai.NewClient(opts...), model: model}
}

func (p *openaiProvider) GenerateWords(ctx context.Context, result *crawler.CrawlResult, prompt string, maxTokens int) ([]string, error) {
	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:     openai.ChatModel(p.model),
		MaxTokens: openai.Int(int64(maxTokens)),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompt),
			openai.UserMessage(BuildUserMessage(result)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}
	return ParseAIResponse(resp.Choices[0].Message.Content), nil
}

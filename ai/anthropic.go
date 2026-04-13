package ai

import (
	"context"
	"fmt"

	"github.com/Chocapikk/cewlai/crawler"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type anthropicProvider struct {
	client anthropic.Client
	model  string
}

var anthropicModels = map[string]string{
	"haiku":  string(anthropic.ModelClaudeHaiku4_5),
	"sonnet": string(anthropic.ModelClaudeSonnet4_6),
	"opus":   string(anthropic.ModelClaudeOpus4_6),
}

func newAnthropicProvider(apiKey, model, baseURL string) *anthropicProvider {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	if resolved, ok := anthropicModels[model]; ok {
		model = resolved
	} else if model == "" {
		model = anthropicModels["sonnet"]
	}
	return &anthropicProvider{client: anthropic.NewClient(opts...), model: model}
}

func (p *anthropicProvider) GenerateWords(ctx context.Context, result *crawler.CrawlResult, prompt string) ([]string, error) {
	resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 4096,
		System:    []anthropic.TextBlockParam{{Text: prompt}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(BuildUserMessage(result))),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic API error: %w", err)
	}

	var text string
	for _, block := range resp.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	return ParseAIResponse(text), nil
}

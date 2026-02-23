// Package llm provides z.ai LLM provider implementation
package llm

import (
	"context"
	"fmt"
	"io"

	"github.com/sashabaranov/go-openai"
)

// ZAIProvider implements LLMProvider for z.ai (Claude API)
type ZAIProvider struct {
	client *openai.Client
	config *ProviderConfig
}

// NewZAIProvider creates a new z.ai provider instance
func NewZAIProvider(cfg *ProviderConfig) (*ZAIProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("ZAI_API_KEY is required")
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)
	clientConfig.BaseURL = cfg.BaseURL

	return &ZAIProvider{
		client: openai.NewClientWithConfig(clientConfig),
		config: cfg,
	}, nil
}

// GenerateRequest sends a request to z.ai provider and returns response
func (p *ZAIProvider) GenerateRequest(ctx context.Context, prompt string, characterCards []string) (string, error) {
	// Combine prompt with character cards context
	fullPrompt := p.buildPrompt(prompt, characterCards)

	req := openai.ChatCompletionRequest{
		Model:       p.config.Model,
		Messages:    []openai.ChatCompletionMessage{{Role: "user", Content: fullPrompt}},
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("z.ai request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("z.ai returned empty response")
	}

	return resp.Choices[0].Message.Content, nil
}

// StreamRequest sends a streaming request to z.ai provider
func (p *ZAIProvider) StreamRequest(ctx context.Context, prompt string, characterCards []string) (<-chan string, error) {
	ch := make(chan string)

	go func() {
		defer close(ch)

		fullPrompt := p.buildPrompt(prompt, characterCards)

		req := openai.ChatCompletionRequest{
			Model:       p.config.Model,
			Messages:    []openai.ChatCompletionMessage{{Role: "user", Content: fullPrompt}},
			MaxTokens:   4096,
			Temperature: 0.7,
			Stream:      true,
		}

		stream, err := p.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			ch <- fmt.Sprintf("Error: z.ai stream failed: %v", err)
			return
		}

		for {
			resp, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				ch <- fmt.Sprintf("Error: z.ai stream error: %v", err)
				return
			}

			for _, choice := range resp.Choices {
				if len(choice.Delta.Content) > 0 {
					ch <- choice.Delta.Content
				}
			}
		}
	}()

	return ch, nil
}

// Close closes the z.ai provider connection
func (p *ZAIProvider) Close() error {
	// No persistent connection to close
	return nil
}

// buildPrompt combines the prompt with character card context
func (p *ZAIProvider) buildPrompt(prompt string, characterCards []string) string {
	if len(characterCards) == 0 {
		return prompt
	}

	contextStr := "--- CHARACTER CARDS ---\n"
	for i, card := range characterCards {
		contextStr += fmt.Sprintf("Character %d:\n%s\n\n", i+1, card)
	}
	contextStr += "--- END CHARACTER CARDS ---\n\n"

	return contextStr + prompt
}

// parseConfig creates ProviderConfig from raw config data
func parseZAIConfig(rawConfig map[string]interface{}) (*ProviderConfig, error) {
	apiKey, ok := rawConfig["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("api_key is required for z.ai provider")
	}

	cfg := &ProviderConfig{
		Name:    "z.ai",
		APIKey:  apiKey,
		BaseURL: "https://api.z.ai/v1",
		Model:   "claude-3-5-sonnet-20240228",
	}

	if model, ok := rawConfig["model"].(string); ok && model != "" {
		cfg.Model = model
	}

	return cfg, nil
}

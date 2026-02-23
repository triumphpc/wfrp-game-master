// Package llm provides OpenAI-compatible LLM provider implementation
package llm

import (
	"context"
	"fmt"
	"io"

	"github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements LLMProvider for OpenAI-compatible APIs
type OpenAIProvider struct {
	client *openai.Client
	config *ProviderConfig
}

// NewOpenAIProvider creates a new OpenAI-compatible provider instance
func NewOpenAIProvider(cfg *ProviderConfig) (*OpenAIProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API_KEY is required for OpenAI provider")
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}

	return &OpenAIProvider{
		client: openai.NewClientWithConfig(clientConfig),
		config: cfg,
	}, nil
}

// GenerateRequest sends a request to OpenAI provider and returns response
func (p *OpenAIProvider) GenerateRequest(ctx context.Context, prompt string, characterCards []string) (string, error) {
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
		return "", fmt.Errorf("OpenAI request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("OpenAI returned empty response")
	}

	return resp.Choices[0].Message.Content, nil
}

// StreamRequest sends a streaming request to OpenAI provider
func (p *OpenAIProvider) StreamRequest(ctx context.Context, prompt string, characterCards []string) (<-chan string, error) {
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
			ch <- fmt.Sprintf("Error: OpenAI stream failed: %v", err)
			return
		}

		for {
			resp, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				ch <- fmt.Sprintf("Error: OpenAI stream error: %v", err)
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

// Close closes the OpenAI provider connection
func (p *OpenAIProvider) Close() error {
	// No persistent connection to close
	return nil
}

// buildPrompt combines the prompt with character card context
func (p *OpenAIProvider) buildPrompt(prompt string, characterCards []string) string {
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

// LoadProviderFromJSON creates a provider from JSON configuration
func LoadProviderFromJSON(data []byte) (*ProviderConfig, LLMProvider, error) {
	cfg, err := (&ConfigJSON{}).FromJSON(data)
	if err != nil {
		return nil, nil, err
	}

	provider, err := NewProviderFromConfig(cfg)
	if err != nil {
		return cfg, nil, err
	}

	return cfg, provider, nil
}

// NewProviderFromConfig creates an LLMProvider from ProviderConfig
func NewProviderFromConfig(cfg *ProviderConfig) (LLMProvider, error) {
	switch cfg.Name {
	case "z.ai", "zai":
		return NewZAIProvider(cfg)
	case "minimax":
		return NewMinimaxProvider(cfg)
	case "openai", "custom":
		return NewOpenAIProvider(cfg)
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Name)
	}
}

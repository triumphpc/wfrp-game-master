// Package llm provides minimax LLM provider implementation
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// MinimaxProvider implements LLMProvider for minimax
type MinimaxProvider struct {
	client *http.Client
	config *ProviderConfig
	apiURL string
}

// NewMinimaxProvider creates a new minimax provider instance
func NewMinimaxProvider(cfg *ProviderConfig) (*MinimaxProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("MINIMAX_API_KEY is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.minimax.chat/v1"
	}

	return &MinimaxProvider{
		client: &http.Client{},
		config: cfg,
		apiURL: baseURL + "/chat/completions",
	}, nil
}

// GenerateRequest sends a request to minimax provider and returns response
func (p *MinimaxProvider) GenerateRequest(ctx context.Context, prompt string, characterCards []string) (string, error) {
	fullPrompt := p.buildPrompt(prompt, characterCards)

	reqBody := minimaxRequest{
		Model:    p.config.Model,
		Messages: []message{{Role: "user", Content: fullPrompt}},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal minimax request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create minimax request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("minimax request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("minimax API error: %d - %s", resp.StatusCode, string(body))
	}

	var result minimaxResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode minimax response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("minimax returned empty response")
	}

	return result.Choices[0].Message.Content, nil
}

// StreamRequest sends a streaming request to minimax provider
func (p *MinimaxProvider) StreamRequest(ctx context.Context, prompt string, characterCards []string) (<-chan string, error) {
	ch := make(chan string)

	go func() {
		defer close(ch)

		fullPrompt := p.buildPrompt(prompt, characterCards)

		reqBody := minimaxRequest{
			Model:    p.config.Model,
			Messages: []message{{Role: "user", Content: fullPrompt}},
			Stream:   true,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			ch <- fmt.Sprintf("Error: failed to marshal minimax request: %v", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL, bytes.NewReader(jsonData))
		if err != nil {
			ch <- fmt.Sprintf("Error: failed to create minimax request: %v", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

		resp, err := p.client.Do(req)
		if err != nil {
			ch <- fmt.Sprintf("Error: minimax stream failed: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			ch <- fmt.Sprintf("Error: minimax API error: %d - %s", resp.StatusCode, string(body))
			return
		}

		decoder := json.NewDecoder(resp.Body)
		for {
			var chunk minimaxStreamChunk
			if err := decoder.Decode(&chunk); err != nil {
				if err == io.EOF {
					break
				}
				ch <- fmt.Sprintf("Error: minimax stream decode error: %v", err)
				return
			}

			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				ch <- chunk.Choices[0].Delta.Content
			}
		}
	}()

	return ch, nil
}

// Close closes the minimax provider connection
func (p *MinimaxProvider) Close() error {
	return nil
}

// buildPrompt combines the prompt with character card context
func (p *MinimaxProvider) buildPrompt(prompt string, characterCards []string) string {
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

// minimaxRequest represents the request payload for minimax API
type minimaxRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

// message represents a chat message
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// minimaxResponse represents the response from minimax API
type minimaxResponse struct {
	Choices []choice `json:"choices"`
}

// minimaxStreamChunk represents a streaming chunk from minimax API
type minimaxStreamChunk struct {
	Choices []streamChoice `json:"choices"`
}

// choice represents a response choice
type choice struct {
	Message delta `json:"message"`
}

// streamChoice represents a streaming choice
type streamChoice struct {
	Delta delta `json:"delta"`
}

// delta represents content in response
type delta struct {
	Content string `json:"content"`
}

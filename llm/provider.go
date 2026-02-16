// Package llm provides LLM provider integration for WFRP Game Master Bot
package llm

import (
	"context"
)

// LLMProvider defines the interface for LLM provider integration
type LLMProvider interface {
	// GenerateRequest sends a request to the LLM provider and returns the response
	GenerateRequest(ctx context.Context, prompt string, characterCards []string) (string, error)
	// StreamRequest sends a streaming request to the LLM provider
	StreamRequest(ctx context.Context, prompt string, characterCards []string) (<-chan string, error)
	// Close closes any open connections and releases resources
	Close() error
}

// RequestConfig holds configuration for LLM requests
type RequestConfig struct {
	Prompt         string
	CharacterCards []string
	Model          string
	MaxTokens      int
	Temperature    float64
}

// Response holds the LLM response
type Response struct {
	Content string
	Model   string
	Tokens  int
}

// ProviderConfig holds configuration for LLM providers
type ProviderConfig struct {
	Name   string
	APIKey string
	BaseURL string
	Model  string
}

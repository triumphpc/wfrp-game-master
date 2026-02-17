// Package llm provides manager for dynamic provider switching and response handling
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// ProviderManager manages LLM providers with hot-reload capability
type ProviderManager struct {
	currentProvider  LLMProvider
	config           *ProviderConfig
	configReloadChan chan struct{}
	mu               sync.Mutex
}

// NewProviderManager creates a new provider manager
func NewProviderManager(cfg *ProviderConfig) (*ProviderManager, error) {
	provider, err := NewProviderFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return &ProviderManager{
		currentProvider:  provider,
		config:           cfg,
		configReloadChan: make(chan struct{}, 1),
		mu:               sync.Mutex{},
	}, nil
}

// GetCurrentProvider returns the current provider
func (pm *ProviderManager) GetCurrentProvider() LLMProvider {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.currentProvider
}

// GetCurrentConfig returns the current provider config
func (pm *ProviderManager) GetCurrentConfig() *ProviderConfig {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.config
}

// ReloadProvider reloads the provider configuration without restarting the bot
func (pm *ProviderManager) ReloadProvider(newConfig *ProviderConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	provider, err := NewProviderFromConfig(newConfig)
	if err != nil {
		log.Printf("Failed to create new provider: %v", err)
		return fmt.Errorf("failed to create new provider: %w", err)
	}

	pm.currentProvider = provider
	pm.config = newConfig

	log.Printf("Provider successfully reloaded: %s", newConfig.Name)

	return nil
}

// GetReloadChannel returns a channel that receives reload signals
func (pm *ProviderManager) GetReloadChannel() <-chan struct{} {
	return pm.configReloadChan
}

// TriggerReload signals that a config reload is needed
func (pm *ProviderManager) TriggerReload() {
	select {
	case pm.configReloadChan <- struct{}{}:
	default:
	}
}

// MonitorConfigReload monitors the config reload channel and triggers reloads
func (pm *ProviderManager) MonitorConfigReload(callback func() error) {
	go func() {
		for range pm.configReloadChan {
			log.Println("Config reload signal received")
			if err := callback(); err != nil {
				log.Printf("Failed to reload config: %v", err)
			}
		}
	}()
}

// ResponseHandler handles LLM responses and formats them appropriately
type ResponseHandler struct {
	pm *ProviderManager
}

// NewResponseHandler creates a new response handler
func NewResponseHandler(pm *ProviderManager) *ResponseHandler {
	return &ResponseHandler{pm: pm}
}

// BuildRequest builds a request with context and character cards
func (rh *ResponseHandler) BuildRequest(prompt string, characterCards []string) string {
	rh.pm.mu.Lock()
	rh.pm.mu.Unlock()

	contextStr := ""
	if len(characterCards) > 0 {
		contextStr = "--- CHARACTER CARDS ---\n"
		for i, card := range characterCards {
			contextStr += fmt.Sprintf("Character %d:\n%s\n\n", i+1, card)
		}
		contextStr += "--- END CHARACTER CARDS ---\n\n"
	}

	return contextStr + prompt
}

// HandleResponse processes LLM responses and can format them as JSON or text
func (rh *ResponseHandler) HandleResponse(response string, format string) (interface{}, error) {
	switch format {
	case "json":
		var data interface{}
		if err := json.Unmarshal([]byte(response), &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		return data, nil
	case "text":
		return response, nil
	default:
		return response, nil
	}
}

// GenerateRequest sends a request to the current provider with error handling
func (rh *ResponseHandler) GenerateRequest(ctx context.Context, prompt string, characterCards []string) (string, error) {
	rh.pm.mu.Lock()
	provider := rh.pm.currentProvider
	cfg := rh.pm.config
	rh.pm.mu.Unlock()

	log.Printf("Making request to %s provider", cfg.Name)

	startTime := time.Now()
	response, err := provider.GenerateRequest(ctx, prompt, characterCards)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("Request to %s failed: %v (duration: %v)", cfg.Name, err, duration)
		return "", fmt.Errorf("provider %s failed: %w", cfg.Name, err)
	}

	log.Printf("Request to %s succeeded (duration: %v)", cfg.Name, duration)
	return response, nil
}

// StreamRequest sends a streaming request with error handling
func (rh *ResponseHandler) StreamRequest(ctx context.Context, prompt string, characterCards []string) (<-chan string, error) {
	rh.pm.mu.Lock()
	provider := rh.pm.currentProvider
	cfg := rh.pm.config
	rh.pm.mu.Unlock()

	log.Printf("Making streaming request to %s provider", cfg.Name)

	stream, err := provider.StreamRequest(ctx, prompt, characterCards)
	if err != nil {
		log.Printf("Failed to create streaming request to %s: %v", cfg.Name, err)
		return nil, fmt.Errorf("failed to create streaming request: %w", err)
	}

	log.Printf("Streaming request to %s started", cfg.Name)
	return stream, nil
}

// ErrorNotifier handles and notifies about API errors
type ErrorNotifier struct {
	adminChatID string
}

// NewErrorNotifier creates a new error notifier
func NewErrorNotifier(chatID string) *ErrorNotifier {
	return &ErrorNotifier{adminChatID: chatID}
}

// NotifyProviderError notifies about provider-specific errors
func (en *ErrorNotifier) NotifyProviderError(providerName, errorType, message string) {
	errMsg := fmt.Sprintf("⚠️ **%s Provider Error** ⚠️\n\n**Error Type:** %s\n**Message:** %s\n\nPlease check your configuration.", providerName, errorType, message)

	log.Printf("Provider error notification: %s", errMsg)

	if en.adminChatID != "" {
		log.Printf("Sending notification to admin chat %s: %s", en.adminChatID, errMsg)
	}
}

// NotifyAPIError notifies about generic API errors
func (en *ErrorNotifier) NotifyAPIError(message string, statusCode int) {
	errMsg := fmt.Sprintf("⚠️ **API Error** ⚠️\n\n**Status Code:** %d\n**Message:** %s", statusCode, message)

	log.Printf("API error notification: %s", errMsg)

	if en.adminChatID != "" {
		log.Printf("Sending notification to admin chat %s: %s", en.adminChatID, errMsg)
	}
}

// NotifyRequestFailed notifies about request failures
func (en *ErrorNotifier) NotifyRequestFailed(providerName, requestType, details string) {
	errMsg := fmt.Sprintf("⚠️ **%s Request Failed** ⚠️\n\n**Provider:** %s\n**Request Type:** %s\n**Details:** %s", requestType, providerName, requestType, details)

	log.Printf("Request failed notification: %s", errMsg)

	if en.adminChatID != "" {
		log.Printf("Sending notification to admin chat %s: %s", en.adminChatID, errMsg)
	}
}

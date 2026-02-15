// Package config provides configuration loading for WFRP Game Master Bot
package config

import (
	"fmt"
	"os"
)

// ProviderConfig represents configuration for an LLM provider
type ProviderConfig struct {
	Name   string
	APIKey  string
	BaseURL string
	Model   string
	Params  map[string]string
}

// BotConfig represents bot configuration loaded from environment variables
type BotConfig struct {
	TelegramToken     string
	DefaultProvider  string
	Providers        map[string]ProviderConfig
	GroupID          string
}

// LoadConfig loads bot configuration from environment variables
func LoadConfig() (BotConfig, error) {
	token := getEnv("TELEGRAM_BOT_TOKEN", "")
	if token == "" {
		return BotConfig{}, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	defaultProvider := getEnv("DEFAULT_PROVIDER", "openai")
	if defaultProvider == "" {
		return BotConfig{}, fmt.Errorf("DEFAULT_PROVIDER is required")
	}

	groupID := getEnv("TELEGRAM_GROUP_ID", "")
	if groupID == "" {
		return BotConfig{}, fmt.Errorf("TELEGRAM_GROUP_ID is required")
	}

	providers := make(map[string]ProviderConfig)

	// Parse provider configurations from environment
	// z.ai provider
	if apiKey := getEnv("ZAI_API_KEY", ""); apiKey != "" {
		providers["zai"] = ProviderConfig{
			Name:   "z.ai",
			APIKey: apiKey,
			BaseURL: "https://api.z.ai/v1",
			Model:  "claude-3-5-sonnet-20240228",
		}
	}

	// minimax provider
	if apiKey := getEnv("MINIMAX_API_KEY", ""); apiKey != "" {
		providers["minimax"] = ProviderConfig{
			Name:   "minimax",
			APIKey: apiKey,
			BaseURL: "https://api.minimax.chat/v1",
			Model:  "minimax-text",
		}
	}

	// OpenAI-compatible providers (e.g., open.ai, others using same API)
	for _, providerName := range []string{"openai", "custom"} {
		if apiKey := getEnv(fmt.Sprintf("%s_API_KEY", providerName), ""); apiKey != "" {
			baseURL := getEnv(fmt.Sprintf("%s_BASE_URL", providerName), "https://api.openai.com/v1")
			model := getEnv(fmt.Sprintf("%s_MODEL", providerName), "gpt-4o")
			providers[providerName] = ProviderConfig{
				Name:   providerName,
				APIKey: apiKey,
				BaseURL: baseURL,
				Model:   model,
			}
		}
	}

	return BotConfig{
		TelegramToken:     token,
		DefaultProvider: defaultProvider,
		Providers:        providers,
		GroupID:          groupID,
	}, nil
}

// getEnv retrieves an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

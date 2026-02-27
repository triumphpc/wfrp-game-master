// Package llm provides common configuration structures for LLM providers
package llm

import (
	"encoding/json"
	"fmt"
)

// ConfigJSON represents common API configuration for JSON parsing
type ConfigJSON struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
}

// FromJSON creates ProviderConfig from JSON configuration
func (c *ConfigJSON) FromJSON(data []byte) (*ProviderConfig, error) {
	var cfg ConfigJSON
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse provider config: %w", err)
	}

	pc := &ProviderConfig{
		Name:   "custom",
		APIKey: cfg.APIKey,
		Model:  cfg.Model,
	}

	if cfg.BaseURL != "" {
		pc.BaseURL = cfg.BaseURL
	} else {
		pc.BaseURL = "https://api.openai.com/v1"
	}

	if pc.Model == "" {
		pc.Model = "gpt-4o"
	}

	return pc, nil
}

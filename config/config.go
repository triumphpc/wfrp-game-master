// Package config provides configuration loading for WFRP Game Master Bot
package config

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// ProviderConfig представляет конфигурацию LLM провайдера
//
// Поля:
//   - Name: имя провайдера (z.ai, minimax, openai, custom)
//   - APIKey: API ключ провайдера
//   - BaseURL: базовый URL API провайдера
//   - Model: название модели для использования
//   - Params: дополнительные параметры провайдера
type ProviderConfig struct {
	Name    string
	APIKey  string
	BaseURL string
	Model   string
	Params  map[string]string
}

// BotConfig представляет конфигурацию бота, загруженную из переменных окружения
//
// Поля:
//   - TelegramToken: токен Telegram бота (обязательное поле)
//   - DefaultProvider: название LLM провайдера по умолчанию
//   - Providers: карта всех зарегистрированных провайдеров
//   - GroupID: идентификатор группы Telegram
type BotConfig struct {
	TelegramToken   string
	DefaultProvider string
	Providers       map[string]ProviderConfig
	GroupID         string
}

// LoadConfig загружает конфигурацию бота из переменных окружения
//
// Возвращает структуру BotConfig с обязательными полями:
// - TELEGRAM_BOT_TOKEN: токен бота (обязательный)
// - DEFAULT_PROVIDER: провайдер по умолчанию (обязательный)
// - TELEGRAM_GROUP_ID: идентификатор группы (обязательный)
//
// Дополнительно загружает конфигурации провайдеров из соответствующих переменных:
// - {PROVIDER}_API_KEY: API ключ
// - {PROVIDER}_BASE_URL: базовый URL (по умолчанию OpenAI или соответствующий провайдеру)
// - {PROVIDER}_MODEL: модель (по умолчанию gpt-4o или модель провайдера)
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
			Name:    "z.ai",
			APIKey:  apiKey,
			BaseURL: "https://api.z.ai/v1",
			Model:   "claude-3-5-sonnet-20240228",
		}
	}

	// minimax provider
	if apiKey := getEnv("MINIMAX_API_KEY", ""); apiKey != "" {
		providers["minimax"] = ProviderConfig{
			Name:    "minimax",
			APIKey:  apiKey,
			BaseURL: "https://api.minimax.chat/v1",
			Model:   "minimax-text",
		}
	}

	// OpenAI-compatible providers (e.g., open.ai, others using same API)
	for _, providerName := range []string{"openai", "custom"} {
		if apiKey := getEnv(fmt.Sprintf("%s_API_KEY", providerName), ""); apiKey != "" {
			baseURL := getEnv(fmt.Sprintf("%s_BASE_URL", providerName), "https://api.openai.com/v1")
			model := getEnv(fmt.Sprintf("%s_MODEL", providerName), "gpt-4o")
			providers[providerName] = ProviderConfig{
				Name:    providerName,
				APIKey:  apiKey,
				BaseURL: baseURL,
				Model:   model,
			}
		}
	}

	return BotConfig{
		TelegramToken:   token,
		DefaultProvider: defaultProvider,
		Providers:       providers,
		GroupID:         groupID,
	}, nil
}

// ReloadConfig перезагружает конфигурацию из переменных окружения
//
// Использует LoadConfig для получения обновленной конфигурации.
// Может быть использован для динамического обновления настроек без перезапуска бота.
func ReloadConfig() (BotConfig, error) {
	return LoadConfig()
}

// SetupConfigReload настраивает обработку сигналов для перезагрузки конфигурации
//
// Регистрирует обработчик сигнала SIGHUP (файл-дескриптор должен быть доступен)
// и вызывает callback функцию при получении сигнала.
//
// Пример использования:
//
//	config.SetupConfigReload(func() error {
//	    newCfg, err := config.ReloadConfig()
//	    if err != nil {
//	        return err
//	    }
//	    return updateBotConfig(newCfg)
//	})
func SetupConfigReload(callback func() error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)

	go func() {
		for sig := range sigChan {
			if sig == syscall.SIGHUP {
				log.Println("SIGHUP received, reloading configuration...")
				if err := callback(); err != nil {
					log.Printf("Failed to reload configuration: %v", err)
				}
			}
		}
	}()
}

// getEnv retrieves an environment variable with a default value
//
// Если переменная окружения не установлена, возвращает defaultValue.
// Используется для удобного доступа к конфигурации без проверки на ноль.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

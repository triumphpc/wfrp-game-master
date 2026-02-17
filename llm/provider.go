// Package llm provides LLM provider integration for WFRP Game Master Bot
package llm

import (
	"context"
)

// LLMProvider определяет интерфейс для интеграции LLM провайдеров
//
// Все провайдеры должны реализовать этот интерфейс:
// - GenerateRequest: отправка запроса и получение полного ответа
// - StreamRequest: отправка потокового запроса с частичными ответами
// - Close: закрытие соединений и освобождение ресурсов
type LLMProvider interface {
	// GenerateRequest отправляет запрос к LLM провайдеру и возвращает полный ответ
	// - ctx: контекст для отмены запроса
	// - prompt: промпт с инструкциями для AI
	// - characterCards: массив карточек персонажей для контекста
	// Возвращает текстовый ответ или ошибку
	GenerateRequest(ctx context.Context, prompt string, characterCards []string) (string, error)

	// StreamRequest отправляет потоковый запрос к LLM провайдеру
	// - ctx: контекст для отмены запроса
	// - prompt: промпт с инструкциями для AI
	// - characterCards: массив карточек персонажей для контекста
	// Возвращает канал для получения фрагментов ответа
	StreamRequest(ctx context.Context, prompt string, characterCards []string) (<-chan string, error)

	// Close закрывает соединения и освобождает ресурсы провайдера
	Close() error
}

// RequestConfig хранит конфигурацию для запросов к LLM
type RequestConfig struct {
	Prompt         string   // Промпт с инструкциями
	CharacterCards []string // Карточки персонажей для контекста
	Model          string   // Модель для использования
	MaxTokens      int      // Максимальное количество токенов
	Temperature    float64  // Параметр температуры (0.0-1.0)
}

// Response хранит ответ от LLM
type Response struct {
	Content string // Содержимое ответа
	Model   string // Использованная модель
	Tokens  int    // Количество использованных токенов
}

// ProviderConfig хранит конфигурацию для LLM провайдеров
type ProviderConfig struct {
	Name    string // Имя провайдера (z.ai, minimax, openai, custom)
	APIKey  string // API ключ провайдера
	BaseURL string // Базовый URL API провайдера
	Model   string // Модель для использования
}

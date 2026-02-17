package main

import (
	"log"

	"wfrp-bot/config"
)

// WFRP Game Master Bot - Telegram бот для ведения игр Warhammer Fantasy Roleplay 4th Edition
//
// Основные возможности:
// - Интеграция с Telegram Bot API
// - Поддержка нескольких LLM провайдеров (OpenAI, z.ai, Minimax, Custom)
// - Горячая перезагрузка конфигурации без перезапуска
// - Управление кампаниями и персонажами
// - Интеграция с RAG-MCP-Server для проверки правил
//
// Подробная документация: https://github.com/your-org/wfrp-game-master
func main() {
	log.Println("WFRP Game Master Bot")
	log.Println("Starting bot...")

	// Загрузка конфигурации из переменных окружения
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Config loaded. Provider: %s, Group ID: %s", cfg.DefaultProvider, cfg.GroupID)

	// TODO: Инициализация провайдера LLM
	// TODO: Настройка Telegram бота
	// TODO: Запуск игровых сессий
}

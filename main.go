package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wfrp-bot/config"
	"wfrp-bot/game"
	"wfrp-bot/llm"
	"wfrp-bot/storage"
	"wfrp-bot/telegram"
)

// WFRP Game Master Bot - Telegram бот для ведения игр Warhammer Fantasy Roleplay 4th Edition
//
// Основные возможности:
// - Интеграция с Telegram Bot API
// - Поддержка нескольких LLM провайдеров (OpenAI, z.ai, Minimax, Custom)
// - Горячая перезагрузка конфигурации без перезапуска
// - Управление кампаниями и персонажами
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

	// Инициализация путей хранения
	basePath := os.Getenv("BASE_PATH")
	if basePath == "" {
		basePath = "./storage"
	}

	// Инициализация LLM провайдера
	_, err = llm.NewProviderFromConfig(&llm.ProviderConfig{
		Name:    cfg.DefaultProvider,
		APIKey:  cfg.Providers[cfg.DefaultProvider].APIKey,
		BaseURL: cfg.Providers[cfg.DefaultProvider].BaseURL,
		Model:   cfg.Providers[cfg.DefaultProvider].Model,
	})
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}
	log.Printf("LLM provider initialized: %s", cfg.DefaultProvider)

	// Создание LLM менеджера
	_, err = llm.NewProviderManager(&llm.ProviderConfig{
		Name:    cfg.DefaultProvider,
		APIKey:  cfg.Providers[cfg.DefaultProvider].APIKey,
		BaseURL: cfg.Providers[cfg.DefaultProvider].BaseURL,
		Model:   cfg.Providers[cfg.DefaultProvider].Model,
	})
	if err != nil {
		log.Fatalf("Failed to create provider manager: %v", err)
	}

	// Инициализация Telegram бота
	bot, err := telegram.NewBot(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}

	// Создание rate limiter
	limiter := telegram.NewRateLimiter(1 * time.Second)

	// Создание менеджера сессий
	sessionManager := game.NewSessionManager()

	// Передача sessionManager в bot для обработки сообщений игроков
	bot.SetSessionManager(sessionManager)

	// Создание менеджера персонажей
	characterManager := game.NewCharacterManager(basePath)

	// Создание менеджера кампаний
	campaignManager := storage.NewCampaignManager(basePath)

	// Создание менеджера истории
	_ = storage.NewHistoryManager(basePath)

	// Создание обработчиков команд
	handlers := telegram.NewCommandHandlers(bot, sessionManager, characterManager, campaignManager)

	// Регистрация всех обработчиков
	handlers.RegisterAllHandlers()

	// Передача обработчиков в бота для обработки создания персонажей
	bot.SetCommandHandlers(handlers)

	// Добавление middleware для логирования, ограничений и работы только в группе
	bot.AddMiddleware(telegram.LoggingMiddleware)
	bot.AddMiddleware(telegram.RateLimitMiddleware(limiter))
	bot.AddMiddleware(telegram.GroupOnlyMiddleware(cfg.GroupID))

	// Запуск бота
	log.Println("Starting Telegram bot polling...")
	if err := bot.Start(10 * time.Second); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	log.Println("Bot started successfully!")
	log.Println("Use /help to see available commands")

	// Ожидание сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down bot...")
	bot.Stop()
	log.Println("Bot stopped")
}

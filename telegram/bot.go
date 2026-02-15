// Package telegram provides Telegram Bot API integration for WFRP Game Master Bot
package telegram

import (
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CommandHandler handles bot commands
type CommandHandler func(update *tgbotapi.Update, args []string) error

// Middleware processes updates before handlers
type Middleware func(update *tgbotapi.Update) (bool, error)

// Bot represents a Telegram bot instance
type Bot struct {
	api      *tgbotapi.BotAPI
	handlers map[string]CommandHandler
	middleware []Middleware
	updates  <-chan tgbotapi.Update
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
}

// NewBot creates a new Telegram bot instance
func NewBot(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		api:      api,
		handlers: make(map[string]CommandHandler),
		middleware: make([]Middleware, 0),
		stopChan: make(chan struct{}),
	}

	log.Printf("Telegram bot authorized as @%s", api.Self.UserName)

	return bot, nil
}

// AddCommand registers a command handler
func (b *Bot) AddCommand(name string, handler CommandHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = handler
	log.Printf("Registered command: /%s", name)
}

// AddMiddleware adds middleware to the bot
func (b *Bot) AddMiddleware(mw Middleware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.middleware = append(b.middleware, mw)
}

// Start begins receiving updates
func (b *Bot) Start(pollingTimeout time.Duration) error {
	u := tgbotapi.NewUpdate(0, 0)
	u.Timeout = pollingTimeout

	updates, err := b.api.GetUpdatesChan(u)
	if err != nil {
		return err
	}

	b.updates = updates
	log.Println("Bot started polling for updates")

	b.wg.Add(1)
	go b.processUpdates()

	return nil
}

// Stop gracefully stops the bot
func (b *Bot) Stop() {
	close(b.stopChan)
	b.wg.Wait()
	log.Println("Bot stopped")
}

// processUpdates processes incoming updates
func (b *Bot) processUpdates() {
	defer b.wg.Done()

	for {
		select {
		case <-b.stopChan:
			return
		case update, ok := <-b.updates:
			if !ok {
				return
			}
			b.handleUpdate(update)
		}
}

// HandleUpdate processes all incoming updates (commands and messages)
func (b *Bot) HandleUpdate(update *tgbotapi.Update) error {
	// Run middleware chain
	for _, mw := range b.middleware {
		cont, err := mw(update)
		if err != nil {
			log.Printf("Middleware error: %v", err)
			return err
		}
		if !cont {
			return nil // Middleware blocked this update
		}
	}

	// Handle commands
	if update.Message != nil && update.Message.IsCommand() {
		return b.handleCommand(update)
	}

	// Handle regular messages from players
	if update.Message != nil && update.Message.Text != "" {
		return b.handlePlayerMessage(update)
	}

	// Handle callback queries
	if update.CallbackQuery != nil {
		return b.handleCallbackQuery(update)
	}

	return nil
}

// handleCommand processes a command
func (b *Bot) handleCommand(update *tgbotapi.Update) error {
	command := update.Message.Command()
	args := update.Message.CommandArguments()

	b.mu.RLock()
	handler, exists := b.handlers[command]
	b.mu.RUnlock()

	if !exists {
		log.Printf("Unknown command: %s", command)
		return nil
	}

	if err := handler(update, args); err != nil {
		log.Printf("Handler error for /%s: %v", command, err)
	}

	return nil
}

// handlePlayerMessage processes non-command messages from players
func (b *Bot) handlePlayerMessage(update *tgbotapi.Update) error {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	text := update.Message.Text

	log.Printf("[MSG] Player %d: %s", userID, text)

	// Forward to session manager for processing
	// This would be integrated with game session
	// For now, just log the message
	return nil
}

// handleCallbackQuery processes callback button presses
func (b *Bot) handleCallbackQuery(update *tgbotapi.Update) error {
	userID := update.CallbackQuery.From.ID
	data := update.CallbackQuery.Data

	log.Printf("[CALLBACK] User %d: %s", userID, data)

	// Handle callback actions
	// This would be integrated with game session for button interactions
	return nil
}

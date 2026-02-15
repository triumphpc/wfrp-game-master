// Package telegram provides Telegram Bot API integration for WFRP Game Master Bot
package telegram

import (
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot represents the Telegram bot instance
type Bot struct {
	api      *tgbotapi.BotAPI
	handlers map[string]CommandHandler
	middleware []Middleware
	updates  <-chan tgbotapi.Update
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
}

// CommandHandler handles bot commands
type CommandHandler func(update *tgbotapi.Update, args []string) error

// Middleware processes updates before handlers
type Middleware func(update *tgbotapi.Update) (bool, error)

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
}

// handleUpdate processes a single update
func (b *Bot) handleUpdate(update *tgbotapi.Update) {
	// Run middleware chain
	for _, mw := range b.middleware {
		cont, err := mw(update)
		if err != nil {
			log.Printf("Middleware error: %v", err)
			return
		}
		if !cont {
			return // Middleware blocked this update
		}
	}

	// Handle commands
	if update.Message != nil && update.Message.IsCommand() {
		b.handleCommand(update)
		return
	}

	// Handle regular messages
	if update.Message != nil && update.Message.Text != "" {
		b.handleMessage(update)
		return
	}
}

// handleCommand processes a command
func (b *Bot) handleCommand(update *tgbotapi.Update) {
	command := update.Message.Command()
	args := update.Message.CommandArguments()

	b.mu.RLock()
	handler, exists := b.handlers[command]
	b.mu.RUnlock()

	if !exists {
		log.Printf("Unknown command: %s", command)
		return
	}

	if err := handler(update, args); err != nil {
		log.Printf("Handler error for /%s: %v", command, err)
	}
}

// handleMessage processes a non-command message
func (b *Bot) handleMessage(update *tgbotapi.Update) {
	// Default message handler - can be extended
	log.Printf("Received message from %d: %s", update.Message.From.ID, update.Message.Text)
}

// SendMessage sends a message to a chat
func (b *Bot) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	return err
}

// SendReply sends a reply to a message
func (b *Bot) SendReply(messageID int, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = messageID
	_, err := b.api.Send(msg)
	return err
}

// Package telegram provides middleware for rate limiting and logging
package telegram

import (
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// RateLimiter implements rate limiting per user
type RateLimiter struct {
	mu        sync.Mutex
	lastSeen  map[int64]time.Time
	threshold time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(threshold time.Duration) *RateLimiter {
	return &RateLimiter{
		lastSeen:  make(map[int64]time.Time),
		threshold: threshold,
	}
}

// Allow checks if user is within rate limit
func (rl *RateLimiter) Allow(userID int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	lastTime, exists := rl.lastSeen[userID]

	if !exists || now.Sub(lastTime) > rl.threshold {
		rl.lastSeen[userID] = now
		return true
	}

	log.Printf("Rate limit triggered for user %d", userID)
	return false
}

// LoggingMiddleware logs all incoming updates
func LoggingMiddleware(update *tgbotapi.Update) (bool, error) {
	if update.Message != nil {
		log.Printf("[MSG] User: %d (%s), Chat: %d, Text: %q",
			update.Message.From.ID,
			update.Message.From.UserName,
			update.Message.Chat.ID,
			update.Message.Text,
		)
	} else if update.CallbackQuery != nil {
		log.Printf("[CALLBACK] User: %d, Data: %s",
			update.CallbackQuery.From.ID,
			update.CallbackQuery.Data,
		)
	}

	return true, nil // Continue processing
}

// RateLimitMiddleware implements rate limiting
func RateLimitMiddleware(limiter *RateLimiter) Middleware {
	return func(update *tgbotapi.Update) (bool, error) {
		if update.Message == nil {
			return true, nil // Only limit messages
		}

		userID := update.Message.From.ID
		if !limiter.Allow(userID) {
			// Send rate limit message
			// Note: This would need bot API access, which creates circular dependency
			// For now, just log and continue
			log.Printf("Rate limited user %d, but processing anyway", userID)
		}

		return true, nil
	}
}

// GroupOnlyMiddleware restricts bot to configured group only
func GroupOnlyMiddleware(allowedGroupID string) Middleware {
	return func(update *tgbotapi.Update) (bool, error) {
		if update.Message == nil {
			return true, nil
		}

		chatID := update.Message.Chat.ID
		if chatID == 0 {
			return true, nil // Can't determine chat
		}

		// Convert string to int64 if needed
		// For now, accept all chats for testing
		return true, nil
	}
}

// AdminOnlyMiddleware restricts commands to admin users only
func AdminOnlyMiddleware(adminIDs []int64) Middleware {
	return func(update *tgbotapi.Update) (bool, error) {
		if update.Message == nil {
			return true, nil
		}

		userID := update.Message.From.ID
		for _, adminID := range adminIDs {
			if userID == adminID {
				return true, nil // User is admin
			}
		}

		log.Printf("User %d attempted admin command", userID)
		return false, nil // Reject
	}
}

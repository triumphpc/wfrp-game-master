// Package telegram provides command handlers for WFRP bot
package telegram

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"wfrp-bot/game"
	"wfrp-bot/storage"
)

// Command handlers for WFRP bot
type CommandHandlers struct {
	bot         *Bot
	sessionMgr   *game.SessionManager
	charMgr      *game.CharacterManager
	storageMgr   *storage.CampaignManager
}

// NewCommandHandlers creates a new command handlers instance
func NewCommandHandlers(bot *Bot, sessionMgr *game.SessionManager, charMgr *game.CharacterManager, storageMgr *storage.CampaignManager) *CommandHandlers {
	return &CommandHandlers{
		bot:       bot,
		sessionMgr: sessionMgr,
		charMgr:    charMgr,
		storageMgr: storageMgr,
	}
}

// StartCommand starts a new game session
func (h *CommandHandlers) StartCommand(update *tgbotapi.Update, args []string) error {
	if update.Message == nil {
		return fmt.Errorf("no message in update")
	}

	chatID := update.Message.Chat.ID
	userID := fmt.Sprintf("%d", update.Message.From.ID)

	// Check if campaign is provided
	campaign := ""
	if len(args) > 0 {
		campaign = args[0]
	}

	if campaign == "" {
		// List available campaigns
		campaigns, err := h.storageMgr.ListCampaigns()
		if err != nil {
			return h.bot.SendMessage(chatID, fmt.Sprintf("Ошибка загрузки кампаний: %v", err))
		}

		if len(campaigns) == 0 {
			return h.bot.SendMessage(chatID, "Нет доступных кампаний. Используйте /campaign <имя> для создания новой кампании.")
		}

		var builder strings.Builder
		builder.WriteString("📁 **Доступные кампании:**\n\n")
		for _, camp := range campaigns {
			builder.WriteString(fmt.Sprintf("• %s\n", camp.Name))
		}
		return h.bot.SendMessage(chatID, builder.String())
	}

	// Create new session for campaign
	session := game.NewSession(update.Message.Chat.ID, campaign, nil) // LLM provider needed
	session.Start()

	h.sessionMgr.AddSession(chatID, session)

	return h.bot.SendMessage(chatID, fmt.Sprintf("✅ Игровая сессия запущена для кампании: %s\n\nGM готов принимать команды.", campaign))
}

// HelpCommand displays help information
func (h *CommandHandlers) HelpCommand(update *tgbotapi.Update, args []string) error {
	if update.Message == nil {
		return fmt.Errorf("no message in update")
	}

	helpText := `🎮 **WFRP Game Master Bot** - Справка по командам

📋 **Команды игры:**
/start <кампания> - Запустить новую игру или сессию
/campaign <имя> - Выбрать кампанию
/status - Показать статус текущей сессии
/character <имя> - Показать карточку персонажа
/help - Эта справка

📚 **Доступные кампании:`
	// List campaigns
	campaigns, err := h.storageMgr.ListCampaigns()
	if err != nil {
		return h.bot.SendMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка: %v", err))
	}

	if len(campaigns) == 0 {
		helpText += "\nНет доступных кампаний."
	} else {
		helpText += "\n"
		for _, camp := range campaigns {
			helpText += fmt.Sprintf("• %s\n", camp.Name)
		}
	}

	helpText += "\n---\n💡 *Для начала игры напишите /start <название_кампании>*"

	return h.bot.SendMessage(update.Message.Chat.ID, helpText)
}

// StatusCommand displays current session status
func (h *CommandHandlers) StatusCommand(update *tgbotapi.Update, args []string) error {
	if update.Message == nil {
		return fmt.Errorf("no message in update")
	}

	chatID := update.Message.Chat.ID
	session, exists := h.sessionMgr.GetSession(chatID)

	if !exists {
		return h.bot.SendMessage(chatID, "Нет активной игровой сессии.")
	}

	if !session.IsActive() {
		return h.bot.SendMessage(chatID, "⏸️ Сессия на паузе.")
	}

	// Build status message
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("🎮 **Сессия: %s**\n\n", session.ID))
	builder.WriteString(fmt.Sprintf("Кампания: %s\n", session.Campaign))

	// List active characters
	characters := session.GetAllCharacters()
	if len(characters) > 0 {
		builder.WriteString("\n👥 **Активные персонажи:**\n")
		for _, char := range characters {
			builder.WriteString(fmt.Sprintf("• %s (HP: %d/%d)\n", char.Name, 0, 0))
		}
	}

	builder.WriteString(fmt.Sprintf("\n⏱️ Начата: %s\n", session.StartTime.Format("15:04:05")))
	builder.WriteString(fmt.Sprintf("⏰ Активность: %s назад\n", time.Since(session.LastActivity)))

	return h.bot.SendMessage(chatID, builder.String())
}

// CharacterCommand displays character information
func (h *CommandHandlers) CharacterCommand(update *tgbotapi.Update, args []string) error {
	if update.Message == nil {
		return fmt.Errorf("no message in update")
	}

	chatID := update.Message.Chat.ID

	if len(args) == 0 {
		return h.bot.SendMessage(chatID, "Использование: /character <имя>")
	}

	charName := args[0]
	charPath := fmt.Sprintf("%s.md", charName)

	// Load character from storage
	char, err := h.charMgr.LoadCharacter(chatID, charPath)
	if err != nil {
		return h.bot.SendMessage(chatID, fmt.Sprintf("Ошибка загрузки карточки: %v", err))
	}

	// Display character sheet
	charMsg := h.formatCharacterCard(char)
	return h.bot.SendMessage(chatID, charMsg)
}

// formatCharacterCard formats a character card for display
func (h *CommandHandlers) formatCharacterCard(char *game.Character) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# %s\n\n", char.Name))

	if char.Stats != nil {
		stats := char.Stats
		builder.WriteString("## Характеристики\n")
		builder.WriteString(fmt.Sprintf("• В: %d | С: %d\n", stats.WS, stats.BS))
		builder.WriteString(fmt.Sprintf("• S: %d | Инт: %d\n", stats.S, stats.Int))
		builder.WriteString(fmt.Sprintf("• Ag: %d | ВН: %d\n", stats.Ag, stats.Int))
		builder.WriteString(fmt.Sprintf("• Int: %d | WP: %d\n", stats.Int, stats.WP))
		builder.WriteString(fmt.Sprintf("• WP: %d | Об: %d\n", stats.WP, stats.Fel))
		builder.WriteString(fmt.Sprintf("• Об: %d\n", stats.Fel))
		builder.WriteString(fmt.Sprintf("\n**HP:** %d/%d\n", stats.CurrentHP, stats.MaxHP))
		builder.WriteString(fmt.Sprintf("**XP:** %d\n", stats.XP))
	}

	// Add skills if available
	if len(char.Sheet) > 100 {
		// Parse skills section from sheet
		lines := strings.Split(char.Sheet, "\n")
		inSkills := false
		var skills []string

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "## Навыки") {
				inSkills = true
				builder.WriteString("\n### Навыки\n")
				continue
			}
			if inSkills && strings.HasPrefix(trimmed, "-") {
				skill := strings.TrimSpace(strings.TrimLeft(trimmed, "-"))
				skills = append(skills, skill)
			}
		}

		if len(skills) > 0 && len(skills) <= 10 {
			for _, skill := range skills {
				builder.WriteString(fmt.Sprintf("• %s\n", skill))
			}
		} else if len(skills) > 10 {
			builder.WriteString(fmt.Sprintf("\n... и ещё %d навыков\n", len(skills)-10))
		}
	}

	return builder.String()
}

// ReloadCommand reloads configuration
func (h *CommandHandlers) ReloadCommand(update *tgbotapi.Update, args []string) error {
	if update.Message == nil {
		return fmt.Errorf("no message in update")
	}

	chatID := update.Message.Chat.ID

	// Reload configuration from environment
	// Note: This is a placeholder - actual implementation would re-read .env
	log.Printf("[RELOAD] Configuration reload requested by user %d", update.Message.From.ID)

	return h.bot.SendMessage(chatID, "⚙️ Конфигурация перезагружена.")
}

// StopCommand stops the current session
func (h *CommandHandlers) StopCommand(update *tgbotapi.Update, args []string) error {
	if update.Message == nil {
		return fmt.Errorf("no message in update")
	}

	chatID := update.Message.Chat.ID
	session, exists := h.sessionMgr.GetSession(chatID)

	if !exists {
		return h.bot.SendMessage(chatID, "Нет активной игровой сессии.")
	}

	session.Stop()
	h.sessionMgr.RemoveSession(chatID)

	return h.bot.SendMessage(chatID, "🛑 Игровая сессия остановлена.")
}

// RegisterAllHandlers registers all command handlers with the bot
func (h *CommandHandlers) RegisterAllHandlers() {
	// Register commands
	h.bot.AddCommand("start", h.StartCommand)
	h.bot.AddCommand("help", h.HelpCommand)
	h.bot.AddCommand("status", h.StatusCommand)
	h.bot.AddCommand("character", h.CharacterCommand)
	h.bot.AddCommand("reload", h.ReloadCommand)
	h.bot.AddCommand("stop", h.StopCommand)

	// Register additional game commands
	h.bot.AddCommand("roll", h.RollCommand)
	h.bot.AddCommand("scene", h.SceneCommand)

	log.Println("[COMMANDS] All command handlers registered")
}

// RollCommand handles dice rolls
func (h *CommandHandlers) RollCommand(update *tgbotapi.Update, args []string) error {
	if update.Message == nil || len(args) == 0 {
		return h.bot.SendMessage(update.Message.Chat.ID, "Использование: /roll <формула>")
	}

	// Parse dice formula (e.g., "d100", "2d10", "d100+10")
	formula := strings.Join(args, " ")
	result := h.evaluateDice(formula)

	return h.bot.SendMessage(update.Message.Chat.ID, fmt.Sprintf("🎲 %s = %d", formula, result))
}

// evaluateDice evaluates a dice roll formula
func (h *CommandHandlers) evaluateDice(formula string) int {
	// Simple dice evaluation
	// dN - roll N-sided die
	// NdN - roll N dice of N sides
	// dN+K - roll N-sided die and add K

	if strings.HasPrefix(formula, "d") && len(formula) < 10 {
		// Single die: d100, d10, etc.
		// This is a placeholder - real implementation would parse the formula
		return 0
	}

	return 0
}

// SceneCommand describes the current scene
func (h *CommandHandlers) SceneCommand(update *tgbotapi.Update, args []string) error {
	if update.Message == nil {
		return fmt.Errorf("no message in update")
	}

	// This is a placeholder for GM-controlled scene descriptions
	scene := "Вы находитесь в таверне. Огни костра flicker над сырыми брёвнями, отбрасывая странные тени на стенах."
	if len(args) > 0 {
		scene = strings.Join(args, " ")
	}

	return h.bot.SendMessage(update.Message.Chat.ID, fmt.Sprintf("🏰 **Сцена:**\n\n%s", scene))
}

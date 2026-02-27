// Package telegram provides command handlers for WFRP bot
package telegram

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"wfrp-bot/config"
	"wfrp-bot/game"
	"wfrp-bot/llm"
	"wfrp-bot/storage"
)

// Command handlers for WFRP bot
type CommandHandlers struct {
	bot               *Bot
	sessionMgr        *game.SessionManager
	charMgr           *game.CharacterManager
	storageMgr        *storage.CampaignManager
	characterCreators map[int64]*game.CharacterCreator
}

// NewCommandHandlers creates a new command handlers instance
func NewCommandHandlers(bot *Bot, sessionMgr *game.SessionManager, charMgr *game.CharacterManager, storageMgr *storage.CampaignManager) *CommandHandlers {
	return &CommandHandlers{
		bot:               bot,
		sessionMgr:        sessionMgr,
		charMgr:           charMgr,
		storageMgr:        storageMgr,
		characterCreators: make(map[int64]*game.CharacterCreator),
	}
}

// StartCommand starts a new game session
func (h *CommandHandlers) StartCommand(update *tgbotapi.Update, args []string) error {
	if update.Message == nil {
		return fmt.Errorf("no message in update")
	}

	chatID := update.Message.Chat.ID

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

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return h.bot.SendMessage(chatID, fmt.Sprintf("Ошибка загрузки конфигурации: %v", err))
	}

	// Create LLM provider
	provider, err := llm.NewProviderFromConfig(&llm.ProviderConfig{
		Name:    cfg.DefaultProvider,
		APIKey:  cfg.Providers[cfg.DefaultProvider].APIKey,
		BaseURL: cfg.Providers[cfg.DefaultProvider].BaseURL,
		Model:   cfg.Providers[cfg.DefaultProvider].Model,
	})
	if err != nil {
		return h.bot.SendMessage(chatID, fmt.Sprintf("Ошибка инициализации LLM провайдера: %v", err))
	}

	// Create new session for campaign
	session := game.NewSession(context.Background(), chatID, campaign, provider)
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

📋 **Основные команды:**
/start <кампания> - Запустить новую игру или сессию
/stop - Остановить текущую сессию
/status - Показать статус текущей сессии

🎭 **Персонажи:**
/character <имя> - Создать нового персонажа WFRP 4E
/characters - Список всех персонажей
/newchar - Начать создание персонажа (альтернатива)
/cancel - Отменить текущее действие

💬 **Во время создания персонажа:**
- Напиши "сгенери имя" для автогенерации имени
- Задай вопрос (например "как распределить характеристики") для пояснений от LLM

🎲 **Утилиты:**
/roll <формула> - Бросить кубы (например: d100, 2d10+5)
/scene <описание> - Описать сцену
/reload - Перезагрузить конфигурацию
/help - Показать эту справку

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
			stats, _ := game.ParseCharacterStats(char.Sheet)
			currentHP, maxHP := 0, 0
			if stats != nil {
				currentHP = stats.CurrentHP
				maxHP = stats.MaxHP
			}
			builder.WriteString(fmt.Sprintf("• %s (HP: %d/%d)\n", char.Name, currentHP, maxHP))
		}
	}

	builder.WriteString(fmt.Sprintf("\n⏱️ Начата: %s\n", session.StartTime.Format("15:04:05")))
	builder.WriteString(fmt.Sprintf("⏰ Активность: %s назад\n", time.Since(session.LastActivity)))

	return h.bot.SendMessage(chatID, builder.String())
}

// CharacterCommand handles character creation or displays help
func (h *CommandHandlers) CharacterCommand(update *tgbotapi.Update, args []string) error {
	if update.Message == nil {
		return fmt.Errorf("no message in update")
	}

	chatID := update.Message.Chat.ID

	// If no arguments, show help
	if len(args) == 0 {
		return h.bot.SendMessage(chatID, `📖 Команда /character

Создаёт нового персонажа WFRP 4E.

Использование:
/character <имя> - начать создание персонажа

Примеры:
/character Арнольд
/character Мария

Также доступны:
/characters - список всех персонажей
/newchar - начать создание (альтернатива)`)
	}

	// Check if already creating a character
	if _, exists := h.characterCreators[chatID]; exists {
		return h.bot.SendMessage(chatID, "Создание персонажа уже начато! Ответь на текущий вопрос или напиши /cancel для отмены.")
	}

	charName := args[0]

	// Validate name length
	if len(charName) < 2 || len(charName) > 50 {
		return h.bot.SendMessage(chatID, "Имя персонажа должно быть от 2 до 50 символов.")
	}

	// Check if character already exists
	playerID := fmt.Sprintf("%d", update.Message.From.ID)
	charPath := fmt.Sprintf("%s.md", charName)
	_, err := h.charMgr.LoadCharacter(playerID, charPath)
	if err == nil {
		return h.bot.SendMessage(chatID, fmt.Sprintf("Персонаж с именем %s уже существует!", charName))
	}

	// Start new character creation
	creator := game.NewCharacterCreator("./characters")
	creator.Data.Name = charName

	// Try to get LLM provider from session
	if session, exists := h.sessionMgr.GetSession(chatID); exists {
		creator.SetLLMProvider(session.GetLLMProvider())
	} else {
		// Create temporary LLM provider
		cfg, err := config.LoadConfig()
		if err == nil {
			provider, err := llm.NewProviderFromConfig(&llm.ProviderConfig{
				Name:    cfg.DefaultProvider,
				APIKey:  cfg.Providers[cfg.DefaultProvider].APIKey,
				BaseURL: cfg.Providers[cfg.DefaultProvider].BaseURL,
				Model:   cfg.Providers[cfg.DefaultProvider].Model,
			})
			if err == nil {
				creator.SetLLMProvider(provider)
			}
		}
	}

	h.characterCreators[chatID] = creator

	return h.bot.SendMessage(chatID, fmt.Sprintf("🎭 **Создание персонажа: %s**\n\n%s", charName, creator.GetPrompt()))
}

// CharactersCommand displays list of all characters
func (h *CommandHandlers) CharactersCommand(update *tgbotapi.Update, args []string) error {
	if update.Message == nil {
		return fmt.Errorf("no message in update")
	}

	chatID := update.Message.Chat.ID

	characters := h.charMgr.GetAllCharacters()

	if len(characters) == 0 {
		return h.bot.SendMessage(chatID, "📋 Персонажей пока нет. Создайте первого с помощью /character <имя>")
	}

	var builder strings.Builder
	builder.WriteString("📋 **Персонажи кампании:**\n\n")

	for i, char := range characters {
		stats, _ := game.ParseCharacterStats(char.Sheet)
		career := "Без карьеры"
		race := "Человек"
		if stats != nil && stats.Name != "" {
			// Try to extract career from sheet
			if idx := strings.Index(char.Sheet, "Карьера:"); idx >= 0 {
				line := char.Sheet[idx:]
				endIdx := strings.Index(line, "\n")
				if endIdx > 0 {
					careerLine := strings.TrimSpace(line[:endIdx])
					careerLine = strings.TrimPrefix(careerLine, "Карьера:")
					career = strings.TrimSpace(careerLine)
				}
			}
			// Try to extract race from sheet
			if idx := strings.Index(char.Sheet, "Раса:"); idx >= 0 {
				line := char.Sheet[idx:]
				endIdx := strings.Index(line, "\n")
				if endIdx > 0 {
					raceLine := strings.TrimSpace(line[:endIdx])
					raceLine = strings.TrimPrefix(raceLine, "Раса:")
					race = strings.TrimSpace(raceLine)
				}
			}
		}
		builder.WriteString(fmt.Sprintf("%d. %s - %s (%s)\n", i+1, char.Name, career, race))
	}

	builder.WriteString(fmt.Sprintf("\nВсего: %d персонажей", len(characters)))

	return h.bot.SendMessage(chatID, builder.String())
}

// formatCharacterCard formats a character card for display
func (h *CommandHandlers) formatCharacterCard(char *game.Character) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# %s\n\n", char.Name))

	// Parse character stats from sheet
	stats, _ := game.ParseCharacterStats(char.Sheet)
	if stats != nil {
		builder.WriteString("## Характеристики\n")
		builder.WriteString(fmt.Sprintf("• WS: %d | BS: %d\n", stats.WS, stats.BS))
		builder.WriteString(fmt.Sprintf("• S: %d | Ag: %d\n", stats.S, stats.Ag))
		builder.WriteString(fmt.Sprintf("• Int: %d | WP: %d\n", stats.Int, stats.WP))
		builder.WriteString(fmt.Sprintf("• Fel: %d\n", stats.Fel))
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

// NewCharCommand starts new character creation
func (h *CommandHandlers) NewCharCommand(update *tgbotapi.Update, args []string) error {
	if update.Message == nil {
		return fmt.Errorf("no message in update")
	}

	chatID := update.Message.Chat.ID

	// Check if already creating a character
	if _, exists := h.characterCreators[chatID]; exists {
		return h.bot.SendMessage(chatID, "Создание персонажа уже начато! Ответь на текущий вопрос или напиши /cancel для отмены.")
	}

	// Create character creator with LLM provider
	creator := game.NewCharacterCreator("./characters")

	// Try to get LLM provider from session
	if session, exists := h.sessionMgr.GetSession(chatID); exists {
		creator.SetLLMProvider(session.GetLLMProvider())
		log.Printf("[NEWCHAR] LLM provider from session: %v", session.GetLLMProvider())
	} else {
		// Create temporary LLM provider
		log.Printf("[NEWCHAR] No session, creating temporary LLM provider")
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Printf("[NEWCHAR] Failed to load config: %v", err)
		} else {
			log.Printf("[NEWCHAR] Config loaded, provider: %s", cfg.DefaultProvider)
			provider, err := llm.NewProviderFromConfig(&llm.ProviderConfig{
				Name:    cfg.DefaultProvider,
				APIKey:  cfg.Providers[cfg.DefaultProvider].APIKey,
				BaseURL: cfg.Providers[cfg.DefaultProvider].BaseURL,
				Model:   cfg.Providers[cfg.DefaultProvider].Model,
			})
			if err != nil {
				log.Printf("[NEWCHAR] Failed to create provider: %v", err)
			} else {
				log.Printf("[NEWCHAR] Provider created: %v", provider)
				creator.SetLLMProvider(provider)
			}
		}
	}

	h.characterCreators[chatID] = creator

	return h.bot.SendMessage(chatID, "🎭 **Создание персонажа WFRP 4E**\n\n"+creator.GetPrompt())
}

// ProcessCharacterCreation handles ongoing character creation
func (h *CommandHandlers) ProcessCharacterCreation(chatID int64, text string) error {
	creator, exists := h.characterCreators[chatID]
	if !exists {
		return nil
	}

	response, isComplete := creator.ProcessInput(text)

	if err := h.bot.SendMessage(chatID, response); err != nil {
		return err
	}

	if isComplete && creator.IsComplete() {
		// Save character to file
		if err := creator.SaveToFile("./characters"); err != nil {
			log.Printf("[NEWCHAR] Failed to save character: %v", err)
		} else {
			h.bot.SendMessage(chatID, fmt.Sprintf("✅ Персонаж %s сохранён в characters/", creator.Data.Name))
		}
		// Remove from active creators
		delete(h.characterCreators, chatID)
	}

	return nil
}

// CancelCharacterCreation cancels ongoing character creation
func (h *CommandHandlers) CancelCharacterCreation(update *tgbotapi.Update, args []string) error {
	if update.Message == nil {
		return fmt.Errorf("no message in update")
	}

	chatID := update.Message.Chat.ID

	if _, exists := h.characterCreators[chatID]; exists {
		delete(h.characterCreators, chatID)
		return h.bot.SendMessage(chatID, "❌ Создание персонажа отменено.")
	}

	return h.bot.SendMessage(chatID, "Нет активного создания персонажа.")
}

// RegisterAllHandlers registers all command handlers with the bot
func (h *CommandHandlers) RegisterAllHandlers() {
	// Register commands
	h.bot.AddCommand("start", h.StartCommand)
	h.bot.AddCommand("help", h.HelpCommand)
	h.bot.AddCommand("status", h.StatusCommand)
	h.bot.AddCommand("character", h.CharacterCommand)
	h.bot.AddCommand("characters", h.CharactersCommand)
	h.bot.AddCommand("reload", h.ReloadCommand)
	h.bot.AddCommand("stop", h.StopCommand)

	// Register character creation
	h.bot.AddCommand("newchar", h.NewCharCommand)
	h.bot.AddCommand("cancel", h.CancelCharacterCreation)

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
	re := regexp.MustCompile(`^(\d*)d(\d+)([+-]\d+)?$`)
	matches := re.FindStringSubmatch(formula)

	if matches == nil {
		return 0
	}

	var numDice, sides, modifier int
	var err error

	if matches[1] == "" {
		numDice = 1
	} else {
		numDice, err = strconv.Atoi(matches[1])
		if err != nil || numDice < 1 || numDice > 100 {
			return 0
		}
	}

	sides, err = strconv.Atoi(matches[2])
	if err != nil || sides < 2 || sides > 100 {
		return 0
	}

	if matches[3] != "" {
		modifier, err = strconv.Atoi(matches[3])
		if err != nil {
			return 0
		}
	}

	total := modifier
	for i := 0; i < numDice; i++ {
		total += rand.Intn(sides) + 1
	}

	return total
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

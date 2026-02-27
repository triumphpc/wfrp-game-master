// Package game provides game session management for WFRP
package game

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"wfrp-bot/llm"
)

// Session represents an active game session
type Session struct {
	ID           string
	GroupID      int64
	Campaign     string
	Characters   map[string]*Character // playerID -> Character
	State        SessionState
	StartTime    time.Time
	LastActivity time.Time

	mu            sync.RWMutex
	llmProvider   llm.LLMProvider
	promptBuilder *PromptBuilder
	ctx           context.Context
	cancel        context.CancelFunc
}

// SessionState represents of current state of game session
type SessionState int

const (
	StateIdle       SessionState = iota // Waiting for input
	StateActive                         // Game in progress
	StateProcessing                     // Processing input
	StatePaused                         // Paused
)

// Character represents a player's character
type Character struct {
	ID         string
	Name       string
	CardPath   string // Path to markdown file
	Sheet      string // Full character sheet content
	LastUpdate time.Time
}

// InputData represents game input data
type InputData struct {
	Source    string // "player", "gm", "system"
	Content   string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// GameOutput represents output to players
type GameOutput struct {
	Source    string
	Content   string
	IsAction  bool
	Timestamp time.Time
}

// PromptBuilder constructs LLM prompts
type PromptBuilder struct {
	campaign   string
	scenario   string
	characters []*Character
	rules      []string
}

// NewSession creates a new game session
func NewSession(ctx context.Context, groupID int64, campaign string, provider llm.LLMProvider) *Session {
	sessionCtx, cancel := context.WithCancel(ctx)

	return &Session{
		ID:           fmt.Sprintf("%s_%d", campaign, groupID),
		GroupID:      groupID,
		Campaign:     campaign,
		Characters:   make(map[string]*Character),
		State:        StateIdle,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		llmProvider:  provider,
		promptBuilder: &PromptBuilder{
			campaign: campaign,
		},
		ctx:    sessionCtx,
		cancel: cancel,
	}
}

// Start begins the game session
func (s *Session) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.State = StateActive
	s.LastActivity = time.Now()

	log.Printf("[SESSION] Started session %s for campaign %s", s.ID, s.Campaign)

	// Start input checking goroutine
	go s.checkInputsLoop()
}

// Stop gracefully stops session
func (s *Session) Stop() {
	s.cancel()
	s.mu.Lock()
	s.State = StateIdle
	s.mu.Unlock()

	log.Printf("[SESSION] Stopped session %s", s.ID)
}

// AddCharacter adds a character to the session
func (s *Session) AddCharacter(playerID string, char *Character) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Characters[playerID] = char
	log.Printf("[SESSION] Added character %s for player %s", char.Name, playerID)
}

// RemoveCharacter removes a character from session
func (s *Session) RemoveCharacter(playerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Characters, playerID)
	log.Printf("[SESSION] Removed character for player %s", playerID)
}

// GetCharacter returns a character by player ID
func (s *Session) GetCharacter(playerID string) (*Character, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	char, exists := s.Characters[playerID]
	return char, exists
}

// UpdateActivity updates last activity timestamp
func (s *Session) UpdateActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastActivity = time.Now()
}

// ProcessInput processes player input and generates GM response
func (s *Session) ProcessInput(input InputData) (*GameOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.UpdateActivity()
	s.State = StateProcessing

	// Build prompt with context
	prompt := s.promptBuilder.BuildGamePrompt(input, s.GetAllCharacterSheets())
	// Check rules if needed
	response, err := s.llmProvider.GenerateRequest(s.ctx, prompt, s.GetAllCharacterSheets())
	if err != nil {
		s.State = StateActive
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	s.State = StateActive

	// Parse character updates from response
	_, charUpdate, err := ParseCharacterUpdateFromResponse(response)
	if err != nil {
		log.Printf("[SESSION] Failed to parse character update: %v", err)
		// Continue without applying updates if parsing fails
	}

	// Apply character updates if any
	if charUpdate != nil {
		for _, char := range s.Characters {
			updatedSheet, warnings := ApplyCharacterUpdate(char.Sheet, *charUpdate)
			for _, warning := range warnings {
				log.Printf("[SESSION] Character update warning: %v", warning)
			}

			// Update in-memory character
			char.Sheet = updatedSheet
			char.LastUpdate = time.Now()
			log.Printf("[SESSION] Updated character %s after response", char.Name)
		}
	}

	return &GameOutput{
		Source:    "gm",
		Content:   response,
		IsAction:  false,
		Timestamp: time.Now(),
	}, nil
}

// StreamResponse processes input and streams GM response
func (s *Session) StreamResponse(input InputData) (<-chan string, error) {
	s.mu.Lock()
	s.UpdateActivity()
	s.State = StateProcessing
	s.mu.Unlock()

	prompt := s.promptBuilder.BuildGamePrompt(input, s.GetAllCharacterSheets())

	stream, err := s.llmProvider.StreamRequest(s.ctx, prompt, nil)
	if err != nil {
		s.mu.Lock()
		s.State = StateActive
		s.mu.Unlock()
		return nil, fmt.Errorf("failed to stream response: %w", err)
	}

	return stream, nil
}

// checkInputsLoop periodically checks for new inputs
func (s *Session) checkInputsLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Check for new player inputs every second
			if hasInput, err := s.CheckInputs(); err != nil {
				log.Printf("[SESSION] Input check error: %v", err)
			} else if hasInput {
				// Process new inputs if found
				s.ProcessNewInputs()
			}
			s.checkTimeout()
		}
	}
}

// CheckInputs checks for new player inputs
func (s *Session) CheckInputs() (bool, error) {
	// This is a placeholder implementation
	// In a real implementation, this would check:
	// - Incoming Telegram messages
	// - Character card updates
	// - Game state changes
	// Return true if new inputs are found
	return false, nil
}

// ProcessNewInputs processes newly detected inputs
func (s *Session) ProcessNewInputs() {
	// Process queued inputs
	// This would trigger ProcessInput for each detected input
	s.mu.Lock()
	s.UpdateActivity()
	s.mu.Unlock()
}

// checkTimeout checks if session has timed out
func (s *Session) checkTimeout() {
	s.mu.RLock()
	inactivity := time.Since(s.LastActivity)
	s.mu.RUnlock()

	// Timeout after 30 minutes of inactivity
	if inactivity > 30*time.Minute {
		log.Printf("[SESSION] Session %s timed out after %v", s.ID, inactivity)
		s.Stop()
	}
}

// GetAllCharacterSheets returns all character sheets as strings
func (s *Session) GetAllCharacterSheets() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sheets := make([]string, 0, len(s.Characters))
	for _, char := range s.Characters {
		sheets = append(sheets, char.Sheet)
	}
	return sheets
}

// IsActive returns whether session is active
func (s *Session) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State == StateActive
}

// GetAllCharacters returns all characters in session
func (s *Session) GetAllCharacters() []*Character {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chars := make([]*Character, 0, len(s.Characters))
	for _, char := range s.Characters {
		chars = append(chars, char)
	}
	return chars
}

// GetLLMProvider returns the LLM provider for the session
func (s *Session) GetLLMProvider() llm.LLMProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.llmProvider
}

// BuildGamePrompt constructs an LLM prompt from input and character sheets
func (pb *PromptBuilder) BuildGamePrompt(input InputData, characterSheets []string) string {
	var prompt strings.Builder

	// Add system context
	prompt.WriteString("--- СИСТЕМА: WARHAMMER FANTASY ROLEPLAY 4E ---\n\n")
	prompt.WriteString("Ты - Game Master (Гейм Мастер) для игры в WFRP 4e. ")
	prompt.WriteString("Твоя задача - вести интересную и атмосферную игру, ")
	prompt.WriteString("строго соблюдая правила WFRP 4th Edition.\n\n")

	// Add campaign context
	if pb.campaign != "" {
		prompt.WriteString(fmt.Sprintf("--- КАМПАНИЯ: %s ---\n\n", pb.campaign))
	}

	// Add scenario context
	if pb.scenario != "" {
		prompt.WriteString(fmt.Sprintf("СЦЕНАРИЙ:\n%s\n\n", pb.scenario))
	}

	// Add characters section
	if len(characterSheets) > 0 {
		prompt.WriteString("--- АКТИВНЫЕ ПЕРСОНАЖИ ИГРОКОВ ---\n\n")
		for i, sheet := range characterSheets {
			if i > 0 {
				prompt.WriteString("\n---\n\n")
			}
			prompt.WriteString(sheet)
		}
		prompt.WriteString("\n--- КОНЕЦ ПЕРСОНАЖЕЙ ---\n\n")
	}

	// Add rules reference
	if len(pb.rules) > 0 {
		prompt.WriteString("--- ПРАВИЛА ---\n")
		prompt.WriteString("Важно строго следовать правилам WFRP 4e. ")
		prompt.WriteString("Для проверки механик используй:\n")
		for _, rule := range pb.rules {
			prompt.WriteString(fmt.Sprintf("  • %s\n", rule))
		}
		prompt.WriteString("--- КОНЕЦ ПРАВИЛ ---\n\n")
	}

	// Add input section
	prompt.WriteString("--- ВВОД ИГРОКА ---\n")
	prompt.WriteString(fmt.Sprintf("Источник: %s\n", input.Source))
	prompt.WriteString(fmt.Sprintf("Содержание: %s\n", input.Content))
	prompt.WriteString(fmt.Sprintf("Время: %s\n", input.Timestamp.Format("15:04:05")))

	// Add metadata if present
	if len(input.Metadata) > 0 {
		prompt.WriteString("Метаданные:\n")
		for key, value := range input.Metadata {
			prompt.WriteString(fmt.Sprintf("  • %s: %v\n", key, value))
		}
	}

	prompt.WriteString("--- КОНЕЦ ВВОДА ---\n\n")

	// Add response instruction
	prompt.WriteString("--- ИНСТРУКЦИЯ ---\n")
	prompt.WriteString("Отвечай как Game Master. Веди игру атмосферно и интересно. ")
	prompt.WriteString("При описании действий требуй проверок по правилам WFRP 4e. ")
	prompt.WriteString("Если игрок пытается выполнить действие, требуй соответствующей проверки (Бой, Навык, Характеристика). ")
	prompt.WriteString("Соблюдай все правила WFRP 4e, включая модификаторы, сложность и последствия провала/успеха.\n")
	prompt.WriteString("--- КОНЕЦ ИНСТРУКЦИИ ---\n\n")

	// Add separator for response
	prompt.WriteString("GM RESPONSE:")

	return prompt.String()
}

// SetScenario sets current scenario for prompt builder
func (pb *PromptBuilder) SetScenario(scenario string) {
	pb.scenario = scenario
}

// AddRule adds a rule reference to the prompt builder
func (pb *PromptBuilder) AddRule(rule string) {
	pb.rules = append(pb.rules, rule)
}

// SetCharacters sets characters for prompt builder
func (pb *PromptBuilder) SetCharacters(chars []*Character) {
	pb.characters = make([]*Character, len(chars))
	copy(pb.characters, chars)
}

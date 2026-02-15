// Package game provides game session management for WFRP
package game

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"wfrp-bot/llm"
	"wfrp-bot/telegram"
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

	mu             sync.RWMutex
	llmProvider    llm.LLMProvider
	promptBuilder  *PromptBuilder
	ruleChecker    *RuleChecker
	ctx            context.Context
	cancel         context.CancelFunc
}

// SessionState represents the current state of the game session
type SessionState int

const (
	StateIdle      SessionState = iota // Waiting for input
	StateActive                       // Game in progress
	StateProcessing                  // Processing input
	StatePaused                      // Paused
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
	Source     string // "player", "gm", "system"
	Content    string
	Timestamp  time.Time
	Metadata   map[string]interface{}
}

// GameOutput represents output to players
type GameOutput struct {
	Source     string
	Content    string
	IsAction   bool
	Timestamp  time.Time
}

// PromptBuilder constructs LLM prompts
type PromptBuilder struct {
	campaign    string
	scenario    string
	characters  []*Character
	rules       []string
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
		ruleChecker:   NewRuleChecker(),
		ctx:           sessionCtx,
		cancel:        cancel,
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

// Stop gracefully stops the session
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

// RemoveCharacter removes a character from the session
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

// UpdateActivity updates the last activity timestamp
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
	if ruleViolations, err := s.ruleChecker.Check(input); err != nil {
		log.Printf("[SESSION] Rule check error: %v", err)
	} else if len(ruleViolations) > 0 {
		log.Printf("[SESSION] Rule violations: %v", ruleViolations)
		// Could add warnings to prompt
	}

	// Generate response from LLM
	response, err := s.llmProvider.GenerateRequest(s.ctx, prompt, nil)
	if err != nil {
		s.State = StateActive
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	s.State = StateActive

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

// IsActive returns whether the session is active
func (s *Session) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State == StateActive
}

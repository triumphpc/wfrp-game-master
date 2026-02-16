package game

import (
	"fmt"
	"log"
	"sync"
)

// SessionManager manages multiple game sessions
type SessionManager struct {
	sessions map[int64]*Session
	mu       sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[int64]*Session),
	}
}

// AddSession adds a session to the manager
func (sm *SessionManager) AddSession(chatID int64, session *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sessions[chatID] = session
	log.Printf("[SESSION MANAGER] Added session for chat %d", chatID)
}

// GetSession retrieves a session by chat ID
func (sm *SessionManager) GetSession(chatID int64) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[chatID]
	return session, exists
}

// RemoveSession removes a session from the manager
func (sm *SessionManager) RemoveSession(chatID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, chatID)
	log.Printf("[SESSION MANAGER] Removed session for chat %d", chatID)
}

// GetAllSessions returns all active sessions
func (sm *SessionManager) GetAllSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// ProcessPlayerMessage processes a player message through the appropriate session
func (sm *SessionManager) ProcessPlayerMessage(chatID int64, playerID string, text string) (*GameOutput, error) {
	session, exists := sm.GetSession(chatID)
	if !exists {
		return nil, fmt.Errorf("no active session for chat %d", chatID)
	}

	if !session.IsActive() {
		return nil, fmt.Errorf("session %s is not active", session.ID)
	}

	// Create input data from player message
	input := InputData{
		Source:    "player",
		Content:   text,
		Timestamp: session.LastActivity,
		Metadata: map[string]interface{}{
			"player_id": playerID,
		},
	}

	// Process input and get GM response
	output, err := session.ProcessInput(input)
	if err != nil {
		log.Printf("[SESSION MANAGER] Failed to process input: %v", err)
		return nil, err
	}

	return output, nil
}

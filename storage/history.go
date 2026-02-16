// Package storage provides session history management for WFRP bot
package storage

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// HistoryManager manages session history storage
type HistoryManager struct {
	basePath  string
	parser    *MarkdownParser
	sessions   map[string]*SessionRecord
	mu         sync.RWMutex
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(basePath string) *HistoryManager {
	return &HistoryManager{
		basePath: basePath,
		parser:    NewMarkdownParser(basePath),
		sessions:   make(map[string]*SessionRecord),
	}
}

// SessionRecord represents a saved session
type SessionRecord struct {
	ID          string
	Date        time.Time
	Title       string
	Summary    string
	Campaign    string
	Characters  []string
	Path        string
}

// CreateSession creates a new session record
func (hm *HistoryManager) CreateSession(campaign, title string) (*SessionRecord, error) {
	now := time.Now()

	// Generate session ID (YYYY-MM-DD_HH-MM)
	sessionID := now.Format("2006-01-02_15-04")
	if title != "" {
		sessionID += "_" + strings.Map(func(r rune) rune {
			for _, ch := range []string{":", "?", "*", "<", ">", "|", "\""} {
				if r == rune(ch[0]) {
					return '_'
				}
			}
			return r
		}, title)
	}

	// Create session filename
	filename := sessionID + ".md"
	path := filepath.Join(hm.basePath, campaign, filename)

	// Create session file with header
	session := &SessionRecord{
		ID:         sessionID,
		Date:       now,
		Title:      title,
		Campaign:   campaign,
		Characters: make([]string, 0),
		Path:       path,
	}

	// Write initial session file
	if err := hm.writeSessionFile(path, session); err != nil {
		return nil, err
	}

	// Add to manager
	hm.mu.Lock()
	hm.sessions[sessionID] = session
	hm.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID
func (hm *HistoryManager) GetSession(sessionID string) (*SessionRecord, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	session, exists := hm.sessions[sessionID]
	if !exists {
		// Try to load from disk
		return hm.loadSession(sessionID)
	}

	return session, nil
}

// ListSessions returns all sessions for a campaign
func (hm *HistoryManager) ListSessions(campaign string) ([]*SessionRecord, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var sessions []*SessionRecord

	for _, session := range hm.sessions {
		if session.Campaign == campaign {
			sessions = append(sessions, session)
		}
	}

	// Sort by date (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[j].Date.After(sessions[i].Date)
	})

	return sessions, nil
}

// AppendToSession adds content to an existing session
func (hm *HistoryManager) AppendToSession(sessionID, content string) error {
	// Resolve session path
	path, err := hm.resolveSessionPath(sessionID)
	if err != nil {
		return err
	}

	// Append content to file
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString("\n" + content); err != nil {
		return fmt.Errorf("failed to write to session file: %w", err)
	}

	return nil
}

// UpdateSessionSummary updates the summary of a session
func (hm *HistoryManager) UpdateSessionSummary(sessionID, summary string) error {
	// Load session
	session, err := hm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.Summary = summary

	// Write updated session
	return hm.writeSessionFile(session.Path, session)
}

// loadSession loads a session from disk
func (hm *HistoryManager) loadSession(sessionID string) (*SessionRecord, error) {
	path, err := hm.resolveSessionPath(sessionID)
	if err != nil {
		return nil, err
	}

	content, err := hm.parser.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse session file
	session := &SessionRecord{
		ID:   sessionID,
		Path: path,
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			session.Title = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		} else if strings.HasPrefix(trimmed, "Дата:") || strings.HasPrefix(trimmed, "Date:") {
			if date, err := time.Parse("2006-01-02 15:04", strings.TrimSpace(strings.Split(trimmed, ":")[1])); err == nil {
				session.Date = date
			}
		} else if strings.HasPrefix(trimmed, "Итог:") || strings.HasPrefix(trimmed, "Summary:") {
			session.Summary = strings.TrimSpace(strings.Split(trimmed, ":")[1])
		}
	}

	// Add to cache
	hm.mu.Lock()
	hm.sessions[sessionID] = session
	hm.mu.Unlock()

	return session, nil
}

// resolveSessionPath finds the full path to a session file
func (hm *HistoryManager) resolveSessionPath(sessionID string) (string, error) {
	// Try to find session file in base path
	entries, err := os.ReadDir(hm.basePath)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check inside campaign directories
			continue
		}

		if strings.HasPrefix(entry.Name(), sessionID) && strings.HasSuffix(entry.Name(), ".md") {
			return filepath.Join(hm.basePath, entry.Name())
		}
	}

	return "", fmt.Errorf("session not found: %s", sessionID)
}

// writeSessionFile writes a session record to file
func (hm *HistoryManager) writeSessionFile(path string, session *SessionRecord) error {
	var builder strings.Builder

	// Header
	builder.WriteString(fmt.Sprintf("# %s\n\n", session.Title))
	builder.WriteString(fmt.Sprintf("## Date: %s\n\n", session.Date.Format("2006-01-02 15:04")))

	// Summary
	if session.Summary != "" {
		builder.WriteString(fmt.Sprintf("## Summary\n%s\n\n", session.Summary))
	}

	// Participants
	if len(session.Characters) > 0 {
		builder.WriteString("## Participants\n")
		for _, char := range session.Characters {
			builder.WriteString(fmt.Sprintf("- %s\n", char))
		}
		builder.WriteString("\n")
	}

	// Log section
	builder.WriteString("## Log\n\n")

	// Write to file
	return hm.parser.WriteFile(path, builder.String())
}

// GetLatestSessions returns recent sessions across all campaigns
func (hm *HistoryManager) GetLatestSessions(limit int) ([]*SessionRecord, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var allSessions []*SessionRecord

	for _, session := range hm.sessions {
		allSessions = append(allSessions, session)
	}

	// Sort by date (newest first)
	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[j].Date.After(allSessions[i].Date)
	})

	// Limit results
	if limit > 0 && len(allSessions) > limit {
		allSessions = allSessions[:limit]
	}

	return allSessions, nil
}

// SearchSessions searches for sessions matching a query
func (hm *HistoryManager) SearchSessions(query string) ([]*SessionRecord, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var results []*SessionRecord

	queryLower := strings.ToLower(query)

	for _, session := range hm.sessions {
		// Search in title and summary
		titleMatch := strings.Contains(strings.ToLower(session.Title), queryLower)
		summaryMatch := strings.Contains(strings.ToLower(session.Summary), queryLower)

		if titleMatch || summaryMatch {
			results = append(results, session)
		}
	}

	return results, nil
}

// DeleteSession removes a session record
func (hm *HistoryManager) DeleteSession(sessionID string) error {
	hm.mu.RLock()
	session, exists := hm.sessions[sessionID]
	hm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Delete file
	if err := os.Remove(session.Path); err != nil {
		return err
	}

	// Remove from cache
	hm.mu.Lock()
	delete(hm.sessions, sessionID)
	hm.mu.Unlock()

	return nil
}

// ArchiveSession moves an old session to archive
func (hm *HistoryManager) ArchiveSession(sessionID string) error {
	session, err := hm.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Create archive directory
	archiveDir := filepath.Join(hm.basePath, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return err
	}

	// Move session to archive
	archivePath := filepath.Join(archiveDir, filepath.Base(session.Path))
	if err := os.Rename(session.Path, archivePath); err != nil {
		return err
	}

	return nil
}

// IndexSessions rebuilds the session index
func (hm *HistoryManager) IndexSessions() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Scan all session files
	hm.sessions = make(map[string]*SessionRecord)

	entries, err := os.ReadDir(hm.basePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Skip directories, scan them recursively
			if err := hm.indexDirectory(filepath.Join(hm.basePath, entry.Name())); err != nil {
				log.Printf("[HISTORY] Failed to index directory %s: %v", entry.Name(), err)
			}
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Try to parse session ID from filename
		sessionID := strings.TrimSuffix(entry.Name(), ".md")
		if strings.Contains(sessionID, "_") {
			// Format: YYYY-MM-DD_HH-MM_description.md
			// Use full filename as ID
			sessionID = entry.Name()
		}

		path := filepath.Join(hm.basePath, entry.Name())
		session := &SessionRecord{
			ID:  sessionID,
			Path: path,
		}

		hm.sessions[sessionID] = session
	}

	return nil
}

// indexDirectory recursively indexes session files
func (hm *HistoryManager) indexDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if err := hm.indexDirectory(filepath.Join(dir, entry.Name())); err != nil {
				return err
			}
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		sessionID := strings.TrimSuffix(entry.Name(), ".md")

		session := &SessionRecord{
			ID:   sessionID,
			Path: path,
		}

		hm.sessions[sessionID] = session
	}

	return nil
}

// SessionFilter filters session queries
type SessionFilter struct {
	Campaign   string
	StartDate  *time.Time
	EndDate    *time.Time
	MinDate    *time.Time
	MaxDate    *time.Time
	Characters  []string
}

// FilterSessions applies filters to session list
func (hm *HistoryManager) FilterSessions(filter SessionFilter) ([]*SessionRecord, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var results []*SessionRecord

	for _, session := range hm.sessions {
		if !hm.matchesFilter(session, filter) {
			continue
		}
		results = append(results, session)
	}

	return results, nil
}

// matchesFilter checks if a session matches the filter criteria
func (hm *HistoryManager) matchesFilter(session *SessionRecord, filter SessionFilter) bool {
	// Filter by campaign
	if filter.Campaign != "" && session.Campaign != filter.Campaign {
		return false
	}

	// Filter by date range
	if filter.StartDate != nil && session.Date.Before(*filter.StartDate) {
		return false
	}

	if filter.EndDate != nil && session.Date.After(*filter.EndDate) {
		return false
	}

	if filter.MinDate != nil && session.Date.Before(*filter.MinDate) {
		return false
	}

	if filter.MaxDate != nil && session.Date.After(*filter.MaxDate) {
		return false
	}

	// Filter by characters
	if len(filter.Characters) > 0 {
		sessionChars := strings.Join(session.Characters, ",")
		for _, char := range filter.Characters {
			if !strings.Contains(sessionChars, char) {
				return false
			}
		}
	}

	return true
}

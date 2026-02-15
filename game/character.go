// Package game provides character card management for WFRP
package game

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// CharacterManager handles character card operations
type CharacterManager struct {
	campaignPath string
	characters  map[string]*Character
	mu          sync.RWMutex
}

// NewCharacterManager creates a new character manager
func NewCharacterManager(campaignPath string) *CharacterManager {
	return &CharacterManager{
		campaignPath: campaignPath,
		characters:  make(map[string]*Character),
	}
}

// LoadCharacter loads a character from markdown file
func (cm *CharacterManager) LoadCharacter(playerID, characterPath string) (*Character, error) {
	// Determine full path
	var fullPath string
	if filepath.IsAbs(characterPath) {
		fullPath = characterPath
	} else {
		fullPath = filepath.Join(cm.campaignPath, "characters", characterPath+".md")
	}

	// Read markdown file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read character file %s: %w", fullPath, err)
	}

	// Parse character card
	char := &Character{
		ID:         playerID,
		Name:       extractCharacterName(content),
		CardPath:   fullPath,
		Sheet:      string(content),
		LastUpdate: time.Now(),
	}

	log.Printf("[CHARACTER] Loaded character %s from %s", char.Name, fullPath)

	// Add to manager
	cm.mu.Lock()
	cm.characters[playerID] = char
	cm.mu.Unlock()

	return char, nil
}

// SaveCharacter updates a character card to file
func (cm *CharacterManager) SaveCharacter(playerID string, updates map[string]interface{}) error {
	cm.mu.RLock()
	char, exists := cm.characters[playerID]
	cm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("character not found for player %s", playerID)
	}

	// Apply updates to sheet
	updatedSheet := cm.applyUpdates(char.Sheet, updates)

	// Write to file
	if err := os.WriteFile(char.CardPath, []byte(updatedSheet), 0644); err != nil {
		return fmt.Errorf("failed to write character file: %w", err)
	}

	// Update in-memory character
	cm.mu.Lock()
	char.Sheet = updatedSheet
	char.LastUpdate = time.Now()
	cm.mu.Unlock()

	log.Printf("[CHARACTER] Saved character %s for player %s", char.Name, playerID)

	return nil
}

// GetCharacter returns a character by player ID
func (cm *CharacterManager) GetCharacter(playerID string) (*Character, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	char, exists := cm.characters[playerID]
	return char, exists
}

// GetAllCharacters returns all characters
func (cm *CharacterManager) GetAllCharacters() []*Character {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	chars := make([]*Character, 0, len(cm.characters))
	for _, char := range cm.characters {
		chars = append(chars, char)
	}
	return chars
}

// RemoveCharacter removes a character from manager
func (cm *CharacterManager) RemoveCharacter(playerID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	char, exists := cm.characters[playerID]
	if !exists {
		return fmt.Errorf("character not found for player %s", playerID)
	}

	// Delete file
	if err := os.Remove(char.CardPath); err != nil {
		return fmt.Errorf("failed to delete character file: %w", err)
	}

	delete(cm.characters, playerID)

	log.Printf("[CHARACTER] Removed character %s for player %s", char.Name, playerID)

	return nil
}

// UpdateCharacterStats updates character statistics after game actions
func (cm *CharacterManager) UpdateCharacterStats(playerID string, statChanges map[string]int) error {
	updates := make(map[string]interface{})
	for stat, change := range statChanges {
		updates[stat] = change
	}

	return cm.SaveCharacter(playerID, updates)
}

// ValidateCharacter checks if character card follows WFRP rules
func (cm *CharacterManager) ValidateCharacter(char *Character) []string {
	var violations []string

	sheet := char.Sheet

	// Check for required sections
	requiredSections := []string{"# Имя", "## Характеристики", "## Навыки"}
	for _, section := range requiredSections {
		if !strings.Contains(sheet, section) {
			violations = append(violations, fmt.Sprintf("Missing section: %s", section))
		}
	}

	// Check characteristic range (0-100 for most stats in WFRP)
	if strings.Contains(sheet, "## Характеристики") {
		// Basic validation - could be enhanced
		if !strings.Contains(sheet, "В") && !strings.Contains(sheet, "Лов") {
			violations = append(violations, "Characteristics section incomplete")
		}
	}

	return violations
}

// applyUpdates applies updates to character sheet
func (cm *CharacterManager) applyUpdates(sheet string, updates map[string]interface{}) string {
	// Simple implementation - replaces patterns in sheet
	// A more sophisticated version would parse markdown structure

	updated := sheet

	for key, value := range updates {
		switch v := value.(type) {
		case int:
			updated = strings.ReplaceAll(updated, key+": XX", key+": "+fmt.Sprint(v))
		case string:
			updated = strings.ReplaceAll(updated, key+": XX", key+": "+v)
		}
	}

	return updated
}

// extractCharacterName extracts character name from markdown content
func extractCharacterName(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# Имя:") || strings.HasPrefix(line, "# Имя ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "# Имя"))
			return strings.TrimSpace(name)
		}
	}
	// Default to filename
	return "Unknown"
}

// CharacterStats represents parsed character statistics
type CharacterStats struct {
	Name        string
	WS          int // Weapon Skill
	BS          int // Ballistic Skill
	S            int // Strength
	Ag          int // Agility
	Int         int // Intelligence
	WP          int // Will Power
	Fel         int // Fellowship
	CurrentHP    int
	MaxHP        int
	XP           int
	Experience   []string
}

// ParseCharacterStats parses character statistics from markdown
func ParseCharacterStats(sheet string) (*CharacterStats, error) {
	stats := &CharacterStats{
		Name:     extractCharacterName(sheet),
		CurrentHP: 0,
		MaxHP:     0,
		XP:        0,
	}

	lines := strings.Split(sheet, "\n")
	for _, line := range lines {
		// Parse characteristic lines like "В: 40"
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Try to parse as integer
				var intValue int
				if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
					switch key {
					case "В", "Weapon Skill":
						stats.WS = intValue
					case "BS", "Ballistic Skill":
						stats.BS = intValue
					case "S", "Strength":
						stats.S = intValue
					case "Ag", "Agility":
						stats.Ag = intValue
					case "Int", "Intelligence":
						stats.Int = intValue
					case "WP", "Will Power":
						stats.WP = intValue
					case "Fel", "Fellowship":
						stats.Fel = intValue
					}
				}
			}
		}

		// Parse HP
		if strings.Contains(line, "HP:") || strings.Contains(line, "Здоровье:") {
			if _, err := fmt.Sscanf(line, "*%*HP: %d", &stats.MaxHP); err == nil {
				stats.CurrentHP = stats.MaxHP
			}
		}

		// Parse XP
		if strings.Contains(line, "XP:") || strings.Contains(line, "Опыт:") {
			if _, err := fmt.Sscanf(line, "*%*XP: %d", &stats.XP); err == nil {
				// XP parsed
			}
		}
	}

	return stats, nil
}

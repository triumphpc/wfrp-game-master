// Package game provides character card management for WFRP
package game

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CharacterManager handles character card operations
type CharacterManager struct {
	campaignPath string
	characters   map[string]*Character
	mu           sync.RWMutex
}

// NewCharacterManager creates a new character manager
func NewCharacterManager(campaignPath string) *CharacterManager {
	return &CharacterManager{
		campaignPath: campaignPath,
		characters:   make(map[string]*Character),
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
		Name:       extractCharacterName(string(content)),
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
	Name       string
	WS         int // Weapon Skill
	BS         int // Ballistic Skill
	S          int // Strength
	Ag         int // Agility
	Int        int // Intelligence
	WP         int // Will Power
	Fel        int // Fellowship
	CurrentHP  int
	MaxHP      int
	XP         int
	Experience []string
}

// ParseCharacterStats parses character statistics from markdown
func ParseCharacterStats(sheet string) (*CharacterStats, error) {
	stats := &CharacterStats{
		Name:      extractCharacterName(sheet),
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

// CharacterUpdate represents changes to apply to a character
type CharacterUpdate struct {
	HPChange      int            // Damage or healing
	MaxHPChange   int            // Permanent HP change
	XPChange      int            // Experience gained
	StatsChanges  map[string]int // Statistic changes (WS, S, Ag, etc.)
	SkillsAdded   []string
	SkillsRemoved []string
	Conditions    []string // Conditions added/removed
}

// ApplyCharacterUpdate applies changes to a character sheet according to WFRP rules
func ApplyCharacterUpdate(sheet string, update CharacterUpdate) (string, []string) {
	var warnings []string
	updated := sheet

	// Parse current stats for validation
	stats, err := ParseCharacterStats(sheet)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Failed to parse stats: %v", err))
	}

	// Apply HP changes
	if update.HPChange != 0 {
		updated = applyHPChange(updated, update.HPChange, stats)
		if update.HPChange < 0 {
			warnings = append(warnings, fmt.Sprintf("Character took %d damage", -update.HPChange))
		} else {
			warnings = append(warnings, fmt.Sprintf("Character healed %d HP", update.HPChange))
		}
	}

	// Apply Max HP changes
	if update.MaxHPChange != 0 {
		updated = applyMaxHPChange(updated, update.MaxHPChange)
	}

	// Apply XP changes
	if update.XPChange != 0 {
		updated = applyXPChange(updated, update.XPChange)
		warnings = append(warnings, fmt.Sprintf("Character gained %d XP", update.XPChange))
	}

	// Apply statistic changes
	for stat, change := range update.StatsChanges {
		updated = applyStatChange(updated, stat, change)
		warnings = append(warnings, fmt.Sprintf("%s changed by %d", stat, change))
	}

	// Add skills
	for _, skill := range update.SkillsAdded {
		updated = addSkillToSheet(updated, skill)
		warnings = append(warnings, fmt.Sprintf("Added skill: %s", skill))
	}

	// Add conditions
	for _, cond := range update.Conditions {
		updated = addConditionToSheet(updated, cond)
		warnings = append(warnings, fmt.Sprintf("Condition added: %s", cond))
	}

	updated = fmt.Sprintf("%s\n\n*(Обновлено: %s)*", updated, time.Now().Format("15:04:05"))

	return updated, warnings
}

// applyHPChange applies HP damage or healing
func applyHPChange(sheet string, change int, stats *CharacterStats) string {
	if stats == nil {
		return sheet
	}

	// Find current HP line and update it
	newCurrentHP := stats.CurrentHP + change
	if newCurrentHP < 0 {
		newCurrentHP = 0
	} else if stats.MaxHP > 0 && newCurrentHP > stats.MaxHP {
		newCurrentHP = stats.MaxHP
	}

	// Replace HP line in sheet
	replacer := strings.NewReplacer(
		fmt.Sprintf("HP: %d", stats.CurrentHP),
		fmt.Sprintf("HP: %d", newCurrentHP),
		fmt.Sprintf("Здоровье: %d", stats.CurrentHP),
		fmt.Sprintf("Здоровье: %d", newCurrentHP),
	)

	return replacer.Replace(sheet)
}

// applyMaxHPChange applies permanent Max HP change
func applyMaxHPChange(sheet string, change int) string {
	// This is for permanent changes like from "Toughened" talent
	// Find Max HP line and update it
	return sheet // Placeholder - needs full markdown parsing
}

// applyXPChange applies experience change
func applyXPChange(sheet string, change int) string {
	// Parse current XP
	var currentXP int
	if idx := strings.Index(sheet, "XP:"); idx >= 0 {
		if _, err := fmt.Sscanf(sheet[idx:], "XP: %d", &currentXP); err == nil {
			newXP := currentXP + change
			replacer := strings.NewReplacer(
				fmt.Sprintf("XP: %d", currentXP),
				fmt.Sprintf("XP: %d", newXP),
			)
			return replacer.Replace(sheet)
		}
	}
	return sheet
}

// applyStatChange applies characteristic change
func applyStatChange(sheet string, stat string, change int) string {
	// Parse current stat value
	var currentValue int
	statMarker := fmt.Sprintf("%s:", stat)

	if idx := strings.Index(sheet, statMarker); idx >= 0 {
		if _, err := fmt.Sscanf(sheet[idx:], statMarker+" %d", &currentValue); err == nil {
			newValue := currentValue + change
			// WFRP stats max at 100 (without advances)
			if newValue > 100 {
				newValue = 100
			}
			if newValue < 0 {
				newValue = 0
			}
			replacer := strings.NewReplacer(
				fmt.Sprintf("%s %d", stat, currentValue),
				fmt.Sprintf("%s %d", stat, newValue),
			)
			return replacer.Replace(sheet)
		}
	}
	return sheet
}

// addSkillToSheet adds a new skill to the character sheet
func addSkillToSheet(sheet string, skill string) string {
	// Find the skills section and add the skill
	skillsSection := "## Навыки"
	if idx := strings.Index(sheet, skillsSection); idx >= 0 {
		// Find end of section
		endIdx := strings.Index(sheet[idx:], "##")
		if endIdx == -1 {
			endIdx = len(sheet[idx:])
		}
		insertPoint := idx + len(skillsSection)

		// Insert skill with proper formatting
		newSkill := fmt.Sprintf("\n- %s", skill)
		return sheet[:insertPoint] + newSkill + sheet[insertPoint:]
	}
	return sheet
}

// addConditionToSheet adds a condition to the character sheet
func addConditionToSheet(sheet string, condition string) string {
	// Add to existing conditions or create new section
	conditionsHeader := "## Состояния"
	conditionsMarker := "### Психологические состояния"

	var insertPoint int
	var newCondition string

	if idx := strings.Index(sheet, conditionsHeader); idx >= 0 {
		// Add to existing section
		if markerIdx := strings.Index(sheet, conditionsMarker); markerIdx > idx {
			insertPoint = markerIdx
			newCondition = fmt.Sprintf("\n- %s", condition)
		} else {
			// No marker, add after header
			insertPoint = idx + len(conditionsHeader)
			newCondition = fmt.Sprintf("\n\n%s\n- %s", conditionsMarker, condition)
		}
	} else {
		// Create new conditions section
		insertPoint = len(sheet)
		newCondition = fmt.Sprintf("\n\n%s\n\n%s\n- %s", conditionsHeader, conditionsMarker, condition)
	}

	return sheet[:insertPoint] + newCondition + sheet[insertPoint:]
}

// ParseCharacterUpdateFromResponse parses LLM response for character updates
func ParseCharacterUpdateFromResponse(response string) (playerID string, update *CharacterUpdate, err error) {
	update = &CharacterUpdate{
		StatsChanges: make(map[string]int),
		SkillsAdded:  make([]string, 0),
		Conditions:   make([]string, 0),
	}

	lines := strings.Split(response, "\n")

	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))

		// Parse HP changes
		if strings.Contains(lower, "получил") || strings.Contains(lower, "took damage") {
			var damage int
			if _, err := fmt.Sscanf(line, "%*[damage ]*%d", &damage); err == nil {
				update.HPChange -= damage
			}
		}

		// Parse healing
		if strings.Contains(lower, "вылечен") || strings.Contains(lower, "healed") {
			var healing int
			if _, err := fmt.Sscanf(line, "%*[healed ]*%d", &healing); err == nil {
				update.HPChange += healing
			}
		}

		// Parse XP gain
		if strings.Contains(lower, "получил опыт") || strings.Contains(lower, "gained xp") {
			var xp int
			if _, err := fmt.Sscanf(line, "%*[xp ]*%d", &xp); err == nil {
				update.XPChange += xp
			}
		}

		// Parse skill gains
		if strings.Contains(lower, "навык") || strings.Contains(lower, "skill") {
			// Extract skill name from line
			skillName := extractSkillFromLine(line)
			if skillName != "" {
				update.SkillsAdded = append(update.SkillsAdded, skillName)
			}
		}

		// Parse conditions
		if strings.Contains(lower, "ранение") || strings.Contains(lower, "wound") {
			update.Conditions = append(update.Conditions, "Wounded")
		}
		if strings.Contains(lower, "кровотечение") || strings.Contains(lower, "bleeding") {
			update.Conditions = append(update.Conditions, "Bleeding")
		}
		if strings.Contains(lower, "крит") || strings.Contains(lower, "critical") {
			update.Conditions = append(update.Conditions, "Critical Wound")
		}
	}

	return "", update, nil
}

// extractSkillFromLine extracts skill name from a line
func extractSkillFromLine(line string) string {
	// Simple extraction - could be enhanced
	parts := strings.Fields(line)
	for _, part := range parts {
		if strings.HasSuffix(part, ":") || strings.HasSuffix(part, "-") {
			continue
		}
		if len(part) > 2 {
			return strings.TrimSpace(part)
		}
	}
	return ""
}

// ValidateUpdate checks if an update is valid per WFRP rules
func ValidateUpdate(update CharacterUpdate, currentStats *CharacterStats) []string {
	var errors []string

	// Check HP bounds
	if currentStats != nil {
		newHP := currentStats.CurrentHP + update.HPChange
		if newHP < 0 {
			errors = append(errors, "HP cannot be negative")
		}
		if newHP > currentStats.MaxHP && update.MaxHPChange == 0 {
			errors = append(errors, "HP cannot exceed Max HP without healing")
		}
	}

	// Check XP
	if update.XPChange < 0 {
		errors = append(errors, "Cannot lose XP")
	}

	// Check stat bounds
	for stat, change := range update.StatsChanges {
		if change < 0 {
			errors = append(errors, fmt.Sprintf("Cannot lose %s stat", stat))
		}
		// Stats normally max at 100 without advances
		if change > 100 {
			errors = append(errors, fmt.Sprintf("%s change exceeds WFRP limits", stat))
		}
	}

	return errors
}

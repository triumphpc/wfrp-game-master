// Package game provides RAG-MCP-Server integration for WFRP rule checking
package game

import (
	"fmt"
	"log"
	"strings"
)

// RuleChecker validates game actions against WFRP rules
type RuleChecker struct {
	ruleCache map[string]string
}

// NewRuleChecker creates a new rule checker
func NewRuleChecker() *RuleChecker {
	return &RuleChecker{
		ruleCache: make(map[string]string),
	}
}

// Check validates an action against WFRP rules
func (rc *RuleChecker) Check(input InputData) ([]string, error) {
	var violations []string

	// Check based on input content
	content := strings.ToLower(input.Content)

	// Check for common rule violations
	if rc.checkCombatRules(content, input) {
		violations = append(violations, "Combat action needs proper skill check")
	}

	if rc.checkSkillRules(content, input) {
		violations = append(violations, "Skill check requires target characteristic")
	}

	// Log all violations for GM consideration
	if len(violations) > 0 {
		log.Printf("[RAG] Rule violations found: %v", violations)
	}

	return violations, nil
}

// CheckRule looks up a specific rule
func (rc *RuleChecker) CheckRule(query string) (string, error) {
	queryLower := strings.ToLower(query)

	// Check cache first
	if cached, exists := rc.ruleCache[queryLower]; exists {
		return cached, nil
	}

	// Try to match against known rule patterns
	rule := rc.findRulePattern(queryLower)
	if rule != "" {
		rc.ruleCache[queryLower] = rule
		return rule, nil
	}

	return "", fmt.Errorf("rule not found: %s", query)
}

// SearchRules searches for rules matching a query
func (rc *RuleChecker) SearchRules(query string) []RuleMatch {
	results := []RuleMatch{}

	// Simple keyword matching - could be enhanced with actual RAG
	queryLower := strings.ToLower(query)

	// Define known rule patterns
	patterns := rc.getRulePatterns()

	for _, pattern := range patterns {
		if strings.Contains(queryLower, pattern.keyword) {
			results = append(results, RuleMatch{
				Rule:     pattern.rule,
				Confidence: 0.7, // Default confidence
				Source:    "pattern-match",
			})
		}
	}

	return results
}

// checkCombatRules validates combat-related actions
func (rc *RuleChecker) checkCombatRules(content string, input InputData) bool {
	combatKeywords := []string{
		"атака", "attack", "бью", "hit", "удар",
		"стреля", "shoot", "защита", "defend", "parry",
	}

	for _, keyword := range combatKeywords {
		if strings.Contains(content, keyword) {
			// Check if there's a characteristic/skill mentioned
			hasSkill := strings.Contains(content, "WS") ||
				strings.Contains(content, "BS") ||
				strings.Contains(content, "В") ||
				strings.Contains(content, "С") ||
				strings.Contains(content, "Лов")

			return !hasSkill
		}
	}

	return false
}

// checkSkillRules validates skill check actions
func (rc *RuleChecker) checkSkillRules(content string, input InputData) bool {
	skillKeywords := []string{
		"проверка", "check", "проверить", "check it",
		"навык", "skill", "способность",
	}

	for _, keyword := range skillKeywords {
		if strings.Contains(content, keyword) {
			// Check if a characteristic is mentioned
			hasChar := strings.ContainsAny(content,
				"В", "С", "Лов", "Инт", "ВН", "Об",
				"WS", "BS", "S", "Ag", "Int", "WP", "Fel")

			return !hasChar
		}
	}

	return false
}

// findRulePattern finds a matching rule pattern
func (rc *RuleChecker) findRulePattern(query string) string {
	patterns := rc.getRulePatterns()

	for _, pattern := range patterns {
		if strings.Contains(query, pattern.keyword) {
			return pattern.rule
		}
	}

	return ""
}

// getRulePatterns returns known rule patterns
func (rc *RuleChecker) getRulePatterns() []rulePattern {
	return []rulePattern{
		// Combat rules
		{"keyword": "инициатива", "rule": "Initiative is rolled at start of combat using Agility (Ag)"},
		{"keyword": "атака", "rule": "Combat uses Weapon Skill (WS) against opponent's Parry (Ag)"},
		{"keyword": "урон", "rule": "Damage is calculated from weapon damage minus enemy Toughness/Armor"},

		// Skill checks
		{"keyword": "проверка навыка", "rule": "Skill checks use d100 + characteristic value vs difficulty"},
		{"keyword": "провал проверки", "rule": "Failed check: result is higher than characteristic + skill"},

		// Character development
		{"keyword": "опыт", "rule": "Experience (XP) is spent to advance characteristics and skills"},
		{"keyword": "карьера", "rule": "Career advancement follows the scheme in КАРЬЕРЫ.md"},

		// Conditions
		{"keyword": "ранение", "rule": "Wounds reduce HP and may cause critical effects"},
		{"keyword": "шок", "rule": "Critical wounds cause Bleeding, Broken, etc."},

		// Movement
		{"keyword": "движение", "rule": "Movement rate (M) is derived from Agility (Ag)"},
	}
}

// RuleMatch represents a search result from rule lookup
type RuleMatch struct {
	Rule      string
	Confidence float64
	Source     string
}

// ValidateRuleCheck validates a skill check format
func (rc *RuleChecker) ValidateRuleCheck(characteristic, skill string, result int) error {
	validChars := []string{"В", "С", "Лов", "Инт", "ВН", "Об"}

	// Check if characteristic is valid
	charValid := false
	for _, vc := range validChars {
		if characteristic == vc {
			charValid = true
			break
		}
	}

	if !charValid {
		return fmt.Errorf("invalid characteristic: %s", characteristic)
	}

	// Check result range
	if result < 0 || result > 200 {
		return fmt.Errorf("invalid roll result: %d", result)
	}

	return nil
}

// GetRulesForContext returns relevant rules for a game context
func (rc *RuleChecker) GetRulesForContext(context string) []string {
	var relevantRules []string

	// Extract keywords from context
	words := strings.Fields(context)

	// Find matching rules
	for _, word := range words {
		for _, pattern := range rc.getRulePatterns() {
			if strings.Contains(word, pattern.keyword) {
				ruleText := fmt.Sprintf("%s: %s", pattern.keyword, pattern.rule)
				if !containsString(relevantRules, ruleText) {
					relevantRules = append(relevantRules, ruleText)
				}
			}
		}
	}

	return relevantRules
}

// containsString checks if a string exists in a slice
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// rulePattern represents a keyword to rule mapping
type rulePattern struct {
	keyword string
	rule    string
}

// stringsContainsAny checks if any of the substrings are in the main string
func stringsContainsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

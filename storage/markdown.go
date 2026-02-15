// Package storage provides Markdown file parsing for WFRP bot
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MarkdownParser handles parsing of WFRP markdown files
type MarkdownParser struct {
	basePath string
}

// NewMarkdownParser creates a new markdown parser
func NewMarkdownParser(basePath string) *MarkdownParser {
	return &MarkdownParser{
		basePath: basePath,
	}
}

// ReadFile reads and parses a markdown file
func (mp *MarkdownParser) ReadFile(path string) (string, error) {
	fullPath := mp.resolvePath(path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	return string(content), nil
}

// WriteFile writes content to a markdown file
func (mp *MarkdownParser) WriteFile(path, content string) error {
	fullPath := mp.resolvePath(path)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

// ParseCharacterSheet parses a character sheet from markdown
func (mp *MarkdownParser) ParseCharacterSheet(content string) (*ParsedCharacter, error) {
	char := &ParsedCharacter{
		Fields: make(map[string]string),
	}

	// Parse using regex patterns
	lines := strings.Split(content, "\n")
	currentSection := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect sections
		if strings.HasPrefix(trimmed, "#") {
			currentSection = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			continue
		}

		// Parse field pairs (key: value)
		if strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				char.Fields[key] = value
			}
		}

		// Parse list items (- item)
		if strings.HasPrefix(trimmed, "-") {
			item := strings.TrimSpace(strings.TrimLeft(trimmed, "-"))
			char.ListItems = append(char.ListItems, item)
		}
	}

	// Extract name from fields or first header
	if name, ok := char.Fields["Имя"]; ok {
		char.Name = name
	} else if name, ok := char.Fields["Name"]; ok {
		char.Name = name
	}

	return char, nil
}

// ParseSessionLog parses a session log from markdown
func (mp *MarkdownParser) ParseSessionLog(content string) (*SessionLog, error) {
	log := &SessionLog{
		Entries: make([]LogEntry, 0),
	}

	// Extract metadata from first lines
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Parse metadata lines
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "##") {
			mp.parseSessionMetadata(trimmed, log)
			continue
		}

		// Parse log entries
		entry := mp.parseLogEntry(trimmed)
		if entry != nil {
			log.Entries = append(log.Entries, *entry)
		}
	}

	return log, nil
}

// ParsedCharacter represents a parsed character sheet
type ParsedCharacter struct {
	Name      string
	Fields    map[string]string
	ListItems []string
	Sections  map[string][]string
}

// SessionLog represents a parsed session log
type SessionLog struct {
	Date      string
	Title     string
	Summary   string
	Entries   []LogEntry
	Characters []string
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string
	Type      string // "action", "dialogue", "system"
	Actor     string
	Content   string
	Roll      *DiceRoll
}

// DiceRoll represents a dice roll result
type DiceRoll struct {
	Type     string // "d100", "d10", "2d10"
	Characteristic string
	Skill     string
	Result    int
	Modifier  int
}

// parseSessionMetadata extracts metadata from log lines
func (mp *MarkdownParser) parseSessionMetadata(line string, log *SessionLog) {
	if strings.Contains(line, "Дата:") || strings.Contains(line, "Date:") {
		// Extract date
		dateStr := strings.TrimSpace(strings.SplitAfter(line, ":")[1])
		log.Date = dateStr
	}

	if strings.Contains(line, "Участники:") || strings.Contains(line, "Participants:") {
		// Extract participants
		participants := strings.TrimSpace(strings.SplitAfter(line, ":")[1])
		log.Characters = strings.Split(participants, ",")
	}

	if strings.Contains(line, "Итог:") || strings.Contains(line, "Summary:") {
		// Extract summary
		summary := strings.TrimSpace(strings.SplitAfter(line, ":")[1])
		log.Summary = summary
	}
}

// parseLogEntry parses a single log entry
func (mp *MarkdownParser) parseLogEntry(line string) *LogEntry {
	if line == "" {
		return nil
	}

	entry := &LogEntry{
		Timestamp: "now",
		Type:      "action",
		Content:   line,
	}

	// Parse timestamp
	timestampPattern := regexp.MustCompile(`\[(\d{2}:\d{2})\]`)
	if matches := timestampPattern.FindStringSubmatch(line); len(matches) > 1 {
		entry.Timestamp = matches[1]
	}

	// Parse dice rolls
	dicePattern := regexp.MustCompile(`d(\d+)|(\d+)d(\d+)`)
	if diceMatches := dicePattern.FindAllString(line, -1); len(diceMatches) > 0 {
		entry.Roll = &DiceRoll{
			Type:    diceMatches[0],
			Result:   mp.extractRollResult(line),
		}
	}

	return entry
}

// extractRollResult extracts dice roll result from line
func (mp *MarkdownParser) extractRollResult(line string) int {
	// Find result after roll
	resultPattern := regexp.MustCompile(`=\s*(\d+)`)
	if matches := resultPattern.FindStringSubmatch(line); len(matches) > 1 {
		if result, err := parseResult(matches[1]); err == nil {
			return result
		}
	}

	return 0
}

// parseResult parses a result string to int
func parseResult(s string) (int, error) {
	result := 0
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// ExtractFrontmatter extracts YAML frontmatter from markdown
func (mp *MarkdownParser) ExtractFrontmatter(content string) (map[string]string, string) {
	// Check for --- delimiters
	if !strings.HasPrefix(content, "---") {
		return make(map[string]string), content
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return make(map[string]string), content
	}

	frontmatter := parts[1]
	body := parts[2]

	// Parse simple key: value pairs
	metadata := make(map[string]string)
	for _, line := range strings.Split(frontmatter, "\n") {
		if strings.Contains(line, ":") {
			kv := strings.SplitN(line, ":", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				metadata[key] = value
			}
		}
	}

	return metadata, body
}

// BuildSessionLog creates a session log from entries
func (mp *MarkdownParser) BuildSessionLog(date, title, summary string, entries []LogEntry) (string, error) {
	var builder strings.Builder

	// Header
	builder.WriteString(fmt.Sprintf("# %s\n\n", title))
	builder.WriteString(fmt.Sprintf("## Date: %s\n\n", date))

	// Summary
	if summary != "" {
		builder.WriteString(fmt.Sprintf("## Summary\n%s\n\n", summary))
	}

	// Entries
	builder.WriteString("## Log\n\n")
	for _, entry := range entries {
		if entry.Timestamp != "now" {
			builder.WriteString(fmt.Sprintf("[%s] %s\n", entry.Timestamp, entry.Content))
		} else {
			builder.WriteString(fmt.Sprintf("%s\n", entry.Content))
		}
	}

	return builder.String(), nil
}

// BuildCharacterSheet creates a character sheet from parsed data
func (mp *MarkdownParser) BuildCharacterSheet(char *ParsedCharacter) (string, error) {
	var builder strings.Builder

	// Header
	builder.WriteString(fmt.Sprintf("# %s\n\n", char.Name))

	// Sections
	if len(char.Fields) > 0 {
		for section, value := range char.Fields {
			builder.WriteString(fmt.Sprintf("## %s\n%s\n\n", section, value))
		}
	}

	// List items
	if len(char.ListItems) > 0 {
		builder.WriteString("## Inventory\n")
		for _, item := range char.ListItems {
			builder.WriteString(fmt.Sprintf("- %s\n", item))
		}
	}

	return builder.String(), nil
}

// resolvePath resolves a relative path against base path
func (mp *MarkdownParser) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(mp.basePath, path)
}

// SplitAfter splits string after first occurrence of separator
func SplitAfter(s, sep string) string {
	idx := strings.Index(s, sep)
	if idx == -1 {
		return s
	}
	return s[idx+len(sep):]
}

// ContainsAny checks if any substring exists in string
func ContainsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

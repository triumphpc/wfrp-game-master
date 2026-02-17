// Package storage provides campaign management for WFRP bot
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// CampaignManager manages WFRP campaigns
type CampaignManager struct {
	basePath  string
	parser    *MarkdownParser
	campaigns map[string]*Campaign
	mu        sync.RWMutex
}

// NewCampaignManager creates a new campaign manager
func NewCampaignManager(basePath string) *CampaignManager {
	return &CampaignManager{
		basePath:  basePath,
		parser:    NewMarkdownParser(basePath),
		campaigns: make(map[string]*Campaign),
	}
}

// Campaign represents a WFRP campaign
type Campaign struct {
	Name         string
	Path         string
	Description  string
	CreatedAt    time.Time
	LastModified time.Time
	Characters   []string
	Sessions     []string
}

// ListCampaigns returns all available campaigns
func (cm *CampaignManager) ListCampaigns() ([]*Campaign, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	campaigns := make([]*Campaign, 0, len(cm.campaigns))

	for _, camp := range cm.campaigns {
		campaigns = append(campaigns, camp)
	}

	// Sort by last modified
	sort.Slice(campaigns, func(i, j int) bool {
		return campaigns[i].LastModified.After(campaigns[j].LastModified)
	})

	return campaigns, nil
}

// GetCampaign returns a campaign by name
func (cm *CampaignManager) GetCampaign(name string) (*Campaign, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	camp, exists := cm.campaigns[name]
	if !exists {
		return nil, fmt.Errorf("campaign not found: %s", name)
	}

	return camp, nil
}

// CreateCampaign creates a new campaign directory
func (cm *CampaignManager) CreateCampaign(name, description string) (*Campaign, error) {
	// Validate name
	if !isValidCampaignName(name) {
		return nil, fmt.Errorf("invalid campaign name: %s", name)
	}

	// Create campaign directory
	campPath := filepath.Join(cm.basePath, name)
	if err := os.MkdirAll(campPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create campaign directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"characters", "sessions"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(campPath, subdir), 0755); err != nil {
			return nil, fmt.Errorf("failed to create %s: %w", subdir, err)
		}
	}

	// Create campaign info file
	campInfo := &Campaign{
		Name:         name,
		Path:         campPath,
		Description:  description,
		CreatedAt:    time.Now(),
		LastModified: time.Now(),
		Characters:   make([]string, 0),
		Sessions:     make([]string, 0),
	}

	// Save campaign metadata
	if err := cm.saveCampaignInfo(campPath, campInfo); err != nil {
		return nil, err
	}

	// Add to manager
	cm.mu.Lock()
	cm.campaigns[name] = campInfo
	cm.mu.Unlock()

	// Index campaign
	cm.indexCampaign(campPath, campInfo)

	return campInfo, nil
}

// DeleteCampaign removes a campaign
func (cm *CampaignManager) DeleteCampaign(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	camp, exists := cm.campaigns[name]
	if !exists {
		return fmt.Errorf("campaign not found: %s", name)
	}

	// Remove directory and all contents
	if err := os.RemoveAll(camp.Path); err != nil {
		return fmt.Errorf("failed to delete campaign: %w", err)
	}

	delete(cm.campaigns, name)

	return nil
}

// ListSessions returns all sessions for a campaign
func (cm *CampaignManager) ListSessions(campaign string) ([]string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	camp, exists := cm.campaigns[campaign]
	if !exists {
		return nil, fmt.Errorf("campaign not found: %s", campaign)
	}

	sessions := make([]string, len(camp.Sessions))
	copy(sessions, camp.Sessions)

	return sessions, nil
}

// ListCharacters returns all characters for a campaign
func (cm *CampaignManager) ListCharacters(campaign string) ([]string, error) {
	charDir := filepath.Join(cm.basePath, campaign, "characters")

	entries, err := os.ReadDir(charDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var characters []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			characters = append(characters, entry.Name())
		}
	}

	return characters, nil
}

// Refresh reloads all campaigns from disk
func (cm *CampaignManager) Refresh() error {
	// Scan base directory for campaign directories
	entries, err := os.ReadDir(cm.basePath)
	if err != nil {
		return fmt.Errorf("failed to read campaigns directory: %w", err)
	}

	newCampaigns := make(map[string]*Campaign)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		campName := entry.Name()

		// Skip non-campaign directories
		if campName == "venv" || campName == ".git" || campName == ".idea" {
			continue
		}

		campPath := filepath.Join(cm.basePath, campName)

		fileInfo, err := entry.Info()
		if err != nil {
			return fmt.Errorf("failed to get file info for %s: %w", campName, err)
		}

		campInfo, err := cm.loadCampaignInfo(campPath)
		if err != nil {
			// Create basic info if metadata doesn't exist
			campInfo = &Campaign{
				Name:         campName,
				Path:         campPath,
				LastModified: fileInfo.ModTime(),
			}
		}

		// Load characters and sessions
		cm.loadCampaignData(campPath, campInfo)

		newCampaigns[campName] = campInfo
	}

	cm.mu.Lock()
	cm.campaigns = newCampaigns
	cm.mu.Unlock()

	return nil
}

// loadCampaignInfo loads campaign metadata from file
func (cm *CampaignManager) loadCampaignInfo(campPath string) (*Campaign, error) {
	infoPath := filepath.Join(campPath, "campaign.md")

	content, err := cm.parser.ReadFile(infoPath)
	if err != nil {
		return nil, err // No info file is OK
	}

	// Parse simple markdown format
	info := &Campaign{
		Path: campPath,
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			info.Name = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		} else if strings.HasPrefix(trimmed, "Описание:") ||
			strings.HasPrefix(trimmed, "Description:") {
			info.Description = strings.TrimSpace(strings.SplitAfter(trimmed, ":")[1])
		}
	}

	return info, nil
}

// loadCampaignData loads character and session lists
func (cm *CampaignManager) loadCampaignData(campPath string, camp *Campaign) {
	// Load characters
	charDir := filepath.Join(campPath, "characters")
	if entries, err := os.ReadDir(charDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				camp.Characters = append(camp.Characters, entry.Name())
			}
		}
	}

	// Load sessions
	sessDir := filepath.Join(campPath, "sessions")
	if entries, err := os.ReadDir(sessDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				camp.Sessions = append(camp.Sessions, entry.Name())
			}
		}
	}
}

// saveCampaignInfo writes campaign metadata to file
func (cm *CampaignManager) saveCampaignInfo(campPath string, camp *Campaign) error {
	infoPath := filepath.Join(campPath, "campaign.md")

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("# %s\n\n", camp.Name))

	if camp.Description != "" {
		builder.WriteString(fmt.Sprintf("Описание: %s\n\n", camp.Description))
	}

	return cm.parser.WriteFile(infoPath, builder.String())
}

// indexCampaign adds campaign to search index
func (cm *CampaignManager) indexCampaign(campPath string, camp *Campaign) {
	// Placeholder for future search functionality
	// Could integrate with Qdrant or other vector DB
}

// isValidCampaignName validates campaign name
func isValidCampaignName(name string) bool {
	if len(name) == 0 || len(name) > 100 {
		return false
	}

	// Check for invalid characters
	invalid := "/\\<>:\"|?*"

	for _, ch := range invalid {
		if strings.Contains(name, string(ch)) {
			return false
		}
	}

	return true
}

// GetCampaignPath returns the full path to a campaign
func (cm *CampaignManager) GetCampaignPath(name string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if camp, exists := cm.campaigns[name]; exists {
		return camp.Path
	}

	return ""
}

// LoadPartySummary loads party summary for a campaign
func (cm *CampaignManager) LoadPartySummary(campaign string) (string, error) {
	camp, exists := cm.campaigns[campaign]
	if !exists {
		return "", fmt.Errorf("campaign not found: %s", campaign)
	}

	summaryPath := filepath.Join(camp.Path, "party_summary.md")

	content, err := cm.parser.ReadFile(summaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No summary is OK
		}
		return "", err
	}

	return content, nil
}

// SavePartySummary writes party summary to file
func (cm *CampaignManager) SavePartySummary(campaign, summary string) error {
	camp, exists := cm.campaigns[campaign]
	if !exists {
		return fmt.Errorf("campaign not found: %s", campaign)
	}

	summaryPath := filepath.Join(camp.Path, "party_summary.md")

	return cm.parser.WriteFile(summaryPath, summary)
}

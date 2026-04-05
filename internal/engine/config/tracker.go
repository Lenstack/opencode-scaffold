package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

const (
	NSConfig     = "config"
	NSConfigHist = "config:history"
)

type Tracker struct {
	db   *hub.Engine
	root string
}

func NewTracker(db *hub.Engine, root string) *Tracker {
	return &Tracker{db: db, root: root}
}

type ConfigEntry struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	Version    int    `json:"version"`
	ModifiedAt string `json:"modified_at"`
	ModifiedBy string `json:"modified_by"`
	Source     string `json:"source"`
}

type ConfigChange struct {
	Path       string `json:"path"`
	OldVersion int    `json:"old_version"`
	NewVersion int    `json:"new_version"`
	Action     string `json:"action"`
	Timestamp  string `json:"timestamp"`
}

func (t *Tracker) TrackConfig(path, content, modifiedBy, source string) error {
	key := normalizeConfigKey(path)

	var existing ConfigEntry
	var version int
	if err := t.db.Get(NSConfig, key, &existing); err == nil {
		version = existing.Version
		if existing.Content == content {
			return nil
		}
		version++
	} else {
		version = 1
	}

	entry := ConfigEntry{
		Path:       path,
		Content:    content,
		Version:    version,
		ModifiedAt: time.Now().UTC().Format(time.RFC3339),
		ModifiedBy: modifiedBy,
		Source:     source,
	}

	if err := t.db.Put(NSConfig, key, entry); err != nil {
		return fmt.Errorf("store config: %w", err)
	}

	change := ConfigChange{
		Path:       path,
		OldVersion: version - 1,
		NewVersion: version,
		Action:     "update",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	if version == 1 {
		change.Action = "create"
		change.OldVersion = 0
	}

	histKey := fmt.Sprintf("%s:%s", time.Now().Format("20060102150405"), key)
	if err := t.db.Put(NSConfigHist, histKey, change); err != nil {
		return fmt.Errorf("store config history: %w", err)
	}

	return nil
}

func (t *Tracker) GetConfig(path string) (*ConfigEntry, error) {
	key := normalizeConfigKey(path)
	var entry ConfigEntry
	if err := t.db.Get(NSConfig, key, &entry); err != nil {
		return nil, fmt.Errorf("config %s not found: %w", path, err)
	}
	return &entry, nil
}

func (t *Tracker) ListConfigs() ([]ConfigEntry, error) {
	var configs []ConfigEntry

	t.db.Iterate(NSConfig, func(key string, value []byte) error {
		var entry ConfigEntry
		if err := json.Unmarshal(value, &entry); err == nil {
			configs = append(configs, entry)
		}
		return nil
	})

	return configs, nil
}

func (t *Tracker) GetHistory(path string) ([]ConfigChange, error) {
	key := normalizeConfigKey(path)
	var changes []ConfigChange

	t.db.Iterate(NSConfigHist, func(histKey string, value []byte) error {
		var change ConfigChange
		if err := json.Unmarshal(value, &change); err == nil {
			if normalizeConfigKey(change.Path) == key {
				changes = append(changes, change)
			}
		}
		return nil
	})

	return changes, nil
}

func (t *Tracker) TrackAllConfigs(modifiedBy, source string) error {
	configFiles := []string{
		"opencode.json",
		"AGENTS.md",
	}

	configDirs := []string{
		".opencode/agents",
		".opencode/skills",
		".opencode/commands",
		".opencode/plugins",
	}

	for _, f := range configFiles {
		content, err := os.ReadFile(filepath.Join(t.root, f))
		if err != nil {
			continue
		}
		t.TrackConfig(f, string(content), modifiedBy, source)
	}

	for _, dir := range configDirs {
		fullDir := filepath.Join(t.root, dir)
		entries, err := os.ReadDir(fullDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				subEntries, _ := os.ReadDir(filepath.Join(fullDir, e.Name()))
				for _, se := range subEntries {
					if !se.IsDir() {
						content, _ := os.ReadFile(filepath.Join(fullDir, e.Name(), se.Name()))
						t.TrackConfig(filepath.Join(dir, e.Name(), se.Name()), string(content), modifiedBy, source)
					}
				}
			} else {
				content, _ := os.ReadFile(filepath.Join(fullDir, e.Name()))
				t.TrackConfig(filepath.Join(dir, e.Name()), string(content), modifiedBy, source)
			}
		}
	}

	return nil
}

func normalizeConfigKey(path string) string {
	key := filepath.Clean(path)
	key = filepath.ToSlash(key)
	return key
}

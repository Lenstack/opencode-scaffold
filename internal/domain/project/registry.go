package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
	"github.com/google/uuid"
)

const registryFile = "project.json"

type Registry struct {
	ProjectID    string `json:"project_id"`
	ProjectName  string `json:"project_name"`
	HubURL       string `json:"hub_url,omitempty"`
	Workspace    string `json:"workspace,omitempty"`
	APIKey       string `json:"api_key,omitempty"`
	RegisteredAt string `json:"registered_at"`
	LastSync     string `json:"last_sync,omitempty"`
	Stack        string `json:"stack,omitempty"`
}

type Manager struct {
	db   *hub.Engine
	root string
}

func NewManager(db *hub.Engine, root string) *Manager {
	return &Manager{db: db, root: root}
}

func (m *Manager) Register(name, hubURL, workspace, apiKey string) (*Registry, error) {
	existing, _ := m.Load()
	if existing != nil && existing.ProjectID != "" {
		return nil, fmt.Errorf("project already registered (id: %s), run unregister first", existing.ProjectID)
	}

	reg := &Registry{
		ProjectID:    uuid.New().String(),
		ProjectName:  name,
		HubURL:       hubURL,
		Workspace:    workspace,
		APIKey:       apiKey,
		RegisteredAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := m.Save(reg); err != nil {
		return nil, err
	}

	if err := m.db.Put("hub:project", "identity", reg); err != nil {
		return nil, fmt.Errorf("store project identity: %w", err)
	}

	return reg, nil
}

func (m *Manager) Unregister() error {
	path := m.registryPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove registry file: %w", err)
	}
	m.db.Delete("hub:project", "identity")
	return nil
}

func (m *Manager) Load() (*Registry, error) {
	var reg Registry
	if err := m.db.Get("hub:project", "identity", &reg); err == nil {
		return &reg, nil
	}

	path := m.registryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project not registered, run 'ocs register'")
		}
		return nil, fmt.Errorf("read registry: %w", err)
	}

	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}

	m.db.Put("hub:project", "identity", reg)
	return &reg, nil
}

func (m *Manager) Save(reg *Registry) error {
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	path := m.registryPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	return nil
}

func (m *Manager) UpdateSyncTime() error {
	reg, err := m.Load()
	if err != nil {
		return err
	}
	reg.LastSync = time.Now().UTC().Format(time.RFC3339)
	return m.Save(reg)
}

func (m *Manager) IsRegistered() bool {
	reg, err := m.Load()
	return err == nil && reg.ProjectID != ""
}

func (m *Manager) registryPath() string {
	return filepath.Join(m.root, ".opencode", "data", registryFile)
}

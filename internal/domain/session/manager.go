package session

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

type Manager struct {
	db *hub.Engine
}

func NewManager(db *hub.Engine) *Manager {
	return &Manager{db: db}
}

func (m *Manager) Create(id, title, agent, model string) (*hub.SessionEntry, error) {
	entry := hub.SessionEntry{
		ID:        id,
		Title:     title,
		Agent:     agent,
		Model:     model,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Status:    "active",
	}
	if err := m.db.Put(hub.NSSessions, id, entry); err != nil {
		return nil, err
	}
	m.db.Put(hub.NSSessions, id+":context", hub.SessionContext{})
	return &entry, nil
}

func (m *Manager) Get(id string) (*hub.SessionEntry, error) {
	var entry hub.SessionEntry
	if err := m.db.Get(hub.NSSessions, id, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (m *Manager) List() ([]hub.SessionEntry, error) {
	var sessions []hub.SessionEntry
	m.db.Iterate(hub.NSSessions, func(key string, value []byte) error {
		if len(key) > 0 && key[0] != ':' {
			var entry hub.SessionEntry
			if err := unmarshal(value, &entry); err == nil {
				sessions = append(sessions, entry)
			}
		}
		return nil
	})
	return sessions, nil
}

func (m *Manager) UpdateStatus(id, status string) error {
	var entry hub.SessionEntry
	if err := m.db.Get(hub.NSSessions, id, &entry); err != nil {
		return err
	}
	entry.Status = status
	return m.db.Put(hub.NSSessions, id, entry)
}

func (m *Manager) AddContext(id string, filesRead, filesWritten, decisions []string) error {
	var ctx hub.SessionContext
	m.db.Get(hub.NSSessions, id+":context", &ctx)
	ctx.FilesRead = append(ctx.FilesRead, filesRead...)
	ctx.FilesWritten = append(ctx.FilesWritten, filesWritten...)
	ctx.Decisions = append(ctx.Decisions, decisions...)
	return m.db.Put(hub.NSSessions, id+":context", ctx)
}

func (m *Manager) GetContext(id string) (*hub.SessionContext, error) {
	var ctx hub.SessionContext
	if err := m.db.Get(hub.NSSessions, id+":context", &ctx); err != nil {
		return nil, err
	}
	return &ctx, nil
}

func (m *Manager) SetSummary(id string, summary hub.SessionSummary) error {
	return m.db.Put(hub.NSSessions, id+":summary", summary)
}

func (m *Manager) GetSummary(id string) (*hub.SessionSummary, error) {
	var s hub.SessionSummary
	if err := m.db.Get(hub.NSSessions, id+":summary", &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (m *Manager) Delete(id string) error {
	m.db.Delete(hub.NSSessions, id)
	m.db.Delete(hub.NSSessions, id+":context")
	m.db.Delete(hub.NSSessions, id+":summary")
	return nil
}

func (m *Manager) GetCurrent() (*hub.SessionEntry, error) {
	sessions, err := m.List()
	if err != nil {
		return nil, err
	}
	for i := len(sessions) - 1; i >= 0; i-- {
		if sessions[i].Status == "active" {
			return &sessions[i], nil
		}
	}
	if len(sessions) > 0 {
		return &sessions[len(sessions)-1], nil
	}
	return nil, fmt.Errorf("no sessions found")
}

func unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

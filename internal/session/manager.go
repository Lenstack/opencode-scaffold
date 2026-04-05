package session

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Lenstack/opencode-scaffold/internal/db"
)

type Manager struct {
	db *db.Engine
}

func NewManager(db *db.Engine) *Manager {
	return &Manager{db: db}
}

func (m *Manager) Create(id, title, agent, model string) (*db.SessionEntry, error) {
	entry := db.SessionEntry{
		ID:        id,
		Title:     title,
		Agent:     agent,
		Model:     model,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Status:    "active",
	}
	if err := m.db.Put(db.NSSessions, id, entry); err != nil {
		return nil, err
	}
	m.db.Put(db.NSSessions, id+":context", db.SessionContext{})
	return &entry, nil
}

func (m *Manager) Get(id string) (*db.SessionEntry, error) {
	var entry db.SessionEntry
	if err := m.db.Get(db.NSSessions, id, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (m *Manager) List() ([]db.SessionEntry, error) {
	var sessions []db.SessionEntry
	m.db.Iterate(db.NSSessions, func(key string, value []byte) error {
		if len(key) > 0 && key[0] != ':' {
			var entry db.SessionEntry
			if err := unmarshal(value, &entry); err == nil {
				sessions = append(sessions, entry)
			}
		}
		return nil
	})
	return sessions, nil
}

func (m *Manager) UpdateStatus(id, status string) error {
	var entry db.SessionEntry
	if err := m.db.Get(db.NSSessions, id, &entry); err != nil {
		return err
	}
	entry.Status = status
	return m.db.Put(db.NSSessions, id, entry)
}

func (m *Manager) AddContext(id string, filesRead, filesWritten, decisions []string) error {
	var ctx db.SessionContext
	m.db.Get(db.NSSessions, id+":context", &ctx)
	ctx.FilesRead = append(ctx.FilesRead, filesRead...)
	ctx.FilesWritten = append(ctx.FilesWritten, filesWritten...)
	ctx.Decisions = append(ctx.Decisions, decisions...)
	return m.db.Put(db.NSSessions, id+":context", ctx)
}

func (m *Manager) GetContext(id string) (*db.SessionContext, error) {
	var ctx db.SessionContext
	if err := m.db.Get(db.NSSessions, id+":context", &ctx); err != nil {
		return nil, err
	}
	return &ctx, nil
}

func (m *Manager) SetSummary(id string, summary db.SessionSummary) error {
	return m.db.Put(db.NSSessions, id+":summary", summary)
}

func (m *Manager) GetSummary(id string) (*db.SessionSummary, error) {
	var s db.SessionSummary
	if err := m.db.Get(db.NSSessions, id+":summary", &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (m *Manager) Delete(id string) error {
	m.db.Delete(db.NSSessions, id)
	m.db.Delete(db.NSSessions, id+":context")
	m.db.Delete(db.NSSessions, id+":summary")
	return nil
}

func (m *Manager) GetCurrent() (*db.SessionEntry, error) {
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

package spec

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

func (m *Manager) Create(name string, reqs hub.SpecRequirements) (*hub.SpecEntry, error) {
	id := slugify(name)
	var existing hub.SpecEntry
	if err := m.db.Get(hub.NSSpecs, id, &existing); err == nil || err != hub.ErrNotFound {
		if err == nil {
			return nil, fmt.Errorf("spec %s already exists", id)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	entry := hub.SpecEntry{
		ID:        id,
		Name:      name,
		Status:    "draft",
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.db.Put(hub.NSSpecs, id, entry)
	m.db.Put(hub.NSSpecs, id+":requirements", reqs)

	return &entry, nil
}

func (m *Manager) Get(id string) (*hub.SpecEntry, error) {
	var entry hub.SpecEntry
	if err := m.db.Get(hub.NSSpecs, id, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (m *Manager) List() ([]hub.SpecEntry, error) {
	var specs []hub.SpecEntry
	m.db.Iterate(hub.NSSpecs, func(key string, value []byte) error {
		if len(key) > 0 && key[0] != ':' {
			var entry hub.SpecEntry
			if err := unmarshal(value, &entry); err == nil {
				specs = append(specs, entry)
			}
		}
		return nil
	})
	return specs, nil
}

func (m *Manager) UpdateStatus(id, status string) error {
	var entry hub.SpecEntry
	if err := m.db.Get(hub.NSSpecs, id, &entry); err != nil {
		return err
	}

	validStatuses := map[string]bool{
		"draft": true, "planned": true, "implementing": true,
		"verifying": true, "done": true, "archived": true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	entry.Status = status
	entry.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return m.db.Put(hub.NSSpecs, id, entry)
}

func (m *Manager) UpdateImplementation(id string, impl hub.SpecImplementation) error {
	impl.VerifiedAt = time.Now().UTC().Format(time.RFC3339)
	return m.db.Put(hub.NSSpecs, id+":implementation", impl)
}

func (m *Manager) Verify(id string, verification hub.SpecVerification) error {
	verification.Status = "verified"
	if len(verification.FailedCriteria) > 0 {
		verification.Status = "failed"
	}
	if err := m.db.Put(hub.NSSpecs, id+":verification", verification); err != nil {
		return err
	}
	if verification.Status == "verified" {
		return m.UpdateStatus(id, "done")
	}
	return nil
}

func (m *Manager) GetRequirements(id string) (*hub.SpecRequirements, error) {
	var reqs hub.SpecRequirements
	if err := m.db.Get(hub.NSSpecs, id+":requirements", &reqs); err != nil {
		return nil, err
	}
	return &reqs, nil
}

func (m *Manager) GetImplementation(id string) (*hub.SpecImplementation, error) {
	var impl hub.SpecImplementation
	if err := m.db.Get(hub.NSSpecs, id+":implementation", &impl); err != nil {
		return nil, err
	}
	return &impl, nil
}

func (m *Manager) GetVerification(id string) (*hub.SpecVerification, error) {
	var v hub.SpecVerification
	if err := m.db.Get(hub.NSSpecs, id+":verification", &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (m *Manager) Delete(id string) error {
	m.db.Delete(hub.NSSpecs, id)
	m.db.Delete(hub.NSSpecs, id+":requirements")
	m.db.Delete(hub.NSSpecs, id+":implementation")
	m.db.Delete(hub.NSSpecs, id+":verification")
	return nil
}

func (m *Manager) Archive(id string) error {
	return m.UpdateStatus(id, "archived")
}

func unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func slugify(name string) string {
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result += string(r)
		} else if r == ' ' || r == '_' {
			result += "-"
		} else {
			result += "-"
		}
	}
	for len(result) > 0 && result[0] == '-' {
		result = result[1:]
	}
	for len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	return result
}

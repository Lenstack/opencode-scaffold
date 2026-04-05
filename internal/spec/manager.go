package spec

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

func (m *Manager) Create(name string, reqs db.SpecRequirements) (*db.SpecEntry, error) {
	id := slugify(name)
	var existing db.SpecEntry
	if err := m.db.Get(db.NSSpecs, id, &existing); err == nil || err != db.ErrNotFound {
		if err == nil {
			return nil, fmt.Errorf("spec %s already exists", id)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	entry := db.SpecEntry{
		ID:        id,
		Name:      name,
		Status:    "draft",
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.db.Put(db.NSSpecs, id, entry)
	m.db.Put(db.NSSpecs, id+":requirements", reqs)

	return &entry, nil
}

func (m *Manager) Get(id string) (*db.SpecEntry, error) {
	var entry db.SpecEntry
	if err := m.db.Get(db.NSSpecs, id, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (m *Manager) List() ([]db.SpecEntry, error) {
	var specs []db.SpecEntry
	m.db.Iterate(db.NSSpecs, func(key string, value []byte) error {
		if len(key) > 0 && key[0] != ':' {
			var entry db.SpecEntry
			if err := unmarshal(value, &entry); err == nil {
				specs = append(specs, entry)
			}
		}
		return nil
	})
	return specs, nil
}

func (m *Manager) UpdateStatus(id, status string) error {
	var entry db.SpecEntry
	if err := m.db.Get(db.NSSpecs, id, &entry); err != nil {
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
	return m.db.Put(db.NSSpecs, id, entry)
}

func (m *Manager) UpdateImplementation(id string, impl db.SpecImplementation) error {
	impl.VerifiedAt = time.Now().UTC().Format(time.RFC3339)
	return m.db.Put(db.NSSpecs, id+":implementation", impl)
}

func (m *Manager) Verify(id string, verification db.SpecVerification) error {
	verification.Status = "verified"
	if len(verification.FailedCriteria) > 0 {
		verification.Status = "failed"
	}
	if err := m.db.Put(db.NSSpecs, id+":verification", verification); err != nil {
		return err
	}
	if verification.Status == "verified" {
		return m.UpdateStatus(id, "done")
	}
	return nil
}

func (m *Manager) GetRequirements(id string) (*db.SpecRequirements, error) {
	var reqs db.SpecRequirements
	if err := m.db.Get(db.NSSpecs, id+":requirements", &reqs); err != nil {
		return nil, err
	}
	return &reqs, nil
}

func (m *Manager) GetImplementation(id string) (*db.SpecImplementation, error) {
	var impl db.SpecImplementation
	if err := m.db.Get(db.NSSpecs, id+":implementation", &impl); err != nil {
		return nil, err
	}
	return &impl, nil
}

func (m *Manager) GetVerification(id string) (*db.SpecVerification, error) {
	var v db.SpecVerification
	if err := m.db.Get(db.NSSpecs, id+":verification", &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (m *Manager) Delete(id string) error {
	m.db.Delete(db.NSSpecs, id)
	m.db.Delete(db.NSSpecs, id+":requirements")
	m.db.Delete(db.NSSpecs, id+":implementation")
	m.db.Delete(db.NSSpecs, id+":verification")
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

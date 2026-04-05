package skill

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

var skillNameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Manager struct {
	db *hub.Engine
}

func NewManager(db *hub.Engine) *Manager {
	return &Manager{db: db}
}

func (m *Manager) Create(name string) (*hub.SkillEntry, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if m.db.Has(hub.NSSkills, name) {
		return nil, fmt.Errorf("skill %s already exists", name)
	}

	entry := hub.SkillEntry{
		Name:          name,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		UsageCount:    0,
		Effectiveness: 0.5,
	}
	m.db.Put(hub.NSSkills, name, entry)
	m.db.Put(hub.NSSkills, name+":knowledge", hub.SkillKnowledge{})
	return &entry, nil
}

func (m *Manager) Get(name string) (*hub.SkillEntry, error) {
	var entry hub.SkillEntry
	if err := m.db.Get(hub.NSSkills, name, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (m *Manager) List() ([]hub.SkillEntry, error) {
	var skills []hub.SkillEntry
	m.db.Iterate(hub.NSSkills, func(key string, value []byte) error {
		if !strings.Contains(key, ":") {
			var entry hub.SkillEntry
			if err := json.Unmarshal(value, &entry); err == nil {
				skills = append(skills, entry)
			}
		}
		return nil
	})
	return skills, nil
}

func (m *Manager) TrackUsage(name, sessionID, outcome string) error {
	var entry hub.SkillEntry
	if err := m.db.Get(hub.NSSkills, name, &entry); err != nil {
		return err
	}

	entry.UsageCount++
	entry.LastUsed = time.Now().UTC().Format(time.RFC3339)
	if outcome == "success" {
		entry.Effectiveness = min(1.0, entry.Effectiveness+0.05)
	} else {
		entry.Effectiveness = max(0.0, entry.Effectiveness-0.10)
	}
	m.db.Put(hub.NSSkills, name, entry)

	usage := hub.SkillUsage{
		SessionID: sessionID,
		LoadedAt:  time.Now().UTC().Format(time.RFC3339),
		Outcome:   outcome,
	}

	var usages []hub.SkillUsage
	usageKey := name + ":usage"
	m.db.Iterate(hub.NSSkills, func(key string, value []byte) error {
		if key == usageKey {
			var u []hub.SkillUsage
			if err := json.Unmarshal(value, &u); err == nil {
				usages = u
			}
		}
		return nil
	})
	usages = append(usages, usage)
	return m.db.Put(hub.NSSkills, usageKey, usages)
}

func (m *Manager) Optimize(name string, knowledge hub.SkillKnowledge) error {
	return m.db.Put(hub.NSSkills, name+":knowledge", knowledge)
}

func (m *Manager) GetKnowledge(name string) (*hub.SkillKnowledge, error) {
	var k hub.SkillKnowledge
	if err := m.db.Get(hub.NSSkills, name+":knowledge", &k); err != nil {
		return nil, err
	}
	return &k, nil
}

func (m *Manager) LogOptimization(skill, change, reason string, delta float64) error {
	log := hub.OptimizationLog{
		Skill:              skill,
		Change:             change,
		Reason:             reason,
		Timestamp:          time.Now().UTC().Format(time.RFC3339),
		EffectivenessDelta: delta,
	}
	return m.db.Put(hub.NSOptimization, time.Now().Format("20060102150405")+"-"+skill, log)
}

func (m *Manager) Archive(name string) error {
	return m.db.Delete(hub.NSSkills, name)
}

func (m *Manager) Suggest(projectStack string) []string {
	suggestions := map[string][]string{
		"go":        {"tdd-workflow", "code-review", "git-workflow"},
		"go-encore": {"tdd-workflow", "code-review", "security-audit", "api-design"},
		"python":    {"tdd-workflow", "code-review", "git-workflow"},
		"node":      {"tdd-workflow", "code-review", "git-workflow"},
		"rust":      {"tdd-workflow", "code-review", "git-workflow"},
	}
	if skills, ok := suggestions[projectStack]; ok {
		return skills
	}
	return suggestions["go"]
}

func ValidateName(name string) error {
	if len(name) < 1 || len(name) > 64 {
		return fmt.Errorf("skill name must be 1-64 characters")
	}
	if !skillNameRe.MatchString(name) {
		return fmt.Errorf("skill name must match ^[a-z0-9]+(-[a-z0-9]+)*$")
	}
	return nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

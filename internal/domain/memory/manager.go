package memory

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

func (m *Manager) AddEpisodic(mem hub.EpisodicMemory) error {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	mem.TS = time.Now().UTC().Format(time.RFC3339)
	mem.ExpiresAt = time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339)
	return m.db.Put(hub.NSMemoryEpisodic, id, mem)
}

func (m *Manager) AddSemantic(mem hub.SemanticMemory) error {
	key := fmt.Sprintf("%x", hashKey(mem.FactKey))
	mem.TS = time.Now().UTC().Format(time.RFC3339)
	mem.ExpiresAt = time.Now().Add(90 * 24 * time.Hour).UTC().Format(time.RFC3339)

	var existing hub.SemanticMemory
	if err := m.db.Get(hub.NSMemorySemantic, key, &existing); err == nil {
		mem.SessionCount = existing.SessionCount + 1
		mem.Confidence = min(1.0, existing.Confidence+0.25)
	} else {
		mem.Confidence = 0.50
		mem.SessionCount = 1
	}

	return m.db.Put(hub.NSMemorySemantic, key, mem)
}

func (m *Manager) AddHeuristic(rule hub.HeuristicRule) error {
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("RULE-%d", time.Now().UnixNano())
	}
	rule.PromotedAt = time.Now().UTC().Format(time.RFC3339)
	return m.db.Put(hub.NSMemoryHeuristic, rule.ID, rule)
}

func (m *Manager) AddQuarantine(fact string, confidence float64, reason string) error {
	key := fmt.Sprintf("%x", hashKey(fact))
	data := map[string]any{
		"fact":       fact,
		"confidence": confidence,
		"moved_at":   time.Now().UTC().Format(time.RFC3339),
		"reason":     reason,
	}
	return m.db.Put(hub.NSMemoryQuarantine, key, data)
}

func (m *Manager) SearchEpisodic(query string) ([]hub.EpisodicMemory, error) {
	var results []hub.EpisodicMemory
	m.db.Iterate(hub.NSMemoryEpisodic, func(key string, value []byte) error {
		var mem hub.EpisodicMemory
		if err := unmarshal(value, &mem); err == nil {
			if containsAny(mem.Feature+mem.KeyLesson, query) {
				results = append(results, mem)
			}
		}
		return nil
	})
	return results, nil
}

func (m *Manager) SearchSemantic(query string, minConfidence float64) ([]hub.SemanticMemory, error) {
	var results []hub.SemanticMemory
	m.db.Iterate(hub.NSMemorySemantic, func(key string, value []byte) error {
		var mem hub.SemanticMemory
		if err := unmarshal(value, &mem); err == nil {
			if mem.Confidence >= minConfidence && containsAny(mem.Fact+mem.FactKey, query) {
				results = append(results, mem)
			}
		}
		return nil
	})
	return results, nil
}

func (m *Manager) ListHeuristics() ([]hub.HeuristicRule, error) {
	var rules []hub.HeuristicRule
	m.db.Iterate(hub.NSMemoryHeuristic, func(key string, value []byte) error {
		var rule hub.HeuristicRule
		if err := unmarshal(value, &rule); err == nil && rule.Active {
			rules = append(rules, rule)
		}
		return nil
	})
	return rules, nil
}

func (m *Manager) PruneExpired() (int, error) {
	episodic, _ := m.db.PruneExpired(hub.NSMemoryEpisodic)
	semantic, _ := m.db.PruneExpired(hub.NSMemorySemantic)

	quarantined := 0
	m.db.Iterate(hub.NSMemorySemantic, func(key string, value []byte) error {
		var mem hub.SemanticMemory
		if err := unmarshal(value, &mem); err == nil {
			if mem.Confidence < 0.60 {
				m.AddQuarantine(mem.Fact, mem.Confidence, "low confidence after TTL")
				m.db.Delete(hub.NSMemorySemantic, key)
				quarantined++
			}
		}
		return nil
	})

	return episodic + semantic + quarantined, nil
}

func (m *Manager) Get(key string) (any, error) {
	return nil, fmt.Errorf("use specific Get methods")
}

func containsAny(text, query string) bool {
	for _, word := range splitWords(query) {
		if len(word) > 2 && contains(text, word) {
			return true
		}
	}
	return false
}

func splitWords(s string) []string {
	var words []string
	current := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			current += string(r)
		} else if len(current) > 0 {
			words = append(words, current)
			current = ""
		}
	}
	if len(current) > 0 {
		words = append(words, current)
	}
	return words
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func hashKey(s string) []byte {
	h := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		h[i] = s[i] ^ byte(i)
	}
	return h
}

func unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
